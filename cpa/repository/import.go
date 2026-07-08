package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (r *Repository) PreviewImport(ctx context.Context, workspaceID string, pkg ExportPackage) (ImportPreview, error) {
	if err := validateExportPackage(pkg); err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{Format: pkg.Format, Service: pkg.Service, Counts: countPackage(pkg)}
	existing, err := r.importExistingOfferKeys(ctx, workspaceID)
	if err != nil {
		return ImportPreview{}, err
	}
	for _, offer := range pkg.Offers {
		if existing[offer.ID] {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "offer", Key: offer.ID})
		}
	}
	return preview, nil
}

func (r *Repository) Import(ctx context.Context, workspaceID string, req ImportRequest) (ImportResult, error) {
	if err := validateExportPackage(req.Package); err != nil {
		return ImportResult{}, err
	}
	strategy := req.ConflictStrategy
	if strategy == "" {
		strategy = ImportConflictFail
	}
	if strategy != ImportConflictFail && strategy != ImportConflictSkip && strategy != ImportConflictUpdate {
		return ImportResult{}, fmt.Errorf("unsupported import conflict strategy: %s", strategy)
	}
	preview, err := r.PreviewImport(ctx, workspaceID, req.Package)
	if err != nil {
		return ImportResult{}, err
	}
	if strategy == ImportConflictFail && len(preview.Conflicts) > 0 {
		return ImportResult{}, fmt.Errorf("import conflicts found: %d", len(preview.Conflicts))
	}
	result := ImportResult{}
	err = r.WithTx(ctx, func(txRepo *Repository) error {
		return txRepo.importBulk(ctx, workspaceID, req.Package, strategy, preview, &result)
	})
	if err != nil {
		return ImportResult{}, err
	}
	return result, r.invalidateCPACache(workspaceID)
}

func (r *Repository) importBulk(ctx context.Context, workspaceID string, pkg ExportPackage, strategy string, preview ImportPreview, result *ImportResult) error {
	if err := r.importOffersBulk(ctx, workspaceID, pkg.Offers, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importLocalizationsBulk(ctx, workspaceID, pkg.Offers, strategy, preview, result); err != nil {
		return err
	}
	return r.importRewardsBulk(ctx, workspaceID, pkg.Offers, strategy, preview, result)
}

func (r *Repository) importOffersBulk(ctx context.Context, workspaceID string, offers []ExportOffer, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(offers))
	for _, offer := range offers {
		if previewHasConflict(preview, "offer", offer.ID) && strategy == ImportConflictSkip {
			result.Skipped.Offers++
			continue
		}
		rows = append(rows, []any{
			workspaceID, offer.ID, defaultJSON(offer.Payload, "{}"), defaultJSON(offer.Target, "null"),
			offer.CodeMode, nullCodeSourceString(offer.CodeSource), nullString(offer.SharedCode),
			nullInt16(offer.GeneratedLength), nullString(offer.GeneratedAlphabet),
			offer.IsActive, nullTime(offer.StartAt), nullTime(offer.EndAt),
		})
		result.Imported.Offers++
	}
	return r.execImportBulk(ctx, "cpa_offer",
		[]string{
			"workspace_id", "id", "payload", "target", "code_mode", "code_source", "shared_code",
			"generated_length", "generated_alphabet", "is_active", "start_at", "end_at",
		},
		rows,
		"payload = EXCLUDED.payload, target = EXCLUDED.target, code_mode = EXCLUDED.code_mode, "+
			"code_source = EXCLUDED.code_source, shared_code = EXCLUDED.shared_code, generated_length = EXCLUDED.generated_length, "+
			"generated_alphabet = EXCLUDED.generated_alphabet, is_active = EXCLUDED.is_active, start_at = EXCLUDED.start_at, "+
			"end_at = EXCLUDED.end_at, updated_at = now()",
	)
}

func (r *Repository) importLocalizationsBulk(ctx context.Context, workspaceID string, offers []ExportOffer, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, offer := range offers {
		if previewHasConflict(preview, "offer", offer.ID) && strategy == ImportConflictSkip {
			continue
		}
		for locale, text := range offer.Localization {
			rows = append(rows, []any{workspaceID, offer.ID, locale, text.Title, text.Description})
			result.Imported.Localizations++
		}
	}
	return r.execImportBulk(ctx, "cpa_localization",
		[]string{"workspace_id", "cpa_id", "locale", "title", "description"},
		rows,
		"title = EXCLUDED.title, description = EXCLUDED.description, updated_at = now()",
	)
}

func (r *Repository) importRewardsBulk(ctx context.Context, workspaceID string, offers []ExportOffer, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, offer := range offers {
		if previewHasConflict(preview, "offer", offer.ID) && strategy == ImportConflictSkip {
			continue
		}
		for _, reward := range offer.Rewards {
			rows = append(rows, []any{
				workspaceID, offer.ID, reward.Key, defaultString(reward.Type, "quantity"),
				reward.Quantity, reward.Scale, nullString(reward.Unit),
			})
			result.Imported.Rewards++
		}
	}
	return r.execImportBulk(ctx, "cpa_reward",
		[]string{"workspace_id", "cpa_id", "reward_key", "reward_type", "quantity", "scale", "duration_unit"},
		rows,
		"reward_type = EXCLUDED.reward_type, quantity = EXCLUDED.quantity, scale = EXCLUDED.scale, "+
			"duration_unit = EXCLUDED.duration_unit, updated_at = now()",
	)
}

func (r *Repository) execImportBulk(ctx context.Context, table string, columns []string, rows [][]any, duplicateUpdate string) error {
	if len(rows) == 0 {
		return nil
	}
	query, args := compileImportBulkUpsert(table, columns, rows, duplicateUpdate)
	_, err := r.executor.ExecContext(ctx, query, args...)
	return err
}

func compileImportBulkUpsert(table string, columns []string, rows [][]any, duplicateUpdate string) (string, []any) {
	var builder strings.Builder
	builder.WriteString("INSERT INTO ")
	builder.WriteString(table)
	builder.WriteString(" (")
	builder.WriteString(strings.Join(columns, ", "))
	builder.WriteString(") VALUES ")
	args := make([]any, 0, len(rows)*len(columns))
	for rowIndex, row := range rows {
		if rowIndex > 0 {
			builder.WriteString(", ")
		}
		builder.WriteByte('(')
		for columnIndex := range columns {
			if columnIndex > 0 {
				builder.WriteString(", ")
			}
			builder.WriteByte('$')
			builder.WriteString(fmt.Sprint(len(args) + columnIndex + 1))
		}
		builder.WriteByte(')')
		args = append(args, row...)
	}
	if duplicateUpdate != "" {
		builder.WriteString(" ON CONFLICT ")
		builder.WriteString(importConflictTarget(table))
		builder.WriteString(" DO UPDATE SET ")
		builder.WriteString(duplicateUpdate)
	}
	return builder.String(), args
}

func importConflictTarget(table string) string {
	switch table {
	case "cpa_offer":
		return "(workspace_id, id)"
	case "cpa_localization":
		return "(workspace_id, cpa_id, locale)"
	case "cpa_reward":
		return "(workspace_id, cpa_id, reward_key)"
	default:
		return ""
	}
}

func validateExportPackage(pkg ExportPackage) error {
	if pkg.Format != ExportFormat {
		return fmt.Errorf("unsupported export format: %s", pkg.Format)
	}
	if pkg.Service != "cpa" {
		return fmt.Errorf("unsupported export service: %s", pkg.Service)
	}
	for _, item := range pkg.Items {
		if item.ID == "" {
			return fmt.Errorf("item id is required")
		}
	}
	return nil
}

func countPackage(pkg ExportPackage) ImportCounts {
	var counts ImportCounts
	counts.Items = uint64(len(pkg.Items))
	counts.Offers = uint64(len(pkg.Offers))
	for _, offer := range pkg.Offers {
		counts.Localizations += uint64(len(offer.Localization))
		counts.Rewards += uint64(len(offer.Rewards))
	}
	return counts
}

func (r *Repository) importExistingOfferKeys(ctx context.Context, workspaceID string) (map[string]bool, error) {
	offers, err := r.ListOffers(ctx, workspaceID, 100000, 0)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(offers))
	for _, offer := range offers {
		result[offer.ID] = true
	}
	return result, nil
}

func previewHasConflict(preview ImportPreview, kind, key string) bool {
	for _, conflict := range preview.Conflicts {
		if conflict.Type == kind && conflict.Key == key {
			return true
		}
	}
	return false
}

func defaultJSON(value []byte, fallback string) string {
	if len(value) == 0 {
		return fallback
	}
	return string(value)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func nullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullCodeSourceString(value *string) sql.NullString {
	return nullString(value)
}

func nullInt16(value *int16) sql.NullInt16 {
	if value == nil {
		return sql.NullInt16{}
	}
	return sql.NullInt16{Int16: *value, Valid: true}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

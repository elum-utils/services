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
	existing, err := r.importExistingPromoCodes(ctx, workspaceID)
	if err != nil {
		return ImportPreview{}, err
	}
	for _, promo := range pkg.Promos {
		key := normalizeCode(promo.Code)
		if existing[key] {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "promo", Key: promo.Code})
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
	return result, r.invalidatePromoCache(workspaceID)
}

func (r *Repository) importBulk(ctx context.Context, workspaceID string, pkg ExportPackage, strategy string, preview ImportPreview, result *ImportResult) error {
	if err := r.importPromosBulk(ctx, workspaceID, pkg.Promos, strategy, preview, result); err != nil {
		return err
	}
	ids, err := r.importPromoIDs(ctx, workspaceID, pkg.Promos, strategy, preview)
	if err != nil {
		return err
	}
	if err := r.importLocalizationsBulk(ctx, workspaceID, pkg.Promos, ids, strategy, preview, result); err != nil {
		return err
	}
	return r.importRewardsBulk(ctx, workspaceID, pkg.Promos, ids, strategy, preview, result)
}

func (r *Repository) importPromosBulk(ctx context.Context, workspaceID string, promos []ExportPromo, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(promos))
	for _, promo := range promos {
		if previewHasConflict(preview, "promo", promo.Code) && strategy == ImportConflictSkip {
			result.Skipped.Promos++
			continue
		}
		rows = append(rows, []any{
			workspaceID, promo.Code, normalizeCode(promo.Code), defaultJSON(promo.Payload, "{}"),
			defaultJSON(promo.Target, "null"), promo.MaxActivations, promo.IsActive,
			nullTime(promo.StartAt), nullTime(promo.EndAt),
		})
		result.Imported.Promos++
	}
	return r.execImportBulk(ctx, "promo_offer",
		[]string{"workspace_id", "code", "code_normalized", "payload", "target", "max_activations", "is_active", "start_at", "end_at"},
		rows,
		"code = VALUES(code), payload = VALUES(payload), target = VALUES(target), max_activations = VALUES(max_activations), "+
			"is_active = VALUES(is_active), start_at = VALUES(start_at), end_at = VALUES(end_at), deleted_at = NULL, updated_at = NOW()",
	)
}

func (r *Repository) importPromoIDs(ctx context.Context, workspaceID string, promos []ExportPromo, strategy string, preview ImportPreview) (map[string]uint64, error) {
	needed := make(map[string]struct{}, len(promos))
	for _, promo := range promos {
		if previewHasConflict(preview, "promo", promo.Code) && strategy == ImportConflictSkip {
			continue
		}
		needed[normalizeCode(promo.Code)] = struct{}{}
	}
	values, err := r.ListPromos(ctx, workspaceID, 100000, 0)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]uint64, len(needed))
	for _, promo := range values {
		key := normalizeCode(promo.Code)
		if _, ok := needed[key]; ok {
			ids[key] = promo.ID
		}
	}
	return ids, nil
}

func (r *Repository) importLocalizationsBulk(ctx context.Context, workspaceID string, promos []ExportPromo, ids map[string]uint64, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, promo := range promos {
		if previewHasConflict(preview, "promo", promo.Code) && strategy == ImportConflictSkip {
			continue
		}
		id := ids[normalizeCode(promo.Code)]
		for locale, text := range promo.Localization {
			rows = append(rows, []any{workspaceID, id, locale, text.Title, text.Description})
			result.Imported.Localizations++
		}
	}
	return r.execImportBulk(ctx, "promo_localization",
		[]string{"workspace_id", "promo_id", "locale", "title", "description"},
		rows,
		"title = VALUES(title), description = VALUES(description), updated_at = NOW()",
	)
}

func (r *Repository) importRewardsBulk(ctx context.Context, workspaceID string, promos []ExportPromo, ids map[string]uint64, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, promo := range promos {
		if previewHasConflict(preview, "promo", promo.Code) && strategy == ImportConflictSkip {
			continue
		}
		id := ids[normalizeCode(promo.Code)]
		for _, reward := range promo.Rewards {
			rows = append(rows, []any{
				workspaceID, id, reward.Key, defaultString(reward.Type, "quantity"),
				reward.Quantity, reward.Scale, nullString(reward.Unit),
			})
			result.Imported.Rewards++
		}
	}
	return r.execImportBulk(ctx, "promo_reward",
		[]string{"workspace_id", "promo_id", "reward_key", "reward_type", "quantity", "scale", "duration_unit"},
		rows,
		"reward_type = VALUES(reward_type), quantity = VALUES(quantity), scale = VALUES(scale), duration_unit = VALUES(duration_unit), updated_at = NOW()",
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
			builder.WriteByte('?')
		}
		builder.WriteByte(')')
		args = append(args, row...)
	}
	if duplicateUpdate != "" {
		builder.WriteString(" ON DUPLICATE KEY UPDATE ")
		builder.WriteString(duplicateUpdate)
	}
	return builder.String(), args
}

func validateExportPackage(pkg ExportPackage) error {
	if pkg.Format != ExportFormat || pkg.Service != "promo" {
		return fmt.Errorf("unsupported export package: %s/%s", pkg.Service, pkg.Format)
	}
	return nil
}

func countPackage(pkg ExportPackage) ImportCounts {
	var counts ImportCounts
	counts.Promos = uint64(len(pkg.Promos))
	for _, promo := range pkg.Promos {
		counts.Localizations += uint64(len(promo.Localization))
		counts.Rewards += uint64(len(promo.Rewards))
	}
	return counts
}

func (r *Repository) importExistingPromoCodes(ctx context.Context, workspaceID string) (map[string]bool, error) {
	promos, err := r.ListPromos(ctx, workspaceID, 100000, 0)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(promos))
	for _, promo := range promos {
		result[normalizeCode(promo.Code)] = true
	}
	return result, nil
}

func previewHasConflict(preview ImportPreview, kind, key string) bool {
	normalized := normalizeCode(key)
	for _, conflict := range preview.Conflicts {
		if conflict.Type == kind && normalizeCode(conflict.Key) == normalized {
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

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

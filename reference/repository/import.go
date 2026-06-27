package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (r *Repository) PreviewImport(ctx context.Context, workspaceID string, pkg ExportPackage) (ImportPreview, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return ImportPreview{}, err
	}
	if err := validateExportPackage(pkg); err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{Format: pkg.Format, Service: pkg.Service, Counts: countPackage(pkg)}
	existing, err := r.importExistingItemKeys(ctx, workspaceID)
	if err != nil {
		return ImportPreview{}, err
	}
	for _, item := range pkg.Items {
		if existing[item.Key] {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "item", Key: item.Key})
		}
	}
	return preview, nil
}

func (r *Repository) Import(ctx context.Context, workspaceID string, req ImportRequest) (ImportResult, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return ImportResult{}, err
	}
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
	methods := append([]string{}, referenceItemMutationCacheMethods...)
	methods = append(methods, referenceLocalizationMutationCacheMethods...)
	return result, r.bumpReferenceCacheVersions(workspaceID, methods...)
}

func (r *Repository) importBulk(ctx context.Context, workspaceID string, pkg ExportPackage, strategy string, preview ImportPreview, result *ImportResult) error {
	if err := r.importItemsBulk(ctx, workspaceID, pkg.Items, strategy, preview, result); err != nil {
		return err
	}
	return r.importLocalizationsBulk(ctx, workspaceID, pkg.Items, strategy, preview, result)
}

func (r *Repository) importItemsBulk(ctx context.Context, workspaceID string, items []ExportItem, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		if previewHasConflict(preview, "item", item.Key) && strategy == ImportConflictSkip {
			result.Skipped.Items++
			continue
		}
		rows = append(rows, []any{
			workspaceID, item.Key, defaultString(item.Type, ItemTypeQuantity),
			defaultJSON(item.Payload, "{}"), item.IsActive, nullableDeletedAt(item.Deleted),
		})
		result.Imported.Items++
	}
	return r.execImportBulk(ctx, "reference_item",
		[]string{"workspace_id", "`key`", "item_type", "payload", "is_active", "deleted_at"},
		rows,
		"item_type = VALUES(item_type), payload = VALUES(payload), is_active = VALUES(is_active), "+
			"deleted_at = VALUES(deleted_at), updated_at = NOW()",
	)
}

func (r *Repository) importLocalizationsBulk(ctx context.Context, workspaceID string, items []ExportItem, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, item := range items {
		if previewHasConflict(preview, "item", item.Key) && strategy == ImportConflictSkip {
			continue
		}
		for locale, text := range item.Localization {
			rows = append(rows, []any{workspaceID, item.Key, locale, text.Title, text.Description})
			result.Imported.Localizations++
		}
	}
	return r.execImportBulk(ctx, "reference_localization",
		[]string{"workspace_id", "item_key", "locale", "title", "description"},
		rows,
		"title = VALUES(title), description = VALUES(description), updated_at = NOW()",
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
	if pkg.Format != ExportFormat {
		return fmt.Errorf("unsupported export format: %s", pkg.Format)
	}
	if pkg.Service != "reference" {
		return fmt.Errorf("unsupported export service: %s", pkg.Service)
	}
	return nil
}

func countPackage(pkg ExportPackage) ImportCounts {
	var counts ImportCounts
	counts.Items = uint64(len(pkg.Items))
	for _, item := range pkg.Items {
		counts.Localizations += uint64(len(item.Localization))
	}
	return counts
}

func (r *Repository) importExistingItemKeys(ctx context.Context, workspaceID string) (map[string]bool, error) {
	items, err := r.AdminListItems(ctx, ListItemsParams{
		WorkspaceID: workspaceID, Limit: 100000, Offset: 0,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(items))
	for _, item := range items {
		result[item.Key] = true
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

func nullableDeletedAt(deleted bool) sql.NullTime {
	if !deleted {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: time.Now().UTC(), Valid: true}
}

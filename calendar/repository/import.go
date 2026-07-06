package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (r *Repository) PreviewImport(ctx context.Context, workspaceID string, pkg ExportPackage) (ImportPreview, error) {
	if err := validateExportPackage(pkg); err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{Format: pkg.Format, Service: pkg.Service, Counts: countPackage(pkg)}
	existing, err := r.importExistingCalendarTypes(ctx, workspaceID)
	if err != nil {
		return ImportPreview{}, err
	}
	for _, calendar := range pkg.Calendars {
		if existing[calendar.Type] != "" {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "calendar", Key: calendar.Type})
		}
	}
	return preview, nil
}

func (r *Repository) Import(ctx context.Context, workspaceID string, req ImportRequest) (ImportResult, error) {
	if workspaceID == "" {
		return ImportResult{}, fmt.Errorf("calendar import workspace is required")
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
	return result, r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) importBulk(ctx context.Context, workspaceID string, pkg ExportPackage, strategy string, preview ImportPreview, result *ImportResult) error {
	existing, err := r.importExistingCalendarTypes(ctx, workspaceID)
	if err != nil {
		return err
	}
	calendarIDs := make(map[string]string, len(pkg.Calendars))
	if err := r.importCalendarsBulk(ctx, workspaceID, pkg.Calendars, existing, calendarIDs, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importLocalizationsBulk(ctx, workspaceID, pkg.Calendars, calendarIDs, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importStepsBulk(ctx, workspaceID, pkg.Calendars, calendarIDs, strategy, preview, result); err != nil {
		return err
	}
	stepIDs, err := r.importStepIDs(ctx, workspaceID, calendarIDs)
	if err != nil {
		return err
	}
	return r.importRewardsBulk(ctx, workspaceID, pkg.Calendars, calendarIDs, stepIDs, strategy, preview, result)
}

func (r *Repository) importCalendarsBulk(ctx context.Context, workspaceID string, calendars []ExportCalendar, existing map[string]string, calendarIDs map[string]string, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(calendars))
	for _, calendar := range calendars {
		if previewHasConflict(preview, "calendar", calendar.Type) && strategy == ImportConflictSkip {
			result.Skipped.Calendars++
			continue
		}
		id := existing[calendar.Type]
		if id == "" {
			id = uuid.NewString()
		}
		calendarIDs[calendar.Type] = id
		rows = append(rows, []any{
			id, workspaceID, calendar.Type, defaultString(calendar.Mode, ModeInterval),
			defaultString(calendar.IntervalType, IntervalCalendar), defaultString(calendar.IntervalUnit, "day"),
			defaultUint32(calendar.IntervalCount, 1), defaultUint32(calendar.ResetAfterIntervals, 1),
			defaultString(calendar.EndBehavior, EndStop), defaultString(calendar.Timezone, "UTC"),
			calendar.HideFutureRewards, calendar.IsActive, nullableTime(calendar.StartAt), nullableTime(calendar.EndAt),
		})
		result.Imported.Calendars++
	}
	return r.execImportBulk(ctx, "calendar_definition",
		[]string{
			"id", "workspace_id", "type", "mode", "interval_type", "interval_unit", "interval_count",
			"reset_after_intervals", "end_behavior", "timezone", "hide_future_rewards", "is_active", "start_at", "end_at",
		},
		rows,
		"mode = VALUES(mode), interval_type = VALUES(interval_type), interval_unit = VALUES(interval_unit), "+
			"interval_count = VALUES(interval_count), reset_after_intervals = VALUES(reset_after_intervals), "+
			"end_behavior = VALUES(end_behavior), timezone = VALUES(timezone), hide_future_rewards = VALUES(hide_future_rewards), "+
			"is_active = VALUES(is_active), start_at = VALUES(start_at), end_at = VALUES(end_at), deleted_at = NULL, updated_at = NOW()",
	)
}

func (r *Repository) importLocalizationsBulk(ctx context.Context, workspaceID string, calendars []ExportCalendar, calendarIDs map[string]string, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, calendar := range calendars {
		if previewHasConflict(preview, "calendar", calendar.Type) && strategy == ImportConflictSkip {
			continue
		}
		calendarID := calendarIDs[calendar.Type]
		for locale, text := range calendar.Localization {
			rows = append(rows, []any{workspaceID, calendarID, locale, text.Title, text.Description})
			result.Imported.Localizations++
		}
	}
	return r.execImportBulk(ctx, "calendar_localization",
		[]string{"workspace_id", "calendar_id", "locale", "title", "description"},
		rows,
		"title = VALUES(title), description = VALUES(description), updated_at = NOW()",
	)
}

func (r *Repository) importStepsBulk(ctx context.Context, workspaceID string, calendars []ExportCalendar, calendarIDs map[string]string, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, calendar := range calendars {
		if previewHasConflict(preview, "calendar", calendar.Type) && strategy == ImportConflictSkip {
			continue
		}
		calendarID := calendarIDs[calendar.Type]
		for _, step := range calendar.Steps {
			rows = append(rows, []any{workspaceID, calendarID, step.Position})
			result.Imported.Steps++
		}
	}
	return r.execImportBulk(ctx, "calendar_step",
		[]string{"workspace_id", "calendar_id", "position"},
		rows,
		"position = VALUES(position), updated_at = NOW()",
	)
}

func (r *Repository) importRewardsBulk(ctx context.Context, workspaceID string, calendars []ExportCalendar, calendarIDs map[string]string, stepIDs map[string]uint64, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, calendar := range calendars {
		if previewHasConflict(preview, "calendar", calendar.Type) && strategy == ImportConflictSkip {
			continue
		}
		calendarID := calendarIDs[calendar.Type]
		for _, step := range calendar.Steps {
			stepID := stepIDs[stepMapKey(calendarID, step.Position)]
			for _, reward := range step.Rewards {
				rows = append(rows, []any{
					workspaceID, calendarID, stepID, reward.Key, defaultString(reward.Type, "quantity"),
					reward.Quantity, reward.Scale, nullableString(reward.Unit), defaultUint32(reward.Position, 1),
				})
				result.Imported.Rewards++
			}
		}
	}
	return r.execImportBulk(ctx, "calendar_reward",
		[]string{"workspace_id", "calendar_id", "step_id", "item_key", "reward_type", "item_count", "scale", "duration_unit", "position"},
		rows,
		"reward_type = VALUES(reward_type), item_count = VALUES(item_count), scale = VALUES(scale), "+
			"duration_unit = VALUES(duration_unit), position = VALUES(position), updated_at = NOW()",
	)
}

func (r *Repository) importStepIDs(ctx context.Context, workspaceID string, calendarIDs map[string]string) (map[string]uint64, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT calendar_id, position, id
FROM calendar_step
WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	allowed := make(map[string]bool, len(calendarIDs))
	for _, calendarID := range calendarIDs {
		allowed[calendarID] = true
	}
	result := make(map[string]uint64)
	for rows.Next() {
		var calendarID string
		var position uint32
		var id uint64
		if err := rows.Scan(&calendarID, &position, &id); err != nil {
			return nil, err
		}
		if allowed[calendarID] {
			result[stepMapKey(calendarID, position)] = id
		}
	}
	return result, rows.Err()
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
	if pkg.Service != "calendar" {
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
	counts.Calendars = uint64(len(pkg.Calendars))
	for _, calendar := range pkg.Calendars {
		counts.Localizations += uint64(len(calendar.Localization))
		counts.Steps += uint64(len(calendar.Steps))
		for _, step := range calendar.Steps {
			counts.Rewards += uint64(len(step.Rewards))
		}
	}
	return counts
}

func (r *Repository) importExistingCalendarTypes(ctx context.Context, workspaceID string) (map[string]string, error) {
	calendars, err := r.ListCalendars(ctx, workspaceID, 100000, 0)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(calendars))
	for _, calendar := range calendars {
		result[calendar.Type] = calendar.ID
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

func stepMapKey(calendarID string, position uint32) string {
	return fmt.Sprintf("%s:%d", calendarID, position)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultUint32(value, fallback uint32) uint32 {
	if value == 0 {
		return fallback
	}
	return value
}

func nullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

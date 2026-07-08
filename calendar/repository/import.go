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
			int32(defaultUint32(calendar.IntervalCount, 1)), int32(defaultUint32(calendar.ResetAfterIntervals, 1)),
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
		"(workspace_id, type)",
		"mode = EXCLUDED.mode, interval_type = EXCLUDED.interval_type, interval_unit = EXCLUDED.interval_unit, "+
			"interval_count = EXCLUDED.interval_count, reset_after_intervals = EXCLUDED.reset_after_intervals, "+
			"end_behavior = EXCLUDED.end_behavior, timezone = EXCLUDED.timezone, hide_future_rewards = EXCLUDED.hide_future_rewards, "+
			"is_active = EXCLUDED.is_active, start_at = EXCLUDED.start_at, end_at = EXCLUDED.end_at, deleted_at = NULL, updated_at = now()",
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
		"(workspace_id, calendar_id, locale)",
		"title = EXCLUDED.title, description = EXCLUDED.description, updated_at = now()",
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
			rows = append(rows, []any{workspaceID, calendarID, int32(step.Position)})
			result.Imported.Steps++
		}
	}
	return r.execImportBulk(ctx, "calendar_step",
		[]string{"workspace_id", "calendar_id", "position"},
		rows,
		"(workspace_id, calendar_id, position)",
		"position = EXCLUDED.position, updated_at = now()",
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
					workspaceID, calendarID, int64(stepID), reward.Key, defaultString(reward.Type, "quantity"),
					reward.Quantity, int16(reward.Scale), nullableString(reward.Unit), int32(defaultUint32(reward.Position, 1)),
				})
				result.Imported.Rewards++
			}
		}
	}
	return r.execImportBulk(ctx, "calendar_reward",
		[]string{"workspace_id", "calendar_id", "step_id", "item_key", "reward_type", "item_count", "scale", "duration_unit", "position"},
		rows,
		"(workspace_id, calendar_id, step_id, item_key)",
		"reward_type = EXCLUDED.reward_type, item_count = EXCLUDED.item_count, scale = EXCLUDED.scale, "+
			"duration_unit = EXCLUDED.duration_unit, position = EXCLUDED.position, updated_at = now()",
	)
}

func (r *Repository) importStepIDs(ctx context.Context, workspaceID string, calendarIDs map[string]string) (map[string]uint64, error) {
	rows, err := r.q.ListImportStepIDs(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool, len(calendarIDs))
	for _, calendarID := range calendarIDs {
		allowed[calendarID] = true
	}
	result := make(map[string]uint64)
	for _, row := range rows {
		if allowed[row.CalendarID] {
			result[stepMapKey(row.CalendarID, uint32(row.Position))] = uint64(row.ID)
		}
	}
	return result, nil
}

func (r *Repository) execImportBulk(ctx context.Context, table string, columns []string, rows [][]any, conflictTarget string, duplicateUpdate string) error {
	if len(rows) == 0 {
		return nil
	}
	query, args := compileImportBulkUpsert(table, columns, rows, conflictTarget, duplicateUpdate)
	_, err := r.executor.ExecContext(ctx, query, args...)
	return err
}

func compileImportBulkUpsert(table string, columns []string, rows [][]any, conflictTarget string, duplicateUpdate string) (string, []any) {
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
			builder.WriteString(fmt.Sprint(rowIndex*len(columns) + columnIndex + 1))
		}
		builder.WriteByte(')')
		args = append(args, row...)
	}
	if duplicateUpdate != "" {
		builder.WriteString(" ON CONFLICT ")
		builder.WriteString(conflictTarget)
		builder.WriteString(" DO UPDATE SET ")
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
	calendars, err := r.q.ListImportCalendarTypes(ctx, workspaceID)
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

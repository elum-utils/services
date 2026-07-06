package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

func (r *Repository) Export(ctx context.Context, workspaceID string, req ExportRequest) (ExportPackage, error) {
	if workspaceID == "" {
		return ExportPackage{}, fmt.Errorf("calendar export workspace is required")
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	calendars, err := r.ListCalendars(ctx, workspaceID, 100000, 0)
	if err != nil {
		return ExportPackage{}, err
	}
	localizations, err := r.exportLocalizations(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	steps, err := r.exportSteps(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	out := ExportPackage{
		Format: ExportFormat, Service: "calendar", CreatedAt: now.UTC(),
		Calendars: make([]ExportCalendar, 0, len(calendars)),
	}
	items := make(exportItemCollector)
	for _, calendar := range calendars {
		item := ExportCalendar{
			ID: calendar.ID, Type: calendar.Type, Mode: calendar.Mode,
			IntervalType: calendar.IntervalType, IntervalUnit: calendar.IntervalUnit,
			IntervalCount: calendar.IntervalCount, ResetAfterIntervals: calendar.ResetAfterIntervals,
			EndBehavior: calendar.EndBehavior, Timezone: calendar.Timezone,
			HideFutureRewards: calendar.HideFutureRewards, IsActive: calendar.IsActive,
			StartAt: calendar.StartAt, EndAt: calendar.EndAt,
			Localization: localizations[calendar.ID], Steps: steps[calendar.ID],
		}
		for _, step := range item.Steps {
			for _, reward := range step.Rewards {
				items.add(reward.Key)
			}
		}
		out.Calendars = append(out.Calendars, item)
	}
	out.Items = items.list()
	return out, nil
}

func (r *Repository) exportLocalizations(ctx context.Context, workspaceID string) (map[string]map[string]ExportText, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT calendar_id, locale, title, description
FROM calendar_localization
WHERE workspace_id = ?
ORDER BY calendar_id, locale`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]map[string]ExportText)
	for rows.Next() {
		var calendarID, locale string
		var text ExportText
		if err := rows.Scan(&calendarID, &locale, &text.Title, &text.Description); err != nil {
			return nil, err
		}
		if result[calendarID] == nil {
			result[calendarID] = make(map[string]ExportText)
		}
		result[calendarID][locale] = text
	}
	return result, rows.Err()
}

type exportItemCollector map[string]struct{}

func (c exportItemCollector) add(id string) {
	if id == "" {
		return
	}
	c[id] = struct{}{}
}

func (c exportItemCollector) list() []ExportItem {
	if len(c) == 0 {
		return nil
	}
	ids := make([]string, 0, len(c))
	for id := range c {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]ExportItem, 0, len(ids))
	for index, id := range ids {
		items = append(items, ExportItem{ID: id, Position: int32((index + 1) * 10)})
	}
	return items
}

func (r *Repository) exportSteps(ctx context.Context, workspaceID string) (map[string][]ExportStep, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT s.calendar_id, s.id, s.position, r.item_key, r.reward_type, r.item_count, r.scale, r.duration_unit, r.position
FROM calendar_step s
LEFT JOIN calendar_reward r
  ON r.workspace_id = s.workspace_id AND r.calendar_id = s.calendar_id AND r.step_id = s.id
WHERE s.workspace_id = ?
ORDER BY s.calendar_id, s.position, r.position, r.id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]ExportStep)
	var lastCalendarID string
	var lastStepID uint64
	for rows.Next() {
		var calendarID string
		var stepID uint64
		var position uint32
		var rewardKey, rewardType, rewardUnit sql.NullString
		var rewardQuantity sql.NullInt64
		var rewardScale sql.NullInt16
		var rewardPosition sql.NullInt32
		if err := rows.Scan(&calendarID, &stepID, &position, &rewardKey, &rewardType, &rewardQuantity, &rewardScale, &rewardUnit, &rewardPosition); err != nil {
			return nil, err
		}
		if calendarID != lastCalendarID || stepID != lastStepID {
			result[calendarID] = append(result[calendarID], ExportStep{Position: position})
			lastCalendarID, lastStepID = calendarID, stepID
		}
		if rewardKey.Valid {
			steps := result[calendarID]
			index := len(steps) - 1
			steps[index].Rewards = append(steps[index].Rewards, ExportReward{
				Key: rewardKey.String, Type: rewardType.String, Quantity: rewardQuantity.Int64,
				Scale: uint16FromSQL(rewardScale), Unit: nullStringPtr(rewardUnit),
				Position: uint32FromSQL(rewardPosition),
			})
			result[calendarID] = steps
		}
	}
	return result, rows.Err()
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func uint16FromSQL(value sql.NullInt16) uint16 {
	if !value.Valid || value.Int16 < 0 {
		return 0
	}
	return uint16(value.Int16)
}

func uint32FromSQL(value sql.NullInt32) uint32 {
	if !value.Valid || value.Int32 < 0 {
		return 0
	}
	return uint32(value.Int32)
}

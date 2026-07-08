package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	calendarsqlc "github.com/elum-utils/services/calendar/sqlc"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

func (r *Repository) Export(ctx context.Context, workspaceID string, req ExportRequest) (ExportPackage, error) {
	if workspaceID == "" {
		return ExportPackage{}, fmt.Errorf("calendar export workspace is required")
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	calendars, err := r.q.ListExportCalendars(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	localizationRows, err := r.q.ListExportLocalizations(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	localizations := mapExportLocalizations(localizationRows)
	stepRows, err := r.q.ListExportStepsWithRewards(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	steps := mapExportSteps(stepRows)
	out := ExportPackage{
		Format: ExportFormat, Service: "calendar", CreatedAt: now.UTC(),
		Calendars: make([]ExportCalendar, 0, len(calendars)),
	}
	items := make(exportItemCollector)
	for _, calendar := range calendars {
		item := ExportCalendar{
			ID:                  calendar.ID,
			Type:                calendar.Type,
			Mode:                calendar.Mode,
			IntervalType:        calendar.IntervalType,
			IntervalUnit:        calendar.IntervalUnit,
			IntervalCount:       uint32(calendar.IntervalCount),
			ResetAfterIntervals: uint32(calendar.ResetAfterIntervals),
			EndBehavior:         calendar.EndBehavior,
			Timezone:            calendar.Timezone,
			HideFutureRewards:   calendar.HideFutureRewards,
			IsActive:            calendar.IsActive,
			StartAt:             sqlwrap.NullTimePtr(calendar.StartAt),
			EndAt:               sqlwrap.NullTimePtr(calendar.EndAt),
			Localization:        localizations[calendar.ID], Steps: steps[calendar.ID],
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

func mapExportLocalizations(rows []calendarsqlc.CalendarLocalization) map[string]map[string]ExportText {
	result := make(map[string]map[string]ExportText)
	for _, row := range rows {
		if result[row.CalendarID] == nil {
			result[row.CalendarID] = make(map[string]ExportText)
		}
		result[row.CalendarID][row.Locale] = ExportText{
			Title:       row.Title,
			Description: row.Description,
		}
	}
	return result
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

func mapExportSteps(rows []calendarsqlc.ListExportStepsWithRewardsRow) map[string][]ExportStep {
	result := make(map[string][]ExportStep)
	var lastCalendarID string
	var lastStepID int64
	for _, row := range rows {
		if row.CalendarID != lastCalendarID || row.StepID != lastStepID {
			result[row.CalendarID] = append(result[row.CalendarID], ExportStep{
				Position: uint32(row.StepPosition),
			})
			lastCalendarID, lastStepID = row.CalendarID, row.StepID
		}
		if row.RewardItemKey.Valid {
			steps := result[row.CalendarID]
			index := len(steps) - 1
			steps[index].Rewards = append(steps[index].Rewards, ExportReward{
				Key:      row.RewardItemKey.String,
				Type:     row.RewardType.String,
				Quantity: row.RewardItemCount.Int64,
				Scale:    uint16FromSQL(row.RewardScale),
				Unit:     nullStringPtr(row.RewardDurationUnit),
				Position: uint32FromSQL(row.RewardPosition),
			})
			result[row.CalendarID] = steps
		}
	}
	return result
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

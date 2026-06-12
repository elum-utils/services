package repository

import (
	"context"
	"database/sql"
	"time"

	calendarsqlc "github.com/elum-utils/services/calendar/sqlc"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

func (r *Repository) GetCalendar(ctx context.Context, workspaceID, ref, locale string) (Calendar, error) {
	key := calendarCacheKey("user_get_calendar", workspaceID, ref, locale)
	rememberCalendarCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Calendar, error) {
		id, calendarType := calendarReference(ref)
		rows, err := r.q.GetCalendarBundle(ctx, calendarsqlc.GetCalendarBundleParams{
			Locale: locale, WorkspaceID: workspaceID, ID: id, Type: calendarType,
		})
		if err != nil {
			return Calendar{}, err
		}
		if len(rows) == 0 {
			return Calendar{}, nil
		}
		first := rows[0]
		value := Calendar{
			ID: first.ID, WorkspaceID: first.WorkspaceID, Type: first.Type,
			Mode: string(first.Mode), IntervalType: string(first.IntervalType),
			IntervalUnit: string(first.IntervalUnit), IntervalCount: first.IntervalCount,
			ResetAfterIntervals: first.ResetAfterIntervals, EndBehavior: string(first.EndBehavior),
			Timezone: first.Timezone, HideFutureRewards: first.HideFutureRewards,
			IsActive: first.IsActive, StartAt: sqlwrap.NullTimePtr(first.StartAt),
			EndAt: sqlwrap.NullTimePtr(first.EndAt), DeletedAt: sqlwrap.NullTimePtr(first.DeletedAt),
			CreatedAt: first.CreatedAt, UpdatedAt: first.UpdatedAt,
			Steps: make([]Step, 0),
		}
		if first.LocalizationLocale.Valid {
			value.Localization = &Localization{
				WorkspaceID: first.WorkspaceID, CalendarID: first.ID,
				Locale: first.LocalizationLocale.String, Title: first.LocalizationTitle.String,
				Description: first.LocalizationDescription.String,
			}
		}
		for _, row := range rows {
			value.Steps = appendStep(value.Steps, row.StepID, row.StepPosition,
				row.RewardID, row.RewardItemKey, row.RewardType,
				row.RewardItemCount, row.RewardDurationUnit)
		}
		return value, nil
	})
}

func (r *Repository) ListActive(ctx context.Context, workspaceID, locale string, now time.Time) ([]Calendar, error) {
	rows, err := r.q.ListActiveCalendars(ctx, calendarsqlc.ListActiveCalendarsParams{
		Locale: locale, WorkspaceID: workspaceID,
		StartAt: sql.NullTime{Time: now, Valid: true}, EndAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	result := make([]Calendar, 0, len(rows))
	for _, row := range rows {
		value := Calendar{
			ID: row.ID, WorkspaceID: row.WorkspaceID, Type: row.Type,
			Mode: string(row.Mode), IsActive: row.IsActive,
			StartAt: sqlwrap.NullTimePtr(row.StartAt), EndAt: sqlwrap.NullTimePtr(row.EndAt),
		}
		if row.Locale.Valid {
			value.Localization = &Localization{
				WorkspaceID: row.WorkspaceID, CalendarID: row.ID, Locale: row.Locale.String,
				Title: row.Title.String, Description: row.Description.String,
			}
		}
		result = append(result, value)
	}
	return result, nil
}

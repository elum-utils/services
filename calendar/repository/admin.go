package repository

import (
	"context"
	"database/sql"
	"time"

	calendarsqlc "github.com/elum-utils/services/calendar/sqlc"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

type SaveCalendarParams struct {
	ID                  string
	WorkspaceID         string
	Type                string
	Mode                string
	IntervalType        string
	IntervalUnit        string
	IntervalCount       uint32
	ResetAfterIntervals uint32
	EndBehavior         string
	Timezone            string
	HideFutureRewards   bool
	IsActive            bool
	StartAt             *time.Time
	EndAt               *time.Time
}

func (r *Repository) CreateCalendar(ctx context.Context, params SaveCalendarParams) error {
	if err := r.q.AdminCreateCalendar(ctx, calendarsqlc.AdminCreateCalendarParams{
		ID: params.ID, WorkspaceID: params.WorkspaceID, Type: params.Type,
		Mode:          calendarsqlc.CalendarDefinitionMode(params.Mode),
		IntervalType:  calendarsqlc.CalendarDefinitionIntervalType(params.IntervalType),
		IntervalUnit:  calendarsqlc.CalendarDefinitionIntervalUnit(params.IntervalUnit),
		IntervalCount: params.IntervalCount, ResetAfterIntervals: params.ResetAfterIntervals,
		EndBehavior: calendarsqlc.CalendarDefinitionEndBehavior(params.EndBehavior),
		Timezone:    params.Timezone, HideFutureRewards: params.HideFutureRewards,
		IsActive: params.IsActive, StartAt: nullableTime(params.StartAt),
		EndAt: nullableTime(params.EndAt),
	}); err != nil {
		return err
	}
	return r.invalidateCalendarCache(params.WorkspaceID)
}

func (r *Repository) UpdateCalendar(ctx context.Context, params SaveCalendarParams) (int64, error) {
	rows, err := r.q.AdminUpdateCalendar(ctx, calendarsqlc.AdminUpdateCalendarParams{
		Type: params.Type, Mode: calendarsqlc.CalendarDefinitionMode(params.Mode),
		IntervalType:  calendarsqlc.CalendarDefinitionIntervalType(params.IntervalType),
		IntervalUnit:  calendarsqlc.CalendarDefinitionIntervalUnit(params.IntervalUnit),
		IntervalCount: params.IntervalCount, ResetAfterIntervals: params.ResetAfterIntervals,
		EndBehavior: calendarsqlc.CalendarDefinitionEndBehavior(params.EndBehavior),
		Timezone:    params.Timezone, HideFutureRewards: params.HideFutureRewards,
		IsActive: params.IsActive, StartAt: nullableTime(params.StartAt),
		EndAt: nullableTime(params.EndAt), WorkspaceID: params.WorkspaceID, ID: params.ID,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCalendarCache(params.WorkspaceID)
}

func (r *Repository) GetCalendarDefinition(ctx context.Context, workspaceID, id string) (Calendar, error) {
	key := calendarCacheKey("admin_get_calendar", workspaceID, id)
	rememberCalendarCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Calendar, error) {
		row, err := r.q.AdminGetCalendar(ctx, calendarsqlc.AdminGetCalendarParams{
			WorkspaceID: workspaceID, ID: id,
		})
		if err != nil {
			return Calendar{}, err
		}
		return mapDefinition(row), nil
	})
}

func (r *Repository) ListCalendars(ctx context.Context, workspaceID string, limit, offset int32) ([]Calendar, error) {
	limit, offset = normalizePage(limit, offset)
	key := calendarCacheKey("admin_list_calendars", workspaceID, limit, offset)
	rememberCalendarCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Calendar, error) {
		rows, err := r.q.AdminListCalendars(ctx, calendarsqlc.AdminListCalendarsParams{
			WorkspaceID: workspaceID, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		result := make([]Calendar, 0, len(rows))
		for _, row := range rows {
			result = append(result, mapDefinition(row))
		}
		return result, nil
	})
}

func (r *Repository) SetCalendarActive(ctx context.Context, workspaceID, id string, active bool) (int64, error) {
	rows, err := r.q.AdminSetCalendarActive(ctx, calendarsqlc.AdminSetCalendarActiveParams{
		IsActive: active, WorkspaceID: workspaceID, ID: id,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) DeleteCalendar(ctx context.Context, workspaceID, id string) (int64, error) {
	rows, err := r.q.AdminSoftDeleteCalendar(ctx, calendarsqlc.AdminSoftDeleteCalendarParams{
		WorkspaceID: workspaceID, ID: id,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) UpsertLocalization(ctx context.Context, value Localization) error {
	if err := r.q.AdminUpsertLocalization(ctx, calendarsqlc.AdminUpsertLocalizationParams{
		WorkspaceID: value.WorkspaceID, CalendarID: value.CalendarID,
		Locale: value.Locale, Title: value.Title, Description: value.Description,
	}); err != nil {
		return err
	}
	return r.invalidateCalendarCache(value.WorkspaceID)
}

func (r *Repository) GetLocalization(ctx context.Context, workspaceID, calendarID, locale string) (Localization, error) {
	key := calendarCacheKey("admin_get_localization", workspaceID, calendarID, locale)
	rememberCalendarCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Localization, error) {
		row, err := r.q.AdminGetLocalization(ctx, calendarsqlc.AdminGetLocalizationParams{
			WorkspaceID: workspaceID, CalendarID: calendarID, Locale: locale,
		})
		if err != nil {
			return Localization{}, err
		}
		return Localization{
			WorkspaceID: row.WorkspaceID, CalendarID: row.CalendarID,
			Locale: row.Locale, Title: row.Title, Description: row.Description,
		}, nil
	})
}

func (r *Repository) ListLocalizations(ctx context.Context, workspaceID, calendarID string) ([]Localization, error) {
	key := calendarCacheKey("admin_list_localizations", workspaceID, calendarID)
	rememberCalendarCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Localization, error) {
		rows, err := r.q.AdminListLocalizations(ctx, calendarsqlc.AdminListLocalizationsParams{
			WorkspaceID: workspaceID, CalendarID: calendarID,
		})
		if err != nil {
			return nil, err
		}
		result := make([]Localization, 0, len(rows))
		for _, row := range rows {
			result = append(result, Localization{
				WorkspaceID: row.WorkspaceID, CalendarID: row.CalendarID,
				Locale: row.Locale, Title: row.Title, Description: row.Description,
			})
		}
		return result, nil
	})
}

func (r *Repository) DeleteLocalization(ctx context.Context, workspaceID, calendarID, locale string) (int64, error) {
	rows, err := r.q.AdminDeleteLocalization(ctx, calendarsqlc.AdminDeleteLocalizationParams{
		WorkspaceID: workspaceID, CalendarID: calendarID, Locale: locale,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) CreateStep(ctx context.Context, workspaceID, calendarID string, position uint32) (uint64, error) {
	id, err := r.q.AdminCreateStep(ctx, calendarsqlc.AdminCreateStepParams{
		WorkspaceID: workspaceID, CalendarID: calendarID, Position: position,
	})
	if err != nil {
		return 0, err
	}
	return uint64(id), r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) UpdateStep(ctx context.Context, workspaceID, calendarID string, id uint64, position uint32) (int64, error) {
	rows, err := r.q.AdminUpdateStep(ctx, calendarsqlc.AdminUpdateStepParams{
		Position: position, WorkspaceID: workspaceID, CalendarID: calendarID, ID: id,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) DeleteStep(ctx context.Context, workspaceID, calendarID string, id uint64) (int64, error) {
	rows, err := r.q.AdminDeleteStep(ctx, calendarsqlc.AdminDeleteStepParams{
		WorkspaceID: workspaceID, CalendarID: calendarID, ID: id,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) UpsertReward(ctx context.Context, workspaceID, calendarID string, stepID uint64, reward Reward, position uint32) (uint64, error) {
	id, err := r.q.AdminUpsertReward(ctx, calendarsqlc.AdminUpsertRewardParams{
		WorkspaceID: workspaceID, CalendarID: calendarID, StepID: stepID,
		ItemKey: reward.Key, RewardType: calendarsqlc.CalendarRewardRewardType(reward.Type),
		ItemCount: reward.Quantity, Scale: reward.Scale, DurationUnit: calendarsqlc.NullCalendarRewardDurationUnit{
			CalendarRewardDurationUnit: calendarsqlc.CalendarRewardDurationUnit(calendarStringValue(reward.Unit)),
			Valid:                      reward.Unit != nil,
		}, Position: position,
	})
	if err != nil {
		return 0, err
	}
	return uint64(id), r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) UpdateReward(ctx context.Context, workspaceID, calendarID string, stepID, id uint64, reward Reward, position uint32) (int64, error) {
	rows, err := r.q.AdminUpdateReward(ctx, calendarsqlc.AdminUpdateRewardParams{
		StepID: stepID, ItemKey: reward.Key,
		RewardType: calendarsqlc.CalendarRewardRewardType(reward.Type),
		ItemCount:  reward.Quantity,
		Scale:      reward.Scale,
		DurationUnit: calendarsqlc.NullCalendarRewardDurationUnit{
			CalendarRewardDurationUnit: calendarsqlc.CalendarRewardDurationUnit(calendarStringValue(reward.Unit)),
			Valid:                      reward.Unit != nil,
		},
		Position:    position,
		WorkspaceID: workspaceID, CalendarID: calendarID, ID: id,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCalendarCache(workspaceID)
}

func (r *Repository) GetReward(ctx context.Context, workspaceID, calendarID string, id uint64) (Reward, error) {
	key := calendarCacheKey("admin_get_reward", workspaceID, calendarID, id)
	rememberCalendarCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Reward, error) {
		row, err := r.q.AdminGetReward(ctx, calendarsqlc.AdminGetRewardParams{
			WorkspaceID: workspaceID, CalendarID: calendarID, ID: id,
		})
		if err != nil {
			return Reward{}, err
		}
		return Reward{
			Key: row.ItemKey, Type: string(row.RewardType), Quantity: row.ItemCount,
			Scale: row.Scale,
			Unit:  calendarDurationUnitPtr(row.DurationUnit),
		}, nil
	})
}

func (r *Repository) DeleteReward(ctx context.Context, workspaceID, calendarID string, id uint64) (int64, error) {
	rows, err := r.q.AdminDeleteReward(ctx, calendarsqlc.AdminDeleteRewardParams{
		WorkspaceID: workspaceID, CalendarID: calendarID, ID: id,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCalendarCache(workspaceID)
}

func nullableTime(value *time.Time) sql.NullTime {
	return sqlwrap.NullFromPtr(value, func(value time.Time) sql.NullTime {
		return sql.NullTime{Time: value, Valid: true}
	})
}

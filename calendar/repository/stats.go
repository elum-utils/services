package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	calendarsqlc "github.com/elum-utils/services/calendar/sqlc"
)

func (r *Repository) ListOperations(ctx context.Context, workspaceID, calendarID string, limit, offset int32) ([]Operation, error) {
	limit, offset = normalizePage(limit, offset)
	rows, err := r.q.AdminListOperations(ctx, calendarsqlc.AdminListOperationsParams{
		WorkspaceID: workspaceID, CalendarID: calendarID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]Operation, 0, len(rows))
	for _, row := range rows {
		result = append(result, Operation{
			ID: row.ID, Identity: Identity{
				WorkspaceID: row.WorkspaceID, AppID: row.AppID,
				PlatformID: row.PlatformID, PlatformUserID: row.PlatformUserID,
			},
			CalendarID: row.CalendarID, OperationID: row.OperationID,
			Granted: row.Granted, Status: row.Status, Position: sqlNullUint32Ptr(row.Position),
			Rewards: row.RewardsSnapshot, CurrentPosition: row.CurrentPosition,
			ClaimCount: row.ClaimCount, OccurredAt: row.OccurredAt,
		})
	}
	return result, nil
}

func (r *Repository) GetStats(ctx context.Context, workspaceID, calendarID string) (Stats, error) {
	row, err := r.q.AdminGetStats(ctx, calendarsqlc.AdminGetStatsParams{
		WorkspaceID: workspaceID, CalendarID: calendarID,
	})
	if err != nil {
		return Stats{}, err
	}
	grants, err := interfaceUint64(row.GrantCount)
	if err != nil {
		return Stats{}, err
	}
	return Stats{
		OperationCount: uint64(row.OperationCount), GrantCount: grants,
		UniqueUsers: uint64(row.UniqueUsers),
	}, nil
}

func (r *Repository) ListDailyStats(ctx context.Context, workspaceID, calendarID string, from, until time.Time) ([]DailyStats, error) {
	rows, err := r.q.AdminListDailyStats(ctx, calendarsqlc.AdminListDailyStatsParams{
		WorkspaceID: workspaceID, CalendarID: calendarID, StatsDate: from, StatsDate_2: until,
	})
	if err != nil {
		return nil, err
	}
	result := make([]DailyStats, 0, len(rows))
	for _, row := range rows {
		result = append(result, DailyStats{
			Date: row.StatsDate, OperationCount: row.OperationCount,
			GrantCount: row.GrantCount, UniqueUsers: row.UniqueUsers,
		})
	}
	return result, nil
}

func (r *Repository) RefreshDailyStats(ctx context.Context, from, until time.Time) error {
	return r.q.RefreshDailyStats(ctx, calendarsqlc.RefreshDailyStatsParams{
		OccurredAt: from, OccurredAt_2: until,
	})
}

func interfaceUint64(value any) (uint64, error) {
	switch value := value.(type) {
	case int64:
		return uint64(value), nil
	case uint64:
		return value, nil
	case []byte:
		return strconv.ParseUint(string(value), 10, 64)
	case string:
		return strconv.ParseUint(value, 10, 64)
	default:
		return 0, fmt.Errorf("calendar: unsupported numeric value %T", value)
	}
}

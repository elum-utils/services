package repository

import (
	"context"
	"time"

	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
)

type recordProgressUpsert struct {
	taskID        uint64
	periodStartAt time.Time
	periodEndAt   time.Time
	progress      uint64
	status        string
	readyAt       *time.Time
}

type recordAutoClaim struct {
	task          Task
	progress      Progress
	exists        bool
	periodStartAt time.Time
	periodEndAt   time.Time
}

func (r *Repository) batchUpsertProgress(
	ctx context.Context,
	identity Identity,
	items []recordProgressUpsert,
) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	return repositoryValue[int64](ctx, r, func(ctx context.Context) (int64, error) {
		var total int64
		for _, item := range items {
			rows, err := r.q.UpsertProgress(ctx, tasksqlc.UpsertProgressParams{
				WorkspaceID: identity.WorkspaceID, TaskID: item.taskID,
				AppID: identity.AppID, PlatformID: identity.PlatformID, PlatformUserID: identity.PlatformUserID,
				PeriodStartAt: item.periodStartAt, PeriodEndAt: item.periodEndAt,
				Progress: item.progress, Status: tasksqlc.TaskProgressStatus(item.status), ReadyAt: nullTime(item.readyAt),
			})
			if err != nil {
				return 0, err
			}
			total += rows
		}
		return total, nil
	})
}

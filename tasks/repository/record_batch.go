package repository

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"
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

const progressUpsertPrefix = `INSERT INTO task_progress (
workspace_id, task_id, app_id, platform_id, platform_user_id,
period_start_at, period_end_at, progress, status, ready_at
) VALUES `

const progressUpsertSuffix = ` ON DUPLICATE KEY UPDATE
period_end_at = VALUES(period_end_at),
progress = VALUES(progress),
status = VALUES(status),
ready_at = VALUES(ready_at)`

var progressUpsertQueryCache sync.Map

func (r *Repository) batchUpsertProgress(
	ctx context.Context,
	identity Identity,
	items []recordProgressUpsert,
) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(items)*10)
	for _, item := range items {
		readyAt := any(nil)
		if item.readyAt != nil {
			readyAt = sql.NullTime{Time: *item.readyAt, Valid: true}
		}
		args = append(args,
			identity.WorkspaceID, item.taskID, identity.AppID, identity.PlatformID,
			identity.PlatformUserID, item.periodStartAt, item.periodEndAt,
			item.progress, item.status, readyAt,
		)
	}
	return repositoryValue[int64](ctx, r, func(ctx context.Context) (int64, error) {
		result, err := r.executor.ExecContext(ctx, progressUpsertQuery(len(items)), args...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	})
}

func progressUpsertQuery(count int) string {
	if cached, ok := progressUpsertQueryCache.Load(count); ok {
		return cached.(string)
	}
	var query strings.Builder
	query.Grow(len(progressUpsertPrefix) + count*32 + len(progressUpsertSuffix))
	query.WriteString(progressUpsertPrefix)
	for index := 0; index < count; index++ {
		if index > 0 {
			query.WriteByte(',')
		}
		query.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	}
	query.WriteString(progressUpsertSuffix)
	value := query.String()
	progressUpsertQueryCache.Store(count, value)
	return value
}

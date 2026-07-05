package repository

import (
	"context"
	"strings"
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

func (r *Repository) batchUpsertProgress(
	ctx context.Context,
	identity Identity,
	items []recordProgressUpsert,
) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	query, args := compileProgressBulkUpsert(identity, items)
	return repositoryValue[int64](ctx, r, func(ctx context.Context) (int64, error) {
		result, err := r.executor.ExecContext(ctx, query, args...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	})
}

func compileProgressBulkUpsert(identity Identity, items []recordProgressUpsert) (string, []any) {
	const columns = 10
	var builder strings.Builder
	builder.Grow(len(items)*columns*4 + 320)
	builder.WriteString("INSERT INTO task_progress (")
	builder.WriteString("workspace_id, task_id, app_id, platform_id, platform_user_id, ")
	builder.WriteString("period_start_at, period_end_at, progress, status, ready_at")
	builder.WriteString(") VALUES ")
	args := make([]any, 0, len(items)*columns)
	for index, item := range items {
		if index > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			identity.WorkspaceID,
			item.taskID,
			identity.AppID,
			identity.PlatformID,
			identity.PlatformUserID,
			item.periodStartAt,
			item.periodEndAt,
			item.progress,
			item.status,
			nullTime(item.readyAt),
		)
	}
	builder.WriteString(" ON DUPLICATE KEY UPDATE ")
	builder.WriteString("period_end_at = VALUES(period_end_at), ")
	builder.WriteString("progress = VALUES(progress), ")
	builder.WriteString("status = VALUES(status), ")
	builder.WriteString("ready_at = VALUES(ready_at)")
	return builder.String(), args
}

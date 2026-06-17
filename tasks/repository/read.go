package repository

import (
	"context"
	"encoding/json"
	"time"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
)

func (r *Repository) ListActive(ctx context.Context, identity Identity, locale string, now time.Time) ([]ActiveTask, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	catalog, err := r.listActiveCatalog(ctx, identity.WorkspaceID, locale)
	if err != nil {
		return nil, err
	}
	tasks := make([]ActiveTask, 0, len(catalog))
	for _, task := range catalog {
		if activeTaskVisibleAt(task, now) {
			task.Progress = nil
			tasks = append(tasks, task)
		}
	}
	progressRows, err := repositoryValue[[]tasksqlc.TaskProgress](ctx, r, func(ctx context.Context) ([]tasksqlc.TaskProgress, error) {
		return r.q.ListCurrentProgressForUser(ctx, tasksqlc.ListCurrentProgressForUserParams{
			WorkspaceID: identity.WorkspaceID,
			AppID:       identity.AppID, PlatformID: identity.PlatformID, PlatformUserID: identity.PlatformUserID,
			PeriodStartAt: now, PeriodEndAt: now,
		})
	})
	if err != nil {
		return nil, err
	}
	progressByTask := make(map[uint64]ActiveProgress, len(progressRows))
	for _, row := range progressRows {
		progressByTask[row.TaskID] = mapActiveProgress(row)
	}
	for index := range tasks {
		if progress, ok := progressByTask[tasks[index].ID]; ok {
			tasks[index].Progress = &progress
		}
	}
	return tasks, nil
}

func (r *Repository) listActiveCatalog(ctx context.Context, workspaceID, locale string) ([]ActiveTask, error) {
	key := activeCatalogCacheKey(workspaceID, locale)
	rememberTaskCacheKey(workspaceID, key)
	out, err := repositoryQuery(ctx, r, sqlwrap.Params{
		Key:          key,
		CacheL1Delay: r.cacheL1Delay,
		CacheL2Delay: r.cacheL2Delay,
	}, func(ctx context.Context) ([]ActiveTask, error) {
		rows, err := r.q.ListActiveTaskBundles(ctx, tasksqlc.ListActiveTaskBundlesParams{
			Locale:      locale,
			WorkspaceID: workspaceID,
		})
		if err != nil {
			return nil, err
		}
		return activeTasksFromTasks(mapActiveBundles(rows)), nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func activeTasksFromTasks(tasks []Task) []ActiveTask {
	result := make([]ActiveTask, 0, len(tasks))
	for _, task := range tasks {
		out := ActiveTask{
			ID: task.ID, Key: task.Key, GroupKey: task.GroupKey, TaskKind: task.TaskKind,
			ActionKey: task.ActionKey, ActionKind: task.ActionKind, ClaimMode: task.ClaimMode,
			TargetCount: task.TargetCount, Payload: task.Payload, ImageURL: task.ImageURL,
			Rewards: task.Rewards, StartAt: task.StartAt, EndAt: task.EndAt,
		}
		if task.Localization != nil {
			out.Title = task.Localization.Title
			out.Description = task.Localization.Description
		}
		result = append(result, out)
	}
	return result
}

func activeTaskVisibleAt(task ActiveTask, now time.Time) bool {
	return (task.StartAt == nil || !task.StartAt.After(now)) && (task.EndAt == nil || task.EndAt.After(now))
}

type CallbackPayload struct {
	WorkspaceID    string          `json:"workspace_id"`
	AppID          int64           `json:"app_id"`
	PlatformID     int64           `json:"platform_id"`
	PlatformUserID string          `json:"platform_user_id"`
	TaskID         uint64          `json:"task_id"`
	TaskKey        string          `json:"task_key"`
	OperationID    string          `json:"operation_id"`
	PeriodStartAt  time.Time       `json:"period_start_at"`
	PeriodEndAt    time.Time       `json:"period_end_at"`
	Rewards        []Reward        `json:"rewards"`
	Payload        json.RawMessage `json:"payload"`
}

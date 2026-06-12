package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
)

const listActiveTasksQuery = `
SELECT t.id, t.` + "`" + `key` + "`" + `, t.group_key,
       t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.payload, t.image_url, t.start_at, t.end_at,
       l.locale, l.title, l.description,
       r.id AS reward_id, r.reward_key, r.reward_type, r.quantity AS reward_quantity, r.duration_unit
FROM task_definition t FORCE INDEX (task_definition_visible_user_list_idx)
LEFT JOIN task_localization l ON l.workspace_id = t.workspace_id AND l.task_id = t.id AND l.locale = ?
LEFT JOIN task_reward r ON r.workspace_id = t.workspace_id AND r.task_id = t.id
WHERE t.workspace_id = ? AND t.is_visible = TRUE AND t.is_active = TRUE
  AND t.deleted_at IS NULL
ORDER BY t.position, t.id, r.position, r.id`

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
		return loadActiveCatalog(ctx, r.db.DB(), workspaceID, locale)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func loadActiveCatalog(ctx context.Context, db *sql.DB, workspaceID, locale string) ([]ActiveTask, error) {
	rows, err := db.QueryContext(ctx, listActiveTasksQuery, locale, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := make([]ActiveTask, 0, 16)
	var lastID uint64
	index := -1
	for rows.Next() {
		var (
			id             uint64
			key            string
			groupKey       string
			actionKey      string
			actionKind     string
			claimMode      string
			targetCount    uint64
			payload        json.RawMessage
			imageURL       sql.NullString
			startAt        sql.NullTime
			endAt          sql.NullTime
			locLocale      sql.NullString
			locTitle       sql.NullString
			locDescription sql.NullString
			rewardID       sql.NullInt64
			rewardKey      sql.NullString
			rewardType     sql.NullString
			rewardQuantity sql.NullInt64
			rewardUnit     sql.NullString
		)
		if err := rows.Scan(
			&id,
			&key,
			&groupKey,
			&actionKey,
			&actionKind,
			&claimMode,
			&targetCount,
			&payload,
			&imageURL,
			&startAt,
			&endAt,
			&locLocale,
			&locTitle,
			&locDescription,
			&rewardID,
			&rewardKey,
			&rewardType,
			&rewardQuantity,
			&rewardUnit,
		); err != nil {
			return nil, err
		}
		if index < 0 || id != lastID {
			task := ActiveTask{
				ID: id, Key: key, GroupKey: groupKey,
				ActionKey: actionKey, ActionKind: actionKind, ClaimMode: claimMode,
				TargetCount: targetCount, Payload: payload, ImageURL: ptrString(imageURL),
				StartAt: ptrTime(startAt), EndAt: ptrTime(endAt),
				Rewards: make([]Reward, 0, 1),
			}
			if locLocale.Valid {
				task.Title = locTitle.String
				task.Description = locDescription.String
			}
			tasks = append(tasks, task)
			index = len(tasks) - 1
			lastID = id
		}
		if rewardID.Valid {
			tasks[index].Rewards = append(tasks[index].Rewards, Reward{
				Key: rewardKey.String, Type: rewardType.String,
				Quantity: rewardQuantity.Int64, Unit: ptrString(rewardUnit),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
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

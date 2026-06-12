package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

const listRecordCatalogQuery = `
SELECT t.id, t.workspace_id, t.` + "`" + `key` + "`" + `, t.group_key, t.sequence_key, t.sequence_position,
       t.action_key, t.action_kind, t.claim_mode, t.target_count, t.reset_unit,
       t.reset_every, t.payload, t.position, t.start_at, t.end_at
FROM task_definition t FORCE INDEX (task_definition_action_idx)
WHERE t.workspace_id = ?
  AND t.action_key = ?
  AND t.is_active = TRUE
  AND t.deleted_at IS NULL
ORDER BY t.branch_sort_key, t.sequence_position, t.position, t.id`

const claimCatalogByIDQuery = `
SELECT t.id, t.workspace_id, t.` + "`" + `key` + "`" + `, t.group_key, t.sequence_key, t.sequence_position,
       t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.payload, t.image_url,
       r.id AS reward_id, r.reward_key, r.reward_type, r.quantity AS reward_quantity, r.duration_unit
FROM task_definition t
LEFT JOIN task_reward r ON r.workspace_id = t.workspace_id AND r.task_id = t.id
WHERE t.workspace_id = ? AND t.id = ?
ORDER BY r.position, r.id`

const claimCatalogByKeyQuery = `
SELECT t.id, t.workspace_id, t.` + "`" + `key` + "`" + `, t.group_key, t.sequence_key, t.sequence_position,
       t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.payload, t.image_url,
       r.id AS reward_id, r.reward_key, r.reward_type, r.quantity AS reward_quantity, r.duration_unit
FROM task_definition t
LEFT JOIN task_reward r ON r.workspace_id = t.workspace_id AND r.task_id = t.id
WHERE t.workspace_id = ? AND t.` + "`" + `key` + "`" + ` = ?
ORDER BY r.position, r.id`

const listRewardsCatalogQuery = `
SELECT reward_key, reward_type, quantity, duration_unit
FROM task_reward
WHERE workspace_id = ? AND task_id = ?
ORDER BY position, id`

const nextSequenceTaskQuery = `
SELECT id
FROM task_definition
WHERE workspace_id = ?
  AND sequence_key = ?
  AND sequence_position > ?
  AND deleted_at IS NULL
ORDER BY sequence_position, id
LIMIT 1`

type nextSequenceTask struct {
	ID     uint64
	Exists bool
}

func (r *Repository) listRecordCatalog(ctx context.Context, workspaceID, actionKey string) ([]Task, error) {
	key := recordCatalogCacheKey(workspaceID, actionKey)
	rememberTaskCacheKey(workspaceID, key)
	out, err := repositoryQuery[[]Task](ctx, r, sqlwrap.Params{
		Key:          key,
		CacheL1Delay: r.cacheL1Delay,
		CacheL2Delay: r.cacheL2Delay,
	}, func(ctx context.Context) ([]Task, error) {
		return loadRecordCatalog(ctx, r.db.DB(), workspaceID, actionKey)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func loadRecordCatalog(ctx context.Context, db *sql.DB, workspaceID, actionKey string) ([]Task, error) {
	rows, err := db.QueryContext(ctx, listRecordCatalogQuery, workspaceID, actionKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := make([]Task, 0, 16)
	for rows.Next() {
		var (
			task             Task
			sequenceKey      sql.NullString
			sequencePosition sql.NullInt32
			actionKind       string
			claimMode        string
			resetUnit        string
			payload          json.RawMessage
			startAt          sql.NullTime
			endAt            sql.NullTime
		)
		if err := rows.Scan(
			&task.ID,
			&task.WorkspaceID,
			&task.Key,
			&task.GroupKey,
			&sequenceKey,
			&sequencePosition,
			&task.ActionKey,
			&actionKind,
			&claimMode,
			&task.TargetCount,
			&resetUnit,
			&task.ResetEvery,
			&payload,
			&task.Position,
			&startAt,
			&endAt,
		); err != nil {
			return nil, err
		}
		task.SequenceKey = ptrString(sequenceKey)
		task.SequencePosition = ptrUint32(sequencePosition)
		task.ActionKind = actionKind
		task.ClaimMode = claimMode
		task.ResetUnit = resetUnit
		task.Payload = payload
		task.StartAt = ptrTime(startAt)
		task.EndAt = ptrTime(endAt)
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *Repository) claimCatalogByID(ctx context.Context, workspaceID string, id uint64) (Task, error) {
	key := claimCatalogByIDCacheKey(workspaceID, id)
	rememberTaskCacheKey(workspaceID, key)
	out, err := repositoryQuery[Task](ctx, r, sqlwrap.Params{
		Key:          key,
		CacheL1Delay: r.cacheL1Delay,
		CacheL2Delay: r.cacheL2Delay,
	}, func(ctx context.Context) (Task, error) {
		return loadClaimCatalog(ctx, r.db.DB(), claimCatalogByIDQuery, workspaceID, id)
	})
	if err != nil {
		return Task{}, err
	}
	return out, nil
}

func (r *Repository) claimCatalogByKey(ctx context.Context, workspaceID, taskKey string) (Task, error) {
	key := claimCatalogByKeyCacheKey(workspaceID, taskKey)
	rememberTaskCacheKey(workspaceID, key)
	out, err := repositoryQuery[Task](ctx, r, sqlwrap.Params{
		Key:          key,
		CacheL1Delay: r.cacheL1Delay,
		CacheL2Delay: r.cacheL2Delay,
	}, func(ctx context.Context) (Task, error) {
		return loadClaimCatalog(ctx, r.db.DB(), claimCatalogByKeyQuery, workspaceID, taskKey)
	})
	if err != nil {
		return Task{}, err
	}
	return out, nil
}

func loadClaimCatalog(ctx context.Context, db *sql.DB, query string, args ...any) (Task, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return Task{}, err
	}
	defer rows.Close()
	var task Task
	found := false
	for rows.Next() {
		var (
			sequenceKey      sql.NullString
			sequencePosition sql.NullInt32
			actionKind       string
			claimMode        string
			payload          json.RawMessage
			imageURL         sql.NullString
			rewardID         sql.NullInt64
			rewardKey        sql.NullString
			rewardType       sql.NullString
			rewardQuantity   sql.NullInt64
			rewardUnit       sql.NullString
		)
		if err := rows.Scan(
			&task.ID,
			&task.WorkspaceID,
			&task.Key,
			&task.GroupKey,
			&sequenceKey,
			&sequencePosition,
			&task.ActionKey,
			&actionKind,
			&claimMode,
			&task.TargetCount,
			&payload,
			&imageURL,
			&rewardID,
			&rewardKey,
			&rewardType,
			&rewardQuantity,
			&rewardUnit,
		); err != nil {
			return Task{}, err
		}
		if !found {
			task.SequenceKey = ptrString(sequenceKey)
			task.SequencePosition = ptrUint32(sequencePosition)
			task.ActionKind = actionKind
			task.ClaimMode = claimMode
			task.Payload = payload
			task.ImageURL = ptrString(imageURL)
			task.Rewards = make([]Reward, 0, 1)
			found = true
		}
		if rewardID.Valid {
			task.Rewards = append(task.Rewards, Reward{
				Key: rewardKey.String, Type: rewardType.String,
				Quantity: rewardQuantity.Int64, Unit: ptrString(rewardUnit),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return Task{}, err
	}
	if !found {
		return Task{}, sql.ErrNoRows
	}
	return task, nil
}

func (r *Repository) rewardsCatalog(ctx context.Context, workspaceID string, taskID uint64) ([]Reward, error) {
	key := rewardsCatalogCacheKey(workspaceID, taskID)
	rememberTaskCacheKey(workspaceID, key)
	out, err := repositoryQuery[[]Reward](ctx, r, sqlwrap.Params{
		Key:          key,
		CacheL1Delay: r.cacheL1Delay,
		CacheL2Delay: r.cacheL2Delay,
	}, func(ctx context.Context) ([]Reward, error) {
		return loadRewardsCatalog(ctx, r.db.DB(), workspaceID, taskID)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func loadRewardsCatalog(ctx context.Context, db *sql.DB, workspaceID string, taskID uint64) ([]Reward, error) {
	rows, err := db.QueryContext(ctx, listRewardsCatalogQuery, workspaceID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rewards := make([]Reward, 0, 1)
	for rows.Next() {
		var reward Reward
		var unit sql.NullString
		if err := rows.Scan(&reward.Key, &reward.Type, &reward.Quantity, &unit); err != nil {
			return nil, err
		}
		reward.Unit = ptrString(unit)
		rewards = append(rewards, reward)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rewards, nil
}

func (r *Repository) nextSequenceTask(ctx context.Context, workspaceID, sequenceKey string, sequencePosition uint32) (nextSequenceTask, error) {
	key := nextSequenceTaskCacheKey(workspaceID, sequenceKey, sequencePosition)
	rememberTaskCacheKey(workspaceID, key)
	out, err := repositoryQuery[nextSequenceTask](ctx, r, sqlwrap.Params{
		Key:          key,
		CacheL1Delay: r.cacheL1Delay,
		CacheL2Delay: r.cacheL2Delay,
	}, func(ctx context.Context) (nextSequenceTask, error) {
		var id uint64
		err := r.db.QueryRowContext(ctx, nextSequenceTaskQuery, workspaceID, sequenceKey, sequencePosition).Scan(&id)
		if err != nil {
			if isNoRows(err) {
				return nextSequenceTask{}, nil
			}
			return nextSequenceTask{}, err
		}
		return nextSequenceTask{ID: id, Exists: true}, nil
	})
	if err != nil {
		return nextSequenceTask{}, err
	}
	return out, nil
}

func taskVisibleAt(task Task, now time.Time) bool {
	return (task.StartAt == nil || !task.StartAt.After(now)) && (task.EndAt == nil || task.EndAt.After(now))
}

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
	"github.com/go-sql-driver/mysql"
)

func (r *Repository) Record(ctx context.Context, params RecordParams) (RecordResult, error) {
	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	amount := params.Amount
	if amount == 0 {
		amount = 1
	}
	if params.ExternalEventKey != "" {
		count, err := repositoryValue[int64](ctx, r, func(ctx context.Context) (int64, error) {
			return r.q.CountProgressEventsByExternalKey(ctx, tasksqlc.CountProgressEventsByExternalKeyParams{
				WorkspaceID:      params.Identity.WorkspaceID,
				AppID:            params.Identity.AppID,
				PlatformID:       params.Identity.PlatformID,
				PlatformUserID:   params.Identity.PlatformUserID,
				Source:           params.Source,
				ExternalEventKey: params.ExternalEventKey,
			})
		})
		if err != nil {
			return RecordResult{}, err
		}
		if count > 0 {
			return RecordResult{Status: RecordStatusDuplicate, Remaining: amount}, nil
		}
	}
	for attempt := 0; attempt < 3; attempt++ {
		result := RecordResult{Status: RecordStatusNoTasks, Remaining: amount}
		err := r.recordInTx(ctx, params, now, amount, &result)
		if errors.Is(err, errRecordDuplicateEvent) {
			return result, nil
		}
		if isRetryableTxError(err) && attempt < 2 {
			continue
		}
		return result, err
	}
	return RecordResult{Status: RecordStatusNoTasks, Remaining: amount}, nil
}

var errRecordDuplicateEvent = errors.New("tasks: duplicate record event")

const listSequenceStatesForUserQuery = `
SELECT sequence_key, current_task_id
FROM task_sequence_state
WHERE workspace_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND status = 'active'`

const currentProgressForUpdateQuery = `
SELECT id, workspace_id, task_id, app_id, platform_id, platform_user_id,
       period_start_at, period_end_at, progress, status, ready_at, claimed_at,
       operation_id, COALESCE(rewards_snapshot, JSON_ARRAY()) AS rewards_snapshot, created_at, updated_at
FROM task_progress
WHERE workspace_id = ?
  AND task_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND period_start_at <= ?
  AND period_end_at > ?
LIMIT 1
FOR UPDATE`

func (r *Repository) sequenceStatesForUser(ctx context.Context, identity Identity) (map[string]uint64, error) {
	return repositoryValue[map[string]uint64](ctx, r, func(ctx context.Context) (map[string]uint64, error) {
		rows, err := r.executor.QueryContext(ctx, listSequenceStatesForUserQuery,
			identity.WorkspaceID,
			identity.AppID,
			identity.PlatformID,
			identity.PlatformUserID,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		result := make(map[string]uint64)
		for rows.Next() {
			var (
				sequenceKey   string
				currentTaskID sql.NullInt64
			)
			if err := rows.Scan(&sequenceKey, &currentTaskID); err != nil {
				return nil, err
			}
			if currentTaskID.Valid {
				result[sequenceKey] = uint64(currentTaskID.Int64)
			} else {
				result[sequenceKey] = 0
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return result, nil
	})
}

func (r *Repository) currentProgressForUpdate(ctx context.Context, identity Identity, taskID uint64, now time.Time) (Progress, error) {
	return repositoryValue[Progress](ctx, r, func(ctx context.Context) (Progress, error) {
		var row tasksqlc.TaskProgress
		err := r.executor.QueryRowContext(ctx, currentProgressForUpdateQuery,
			identity.WorkspaceID,
			taskID,
			identity.AppID,
			identity.PlatformID,
			identity.PlatformUserID,
			now,
			now,
		).Scan(
			&row.ID,
			&row.WorkspaceID,
			&row.TaskID,
			&row.AppID,
			&row.PlatformID,
			&row.PlatformUserID,
			&row.PeriodStartAt,
			&row.PeriodEndAt,
			&row.Progress,
			&row.Status,
			&row.ReadyAt,
			&row.ClaimedAt,
			&row.OperationID,
			&row.RewardsSnapshot,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return Progress{}, err
		}
		return mapProgress(row), nil
	})
}

func (r *Repository) recordInTx(ctx context.Context, params RecordParams, now time.Time, amount uint64, result *RecordResult) error {
	return r.WithTx(ctx, func(txRepo *Repository) error {
		catalog, err := txRepo.listRecordCatalog(ctx, params.Identity.WorkspaceID, params.ActionKey)
		if err != nil {
			return err
		}
		if len(catalog) == 0 {
			return nil
		}
		sequenceStates, err := txRepo.sequenceStatesForUser(ctx, params.Identity)
		if err != nil {
			return err
		}
		progressRows, err := repositoryValue[[]tasksqlc.TaskProgress](ctx, txRepo, func(ctx context.Context) ([]tasksqlc.TaskProgress, error) {
			return txRepo.q.ListCurrentProgressForUserForUpdate(ctx, tasksqlc.ListCurrentProgressForUserForUpdateParams{
				WorkspaceID: params.Identity.WorkspaceID,
				AppID:       params.Identity.AppID, PlatformID: params.Identity.PlatformID, PlatformUserID: params.Identity.PlatformUserID,
				PeriodStartAt: now, PeriodEndAt: now,
			})
		})
		if err != nil {
			return err
		}
		progressByTask := make(map[uint64]Progress, len(progressRows))
		for _, row := range progressRows {
			progressByTask[row.TaskID] = mapProgress(row)
		}
		branches := make(map[string]struct{})
		progressUpserts := make([]recordProgressUpsert, 0, len(catalog))
		autoClaims := make([]recordAutoClaim, 0)
		shouldInsertEvent := false
		var totalConsumed uint64
		var maxConsumed uint64
		for _, task := range catalog {
			if !taskVisibleAt(task, now) {
				continue
			}
			branch := branchKey(task)
			if _, done := branches[branch]; done {
				continue
			}
			periodStart, periodEnd := periodFor(task, now)
			progress, exists := progressByTask[task.ID]
			if task.SequenceKey != nil {
				if exists && progress.Status == StatusClaimed {
					continue
				}
				currentTaskID, hasState := sequenceStates[*task.SequenceKey]
				if hasState {
					if currentTaskID != task.ID {
						continue
					}
				} else if task.SequencePosition == nil || *task.SequencePosition != 1 {
					continue
				}
				branches[branch] = struct{}{}
			} else if task.ActionKey != params.ActionKey {
				continue
			} else {
				branches[branch] = struct{}{}
			}
			if task.ActionKey != params.ActionKey {
				continue
			}
			if params.ExternalEventKey != "" {
				shouldInsertEvent = true
			}
			if progress.Status == StatusClaimed || progress.Status == StatusReady {
				continue
			}
			before := progress.Progress
			need := task.TargetCount - progress.Progress
			consume := amount
			if consume > need {
				consume = need
			}
			progress.Progress += consume
			claimed := false
			if progress.Progress >= task.TargetCount {
				if task.ClaimMode == ClaimModeAuto {
					autoClaims = append(autoClaims, recordAutoClaim{
						task: task, progress: progress, exists: exists,
						periodStartAt: periodStart, periodEndAt: periodEnd,
					})
					claimed = true
				} else {
					progress.Status = StatusReady
					progress.ReadyAt = &now
					progressUpserts = append(progressUpserts, recordProgressUpsert{
						taskID: task.ID, periodStartAt: periodStart, periodEndAt: periodEnd,
						progress: progress.Progress, status: progress.Status, readyAt: progress.ReadyAt,
					})
				}
			} else {
				progressUpserts = append(progressUpserts, recordProgressUpsert{
					taskID: task.ID, periodStartAt: periodStart, periodEndAt: periodEnd,
					progress: progress.Progress, status: StatusOpen,
				})
			}
			result.Status = RecordStatusRecorded
			totalConsumed += consume
			if consume > maxConsumed {
				maxConsumed = consume
			}
			result.Tasks = append(result.Tasks, TaskResult{
				Task: task, Before: before, After: progress.Progress, Consumed: consume, Claimed: claimed,
			})
		}
		if shouldInsertEvent {
			eventPayload := params.Payload
			if len(eventPayload) == 0 {
				eventPayload = []byte("{}")
			}
			affected, err := repositoryValue[int64](ctx, txRepo, func(ctx context.Context) (int64, error) {
				return txRepo.q.InsertProgressEvent(ctx, tasksqlc.InsertProgressEventParams{
					WorkspaceID:      params.Identity.WorkspaceID,
					AppID:            params.Identity.AppID,
					PlatformID:       params.Identity.PlatformID,
					PlatformUserID:   params.Identity.PlatformUserID,
					Source:           params.Source,
					ExternalEventKey: params.ExternalEventKey,
					ActionKey:        params.ActionKey,
					Amount:           amount,
					Payload:          eventPayload,
				})
			})
			if err != nil {
				return err
			}
			if affected != 1 {
				*result = RecordResult{Status: RecordStatusDuplicate, Remaining: amount}
				return errRecordDuplicateEvent
			}
		}
		if _, err := txRepo.batchUpsertProgress(ctx, params.Identity, progressUpserts); err != nil {
			return err
		}
		for _, item := range autoClaims {
			progress := item.progress
			if !item.exists {
				progress, err = txRepo.ensureProgress(ctx, params.Identity, item.task, item.periodStartAt, item.periodEndAt)
				if err != nil {
					return err
				}
				progress.Progress = item.progress.Progress
			}
			if err := txRepo.claimProgress(ctx, params.Identity, &item.task, &progress, autoOperationID(params.ExternalEventKey, item.task.ID), now); err != nil {
				return err
			}
		}
		if len(result.Tasks) == 0 {
			return nil
		}
		result.Consumed = totalConsumed
		result.Remaining = amount - maxConsumed
		return nil
	})
}

func isRetryableTxError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	return mysqlErr.Number == 1213 || mysqlErr.Number == 1205
}

func branchKey(task Task) string {
	if task.SequenceKey != nil {
		return "sequence:" + *task.SequenceKey
	}
	return fmt.Sprintf("task:%d", task.ID)
}

func (r *Repository) Claim(ctx context.Context, params ClaimParams) (ClaimResult, error) {
	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result := ClaimResult{Status: ClaimStatusNotFound}
	err := r.WithTx(ctx, func(txRepo *Repository) error {
		id, key := taskRef(params.TaskRef)
		var task Task
		var err error
		if id != 0 {
			task, err = txRepo.claimCatalogByID(ctx, params.Identity.WorkspaceID, id)
		} else {
			task, err = txRepo.claimCatalogByKey(ctx, params.Identity.WorkspaceID, key)
		}
		if err != nil {
			if isNoRows(err) {
				return nil
			}
			return err
		}
		result.Task = &task
		progress, err := txRepo.currentProgressForUpdate(ctx, params.Identity, task.ID, now)
		if err != nil {
			if isNoRows(err) {
				result.Status = ClaimStatusNotReady
				return nil
			}
			return err
		}
		task.Progress = &progress
		result.Task = &task
		if task.Progress == nil {
			result.Status = ClaimStatusNotReady
			return nil
		}
		switch task.Progress.Status {
		case StatusClaimed:
			result.Status = ClaimStatusAlreadyDone
			return nil
		case StatusReady:
			if err := txRepo.claimProgress(ctx, params.Identity, &task, task.Progress, params.OperationID, now); err != nil {
				return err
			}
			result.Task = &task
			result.Status = ClaimStatusClaimed
			return nil
		default:
			result.Status = ClaimStatusNotReady
			return nil
		}
	})
	return result, err
}

func (r *Repository) ensureProgress(ctx context.Context, identity Identity, task Task, start, end time.Time) (Progress, error) {
	id, err := repositoryValue[int64](ctx, r, func(ctx context.Context) (int64, error) {
		return r.q.EnsureProgress(ctx, tasksqlc.EnsureProgressParams{
			WorkspaceID: identity.WorkspaceID, TaskID: task.ID, AppID: identity.AppID,
			PlatformID: identity.PlatformID, PlatformUserID: identity.PlatformUserID,
			PeriodStartAt: start, PeriodEndAt: end,
		})
	})
	if err != nil {
		return Progress{}, err
	}
	return Progress{
		ID: uint64(id), Progress: 0, Status: StatusOpen,
		PeriodStartAt: start, PeriodEndAt: end, Rewards: make([]Reward, 0),
	}, nil
}

func (r *Repository) saveProgress(ctx context.Context, progress Progress) error {
	_, err := repositoryValue[int64](ctx, r, func(ctx context.Context) (int64, error) {
		return r.q.UpdateProgress(ctx, tasksqlc.UpdateProgressParams{
			Progress: progress.Progress, Status: tasksqlc.TaskProgressStatus(progress.Status),
			ReadyAt: nullTime(progress.ReadyAt), ClaimedAt: nullTime(progress.ClaimedAt),
			OperationID: nullString(progress.OperationID), RewardsSnapshot: rewardsSnapshot(progress.Rewards),
			ID: progress.ID,
		})
	})
	return err
}

func (r *Repository) claimProgress(ctx context.Context, identity Identity, task *Task, progress *Progress, operationID string, now time.Time) error {
	rewards := task.Rewards
	if rewards == nil {
		var err error
		rewards, err = r.rewards(ctx, task.WorkspaceID, task.ID)
		if err != nil {
			return err
		}
	}
	progress.Status = StatusClaimed
	progress.ClaimedAt = &now
	progress.OperationID = &operationID
	progress.Rewards = rewards
	if err := r.saveProgress(ctx, *progress); err != nil {
		return err
	}
	if err := r.advanceSequenceState(ctx, identity, task); err != nil {
		return err
	}
	task.Rewards = rewards
	payload, err := json.Marshal(CallbackPayload{
		WorkspaceID: identity.WorkspaceID, AppID: identity.AppID, PlatformID: identity.PlatformID,
		PlatformUserID: identity.PlatformUserID, TaskID: task.ID, TaskKey: task.Key,
		OperationID: operationID, PeriodStartAt: progress.PeriodStartAt,
		PeriodEndAt: progress.PeriodEndAt, Rewards: rewards, Payload: task.Payload,
	})
	if err != nil {
		return err
	}
	eventKey := fmt.Sprintf("tasks.claimed:%d", progress.ID)
	_, err = repositoryValue[uint64](ctx, r, func(ctx context.Context) (uint64, error) {
		return r.callbacks.CreateEvent(ctx, callbackutil.CreateParams{
			SourceService: "tasks", EventType: CallbackEventClaimed,
			EventKey: eventKey, IdempotencyKey: eventKey,
			Payload: payload, NextAttemptAt: now,
		})
	})
	return err
}

func (r *Repository) advanceSequenceState(ctx context.Context, identity Identity, task *Task) error {
	if task.SequenceKey == nil || task.SequencePosition == nil {
		return nil
	}
	next, err := r.nextSequenceTask(ctx, task.WorkspaceID, *task.SequenceKey, *task.SequencePosition)
	status := tasksqlc.TaskSequenceStateStatusActive
	currentTaskID := sql.NullInt64{}
	if err != nil {
		return err
	}
	if !next.Exists {
		status = tasksqlc.TaskSequenceStateStatusCompleted
	} else {
		currentTaskID = sql.NullInt64{Int64: int64(next.ID), Valid: true}
	}
	return repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.UpsertSequenceState(ctx, tasksqlc.UpsertSequenceStateParams{
			WorkspaceID: task.WorkspaceID, SequenceKey: *task.SequenceKey,
			AppID: identity.AppID, PlatformID: identity.PlatformID, PlatformUserID: identity.PlatformUserID,
			CurrentTaskID: currentTaskID, Status: status,
		})
	})
}

func (r *Repository) rewards(ctx context.Context, workspaceID string, taskID uint64) ([]Reward, error) {
	return r.rewardsCatalog(ctx, workspaceID, taskID)
}

func rewardsSnapshot(rewards []Reward) json.RawMessage {
	if rewards == nil {
		return nil
	}
	raw, _ := json.Marshal(rewards)
	return raw
}

func autoOperationID(eventKey string, taskID uint64) string {
	if eventKey != "" {
		return eventKey
	}
	return fmt.Sprintf("auto-%d-%d", taskID, time.Now().UnixNano())
}

package repository

import (
	"context"
	"database/sql"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
)

func (r *Repository) UpsertGroup(ctx context.Context, workspaceID, key string, position int32, active bool) error {
	if err := repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.AdminUpsertGroup(ctx, tasksqlc.AdminUpsertGroupParams{
			WorkspaceID: workspaceID, Key: key, Position: position, IsActive: active,
		})
	}); err != nil {
		return err
	}
	return r.invalidateTaskCache(ctx, workspaceID)
}

func (r *Repository) UpsertGroupLocalization(ctx context.Context, workspaceID, key, locale, title, description string) error {
	if err := repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.AdminUpsertGroupLocalization(ctx, tasksqlc.AdminUpsertGroupLocalizationParams{
			WorkspaceID: workspaceID, GroupKey: key, Locale: locale, Title: title, Description: description,
		})
	}); err != nil {
		return err
	}
	return r.invalidateTaskCache(ctx, workspaceID)
}

func (r *Repository) UpsertSequence(ctx context.Context, workspaceID, key string, position int32, active bool) error {
	if err := repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.AdminUpsertSequence(ctx, tasksqlc.AdminUpsertSequenceParams{
			WorkspaceID: workspaceID, Key: key, Position: position, IsActive: active,
		})
	}); err != nil {
		return err
	}
	return r.invalidateTaskCache(ctx, workspaceID)
}

func (r *Repository) SaveTask(ctx context.Context, params SaveTaskParams) (uint64, error) {
	taskKind := params.TaskKind
	if taskKind == "" {
		taskKind = TaskKindInternal
	}
	startMode := params.StartMode
	if startMode == "" {
		startMode = StartModeNone
	}
	payload := params.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	target := params.Target
	if len(target) == 0 {
		target = []byte("null")
	}
	integrationPayload := params.IntegrationPayload
	if len(integrationPayload) == 0 {
		integrationPayload = []byte("null")
	}
	if params.ID == 0 {
		id, err := repositoryValue(ctx, r, func(ctx context.Context) (int64, error) {
			return r.q.AdminCreateTask(ctx, tasksqlc.AdminCreateTaskParams{
				WorkspaceID: params.WorkspaceID, Key: params.Key, GroupKey: params.GroupKey,
				SequenceKey: nullString(params.SequenceKey), SequencePosition: nullInt32FromUint32(params.SequencePosition),
				TaskKind:  taskKind,
				ActionKey: params.ActionKey, ActionKind: tasksqlc.TaskDefinitionActionKind(params.ActionKind),
				ClaimMode: tasksqlc.TaskDefinitionClaimMode(params.ClaimMode), StartMode: tasksqlc.TaskDefinitionStartMode(startMode), TargetCount: params.TargetCount,
				ResetUnit: tasksqlc.TaskDefinitionResetUnit(params.ResetUnit), ResetEvery: params.ResetEvery,
				Position: params.Position, Payload: payload, Target: target, IntegrationKind: nullString(params.IntegrationKind),
				IntegrationProvider: nullString(params.IntegrationProvider), IntegrationPayload: integrationPayload,
				ImageUrl:  nullString(params.ImageURL),
				IsVisible: params.IsVisible, IsActive: params.IsActive,
				StartAt: nullTime(params.StartAt), EndAt: nullTime(params.EndAt),
			})
		})
		if err != nil {
			return 0, err
		}
		return uint64(id), r.invalidateTaskCache(ctx, params.WorkspaceID)
	}
	_, err := repositoryValue(ctx, r, func(ctx context.Context) (int64, error) {
		return r.q.AdminUpdateTask(ctx, tasksqlc.AdminUpdateTaskParams{
			GroupKey: params.GroupKey, SequenceKey: nullString(params.SequenceKey),
			SequencePosition: nullInt32FromUint32(params.SequencePosition), TaskKind: taskKind, ActionKey: params.ActionKey,
			ActionKind: tasksqlc.TaskDefinitionActionKind(params.ActionKind),
			ClaimMode:  tasksqlc.TaskDefinitionClaimMode(params.ClaimMode), StartMode: tasksqlc.TaskDefinitionStartMode(startMode), TargetCount: params.TargetCount,
			ResetUnit: tasksqlc.TaskDefinitionResetUnit(params.ResetUnit), ResetEvery: params.ResetEvery,
			Position: params.Position, Payload: payload, Target: target, IntegrationKind: nullString(params.IntegrationKind),
			IntegrationProvider: nullString(params.IntegrationProvider), IntegrationPayload: integrationPayload,
			ImageUrl:  nullString(params.ImageURL),
			IsVisible: params.IsVisible, IsActive: params.IsActive,
			StartAt: nullTime(params.StartAt), EndAt: nullTime(params.EndAt),
			WorkspaceID: params.WorkspaceID, ID: params.ID,
		})
	})
	if err != nil {
		return 0, err
	}
	return params.ID, r.invalidateTaskCache(ctx, params.WorkspaceID)
}

func (r *Repository) DeleteTask(ctx context.Context, workspaceID string, id uint64) (int64, error) {
	rows, err := repositoryValue(ctx, r, func(ctx context.Context) (int64, error) {
		return r.q.AdminDeleteTask(ctx, tasksqlc.AdminDeleteTaskParams{WorkspaceID: workspaceID, ID: id})
	})
	if err != nil {
		return 0, err
	}
	if rows > 0 {
		return rows, r.invalidateTaskCache(ctx, workspaceID)
	}
	return rows, nil
}

func (r *Repository) GetTask(ctx context.Context, workspaceID string, id uint64) (Task, error) {
	key := adminGetTaskCacheKey(workspaceID, id)
	rememberTaskCacheKey(workspaceID, key)
	out, err := repositoryQuery[Task](ctx, r, sqlwrap.Params{
		Key:          key,
		CacheL1Delay: r.cacheL1Delay,
		CacheL2Delay: r.cacheL2Delay,
	}, func(ctx context.Context) (Task, error) {
		row, err := tasksqlc.New(r.db.DB()).AdminGetTask(ctx, tasksqlc.AdminGetTaskParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			return Task{}, err
		}
		return mapTask(row), nil
	})
	if err != nil {
		return Task{}, err
	}
	return out, nil
}

func (r *Repository) ListTasks(ctx context.Context, workspaceID, groupKey string, limit, offset int32) ([]Task, error) {
	limit, offset = normalizePage(limit, offset)
	key := adminListTasksCacheKey(workspaceID, groupKey, limit, offset)
	rememberTaskCacheKey(workspaceID, key)
	out, err := repositoryQuery[[]Task](ctx, r, sqlwrap.Params{
		Key:          key,
		CacheL1Delay: r.cacheL1Delay,
		CacheL2Delay: r.cacheL2Delay,
	}, func(ctx context.Context) ([]Task, error) {
		q := tasksqlc.New(r.db.DB())
		var result []Task
		if groupKey != "" {
			rows, err := q.AdminListTasksByGroup(ctx, tasksqlc.AdminListTasksByGroupParams{
				WorkspaceID: workspaceID, GroupKey: groupKey, Limit: limit, Offset: offset,
			})
			if err != nil {
				return nil, err
			}
			result = make([]Task, 0, len(rows))
			for _, row := range rows {
				result = append(result, mapTask(tasksqlc.TaskDefinition(row)))
			}
			return result, nil
		}
		rows, err := q.AdminListTasks(ctx, tasksqlc.AdminListTasksParams{
			WorkspaceID: workspaceID, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		result = make([]Task, 0, len(rows))
		for _, row := range rows {
			result = append(result, mapTask(tasksqlc.TaskDefinition(row)))
		}
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) UpsertTaskLocalization(ctx context.Context, workspaceID string, taskID uint64, locale, title, description string) error {
	if err := repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.AdminUpsertTaskLocalization(ctx, tasksqlc.AdminUpsertTaskLocalizationParams{
			WorkspaceID: workspaceID, TaskID: taskID, Locale: locale, Title: title, Description: description,
		})
	}); err != nil {
		return err
	}
	return r.invalidateTaskCache(ctx, workspaceID)
}

func (r *Repository) UpsertReward(ctx context.Context, workspaceID string, taskID uint64, reward Reward, position int32) error {
	if err := repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.AdminUpsertReward(ctx, tasksqlc.AdminUpsertRewardParams{
			WorkspaceID: workspaceID, TaskID: taskID, RewardKey: reward.Key,
			RewardType: tasksqlc.TaskRewardRewardType(reward.Type),
			Quantity:   reward.Quantity,
			Scale:      reward.Scale,
			DurationUnit: tasksqlc.NullTaskRewardDurationUnit{
				TaskRewardDurationUnit: tasksqlc.TaskRewardDurationUnit(taskStringValue(reward.Unit)),
				Valid:                  reward.Unit != nil,
			},
			Position: position,
		})
	}); err != nil {
		return err
	}
	return r.invalidateTaskCache(ctx, workspaceID)
}

func (r *Repository) DeleteReward(ctx context.Context, workspaceID string, taskID uint64, key string) (int64, error) {
	rows, err := repositoryValue(ctx, r, func(ctx context.Context) (int64, error) {
		return r.q.AdminDeleteReward(ctx, tasksqlc.AdminDeleteRewardParams{
			WorkspaceID: workspaceID, TaskID: taskID, RewardKey: key,
		})
	})
	if err != nil {
		return 0, err
	}
	if rows > 0 {
		return rows, r.invalidateTaskCache(ctx, workspaceID)
	}
	return rows, nil
}

func (r *Repository) UpsertComplexCondition(ctx context.Context, params SaveComplexConditionParams) error {
	requiredStatus := params.RequiredStatus
	if requiredStatus == "" {
		requiredStatus = ComplexRequiredStatusReady
	}
	if err := repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.AdminUpsertComplexCondition(ctx, tasksqlc.AdminUpsertComplexConditionParams{
			WorkspaceID:     params.WorkspaceID,
			ParentTaskID:    params.ParentTaskID,
			ConditionTaskID: params.ConditionTaskID,
			RequiredStatus:  tasksqlc.TaskComplexConditionRequiredStatus(requiredStatus),
			Position:        params.Position,
			IsRequired:      params.IsRequired,
		})
	}); err != nil {
		return err
	}
	return r.invalidateTaskCache(ctx, params.WorkspaceID)
}

func (r *Repository) DeleteComplexCondition(ctx context.Context, workspaceID string, parentTaskID uint64, conditionTaskID uint64) (int64, error) {
	rows, err := repositoryValue(ctx, r, func(ctx context.Context) (int64, error) {
		return r.q.AdminDeleteComplexCondition(ctx, tasksqlc.AdminDeleteComplexConditionParams{
			WorkspaceID: workspaceID, ParentTaskID: parentTaskID, ConditionTaskID: conditionTaskID,
		})
	})
	if err != nil {
		return 0, err
	}
	if rows > 0 {
		return rows, r.invalidateTaskCache(ctx, workspaceID)
	}
	return rows, nil
}

func (r *Repository) ListComplexConditions(ctx context.Context, workspaceID string) ([]ComplexCondition, error) {
	rows, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskComplexCondition, error) {
		return r.q.AdminListComplexConditions(ctx, workspaceID)
	})
	if err != nil {
		return nil, err
	}
	out := make([]ComplexCondition, 0, len(rows))
	for _, row := range rows {
		out = append(out, ComplexCondition{
			WorkspaceID:     row.WorkspaceID,
			ParentTaskID:    row.ParentTaskID,
			ConditionTaskID: row.ConditionTaskID,
			RequiredStatus:  string(row.RequiredStatus),
			Position:        row.Position,
			IsRequired:      row.IsRequired,
		})
	}
	return out, nil
}

func sqlNullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

package repository

import (
	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
)

func mapActionTask(row tasksqlc.ListRecordTasksRow) Task {
	return Task{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Key: row.Key, GroupKey: row.GroupKey,
		SequenceKey: ptrString(row.SequenceKey), SequencePosition: ptrUint32(row.SequencePosition),
		TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind), ClaimMode: string(row.ClaimMode),
		TargetCount: row.TargetCount, ResetUnit: string(row.ResetUnit), ResetEvery: row.ResetEvery,
		Position: row.Position, Payload: row.Payload, Target: row.Target, Rewards: make([]Reward, 0),
	}
}

func mapClaimTaskByID(rows []tasksqlc.GetClaimBundleByIDForUpdateRow) Task {
	row := rows[0]
	task := Task{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Key: row.Key, GroupKey: row.GroupKey,
		SequenceKey: ptrString(row.SequenceKey), SequencePosition: ptrUint32(row.SequencePosition),
		TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind), ClaimMode: string(row.ClaimMode),
		TargetCount: row.TargetCount, Payload: row.Payload, Target: row.Target, ImageURL: ptrString(row.ImageUrl),
		IntegrationKind: ptrString(row.IntegrationKind), IntegrationProvider: ptrString(row.IntegrationProvider),
		IntegrationPayload: row.IntegrationPayload,
		Rewards:            make([]Reward, 0, len(rows)),
	}
	for _, item := range rows {
		if item.RewardID.Valid {
			task.Rewards = append(task.Rewards, Reward{
				Key: item.RewardKey.String, Type: string(item.RewardType.TaskRewardRewardType),
				Quantity: item.RewardQuantity.Int64, Unit: taskDurationUnitPtr(item.DurationUnit),
			})
		}
	}
	if row.ProgressID.Valid {
		task.Progress = &Progress{
			ID: uint64(row.ProgressID.Int64), Progress: uint64(row.Progress.Int64),
			Status: string(row.Status.TaskProgressStatus), PeriodStartAt: row.PeriodStartAt.Time,
			PeriodEndAt: row.PeriodEndAt.Time, ReadyAt: ptrTime(row.ReadyAt),
			ClaimedAt: ptrTime(row.ClaimedAt), OperationID: ptrString(row.OperationID),
			Rewards: decodeRewards(row.RewardsSnapshot),
		}
	}
	return task
}

func mapClaimTaskByKey(rows []tasksqlc.GetClaimBundleByKeyForUpdateRow) Task {
	row := rows[0]
	task := Task{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Key: row.Key, GroupKey: row.GroupKey,
		SequenceKey: ptrString(row.SequenceKey), SequencePosition: ptrUint32(row.SequencePosition),
		TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind), ClaimMode: string(row.ClaimMode),
		TargetCount: row.TargetCount, Payload: row.Payload, Target: row.Target, ImageURL: ptrString(row.ImageUrl),
		IntegrationKind: ptrString(row.IntegrationKind), IntegrationProvider: ptrString(row.IntegrationProvider),
		IntegrationPayload: row.IntegrationPayload,
		Rewards:            make([]Reward, 0, len(rows)),
	}
	for _, item := range rows {
		if item.RewardID.Valid {
			task.Rewards = append(task.Rewards, Reward{
				Key: item.RewardKey.String, Type: string(item.RewardType.TaskRewardRewardType),
				Quantity: item.RewardQuantity.Int64, Unit: taskDurationUnitPtr(item.DurationUnit),
			})
		}
	}
	if row.ProgressID.Valid {
		task.Progress = &Progress{
			ID: uint64(row.ProgressID.Int64), Progress: uint64(row.Progress.Int64),
			Status: string(row.Status.TaskProgressStatus), PeriodStartAt: row.PeriodStartAt.Time,
			PeriodEndAt: row.PeriodEndAt.Time, ReadyAt: ptrTime(row.ReadyAt),
			ClaimedAt: ptrTime(row.ClaimedAt), OperationID: ptrString(row.OperationID),
			Rewards: decodeRewards(row.RewardsSnapshot),
		}
	}
	return task
}

func mapActiveBundles(rows []tasksqlc.ListActiveTaskBundlesRow) []Task {
	result := make([]Task, 0, len(rows))
	var lastID uint64
	index := -1
	for _, row := range rows {
		if index < 0 || row.ID != lastID {
			task := Task{
				ID: row.ID, Key: row.Key, GroupKey: row.GroupKey,
				TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind), ClaimMode: string(row.ClaimMode),
				TargetCount: row.TargetCount, Payload: row.Payload, Target: row.Target, ImageURL: ptrString(row.ImageUrl),
				StartAt: ptrTime(row.StartAt), EndAt: ptrTime(row.EndAt),
				Rewards: make([]Reward, 0),
			}
			if row.Locale.Valid {
				task.Localization = &Localization{Locale: row.Locale.String, Title: row.Title.String, Description: row.Description.String}
			}
			result = append(result, task)
			index = len(result) - 1
			lastID = row.ID
		}
		if row.RewardID.Valid {
			result[index].Rewards = append(result[index].Rewards, Reward{
				Key: row.RewardKey.String, Type: string(row.RewardType.TaskRewardRewardType),
				Quantity: row.RewardQuantity.Int64, Unit: taskDurationUnitPtr(row.DurationUnit),
			})
		}
	}
	return result
}

func mapRecordCatalogTask(row tasksqlc.ListRecordCatalogRow) Task {
	return Task{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Key: row.Key, GroupKey: row.GroupKey,
		SequenceKey: ptrString(row.SequenceKey), SequencePosition: ptrUint32(row.SequencePosition),
		TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind),
		ClaimMode: string(row.ClaimMode), TargetCount: row.TargetCount, ResetUnit: string(row.ResetUnit),
		ResetEvery: row.ResetEvery, Position: row.Position, Payload: row.Payload, Target: row.Target,
		StartAt: ptrTime(row.StartAt), EndAt: ptrTime(row.EndAt),
	}
}

func mapIntegrationCheckTaskByID(row tasksqlc.GetIntegrationCheckTaskByIDRow) Task {
	return Task{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Key: row.Key, GroupKey: row.GroupKey,
		SequenceKey: ptrString(row.SequenceKey), SequencePosition: ptrUint32(row.SequencePosition),
		TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind),
		ClaimMode: string(row.ClaimMode), TargetCount: row.TargetCount, ResetUnit: string(row.ResetUnit),
		ResetEvery: row.ResetEvery, Payload: row.Payload, Target: row.Target, IntegrationKind: ptrString(row.IntegrationKind),
		IntegrationProvider: ptrString(row.IntegrationProvider), IntegrationPayload: row.IntegrationPayload,
		ImageURL: ptrString(row.ImageUrl), StartAt: ptrTime(row.StartAt), EndAt: ptrTime(row.EndAt),
		Rewards: make([]Reward, 0),
	}
}

func mapIntegrationCheckTaskByKey(row tasksqlc.GetIntegrationCheckTaskByKeyRow) Task {
	return Task{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Key: row.Key, GroupKey: row.GroupKey,
		SequenceKey: ptrString(row.SequenceKey), SequencePosition: ptrUint32(row.SequencePosition),
		TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind),
		ClaimMode: string(row.ClaimMode), TargetCount: row.TargetCount, ResetUnit: string(row.ResetUnit),
		ResetEvery: row.ResetEvery, Payload: row.Payload, Target: row.Target, IntegrationKind: ptrString(row.IntegrationKind),
		IntegrationProvider: ptrString(row.IntegrationProvider), IntegrationPayload: row.IntegrationPayload,
		ImageURL: ptrString(row.ImageUrl), StartAt: ptrTime(row.StartAt), EndAt: ptrTime(row.EndAt),
		Rewards: make([]Reward, 0),
	}
}

func mapClaimCatalogTaskByID(rows []tasksqlc.GetClaimCatalogByIDRow) Task {
	row := rows[0]
	task := Task{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Key: row.Key, GroupKey: row.GroupKey,
		SequenceKey: ptrString(row.SequenceKey), SequencePosition: ptrUint32(row.SequencePosition),
		TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind),
		ClaimMode: string(row.ClaimMode), TargetCount: row.TargetCount, Payload: row.Payload, Target: row.Target,
		IntegrationKind: ptrString(row.IntegrationKind), IntegrationProvider: ptrString(row.IntegrationProvider),
		IntegrationPayload: row.IntegrationPayload, ImageURL: ptrString(row.ImageUrl),
		Rewards: make([]Reward, 0, len(rows)),
	}
	for _, item := range rows {
		if item.RewardID.Valid {
			task.Rewards = append(task.Rewards, Reward{
				Key: item.RewardKey.String, Type: string(item.RewardType.TaskRewardRewardType),
				Quantity: item.RewardQuantity.Int64, Unit: taskDurationUnitPtr(item.DurationUnit),
			})
		}
	}
	return task
}

func mapClaimCatalogTaskByKey(rows []tasksqlc.GetClaimCatalogByKeyRow) Task {
	row := rows[0]
	task := Task{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Key: row.Key, GroupKey: row.GroupKey,
		SequenceKey: ptrString(row.SequenceKey), SequencePosition: ptrUint32(row.SequencePosition),
		TaskKind: row.TaskKind, ActionKey: row.ActionKey, ActionKind: string(row.ActionKind),
		ClaimMode: string(row.ClaimMode), TargetCount: row.TargetCount, Payload: row.Payload, Target: row.Target,
		IntegrationKind: ptrString(row.IntegrationKind), IntegrationProvider: ptrString(row.IntegrationProvider),
		IntegrationPayload: row.IntegrationPayload, ImageURL: ptrString(row.ImageUrl),
		Rewards: make([]Reward, 0, len(rows)),
	}
	for _, item := range rows {
		if item.RewardID.Valid {
			task.Rewards = append(task.Rewards, Reward{
				Key: item.RewardKey.String, Type: string(item.RewardType.TaskRewardRewardType),
				Quantity: item.RewardQuantity.Int64, Unit: taskDurationUnitPtr(item.DurationUnit),
			})
		}
	}
	return task
}

func taskDurationUnitPtr(value tasksqlc.NullTaskRewardDurationUnit) *string {
	if !value.Valid {
		return nil
	}
	unit := string(value.TaskRewardDurationUnit)
	return &unit
}

func taskStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mapProgress(row tasksqlc.TaskProgress) Progress {
	return Progress{
		ID: row.ID, Progress: row.Progress, Status: string(row.Status),
		PeriodStartAt: row.PeriodStartAt, PeriodEndAt: row.PeriodEndAt,
		ReadyAt: ptrTime(row.ReadyAt), ClaimedAt: ptrTime(row.ClaimedAt),
		OperationID: ptrString(row.OperationID), Rewards: decodeRewards(row.RewardsSnapshot),
	}
}

func mapActiveProgress(row tasksqlc.TaskProgress) ActiveProgress {
	return ActiveProgress{
		Progress: row.Progress, Status: string(row.Status),
		PeriodStartAt: row.PeriodStartAt, PeriodEndAt: row.PeriodEndAt,
		ReadyAt: ptrTime(row.ReadyAt), ClaimedAt: ptrTime(row.ClaimedAt),
	}
}

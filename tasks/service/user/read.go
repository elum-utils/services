package user

import (
	"context"
	"time"

	"github.com/elum-utils/services/tasks/repository"
)

func (u *User) ListActive(ctx context.Context, identity Identity, locale string, now time.Time) ([]TaskModel, error) {
	mergedCtx, cancel := u.withContext(ctx)
	defer cancel()
	tasks, err := u.repository.ListActive(mergedCtx, identity, locale, now)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (u *User) Claim(ctx context.Context, params ClaimParams) (ClaimResult, error) {
	mergedCtx, cancel := u.withContext(ctx)
	defer cancel()
	result, err := u.repository.Claim(mergedCtx, repository.ClaimParams(params))
	if err != nil {
		return ClaimResult{}, err
	}
	output := ClaimResult{Status: result.Status}
	if result.Task != nil {
		task := mapTask(*result.Task)
		output.Task = &task
	}
	return output, nil
}

func mapTask(task repository.Task) TaskModel {
	result := TaskModel{
		ID: task.ID, Key: task.Key, GroupKey: task.GroupKey,
		ActionKey: task.ActionKey, ActionKind: task.ActionKind, ClaimMode: task.ClaimMode,
		TargetCount: task.TargetCount, Payload: task.Payload, ImageURL: task.ImageURL,
		Rewards: task.Rewards,
	}
	if task.Localization != nil {
		result.Title = task.Localization.Title
		result.Description = task.Localization.Description
	}
	if task.Progress != nil {
		result.Progress = &repository.ActiveProgress{
			Progress: task.Progress.Progress, Status: task.Progress.Status,
			PeriodStartAt: task.Progress.PeriodStartAt, PeriodEndAt: task.Progress.PeriodEndAt,
			ReadyAt: task.Progress.ReadyAt, ClaimedAt: task.Progress.ClaimedAt,
		}
	}
	return result
}

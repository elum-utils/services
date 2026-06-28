package user

import (
	"context"
	"time"

	"github.com/elum-utils/services/tasks/repository"
)

func (u *User) ListActive(ctx context.Context, identity Identity, locale string, now time.Time) ([]TaskGroupModel, error) {
	mergedCtx, cancel := u.withContext(ctx)
	defer cancel()
	tasks, err := u.repository.ListActive(mergedCtx, identity, locale, now)
	if err != nil {
		return nil, err
	}
	return groupTasks(tasks), nil
}

func (u *User) Claim(ctx context.Context, params ClaimParams) (ClaimResult, error) {
	mergedCtx, cancel := u.withContext(ctx)
	defer cancel()
	if issueID, ok := repository.ParsePartnerIssueRef(params.TaskRef); ok {
		result, err := u.repository.ClaimPartnerIssue(mergedCtx, params.Identity, issueID, params.OperationID, params.Now)
		if err != nil {
			return ClaimResult{}, err
		}
		output := ClaimResult{Status: result.Status}
		if result.Issue.ID != 0 {
			now := params.Now
			if now.IsZero() {
				now = time.Now().UTC()
			}
			task := partnerIssueTask(result.Issue, result.Rewards, now)
			output.Task = &task
		}
		return output, nil
	}
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

func groupTasks(tasks []repository.ActiveTask) []TaskGroupModel {
	groups := make([]TaskGroupModel, 0)
	indexByKey := make(map[string]int, len(tasks))
	for _, task := range tasks {
		index, ok := indexByKey[task.GroupKey]
		if !ok {
			title := task.GroupTitle
			if title == "" {
				title = task.GroupKey
			}
			groups = append(groups, TaskGroupModel{
				Key:         task.GroupKey,
				Title:       title,
				Description: task.GroupDesc,
				Tasks:       make([]TaskModel, 0),
			})
			index = len(groups) - 1
			indexByKey[task.GroupKey] = index
		}
		groups[index].Tasks = append(groups[index].Tasks, task)
	}
	return groups
}

func mapTask(task repository.Task) TaskModel {
	result := TaskModel{
		ID: task.ID, Key: task.Key, GroupKey: task.GroupKey, TaskKind: task.TaskKind,
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

package user

import (
	"time"

	"github.com/elum-utils/services/tasks/repository"
)

type Identity = repository.Identity

type RewardModel = repository.Reward
type ProgressModel = repository.ActiveProgress
type TaskModel = repository.ActiveTask

type ClaimParams struct {
	Identity    Identity
	TaskRef     string
	OperationID string
	Now         time.Time
}

type ClaimResult struct {
	Status string     `json:"status"`
	Task   *TaskModel `json:"task,omitempty"`
}

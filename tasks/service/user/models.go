package user

import (
	"context"
	"time"

	json "github.com/goccy/go-json"

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

type PartnerProvider interface {
	ListPartnerTasks(ctx context.Context, params PartnerListProviderParams) ([]PartnerExternalTask, error)
	CheckPartnerTask(ctx context.Context, params PartnerCheckProviderParams) (PartnerCheckResult, error)
}

type PartnerListParams struct {
	Identity  Identity
	Provider  string
	GroupKey  string
	Platform  string
	Locale    string
	Limit     int32
	Variables map[string]string
	Now       time.Time
}

type PartnerCheckParams struct {
	Identity  Identity
	IssueRef  string
	Variables map[string]string
	Now       time.Time
}

type PartnerListProviderParams struct {
	Identity  Identity
	Config    repository.PartnerConfig
	Locale    string
	Limit     int32
	Variables map[string]string
	Now       time.Time
}

type PartnerCheckProviderParams struct {
	Identity  Identity
	Config    repository.PartnerConfig
	Issue     repository.PartnerIssue
	Variables map[string]string
	Now       time.Time
}

type PartnerExternalTask struct {
	ExternalID     string
	ExternalType   string
	PublicPayload  json.RawMessage
	PrivatePayload json.RawMessage
	ExpiresAt      *time.Time
}

type PartnerCheckResult struct {
	Completed bool
	Status    string
	Payload   json.RawMessage
}

type PartnerCheckOutput struct {
	Status    string     `json:"status"`
	Completed bool       `json:"completed"`
	Task      *TaskModel `json:"task,omitempty"`
}

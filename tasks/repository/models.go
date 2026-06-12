package repository

import (
	"encoding/json"
	"time"
)

const (
	ActionKindAppAction         = "app_action"
	ActionKindAmountAction      = "amount_action"
	ActionKindChannelSubscribe  = "channel_subscribe"
	ActionKindAdvertisementView = "advertisement_view"
	ActionKindExternal          = "external"

	ClaimModeManual = "manual"
	ClaimModeAuto   = "auto"

	ResetNever  = "never"
	ResetSecond = "second"
	ResetMinute = "minute"
	ResetHour   = "hour"
	ResetDay    = "day"
	ResetYear   = "year"

	StatusOpen    = "open"
	StatusReady   = "ready"
	StatusClaimed = "claimed"

	RecordStatusRecorded   = "recorded"
	RecordStatusDuplicate  = "duplicate"
	RecordStatusNoTasks    = "no_tasks"
	ClaimStatusClaimed     = "claimed"
	ClaimStatusAlreadyDone = "already_claimed"
	ClaimStatusNotReady    = "not_ready"
	ClaimStatusNotFound    = "not_found"

	CallbackEventClaimed = "task.claimed"
)

type Identity struct {
	WorkspaceID    string
	AppID          int64
	PlatformID     int64
	PlatformUserID string
}

type Task struct {
	ID               uint64
	WorkspaceID      string
	Key              string
	GroupKey         string
	SequenceKey      *string
	SequencePosition *uint32
	ActionKey        string
	ActionKind       string
	ClaimMode        string
	TargetCount      uint64
	ResetUnit        string
	ResetEvery       uint32
	Position         int32
	Payload          json.RawMessage
	ImageURL         *string
	IsVisible        bool
	IsActive         bool
	StartAt          *time.Time
	EndAt            *time.Time
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Localization     *Localization
	Rewards          []Reward
	Progress         *Progress
}

type ActiveTask struct {
	ID          uint64          `json:"id"`
	Key         string          `json:"key"`
	GroupKey    string          `json:"group_key"`
	ActionKey   string          `json:"action_key"`
	ActionKind  string          `json:"action_kind"`
	ClaimMode   string          `json:"claim_mode"`
	TargetCount uint64          `json:"target_count"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	ImageURL    *string         `json:"image_url,omitempty"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	Rewards     []Reward        `json:"rewards"`
	Progress    *ActiveProgress `json:"progress,omitempty"`
	StartAt     *time.Time      `json:"-" msgpack:"start_at"`
	EndAt       *time.Time      `json:"-" msgpack:"end_at"`
}

type ActiveProgress struct {
	Progress      uint64     `json:"progress"`
	Status        string     `json:"status"`
	PeriodStartAt time.Time  `json:"period_start_at"`
	PeriodEndAt   time.Time  `json:"period_end_at"`
	ReadyAt       *time.Time `json:"ready_at,omitempty"`
	ClaimedAt     *time.Time `json:"claimed_at,omitempty"`
}

type Localization struct {
	Locale      string
	Title       string
	Description string
}

type Reward struct {
	Key      string `json:"key"`
	Type     string `json:"type"`
	Quantity int64  `json:"quantity"`
	Unit     *string `json:"unit,omitempty"`
}

type Progress struct {
	ID            uint64
	Progress      uint64
	Status        string
	PeriodStartAt time.Time
	PeriodEndAt   time.Time
	ReadyAt       *time.Time
	ClaimedAt     *time.Time
	OperationID   *string
	Rewards       []Reward
}

type SaveTaskParams struct {
	ID               uint64
	WorkspaceID      string
	Key              string
	GroupKey         string
	SequenceKey      *string
	SequencePosition *uint32
	ActionKey        string
	ActionKind       string
	ClaimMode        string
	TargetCount      uint64
	ResetUnit        string
	ResetEvery       uint32
	Position         int32
	Payload          json.RawMessage
	ImageURL         *string
	IsVisible        bool
	IsActive         bool
	StartAt          *time.Time
	EndAt            *time.Time
}

type RecordParams struct {
	Identity         Identity
	ActionKey        string
	Amount           uint64
	Source           string
	ExternalEventKey string
	Payload          json.RawMessage
	Now              time.Time
}

type RecordResult struct {
	Status    string
	Consumed  uint64
	Remaining uint64
	Tasks     []TaskResult
}

type TaskResult struct {
	Task     Task
	Before   uint64
	After    uint64
	Consumed uint64
	Claimed  bool
}

type ClaimParams struct {
	Identity    Identity
	TaskRef     string
	OperationID string
	Now         time.Time
}

type ClaimResult struct {
	Status string
	Task   *Task
}

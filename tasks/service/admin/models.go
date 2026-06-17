package admin

import (
	"encoding/json"
	"time"
)

type SaveTaskParams struct {
	ID                  uint64
	WorkspaceID         string
	Key                 string
	GroupKey            string
	SequenceKey         *string
	SequencePosition    *uint32
	TaskKind            string
	ActionKey           string
	ActionKind          string
	ClaimMode           string
	TargetCount         uint64
	ResetUnit           string
	ResetEvery          uint32
	Position            int32
	Payload             json.RawMessage
	IntegrationKind     *string
	IntegrationProvider *string
	IntegrationPayload  json.RawMessage
	ImageURL            *string
	IsVisible           bool
	IsActive            bool
	StartAt             *time.Time
	EndAt               *time.Time
}

type TaskModel struct {
	ID                  uint64          `json:"id"`
	Key                 string          `json:"key"`
	GroupKey            string          `json:"group_key"`
	SequenceKey         *string         `json:"sequence_key,omitempty"`
	SequencePosition    *uint32         `json:"sequence_position,omitempty"`
	TaskKind            string          `json:"task_kind"`
	ActionKey           string          `json:"action_key"`
	ActionKind          string          `json:"action_kind"`
	ClaimMode           string          `json:"claim_mode"`
	TargetCount         uint64          `json:"target_count"`
	ResetUnit           string          `json:"reset_unit"`
	ResetEvery          uint32          `json:"reset_every"`
	Position            int32           `json:"position"`
	Payload             json.RawMessage `json:"payload,omitempty"`
	IntegrationKind     *string         `json:"integration_kind,omitempty"`
	IntegrationProvider *string         `json:"integration_provider,omitempty"`
	IntegrationPayload  json.RawMessage `json:"integration_payload,omitempty"`
	ImageURL            *string         `json:"image_url,omitempty"`
	IsVisible           bool            `json:"is_visible"`
	IsActive            bool            `json:"is_active"`
	StartAt             *time.Time      `json:"start_at,omitempty"`
	EndAt               *time.Time      `json:"end_at,omitempty"`
	DeletedAt           *time.Time      `json:"deleted_at,omitempty"`
}

type RewardModel struct {
	Key      string  `json:"key"`
	Type     string  `json:"type"`
	Quantity int64   `json:"quantity"`
	Unit     *string `json:"unit,omitempty"`
}

type StatsModel struct {
	TasksTotal         uint64 `json:"tasks_total"`
	ActiveTasks        uint64 `json:"active_tasks"`
	VisibleTasks       uint64 `json:"visible_tasks"`
	ProgressTotal      uint64 `json:"progress_total"`
	OpenProgress       uint64 `json:"open_progress"`
	ReadyProgress      uint64 `json:"ready_progress"`
	ClaimedProgress    uint64 `json:"claimed_progress"`
	ProgressCreated    uint64 `json:"progress_created"`
	ProgressAmount     uint64 `json:"progress_amount"`
	ReadyCount         uint64 `json:"ready_count"`
	ClaimedCount       uint64 `json:"claimed_count"`
	ManualClaimedCount uint64 `json:"manual_claimed_count"`
	AutoClaimedCount   uint64 `json:"auto_claimed_count"`
	UniqueParticipants uint64 `json:"unique_participants"`
	UniqueClaimers     uint64 `json:"unique_claimers"`
}

type TaskStatsModel struct {
	TaskID             uint64 `json:"task_id"`
	ProgressTotal      uint64 `json:"progress_total"`
	OpenProgress       uint64 `json:"open_progress"`
	ReadyProgress      uint64 `json:"ready_progress"`
	ClaimedProgress    uint64 `json:"claimed_progress"`
	ProgressCreated    uint64 `json:"progress_created"`
	ProgressAmount     uint64 `json:"progress_amount"`
	ReadyCount         uint64 `json:"ready_count"`
	ClaimedCount       uint64 `json:"claimed_count"`
	ManualClaimedCount uint64 `json:"manual_claimed_count"`
	AutoClaimedCount   uint64 `json:"auto_claimed_count"`
	UniqueParticipants uint64 `json:"unique_participants"`
	UniqueClaimers     uint64 `json:"unique_claimers"`
}

type DailyStatsModel struct {
	Date               time.Time `json:"date"`
	TaskID             uint64    `json:"task_id"`
	ProgressCreated    uint64    `json:"progress_created"`
	ProgressAmount     uint64    `json:"progress_amount"`
	ReadyCount         uint64    `json:"ready_count"`
	ClaimedCount       uint64    `json:"claimed_count"`
	ManualClaimedCount uint64    `json:"manual_claimed_count"`
	AutoClaimedCount   uint64    `json:"auto_claimed_count"`
	UniqueParticipants uint64    `json:"unique_participants"`
	UniqueClaimers     uint64    `json:"unique_claimers"`
}

type DailyOverviewModel struct {
	Date               time.Time `json:"date"`
	TasksTotal         uint64    `json:"tasks_total"`
	ActiveTasks        uint64    `json:"active_tasks"`
	VisibleTasks       uint64    `json:"visible_tasks"`
	ProgressCreated    uint64    `json:"progress_created"`
	ProgressAmount     uint64    `json:"progress_amount"`
	ReadyCount         uint64    `json:"ready_count"`
	ClaimedCount       uint64    `json:"claimed_count"`
	ManualClaimedCount uint64    `json:"manual_claimed_count"`
	AutoClaimedCount   uint64    `json:"auto_claimed_count"`
	UniqueParticipants uint64    `json:"unique_participants"`
	UniqueClaimers     uint64    `json:"unique_claimers"`
}

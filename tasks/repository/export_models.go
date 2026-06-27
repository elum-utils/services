package repository

import (
	"time"

	json "github.com/goccy/go-json"
)

const (
	ExportFormat = "tasks.export.v1"

	ExportSectionGroups         = "groups"
	ExportSectionSequences      = "sequences"
	ExportSectionTasks          = "tasks"
	ExportSectionLocalization   = "localization"
	ExportSectionRewards        = "rewards"
	ExportSectionTarget         = "target"
	ExportSectionIntegration    = "integration"
	ExportSectionPartnerConfigs = "partner_configs"
	ExportSectionPartnerRewards = "partner_rewards"

	ImportConflictFail   = "fail_on_conflict"
	ImportConflictSkip   = "skip_existing"
	ImportConflictUpdate = "update_existing"
)

type ExportRequest struct {
	Sections []string  `json:"sections,omitempty"`
	Now      time.Time `json:"-"`
}

type ExportManifest struct {
	Format   string                  `json:"format"`
	Service  string                  `json:"service"`
	Sections []ExportManifestSection `json:"sections"`
}

type ExportManifestSection struct {
	Key            string `json:"key"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	DefaultEnabled bool   `json:"default_enabled"`
}

type ExportPackage struct {
	Format    string           `json:"format"`
	Service   string           `json:"service"`
	CreatedAt time.Time        `json:"created_at"`
	Groups    []ExportGroup    `json:"groups"`
	Sequences []ExportSequence `json:"sequences,omitempty"`
}

type ExportGroup struct {
	Key                string                    `json:"key"`
	Position           int32                     `json:"position"`
	IsActive           bool                      `json:"is_active"`
	Localization       map[string]ExportText     `json:"localization,omitempty"`
	Tasks              []ExportTask              `json:"tasks,omitempty"`
	PartnerConfigs     []ExportPartnerConfig     `json:"partner_configs,omitempty"`
	PartnerRewardRules []ExportPartnerRewardRule `json:"partner_reward_rules,omitempty"`
}

type ExportText struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type ExportSequence struct {
	Key      string `json:"key"`
	Position int32  `json:"position"`
	IsActive bool   `json:"is_active"`
}

type ExportTask struct {
	Key              string                `json:"key"`
	SequenceKey      *string               `json:"sequence_key,omitempty"`
	SequencePosition *uint32               `json:"sequence_position,omitempty"`
	TaskKind         string                `json:"task_kind"`
	ActionKey        string                `json:"action_key"`
	ActionKind       string                `json:"action_kind"`
	ClaimMode        string                `json:"claim_mode"`
	TargetCount      uint64                `json:"target_count"`
	Reset            ExportReset           `json:"reset"`
	Position         int32                 `json:"position"`
	Payload          json.RawMessage       `json:"payload,omitempty"`
	Target           json.RawMessage       `json:"target,omitempty"`
	Integration      ExportIntegration     `json:"integration"`
	ImageURL         *string               `json:"image_url,omitempty"`
	IsVisible        bool                  `json:"is_visible"`
	IsActive         bool                  `json:"is_active"`
	StartAt          *time.Time            `json:"start_at,omitempty"`
	EndAt            *time.Time            `json:"end_at,omitempty"`
	Localization     map[string]ExportText `json:"localization,omitempty"`
	Rewards          []ExportReward        `json:"rewards,omitempty"`
}

type ExportReset struct {
	Unit  string `json:"unit"`
	Every uint32 `json:"every"`
}

type ExportIntegration struct {
	Kind     *string         `json:"kind,omitempty"`
	Provider *string         `json:"provider,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

type ExportReward struct {
	Key      string  `json:"key"`
	Type     string  `json:"type"`
	Quantity int64   `json:"quantity"`
	Scale    uint16  `json:"scale"`
	Unit     *string `json:"unit,omitempty"`
	Position int32   `json:"position"`
}

type ExportPartnerConfig struct {
	Provider  string          `json:"provider"`
	Platform  string          `json:"platform"`
	IsEnabled bool            `json:"is_enabled"`
	Secret    *ExportSecret   `json:"secret,omitempty"`
	Target    json.RawMessage `json:"target,omitempty"`
	Settings  json.RawMessage `json:"settings,omitempty"`
}

type ExportSecret struct {
	Mode string `json:"mode"`
	Key  string `json:"key"`
}

type ExportPartnerRewardRule struct {
	Provider     string       `json:"provider"`
	ExternalType string       `json:"external_type"`
	Reward       ExportReward `json:"reward"`
	Position     int32        `json:"position"`
	IsEnabled    bool         `json:"is_enabled"`
}

type ImportRequest struct {
	Package          ExportPackage     `json:"package"`
	ConflictStrategy string            `json:"conflict_strategy"`
	Secrets          map[string]string `json:"secrets,omitempty"`
}

type ImportPreview struct {
	Format          string           `json:"format"`
	Service         string           `json:"service"`
	Counts          ImportCounts     `json:"counts"`
	Conflicts       []ImportConflict `json:"conflicts,omitempty"`
	Warnings        []string         `json:"warnings,omitempty"`
	RequiredSecrets []ExportSecret   `json:"required_secrets,omitempty"`
}

type ImportCounts struct {
	Groups             int `json:"groups"`
	Sequences          int `json:"sequences"`
	Tasks              int `json:"tasks"`
	TaskLocalizations  int `json:"task_localizations"`
	GroupLocalizations int `json:"group_localizations"`
	Rewards            int `json:"rewards"`
	PartnerConfigs     int `json:"partner_configs"`
	PartnerRewards     int `json:"partner_rewards"`
}

type ImportConflict struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type ImportResult struct {
	Imported ImportCounts `json:"imported"`
	Skipped  ImportCounts `json:"skipped"`
}

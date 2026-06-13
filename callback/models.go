package callback

// Identity identifies one user across all service callback payloads.
type Identity struct {
	WorkspaceID    string `json:"workspace_id"`
	AppID          int64  `json:"app_id"`
	PlatformID     int64  `json:"platform_id"`
	PlatformUserID string `json:"platform_user_id"`
}

// Reward describes a quantity or duration reward emitted by a service.
type Reward struct {
	Key      string  `json:"key"`
	Type     string  `json:"type"`
	Quantity int64   `json:"quantity"`
	Unit     *string `json:"unit,omitempty"`
}

// RewardPayload is the common reward-bearing view exposed by every service callback context.
type RewardPayload struct {
	Identity
	Rewards []Reward `json:"rewards"`
}

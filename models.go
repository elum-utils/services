package services

// Identity identifies one user across all public service user methods and callback payloads.
type Identity struct {
	WorkspaceID    string `json:"workspace_id"`
	AppID          int64  `json:"app_id"`
	PlatformID     int64  `json:"platform_id"`
	Platform       string `json:"platform,omitempty"`
	PlatformUserID string `json:"platform_user_id"`
	IsPremium      bool   `json:"is_premium,omitempty"`
	Sex            string `json:"sex,omitempty"`
	Country        string `json:"country,omitempty"`
}

// Actor identifies a related user actor, for example a payer for a gifted purchase.
type Actor struct {
	PlatformID     int64  `json:"platform_id"`
	Platform       string `json:"platform,omitempty"`
	PlatformUserID string `json:"platform_user_id"`
	InternalUserID *int64 `json:"internal_user_id,omitempty"`
}

// Reward describes a quantity or duration reward emitted by a service.
type Reward struct {
	Key      string  `json:"key"`
	Type     string  `json:"type"`
	Quantity int64   `json:"quantity"`
	Scale    uint16  `json:"scale"`
	Unit     *string `json:"unit,omitempty"`
}

// RewardPayload is the common reward-bearing view exposed by every service callback context.
type RewardPayload struct {
	Identity
	Rewards []Reward `json:"rewards"`
}

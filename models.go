package services

import (
	"strings"

	serviceerrors "github.com/elum-utils/services/errors"
)

var (
	ErrIdentityWorkspaceRequired      = serviceerrors.New(serviceerrors.CodeInvalidFields, "identity workspace id is required")
	ErrIdentityAppIDInvalid           = serviceerrors.New(serviceerrors.CodeInvalidFields, "identity app id must be positive")
	ErrIdentityPlatformIDInvalid      = serviceerrors.New(serviceerrors.CodeInvalidFields, "identity platform id must be positive")
	ErrIdentityPlatformUserIDRequired = serviceerrors.New(serviceerrors.CodeInvalidFields, "identity platform user id is required")
)

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

// Validate verifies the identity fields required by public user operations.
func (i Identity) Validate() error {
	if strings.TrimSpace(i.WorkspaceID) == "" {
		return ErrIdentityWorkspaceRequired
	}
	if i.AppID <= 0 {
		return ErrIdentityAppIDInvalid
	}
	if i.PlatformID <= 0 {
		return ErrIdentityPlatformIDInvalid
	}
	if strings.TrimSpace(i.PlatformUserID) == "" {
		return ErrIdentityPlatformUserIDRequired
	}
	return nil
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

package repository

import (
	"time"

	json "github.com/goccy/go-json"
)

const (
	CodeModeShared   = "shared_code"
	CodeModePersonal = "personal_code"

	CodeSourceGenerated = "generated"
	CodeSourcePool      = "pool"

	StatusIssued    = "issued"
	StatusCompleted = "completed"
)

type Offer struct {
	WorkspaceID       string
	ID                string
	Payload           json.RawMessage
	Target            json.RawMessage
	CodeMode          string
	CodeSource        *string
	SharedCode        *string
	GeneratedLength   *int16
	GeneratedAlphabet *string
	IsActive          bool
	StartAt           *time.Time
	EndAt             *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type OfferBundle struct {
	Offer         Offer
	Localization  *Localization
	Localizations []Localization
	Rewards       []Reward
	Assignment    *Assignment
}

type Localization struct {
	WorkspaceID string
	CPAID       string
	Locale      string
	Title       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Reward struct {
	WorkspaceID string
	CPAID       string
	Key         string
	Type        string
	Quantity    int64
	Scale       uint16
	Unit        *string
}

type Assignment struct {
	ID             uint64
	WorkspaceID    string
	CPAID          string
	AppID          int64
	PlatformID     int64
	PlatformUserID string
	CodeID         *uint64
	Code           string
	CodeMode       string
	Status         string
	IssuedAt       time.Time
	CompletedAt    *time.Time
}

type Code struct {
	ID          uint64
	WorkspaceID string
	CPAID       string
	Code        string
	Source      string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

type AssignmentEvent struct {
	ID           uint64
	WorkspaceID  string
	CPAID        string
	AssignmentID uint64
	EventType    string
	OccurredAt   time.Time
}

type DailyStats struct {
	Date           time.Time
	IssuedCount    uint64
	CompletedCount uint64
	UniqueUsers    uint64
}

type Stats struct {
	AssignmentsTotal uint64
	IssuedTotal      uint64
	CompletedTotal   uint64
	DeletedTotal     uint64
	CodesTotal       uint64
	AvailableCodes   uint64
	IssuedCodes      uint64
	CompletedCodes   uint64
	DeletedCodes     uint64
}

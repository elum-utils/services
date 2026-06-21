package repository

import (
	json "github.com/goccy/go-json"
	"time"
)

type Account struct {
	ID, DisplayName, Status string
	CreatedAt, UpdatedAt    time.Time
}

type Identity struct {
	AccountID, Provider, ProviderSubject string
	Payload                              json.RawMessage
	CreatedAt, UpdatedAt                 time.Time
}

type Session struct {
	ID, AccountID, IP, UserAgent string
	BindToIP                     bool
	ExpiresAt                    time.Time
	RevokedAt                    *time.Time
	LastUsedAt, CreatedAt        time.Time
}

type Workspace struct {
	ID, Slug, Title, Status, CreatedBy string
	CreatedAt, UpdatedAt               time.Time
}

type Member struct {
	WorkspaceID, AccountID, DisplayName string
	Position                            int32
	JoinedAt, UpdatedAt                 time.Time
}

type Invite struct {
	ID, WorkspaceID, CreatedBy string
	MaxUses, UsedCount         *uint32
	ExpiresAt, RevokedAt       *time.Time
	CreatedAt                  time.Time
	RoleIDs                    []string
}

type Role struct {
	ID, WorkspaceID, Code, Title, Description string
	Position                                  int32
	IsOwner                                   bool
	MemberCount                               int64
	CreatedAt, UpdatedAt                      time.Time
}

type Method struct {
	Key, Service, GroupKey, Title string
	WorkspaceScoped, Sensitive    bool
	SchemaRevision                uint32
	Status                        string
	CreatedAt, UpdatedAt          time.Time
}

type AuditEvent struct {
	ID, WorkspaceID, ActorID, MethodKey, TargetType, TargetID, Result, RequestID string
	BeforeData, AfterData                                                        json.RawMessage
	OccurredAt                                                                   time.Time
}

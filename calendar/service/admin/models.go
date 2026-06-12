package admin

import (
	"time"

	"github.com/elum-utils/services/calendar/service/user"
)

type Page struct {
	Limit  int32
	Offset int32
}

type LocalizationModel struct {
	Locale      string `json:"locale"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CalendarModel struct {
	user.CalendarModel
	DeletedAt     *time.Time          `json:"deleted_at,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Localizations []LocalizationModel `json:"localizations,omitempty"`
}

type StatsModel struct {
	OperationCount uint64 `json:"operation_count"`
	GrantCount     uint64 `json:"grant_count"`
	UniqueUsers    uint64 `json:"unique_users"`
}

type DailyStatsModel struct {
	Date           time.Time `json:"date"`
	OperationCount uint64    `json:"operation_count"`
	GrantCount     uint64    `json:"grant_count"`
	UniqueUsers    uint64    `json:"unique_users"`
}

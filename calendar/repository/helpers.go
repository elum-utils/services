package repository

import (
	"database/sql"
	"encoding/json"

	calendarsqlc "github.com/elum-utils/services/calendar/sqlc"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/google/uuid"
)

func mapDefinition(row calendarsqlc.CalendarDefinition) Calendar {
	return Calendar{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Type: row.Type,
		Mode: string(row.Mode), IntervalType: string(row.IntervalType),
		IntervalUnit: string(row.IntervalUnit), IntervalCount: row.IntervalCount,
		ResetAfterIntervals: row.ResetAfterIntervals, EndBehavior: string(row.EndBehavior),
		Timezone: row.Timezone, HideFutureRewards: row.HideFutureRewards,
		IsActive: row.IsActive, StartAt: sqlwrap.NullTimePtr(row.StartAt),
		EndAt: sqlwrap.NullTimePtr(row.EndAt), DeletedAt: sqlwrap.NullTimePtr(row.DeletedAt),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func appendStep(steps []Step, stepID sql.NullInt64, position sql.NullInt32,
	rewardID sql.NullInt64, rewardKey sql.NullString,
	rewardType calendarsqlc.NullCalendarRewardRewardType, rewardCount sql.NullInt64,
	rewardUnit calendarsqlc.NullCalendarRewardDurationUnit,
) []Step {
	if !stepID.Valid {
		return steps
	}
	if len(steps) == 0 || steps[len(steps)-1].ID != uint64(stepID.Int64) {
		steps = append(steps, Step{
			ID: uint64(stepID.Int64), Position: uint32(position.Int32), Rewards: make([]Reward, 0),
		})
	}
	if rewardID.Valid {
		index := len(steps) - 1
		steps[index].Rewards = append(steps[index].Rewards, Reward{
			Key: rewardKey.String, Type: string(rewardType.CalendarRewardRewardType),
			Quantity: rewardCount.Int64, Unit: calendarDurationUnitPtr(rewardUnit),
		})
	}
	return steps
}

func calendarDurationUnitPtr(value calendarsqlc.NullCalendarRewardDurationUnit) *string {
	if !value.Valid {
		return nil
	}
	unit := string(value.CalendarRewardDurationUnit)
	return &unit
}

func calendarStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func decodeRewards(raw json.RawMessage) ([]Reward, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var rewards []Reward
	if err := json.Unmarshal(raw, &rewards); err != nil {
		return nil, err
	}
	return rewards, nil
}

func calendarReference(ref string) (id, calendarType string) {
	if _, err := uuid.Parse(ref); err == nil {
		return ref, ""
	}
	return "", ref
}

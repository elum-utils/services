package admin

import (
	"context"

	"github.com/elum-utils/services/calendar/repository"
	"github.com/elum-utils/services/calendar/service/user"
)

type SaveRewardParams struct {
	WorkspaceID string
	CalendarID  string
	StepID      uint64
	ID          uint64
	Key         string
	Type        string
	Quantity    int64
	Scale       uint16
	Unit        *string
	Position    uint32
}

func (a *Admin) CreateReward(ctx context.Context, params SaveRewardParams) (uint64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	if err := validateReward(params); err != nil {
		return 0, err
	}
	return a.repository.UpsertReward(mergedCtx, params.WorkspaceID, params.CalendarID,
		params.StepID, repository.Reward{
			Key: params.Key, Type: normalizedRewardType(params.Type),
			Quantity: params.Quantity, Scale: params.Scale, Unit: params.Unit,
		}, params.Position)
}

func (a *Admin) UpdateReward(ctx context.Context, params SaveRewardParams) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	if params.ID == 0 {
		return 0, ErrRewardIDRequired
	}
	if err := validateReward(params); err != nil {
		return 0, err
	}
	return a.repository.UpdateReward(mergedCtx, params.WorkspaceID, params.CalendarID,
		params.StepID, params.ID, repository.Reward{
			Key: params.Key, Type: normalizedRewardType(params.Type),
			Quantity: params.Quantity, Scale: params.Scale, Unit: params.Unit,
		},
		params.Position)
}

func (a *Admin) GetReward(ctx context.Context, workspaceID, calendarID string, id uint64) (user.RewardModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	value, err := a.repository.GetReward(mergedCtx, workspaceID, calendarID, id)
	if err != nil {
		return user.RewardModel{}, err
	}
	return user.RewardModel{
		Key: value.Key, Type: value.Type, Quantity: value.Quantity, Scale: value.Scale, Unit: value.Unit,
	}, nil
}

func (a *Admin) DeleteReward(ctx context.Context, workspaceID, calendarID string, id uint64) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteReward(mergedCtx, workspaceID, calendarID, id)
}

func validateReward(params SaveRewardParams) error {
	if params.WorkspaceID == "" || params.CalendarID == "" || params.StepID == 0 ||
		params.Key == "" || params.Quantity <= 0 || params.Position == 0 {
		return ErrRewardRequired
	}
	switch normalizedRewardType(params.Type) {
	case "quantity":
		if params.Unit != nil {
			return ErrRewardQuantityUnit
		}
	case "duration":
		if params.Unit == nil || !validDurationUnit(*params.Unit) {
			return ErrRewardDurationUnit
		}
	default:
		return ErrRewardTypeUnsupported
	}
	return nil
}

func normalizedRewardType(value string) string {
	if value == "" {
		return "quantity"
	}
	return value
}

func validDurationUnit(unit string) bool {
	switch unit {
	case "second", "minute", "hour", "day", "week", "month", "year":
		return true
	default:
		return false
	}
}

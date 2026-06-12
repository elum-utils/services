package admin

import (
	"context"
	"errors"

	"github.com/elum-utils/services/cpa/repository"
	"github.com/elum-utils/services/cpa/service/user"
)

type UpsertRewardParams struct {
	WorkspaceID string
	CPAID       string
	Key         string
	Type        string
	Quantity    int64
	Unit        *string
}

func (a *Admin) UpsertReward(ctx context.Context, params UpsertRewardParams) error {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	rewardType, err := validateReward(params.Key, params.Type, params.Quantity, params.Unit)
	if err != nil {
		return err
	}
	return a.repository.UpsertReward(mergedCtx, repository.Reward{
		WorkspaceID: params.WorkspaceID,
		CPAID:       params.CPAID,
		Key:         params.Key,
		Type:        rewardType,
		Quantity:    params.Quantity,
		Unit:        params.Unit,
	})
}

func (a *Admin) ListRewards(ctx context.Context, workspaceID, cpaID string) ([]user.RewardModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	values, err := a.repository.ListRewards(mergedCtx, workspaceID, cpaID)
	if err != nil {
		return nil, err
	}
	result := mapOffer(repository.Offer{}, nil, values)
	return result.Rewards, nil
}

func (a *Admin) DeleteReward(ctx context.Context, workspaceID, cpaID, rewardID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteReward(mergedCtx, workspaceID, cpaID, rewardID)
}

func validateReward(key, rewardType string, quantity int64, unit *string) (string, error) {
	if key == "" || quantity <= 0 {
		return "", errors.New("cpa admin: reward key and positive quantity are required")
	}
	if rewardType == "" {
		rewardType = "quantity"
	}
	switch rewardType {
	case "quantity":
		if unit != nil {
			return "", errors.New("cpa admin: quantity reward must not have duration unit")
		}
	case "duration":
		if unit == nil || !validDurationUnit(*unit) {
			return "", errors.New("cpa admin: duration reward requires a valid duration unit")
		}
	default:
		return "", errors.New("cpa admin: reward type must be quantity or duration")
	}
	return rewardType, nil
}

func validDurationUnit(unit string) bool {
	switch unit {
	case "second", "minute", "hour", "day", "week", "month", "year":
		return true
	default:
		return false
	}
}

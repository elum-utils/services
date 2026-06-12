package admin

import (
	"context"
	"errors"

	"github.com/elum-utils/services/promo/repository"
	"github.com/elum-utils/services/promo/service/user"
)

type SaveRewardParams struct {
	WorkspaceID string
	PromoID     uint64
	Key         string
	Type        string
	Quantity    int64
	Unit        *string
}

func (a *Admin) UpsertReward(ctx context.Context, params SaveRewardParams) error {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	rewardType, err := validateReward(params.Key, params.Type, params.Quantity, params.Unit)
	if err != nil {
		return err
	}
	return a.repository.UpsertReward(mergedCtx, params.WorkspaceID, params.PromoID, repository.Reward{
		Key: params.Key, Type: rewardType, Quantity: params.Quantity, Unit: params.Unit,
	})
}

func (a *Admin) GetReward(ctx context.Context, workspaceID string, promoID uint64, key string) (user.RewardModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	value, err := a.repository.GetReward(mergedCtx, workspaceID, promoID, key)
	if err != nil {
		return user.RewardModel{}, err
	}
	return user.RewardModel{Key: value.Key, Type: value.Type, Quantity: value.Quantity, Unit: value.Unit}, nil
}

func (a *Admin) ListRewards(ctx context.Context, workspaceID string, promoID uint64) ([]user.RewardModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	values, err := a.repository.ListRewards(mergedCtx, workspaceID, promoID)
	if err != nil {
		return nil, err
	}
	result := make([]user.RewardModel, 0, len(values))
	for _, value := range values {
		result = append(result, user.RewardModel{
			Key: value.Key, Type: value.Type, Quantity: value.Quantity, Unit: value.Unit,
		})
	}
	return result, nil
}

func (a *Admin) DeleteReward(ctx context.Context, workspaceID string, promoID uint64, key string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteReward(mergedCtx, workspaceID, promoID, key)
}

func validateReward(key, rewardType string, quantity int64, unit *string) (string, error) {
	if key == "" || quantity <= 0 {
		return "", errors.New("promo admin: reward key and positive quantity are required")
	}
	if rewardType == "" {
		rewardType = "quantity"
	}
	switch rewardType {
	case "quantity":
		if unit != nil {
			return "", errors.New("promo admin: quantity reward must not have duration unit")
		}
	case "duration":
		if unit == nil || !validDurationUnit(*unit) {
			return "", errors.New("promo admin: duration reward requires a valid duration unit")
		}
	default:
		return "", errors.New("promo admin: reward type must be quantity or duration")
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

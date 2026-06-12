package repository

import (
	"context"
	"encoding/json"
	"time"

	promosqlc "github.com/elum-utils/services/promo/sqlc"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

func (r *Repository) Apply(ctx context.Context, identity Identity, code, locale string) (ApplyResult, error) {
	result := ApplyResult{Status: StatusNotFound}
	err := r.WithTx(ctx, func(txRepo *Repository) error {
		rows, err := txRepo.q.GetApplyBundleForUpdate(ctx, promosqlc.GetApplyBundleForUpdateParams{
			Locale: locale, AppID: identity.AppID, PlatformID: identity.PlatformID,
			PlatformUserID: identity.PlatformUserID, WorkspaceID: identity.WorkspaceID,
			CodeNormalized: normalizeCode(code),
		})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		result = mapApplyBundle(rows)
		if result.Redemption != nil {
			result.Status = StatusAlreadyApplied
			return nil
		}

		now := time.Now()
		switch {
		case result.Promo.DeletedAt != nil:
			result.Status = StatusNotFound
		case !result.Promo.IsActive:
			result.Status = StatusInactive
		case result.Promo.StartAt != nil && now.Before(*result.Promo.StartAt):
			result.Status = StatusNotStarted
		case result.Promo.EndAt != nil && !now.Before(*result.Promo.EndAt):
			result.Status = StatusExpired
		case result.Promo.MaxActivations > 0 &&
			result.Promo.ActivationCount >= result.Promo.MaxActivations:
			result.Status = StatusLimitReached
		default:
			rewardSnapshot, err := json.Marshal(result.Rewards)
			if err != nil {
				return err
			}
			id, err := txRepo.q.CreateRedemption(ctx, promosqlc.CreateRedemptionParams{
				WorkspaceID: identity.WorkspaceID, PromoID: result.Promo.ID,
				AppID: identity.AppID, PlatformID: identity.PlatformID,
				PlatformUserID: identity.PlatformUserID, RewardSnapshot: rewardSnapshot,
			})
			if err != nil {
				return err
			}
			redemption := Redemption{
				ID: uint64(id), WorkspaceID: identity.WorkspaceID, PromoID: result.Promo.ID,
				AppID: identity.AppID, PlatformID: identity.PlatformID,
				PlatformUserID: identity.PlatformUserID, RedeemedAt: now,
			}
			result.Redemption = &redemption
			result.Status = StatusSuccess
			result.Promo.ActivationCount++
		}
		return nil
	})
	return result, err
}

func mapApplyBundle(rows []promosqlc.GetApplyBundleForUpdateRow) ApplyResult {
	first := rows[0]
	result := ApplyResult{
		Status: StatusNotFound,
		Promo: Promo{
			ID: first.ID, WorkspaceID: first.WorkspaceID, Code: first.Code, Payload: first.Payload,
			MaxActivations: first.MaxActivations, ActivationCount: first.ActivationCount,
			IsActive: first.IsActive, StartAt: sqlwrap.NullTimePtr(first.StartAt),
			EndAt: sqlwrap.NullTimePtr(first.EndAt), DeletedAt: sqlwrap.NullTimePtr(first.DeletedAt),
			CreatedAt: first.CreatedAt, UpdatedAt: first.UpdatedAt,
		},
		Rewards: make([]Reward, 0, len(rows)),
	}
	if first.LocalizationLocale.Valid {
		result.Localization = &Localization{
			WorkspaceID: first.WorkspaceID, PromoID: first.ID,
			Locale: first.LocalizationLocale.String, Title: first.LocalizationTitle.String,
			Description: first.LocalizationDescription.String,
		}
	}
	if first.RedemptionID.Valid {
		result.Redemption = &Redemption{
			ID: uint64(first.RedemptionID.Int64), WorkspaceID: first.WorkspaceID, PromoID: first.ID,
			AppID: first.RedemptionAppID.Int64, PlatformID: first.RedemptionPlatformID.Int64,
			PlatformUserID: first.RedemptionPlatformUserID.String,
			RedeemedAt:     first.RedemptionRedeemedAt.Time,
		}
	}
	for _, row := range rows {
		if row.RewardID.Valid {
			result.Rewards = append(result.Rewards, Reward{
				Key:      row.RewardKey.String,
				Type:     string(row.RewardType.PromoRewardRewardType),
				Quantity: row.RewardQuantity.Int64,
				Unit:     promoDurationUnitPtr(row.DurationUnit),
			})
		}
	}
	return result
}

func mapRedemption(row promosqlc.PromoRedemption) Redemption {
	return Redemption{
		ID: row.ID, WorkspaceID: row.WorkspaceID, PromoID: row.PromoID,
		AppID: row.AppID, PlatformID: row.PlatformID, PlatformUserID: row.PlatformUserID,
		RedeemedAt: row.RedeemedAt,
	}
}

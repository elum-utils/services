package repository

import (
	"context"
	"database/sql"
	json "github.com/goccy/go-json"
	"time"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	promosqlc "github.com/elum-utils/services/promo/sqlc"
)

type SavePromoParams struct {
	ID             uint64
	WorkspaceID    string
	Code           string
	Payload        json.RawMessage
	Target         json.RawMessage
	MaxActivations uint64
	IsActive       bool
	StartAt        *time.Time
	EndAt          *time.Time
}

func (r *Repository) CreatePromo(ctx context.Context, params SavePromoParams) (uint64, error) {
	target := params.Target
	if len(target) == 0 {
		target = []byte("null")
	}
	id, err := r.q.AdminCreatePromo(ctx, promosqlc.AdminCreatePromoParams{
		WorkspaceID:    params.WorkspaceID,
		Code:           params.Code,
		CodeNormalized: normalizeCode(params.Code),
		Payload:        params.Payload,
		Target:         target,
		MaxActivations: params.MaxActivations,
		IsActive:       params.IsActive,
		StartAt: sqlwrap.NullFromPtr(params.StartAt, func(v time.Time) sql.NullTime {
			return sql.NullTime{Time: v, Valid: true}
		}),
		EndAt: sqlwrap.NullFromPtr(params.EndAt, func(v time.Time) sql.NullTime {
			return sql.NullTime{Time: v, Valid: true}
		}),
	})
	if err != nil {
		return 0, err
	}
	return uint64(id), r.invalidatePromoCache(params.WorkspaceID)
}

func (r *Repository) UpdatePromo(ctx context.Context, params SavePromoParams) (int64, error) {
	target := params.Target
	if len(target) == 0 {
		target = []byte("null")
	}
	rows, err := r.q.AdminUpdatePromo(ctx, promosqlc.AdminUpdatePromoParams{
		Code:           params.Code,
		CodeNormalized: normalizeCode(params.Code),
		Payload:        params.Payload,
		Target:         target,
		MaxActivations: params.MaxActivations,
		IsActive:       params.IsActive,
		StartAt: sqlwrap.NullFromPtr(params.StartAt, func(v time.Time) sql.NullTime {
			return sql.NullTime{Time: v, Valid: true}
		}),
		EndAt: sqlwrap.NullFromPtr(params.EndAt, func(v time.Time) sql.NullTime {
			return sql.NullTime{Time: v, Valid: true}
		}),
		WorkspaceID: params.WorkspaceID,
		ID:          params.ID,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidatePromoCache(params.WorkspaceID)
}

func (r *Repository) GetPromo(ctx context.Context, workspaceID string, id uint64) (Promo, error) {
	key := promoCacheKey("admin_get_promo", workspaceID, id)
	rememberPromoCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Promo, error) {
		row, err := r.q.AdminGetPromo(ctx, promosqlc.AdminGetPromoParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			return Promo{}, err
		}
		return mapPromo(row), nil
	})
}

func (r *Repository) ListPromos(ctx context.Context, workspaceID string, limit, offset int32) ([]Promo, error) {
	limit, offset = normalizePage(limit, offset)
	key := promoCacheKey("admin_list_promos", workspaceID, limit, offset)
	rememberPromoCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Promo, error) {
		rows, err := r.q.AdminListPromos(ctx, promosqlc.AdminListPromosParams{
			WorkspaceID: workspaceID, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, err
		}
		result := make([]Promo, 0, len(rows))
		for _, row := range rows {
			result = append(result, mapPromo(row))
		}
		return result, nil
	})
}

func (r *Repository) SoftDeletePromo(ctx context.Context, workspaceID string, id uint64) (int64, error) {
	rows, err := r.q.AdminSoftDeletePromo(ctx, promosqlc.AdminSoftDeletePromoParams{WorkspaceID: workspaceID, ID: id})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidatePromoCache(workspaceID)
}

func (r *Repository) UpsertLocalization(ctx context.Context, value Localization) error {
	if err := r.q.AdminUpsertLocalization(ctx, promosqlc.AdminUpsertLocalizationParams{
		WorkspaceID: value.WorkspaceID, PromoID: value.PromoID, Locale: value.Locale,
		Title: value.Title, Description: value.Description,
	}); err != nil {
		return err
	}
	return r.invalidatePromoCache(value.WorkspaceID)
}

func (r *Repository) GetLocalization(ctx context.Context, workspaceID string, promoID uint64, locale string) (Localization, error) {
	key := promoCacheKey("admin_get_localization", workspaceID, promoID, locale)
	rememberPromoCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Localization, error) {
		row, err := r.q.AdminGetLocalization(ctx, promosqlc.AdminGetLocalizationParams{
			WorkspaceID: workspaceID, PromoID: promoID, Locale: locale,
		})
		if err != nil {
			return Localization{}, err
		}
		return mapLocalization(row), nil
	})
}

func (r *Repository) ListLocalizations(ctx context.Context, workspaceID string, promoID uint64) ([]Localization, error) {
	key := promoCacheKey("admin_list_localizations", workspaceID, promoID)
	rememberPromoCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Localization, error) {
		rows, err := r.q.AdminListLocalizations(ctx, promosqlc.AdminListLocalizationsParams{
			WorkspaceID: workspaceID, PromoID: promoID,
		})
		if err != nil {
			return nil, err
		}
		result := make([]Localization, 0, len(rows))
		for _, row := range rows {
			result = append(result, mapLocalization(row))
		}
		return result, nil
	})
}

func (r *Repository) DeleteLocalization(ctx context.Context, workspaceID string, promoID uint64, locale string) (int64, error) {
	rows, err := r.q.AdminDeleteLocalization(ctx, promosqlc.AdminDeleteLocalizationParams{
		WorkspaceID: workspaceID, PromoID: promoID, Locale: locale,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidatePromoCache(workspaceID)
}

func (r *Repository) UpsertReward(ctx context.Context, workspaceID string, promoID uint64, reward Reward) error {
	if err := r.q.AdminUpsertReward(ctx, promosqlc.AdminUpsertRewardParams{
		WorkspaceID: workspaceID, PromoID: promoID, RewardKey: reward.Key,
		RewardType: promosqlc.PromoRewardRewardType(reward.Type), Quantity: reward.Quantity,
		DurationUnit: promosqlc.NullPromoRewardDurationUnit{
			PromoRewardDurationUnit: promosqlc.PromoRewardDurationUnit(stringValue(reward.Unit)),
			Valid:                   reward.Unit != nil,
		},
	}); err != nil {
		return err
	}
	return r.invalidatePromoCache(workspaceID)
}

func (r *Repository) GetReward(ctx context.Context, workspaceID string, promoID uint64, key string) (Reward, error) {
	cacheKey := promoCacheKey("admin_get_reward", workspaceID, promoID, key)
	rememberPromoCacheKey(workspaceID, cacheKey)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          cacheKey,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Reward, error) {
		row, err := r.q.AdminGetReward(ctx, promosqlc.AdminGetRewardParams{
			WorkspaceID: workspaceID, PromoID: promoID, RewardKey: key,
		})
		if err != nil {
			return Reward{}, err
		}
		return mapReward(row), nil
	})
}

func (r *Repository) ListRewards(ctx context.Context, workspaceID string, promoID uint64) ([]Reward, error) {
	key := promoCacheKey("list_rewards", workspaceID, promoID)
	rememberPromoCacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Reward, error) {
		rows, err := r.q.ListRewards(ctx, promosqlc.ListRewardsParams{WorkspaceID: workspaceID, PromoID: promoID})
		if err != nil {
			return nil, err
		}
		result := make([]Reward, 0, len(rows))
		for _, row := range rows {
			result = append(result, mapReward(row))
		}
		return result, nil
	})
}

func mapReward(row promosqlc.PromoReward) Reward {
	return Reward{
		Key:      row.RewardKey,
		Type:     string(row.RewardType),
		Quantity: row.Quantity,
		Unit:     promoDurationUnitPtr(row.DurationUnit),
	}
}

func promoDurationUnitPtr(value promosqlc.NullPromoRewardDurationUnit) *string {
	if !value.Valid {
		return nil
	}
	unit := string(value.PromoRewardDurationUnit)
	return &unit
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (r *Repository) DeleteReward(ctx context.Context, workspaceID string, promoID uint64, key string) (int64, error) {
	rows, err := r.q.AdminDeleteReward(ctx, promosqlc.AdminDeleteRewardParams{
		WorkspaceID: workspaceID, PromoID: promoID, RewardKey: key,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidatePromoCache(workspaceID)
}

func mapPromo(row promosqlc.PromoOffer) Promo {
	return Promo{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Code: row.Code, Payload: row.Payload, Target: row.Target,
		MaxActivations: row.MaxActivations, ActivationCount: row.ActivationCount,
		IsActive: row.IsActive, StartAt: sqlwrap.NullTimePtr(row.StartAt),
		EndAt: sqlwrap.NullTimePtr(row.EndAt), DeletedAt: sqlwrap.NullTimePtr(row.DeletedAt),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func mapLocalization(row promosqlc.PromoLocalization) Localization {
	return Localization{
		WorkspaceID: row.WorkspaceID, PromoID: row.PromoID, Locale: row.Locale,
		Title: row.Title, Description: row.Description,
	}
}

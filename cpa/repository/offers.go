package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"

	cpasqlc "github.com/elum-utils/services/cpa/sqlc"
)

type UpsertOfferParams struct {
	WorkspaceID       string
	ID                string
	Payload           json.RawMessage
	CodeMode          string
	CodeSource        *string
	SharedCode        *string
	GeneratedLength   *int16
	GeneratedAlphabet *string
	IsActive          bool
	StartAt           *time.Time
	EndAt             *time.Time
}

func (r *Repository) UpsertOffer(ctx context.Context, params UpsertOfferParams) error {
	if err := requireScope(params.WorkspaceID, params.ID); err != nil {
		return err
	}
	if err := r.q.AdminUpsertOffer(ctx, cpasqlc.AdminUpsertOfferParams{
		WorkspaceID: params.WorkspaceID,
		ID:          params.ID,
		Payload:     params.Payload,
		CodeMode:    cpasqlc.CpaOfferCodeMode(params.CodeMode),
		CodeSource: sqlwrap.NullFromPtr(params.CodeSource, func(v string) cpasqlc.NullCpaOfferCodeSource {
			return cpasqlc.NullCpaOfferCodeSource{
				CpaOfferCodeSource: cpasqlc.CpaOfferCodeSource(v),
				Valid:              true,
			}
		}),
		SharedCode: sqlwrap.NullFromPtr(params.SharedCode, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		GeneratedLength: sqlwrap.NullFromPtr(params.GeneratedLength, func(v int16) sql.NullInt16 {
			return sql.NullInt16{Int16: v, Valid: true}
		}),
		GeneratedAlphabet: sqlwrap.NullFromPtr(params.GeneratedAlphabet, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		IsActive: params.IsActive,
		StartAt: sqlwrap.NullFromPtr(params.StartAt, func(v time.Time) sql.NullTime {
			return sql.NullTime{Time: v, Valid: true}
		}),
		EndAt: sqlwrap.NullFromPtr(params.EndAt, func(v time.Time) sql.NullTime {
			return sql.NullTime{Time: v, Valid: true}
		}),
	}); err != nil {
		return err
	}
	return r.invalidateCPACache(params.WorkspaceID)
}

func (r *Repository) GetOffer(ctx context.Context, workspaceID, cpaID string) (Offer, error) {
	if err := requireScope(workspaceID, cpaID); err != nil {
		return Offer{}, err
	}
	key := cpaCacheKey("admin_get_offer", workspaceID, cpaID)
	rememberCPACacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Offer, error) {
		row, err := r.q.AdminGetOffer(ctx, cpasqlc.AdminGetOfferParams{WorkspaceID: workspaceID, ID: cpaID})
		if err != nil {
			return Offer{}, err
		}
		return mapOffer(row), nil
	})
}

func (r *Repository) ListOffers(ctx context.Context, workspaceID string, limit, offset int32) ([]Offer, error) {
	if workspaceID == "" {
		return nil, ErrWorkspaceRequired
	}
	limit, offset = normalizePage(limit, offset)
	key := cpaCacheKey("admin_list_offers", workspaceID, limit, offset)
	rememberCPACacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Offer, error) {
		rows, err := r.q.AdminListOffers(ctx, cpasqlc.AdminListOffersParams{
			WorkspaceID: workspaceID,
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			return nil, err
		}
		return mapOffers(rows), nil
	})
}

func (r *Repository) ListActiveOffers(ctx context.Context, workspaceID string) ([]Offer, error) {
	if workspaceID == "" {
		return nil, ErrWorkspaceRequired
	}
	rows, err := r.q.ListActiveOffers(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return mapOffers(rows), nil
}

func (r *Repository) ListOfferBundles(ctx context.Context, workspaceID string, limit, offset int32) ([]OfferBundle, error) {
	if workspaceID == "" {
		return nil, ErrWorkspaceRequired
	}
	limit, offset = normalizePage(limit, offset)
	key := cpaCacheKey("admin_list_offer_bundles", workspaceID, limit, offset)
	rememberCPACacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]OfferBundle, error) {
		rows, err := r.q.AdminListOfferBundles(ctx, cpasqlc.AdminListOfferBundlesParams{
			WorkspaceID: workspaceID,
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			return nil, err
		}
		rewardRows, err := r.q.AdminListOfferBundleRewards(ctx, cpasqlc.AdminListOfferBundleRewardsParams{
			WorkspaceID: workspaceID,
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			return nil, err
		}
		result := make([]OfferBundle, 0, int(limit))
		indexByID := make(map[string]int, int(limit))
		for _, row := range rows {
			index, exists := indexByID[row.ID]
			if !exists {
				index = len(result)
				indexByID[row.ID] = index
				result = append(result, OfferBundle{
					Offer:         mapBundleOffer(row.WorkspaceID, row.ID, row.Payload, row.CodeMode, row.CodeSource, row.SharedCode, row.GeneratedLength, row.GeneratedAlphabet, row.IsActive, row.StartAt, row.EndAt, row.CreatedAt, row.UpdatedAt),
					Localizations: make([]Localization, 0),
					Rewards:       make([]Reward, 0),
				})
			}
			if row.Locale.Valid {
				result[index].Localizations = append(result[index].Localizations, Localization{
					WorkspaceID: row.WorkspaceID,
					CPAID:       row.ID,
					Locale:      row.Locale.String,
					Title:       row.LocalizationTitle.String,
					Description: row.LocalizationDescription.String,
				})
			}
		}
		for _, row := range rewardRows {
			index, exists := indexByID[row.CpaID]
			if !exists {
				continue
			}
			result[index].Rewards = append(result[index].Rewards, Reward{
				WorkspaceID: row.WorkspaceID,
				CPAID:       row.CpaID,
				Key:         row.RewardKey,
				Type:        string(row.RewardType),
				Quantity:    row.RewardQuantity,
				Unit:        cpaDurationUnitPtr(row.DurationUnit),
			})
		}
		for index := range result {
			sort.Slice(result[index].Localizations, func(i, j int) bool {
				return result[index].Localizations[i].Locale < result[index].Localizations[j].Locale
			})
		}
		return result, nil
	})
}

func (r *Repository) ListActiveOfferBundles(ctx context.Context, scope UserScope, locale string) ([]OfferBundle, error) {
	if scope.WorkspaceID == "" {
		return nil, ErrWorkspaceRequired
	}
	rows, err := r.q.ListActiveOfferBundles(ctx, cpasqlc.ListActiveOfferBundlesParams{
		Locale:         locale,
		AppID:          scope.AppID,
		PlatformID:     scope.PlatformID,
		PlatformUserID: scope.PlatformUserID,
		WorkspaceID:    scope.WorkspaceID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]OfferBundle, 0, len(rows))
	indexByID := make(map[string]int, len(rows))
	for _, row := range rows {
		index, exists := indexByID[row.ID]
		if !exists {
			bundle := OfferBundle{
				Offer: mapBundleOffer(row.WorkspaceID, row.ID, row.Payload, row.CodeMode, row.CodeSource, row.SharedCode, row.GeneratedLength, row.GeneratedAlphabet, row.IsActive, row.StartAt, row.EndAt, row.CreatedAt, row.UpdatedAt),
				Localization: &Localization{
					WorkspaceID: row.WorkspaceID,
					CPAID:       row.ID,
					Locale:      row.LocalizedLocale.String,
					Title:       row.LocalizedTitle.String,
					Description: row.LocalizedDescription.String,
				},
				Rewards: make([]Reward, 0),
			}
			if row.AssignmentID.Valid {
				bundle.Assignment = &Assignment{
					ID:             uint64(row.AssignmentID.Int64),
					WorkspaceID:    row.WorkspaceID,
					CPAID:          row.ID,
					AppID:          scope.AppID,
					PlatformID:     scope.PlatformID,
					PlatformUserID: scope.PlatformUserID,
					Code:           row.AssignmentCode.String,
					CodeMode:       string(row.AssignmentCodeMode.CpaAssignmentCodeMode),
					Status:         string(row.AssignmentStatus.CpaAssignmentStatus),
					IssuedAt:       row.AssignmentIssuedAt.Time,
					CompletedAt:    sqlwrap.NullTimePtr(row.AssignmentCompletedAt),
				}
			}
			index = len(result)
			indexByID[row.ID] = index
			result = append(result, bundle)
		}
		if row.RewardKey.Valid {
			result[index].Rewards = append(result[index].Rewards, Reward{
				WorkspaceID: row.WorkspaceID,
				CPAID:       row.ID,
				Key:         row.RewardKey.String,
				Type:        string(row.RewardType.CpaRewardRewardType),
				Quantity:    row.RewardQuantity.Int64,
				Unit:        cpaDurationUnitPtr(row.DurationUnit),
			})
		}
	}
	return result, nil
}

func (r *Repository) DeleteOffer(ctx context.Context, workspaceID, cpaID string) (int64, error) {
	if err := requireScope(workspaceID, cpaID); err != nil {
		return 0, err
	}
	rows, err := r.q.AdminDeleteOffer(ctx, cpasqlc.AdminDeleteOfferParams{WorkspaceID: workspaceID, ID: cpaID})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCPACache(workspaceID)
}

func (r *Repository) UpsertLocalization(ctx context.Context, value Localization) error {
	if err := requireScope(value.WorkspaceID, value.CPAID); err != nil {
		return err
	}
	if err := r.q.AdminUpsertLocalization(ctx, cpasqlc.AdminUpsertLocalizationParams{
		WorkspaceID: value.WorkspaceID,
		CpaID:       value.CPAID,
		Locale:      value.Locale,
		Title:       value.Title,
		Description: value.Description,
	}); err != nil {
		return err
	}
	return r.invalidateCPACache(value.WorkspaceID)
}

func (r *Repository) GetLocalization(ctx context.Context, workspaceID, cpaID, locale string) (Localization, error) {
	key := cpaCacheKey("get_localization", workspaceID, cpaID, locale)
	rememberCPACacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Localization, error) {
		row, err := r.q.GetLocalization(ctx, cpasqlc.GetLocalizationParams{
			WorkspaceID: workspaceID,
			CpaID:       cpaID,
			Locale:      locale,
		})
		if err != nil {
			return Localization{}, err
		}
		return mapLocalization(row), nil
	})
}

func (r *Repository) ResolveLocalization(ctx context.Context, workspaceID, cpaID, locale string) (*Localization, error) {
	if locale != "" {
		value, err := r.GetLocalization(ctx, workspaceID, cpaID, locale)
		if err == nil {
			return &value, nil
		}
		if !isNoRows(err) {
			return nil, err
		}
	}
	values, err := r.ListLocalizations(ctx, workspaceID, cpaID)
	if err != nil || len(values) == 0 {
		return nil, err
	}
	return &values[0], nil
}

func (r *Repository) ListLocalizations(ctx context.Context, workspaceID, cpaID string) ([]Localization, error) {
	key := cpaCacheKey("list_localizations", workspaceID, cpaID)
	rememberCPACacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Localization, error) {
		rows, err := r.q.ListLocalizations(ctx, cpasqlc.ListLocalizationsParams{
			WorkspaceID: workspaceID,
			CpaID:       cpaID,
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

func (r *Repository) DeleteLocalization(ctx context.Context, workspaceID, cpaID, locale string) (int64, error) {
	rows, err := r.q.AdminDeleteLocalization(ctx, cpasqlc.AdminDeleteLocalizationParams{
		WorkspaceID: workspaceID,
		CpaID:       cpaID,
		Locale:      locale,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCPACache(workspaceID)
}

func (r *Repository) UpsertReward(ctx context.Context, value Reward) error {
	if err := r.q.AdminUpsertReward(ctx, cpasqlc.AdminUpsertRewardParams{
		WorkspaceID: value.WorkspaceID,
		CpaID:       value.CPAID,
		RewardKey:   value.Key,
		RewardType:  cpasqlc.CpaRewardRewardType(value.Type),
		Quantity:    value.Quantity,
		DurationUnit: cpasqlc.NullCpaRewardDurationUnit{
			CpaRewardDurationUnit: cpasqlc.CpaRewardDurationUnit(valueOrEmpty(value.Unit)),
			Valid:                 value.Unit != nil,
		},
	}); err != nil {
		return err
	}
	return r.invalidateCPACache(value.WorkspaceID)
}

func (r *Repository) ListRewards(ctx context.Context, workspaceID, cpaID string) ([]Reward, error) {
	key := cpaCacheKey("list_rewards", workspaceID, cpaID)
	rememberCPACacheKey(workspaceID, key)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key:          key,
		Timeout:      r.timeout,
		CacheL1Delay: r.cacheL1,
		CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Reward, error) {
		rows, err := r.q.ListRewards(ctx, cpasqlc.ListRewardsParams{
			WorkspaceID: workspaceID,
			CpaID:       cpaID,
		})
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

func (r *Repository) DeleteReward(ctx context.Context, workspaceID, cpaID, rewardKey string) (int64, error) {
	rows, err := r.q.AdminDeleteReward(ctx, cpasqlc.AdminDeleteRewardParams{
		WorkspaceID: workspaceID,
		CpaID:       cpaID,
		RewardKey:   rewardKey,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateCPACache(workspaceID)
}

func mapOffer(row cpasqlc.CpaOffer) Offer {
	return Offer{
		WorkspaceID:       row.WorkspaceID,
		ID:                row.ID,
		Payload:           row.Payload,
		CodeMode:          string(row.CodeMode),
		CodeSource:        nullCodeSourcePtr(row.CodeSource),
		SharedCode:        sqlwrap.NullStringPtr(row.SharedCode),
		GeneratedLength:   nullInt16Ptr(row.GeneratedLength),
		GeneratedAlphabet: sqlwrap.NullStringPtr(row.GeneratedAlphabet),
		IsActive:          row.IsActive,
		StartAt:           sqlwrap.NullTimePtr(row.StartAt),
		EndAt:             sqlwrap.NullTimePtr(row.EndAt),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func mapOffers(rows []cpasqlc.CpaOffer) []Offer {
	result := make([]Offer, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapOffer(row))
	}
	return result
}

func mapLocalization(row cpasqlc.CpaLocalization) Localization {
	return Localization{
		WorkspaceID: row.WorkspaceID,
		CPAID:       row.CpaID,
		Locale:      row.Locale,
		Title:       row.Title,
		Description: row.Description,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapReward(row cpasqlc.CpaReward) Reward {
	return Reward{
		WorkspaceID: row.WorkspaceID,
		CPAID:       row.CpaID,
		Key:         row.RewardKey,
		Type:        string(row.RewardType),
		Quantity:    row.Quantity,
		Unit:        cpaDurationUnitPtr(row.DurationUnit),
	}
}

func cpaDurationUnitPtr(value cpasqlc.NullCpaRewardDurationUnit) *string {
	if !value.Valid {
		return nil
	}
	unit := string(value.CpaRewardDurationUnit)
	return &unit
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mapBundleOffer(
	workspaceID string,
	id string,
	payload json.RawMessage,
	codeMode cpasqlc.CpaOfferCodeMode,
	codeSource cpasqlc.NullCpaOfferCodeSource,
	sharedCode sql.NullString,
	generatedLength sql.NullInt16,
	generatedAlphabet sql.NullString,
	isActive bool,
	startAt sql.NullTime,
	endAt sql.NullTime,
	createdAt time.Time,
	updatedAt time.Time,
) Offer {
	return mapOffer(cpasqlc.CpaOffer{
		WorkspaceID:       workspaceID,
		ID:                id,
		Payload:           payload,
		CodeMode:          codeMode,
		CodeSource:        codeSource,
		SharedCode:        sharedCode,
		GeneratedLength:   generatedLength,
		GeneratedAlphabet: generatedAlphabet,
		IsActive:          isActive,
		StartAt:           startAt,
		EndAt:             endAt,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	})
}

func nullCodeSourcePtr(value cpasqlc.NullCpaOfferCodeSource) *string {
	if !value.Valid {
		return nil
	}
	result := string(value.CpaOfferCodeSource)
	return &result
}

func nullInt16Ptr(value sql.NullInt16) *int16 {
	if !value.Valid {
		return nil
	}
	return &value.Int16
}

func normalizePage(limit, offset int32) (int32, int32) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

package repository

import (
	"context"

	refsqlc "github.com/elum-utils/services/reference/sqlc"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

func (r *Repository) CreateItem(ctx context.Context, params SaveItemParams) error {
	if err := requireWorkspace(params.WorkspaceID); err != nil {
		return err
	}
	if err := r.q.AdminCreateItem(ctx, refsqlc.AdminCreateItemParams{
		WorkspaceID: params.WorkspaceID, Key: params.Key,
		ItemType: refsqlc.ReferenceItemItemType(params.Type),
		Payload:  params.Payload, IsActive: params.IsActive,
	}); err != nil {
		return err
	}
	return r.invalidateWorkspaceCache(params.WorkspaceID)
}

func (r *Repository) UpdateItem(ctx context.Context, params SaveItemParams) (int64, error) {
	if err := requireWorkspace(params.WorkspaceID); err != nil {
		return 0, err
	}
	rows, err := r.q.AdminUpdateItem(ctx, refsqlc.AdminUpdateItemParams{
		Payload: params.Payload, IsActive: params.IsActive,
		WorkspaceID: params.WorkspaceID, Key: params.Key,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateWorkspaceCache(params.WorkspaceID)
}

func (r *Repository) DangerousChangeType(ctx context.Context, params DangerousChangeTypeParams) (int64, error) {
	if err := requireWorkspace(params.WorkspaceID); err != nil {
		return 0, err
	}
	rows, err := r.q.AdminDangerousChangeType(ctx, refsqlc.AdminDangerousChangeTypeParams{
		ItemType:    refsqlc.ReferenceItemItemType(params.NewType),
		WorkspaceID: params.WorkspaceID, Key: params.Key,
		ItemType_2: refsqlc.ReferenceItemItemType(params.CurrentType),
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateWorkspaceCache(params.WorkspaceID)
}

func (r *Repository) SoftDeleteItem(ctx context.Context, workspaceID, key string) (int64, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return 0, err
	}
	rows, err := r.q.AdminSoftDeleteItem(ctx, refsqlc.AdminSoftDeleteItemParams{
		WorkspaceID: workspaceID, Key: key,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateWorkspaceCache(workspaceID)
}

func (r *Repository) RestoreItem(ctx context.Context, workspaceID, key string, active bool) (int64, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return 0, err
	}
	rows, err := r.q.AdminRestoreItem(ctx, refsqlc.AdminRestoreItemParams{
		IsActive: active, WorkspaceID: workspaceID, Key: key,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateWorkspaceCache(workspaceID)
}

func (r *Repository) AdminGetItem(ctx context.Context, workspaceID, key string) (Item, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return Item{}, err
	}
	cacheKey := referenceCacheKey("admin_get", workspaceID, key)
	rememberReferenceCacheKey(workspaceID, cacheKey)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key: cacheKey, Timeout: r.timeout,
		CacheL1Delay: r.cacheL1, CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Item, error) {
		rows, err := r.q.AdminGetItemBundle(ctx, refsqlc.AdminGetItemBundleParams{
			WorkspaceID: workspaceID, Key: key,
		})
		if err != nil {
			return Item{}, err
		}
		if len(rows) == 0 {
			return Item{}, ErrItemNotFound
		}
		first := rows[0]
		item := Item{
			WorkspaceID: first.WorkspaceID, Key: first.Key, Type: string(first.ItemType),
			Payload: first.Payload, IsActive: first.IsActive,
			DeletedAt: sqlwrap.NullTimePtr(first.DeletedAt),
			CreatedAt: first.CreatedAt, UpdatedAt: first.UpdatedAt,
			Localizations: make([]Localization, 0, len(rows)),
		}
		for _, row := range rows {
			if row.Locale.Valid {
				item.Localizations = append(item.Localizations, Localization{
					WorkspaceID: row.WorkspaceID, ItemKey: row.Key,
					Locale: row.Locale.String, Title: row.Title.String,
					Description: row.Description.String,
				})
			}
		}
		return item, nil
	})
}

func (r *Repository) AdminListItems(ctx context.Context, params ListItemsParams) ([]Item, error) {
	if err := requireWorkspace(params.WorkspaceID); err != nil {
		return nil, err
	}
	cacheKey := referenceCacheKey(
		"admin_list", params.WorkspaceID, params.Type,
		params.OnlyNotDeleted, params.Limit, params.Offset,
	)
	rememberReferenceCacheKey(params.WorkspaceID, cacheKey)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key: cacheKey, Timeout: r.timeout,
		CacheL1Delay: r.cacheL1, CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Item, error) {
		rows, err := r.q.AdminListItems(ctx, refsqlc.AdminListItemsParams{
			WorkspaceID: params.WorkspaceID,
			Column2:     params.Type, ItemType: refsqlc.ReferenceItemItemType(params.Type),
			Column4: params.OnlyNotDeleted, Limit: params.Limit, Offset: params.Offset,
		})
		if err != nil {
			return nil, err
		}
		result := make([]Item, 0, len(rows))
		for _, row := range rows {
			result = append(result, Item{
				WorkspaceID: row.WorkspaceID, Key: row.Key, Type: string(row.ItemType),
				Payload: row.Payload, IsActive: row.IsActive,
				DeletedAt: sqlwrap.NullTimePtr(row.DeletedAt),
				CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			})
		}
		return result, nil
	})
}

func (r *Repository) UpsertLocalization(ctx context.Context, value Localization) error {
	if err := requireWorkspace(value.WorkspaceID); err != nil {
		return err
	}
	if err := r.q.AdminUpsertLocalization(ctx, refsqlc.AdminUpsertLocalizationParams{
		WorkspaceID: value.WorkspaceID, ItemKey: value.ItemKey, Locale: value.Locale,
		Title: value.Title, Description: value.Description,
	}); err != nil {
		return err
	}
	return r.invalidateWorkspaceCache(value.WorkspaceID)
}

func (r *Repository) GetLocalization(ctx context.Context, workspaceID, key, locale string) (Localization, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return Localization{}, err
	}
	cacheKey := referenceCacheKey("admin_get_localization", workspaceID, key, locale)
	rememberReferenceCacheKey(workspaceID, cacheKey)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key: cacheKey, Timeout: r.timeout,
		CacheL1Delay: r.cacheL1, CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Localization, error) {
		row, err := r.q.AdminGetLocalization(ctx, refsqlc.AdminGetLocalizationParams{
			WorkspaceID: workspaceID, ItemKey: key, Locale: locale,
		})
		if err != nil {
			return Localization{}, err
		}
		return mapLocalization(row), nil
	})
}

func (r *Repository) ListLocalizations(ctx context.Context, workspaceID, key string) ([]Localization, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return nil, err
	}
	cacheKey := referenceCacheKey("admin_list_localizations", workspaceID, key)
	rememberReferenceCacheKey(workspaceID, cacheKey)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key: cacheKey, Timeout: r.timeout,
		CacheL1Delay: r.cacheL1, CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) ([]Localization, error) {
		rows, err := r.q.AdminListLocalizations(ctx, refsqlc.AdminListLocalizationsParams{
			WorkspaceID: workspaceID, ItemKey: key,
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

func (r *Repository) DeleteLocalization(ctx context.Context, workspaceID, key, locale string) (int64, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return 0, err
	}
	rows, err := r.q.AdminDeleteLocalization(ctx, refsqlc.AdminDeleteLocalizationParams{
		WorkspaceID: workspaceID, ItemKey: key, Locale: locale,
	})
	if err != nil || rows == 0 {
		return rows, err
	}
	return rows, r.invalidateWorkspaceCache(workspaceID)
}

func (r *Repository) GetStats(ctx context.Context, workspaceID string) (Stats, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return Stats{}, err
	}
	cacheKey := referenceCacheKey("admin_stats", workspaceID)
	rememberReferenceCacheKey(workspaceID, cacheKey)
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{
		Key: cacheKey, Timeout: r.timeout,
		CacheL1Delay: r.cacheL1, CacheL2Delay: r.cacheL2,
	}, func(ctx context.Context) (Stats, error) {
		row, err := r.q.AdminGetStats(ctx, workspaceID)
		if err != nil {
			return Stats{}, err
		}
		return Stats{
			ItemsTotal: uint64(row.ItemsTotal), ItemsNotDeleted: uint64(row.ItemsNotDeleted),
			ActiveItems: uint64(row.ActiveItems), DeletedItems: uint64(row.DeletedItems),
			QuantityItems: uint64(row.QuantityItems), DurationItems: uint64(row.DurationItems),
		}, nil
	})
}

func mapLocalization(row refsqlc.ReferenceLocalization) Localization {
	return Localization{
		WorkspaceID: row.WorkspaceID, ItemKey: row.ItemKey, Locale: row.Locale,
		Title: row.Title, Description: row.Description,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

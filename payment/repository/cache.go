package repository

import (
	"context"
	"sync"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

const paymentGlobalCacheScope = "*"

var paymentCacheKeys sync.Map

func paymentCacheKey(parts ...any) string {
	args := append([]any{"payment"}, parts...)
	return sqlwrap.CreateKey(args...)
}

func rememberPaymentCacheKey(scope string, key string) {
	if scope == "" || key == "" {
		return
	}
	paymentCacheKeys.Store(key, scope)
}

func queryPaymentCache[T any](
	ctx context.Context,
	repository *PaymentRepository,
	scope string,
	key string,
	loader func(context.Context) (T, error),
) (T, error) {
	value, err := sqlwrap.Query(ctx, repository.db, sqlwrap.Params{
		Key:          key,
		Timeout:      repository.timeout,
		CacheL1Delay: repository.cacheL1,
		CacheL2Delay: repository.cacheL2,
	}, loader)
	if err == nil {
		rememberPaymentCacheKey(scope, key)
	}
	return value, err
}

func queryPaymentVersionedCache[T any](
	ctx context.Context,
	repository *PaymentRepository,
	scope string,
	versionScope []any,
	key string,
	loader func(context.Context) (T, error),
) (T, error) {
	value, err := sqlwrap.Query(ctx, repository.db, sqlwrap.Params{
		Key:               key,
		Timeout:           repository.timeout,
		CacheL1Delay:      repository.cacheL1,
		CacheL2Delay:      repository.cacheL2,
		CacheVersionScope: versionScope,
	}, loader)
	if err == nil {
		rememberPaymentCacheKey(scope, key)
	}
	return value, err
}

func paymentProductLimitConfigVersionScope(workspaceID string) []any {
	return []any{"payment", "product_limit_config", workspaceID}
}

func cloneSlice[T any](items []T) []T {
	if len(items) == 0 {
		return nil
	}
	out := make([]T, len(items))
	copy(out, items)
	return out
}

func InvalidateWorkspaceCache(db *sqlwrap.Client, workspaceID string) error {
	if db == nil || workspaceID == "" {
		return nil
	}
	outErr := db.BumpCacheVersion(paymentProductLimitConfigVersionScope(workspaceID)...)
	deleteErr := invalidatePaymentCache(db, func(scope string) bool {
		return scope == workspaceID
	})
	if outErr != nil {
		return outErr
	}
	return deleteErr
}

func InvalidateAllCache(db *sqlwrap.Client) error {
	if db == nil {
		return nil
	}
	return invalidatePaymentCache(db, func(string) bool {
		return true
	})
}

func invalidatePaymentCache(db *sqlwrap.Client, match func(scope string) bool) error {
	var outErr error
	paymentCacheKeys.Range(func(rawKey, rawScope any) bool {
		key, keyOK := rawKey.(string)
		scope, scopeOK := rawScope.(string)
		if !keyOK || !scopeOK || !match(scope) {
			return true
		}
		if err := db.DeleteCache(key); err != nil && outErr == nil {
			outErr = err
		}
		paymentCacheKeys.Delete(rawKey)
		return true
	})
	return outErr
}

func (r *PaymentRepository) invalidateWorkspaceCache(workspaceID string) error {
	if r == nil {
		return nil
	}
	return InvalidateWorkspaceCache(r.db, workspaceID)
}

func (r *PaymentRepository) invalidateAllCache() error {
	if r == nil {
		return nil
	}
	return InvalidateAllCache(r.db)
}

func (r *PaymentRepository) RebuildWorkspaceProductCache(ctx context.Context, workspaceID string) error {
	workspaceID, err := requireWorkspaceID(workspaceID)
	if err != nil {
		return err
	}
	err = r.inTransaction(ctx, func(tx *PaymentRepository) error {
		if _, err := tx.q.DeleteWorkspaceProductCache(ctx, workspaceID); err != nil {
			return err
		}
		return tx.q.RebuildWorkspaceProductCache(ctx, paymentsqlc.RebuildWorkspaceProductCacheParams{
			WorkspaceID:   workspaceID,
			WorkspaceID_2: workspaceID,
		})
	})
	if err != nil {
		return err
	}
	return r.invalidateWorkspaceCache(workspaceID)
}

func (r *PaymentRepository) RebuildProductCache(ctx context.Context, workspaceID string, productID string) error {
	workspaceID, err := requireWorkspaceID(workspaceID)
	if err != nil {
		return err
	}
	err = r.inTransaction(ctx, func(tx *PaymentRepository) error {
		if _, err := tx.q.DeleteProductCache(ctx, paymentsqlc.DeleteProductCacheParams{
			WorkspaceID: workspaceID,
			ProductID:   productID,
		}); err != nil {
			return err
		}
		return tx.q.RebuildProductCache(ctx, paymentsqlc.RebuildProductCacheParams{
			WorkspaceID:   workspaceID,
			WorkspaceID_2: workspaceID,
			ID:            productID,
		})
	})
	if err != nil {
		return err
	}
	return r.invalidateWorkspaceCache(workspaceID)
}

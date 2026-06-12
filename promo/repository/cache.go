package repository

import (
	"sync"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

var promoCacheKeys sync.Map

func promoCacheKey(parts ...any) string {
	args := append([]any{"promo"}, parts...)
	return sqlwrap.CreateKey(args...)
}

func rememberPromoCacheKey(workspaceID, key string) {
	if workspaceID == "" || key == "" {
		return
	}
	promoCacheKeys.Store(key, workspaceID)
}

func (r *Repository) invalidatePromoCache(workspaceID string) error {
	if r == nil || r.db == nil || workspaceID == "" {
		return nil
	}
	var outErr error
	promoCacheKeys.Range(func(rawKey, rawWorkspaceID any) bool {
		key, keyOK := rawKey.(string)
		cachedWorkspaceID, workspaceOK := rawWorkspaceID.(string)
		if !keyOK || !workspaceOK || cachedWorkspaceID != workspaceID {
			return true
		}
		if err := r.db.DeleteCache(key); err != nil && outErr == nil {
			outErr = err
		}
		promoCacheKeys.Delete(rawKey)
		return true
	})
	return outErr
}

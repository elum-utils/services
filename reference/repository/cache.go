package repository

import (
	"sync"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

var referenceCacheKeys sync.Map

func referenceCacheKey(parts ...any) string {
	return sqlwrap.CreateKey(append([]any{"reference"}, parts...)...)
}

func rememberReferenceCacheKey(workspaceID, key string) {
	if workspaceID != "" && key != "" {
		referenceCacheKeys.Store(key, workspaceID)
	}
}

func (r *Repository) invalidateWorkspaceCache(workspaceID string) error {
	if r == nil || r.db == nil || workspaceID == "" {
		return nil
	}
	var result error
	referenceCacheKeys.Range(func(rawKey, rawWorkspace any) bool {
		key, keyOK := rawKey.(string)
		cachedWorkspace, workspaceOK := rawWorkspace.(string)
		if !keyOK || !workspaceOK || cachedWorkspace != workspaceID {
			return true
		}
		if err := r.db.DeleteCache(key); err != nil && result == nil {
			result = err
		}
		referenceCacheKeys.Delete(rawKey)
		return true
	})
	return result
}

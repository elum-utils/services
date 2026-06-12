package repository

import (
	"sync"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

var cpaCacheKeys sync.Map

func cpaCacheKey(parts ...any) string {
	args := append([]any{"cpa"}, parts...)
	return sqlwrap.CreateKey(args...)
}

func rememberCPACacheKey(workspaceID, key string) {
	if workspaceID == "" || key == "" {
		return
	}
	cpaCacheKeys.Store(key, workspaceID)
}

func (r *Repository) invalidateCPACache(workspaceID string) error {
	if r == nil || r.db == nil || workspaceID == "" {
		return nil
	}
	var outErr error
	cpaCacheKeys.Range(func(rawKey, rawWorkspaceID any) bool {
		key, keyOK := rawKey.(string)
		cachedWorkspaceID, workspaceOK := rawWorkspaceID.(string)
		if !keyOK || !workspaceOK || cachedWorkspaceID != workspaceID {
			return true
		}
		if err := r.db.DeleteCache(key); err != nil && outErr == nil {
			outErr = err
		}
		cpaCacheKeys.Delete(rawKey)
		return true
	})
	return outErr
}

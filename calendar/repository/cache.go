package repository

import (
	"sync"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

var calendarCacheKeys sync.Map

func calendarCacheKey(parts ...any) string {
	args := append([]any{"calendar"}, parts...)
	return sqlwrap.CreateKey(args...)
}

func rememberCalendarCacheKey(workspaceID, key string) {
	if workspaceID == "" || key == "" {
		return
	}
	calendarCacheKeys.Store(key, workspaceID)
}

func (r *Repository) invalidateCalendarCache(workspaceID string) error {
	if r == nil || r.db == nil || workspaceID == "" {
		return nil
	}
	var outErr error
	calendarCacheKeys.Range(func(rawKey, rawWorkspaceID any) bool {
		key, keyOK := rawKey.(string)
		cachedWorkspaceID, workspaceOK := rawWorkspaceID.(string)
		if !keyOK || !workspaceOK || cachedWorkspaceID != workspaceID {
			return true
		}
		if err := r.db.DeleteCache(key); err != nil && outErr == nil {
			outErr = err
		}
		calendarCacheKeys.Delete(rawKey)
		return true
	})
	return outErr
}

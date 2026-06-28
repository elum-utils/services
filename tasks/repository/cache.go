package repository

import (
	"context"
	"sync"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

var taskCacheKeys sync.Map

func activeCatalogCacheKey(workspaceID, locale, groupKey string) string {
	return sqlwrap.CreateKey("tasks", "active_catalog", workspaceID, locale, groupKey)
}

func recordCatalogCacheKey(workspaceID, actionKey string) string {
	return sqlwrap.CreateKey("tasks", "record_catalog", workspaceID, actionKey)
}

func claimCatalogByIDCacheKey(workspaceID string, id uint64) string {
	return sqlwrap.CreateKey("tasks", "claim_catalog_id", workspaceID, id)
}

func claimCatalogByKeyCacheKey(workspaceID, key string) string {
	return sqlwrap.CreateKey("tasks", "claim_catalog_key", workspaceID, key)
}

func integrationCheckTaskByIDCacheKey(workspaceID string, id uint64) string {
	return sqlwrap.CreateKey("tasks", "integration_check_task_id", workspaceID, id)
}

func integrationCheckTaskByKeyCacheKey(workspaceID, key string) string {
	return sqlwrap.CreateKey("tasks", "integration_check_task_key", workspaceID, key)
}

func rewardsCatalogCacheKey(workspaceID string, taskID uint64) string {
	return sqlwrap.CreateKey("tasks", "rewards_catalog", workspaceID, taskID)
}

func nextSequenceTaskCacheKey(workspaceID, sequenceKey string, sequencePosition uint32) string {
	return sqlwrap.CreateKey("tasks", "next_sequence_task", workspaceID, sequenceKey, sequencePosition)
}

func adminGetTaskCacheKey(workspaceID string, id uint64) string {
	return sqlwrap.CreateKey("tasks", "admin_get_task", workspaceID, id)
}

func adminListTasksCacheKey(workspaceID, groupKey string, limit, offset int32) string {
	return sqlwrap.CreateKey("tasks", "admin_list_tasks", workspaceID, groupKey, limit, offset)
}

func rememberTaskCacheKey(workspaceID, key string) {
	if workspaceID == "" || key == "" {
		return
	}
	taskCacheKeys.Store(key, workspaceID)
}

func (r *Repository) invalidateTaskCache(_ context.Context, workspaceID string) error {
	if r == nil || r.db == nil || workspaceID == "" {
		return nil
	}
	var outErr error
	taskCacheKeys.Range(func(rawKey, rawWorkspaceID any) bool {
		key, keyOK := rawKey.(string)
		cachedWorkspaceID, workspaceOK := rawWorkspaceID.(string)
		if !keyOK || !workspaceOK || cachedWorkspaceID != workspaceID {
			return true
		}
		if err := r.db.DeleteCache(key); err != nil && outErr == nil {
			outErr = err
		}
		taskCacheKeys.Delete(rawKey)
		return true
	})
	return outErr
}

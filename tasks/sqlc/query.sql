-- name: AdminUpsertGroup :exec
INSERT INTO task_group (workspace_id, `key`, position, is_active)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE position = VALUES(position), is_active = VALUES(is_active), deleted_at = NULL;

-- name: AdminUpsertGroupLocalization :exec
INSERT INTO task_group_localization (workspace_id, group_key, locale, title, description)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE title = VALUES(title), description = VALUES(description);

-- name: AdminUpsertSequence :exec
INSERT INTO task_sequence (workspace_id, `key`, position, is_active)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE position = VALUES(position), is_active = VALUES(is_active), deleted_at = NULL;

-- name: AdminCreateTask :execlastid
INSERT INTO task_definition (
    workspace_id, `key`, group_key, sequence_key, sequence_position, task_kind,
    action_key, action_kind, claim_mode, target_count, reset_unit,
    reset_every, position, payload, target, integration_kind, integration_provider,
    integration_payload, image_url, is_visible, is_active,
    start_at, end_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AdminUpdateTask :execrows
UPDATE task_definition
SET group_key = ?, sequence_key = ?, sequence_position = ?, task_kind = ?, action_key = ?,
    action_kind = ?, claim_mode = ?, target_count = ?, reset_unit = ?,
    reset_every = ?, position = ?, payload = ?, target = ?, integration_kind = ?,
    integration_provider = ?, integration_payload = ?, image_url = ?,
    is_visible = ?, is_active = ?, start_at = ?, end_at = ?
WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL;

-- name: AdminDeleteTask :execrows
UPDATE task_definition
SET deleted_at = NOW(), is_active = FALSE, is_visible = FALSE
WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL;

-- name: AdminGetTask :one
SELECT id, workspace_id, `key`, group_key, sequence_key, sequence_position,
       task_kind, action_key, action_kind, claim_mode, target_count, reset_unit,
       reset_every, position, payload, target, integration_kind, integration_provider,
       integration_payload, image_url, is_visible, is_active,
       start_at, end_at, deleted_at, branch_sort_key, created_at, updated_at
FROM task_definition
WHERE workspace_id = ? AND id = ?
LIMIT 1;

-- name: AdminGetTaskByKey :one
SELECT id, workspace_id, `key`, group_key, sequence_key, sequence_position,
       task_kind, action_key, action_kind, claim_mode, target_count, reset_unit,
       reset_every, position, payload, target, integration_kind, integration_provider,
       integration_payload, image_url, is_visible, is_active,
       start_at, end_at, deleted_at, branch_sort_key, created_at, updated_at
FROM task_definition
WHERE workspace_id = ? AND `key` = ? AND deleted_at IS NULL
LIMIT 1;

-- name: AdminListGroups :many
SELECT workspace_id, `key`, position, is_active, deleted_at, created_at, updated_at
FROM task_group
WHERE workspace_id = ? AND deleted_at IS NULL
ORDER BY position, `key`;

-- name: AdminListGroupLocalizations :many
SELECT workspace_id, group_key, locale, title, description, created_at, updated_at
FROM task_group_localization
WHERE workspace_id = ?
ORDER BY group_key, locale;

-- name: AdminListSequences :many
SELECT workspace_id, `key`, position, is_active, deleted_at, created_at, updated_at
FROM task_sequence
WHERE workspace_id = ? AND deleted_at IS NULL
ORDER BY position, `key`;

-- name: AdminListTaskLocalizations :many
SELECT workspace_id, task_id, locale, title, description, created_at, updated_at
FROM task_localization
WHERE workspace_id = ?
ORDER BY task_id, locale;

-- name: AdminListAllRewards :many
SELECT id, workspace_id, task_id, reward_key, reward_type, quantity, scale, duration_unit, position, created_at, updated_at
FROM task_reward
WHERE workspace_id = ?
ORDER BY task_id, position, id;

-- name: AdminListPartnerRewardRules :many
SELECT workspace_id, provider, group_key, external_type, reward_key,
       reward_type, quantity, scale, duration_unit, position, is_enabled, created_at, updated_at
FROM task_partner_reward_rule
WHERE workspace_id = ?
ORDER BY group_key, provider, external_type, position, reward_key;

-- name: AdminListTasks :many
SELECT id, workspace_id, `key`, group_key, sequence_key, sequence_position,
       task_kind, action_key, action_kind, claim_mode, target_count, reset_unit,
       reset_every, position, payload, target, integration_kind, integration_provider,
       integration_payload, image_url, is_visible, is_active,
       start_at, end_at, deleted_at, branch_sort_key, created_at, updated_at
FROM task_definition
WHERE workspace_id = ? AND deleted_at IS NULL
ORDER BY position, id
LIMIT ? OFFSET ?;

-- name: AdminListTasksByGroup :many
SELECT id, workspace_id, `key`, group_key, sequence_key, sequence_position,
       task_kind, action_key, action_kind, claim_mode, target_count, reset_unit,
       reset_every, position, payload, target, integration_kind, integration_provider,
       integration_payload, image_url, is_visible, is_active,
       start_at, end_at, deleted_at, branch_sort_key, created_at, updated_at
FROM task_definition
WHERE workspace_id = ? AND group_key = ? AND deleted_at IS NULL
ORDER BY position, id
LIMIT ? OFFSET ?;

-- name: ExportListTasks :many
SELECT id, workspace_id, `key`, group_key, sequence_key, sequence_position,
       task_kind, action_key, action_kind, claim_mode, target_count, reset_unit,
       reset_every, position, payload, target, integration_kind, integration_provider,
       integration_payload, image_url, is_visible, is_active,
       start_at, end_at, deleted_at, branch_sort_key, created_at, updated_at
FROM task_definition
WHERE workspace_id = ? AND deleted_at IS NULL
ORDER BY group_key, position, id;

-- name: AdminUpsertTaskLocalization :exec
INSERT INTO task_localization (workspace_id, task_id, locale, title, description)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE title = VALUES(title), description = VALUES(description);

-- name: AdminUpsertReward :exec
INSERT INTO task_reward (
    workspace_id, task_id, reward_key, reward_type, quantity, scale, duration_unit, position
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    reward_type = VALUES(reward_type),
    quantity = VALUES(quantity),
    scale = VALUES(scale),
    duration_unit = VALUES(duration_unit),
    position = VALUES(position);

-- name: AdminDeleteReward :execrows
DELETE FROM task_reward
WHERE workspace_id = ? AND task_id = ? AND reward_key = ?;

-- name: ListRecordTasks :many
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count, t.reset_unit,
       t.reset_every, t.payload, t.target, t.branch_sort_key, t.position
FROM task_definition t FORCE INDEX (task_definition_action_idx)
WHERE t.workspace_id = ?
  AND t.action_key = ?
  AND t.sequence_key IS NULL
  AND t.is_active = TRUE
  AND t.deleted_at IS NULL
  AND (t.start_at IS NULL OR t.start_at <= ?)
  AND (t.end_at IS NULL OR t.end_at > ?)
UNION ALL
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count, t.reset_unit,
       t.reset_every, t.payload, t.target, t.branch_sort_key, t.position
FROM task_sequence_state s
JOIN task_definition t
  ON t.workspace_id = s.workspace_id AND t.id = s.current_task_id
WHERE s.workspace_id = ?
  AND s.app_id = ?
  AND s.platform_id = ?
  AND s.platform_user_id = ?
  AND s.status = 'active'
  AND t.action_key = ?
  AND t.is_active = TRUE
  AND t.deleted_at IS NULL
  AND (t.start_at IS NULL OR t.start_at <= ?)
  AND (t.end_at IS NULL OR t.end_at > ?)
UNION ALL
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count, t.reset_unit,
       t.reset_every, t.payload, t.target, t.branch_sort_key, t.position
FROM task_definition t FORCE INDEX (task_definition_action_idx)
LEFT JOIN task_sequence_state s
  ON s.workspace_id = t.workspace_id
 AND s.sequence_key = t.sequence_key
 AND s.app_id = ?
 AND s.platform_id = ?
 AND s.platform_user_id = ?
WHERE t.workspace_id = ?
  AND t.action_key = ?
  AND t.sequence_key IS NOT NULL
  AND t.sequence_position = 1
  AND s.sequence_key IS NULL
  AND t.is_active = TRUE
  AND t.deleted_at IS NULL
  AND (t.start_at IS NULL OR t.start_at <= ?)
  AND (t.end_at IS NULL OR t.end_at > ?)
ORDER BY branch_sort_key, sequence_position, position, id;

-- name: ListRecordCatalog :many
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count, t.reset_unit,
       t.reset_every, t.payload, t.target, t.position, t.start_at, t.end_at
FROM task_definition t FORCE INDEX (task_definition_action_idx)
WHERE t.workspace_id = ?
  AND t.action_key = ?
  AND t.is_active = TRUE
  AND t.deleted_at IS NULL
ORDER BY t.branch_sort_key, t.sequence_position, t.position, t.id;

-- name: GetNextSequenceTaskID :one
SELECT id
FROM task_definition
WHERE workspace_id = ?
  AND sequence_key = ?
  AND sequence_position > ?
  AND deleted_at IS NULL
ORDER BY sequence_position, id
LIMIT 1;

-- name: ListSequenceStatesForUser :many
SELECT sequence_key, current_task_id
FROM task_sequence_state
WHERE workspace_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND status = 'active';

-- name: GetSequenceStateForUpdate :one
SELECT current_task_id, status
FROM task_sequence_state
WHERE workspace_id = ?
  AND sequence_key = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
FOR UPDATE;

-- name: UpsertSequenceState :exec
INSERT INTO task_sequence_state (
    workspace_id, sequence_key, app_id, platform_id, platform_user_id,
    current_task_id, status
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    current_task_id = VALUES(current_task_id),
    status = VALUES(status);

-- name: ListCurrentProgressForUserForUpdate :many
SELECT id, workspace_id, task_id, app_id, platform_id, platform_user_id,
       period_start_at, period_end_at, progress, status, ready_at, claimed_at,
       operation_id, COALESCE(rewards_snapshot, JSON_ARRAY()) AS rewards_snapshot, created_at, updated_at
FROM task_progress
WHERE workspace_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND period_start_at <= ?
  AND period_end_at > ?
FOR UPDATE;

-- name: GetCurrentProgressForUpdate :one
SELECT id, workspace_id, task_id, app_id, platform_id, platform_user_id,
       period_start_at, period_end_at, progress, status, ready_at, claimed_at,
       operation_id, COALESCE(rewards_snapshot, JSON_ARRAY()) AS rewards_snapshot, created_at, updated_at
FROM task_progress
WHERE workspace_id = ?
  AND task_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND period_start_at <= ?
  AND period_end_at > ?
LIMIT 1
FOR UPDATE;

-- name: ListCurrentProgressForUser :many
SELECT id, workspace_id, task_id, app_id, platform_id, platform_user_id,
       period_start_at, period_end_at, progress, status, ready_at, claimed_at,
       operation_id, COALESCE(rewards_snapshot, JSON_ARRAY()) AS rewards_snapshot, created_at, updated_at
FROM task_progress
WHERE workspace_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND period_start_at <= ?
  AND period_end_at > ?;

-- name: EnsureProgress :execlastid
INSERT INTO task_progress (
    workspace_id, task_id, app_id, platform_id, platform_user_id,
    period_start_at, period_end_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id), period_end_at = VALUES(period_end_at);

-- name: UpsertProgress :execrows
INSERT INTO task_progress (
    workspace_id, task_id, app_id, platform_id, platform_user_id,
    period_start_at, period_end_at, progress, status, ready_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    period_end_at = VALUES(period_end_at),
    progress = VALUES(progress),
    status = VALUES(status),
    ready_at = VALUES(ready_at);

-- name: UpdateProgress :execrows
UPDATE task_progress
SET progress = ?, status = ?, ready_at = ?, claimed_at = ?,
    operation_id = ?, rewards_snapshot = ?
WHERE id = ?;

-- name: InsertProgressEvent :execrows
INSERT IGNORE INTO task_progress_event (
    workspace_id, app_id, platform_id, platform_user_id,
    source, external_event_key, action_key, amount, payload
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, sqlc.narg(payload));

-- name: CountProgressEventsByExternalKey :one
SELECT COUNT(*)
FROM task_progress_event
WHERE workspace_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND source = ?
  AND external_event_key = ?;

-- name: ListRewards :many
SELECT id, workspace_id, task_id, reward_key, reward_type, quantity, scale, duration_unit, position, created_at, updated_at
FROM task_reward
WHERE workspace_id = ? AND task_id = ?
ORDER BY position, id;

-- name: ListRewardsCatalog :many
SELECT reward_key, reward_type, quantity, scale, duration_unit
FROM task_reward
WHERE workspace_id = ? AND task_id = ?
ORDER BY position, id;

-- name: GetClaimCatalogByID :many
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.payload, t.target, t.integration_kind, t.integration_provider, t.integration_payload, t.image_url,
       r.id AS reward_id, r.reward_key, r.reward_type, r.quantity AS reward_quantity, r.scale AS reward_scale, r.duration_unit
FROM task_definition t
LEFT JOIN task_reward r ON r.workspace_id = t.workspace_id AND r.task_id = t.id
WHERE t.workspace_id = ? AND t.id = ?
ORDER BY r.position, r.id;

-- name: GetClaimCatalogByKey :many
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.payload, t.target, t.integration_kind, t.integration_provider, t.integration_payload, t.image_url,
       r.id AS reward_id, r.reward_key, r.reward_type, r.quantity AS reward_quantity, r.scale AS reward_scale, r.duration_unit
FROM task_definition t
LEFT JOIN task_reward r ON r.workspace_id = t.workspace_id AND r.task_id = t.id
WHERE t.workspace_id = ? AND t.`key` = ?
ORDER BY r.position, r.id;

-- name: GetIntegrationCheckTaskByID :one
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.reset_unit, t.reset_every, t.payload, t.target, t.integration_kind, t.integration_provider,
       t.integration_payload, t.image_url, t.start_at, t.end_at
FROM task_definition t
WHERE t.workspace_id = ? AND t.id = ? AND t.is_active = TRUE AND t.deleted_at IS NULL;

-- name: GetIntegrationCheckTaskByKey :one
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.reset_unit, t.reset_every, t.payload, t.target, t.integration_kind, t.integration_provider,
       t.integration_payload, t.image_url, t.start_at, t.end_at
FROM task_definition t
WHERE t.workspace_id = ? AND t.`key` = ? AND t.is_active = TRUE AND t.deleted_at IS NULL;

-- name: GetClaimBundleByIDForUpdate :many
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.payload, t.target, t.integration_kind, t.integration_provider, t.integration_payload, t.image_url,
       p.id AS progress_id, p.progress, p.status, p.period_start_at, p.period_end_at,
    p.ready_at, p.claimed_at, p.operation_id, COALESCE(p.rewards_snapshot, JSON_ARRAY()) AS rewards_snapshot,
       r.id AS reward_id, r.reward_key, r.reward_type,
       r.quantity AS reward_quantity, r.scale AS reward_scale, r.duration_unit, r.position AS reward_position
FROM task_definition t
LEFT JOIN task_progress p
  ON p.workspace_id = t.workspace_id AND p.task_id = t.id
 AND p.app_id = ? AND p.platform_id = ? AND p.platform_user_id = ?
 AND p.period_start_at <= ? AND p.period_end_at > ?
LEFT JOIN task_reward r ON r.workspace_id = t.workspace_id AND r.task_id = t.id
WHERE t.workspace_id = ? AND t.id = ?
ORDER BY r.position, r.id
FOR UPDATE;

-- name: GetClaimBundleByKeyForUpdate :many
SELECT t.id, t.workspace_id, t.`key`, t.group_key, t.sequence_key, t.sequence_position,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.payload, t.target, t.integration_kind, t.integration_provider, t.integration_payload, t.image_url,
       p.id AS progress_id, p.progress, p.status, p.period_start_at, p.period_end_at,
    p.ready_at, p.claimed_at, p.operation_id, COALESCE(p.rewards_snapshot, JSON_ARRAY()) AS rewards_snapshot,
       r.id AS reward_id, r.reward_key, r.reward_type,
       r.quantity AS reward_quantity, r.scale AS reward_scale, r.duration_unit, r.position AS reward_position
FROM task_definition t
LEFT JOIN task_progress p
  ON p.workspace_id = t.workspace_id AND p.task_id = t.id
 AND p.app_id = ? AND p.platform_id = ? AND p.platform_user_id = ?
 AND p.period_start_at <= ? AND p.period_end_at > ?
LEFT JOIN task_reward r ON r.workspace_id = t.workspace_id AND r.task_id = t.id
WHERE t.workspace_id = ? AND t.`key` = ?
ORDER BY r.position, r.id
FOR UPDATE;

-- name: ListActiveTaskBundles :many
SELECT t.id, t.`key`, t.group_key,
       t.task_kind, t.action_key, t.action_kind, t.claim_mode, t.target_count,
       t.payload, t.target, t.image_url, t.start_at, t.end_at,
       l.locale, l.title, l.description,
       r.id AS reward_id, r.reward_key, r.reward_type,
       r.quantity AS reward_quantity, r.scale AS reward_scale, r.duration_unit
FROM task_definition t FORCE INDEX (task_definition_visible_user_list_idx)
LEFT JOIN task_localization l ON l.workspace_id = t.workspace_id AND l.task_id = t.id AND l.locale = ?
LEFT JOIN task_reward r ON r.workspace_id = t.workspace_id AND r.task_id = t.id
WHERE t.workspace_id = ? AND t.is_visible = TRUE AND t.is_active = TRUE
  AND t.deleted_at IS NULL
ORDER BY t.position, t.id, r.position, r.id;

-- name: AdminGetTaskStats :one
SELECT
    definitions.tasks_total,
    definitions.active_tasks,
    definitions.visible_tasks,
    progress.progress_total,
    progress.open_progress,
    progress.ready_progress,
    progress.claimed_progress,
    events.progress_created,
    events.progress_amount,
    events.ready_count,
    events.claimed_count,
    events.manual_claimed_count,
    events.auto_claimed_count,
    events.unique_participants,
    events.unique_claimers
FROM (
    SELECT
        COUNT(*) AS tasks_total,
        CAST(COALESCE(SUM(
            is_active = TRUE
            AND deleted_at IS NULL
            AND (start_at IS NULL OR start_at <= NOW())
            AND (end_at IS NULL OR end_at > NOW())
        ), 0) AS UNSIGNED) AS active_tasks,
        CAST(COALESCE(SUM(
            is_visible = TRUE
            AND is_active = TRUE
            AND deleted_at IS NULL
            AND (start_at IS NULL OR start_at <= NOW())
            AND (end_at IS NULL OR end_at > NOW())
        ), 0) AS UNSIGNED) AS visible_tasks
    FROM task_definition stats_definitions
    WHERE stats_definitions.workspace_id = ?
) definitions
CROSS JOIN (
    SELECT
        COUNT(*) AS progress_total,
        CAST(COALESCE(SUM(status = 'open'), 0) AS UNSIGNED) AS open_progress,
        CAST(COALESCE(SUM(status = 'ready'), 0) AS UNSIGNED) AS ready_progress,
        CAST(COALESCE(SUM(status = 'claimed'), 0) AS UNSIGNED) AS claimed_progress
    FROM task_progress stats_progress
    WHERE stats_progress.workspace_id = ?
) progress
CROSS JOIN (
    SELECT
        CAST(COALESCE(SUM(event_type = 'progress_created'), 0) AS UNSIGNED) AS progress_created,
        CAST(COALESCE(SUM(IF(event_type = 'progress_added', amount, 0)), 0) AS UNSIGNED) AS progress_amount,
        CAST(COALESCE(SUM(event_type = 'ready'), 0) AS UNSIGNED) AS ready_count,
        CAST(COALESCE(SUM(event_type = 'claimed'), 0) AS UNSIGNED) AS claimed_count,
        CAST(COALESCE(SUM(event_type = 'claimed' AND claim_mode = 'manual'), 0) AS UNSIGNED) AS manual_claimed_count,
        CAST(COALESCE(SUM(event_type = 'claimed' AND claim_mode = 'auto'), 0) AS UNSIGNED) AS auto_claimed_count,
        COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)) AS unique_participants,
        COUNT(DISTINCT IF(
            event_type = 'claimed',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_claimers
    FROM task_stats_event stats_events
    WHERE stats_events.workspace_id = ?
) events;

-- name: AdminGetSingleTaskStats :one
SELECT
    definition.id AS task_id,
    progress.progress_total,
    progress.open_progress,
    progress.ready_progress,
    progress.claimed_progress,
    events.progress_created,
    events.progress_amount,
    events.ready_count,
    events.claimed_count,
    events.manual_claimed_count,
    events.auto_claimed_count,
    events.unique_participants,
    events.unique_claimers
FROM task_definition definition
CROSS JOIN (
    SELECT
        COUNT(*) AS progress_total,
        CAST(COALESCE(SUM(status = 'open'), 0) AS UNSIGNED) AS open_progress,
        CAST(COALESCE(SUM(status = 'ready'), 0) AS UNSIGNED) AS ready_progress,
        CAST(COALESCE(SUM(status = 'claimed'), 0) AS UNSIGNED) AS claimed_progress
    FROM task_progress single_progress
    WHERE single_progress.workspace_id = ? AND single_progress.task_id = ?
) progress
CROSS JOIN (
    SELECT
        CAST(COALESCE(SUM(event_type = 'progress_created'), 0) AS UNSIGNED) AS progress_created,
        CAST(COALESCE(SUM(IF(event_type = 'progress_added', amount, 0)), 0) AS UNSIGNED) AS progress_amount,
        CAST(COALESCE(SUM(event_type = 'ready'), 0) AS UNSIGNED) AS ready_count,
        CAST(COALESCE(SUM(event_type = 'claimed'), 0) AS UNSIGNED) AS claimed_count,
        CAST(COALESCE(SUM(event_type = 'claimed' AND claim_mode = 'manual'), 0) AS UNSIGNED) AS manual_claimed_count,
        CAST(COALESCE(SUM(event_type = 'claimed' AND claim_mode = 'auto'), 0) AS UNSIGNED) AS auto_claimed_count,
        COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)) AS unique_participants,
        COUNT(DISTINCT IF(
            event_type = 'claimed',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_claimers
    FROM task_stats_event single_events
    WHERE single_events.workspace_id = ? AND single_events.task_id = ?
) events
WHERE definition.workspace_id = ? AND definition.id = ?
LIMIT 1;

-- name: AdminListTaskDailyStats :many
SELECT
    workspace_id,
    task_id,
    stats_date,
    progress_created,
    progress_amount,
    ready_count,
    claimed_count,
    manual_claimed_count,
    auto_claimed_count,
    unique_participants,
    unique_claimers,
    updated_at
FROM task_stats_daily stored_stats
WHERE stored_stats.workspace_id = ?
  AND stored_stats.task_id = ?
  AND stored_stats.stats_date >= ?
  AND stored_stats.stats_date <= ?
  AND stored_stats.stats_date < CURRENT_DATE
UNION ALL
SELECT
    ? AS workspace_id,
    ? AS task_id,
    CURRENT_DATE AS stats_date,
    CAST(COALESCE(SUM(event_type = 'progress_created'), 0) AS UNSIGNED) AS progress_created,
    CAST(COALESCE(SUM(IF(event_type = 'progress_added', amount, 0)), 0) AS UNSIGNED) AS progress_amount,
    CAST(COALESCE(SUM(event_type = 'ready'), 0) AS UNSIGNED) AS ready_count,
    CAST(COALESCE(SUM(event_type = 'claimed'), 0) AS UNSIGNED) AS claimed_count,
    CAST(COALESCE(SUM(event_type = 'claimed' AND claim_mode = 'manual'), 0) AS UNSIGNED) AS manual_claimed_count,
    CAST(COALESCE(SUM(event_type = 'claimed' AND claim_mode = 'auto'), 0) AS UNSIGNED) AS auto_claimed_count,
    COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)) AS unique_participants,
    COUNT(DISTINCT IF(
        event_type = 'claimed',
        CONCAT_WS(':', app_id, platform_id, platform_user_id),
        NULL
    )) AS unique_claimers,
    NOW() AS updated_at
FROM task_stats_event current_events
WHERE current_events.workspace_id = ?
  AND current_events.task_id = ?
  AND current_events.occurred_at >= CURRENT_DATE
  AND current_events.occurred_at < CURRENT_DATE + INTERVAL 1 DAY
  AND CURRENT_DATE >= ?
  AND CURRENT_DATE <= ?
ORDER BY stats_date;

-- name: AdminListTaskDailyOverview :many
SELECT
    workspace_id,
    stats_date,
    tasks_total,
    active_tasks,
    visible_tasks,
    progress_created,
    progress_amount,
    ready_count,
    claimed_count,
    manual_claimed_count,
    auto_claimed_count,
    unique_participants,
    unique_claimers,
    updated_at
FROM task_stats_daily_overview stored_overview
WHERE stored_overview.workspace_id = ?
  AND stored_overview.stats_date >= ?
  AND stored_overview.stats_date <= ?
  AND stored_overview.stats_date < CURRENT_DATE
UNION ALL
SELECT
    ? AS workspace_id,
    CURRENT_DATE AS stats_date,
    definitions.tasks_total,
    definitions.active_tasks,
    definitions.visible_tasks,
    events.progress_created,
    events.progress_amount,
    events.ready_count,
    events.claimed_count,
    events.manual_claimed_count,
    events.auto_claimed_count,
    events.unique_participants,
    events.unique_claimers,
    NOW() AS updated_at
FROM (
    SELECT
        COUNT(*) AS tasks_total,
        CAST(COALESCE(SUM(
            is_active = TRUE
            AND deleted_at IS NULL
            AND (start_at IS NULL OR start_at <= NOW())
            AND (end_at IS NULL OR end_at > NOW())
        ), 0) AS UNSIGNED) AS active_tasks,
        CAST(COALESCE(SUM(
            is_visible = TRUE
            AND is_active = TRUE
            AND deleted_at IS NULL
            AND (start_at IS NULL OR start_at <= NOW())
            AND (end_at IS NULL OR end_at > NOW())
        ), 0) AS UNSIGNED) AS visible_tasks
    FROM task_definition current_definitions
    WHERE current_definitions.workspace_id = ?
) definitions
CROSS JOIN (
    SELECT
        CAST(COALESCE(SUM(event_type = 'progress_created'), 0) AS UNSIGNED) AS progress_created,
        CAST(COALESCE(SUM(IF(event_type = 'progress_added', amount, 0)), 0) AS UNSIGNED) AS progress_amount,
        CAST(COALESCE(SUM(event_type = 'ready'), 0) AS UNSIGNED) AS ready_count,
        CAST(COALESCE(SUM(event_type = 'claimed'), 0) AS UNSIGNED) AS claimed_count,
        CAST(COALESCE(SUM(event_type = 'claimed' AND claim_mode = 'manual'), 0) AS UNSIGNED) AS manual_claimed_count,
        CAST(COALESCE(SUM(event_type = 'claimed' AND claim_mode = 'auto'), 0) AS UNSIGNED) AS auto_claimed_count,
        COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)) AS unique_participants,
        COUNT(DISTINCT IF(
            event_type = 'claimed',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_claimers
    FROM task_stats_event current_overview_events
    WHERE current_overview_events.workspace_id = ?
      AND current_overview_events.occurred_at >= CURRENT_DATE
      AND current_overview_events.occurred_at < CURRENT_DATE + INTERVAL 1 DAY
) events
WHERE CURRENT_DATE >= ?
  AND CURRENT_DATE <= ?
ORDER BY stats_date;

-- name: RefreshTaskDailyStats :exec
INSERT INTO task_stats_daily (
    workspace_id,
    task_id,
    stats_date,
    progress_created,
    progress_amount,
    ready_count,
    claimed_count,
    manual_claimed_count,
    auto_claimed_count,
    unique_participants,
    unique_claimers
)
SELECT
    workspace_id,
    task_id,
    DATE(occurred_at),
    SUM(event_type = 'progress_created'),
    SUM(IF(event_type = 'progress_added', amount, 0)),
    SUM(event_type = 'ready'),
    SUM(event_type = 'claimed'),
    SUM(event_type = 'claimed' AND claim_mode = 'manual'),
    SUM(event_type = 'claimed' AND claim_mode = 'auto'),
    COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)),
    COUNT(DISTINCT IF(
        event_type = 'claimed',
        CONCAT_WS(':', app_id, platform_id, platform_user_id),
        NULL
    ))
FROM task_stats_event
WHERE occurred_at >= ? AND occurred_at < ?
GROUP BY workspace_id, task_id, DATE(occurred_at)
ON DUPLICATE KEY UPDATE
    progress_created = VALUES(progress_created),
    progress_amount = VALUES(progress_amount),
    ready_count = VALUES(ready_count),
    claimed_count = VALUES(claimed_count),
    manual_claimed_count = VALUES(manual_claimed_count),
    auto_claimed_count = VALUES(auto_claimed_count),
    unique_participants = VALUES(unique_participants),
    unique_claimers = VALUES(unique_claimers),
    updated_at = NOW();

-- name: RefreshTaskDailyOverview :exec
INSERT INTO task_stats_daily_overview (
    workspace_id,
    stats_date,
    tasks_total,
    active_tasks,
    visible_tasks,
    progress_created,
    progress_amount,
    ready_count,
    claimed_count,
    manual_claimed_count,
    auto_claimed_count,
    unique_participants,
    unique_claimers
)
SELECT
    event_rows.workspace_id,
    event_rows.stats_date,
    definitions.tasks_total,
    definitions.active_tasks,
    definitions.visible_tasks,
    event_rows.progress_created,
    event_rows.progress_amount,
    event_rows.ready_count,
    event_rows.claimed_count,
    event_rows.manual_claimed_count,
    event_rows.auto_claimed_count,
    event_rows.unique_participants,
    event_rows.unique_claimers
FROM (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        SUM(event_type = 'progress_created') AS progress_created,
        SUM(IF(event_type = 'progress_added', amount, 0)) AS progress_amount,
        SUM(event_type = 'ready') AS ready_count,
        SUM(event_type = 'claimed') AS claimed_count,
        SUM(event_type = 'claimed' AND claim_mode = 'manual') AS manual_claimed_count,
        SUM(event_type = 'claimed' AND claim_mode = 'auto') AS auto_claimed_count,
        COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)) AS unique_participants,
        COUNT(DISTINCT IF(
            event_type = 'claimed',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_claimers
    FROM task_stats_event refresh_events
    WHERE refresh_events.occurred_at >= ? AND refresh_events.occurred_at < ?
    GROUP BY refresh_events.workspace_id, DATE(refresh_events.occurred_at)
) event_rows
JOIN (
    SELECT
        workspace_id,
        COUNT(*) AS tasks_total,
        SUM(
            is_active = TRUE
            AND deleted_at IS NULL
            AND (start_at IS NULL OR start_at <= NOW())
            AND (end_at IS NULL OR end_at > NOW())
        ) AS active_tasks,
        SUM(
            is_visible = TRUE
            AND is_active = TRUE
            AND deleted_at IS NULL
            AND (start_at IS NULL OR start_at <= NOW())
            AND (end_at IS NULL OR end_at > NOW())
        ) AS visible_tasks
    FROM task_definition
    GROUP BY workspace_id
) definitions ON definitions.workspace_id = event_rows.workspace_id
ON DUPLICATE KEY UPDATE
    progress_created = VALUES(progress_created),
    progress_amount = VALUES(progress_amount),
    ready_count = VALUES(ready_count),
    claimed_count = VALUES(claimed_count),
    manual_claimed_count = VALUES(manual_claimed_count),
    auto_claimed_count = VALUES(auto_claimed_count),
    unique_participants = VALUES(unique_participants),
    unique_claimers = VALUES(unique_claimers),
    updated_at = NOW();

-- name: AdminUpsertPartnerConfig :exec
INSERT INTO task_partner_config (
    workspace_id, provider, group_key, platform, is_enabled, secret, target, settings
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    is_enabled = VALUES(is_enabled),
    secret = VALUES(secret),
    target = VALUES(target),
    settings = VALUES(settings);

-- name: AdminGetPartnerConfig :one
SELECT workspace_id, provider, group_key, platform, is_enabled, secret, target, settings, created_at, updated_at
FROM task_partner_config
WHERE workspace_id = ? AND provider = ? AND group_key = ? AND platform = ?
LIMIT 1;

-- name: AdminListPartnerConfigs :many
SELECT workspace_id, provider, group_key, platform, is_enabled, secret, target, settings, created_at, updated_at
FROM task_partner_config
WHERE workspace_id = ?
ORDER BY provider, group_key, platform;

-- name: AdminUpsertPartnerRewardRule :exec
INSERT INTO task_partner_reward_rule (
    workspace_id, provider, group_key, external_type, reward_key,
    reward_type, quantity, scale, duration_unit, position, is_enabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    reward_type = VALUES(reward_type),
    quantity = VALUES(quantity),
    scale = VALUES(scale),
    duration_unit = VALUES(duration_unit),
    position = VALUES(position),
    is_enabled = VALUES(is_enabled);

-- name: AdminDeletePartnerRewardRule :execrows
DELETE FROM task_partner_reward_rule
WHERE workspace_id = ? AND provider = ? AND group_key = ? AND external_type = ? AND reward_key = ?;

-- name: ListPartnerRewardRules :many
SELECT workspace_id, provider, group_key, external_type, reward_key,
       reward_type, quantity, scale, duration_unit, position, is_enabled, created_at, updated_at
FROM task_partner_reward_rule
WHERE workspace_id = ?
  AND provider = ?
  AND group_key = ?
  AND external_type IN (?, '*')
  AND is_enabled = TRUE
ORDER BY CASE WHEN external_type = ? THEN 0 ELSE 1 END, position, reward_key;

-- name: CreatePartnerIssue :execlastid
INSERT INTO task_partner_issue (
    workspace_id, provider, group_key, platform, external_id, external_type, issue_key,
    app_id, platform_id, platform_user_id, public_payload, private_payload, status, issued_at, expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'issued', ?, ?)
ON DUPLICATE KEY UPDATE
    id = LAST_INSERT_ID(id),
    public_payload = VALUES(public_payload),
    private_payload = VALUES(private_payload),
    expires_at = VALUES(expires_at),
    updated_at = CURRENT_TIMESTAMP;

-- name: GetPartnerIssueByID :one
SELECT id, workspace_id, provider, group_key, platform, external_id, external_type, issue_key,
       app_id, platform_id, platform_user_id, public_payload, private_payload,
       status, issued_at, completed_at, claimed_at, expires_at, created_at, updated_at
FROM task_partner_issue
WHERE workspace_id = ? AND id = ?
LIMIT 1;

-- name: GetPartnerIssueByIDForUpdate :one
SELECT id, workspace_id, provider, group_key, platform, external_id, external_type, issue_key,
       app_id, platform_id, platform_user_id, public_payload, private_payload,
       status, issued_at, completed_at, claimed_at, expires_at, created_at, updated_at
FROM task_partner_issue
WHERE workspace_id = ? AND id = ?
LIMIT 1
FOR UPDATE;

-- name: ListPartnerIssuesForUser :many
SELECT id, workspace_id, provider, group_key, platform, external_id, external_type, issue_key,
       app_id, platform_id, platform_user_id, public_payload, private_payload,
       status, issued_at, completed_at, claimed_at, expires_at, created_at, updated_at
FROM task_partner_issue
WHERE workspace_id = ?
  AND provider = ?
  AND group_key = ?
  AND platform = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND status IN ('issued', 'completed')
  AND (expires_at IS NULL OR expires_at > ?)
ORDER BY issued_at DESC, id DESC;

-- name: CompletePartnerIssue :execrows
UPDATE task_partner_issue
SET status = 'completed', completed_at = ?, updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = ? AND id = ? AND status = 'issued';

-- name: ClaimPartnerIssue :execrows
UPDATE task_partner_issue
SET status = 'claimed', claimed_at = ?, updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = ? AND id = ? AND status = 'completed';

-- name: InsertPartnerRewardGrant :execrows
INSERT IGNORE INTO task_partner_reward_grant (
    workspace_id, issue_id, provider, group_key, external_type,
    app_id, platform_id, platform_user_id, operation_id, reward_snapshot, claimed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertPartnerStatsEvent :execrows
INSERT IGNORE INTO task_partner_stats_event (
    workspace_id, provider, group_key, external_type, issue_id, external_id,
    app_id, platform_id, platform_user_id, event_type, event_key, status, payload, occurred_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertPartnerStatsUniqueUser :execrows
INSERT IGNORE INTO task_partner_stats_unique_user (
    workspace_id, stats_date, provider, group_key, external_type, event_type,
    app_id, platform_id, platform_user_id
) VALUES (?, DATE(?), ?, ?, ?, ?, ?, ?, ?);

-- name: IncrementPartnerStatsDaily :exec
INSERT INTO task_partner_stats_daily (
    workspace_id, stats_date, provider, group_key, external_type,
    issued_count, completed_count, claimed_count, failed_count, fake_count, expired_count,
    unique_issued_users, unique_completed_users, unique_claimers
) VALUES (?, DATE(?), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    issued_count = issued_count + VALUES(issued_count),
    completed_count = completed_count + VALUES(completed_count),
    claimed_count = claimed_count + VALUES(claimed_count),
    failed_count = failed_count + VALUES(failed_count),
    fake_count = fake_count + VALUES(fake_count),
    expired_count = expired_count + VALUES(expired_count),
    unique_issued_users = unique_issued_users + VALUES(unique_issued_users),
    unique_completed_users = unique_completed_users + VALUES(unique_completed_users),
    unique_claimers = unique_claimers + VALUES(unique_claimers);

-- name: AdminListPartnerDailyStats :many
SELECT workspace_id, stats_date, provider, group_key, external_type,
       issued_count, completed_count, claimed_count, failed_count, fake_count, expired_count,
       unique_issued_users, unique_completed_users, unique_claimers, updated_at
FROM task_partner_stats_daily
WHERE workspace_id = ?
  AND stats_date >= ?
  AND stats_date < ?
  AND (? = '' OR provider = ?)
  AND (? = '' OR group_key = ?)
ORDER BY stats_date, provider, group_key, external_type;

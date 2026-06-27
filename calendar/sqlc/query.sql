-- name: AdminCreateCalendar :exec
INSERT INTO calendar_definition (
    id, workspace_id, type, mode, interval_type, interval_unit,
    interval_count, reset_after_intervals, end_behavior, timezone,
    hide_future_rewards, is_active, start_at, end_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AdminUpdateCalendar :execrows
UPDATE calendar_definition
SET type = ?,
    mode = ?,
    interval_type = ?,
    interval_unit = ?,
    interval_count = ?,
    reset_after_intervals = ?,
    end_behavior = ?,
    timezone = ?,
    hide_future_rewards = ?,
    is_active = ?,
    start_at = ?,
    end_at = ?
WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL;

-- name: AdminGetCalendar :one
SELECT *
FROM calendar_definition
WHERE workspace_id = ? AND id = ?
LIMIT 1;

-- name: AdminListCalendars :many
SELECT *
FROM calendar_definition
WHERE workspace_id = ?
ORDER BY created_at DESC, id
LIMIT ? OFFSET ?;

-- name: AdminSetCalendarActive :execrows
UPDATE calendar_definition
SET is_active = ?
WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL;

-- name: AdminSoftDeleteCalendar :execrows
UPDATE calendar_definition
SET deleted_at = NOW(), is_active = FALSE
WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL;

-- name: AdminUpsertLocalization :exec
INSERT INTO calendar_localization (
    workspace_id, calendar_id, locale, title, description
) VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    description = VALUES(description);

-- name: AdminGetLocalization :one
SELECT *
FROM calendar_localization
WHERE workspace_id = ? AND calendar_id = ? AND locale = ?
LIMIT 1;

-- name: AdminListLocalizations :many
SELECT *
FROM calendar_localization
WHERE workspace_id = ? AND calendar_id = ?
ORDER BY locale;

-- name: AdminDeleteLocalization :execrows
DELETE FROM calendar_localization
WHERE workspace_id = ? AND calendar_id = ? AND locale = ?;

-- name: AdminCreateStep :execlastid
INSERT INTO calendar_step (workspace_id, calendar_id, position)
VALUES (?, ?, ?);

-- name: AdminUpdateStep :execrows
UPDATE calendar_step
SET position = ?
WHERE workspace_id = ? AND calendar_id = ? AND id = ?;

-- name: AdminDeleteStep :execrows
DELETE FROM calendar_step
WHERE workspace_id = ? AND calendar_id = ? AND id = ?;

-- name: AdminUpsertReward :execlastid
INSERT INTO calendar_reward (
    workspace_id, calendar_id, step_id, item_key,
    reward_type, item_count, scale, duration_unit, position
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    id = LAST_INSERT_ID(id),
    reward_type = VALUES(reward_type),
    item_count = VALUES(item_count),
    scale = VALUES(scale),
    duration_unit = VALUES(duration_unit),
    position = VALUES(position);

-- name: AdminGetReward :one
SELECT *
FROM calendar_reward
WHERE workspace_id = ? AND calendar_id = ? AND id = ?
LIMIT 1;

-- name: AdminUpdateReward :execrows
UPDATE calendar_reward
SET step_id = ?,
    item_key = ?,
    reward_type = ?,
    item_count = ?,
    scale = ?,
    duration_unit = ?,
    position = ?
WHERE workspace_id = ? AND calendar_id = ? AND id = ?;

-- name: AdminDeleteReward :execrows
DELETE FROM calendar_reward
WHERE workspace_id = ? AND calendar_id = ? AND id = ?;

-- name: GetCalendarBundle :many
SELECT
    c.id,
    c.workspace_id,
    c.type,
    c.mode,
    c.interval_type,
    c.interval_unit,
    c.interval_count,
    c.reset_after_intervals,
    c.end_behavior,
    c.timezone,
    c.hide_future_rewards,
    c.is_active,
    c.start_at,
    c.end_at,
    c.deleted_at,
    c.created_at,
    c.updated_at,
    l.locale AS localization_locale,
    l.title AS localization_title,
    l.description AS localization_description,
    s.id AS step_id,
    s.position AS step_position,
    r.id AS reward_id,
    r.item_key AS reward_item_key,
    r.reward_type AS reward_type,
    r.item_count AS reward_item_count,
    r.scale AS reward_scale,
    r.duration_unit AS reward_duration_unit,
    r.position AS reward_position
FROM calendar_definition c
LEFT JOIN calendar_localization l
  ON l.workspace_id = c.workspace_id
 AND l.calendar_id = c.id
 AND l.locale = ?
LEFT JOIN calendar_step s
  ON s.workspace_id = c.workspace_id
 AND s.calendar_id = c.id
LEFT JOIN calendar_reward r
  ON r.workspace_id = s.workspace_id
 AND r.calendar_id = s.calendar_id
 AND r.step_id = s.id
WHERE c.workspace_id = ?
  AND (c.id = ? OR c.type = ?)
ORDER BY s.position, r.position, r.id;

-- name: ListActiveCalendars :many
SELECT
    c.id,
    c.workspace_id,
    c.type,
    c.mode,
    c.is_active,
    c.start_at,
    c.end_at,
    l.locale,
    l.title,
    l.description
FROM calendar_definition c
LEFT JOIN calendar_localization l
  ON l.workspace_id = c.workspace_id
 AND l.calendar_id = c.id
 AND l.locale = ?
WHERE c.workspace_id = ?
  AND c.is_active = TRUE
  AND c.deleted_at IS NULL
  AND (c.start_at IS NULL OR c.start_at <= ?)
  AND (c.end_at IS NULL OR c.end_at > ?)
ORDER BY c.created_at DESC, c.id;

-- name: GetRecordBundleForUpdate :many
SELECT
    c.id,
    c.workspace_id,
    c.type,
    c.mode,
    c.interval_type,
    c.interval_unit,
    c.interval_count,
    c.reset_after_intervals,
    c.end_behavior,
    c.timezone,
    c.hide_future_rewards,
    c.is_active,
    c.start_at,
    c.end_at,
    c.deleted_at,
    c.created_at,
    c.updated_at,
    p.current_position,
    p.claim_count,
    p.last_claim_position,
    p.last_claim_at,
    p.next_claim_at,
    p.is_completed,
    p.reset_count,
    p.last_was_reset,
    o.id AS operation_row_id,
    o.operation_id AS existing_operation_id,
    o.granted AS operation_granted,
    o.status AS operation_status,
    o.position AS operation_position,
    COALESCE(o.rewards_snapshot, JSON_ARRAY()) AS operation_rewards_snapshot,
    o.current_position AS operation_current_position,
    o.claim_count AS operation_claim_count,
    o.last_claim_position AS operation_last_claim_position,
    o.last_claim_at AS operation_last_claim_at,
    o.next_claim_at AS operation_next_claim_at,
    o.is_completed AS operation_is_completed,
    o.reset_count AS operation_reset_count,
    o.was_reset AS operation_was_reset,
    o.occurred_at AS operation_occurred_at,
    s.id AS step_id,
    s.position AS step_position,
    r.id AS reward_id,
    r.item_key AS reward_item_key,
    r.reward_type AS reward_type,
    r.item_count AS reward_item_count,
    r.scale AS reward_scale,
    r.duration_unit AS reward_duration_unit,
    r.position AS reward_position
FROM calendar_definition c
LEFT JOIN calendar_progress p
  ON p.workspace_id = c.workspace_id
 AND p.calendar_id = c.id
 AND p.app_id = ?
 AND p.platform_id = ?
 AND p.platform_user_id = ?
LEFT JOIN calendar_operation o
  ON o.workspace_id = c.workspace_id
 AND o.calendar_id = c.id
 AND o.app_id = ?
 AND o.platform_id = ?
 AND o.platform_user_id = ?
 AND o.operation_id = ?
LEFT JOIN calendar_step s
  ON s.workspace_id = c.workspace_id
 AND s.calendar_id = c.id
LEFT JOIN calendar_reward r
  ON r.workspace_id = s.workspace_id
 AND r.calendar_id = s.calendar_id
 AND r.step_id = s.id
WHERE c.workspace_id = ?
  AND (c.id = ? OR c.type = ?)
ORDER BY s.position, r.position, r.id
FOR UPDATE;

-- name: CreateOperation :execlastid
INSERT INTO calendar_operation (
    workspace_id, calendar_id, app_id, platform_id, platform_user_id,
    operation_id, granted, status, position, rewards_snapshot,
    current_position, claim_count, last_claim_position, last_claim_at,
    next_claim_at, is_completed, reset_count, was_reset, occurred_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProgress :one
SELECT *
FROM calendar_progress
WHERE workspace_id = ?
  AND calendar_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
LIMIT 1;

-- name: AdminListOperations :many
SELECT *
FROM calendar_operation
WHERE workspace_id = ? AND calendar_id = ?
ORDER BY occurred_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetStats :one
SELECT
    COUNT(*) AS operation_count,
    COALESCE(SUM(granted), 0) AS grant_count,
    COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)) AS unique_users
FROM calendar_operation
WHERE workspace_id = ? AND calendar_id = ?;

-- name: AdminListDailyStats :many
SELECT *
FROM calendar_stats_daily
WHERE workspace_id = ?
  AND calendar_id = ?
  AND stats_date >= ?
  AND stats_date <= ?
ORDER BY stats_date;

-- name: RefreshDailyStats :exec
INSERT INTO calendar_stats_daily (
    workspace_id, calendar_id, stats_date,
    operation_count, grant_count, unique_users
)
SELECT
    workspace_id,
    calendar_id,
    DATE(occurred_at),
    COUNT(*),
    COALESCE(SUM(granted), 0),
    COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id))
FROM calendar_operation
WHERE occurred_at >= ? AND occurred_at < ?
GROUP BY workspace_id, calendar_id, DATE(occurred_at)
ON DUPLICATE KEY UPDATE
    operation_count = VALUES(operation_count),
    grant_count = VALUES(grant_count),
    unique_users = VALUES(unique_users),
    updated_at = NOW();

-- name: AdminCreatePromo :execlastid
INSERT INTO promo_offer (
    workspace_id, code, code_normalized, payload, max_activations,
    is_active, start_at, end_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: AdminUpdatePromo :execrows
UPDATE promo_offer
SET code = ?,
    code_normalized = ?,
    payload = ?,
    max_activations = ?,
    is_active = ?,
    start_at = ?,
    end_at = ?
WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL;

-- name: AdminGetPromo :one
SELECT *
FROM promo_offer
WHERE workspace_id = ? AND id = ?
LIMIT 1;

-- name: AdminListPromos :many
SELECT *
FROM promo_offer
WHERE workspace_id = ?
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminSoftDeletePromo :execrows
UPDATE promo_offer
SET deleted_at = NOW(), is_active = FALSE
WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL;

-- name: GetApplyBundleForUpdate :many
SELECT
    o.id,
    o.workspace_id,
    o.code,
    o.payload,
    o.max_activations,
    o.activation_count,
    o.is_active,
    o.start_at,
    o.end_at,
    o.deleted_at,
    o.created_at,
    o.updated_at,
    l.locale AS localization_locale,
    l.title AS localization_title,
    l.description AS localization_description,
    a.id AS redemption_id,
    a.app_id AS redemption_app_id,
    a.platform_id AS redemption_platform_id,
    a.platform_user_id AS redemption_platform_user_id,
    a.redeemed_at AS redemption_redeemed_at,
    r.id AS reward_id,
    r.reward_key,
    r.reward_type,
    r.quantity AS reward_quantity,
    r.duration_unit
FROM promo_offer o
LEFT JOIN promo_localization l
  ON l.workspace_id = o.workspace_id
 AND l.promo_id = o.id
 AND l.locale = ?
LEFT JOIN promo_redemption a
  ON a.workspace_id = o.workspace_id
 AND a.promo_id = o.id
 AND a.app_id = ?
 AND a.platform_id = ?
 AND a.platform_user_id = ?
LEFT JOIN promo_reward r
  ON r.workspace_id = o.workspace_id
 AND r.promo_id = o.id
WHERE o.workspace_id = ?
  AND o.code_normalized = ?
ORDER BY r.id
FOR UPDATE;

-- name: AdminUpsertLocalization :exec
INSERT INTO promo_localization (
    workspace_id, promo_id, locale, title, description
) VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    description = VALUES(description);

-- name: AdminListLocalizations :many
SELECT *
FROM promo_localization
WHERE workspace_id = ? AND promo_id = ?
ORDER BY locale;

-- name: AdminGetLocalization :one
SELECT *
FROM promo_localization
WHERE workspace_id = ? AND promo_id = ? AND locale = ?
LIMIT 1;

-- name: AdminDeleteLocalization :execrows
DELETE FROM promo_localization
WHERE workspace_id = ? AND promo_id = ? AND locale = ?;

-- name: AdminUpsertReward :exec
INSERT INTO promo_reward (
    workspace_id, promo_id, reward_key, reward_type, quantity, duration_unit
)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    reward_type = VALUES(reward_type),
    quantity = VALUES(quantity),
    duration_unit = VALUES(duration_unit);

-- name: AdminGetReward :one
SELECT *
FROM promo_reward
WHERE workspace_id = ? AND promo_id = ? AND reward_key = ?
LIMIT 1;

-- name: ListRewards :many
SELECT *
FROM promo_reward
WHERE workspace_id = ? AND promo_id = ?
ORDER BY id;

-- name: AdminDeleteReward :execrows
DELETE FROM promo_reward
WHERE workspace_id = ? AND promo_id = ? AND reward_key = ?;

-- name: GetRedemption :one
SELECT *
FROM promo_redemption
WHERE workspace_id = ?
  AND promo_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
LIMIT 1;

-- name: CreateRedemption :execlastid
INSERT INTO promo_redemption (
    workspace_id, promo_id, app_id, platform_id, platform_user_id,
    reward_snapshot
) VALUES (?, ?, ?, ?, ?, ?);

-- name: AdminListRedemptions :many
SELECT *
FROM promo_redemption
WHERE workspace_id = ?
  AND promo_id = ?
ORDER BY redeemed_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetStats :one
SELECT
    activation_count,
    max_activations,
    CASE
        WHEN max_activations = 0 THEN -1
        ELSE CAST(max_activations - activation_count AS SIGNED)
    END AS remaining_activations
FROM promo_offer
WHERE workspace_id = ? AND id = ?
LIMIT 1;

-- name: AdminListDailyStats :many
SELECT *
FROM promo_stats_daily
WHERE workspace_id = ?
  AND promo_id = ?
  AND stats_date >= ?
  AND stats_date <= ?
ORDER BY stats_date;

-- name: RefreshDailyStats :exec
INSERT INTO promo_stats_daily (
    workspace_id, promo_id, stats_date, redemption_count, unique_users
)
SELECT
    e.workspace_id,
    e.promo_id,
    DATE(e.occurred_at),
    COUNT(*),
    COUNT(DISTINCT CONCAT_WS(':', r.app_id, r.platform_id, r.platform_user_id))
FROM promo_redemption_event e
JOIN promo_redemption r
  ON r.workspace_id = e.workspace_id
 AND r.promo_id = e.promo_id
 AND r.id = e.redemption_id
WHERE e.occurred_at >= ? AND e.occurred_at < ?
GROUP BY e.workspace_id, e.promo_id, DATE(e.occurred_at)
ON DUPLICATE KEY UPDATE
    redemption_count = VALUES(redemption_count),
    unique_users = VALUES(unique_users),
    updated_at = NOW();

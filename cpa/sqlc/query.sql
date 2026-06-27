-- name: AdminUpsertOffer :exec
INSERT INTO cpa_offer (
    workspace_id, id, payload, target, code_mode, code_source, shared_code,
    generated_length, generated_alphabet, is_active, start_at, end_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    payload = VALUES(payload),
    target = VALUES(target),
    code_mode = VALUES(code_mode),
    code_source = VALUES(code_source),
    shared_code = VALUES(shared_code),
    generated_length = VALUES(generated_length),
    generated_alphabet = VALUES(generated_alphabet),
    is_active = VALUES(is_active),
    start_at = VALUES(start_at),
    end_at = VALUES(end_at);

-- name: AdminGetOffer :one
SELECT *
FROM cpa_offer
WHERE workspace_id = ? AND id = ?
LIMIT 1;

-- name: GetActiveOfferForUpdate :one
SELECT *
FROM cpa_offer
WHERE workspace_id = ?
  AND id = ?
  AND is_active = TRUE
  AND (start_at IS NULL OR start_at <= NOW())
  AND (end_at IS NULL OR end_at > NOW())
LIMIT 1
FOR UPDATE;

-- name: AdminListOffers :many
SELECT *
FROM cpa_offer
WHERE workspace_id = ?
ORDER BY created_at DESC, id
LIMIT ? OFFSET ?;

-- name: AdminListOfferBundles :many
SELECT
    o.*,
    l.locale,
    l.title AS localization_title,
    l.description AS localization_description
FROM (
    SELECT *
    FROM cpa_offer
    WHERE cpa_offer.workspace_id = ?
    ORDER BY cpa_offer.created_at DESC, cpa_offer.id
    LIMIT ? OFFSET ?
) o
LEFT JOIN cpa_localization l
    ON l.workspace_id = o.workspace_id
   AND l.cpa_id = o.id
ORDER BY o.created_at DESC, o.id, l.locale;

-- name: AdminListOfferBundleRewards :many
SELECT
    o.workspace_id,
    o.id AS cpa_id,
    r.reward_key,
    r.reward_type,
    r.quantity AS reward_quantity,
    r.scale AS reward_scale,
    r.duration_unit
FROM (
    SELECT *
    FROM cpa_offer
    WHERE cpa_offer.workspace_id = ?
    ORDER BY cpa_offer.created_at DESC, cpa_offer.id
    LIMIT ? OFFSET ?
) o
JOIN cpa_reward r
    ON r.workspace_id = o.workspace_id
   AND r.cpa_id = o.id
ORDER BY o.created_at DESC, o.id, r.id;

-- name: ListActiveOffers :many
SELECT *
FROM cpa_offer
WHERE workspace_id = ?
  AND is_active = TRUE
  AND (start_at IS NULL OR start_at <= NOW())
  AND (end_at IS NULL OR end_at > NOW())
ORDER BY created_at DESC, id;

-- name: ListActiveOfferBundles :many
SELECT
    o.workspace_id,
    o.id,
    o.payload,
    o.target,
    o.code_mode,
    o.code_source,
    o.shared_code,
    o.generated_length,
    o.generated_alphabet,
    o.is_active,
    o.start_at,
    o.end_at,
    o.created_at,
    o.updated_at,
    l.locale AS localized_locale,
    l.title AS localized_title,
    l.description AS localized_description,
    a.id AS assignment_id,
    a.code AS assignment_code,
    a.code_mode AS assignment_code_mode,
    a.status AS assignment_status,
    a.issued_at AS assignment_issued_at,
    a.completed_at AS assignment_completed_at,
    r.reward_key,
    r.reward_type,
    r.quantity AS reward_quantity,
    r.scale AS reward_scale,
    r.duration_unit
FROM cpa_offer o
LEFT JOIN cpa_localization l
    ON l.workspace_id = o.workspace_id
   AND l.cpa_id = o.id
   AND l.locale = ?
LEFT JOIN cpa_assignment a
    ON a.workspace_id = o.workspace_id
   AND a.cpa_id = o.id
   AND a.app_id = ?
   AND a.platform_id = ?
   AND a.platform_user_id = ?
   AND a.deleted_at IS NULL
LEFT JOIN cpa_reward r
    ON r.workspace_id = o.workspace_id
   AND r.cpa_id = o.id
WHERE o.workspace_id = ?
  AND o.is_active = TRUE
  AND (o.start_at IS NULL OR o.start_at <= NOW())
  AND (o.end_at IS NULL OR o.end_at > NOW())
ORDER BY o.created_at DESC, o.id, r.id;

-- name: AdminDeleteOffer :execrows
DELETE FROM cpa_offer
WHERE workspace_id = ? AND id = ?;

-- name: AdminUpsertLocalization :exec
INSERT INTO cpa_localization (
    workspace_id, cpa_id, locale, title, description
) VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    description = VALUES(description);

-- name: GetLocalization :one
SELECT *
FROM cpa_localization
WHERE workspace_id = ? AND cpa_id = ? AND locale = ?
LIMIT 1;

-- name: ListLocalizations :many
SELECT *
FROM cpa_localization
WHERE workspace_id = ? AND cpa_id = ?
ORDER BY locale;

-- name: AdminDeleteLocalization :execrows
DELETE FROM cpa_localization
WHERE workspace_id = ? AND cpa_id = ? AND locale = ?;

-- name: AdminUpsertReward :exec
INSERT INTO cpa_reward (
    workspace_id, cpa_id, reward_key, reward_type, quantity, scale, duration_unit
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    reward_type = VALUES(reward_type),
    quantity = VALUES(quantity),
    scale = VALUES(scale),
    duration_unit = VALUES(duration_unit);

-- name: ListRewards :many
SELECT *
FROM cpa_reward
WHERE workspace_id = ? AND cpa_id = ?
ORDER BY id;

-- name: AdminDeleteReward :execrows
DELETE FROM cpa_reward
WHERE workspace_id = ? AND cpa_id = ? AND reward_key = ?;

-- name: AdminAddCode :execrows
INSERT IGNORE INTO cpa_code (workspace_id, cpa_id, code, source)
VALUES (?, ?, ?, ?);

-- name: CreateGeneratedCode :execlastid
INSERT INTO cpa_code (workspace_id, cpa_id, code, source)
VALUES (?, ?, ?, 'generated');

-- name: GetAvailableCodeForUpdate :one
SELECT *
FROM cpa_code
WHERE workspace_id = ?
  AND cpa_id = ?
  AND source = 'pool'
  AND status = 'available'
ORDER BY id
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: GetCodeByValue :one
SELECT *
FROM cpa_code
WHERE workspace_id = ? AND cpa_id = ? AND code = ?
LIMIT 1;

-- name: MarkCodeIssued :execrows
UPDATE cpa_code
SET status = 'issued'
WHERE id = ? AND status = 'available';

-- name: MarkCodeCompleted :execrows
UPDATE cpa_code
SET status = 'completed'
WHERE id = ? AND status = 'issued';

-- name: GetAssignment :one
SELECT *
FROM cpa_assignment
WHERE workspace_id = ?
  AND cpa_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetAssignmentForUpdate :one
SELECT *
FROM cpa_assignment
WHERE workspace_id = ?
  AND cpa_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND deleted_at IS NULL
LIMIT 1
FOR UPDATE;

-- name: CreateAssignment :execlastid
INSERT INTO cpa_assignment (
    workspace_id, cpa_id, app_id, platform_id, platform_user_id,
    code_id, code, code_mode
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAssignmentByID :one
SELECT *
FROM cpa_assignment
WHERE workspace_id = ? AND id = ? AND deleted_at IS NULL
LIMIT 1;

-- name: CompleteAssignment :execrows
UPDATE cpa_assignment
SET status = 'completed', completed_at = NOW()
WHERE workspace_id = ?
  AND id = ?
  AND status = 'issued'
  AND deleted_at IS NULL;

-- name: ListUserAssignments :many
SELECT *
FROM cpa_assignment
WHERE workspace_id = ?
  AND app_id = ?
  AND platform_id = ?
  AND platform_user_id = ?
  AND deleted_at IS NULL
ORDER BY issued_at DESC, id DESC;

-- name: AdminListAssignments :many
SELECT *
FROM cpa_assignment
WHERE workspace_id = ?
  AND cpa_id = ?
  AND (? = '' OR CAST(status AS CHAR) = ?)
ORDER BY issued_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminListCodes :many
SELECT *
FROM cpa_code
WHERE workspace_id = ?
  AND cpa_id = ?
  AND (? = '' OR CAST(status AS CHAR) = ?)
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: AdminListAssignmentEvents :many
SELECT *
FROM cpa_assignment_event
WHERE workspace_id = ?
  AND cpa_id = ?
  AND (? = '' OR CAST(event_type AS CHAR) = ?)
ORDER BY occurred_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CreateAssignmentEvent :execlastid
INSERT INTO cpa_assignment_event (
    workspace_id, cpa_id, assignment_id, event_type
) VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id);

-- name: AdminDeleteAvailableCodes :execrows
UPDATE cpa_code
SET status = 'deleted', deleted_at = NOW()
WHERE workspace_id = ?
  AND cpa_id = ?
  AND status = 'available';

-- name: AdminDeleteIssuedCodes :execrows
UPDATE cpa_assignment
SET deleted_at = NOW()
WHERE workspace_id = ?
  AND cpa_id = ?
  AND status = 'issued'
  AND deleted_at IS NULL;

-- name: AdminDeleteIssuedCodeRows :execrows
UPDATE cpa_code c
JOIN cpa_assignment a ON a.code_id = c.id
SET c.status = 'deleted', c.deleted_at = NOW()
WHERE a.workspace_id = ?
  AND a.cpa_id = ?
  AND a.status = 'issued'
  AND a.deleted_at IS NOT NULL;

-- name: AdminDeleteCompletedCodes :execrows
UPDATE cpa_assignment
SET deleted_at = NOW()
WHERE workspace_id = ?
  AND cpa_id = ?
  AND status = 'completed'
  AND deleted_at IS NULL;

-- name: AdminDeleteCompletedCodeRows :execrows
UPDATE cpa_code c
JOIN cpa_assignment a ON a.code_id = c.id
SET c.status = 'deleted', c.deleted_at = NOW()
WHERE a.workspace_id = ?
  AND a.cpa_id = ?
  AND a.status = 'completed'
  AND a.deleted_at IS NOT NULL;

-- name: AdminGetOfferStats :one
SELECT
    COUNT(*) AS assignments_total,
    CAST(COALESCE(SUM(status = 'issued'), 0) AS UNSIGNED) AS issued_total,
    CAST(COALESCE(SUM(status = 'completed'), 0) AS UNSIGNED) AS completed_total,
    CAST(COALESCE(SUM(deleted_at IS NOT NULL), 0) AS UNSIGNED) AS deleted_total
FROM cpa_assignment
WHERE workspace_id = ? AND cpa_id = ?;

-- name: AdminGetCodeStats :one
SELECT
    COUNT(*) AS codes_total,
    CAST(COALESCE(SUM(status = 'available'), 0) AS UNSIGNED) AS available_total,
    CAST(COALESCE(SUM(status = 'issued'), 0) AS UNSIGNED) AS issued_total,
    CAST(COALESCE(SUM(status = 'completed'), 0) AS UNSIGNED) AS completed_total,
    CAST(COALESCE(SUM(status = 'deleted'), 0) AS UNSIGNED) AS deleted_total
FROM cpa_code
WHERE workspace_id = ? AND cpa_id = ?;

-- name: AdminListDailyStats :many
SELECT *
FROM cpa_stats_daily
WHERE workspace_id = ?
  AND cpa_id = ?
  AND stats_date >= ?
  AND stats_date <= ?
ORDER BY stats_date;

-- name: RefreshDailyStats :exec
INSERT INTO cpa_stats_daily (
    workspace_id, cpa_id, stats_date,
    issued_count, completed_count, unique_users
)
SELECT
    workspace_id,
    cpa_id,
    DATE(occurred_at),
    SUM(event_type = 'issued'),
    SUM(event_type = 'completed'),
    COUNT(DISTINCT assignment_id)
FROM cpa_assignment_event
WHERE occurred_at >= ?
  AND occurred_at < ?
GROUP BY workspace_id, cpa_id, DATE(occurred_at)
ON DUPLICATE KEY UPDATE
    issued_count = VALUES(issued_count),
    completed_count = VALUES(completed_count),
    unique_users = VALUES(unique_users),
    updated_at = NOW();

-- name: AdminCreateItem :exec
INSERT INTO reference_item (
    workspace_id, `key`, item_type, payload, is_active
) VALUES (?, ?, ?, ?, ?);

-- name: AdminUpdateItem :execrows
UPDATE reference_item
SET payload = ?,
    is_active = ?
WHERE workspace_id = ?
  AND `key` = ?
  AND deleted_at IS NULL;

-- name: AdminDangerousChangeType :execrows
UPDATE reference_item
SET item_type = ?
WHERE workspace_id = ?
  AND `key` = ?
  AND item_type = ?
  AND deleted_at IS NULL;

-- name: AdminSoftDeleteItem :execrows
UPDATE reference_item
SET is_active = FALSE,
    deleted_at = NOW()
WHERE workspace_id = ?
  AND `key` = ?
  AND deleted_at IS NULL;

-- name: AdminRestoreItem :execrows
UPDATE reference_item
SET is_active = ?,
    deleted_at = NULL
WHERE workspace_id = ?
  AND `key` = ?
  AND deleted_at IS NOT NULL;

-- name: GetItemBundle :many
SELECT
    i.workspace_id,
    i.`key`,
    i.item_type,
    i.payload,
    i.is_active,
    i.deleted_at,
    i.created_at,
    i.updated_at,
    l.locale,
    l.title,
    l.description
FROM reference_item i
LEFT JOIN reference_localization l
  ON l.workspace_id = i.workspace_id
 AND l.item_key = i.`key`
 AND l.locale = ?
WHERE i.workspace_id = ?
  AND i.`key` = ?
  AND i.deleted_at IS NULL
  AND i.is_active = TRUE
LIMIT 1;

-- name: ResolveItemBundles :many
SELECT
    i.workspace_id,
    i.`key`,
    i.item_type,
    i.payload,
    i.is_active,
    i.deleted_at,
    i.created_at,
    i.updated_at,
    l.locale,
    l.title,
    l.description
FROM reference_item i
LEFT JOIN reference_localization l
  ON l.workspace_id = i.workspace_id
 AND l.item_key = i.`key`
 AND l.locale = ?
WHERE i.workspace_id = ?
  AND i.`key` IN (sqlc.slice('keys'))
  AND i.deleted_at IS NULL
  AND i.is_active = TRUE
ORDER BY i.`key`;

-- name: ListItemBundles :many
SELECT
    i.workspace_id,
    i.`key`,
    i.item_type,
    i.payload,
    i.is_active,
    i.deleted_at,
    i.created_at,
    i.updated_at,
    l.locale,
    l.title,
    l.description
FROM reference_item i
LEFT JOIN reference_localization l
  ON l.workspace_id = i.workspace_id
 AND l.item_key = i.`key`
 AND l.locale = ?
WHERE i.workspace_id = ?
  AND i.deleted_at IS NULL
  AND i.is_active = TRUE
ORDER BY i.`key`
LIMIT ? OFFSET ?;

-- name: AdminGetItemBundle :many
SELECT
    i.workspace_id,
    i.`key`,
    i.item_type,
    i.payload,
    i.is_active,
    i.deleted_at,
    i.created_at,
    i.updated_at,
    l.locale,
    l.title,
    l.description
FROM reference_item i
LEFT JOIN reference_localization l
  ON l.workspace_id = i.workspace_id
 AND l.item_key = i.`key`
WHERE i.workspace_id = ?
  AND i.`key` = ?
ORDER BY l.locale;

-- name: AdminListItems :many
SELECT
    workspace_id,
    `key`,
    item_type,
    payload,
    is_active,
    deleted_at,
    created_at,
    updated_at
FROM reference_item
WHERE workspace_id = ?
  AND (? = '' OR item_type = ?)
  AND (? = FALSE OR deleted_at IS NULL)
ORDER BY `key`
LIMIT ? OFFSET ?;

-- name: AdminUpsertLocalization :exec
INSERT INTO reference_localization (
    workspace_id, item_key, locale, title, description
) VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    description = VALUES(description);

-- name: AdminGetLocalization :one
SELECT
    workspace_id, item_key, locale, title, description, created_at, updated_at
FROM reference_localization
WHERE workspace_id = ?
  AND item_key = ?
  AND locale = ?
LIMIT 1;

-- name: AdminListLocalizations :many
SELECT
    workspace_id, item_key, locale, title, description, created_at, updated_at
FROM reference_localization
WHERE workspace_id = ?
  AND item_key = ?
ORDER BY locale;

-- name: AdminDeleteLocalization :execrows
DELETE FROM reference_localization
WHERE workspace_id = ?
  AND item_key = ?
  AND locale = ?;

-- name: AdminGetStats :one
SELECT
    COUNT(*) AS items_total,
    CAST(COALESCE(SUM(deleted_at IS NULL), 0) AS UNSIGNED) AS items_not_deleted,
    CAST(COALESCE(SUM(deleted_at IS NULL AND is_active = TRUE), 0) AS UNSIGNED) AS active_items,
    CAST(COALESCE(SUM(deleted_at IS NOT NULL), 0) AS UNSIGNED) AS deleted_items,
    CAST(COALESCE(SUM(deleted_at IS NULL AND item_type = 'quantity'), 0) AS UNSIGNED) AS quantity_items,
    CAST(COALESCE(SUM(deleted_at IS NULL AND item_type = 'duration'), 0) AS UNSIGNED) AS duration_items
FROM reference_item
WHERE workspace_id = ?;

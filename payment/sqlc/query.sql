-- name: ListProviders :many
SELECT
    code,
    title,
    provider_kind,
    supports_create,
    supports_redirect,
    supports_webhook,
    supports_refund,
    is_active,
    created_at,
    updated_at
FROM payment_provider
ORDER BY code;

-- name: ListAssets :many
SELECT
    code,
    title,
    asset_kind,
    scale,
    chain,
    network,
    contract_address,
    is_active,
    created_at,
    updated_at
FROM payment_asset
ORDER BY code;

-- name: GetAsset :one
SELECT
    code,
    title,
    asset_kind,
    scale,
    chain,
    network,
    contract_address,
    is_active,
    created_at,
    updated_at
FROM payment_asset
WHERE code = ?
  AND is_active = 1
LIMIT 1;

-- name: GetAssetByChainContract :one
SELECT
    code,
    title,
    asset_kind,
    scale,
    chain,
    network,
    contract_address,
    is_active,
    created_at,
    updated_at
FROM payment_asset
WHERE chain = ?
  AND network = ?
  AND contract_address = ?
  AND asset_kind = 'crypto_jetton'
  AND is_active = 1
LIMIT 1;

-- name: UpsertAsset :exec
INSERT INTO payment_asset (
    code,
    title,
    asset_kind,
    scale,
    chain,
    network,
    contract_address,
    is_active
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    asset_kind = VALUES(asset_kind),
    scale = VALUES(scale),
    chain = VALUES(chain),
    network = VALUES(network),
    contract_address = VALUES(contract_address),
    is_active = VALUES(is_active),
    updated_at = NOW();

-- name: DeleteAsset :execrows
DELETE FROM payment_asset
WHERE code = ?;

-- name: GetProviderAsset :one
SELECT
    provider_code,
    asset_code,
    min_amount_minor,
    max_amount_minor,
    merchant_account,
    is_active,
    created_at,
    updated_at
FROM payment_provider_asset
WHERE provider_code = ?
  AND asset_code = ?
LIMIT 1;

-- name: UpsertProviderAsset :exec
INSERT INTO payment_provider_asset (
    provider_code,
    asset_code,
    min_amount_minor,
    max_amount_minor,
    merchant_account,
    is_active
)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    min_amount_minor = VALUES(min_amount_minor),
    max_amount_minor = VALUES(max_amount_minor),
    merchant_account = VALUES(merchant_account),
    is_active = VALUES(is_active),
    updated_at = NOW();

-- name: DeleteProviderAsset :execrows
DELETE FROM payment_provider_asset
WHERE provider_code = ?
  AND asset_code = ?;

-- name: UpsertProductGroup :exec
INSERT INTO payment_product_group (
    workspace_id,
    code,
    title_key,
    description_key,
    position,
    is_active
)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    title_key = VALUES(title_key),
    description_key = VALUES(description_key),
    position = VALUES(position),
    is_active = VALUES(is_active),
    updated_at = NOW();

-- name: DeleteProductGroup :execrows
DELETE FROM payment_product_group
WHERE workspace_id = ?
  AND code = ?;

-- name: UpsertLocalization :exec
INSERT INTO payment_localization (
    workspace_id,
    locale,
    localization_key,
    value
)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    value = VALUES(value),
    updated_at = NOW();

-- name: DeleteLocalization :execrows
DELETE FROM payment_localization
WHERE locale = ?
  AND localization_key = ?
  AND workspace_id = ?;

-- name: UpsertProduct :exec
INSERT INTO payment_product (
    workspace_id,
    id,
    group_code,
    title_key,
    description_key,
    image_url,
    link_url,
    size_label,
    period_seconds,
    trial_duration_seconds,
    quantity_mode,
    position,
    global_limit,
    global_interval,
    global_interval_count,
    user_limit,
    user_interval,
    user_interval_count,
    available_from,
    available_until,
    is_visible,
    is_closed
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    group_code = VALUES(group_code),
    title_key = VALUES(title_key),
    description_key = VALUES(description_key),
    image_url = VALUES(image_url),
    link_url = VALUES(link_url),
    size_label = VALUES(size_label),
    period_seconds = VALUES(period_seconds),
    trial_duration_seconds = VALUES(trial_duration_seconds),
    quantity_mode = VALUES(quantity_mode),
    position = VALUES(position),
    global_limit = VALUES(global_limit),
    global_interval = VALUES(global_interval),
    global_interval_count = VALUES(global_interval_count),
    user_limit = VALUES(user_limit),
    user_interval = VALUES(user_interval),
    user_interval_count = VALUES(user_interval_count),
    available_from = VALUES(available_from),
    available_until = VALUES(available_until),
    is_visible = VALUES(is_visible),
    is_closed = VALUES(is_closed),
    updated_at = NOW();

-- name: DeleteProduct :execrows
DELETE FROM payment_product
WHERE workspace_id = ?
  AND id = ?;

-- name: UpsertItem :exec
INSERT INTO payment_item (
    workspace_id,
    id,
    item_type,
    title_key,
    description_key,
    rarity,
    position
)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    item_type = VALUES(item_type),
    title_key = VALUES(title_key),
    description_key = VALUES(description_key),
    rarity = VALUES(rarity),
    position = VALUES(position),
    updated_at = NOW();

-- name: DeleteItem :execrows
DELETE FROM payment_item
WHERE workspace_id = ?
  AND id = ?;

-- name: ListProductIDsForItem :many
SELECT product_id
FROM payment_product_item
WHERE workspace_id = ?
  AND item_id = ?;

-- name: UpsertProductItem :exec
INSERT INTO payment_product_item (
    workspace_id,
    product_id,
    item_id,
    reward_type,
    quantity,
    duration_unit
)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    reward_type = VALUES(reward_type),
    quantity = VALUES(quantity),
    duration_unit = VALUES(duration_unit),
    updated_at = NOW();

-- name: DeleteProductItem :execrows
DELETE FROM payment_product_item
WHERE product_id = ?
  AND item_id = ?
  AND workspace_id = ?;

-- name: CreateProductPrice :execlastid
INSERT INTO payment_price (
    workspace_id,
    product_id,
    asset_code,
    list_amount_minor,
    discount_amount_minor,
    is_promotion,
    starts_at,
    ends_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateDynamicProductPrice :execlastid
INSERT INTO payment_price (
    workspace_id, product_id, asset_code, list_amount_minor, discount_amount_minor,
    pricing_mode, reference_asset_code, reference_list_amount_minor,
    reference_discount_amount_minor, coefficient, is_promotion, starts_at, ends_at
)
VALUES (?, ?, ?, ?, ?, 'dynamic', ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProductPrice :execrows
UPDATE payment_price
SET asset_code = ?,
    list_amount_minor = ?,
    discount_amount_minor = ?,
    pricing_mode = 'fixed',
    reference_asset_code = NULL,
    reference_list_amount_minor = NULL,
    reference_discount_amount_minor = NULL,
    coefficient = NULL,
    is_promotion = ?,
    starts_at = ?,
    ends_at = ?,
    updated_at = NOW()
WHERE workspace_id = ?
  AND id = ?;

-- name: UpdateDynamicProductPrice :execrows
UPDATE payment_price
SET asset_code = ?,
    list_amount_minor = ?,
    discount_amount_minor = ?,
    pricing_mode = 'dynamic',
    reference_asset_code = ?,
    reference_list_amount_minor = ?,
    reference_discount_amount_minor = ?,
    coefficient = ?,
    is_promotion = ?,
    starts_at = ?,
    ends_at = ?,
    updated_at = NOW()
WHERE workspace_id = ?
  AND id = ?;

-- name: GetAssetRateForPricing :one
SELECT r.reference_per_asset_minor, target.scale AS target_scale
FROM payment_asset_rate r
JOIN payment_asset target ON target.code = r.asset_code
WHERE r.asset_code = ?
  AND r.reference_asset_code = ?
LIMIT 1
FOR UPDATE;

-- name: GetAssetUSDTPrice :one
SELECT
    r.asset_code, a.title AS asset_title, a.scale, r.reference_asset_code,
    r.reference_per_asset_minor, r.source, r.observed_at, r.updated_at
FROM payment_asset_rate r
JOIN payment_asset a ON a.code = r.asset_code
WHERE r.asset_code = ?
  AND r.reference_asset_code = ?
  AND a.is_active = 1
LIMIT 1;

-- name: ListAssetUSDTPrices :many
SELECT
    r.asset_code, a.title AS asset_title, a.scale, r.reference_asset_code,
    r.reference_per_asset_minor, r.source, r.observed_at, r.updated_at
FROM payment_asset_rate r
JOIN payment_asset a ON a.code = r.asset_code
WHERE r.reference_asset_code = ?
  AND a.is_active = 1
ORDER BY r.asset_code;

-- name: UpsertAssetRate :exec
INSERT INTO payment_asset_rate (
    asset_code, reference_asset_code, reference_per_asset_minor, source, observed_at
)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    reference_per_asset_minor = VALUES(reference_per_asset_minor),
    source = VALUES(source),
    observed_at = VALUES(observed_at),
    updated_at = NOW();

-- name: ConfigureAssetRateAutoUpdate :execrows
UPDATE payment_asset_rate
SET auto_update_enabled = ?,
    auto_update_source = ?,
    source_chain_id = ?,
    source_token_address = ?,
    last_error = NULL,
    lease_owner = NULL,
    lease_until = NULL,
    updated_at = NOW()
WHERE asset_code = ?
  AND reference_asset_code = ?;

-- name: ListDueAssetRateUpdates :many
SELECT
    r.asset_code, r.reference_asset_code, r.auto_update_source, r.source_chain_id,
    COALESCE(r.source_token_address, a.contract_address) AS source_token_address
FROM payment_asset_rate r
JOIN payment_asset a ON a.code = r.asset_code
WHERE r.auto_update_enabled = 1
  AND (r.lease_until IS NULL OR r.lease_until < NOW())
ORDER BY r.asset_code
LIMIT ?;

-- name: ClaimAssetRateUpdate :execrows
UPDATE payment_asset_rate
SET lease_owner = ?,
    lease_until = DATE_ADD(NOW(), INTERVAL ? SECOND),
    last_attempt_at = NOW(),
    updated_at = NOW()
WHERE asset_code = ?
  AND reference_asset_code = ?
  AND auto_update_enabled = 1
  AND (lease_until IS NULL OR lease_until < NOW());

-- name: CompleteAssetRateUpdate :execrows
UPDATE payment_asset_rate
SET last_attempt_at = NOW(),
    last_error = NULL,
    lease_owner = NULL,
    lease_until = NULL,
    updated_at = NOW()
WHERE asset_code = ?
  AND reference_asset_code = ?
  AND lease_owner = ?;

-- name: FailAssetRateUpdate :execrows
UPDATE payment_asset_rate
SET last_attempt_at = NOW(),
    last_error = ?,
    lease_owner = NULL,
    lease_until = NULL,
    updated_at = NOW()
WHERE asset_code = ?
  AND reference_asset_code = ?
  AND lease_owner = ?;

-- name: ListDynamicPricesForRate :many
SELECT
    workspace_id, id, product_id, reference_list_amount_minor,
    reference_discount_amount_minor, coefficient
FROM payment_price
WHERE asset_code = ?
  AND reference_asset_code = ?
  AND pricing_mode = 'dynamic'
ORDER BY id
FOR UPDATE;

-- name: UpdateDynamicPriceAmounts :execrows
UPDATE payment_price
SET list_amount_minor = ?,
    discount_amount_minor = ?,
    updated_at = NOW()
WHERE workspace_id = ?
  AND id = ?
  AND pricing_mode = 'dynamic';

-- name: GetProductPriceProductID :one
SELECT product_id
FROM payment_price
WHERE workspace_id = ?
  AND id = ?
LIMIT 1;

-- name: DeleteProductPrice :execrows
DELETE FROM payment_price
WHERE workspace_id = ?
  AND id = ?;

-- name: GetCurrentProductPrice :one
SELECT
    pp.id,
    pp.product_id,
    pp.asset_code,
    pp.list_amount_minor,
    pp.discount_amount_minor,
    pp.pricing_mode,
    pp.reference_asset_code,
    pp.reference_list_amount_minor,
    pp.reference_discount_amount_minor,
    pp.coefficient,
    pp.is_promotion,
    pp.starts_at,
    pp.ends_at,
    pp.created_at,
    pp.updated_at
FROM payment_price pp
JOIN payment_product p
    ON p.workspace_id = pp.workspace_id
   AND p.id = pp.product_id
WHERE pp.workspace_id = ?
  AND p.workspace_id = ?
  AND pp.product_id = ?
  AND pp.asset_code = ?
  AND p.is_visible = 1
  AND p.is_closed = 0
  AND NOW() BETWEEN p.available_from AND p.available_until
  AND NOW() BETWEEN pp.starts_at AND pp.ends_at
ORDER BY pp.is_promotion DESC, pp.starts_at DESC, pp.id DESC
LIMIT 1;

-- name: DeleteWorkspaceProductCache :execrows
DELETE FROM payment_product_cache
WHERE workspace_id = ?;

-- name: DeleteProductCache :execrows
DELETE FROM payment_product_cache
WHERE workspace_id = ?
  AND product_id = ?;

-- name: RebuildWorkspaceProductCache :exec
INSERT INTO payment_product_cache (
    workspace_id,
    product_id,
    asset_code,
    locale,
    price_id,
    item_id,
    link_url,
    size_label,
    group_code,
    product_title,
    product_description,
    image_url,
    period_seconds,
    trial_duration_seconds,
    quantity_mode,
    product_position,
    global_limit,
    global_interval,
    global_interval_count,
    user_limit,
    user_interval,
    user_interval_count,
    is_visible,
    is_closed,
    available_from,
    available_until,
    list_amount_minor,
    discount_amount_minor,
    is_promotion,
    price_starts_at,
    price_ends_at,
    item_quantity,
    reward_type,
    duration_unit,
    item_type,
    item_title,
    item_description,
    item_rarity,
    item_position
)
SELECT
    p.workspace_id,
    p.id AS product_id,
    pp.asset_code,
    loc.locale,
    pp.id AS price_id,
    COALESCE(pi.item_id, '') AS item_id,
    p.link_url,
    p.size_label,
    p.group_code,
    COALESCE(lp_title.value, p.title_key) AS product_title,
    COALESCE(lp_description.value, p.description_key, '') AS product_description,
    p.image_url,
    p.period_seconds,
    p.trial_duration_seconds,
    p.quantity_mode,
    p.position AS product_position,
    p.global_limit,
    p.global_interval,
    p.global_interval_count,
    p.user_limit,
    p.user_interval,
    p.user_interval_count,
    p.is_visible,
    p.is_closed,
    p.available_from,
    p.available_until,
    pp.list_amount_minor,
    pp.discount_amount_minor,
    pp.is_promotion,
    pp.starts_at AS price_starts_at,
    pp.ends_at AS price_ends_at,
    COALESCE(pi.quantity, 0) AS item_quantity,
    COALESCE(pi.reward_type, 'quantity') AS reward_type,
    pi.duration_unit,
    i.item_type,
    COALESCE(li_title.value, i.title_key, '') AS item_title,
    COALESCE(li_description.value, i.description_key, '') AS item_description,
    i.rarity AS item_rarity,
    i.position AS item_position
FROM payment_product p
JOIN payment_price pp
    ON pp.workspace_id = p.workspace_id
   AND pp.product_id = p.id
JOIN (
    SELECT 'ru' AS locale
    UNION SELECT 'en' AS locale
    UNION SELECT 'tr' AS locale
    UNION SELECT 'es' AS locale
    UNION
    SELECT DISTINCT locale
    FROM payment_localization
    WHERE payment_localization.workspace_id = ?
) loc
LEFT JOIN payment_localization lp_title
    ON lp_title.localization_key = p.title_key
   AND lp_title.locale = loc.locale
   AND lp_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization lp_description
    ON lp_description.localization_key = p.description_key
   AND lp_description.locale = loc.locale
   AND lp_description.workspace_id = p.workspace_id
LEFT JOIN payment_product_item pi
    ON pi.product_id = p.id
   AND pi.workspace_id = p.workspace_id
LEFT JOIN payment_item i
    ON i.id = pi.item_id
   AND i.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_title
    ON li_title.localization_key = i.title_key
   AND li_title.locale = loc.locale
   AND li_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_description
    ON li_description.localization_key = i.description_key
   AND li_description.locale = loc.locale
   AND li_description.workspace_id = p.workspace_id
WHERE p.workspace_id = ?;

-- name: RebuildProductCache :exec
INSERT INTO payment_product_cache (
    workspace_id,
    product_id,
    asset_code,
    locale,
    price_id,
    item_id,
    link_url,
    size_label,
    group_code,
    product_title,
    product_description,
    image_url,
    period_seconds,
    trial_duration_seconds,
    quantity_mode,
    product_position,
    global_limit,
    global_interval,
    global_interval_count,
    user_limit,
    user_interval,
    user_interval_count,
    is_visible,
    is_closed,
    available_from,
    available_until,
    list_amount_minor,
    discount_amount_minor,
    is_promotion,
    price_starts_at,
    price_ends_at,
    item_quantity,
    reward_type,
    duration_unit,
    item_type,
    item_title,
    item_description,
    item_rarity,
    item_position
)
SELECT
    p.workspace_id,
    p.id AS product_id,
    pp.asset_code,
    loc.locale,
    pp.id AS price_id,
    COALESCE(pi.item_id, '') AS item_id,
    p.link_url,
    p.size_label,
    p.group_code,
    COALESCE(lp_title.value, p.title_key) AS product_title,
    COALESCE(lp_description.value, p.description_key, '') AS product_description,
    p.image_url,
    p.period_seconds,
    p.trial_duration_seconds,
    p.quantity_mode,
    p.position AS product_position,
    p.global_limit,
    p.global_interval,
    p.global_interval_count,
    p.user_limit,
    p.user_interval,
    p.user_interval_count,
    p.is_visible,
    p.is_closed,
    p.available_from,
    p.available_until,
    pp.list_amount_minor,
    pp.discount_amount_minor,
    pp.is_promotion,
    pp.starts_at AS price_starts_at,
    pp.ends_at AS price_ends_at,
    COALESCE(pi.quantity, 0) AS item_quantity,
    COALESCE(pi.reward_type, 'quantity') AS reward_type,
    pi.duration_unit,
    i.item_type,
    COALESCE(li_title.value, i.title_key, '') AS item_title,
    COALESCE(li_description.value, i.description_key, '') AS item_description,
    i.rarity AS item_rarity,
    i.position AS item_position
FROM payment_product p
JOIN payment_price pp
    ON pp.workspace_id = p.workspace_id
   AND pp.product_id = p.id
JOIN (
    SELECT 'ru' AS locale
    UNION SELECT 'en' AS locale
    UNION SELECT 'tr' AS locale
    UNION SELECT 'es' AS locale
    UNION
    SELECT DISTINCT locale
    FROM payment_localization
    WHERE payment_localization.workspace_id = ?
) loc
LEFT JOIN payment_localization lp_title
    ON lp_title.localization_key = p.title_key
   AND lp_title.locale = loc.locale
   AND lp_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization lp_description
    ON lp_description.localization_key = p.description_key
   AND lp_description.locale = loc.locale
   AND lp_description.workspace_id = p.workspace_id
LEFT JOIN payment_product_item pi
    ON pi.product_id = p.id
   AND pi.workspace_id = p.workspace_id
LEFT JOIN payment_item i
    ON i.id = pi.item_id
   AND i.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_title
    ON li_title.localization_key = i.title_key
   AND li_title.locale = loc.locale
   AND li_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_description
    ON li_description.localization_key = i.description_key
   AND li_description.locale = loc.locale
   AND li_description.workspace_id = p.workspace_id
WHERE p.workspace_id = ?
  AND p.id = ?;

-- name: GetProductRows :many
SELECT
    pc.product_id,
    pc.workspace_id,
    pc.link_url,
    pc.size_label,
    pc.group_code,
    pc.product_title,
    pc.product_description,
    pc.image_url,
    pc.period_seconds,
    pc.trial_duration_seconds,
    pc.quantity_mode,
    pc.global_limit,
    pc.global_interval,
    pc.global_interval_count,
    pc.user_limit,
    pc.user_interval,
    pc.user_interval_count,
    pc.price_id,
    pc.asset_code,
    pc.list_amount_minor,
    pc.discount_amount_minor,
    pc.item_id,
    pc.item_quantity,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.product_id = ?
  AND pc.workspace_id = ?
  AND pc.asset_code = ?
  AND pc.locale = ?
  AND pc.is_visible = 1
  AND pc.is_closed = 0
  AND NOW() BETWEEN pc.available_from AND pc.available_until
  AND NOW() BETWEEN pc.price_starts_at AND pc.price_ends_at
  AND pc.price_id = (
      SELECT pc2.price_id
      FROM payment_product_cache pc2
      WHERE pc2.product_id = pc.product_id
        AND pc2.workspace_id = pc.workspace_id
        AND pc2.asset_code = pc.asset_code
        AND pc2.locale = pc.locale
        AND pc2.is_visible = 1
        AND pc2.is_closed = 0
        AND NOW() BETWEEN pc2.available_from AND pc2.available_until
        AND NOW() BETWEEN pc2.price_starts_at AND pc2.price_ends_at
      ORDER BY pc2.is_promotion DESC, pc2.price_starts_at DESC, pc2.price_id DESC
      LIMIT 1
  )
ORDER BY pc.item_position, pc.item_id;

-- name: ListProductCatalogCacheRows :many
SELECT
    pc.product_id,
    pc.workspace_id,
    pc.link_url,
    pc.size_label,
    pc.group_code,
    pc.product_title,
    pc.product_description,
    pc.image_url,
    pc.period_seconds,
    pc.trial_duration_seconds,
    pc.quantity_mode,
    pc.global_limit,
    pc.global_interval,
    pc.global_interval_count,
    pc.user_limit,
    pc.user_interval,
    pc.user_interval_count,
    pc.is_visible,
    pc.is_closed,
    pc.available_from,
    pc.available_until,
    pc.price_id,
    pc.asset_code,
    pc.list_amount_minor,
    pc.discount_amount_minor,
    pc.is_promotion,
    pc.price_starts_at,
    pc.price_ends_at,
    pc.item_id,
    pc.item_quantity,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.product_id = ?
  AND pc.workspace_id = ?
  AND pc.asset_code = ?
  AND pc.locale = ?
ORDER BY
    pc.is_promotion DESC,
    pc.price_starts_at DESC,
    pc.price_id DESC,
    pc.item_position,
    pc.item_id;

-- name: ListProductsCatalogCacheRows :many
SELECT
    pc.product_id,
    pc.workspace_id,
    pc.link_url,
    pc.size_label,
    pc.group_code,
    pc.product_title,
    pc.product_description,
    pc.image_url,
    pc.period_seconds,
    pc.trial_duration_seconds,
    pc.quantity_mode,
    pc.product_position,
    pc.global_limit,
    pc.global_interval,
    pc.global_interval_count,
    pc.user_limit,
    pc.user_interval,
    pc.user_interval_count,
    pc.is_visible,
    pc.is_closed,
    pc.available_from,
    pc.available_until,
    pc.price_id,
    pc.asset_code,
    pc.list_amount_minor,
    pc.discount_amount_minor,
    pc.is_promotion,
    pc.price_starts_at,
    pc.price_ends_at,
    pc.item_id,
    pc.item_quantity,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.workspace_id = ?
  AND pc.asset_code = ?
  AND pc.locale = ?
ORDER BY
    pc.product_position,
    pc.product_id,
    pc.is_promotion DESC,
    pc.price_starts_at DESC,
    pc.price_id DESC,
    pc.item_position,
    pc.item_id;

-- name: GetCheckoutProduct :one
SELECT
    pc.product_id,
    pc.workspace_id,
    pc.global_limit,
    pc.global_interval,
    pc.global_interval_count,
    pc.user_limit,
    pc.user_interval,
    pc.user_interval_count,
    pc.quantity_mode,
    pc.price_id,
    pc.asset_code,
    pc.list_amount_minor,
    pc.discount_amount_minor
FROM payment_product_cache pc
JOIN (
    SELECT
        pc2.price_id
    FROM payment_product_cache pc2
    WHERE pc2.product_id = ?
      AND pc2.workspace_id = ?
      AND pc2.asset_code = ?
      AND pc2.locale = ?
      AND pc2.is_visible = 1
      AND pc2.is_closed = 0
      AND NOW() BETWEEN pc2.available_from AND pc2.available_until
      AND NOW() BETWEEN pc2.price_starts_at AND pc2.price_ends_at
    ORDER BY
        pc2.is_promotion DESC,
        pc2.price_starts_at DESC,
        pc2.price_id DESC
    LIMIT 1
) ap ON ap.price_id = pc.price_id
WHERE pc.product_id = ?
  AND pc.workspace_id = ?
  AND pc.asset_code = ?
  AND pc.locale = ?
  AND pc.is_visible = 1
  AND pc.is_closed = 0
  AND NOW() BETWEEN pc.available_from AND pc.available_until
  AND NOW() BETWEEN pc.price_starts_at AND pc.price_ends_at
LIMIT 1;

-- name: GetProductRowsRaw :many
SELECT
    p.id AS product_id,
    p.workspace_id,
    p.link_url,
    p.size_label,
    p.group_code,
    COALESCE(lp_title.value, p.title_key) AS product_title,
    COALESCE(lp_description.value, p.description_key, '') AS product_description,
    p.image_url,
    p.period_seconds,
    p.trial_duration_seconds,
    p.quantity_mode,
    p.global_limit,
    p.global_interval,
    p.global_interval_count,
    p.user_limit,
    p.user_interval,
    p.user_interval_count,
    pp.id AS price_id,
    pp.asset_code,
    pp.list_amount_minor,
    pp.discount_amount_minor,
    pi.item_id,
    pi.quantity AS item_quantity,
    pi.reward_type,
    pi.duration_unit,
    i.item_type,
    COALESCE(li_title.value, i.title_key, '') AS item_title,
    COALESCE(li_description.value, i.description_key, '') AS item_description,
    i.rarity AS item_rarity,
    i.position AS item_position
FROM payment_product p
JOIN payment_price pp ON pp.id = (
    SELECT pp2.id
    FROM payment_price pp2
    WHERE pp2.workspace_id = p.workspace_id
      AND pp2.product_id = p.id
      AND pp2.asset_code = ?
      AND NOW() BETWEEN pp2.starts_at AND pp2.ends_at
    ORDER BY pp2.is_promotion DESC, pp2.starts_at DESC, pp2.id DESC
    LIMIT 1
)
LEFT JOIN payment_localization lp_title
    ON lp_title.localization_key = p.title_key
   AND lp_title.locale = ?
   AND lp_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization lp_description
    ON lp_description.localization_key = p.description_key
   AND lp_description.locale = ?
   AND lp_description.workspace_id = p.workspace_id
LEFT JOIN payment_product_item pi
    ON pi.product_id = p.id
   AND pi.workspace_id = p.workspace_id
LEFT JOIN payment_item i
    ON i.id = pi.item_id
   AND i.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_title
    ON li_title.localization_key = i.title_key
   AND li_title.locale = ?
   AND li_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_description
    ON li_description.localization_key = i.description_key
   AND li_description.locale = ?
   AND li_description.workspace_id = p.workspace_id
WHERE p.id = ?
  AND p.workspace_id = ?
  AND p.is_visible = 1
  AND p.is_closed = 0
  AND NOW() BETWEEN p.available_from AND p.available_until;

-- name: GetProductPreviewRows :many
SELECT
    pc.product_id,
    pc.workspace_id,
    pc.link_url,
    pc.size_label,
    pc.group_code,
    pc.product_title,
    pc.product_description,
    pc.image_url,
    pc.period_seconds,
    pc.trial_duration_seconds,
    pc.quantity_mode,
    pc.global_limit,
    pc.global_interval,
    pc.global_interval_count,
    pc.user_limit,
    pc.user_interval,
    pc.user_interval_count,
    pc.item_id,
    pc.item_quantity,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.product_id = ?
  AND pc.workspace_id = ?
  AND pc.locale = ?
  AND pc.is_visible = 1
  AND pc.is_closed = 0
  AND NOW() BETWEEN pc.available_from AND pc.available_until
  AND NOW() BETWEEN pc.price_starts_at AND pc.price_ends_at
  AND pc.price_id = (
      SELECT pc2.price_id
      FROM payment_product_cache pc2
      WHERE pc2.product_id = pc.product_id
        AND pc2.workspace_id = pc.workspace_id
        AND pc2.locale = pc.locale
        AND pc2.is_visible = 1
        AND pc2.is_closed = 0
        AND NOW() BETWEEN pc2.available_from AND pc2.available_until
        AND NOW() BETWEEN pc2.price_starts_at AND pc2.price_ends_at
      ORDER BY pc2.is_promotion DESC, pc2.price_starts_at DESC, pc2.price_id DESC
      LIMIT 1
  )
ORDER BY pc.item_position, pc.item_id;

-- name: ListProductPreviewCatalogCacheRows :many
SELECT
    pc.product_id,
    pc.workspace_id,
    pc.link_url,
    pc.size_label,
    pc.group_code,
    pc.product_title,
    pc.product_description,
    pc.image_url,
    pc.period_seconds,
    pc.trial_duration_seconds,
    pc.quantity_mode,
    pc.global_limit,
    pc.global_interval,
    pc.global_interval_count,
    pc.user_limit,
    pc.user_interval,
    pc.user_interval_count,
    pc.is_visible,
    pc.is_closed,
    pc.available_from,
    pc.available_until,
    pc.price_id,
    pc.is_promotion,
    pc.price_starts_at,
    pc.price_ends_at,
    pc.item_id,
    pc.item_quantity,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.product_id = ?
  AND pc.workspace_id = ?
  AND pc.locale = ?
ORDER BY
    pc.is_promotion DESC,
    pc.price_starts_at DESC,
    pc.price_id DESC,
    pc.item_position,
    pc.item_id;

-- name: GetProductPreviewRowsRaw :many
SELECT
    p.id AS product_id,
    p.workspace_id,
    p.link_url,
    p.size_label,
    p.group_code,
    COALESCE(lp_title.value, p.title_key) AS product_title,
    COALESCE(lp_description.value, p.description_key, '') AS product_description,
    p.image_url,
    p.period_seconds,
    p.trial_duration_seconds,
    p.quantity_mode,
    p.global_limit,
    p.global_interval,
    p.global_interval_count,
    p.user_limit,
    p.user_interval,
    p.user_interval_count,
    pi.item_id,
    pi.quantity AS item_quantity,
    pi.reward_type,
    pi.duration_unit,
    i.item_type,
    COALESCE(li_title.value, i.title_key, '') AS item_title,
    COALESCE(li_description.value, i.description_key, '') AS item_description,
    i.rarity AS item_rarity,
    i.position AS item_position
FROM payment_product p
LEFT JOIN payment_localization lp_title
    ON lp_title.localization_key = p.title_key
   AND lp_title.locale = ?
   AND lp_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization lp_description
    ON lp_description.localization_key = p.description_key
   AND lp_description.locale = ?
   AND lp_description.workspace_id = p.workspace_id
LEFT JOIN payment_product_item pi
    ON pi.product_id = p.id
   AND pi.workspace_id = p.workspace_id
LEFT JOIN payment_item i
    ON i.id = pi.item_id
   AND i.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_title
    ON li_title.localization_key = i.title_key
   AND li_title.locale = ?
   AND li_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_description
    ON li_description.localization_key = i.description_key
   AND li_description.locale = ?
   AND li_description.workspace_id = p.workspace_id
WHERE p.id = ?
  AND p.workspace_id = ?
  AND p.is_visible = 1
  AND p.is_closed = 0
  AND NOW() BETWEEN p.available_from AND p.available_until
ORDER BY i.position, i.id;

-- name: ListProductPriceOptions :many
SELECT
    pp.id AS price_id,
    pp.product_id,
    pp.asset_code,
    a.title AS asset_title,
    a.asset_kind,
    a.scale,
    a.chain,
    a.network,
    a.contract_address,
    pp.list_amount_minor,
    pp.discount_amount_minor,
    GROUP_CONCAT(pa.provider_code ORDER BY pa.provider_code SEPARATOR ',') AS provider_codes
FROM payment_price pp
JOIN payment_asset a
    ON a.code = pp.asset_code
   AND a.is_active = 1
JOIN payment_provider_asset pa
    ON pa.asset_code = pp.asset_code
   AND pa.is_active = 1
JOIN payment_provider p
    ON p.code = pa.provider_code
   AND p.is_active = 1
WHERE pp.workspace_id = ?
  AND pp.product_id = ?
  AND NOW() BETWEEN pp.starts_at AND pp.ends_at
GROUP BY
    pp.id,
    pp.product_id,
    pp.asset_code,
    a.title,
    a.asset_kind,
    a.scale,
    a.chain,
    a.network,
    a.contract_address,
    pp.list_amount_minor,
    pp.discount_amount_minor
ORDER BY pp.asset_code;

-- name: ListProductPriceOptionCatalogRows :many
SELECT
    pp.id AS price_id,
    pp.product_id,
    pp.asset_code,
    a.title AS asset_title,
    a.asset_kind,
    a.scale,
    a.chain,
    a.network,
    a.contract_address,
    pp.list_amount_minor,
    pp.discount_amount_minor,
    pp.starts_at,
    pp.ends_at,
    GROUP_CONCAT(pa.provider_code ORDER BY pa.provider_code SEPARATOR ',') AS provider_codes
FROM payment_price pp
JOIN payment_asset a
    ON a.code = pp.asset_code
   AND a.is_active = 1
JOIN payment_provider_asset pa
    ON pa.asset_code = pp.asset_code
   AND pa.is_active = 1
JOIN payment_provider p
    ON p.code = pa.provider_code
   AND p.is_active = 1
WHERE pp.workspace_id = ?
  AND pp.product_id = ?
GROUP BY
    pp.id,
    pp.product_id,
    pp.asset_code,
    a.title,
    a.asset_kind,
    a.scale,
    a.chain,
    a.network,
    a.contract_address,
    pp.list_amount_minor,
    pp.discount_amount_minor,
    pp.starts_at,
    pp.ends_at
ORDER BY pp.asset_code, pp.starts_at DESC, pp.id DESC;

-- name: ListProductLocales :many
SELECT DISTINCT pc.locale
FROM payment_product_cache pc
WHERE pc.product_id = ?
  AND pc.workspace_id = ?
ORDER BY pc.locale;

-- name: GetProductLimitCounterCount :one
SELECT paid_count
FROM payment_product_limit_counter
WHERE workspace_id = ?
  AND platform_id = ?
  AND product_id = ?
  AND counter_scope = ?
  AND platform_user_id = ?
  AND window_start = ?
  AND window_end = ?
LIMIT 1;

-- name: ListActiveProductLimitCounters :many
SELECT
    product_id,
    counter_scope,
    platform_user_id,
    window_start,
    window_end,
    paid_count
FROM payment_product_limit_counter
WHERE workspace_id = ?
  AND platform_id = ?
  AND platform_user_id IN ('', ?)
  AND window_start <= ?
  AND window_end > ?
ORDER BY product_id, counter_scope, platform_user_id;

-- name: EnsureProductLimitCounter :execrows
INSERT IGNORE INTO payment_product_limit_counter (
    workspace_id,
    platform_id,
    product_id,
    counter_scope,
    platform_user_id,
    window_start,
    window_end,
    paid_count
)
VALUES (?, ?, ?, ?, ?, ?, ?, 0);

-- name: IncrementProductLimitCounter :execrows
UPDATE payment_product_limit_counter
SET paid_count = paid_count + ?,
    updated_at = NOW()
WHERE workspace_id = ?
  AND platform_id = ?
  AND product_id = ?
  AND counter_scope = ?
  AND platform_user_id = ?
  AND window_start = ?
  AND window_end = ?
  AND paid_count + ? <= ?;

-- name: GetProductLimitConfig :one
SELECT
    global_limit,
    global_interval,
    global_interval_count,
    user_limit,
    user_interval,
    user_interval_count
FROM payment_product
WHERE workspace_id = ?
  AND id = ?
LIMIT 1;

-- name: GetPurchaseKeyByHash :one
SELECT
    id,
    workspace_id,
    key_hash,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    status,
    max_uses,
    used_count,
    expires_at,
    created_at,
    updated_at
FROM payment_purchase_key
WHERE key_hash = ?
LIMIT 1;

-- name: LockPurchaseKeyByHash :one
SELECT
    id,
    workspace_id,
    key_hash,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    status,
    max_uses,
    used_count,
    expires_at,
    created_at,
    updated_at
FROM payment_purchase_key
WHERE key_hash = ?
LIMIT 1
FOR UPDATE;

-- name: CreatePurchaseKey :execlastid
INSERT INTO payment_purchase_key (
    workspace_id,
    key_hash,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    max_uses,
    expires_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: IncrementPurchaseKeyUsage :execrows
UPDATE payment_purchase_key
SET used_count = used_count + 1,
    status = IF(used_count + 1 >= max_uses, 'used', status),
    updated_at = NOW()
WHERE id = ?
  AND status = 'active'
  AND used_count < max_uses;

-- name: CreatePaymentOrder :execlastid
INSERT INTO payment_order (
    public_id,
    workspace_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    payer_platform_id,
    payer_platform_user_id,
    payer_internal_user_id,
    purchase_key_id,
    product_id,
    quantity,
    price_id,
    asset_code,
    locale,
    list_amount_minor,
    discount_amount_minor,
    payable_amount_minor,
    status,
    reserved_until,
    expires_at
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: SnapshotPaymentOrderItems :exec
INSERT INTO payment_order_item (
    order_id,
    workspace_id,
    item_id,
    reward_type,
    quantity,
    duration_unit
)
SELECT
    ?,
    pi.workspace_id,
    pi.item_id,
    pi.reward_type,
    pi.quantity * ?,
    pi.duration_unit
FROM payment_product_item pi
WHERE pi.workspace_id = ?
  AND pi.product_id = ?;

-- name: GetPaymentOrder :one
SELECT
    id,
    public_id,
    workspace_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    payer_platform_id,
    payer_platform_user_id,
    payer_internal_user_id,
    purchase_key_id,
    product_id,
    quantity,
    price_id,
    asset_code,
    locale,
    list_amount_minor,
    discount_amount_minor,
    payable_amount_minor,
    status,
    reserved_until,
    paid_at,
    fulfilled_at,
    canceled_at,
    expires_at,
    created_at,
    updated_at
FROM payment_order
WHERE id = ?
LIMIT 1;

-- name: GetPaymentOrderByPublicID :one
SELECT
    id,
    public_id,
    workspace_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    payer_platform_id,
    payer_platform_user_id,
    payer_internal_user_id,
    purchase_key_id,
    product_id,
    quantity,
    price_id,
    asset_code,
    locale,
    list_amount_minor,
    discount_amount_minor,
    payable_amount_minor,
    status,
    reserved_until,
    paid_at,
    fulfilled_at,
    canceled_at,
    expires_at,
    created_at,
    updated_at
FROM payment_order
WHERE public_id = ?
LIMIT 1;

-- name: LockPaymentOrder :one
SELECT
    id,
    public_id,
    workspace_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    payer_platform_id,
    payer_platform_user_id,
    payer_internal_user_id,
    purchase_key_id,
    product_id,
    quantity,
    price_id,
    asset_code,
    locale,
    list_amount_minor,
    discount_amount_minor,
    payable_amount_minor,
    status,
    reserved_until,
    paid_at,
    fulfilled_at,
    canceled_at,
    expires_at,
    created_at,
    updated_at
FROM payment_order
WHERE id = ?
LIMIT 1
FOR UPDATE;

-- name: CreatePaymentAttempt :execlastid
INSERT INTO payment_attempt (
    order_id,
    provider_code,
    asset_code,
    amount_minor,
    status,
    provider_payment_id,
    provider_invoice_id,
    provider_charge_id,
    provider_subscription_id,
    idempotency_key,
    confirmation_url,
    return_url,
    expires_at
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: CreatePaymentAttemptFromOrder :execlastid
INSERT INTO payment_attempt (
    order_id,
    provider_code,
    asset_code,
    amount_minor,
    status,
    provider_payment_id,
    provider_invoice_id,
    provider_charge_id,
    provider_subscription_id,
    idempotency_key,
    confirmation_url,
    return_url,
    expires_at
)
SELECT
    po.id,
    ?,
    po.asset_code,
    po.payable_amount_minor,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
FROM payment_order po
JOIN payment_provider_asset ppa
  ON ppa.provider_code = ?
 AND ppa.asset_code = po.asset_code
 AND ppa.is_active = 1
WHERE po.id = ?
  AND po.status IN ('draft', 'pending_payment');

-- name: GetPaymentAttemptByProviderPaymentID :one
SELECT
    id,
    order_id,
    provider_code,
    asset_code,
    amount_minor,
    status,
    provider_payment_id,
    provider_invoice_id,
    provider_charge_id,
    provider_subscription_id,
    idempotency_key,
    confirmation_url,
    return_url,
    expires_at,
    created_at,
    updated_at
FROM payment_attempt
WHERE provider_code = ?
  AND provider_payment_id = ?
LIMIT 1;

-- name: GetProviderCursor :one
SELECT
    workspace_id,
    provider_code,
    network,
    source_key,
    cursor_value,
    cursor_sequence,
    updated_at
FROM payment_provider_cursor
WHERE workspace_id = ?
  AND provider_code = ?
  AND network = ?
  AND source_key = ?
LIMIT 1;

-- name: UpsertProviderCursor :execrows
INSERT INTO payment_provider_cursor (
    workspace_id,
    provider_code,
    network,
    source_key,
    cursor_value,
    cursor_sequence
)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    cursor_value = IF(
        VALUES(cursor_sequence) >= cursor_sequence,
        VALUES(cursor_value),
        cursor_value
    ),
    cursor_sequence = GREATEST(cursor_sequence, VALUES(cursor_sequence)),
    updated_at = NOW();

-- name: CreateProviderTransaction :execlastid
INSERT INTO payment_provider_transaction (
    workspace_id,
    provider_code,
    network,
    source_key,
    asset_code,
    external_transaction_id,
    sequence_number,
    source_address,
    destination_address,
    amount_minor,
    payment_reference,
    sender_reference,
    order_id,
    attempt_id,
    status,
    error,
    occurred_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetProviderTransactionByExternalID :one
SELECT
    id,
    workspace_id,
    provider_code,
    network,
    source_key,
    asset_code,
    external_transaction_id,
    sequence_number,
    source_address,
    destination_address,
    amount_minor,
    payment_reference,
    sender_reference,
    order_id,
    attempt_id,
    status,
    error,
    occurred_at,
    created_at
FROM payment_provider_transaction
WHERE workspace_id = ?
  AND provider_code = ?
  AND network = ?
  AND source_key = ?
  AND external_transaction_id = ?
LIMIT 1;

-- name: UpsertPaymentSubscription :execlastid
INSERT INTO payment_subscription (
    workspace_id,
    provider_code,
    provider_subscription_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    order_id,
    attempt_id,
    status,
    cancel_reason,
    started_at,
    ended_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    app_id = VALUES(app_id),
    platform_id = VALUES(platform_id),
    platform_user_id = VALUES(platform_user_id),
    internal_user_id = VALUES(internal_user_id),
    product_id = VALUES(product_id),
    order_id = VALUES(order_id),
    attempt_id = VALUES(attempt_id),
    status = VALUES(status),
    cancel_reason = VALUES(cancel_reason),
    started_at = VALUES(started_at),
    ended_at = VALUES(ended_at),
    updated_at = NOW();

-- name: UpdatePaymentSubscriptionStatus :execrows
UPDATE payment_subscription
SET status = ?,
    cancel_reason = ?,
    ended_at = ?,
    updated_at = NOW()
WHERE provider_code = ?
  AND provider_subscription_id = ?;

-- name: GetPaymentSubscriptionByProviderID :one
SELECT
    id,
    workspace_id,
    provider_code,
    provider_subscription_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    order_id,
    attempt_id,
    status,
    cancel_reason,
    started_at,
    ended_at,
    created_at,
    updated_at
FROM payment_subscription
WHERE provider_code = ?
  AND provider_subscription_id = ?
LIMIT 1;

-- name: CountActivePaymentSubscriptionsAll :one
SELECT COUNT(*)
FROM payment_subscription
WHERE platform_id = ?
  AND platform_user_id = ?
  AND workspace_id = ?
  AND status = 'active'
  AND (ended_at IS NULL OR ended_at > ?);

-- name: CountActivePaymentSubscriptionsForProduct :one
SELECT COUNT(*)
FROM payment_subscription
WHERE platform_id = ?
  AND platform_user_id = ?
  AND workspace_id = ?
  AND product_id = ?
  AND status = 'active'
  AND (ended_at IS NULL OR ended_at > ?);

-- name: CountActivePaymentSubscriptionsForProvider :one
SELECT COUNT(*)
FROM payment_subscription
WHERE platform_id = ?
  AND platform_user_id = ?
  AND workspace_id = ?
  AND provider_code = ?
  AND status = 'active'
  AND (ended_at IS NULL OR ended_at > ?);

-- name: CountActivePaymentSubscriptionsForProductProvider :one
SELECT COUNT(*)
FROM payment_subscription
WHERE platform_id = ?
  AND platform_user_id = ?
  AND workspace_id = ?
  AND product_id = ?
  AND provider_code = ?
  AND status = 'active'
  AND (ended_at IS NULL OR ended_at > ?);

-- name: LockPaymentAttempt :one
SELECT
    id,
    order_id,
    provider_code,
    asset_code,
    amount_minor,
    status,
    provider_payment_id,
    provider_invoice_id,
    provider_charge_id,
    provider_subscription_id,
    idempotency_key,
    confirmation_url,
    return_url,
    expires_at,
    created_at,
    updated_at
FROM payment_attempt
WHERE id = ?
LIMIT 1
FOR UPDATE;

-- name: GetFulfilledAttemptResult :one
SELECT
    pa.order_id,
    pa.id AS attempt_id
FROM payment_attempt pa
JOIN payment_order po
  ON po.id = pa.order_id
WHERE pa.id = ?
  AND po.status = 'fulfilled'
LIMIT 1;

-- name: UpdatePaymentAttemptStatus :exec
UPDATE payment_attempt
SET status = ?,
    updated_at = NOW()
WHERE id = ?;

-- name: SetPaymentAttemptProviderChargeID :execrows
UPDATE payment_attempt
SET provider_charge_id = ?,
    updated_at = NOW()
WHERE id = ?
  AND provider_code = ?
  AND (provider_charge_id IS NULL OR provider_charge_id = ?);

-- name: MarkOrderPaid :execrows
UPDATE payment_order
SET status = 'paid',
    paid_at = COALESCE(paid_at, NOW()),
    updated_at = NOW()
WHERE id = ?
  AND status IN ('draft', 'pending_payment');

-- name: InsertPaidOrderIndexFromOrder :execrows
INSERT IGNORE INTO payment_paid_order_index (
    order_id,
    workspace_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    payer_platform_id,
    payer_platform_user_id,
    payer_internal_user_id,
    purchase_key_id,
    product_id,
    quantity,
    price_id,
    asset_code,
    locale,
    list_amount_minor,
    discount_amount_minor,
    payable_amount_minor,
    status,
    paid_at,
    fulfilled_at
)
SELECT
    id,
    workspace_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    payer_platform_id,
    payer_platform_user_id,
    payer_internal_user_id,
    purchase_key_id,
    product_id,
    quantity,
    price_id,
    asset_code,
    locale,
    list_amount_minor,
    discount_amount_minor,
    payable_amount_minor,
    IF(status = 'fulfilled', 'fulfilled', 'paid'),
    COALESCE(paid_at, NOW()),
    fulfilled_at
FROM payment_order
WHERE id = ?
  AND status IN ('paid', 'fulfilled');

-- name: MarkOrderPendingPayment :execrows
UPDATE payment_order
SET status = 'pending_payment',
    updated_at = NOW()
WHERE id = ?
  AND status = 'draft';

-- name: MarkOrderFulfilled :execrows
UPDATE payment_order
SET status = 'fulfilled',
    fulfilled_at = COALESCE(fulfilled_at, NOW()),
    updated_at = NOW()
WHERE id = ?
  AND status IN ('paid', 'fulfilled');

-- name: MarkPaidOrderIndexFulfilled :execrows
UPDATE payment_paid_order_index
SET status = 'fulfilled',
    fulfilled_at = COALESCE(fulfilled_at, NOW()),
    updated_at = NOW()
WHERE order_id = ?;

-- name: CreatePaymentEvent :execlastid
INSERT INTO payment_event (
    provider_code,
    attempt_id,
    order_id,
    provider_event_id,
    provider_payment_id,
    event_type,
    event_status,
    payload_hash,
    signature_valid
)
VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: MarkPaymentEventProcessed :exec
UPDATE payment_event
SET processing_status = ?,
    processing_error = ?,
    processed_at = NOW()
WHERE id = ?;

-- name: CreateFulfillment :execlastid
INSERT INTO payment_fulfillment (
    order_id,
    attempt_id,
    internal_user_id,
    status
)
VALUES (?, ?, ?, ?);

-- name: CreateFulfillmentItem :exec
INSERT INTO payment_fulfillment_item (
    fulfillment_id,
    workspace_id,
    item_id,
    reward_type,
    quantity,
    duration_unit
)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetFulfillmentItemsForProduct :many
SELECT
    item_id,
    reward_type,
    quantity,
    duration_unit
FROM payment_product_item
WHERE workspace_id = ?
  AND product_id = ?
ORDER BY item_id;

-- name: GetFulfillmentItemsForOrder :many
SELECT
    item_id,
    reward_type,
    quantity,
    duration_unit
FROM payment_order_item
WHERE order_id = ?
ORDER BY item_id;

-- Admin queries.

-- name: AdminGetProvider :one
SELECT
    code,
    title,
    provider_kind,
    supports_create,
    supports_redirect,
    supports_webhook,
    supports_refund,
    is_active,
    created_at,
    updated_at
FROM payment_provider
WHERE code = ?
LIMIT 1;

-- name: AdminUpsertProvider :exec
INSERT INTO payment_provider (
    code,
    title,
    provider_kind,
    supports_create,
    supports_redirect,
    supports_webhook,
    supports_refund,
    is_active
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    provider_kind = VALUES(provider_kind),
    supports_create = VALUES(supports_create),
    supports_redirect = VALUES(supports_redirect),
    supports_webhook = VALUES(supports_webhook),
    supports_refund = VALUES(supports_refund),
    is_active = VALUES(is_active),
    updated_at = NOW();

-- name: AdminDeleteProvider :execrows
DELETE FROM payment_provider
WHERE code = ?;

-- name: AdminGetAsset :one
SELECT
    code,
    title,
    asset_kind,
    scale,
    chain,
    network,
    contract_address,
    is_active,
    created_at,
    updated_at
FROM payment_asset
WHERE code = ?
LIMIT 1;

-- name: AdminListProviderAssets :many
SELECT
    provider_code,
    asset_code,
    min_amount_minor,
    max_amount_minor,
    merchant_account,
    is_active,
    created_at,
    updated_at
FROM payment_provider_asset
WHERE (? = '' OR provider_code = ?)
  AND (? = '' OR asset_code = ?)
ORDER BY provider_code, asset_code
LIMIT ? OFFSET ?;

-- name: AdminGetProductGroup :one
SELECT
    workspace_id,
    code,
    title_key,
    description_key,
    position,
    is_active,
    created_at,
    updated_at
FROM payment_product_group
WHERE workspace_id = ?
  AND code = ?
LIMIT 1;

-- name: AdminListProductGroups :many
SELECT
    workspace_id,
    code,
    title_key,
    description_key,
    position,
    is_active,
    created_at,
    updated_at
FROM payment_product_group
WHERE workspace_id = ?
ORDER BY position, code
LIMIT ? OFFSET ?;

-- name: AdminGetLocalization :one
SELECT
    id,
    workspace_id,
    locale,
    localization_key,
    value,
    created_at,
    updated_at
FROM payment_localization
WHERE workspace_id = ?
  AND locale = ?
  AND localization_key = ?
LIMIT 1;

-- name: AdminListLocalizations :many
SELECT
    id,
    workspace_id,
    locale,
    localization_key,
    value,
    created_at,
    updated_at
FROM payment_localization
WHERE workspace_id = ?
  AND (? = '' OR locale = ?)
ORDER BY locale, localization_key
LIMIT ? OFFSET ?;

-- name: AdminGetProduct :one
SELECT
    workspace_id,
    id,
    group_code,
    title_key,
    description_key,
    image_url,
    link_url,
    size_label,
    period_seconds,
    trial_duration_seconds,
    quantity_mode,
    position,
    global_limit,
    global_interval,
    global_interval_count,
    user_limit,
    user_interval,
    user_interval_count,
    available_from,
    available_until,
    is_visible,
    is_closed,
    created_at,
    updated_at
FROM payment_product
WHERE workspace_id = ?
  AND id = ?
LIMIT 1;

-- name: AdminListProducts :many
SELECT
    workspace_id,
    id,
    group_code,
    title_key,
    description_key,
    image_url,
    link_url,
    size_label,
    period_seconds,
    trial_duration_seconds,
    quantity_mode,
    position,
    global_limit,
    global_interval,
    global_interval_count,
    user_limit,
    user_interval,
    user_interval_count,
    available_from,
    available_until,
    is_visible,
    is_closed,
    created_at,
    updated_at
FROM payment_product
WHERE workspace_id = ?
  AND (? = '' OR group_code = ?)
  AND (? = '' OR CAST(quantity_mode AS CHAR) = ?)
ORDER BY position, id
LIMIT ? OFFSET ?;

-- name: AdminGetItem :one
SELECT
    workspace_id,
    id,
    item_type,
    title_key,
    description_key,
    rarity,
    position,
    created_at,
    updated_at
FROM payment_item
WHERE workspace_id = ?
  AND id = ?
LIMIT 1;

-- name: AdminListItems :many
SELECT
    workspace_id,
    id,
    item_type,
    title_key,
    description_key,
    rarity,
    position,
    created_at,
    updated_at
FROM payment_item
WHERE workspace_id = ?
  AND (? = '' OR item_type = ?)
ORDER BY position, id
LIMIT ? OFFSET ?;

-- name: AdminListProductItems :many
SELECT
    id,
    workspace_id,
    product_id,
    item_id,
    reward_type,
    quantity,
    duration_unit,
    created_at,
    updated_at
FROM payment_product_item
WHERE workspace_id = ?
  AND (? = '' OR product_id = ?)
  AND (? = '' OR item_id = ?)
ORDER BY product_id, item_id
LIMIT ? OFFSET ?;

-- name: AdminGetPrice :one
SELECT
    id,
    workspace_id,
    product_id,
    asset_code,
    list_amount_minor,
    discount_amount_minor,
    pricing_mode,
    reference_asset_code,
    reference_list_amount_minor,
    reference_discount_amount_minor,
    coefficient,
    is_promotion,
    starts_at,
    ends_at,
    created_at,
    updated_at
FROM payment_price
WHERE workspace_id = ?
  AND id = ?
LIMIT 1;

-- name: AdminGetAssetRate :one
SELECT
    asset_code, reference_asset_code, reference_per_asset_minor, source, observed_at,
    auto_update_enabled, auto_update_source,
    source_chain_id, source_token_address, last_attempt_at,
    last_error, lease_owner, lease_until, created_at, updated_at
FROM payment_asset_rate
WHERE asset_code = ?
  AND reference_asset_code = ?
LIMIT 1;

-- name: AdminListAssetRates :many
SELECT
    asset_code, reference_asset_code, reference_per_asset_minor, source, observed_at,
    auto_update_enabled, auto_update_source,
    source_chain_id, source_token_address, last_attempt_at,
    last_error, lease_owner, lease_until, created_at, updated_at
FROM payment_asset_rate
WHERE (? = '' OR asset_code = ?)
  AND (? = '' OR reference_asset_code = ?)
ORDER BY asset_code, reference_asset_code
LIMIT ? OFFSET ?;

-- name: AdminListPrices :many
SELECT
    id,
    workspace_id,
    product_id,
    asset_code,
    list_amount_minor,
    discount_amount_minor,
    pricing_mode,
    reference_asset_code,
    reference_list_amount_minor,
    reference_discount_amount_minor,
    coefficient,
    is_promotion,
    starts_at,
    ends_at,
    created_at,
    updated_at
FROM payment_price
WHERE workspace_id = ?
  AND (? = '' OR product_id = ?)
  AND (? = '' OR asset_code = ?)
ORDER BY product_id, asset_code, starts_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminListProductLimitCounters :many
SELECT
    workspace_id,
    platform_id,
    product_id,
    counter_scope,
    platform_user_id,
    window_start,
    window_end,
    paid_count,
    updated_at
FROM payment_product_limit_counter
WHERE workspace_id = ?
  AND (? = '' OR product_id = ?)
  AND (? = 0 OR platform_id = ?)
  AND (? = '' OR platform_user_id = ?)
ORDER BY window_end DESC, product_id, counter_scope, platform_user_id
LIMIT ? OFFSET ?;

-- name: AdminDeleteProductLimitCounter :execrows
DELETE FROM payment_product_limit_counter
WHERE workspace_id = ?
  AND platform_id = ?
  AND product_id = ?
  AND counter_scope = ?
  AND platform_user_id = ?
  AND window_start = ?
  AND window_end = ?;

-- name: AdminGetPurchaseKey :one
SELECT
    id,
    workspace_id,
    key_hash,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    status,
    max_uses,
    used_count,
    expires_at,
    created_at,
    updated_at
FROM payment_purchase_key
WHERE workspace_id = ?
  AND id = ?
LIMIT 1;

-- name: AdminListPurchaseKeys :many
SELECT
    id,
    workspace_id,
    key_hash,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    status,
    max_uses,
    used_count,
    expires_at,
    created_at,
    updated_at
FROM payment_purchase_key
WHERE workspace_id = ?
  AND (? = '' OR product_id = ?)
  AND (? = '' OR CAST(status AS CHAR) = ?)
  AND (? = 0 OR platform_id = ?)
  AND (? = '' OR platform_user_id = ?)
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminUpdatePurchaseKeyStatus :execrows
UPDATE payment_purchase_key
SET status = ?,
    updated_at = NOW()
WHERE workspace_id = ?
  AND id = ?;

-- name: AdminListOrders :many
SELECT
    id,
    public_id,
    workspace_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    payer_platform_id,
    payer_platform_user_id,
    payer_internal_user_id,
    purchase_key_id,
    product_id,
    quantity,
    price_id,
    asset_code,
    locale,
    list_amount_minor,
    discount_amount_minor,
    payable_amount_minor,
    status,
    reserved_until,
    paid_at,
    fulfilled_at,
    canceled_at,
    expires_at,
    created_at,
    updated_at
FROM payment_order
WHERE workspace_id = ?
  AND (? = '' OR CAST(status AS CHAR) = ?)
  AND (? = '' OR product_id = ?)
  AND (? = 0 OR platform_id = ?)
  AND (? = '' OR platform_user_id = ?)
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminUpdateOrderStatus :execrows
UPDATE payment_order
SET status = ?,
    paid_at = IF(? = 'paid' AND paid_at IS NULL, NOW(), paid_at),
    fulfilled_at = IF(? = 'fulfilled' AND fulfilled_at IS NULL, NOW(), fulfilled_at),
    canceled_at = IF(? = 'canceled' AND canceled_at IS NULL, NOW(), canceled_at),
    updated_at = NOW()
WHERE workspace_id = ?
  AND id = ?;

-- name: AdminGetPaymentAttempt :one
SELECT
    id,
    order_id,
    provider_code,
    asset_code,
    amount_minor,
    status,
    provider_payment_id,
    provider_invoice_id,
    provider_charge_id,
    provider_subscription_id,
    idempotency_key,
    confirmation_url,
    return_url,
    expires_at,
    created_at,
    updated_at
FROM payment_attempt
WHERE id = ?
LIMIT 1;

-- name: AdminListPaymentAttempts :many
SELECT
    pa.id,
    pa.order_id,
    pa.provider_code,
    pa.asset_code,
    pa.amount_minor,
    pa.status,
    pa.provider_payment_id,
    pa.provider_invoice_id,
    pa.provider_charge_id,
    pa.provider_subscription_id,
    pa.idempotency_key,
    pa.confirmation_url,
    pa.return_url,
    pa.expires_at,
    pa.created_at,
    pa.updated_at
FROM payment_attempt pa
JOIN payment_order po ON po.id = pa.order_id
WHERE po.workspace_id = ?
  AND (? = 0 OR pa.order_id = ?)
  AND (? = '' OR pa.provider_code = ?)
  AND (? = '' OR CAST(pa.status AS CHAR) = ?)
ORDER BY pa.created_at DESC, pa.id DESC
LIMIT ? OFFSET ?;

-- name: AdminListPaymentEvents :many
SELECT
    pe.id,
    pe.provider_code,
    pe.attempt_id,
    pe.order_id,
    pe.provider_event_id,
    pe.provider_payment_id,
    pe.event_type,
    pe.event_status,
    pe.payload_hash,
    pe.signature_valid,
    pe.processing_status,
    pe.processing_error,
    pe.received_at,
    pe.processed_at
FROM payment_event pe
LEFT JOIN payment_order po ON po.id = pe.order_id
LEFT JOIN payment_attempt pa ON pa.id = pe.attempt_id
LEFT JOIN payment_order pao ON pao.id = pa.order_id
WHERE (po.workspace_id = ? OR pao.workspace_id = ?)
  AND (? = '' OR pe.provider_code = ?)
  AND (? = '' OR CAST(pe.processing_status AS CHAR) = ?)
ORDER BY pe.received_at DESC, pe.id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetPaymentEvent :one
SELECT
    id,
    provider_code,
    attempt_id,
    order_id,
    provider_event_id,
    provider_payment_id,
    event_type,
    event_status,
    payload_hash,
    signature_valid,
    processing_status,
    processing_error,
    received_at,
    processed_at
FROM payment_event
WHERE id = ?
LIMIT 1;

-- name: AdminGetSubscription :one
SELECT
    id,
    workspace_id,
    provider_code,
    provider_subscription_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    order_id,
    attempt_id,
    status,
    cancel_reason,
    started_at,
    ended_at,
    created_at,
    updated_at
FROM payment_subscription
WHERE workspace_id = ?
  AND id = ?
LIMIT 1;

-- name: AdminListSubscriptions :many
SELECT
    id,
    workspace_id,
    provider_code,
    provider_subscription_id,
    app_id,
    platform_id,
    platform_user_id,
    internal_user_id,
    product_id,
    order_id,
    attempt_id,
    status,
    cancel_reason,
    started_at,
    ended_at,
    created_at,
    updated_at
FROM payment_subscription
WHERE workspace_id = ?
  AND (? = '' OR provider_code = ?)
  AND (? = '' OR product_id = ?)
  AND (? = '' OR CAST(status AS CHAR) = ?)
  AND (? = 0 OR platform_id = ?)
  AND (? = '' OR platform_user_id = ?)
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetFulfillment :one
SELECT
    id,
    order_id,
    attempt_id,
    internal_user_id,
    status,
    error,
    created_at,
    updated_at,
    fulfilled_at,
    revoked_at
FROM payment_fulfillment
WHERE id = ?
LIMIT 1;

-- name: AdminListFulfillments :many
SELECT
    pf.id,
    pf.order_id,
    pf.attempt_id,
    pf.internal_user_id,
    pf.status,
    pf.error,
    pf.created_at,
    pf.updated_at,
    pf.fulfilled_at,
    pf.revoked_at
FROM payment_fulfillment pf
JOIN payment_order po ON po.id = pf.order_id
WHERE po.workspace_id = ?
  AND (? = '' OR CAST(pf.status AS CHAR) = ?)
  AND (? = 0 OR pf.order_id = ?)
ORDER BY pf.created_at DESC, pf.id DESC
LIMIT ? OFFSET ?;

-- name: AdminUpdateFulfillmentStatus :execrows
UPDATE payment_fulfillment
SET status = ?,
    error = ?,
    fulfilled_at = IF(? = 'succeeded' AND fulfilled_at IS NULL, NOW(), fulfilled_at),
    revoked_at = IF(? = 'revoked' AND revoked_at IS NULL, NOW(), revoked_at),
    updated_at = NOW()
WHERE id = ?;

-- name: AdminListFulfillmentItems :many
SELECT
    id,
    fulfillment_id,
    workspace_id,
    item_id,
    reward_type,
    quantity,
    duration_unit,
    created_at
FROM payment_fulfillment_item
WHERE workspace_id = ?
  AND (? = 0 OR fulfillment_id = ?)
ORDER BY fulfillment_id, item_id
LIMIT ? OFFSET ?;

-- name: AdminCreateRefund :execlastid
INSERT INTO payment_refund (
    order_id,
    attempt_id,
    provider_code,
    provider_refund_id,
    amount_minor,
    asset_code,
    status,
    reason
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    id = LAST_INSERT_ID(id),
    status = VALUES(status),
    reason = VALUES(reason),
    updated_at = NOW();

-- name: AdminGetRefund :one
SELECT
    id,
    order_id,
    attempt_id,
    provider_code,
    provider_refund_id,
    amount_minor,
    asset_code,
    status,
    reason,
    created_at,
    updated_at
FROM payment_refund
WHERE id = ?
LIMIT 1;

-- name: AdminListRefunds :many
SELECT
    pr.id,
    pr.order_id,
    pr.attempt_id,
    pr.provider_code,
    pr.provider_refund_id,
    pr.amount_minor,
    pr.asset_code,
    pr.status,
    pr.reason,
    pr.created_at,
    pr.updated_at
FROM payment_refund pr
JOIN payment_order po ON po.id = pr.order_id
WHERE po.workspace_id = ?
  AND (? = 0 OR pr.order_id = ?)
  AND (? = '' OR pr.provider_code = ?)
  AND (? = '' OR CAST(pr.status AS CHAR) = ?)
ORDER BY pr.created_at DESC, pr.id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetPaymentStats :one
SELECT
    p.products_total,
    p.active_products,
    p.visible_products,
    o.orders_total,
    o.pending_orders,
    o.fulfilled_orders,
    o.refunded_orders,
    o.failed_orders,
    o.canceled_orders,
    e.purchase_count,
    e.purchase_quantity,
    e.unique_buyers
FROM (
    SELECT
        COUNT(*) AS products_total,
        CAST(COALESCE(SUM(
            is_closed = FALSE
            AND available_from <= NOW()
            AND available_until > NOW()
        ), 0) AS UNSIGNED) AS active_products,
        CAST(COALESCE(SUM(
            is_visible = TRUE
            AND is_closed = FALSE
            AND available_from <= NOW()
            AND available_until > NOW()
        ), 0) AS UNSIGNED) AS visible_products
    FROM payment_product product_rows
    WHERE product_rows.workspace_id = ?
) p
CROSS JOIN (
    SELECT
        COUNT(*) AS orders_total,
        CAST(COALESCE(SUM(status IN ('draft', 'pending_payment', 'paid')), 0) AS UNSIGNED) AS pending_orders,
        CAST(COALESCE(SUM(status = 'fulfilled'), 0) AS UNSIGNED) AS fulfilled_orders,
        CAST(COALESCE(SUM(status = 'refunded'), 0) AS UNSIGNED) AS refunded_orders,
        CAST(COALESCE(SUM(status = 'failed'), 0) AS UNSIGNED) AS failed_orders,
        CAST(COALESCE(SUM(status IN ('canceled', 'expired')), 0) AS UNSIGNED) AS canceled_orders
    FROM payment_order order_rows
    WHERE order_rows.workspace_id = ?
) o
CROSS JOIN (
    SELECT
        CAST(COALESCE(SUM(event_type = 'purchase'), 0) AS UNSIGNED) AS purchase_count,
        CAST(COALESCE(SUM(IF(event_type = 'purchase', quantity, 0)), 0) AS UNSIGNED) AS purchase_quantity,
        COUNT(DISTINCT IF(
            event_type = 'purchase',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_buyers
    FROM payment_stats_event event_rows
    WHERE event_rows.workspace_id = ?
) e;

-- name: AdminGetPaymentProductStats :one
SELECT
    p.id AS product_id,
    COALESCE(o.orders_total, 0) AS orders_total,
    COALESCE(o.pending_orders, 0) AS pending_orders,
    COALESCE(o.fulfilled_orders, 0) AS fulfilled_orders,
    COALESCE(o.refunded_orders, 0) AS refunded_orders,
    COALESCE(o.failed_orders, 0) AS failed_orders,
    COALESCE(o.canceled_orders, 0) AS canceled_orders,
    COALESCE(e.purchase_count, 0) AS purchase_count,
    COALESCE(e.purchase_quantity, 0) AS purchase_quantity,
    COALESCE(e.unique_buyers, 0) AS unique_buyers
FROM payment_product p
LEFT JOIN (
    SELECT
        order_rows.workspace_id,
        order_rows.product_id,
        COUNT(*) AS orders_total,
        CAST(COALESCE(SUM(status IN ('draft', 'pending_payment', 'paid')), 0) AS UNSIGNED) AS pending_orders,
        CAST(COALESCE(SUM(status = 'fulfilled'), 0) AS UNSIGNED) AS fulfilled_orders,
        CAST(COALESCE(SUM(status = 'refunded'), 0) AS UNSIGNED) AS refunded_orders,
        CAST(COALESCE(SUM(status = 'failed'), 0) AS UNSIGNED) AS failed_orders,
        CAST(COALESCE(SUM(status IN ('canceled', 'expired')), 0) AS UNSIGNED) AS canceled_orders
    FROM payment_order order_rows
    WHERE order_rows.workspace_id = ? AND order_rows.product_id = ?
    GROUP BY order_rows.workspace_id, order_rows.product_id
) o ON o.workspace_id = p.workspace_id AND o.product_id = p.id
LEFT JOIN (
    SELECT
        event_rows.workspace_id,
        event_rows.product_id,
        CAST(COALESCE(SUM(event_type = 'purchase'), 0) AS UNSIGNED) AS purchase_count,
        CAST(COALESCE(SUM(IF(event_type = 'purchase', quantity, 0)), 0) AS UNSIGNED) AS purchase_quantity,
        COUNT(DISTINCT IF(
            event_type = 'purchase',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_buyers
    FROM payment_stats_event event_rows
    WHERE event_rows.workspace_id = ? AND event_rows.product_id = ?
    GROUP BY event_rows.workspace_id, event_rows.product_id
) e ON e.workspace_id = p.workspace_id AND e.product_id = p.id
WHERE p.workspace_id = ? AND p.id = ?
LIMIT 1;

-- name: AdminListPaymentAssetStats :many
SELECT
    asset_code,
    CAST(SUM(event_type = 'purchase') AS UNSIGNED) AS purchase_count,
    CAST(SUM(IF(event_type = 'purchase', quantity, 0)) AS UNSIGNED) AS purchase_quantity,
    CAST(SUM(IF(event_type = 'purchase', amount_minor, 0)) AS UNSIGNED) AS gross_amount_minor,
    CAST(SUM(event_type = 'refund') AS UNSIGNED) AS refund_count,
    CAST(SUM(IF(event_type = 'refund', amount_minor, 0)) AS UNSIGNED) AS refund_amount_minor
FROM payment_stats_event
WHERE workspace_id = ?
  AND (? = '' OR product_id = ?)
GROUP BY asset_code
ORDER BY asset_code;

-- name: AdminListPaymentDailyStats :many
SELECT
    workspace_id,
    product_id,
    asset_code,
    stats_date,
    purchase_count,
    purchase_quantity,
    unique_buyers,
    gross_amount_minor,
    refund_count,
    refund_amount_minor,
    updated_at
FROM payment_stats_daily
WHERE workspace_id = ?
  AND product_id = ?
  AND stats_date >= ?
  AND stats_date <= ?
ORDER BY stats_date, asset_code;

-- name: AdminListPaymentDailyOverview :many
SELECT
    workspace_id,
    stats_date,
    products_total,
    active_products,
    visible_products,
    orders_created,
    draft_orders,
    pending_payment_orders,
    paid_orders,
    fulfilled_orders,
    canceled_orders,
    expired_orders,
    refunded_orders,
    chargebacked_orders,
    failed_orders,
    purchase_count,
    purchase_quantity,
    unique_buyers,
    refund_count,
    updated_at
FROM payment_stats_daily_overview stored_overview
WHERE stored_overview.workspace_id = ?
  AND stored_overview.stats_date >= ?
  AND stored_overview.stats_date <= ?
  AND stored_overview.stats_date < CURRENT_DATE
UNION ALL
SELECT
    ? AS workspace_id,
    CURRENT_DATE AS stats_date,
    products.products_total,
    products.active_products,
    products.visible_products,
    overview.orders_created,
    overview.draft_orders,
    overview.pending_payment_orders,
    overview.paid_orders,
    overview.fulfilled_orders,
    overview.canceled_orders,
    overview.expired_orders,
    overview.refunded_orders,
    overview.chargebacked_orders,
    overview.failed_orders,
    overview.purchase_count,
    overview.purchase_quantity,
    overview.unique_buyers,
    overview.refund_count,
    NOW() AS updated_at
FROM (
    SELECT
        COUNT(*) AS products_total,
        CAST(COALESCE(SUM(
            is_closed = FALSE
            AND available_from <= NOW()
            AND available_until > NOW()
        ), 0) AS UNSIGNED) AS active_products,
        CAST(COALESCE(SUM(
            is_visible = TRUE
            AND is_closed = FALSE
            AND available_from <= NOW()
            AND available_until > NOW()
        ), 0) AS UNSIGNED) AS visible_products
    FROM payment_product current_products
    WHERE current_products.workspace_id = ?
) products
CROSS JOIN (
    SELECT
        CAST(COALESCE(MAX(orders_created), 0) AS UNSIGNED) AS orders_created,
        CAST(COALESCE(MAX(draft_orders), 0) AS UNSIGNED) AS draft_orders,
        CAST(COALESCE(MAX(pending_payment_orders), 0) AS UNSIGNED) AS pending_payment_orders,
        CAST(COALESCE(MAX(paid_orders), 0) AS UNSIGNED) AS paid_orders,
        CAST(COALESCE(MAX(fulfilled_orders), 0) AS UNSIGNED) AS fulfilled_orders,
        CAST(COALESCE(MAX(canceled_orders), 0) AS UNSIGNED) AS canceled_orders,
        CAST(COALESCE(MAX(expired_orders), 0) AS UNSIGNED) AS expired_orders,
        CAST(COALESCE(MAX(refunded_orders), 0) AS UNSIGNED) AS refunded_orders,
        CAST(COALESCE(MAX(chargebacked_orders), 0) AS UNSIGNED) AS chargebacked_orders,
        CAST(COALESCE(MAX(failed_orders), 0) AS UNSIGNED) AS failed_orders,
        CAST(COALESCE(MAX(purchase_count), 0) AS UNSIGNED) AS purchase_count,
        CAST(COALESCE(MAX(purchase_quantity), 0) AS UNSIGNED) AS purchase_quantity,
        CAST(COALESCE(MAX(unique_buyers), 0) AS UNSIGNED) AS unique_buyers,
        CAST(COALESCE(MAX(refund_count), 0) AS UNSIGNED) AS refund_count
    FROM payment_stats_daily_overview current_overview
    WHERE current_overview.workspace_id = ?
      AND current_overview.stats_date = CURRENT_DATE
) overview
WHERE CURRENT_DATE >= ?
  AND CURRENT_DATE <= ?
ORDER BY stats_date;

-- name: RefreshPaymentDailyStats :exec
INSERT INTO payment_stats_daily (
    workspace_id, product_id, asset_code, stats_date,
    purchase_count, purchase_quantity, unique_buyers,
    gross_amount_minor, refund_count, refund_amount_minor
)
SELECT
    e.workspace_id,
    e.product_id,
    e.asset_code,
    DATE(e.occurred_at),
    SUM(e.event_type = 'purchase'),
    SUM(IF(e.event_type = 'purchase', e.quantity, 0)),
    COUNT(DISTINCT IF(
        e.event_type = 'purchase',
        CONCAT_WS(':', e.app_id, e.platform_id, e.platform_user_id),
        NULL
    )),
    SUM(IF(e.event_type = 'purchase', e.amount_minor, 0)),
    SUM(e.event_type = 'refund'),
    SUM(IF(e.event_type = 'refund', e.amount_minor, 0))
FROM payment_stats_event e
WHERE e.occurred_at >= ? AND e.occurred_at < ?
GROUP BY e.workspace_id, e.product_id, e.asset_code, DATE(e.occurred_at)
UNION ALL
SELECT
    e.workspace_id,
    '',
    e.asset_code,
    DATE(e.occurred_at),
    SUM(e.event_type = 'purchase'),
    SUM(IF(e.event_type = 'purchase', e.quantity, 0)),
    COUNT(DISTINCT IF(
        e.event_type = 'purchase',
        CONCAT_WS(':', e.app_id, e.platform_id, e.platform_user_id),
        NULL
    )),
    SUM(IF(e.event_type = 'purchase', e.amount_minor, 0)),
    SUM(e.event_type = 'refund'),
    SUM(IF(e.event_type = 'refund', e.amount_minor, 0))
FROM payment_stats_event e
WHERE e.occurred_at >= ? AND e.occurred_at < ?
GROUP BY e.workspace_id, e.asset_code, DATE(e.occurred_at)
ON DUPLICATE KEY UPDATE
    purchase_count = VALUES(purchase_count),
    purchase_quantity = VALUES(purchase_quantity),
    unique_buyers = VALUES(unique_buyers),
    gross_amount_minor = VALUES(gross_amount_minor),
    refund_count = VALUES(refund_count),
    refund_amount_minor = VALUES(refund_amount_minor),
    updated_at = NOW();

-- name: RefreshPaymentDailyOverview :exec
INSERT INTO payment_stats_daily_overview (
    workspace_id,
    stats_date,
    products_total,
    active_products,
    visible_products,
    orders_created,
    draft_orders,
    pending_payment_orders,
    paid_orders,
    fulfilled_orders,
    canceled_orders,
    expired_orders,
    refunded_orders,
    chargebacked_orders,
    failed_orders,
    purchase_count,
    purchase_quantity,
    unique_buyers,
    refund_count
)
SELECT
    dates.workspace_id,
    dates.stats_date,
    COALESCE(products.products_total, 0),
    COALESCE(products.active_products, 0),
    COALESCE(products.visible_products, 0),
    COALESCE(orders.orders_created, 0),
    COALESCE(orders.draft_orders, 0),
    COALESCE(orders.pending_payment_orders, 0),
    COALESCE(orders.paid_orders, 0),
    COALESCE(orders.fulfilled_orders, 0),
    COALESCE(orders.canceled_orders, 0),
    COALESCE(orders.expired_orders, 0),
    COALESCE(orders.refunded_orders, 0),
    COALESCE(orders.chargebacked_orders, 0),
    COALESCE(orders.failed_orders, 0),
    COALESCE(payments.purchase_count, 0),
    COALESCE(payments.purchase_quantity, 0),
    COALESCE(payments.unique_buyers, 0),
    COALESCE(payments.refund_count, 0)
FROM (
    SELECT order_dates.workspace_id, DATE(order_dates.occurred_at) AS stats_date
    FROM payment_stats_order_event order_dates
    WHERE order_dates.occurred_at >= ? AND order_dates.occurred_at < ?
    UNION
    SELECT payment_dates.workspace_id, DATE(payment_dates.occurred_at) AS stats_date
    FROM payment_stats_event payment_dates
    WHERE payment_dates.occurred_at >= ? AND payment_dates.occurred_at < ?
) dates
LEFT JOIN (
    SELECT
        workspace_id,
        COUNT(*) AS products_total,
        SUM(
            is_closed = FALSE
            AND available_from <= NOW()
            AND available_until > NOW()
        ) AS active_products,
        SUM(
            is_visible = TRUE
            AND is_closed = FALSE
            AND available_from <= NOW()
            AND available_until > NOW()
        ) AS visible_products
    FROM payment_product
    GROUP BY workspace_id
) products ON products.workspace_id = dates.workspace_id
LEFT JOIN (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        SUM(event_type = 'created') AS orders_created,
        SUM(event_type = 'status' AND order_status = 'draft') AS draft_orders,
        SUM(event_type = 'status' AND order_status = 'pending_payment') AS pending_payment_orders,
        SUM(event_type = 'status' AND order_status = 'paid') AS paid_orders,
        SUM(event_type = 'status' AND order_status = 'fulfilled') AS fulfilled_orders,
        SUM(event_type = 'status' AND order_status = 'canceled') AS canceled_orders,
        SUM(event_type = 'status' AND order_status = 'expired') AS expired_orders,
        SUM(event_type = 'status' AND order_status = 'refunded') AS refunded_orders,
        SUM(event_type = 'status' AND order_status = 'chargebacked') AS chargebacked_orders,
        SUM(event_type = 'status' AND order_status = 'failed') AS failed_orders
    FROM payment_stats_order_event overview_orders
    WHERE overview_orders.occurred_at >= ? AND overview_orders.occurred_at < ?
    GROUP BY overview_orders.workspace_id, DATE(overview_orders.occurred_at)
) orders
    ON orders.workspace_id = dates.workspace_id
   AND orders.stats_date = dates.stats_date
LEFT JOIN (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        SUM(event_type = 'purchase') AS purchase_count,
        SUM(IF(event_type = 'purchase', quantity, 0)) AS purchase_quantity,
        COUNT(DISTINCT IF(
            event_type = 'purchase',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_buyers,
        SUM(event_type = 'refund') AS refund_count
    FROM payment_stats_event overview_payments
    WHERE overview_payments.occurred_at >= ? AND overview_payments.occurred_at < ?
    GROUP BY overview_payments.workspace_id, DATE(overview_payments.occurred_at)
) payments
    ON payments.workspace_id = dates.workspace_id
   AND payments.stats_date = dates.stats_date
WHERE TRUE
ON DUPLICATE KEY UPDATE
    orders_created = VALUES(orders_created),
    draft_orders = VALUES(draft_orders),
    pending_payment_orders = VALUES(pending_payment_orders),
    paid_orders = VALUES(paid_orders),
    fulfilled_orders = VALUES(fulfilled_orders),
    canceled_orders = VALUES(canceled_orders),
    expired_orders = VALUES(expired_orders),
    refunded_orders = VALUES(refunded_orders),
    chargebacked_orders = VALUES(chargebacked_orders),
    failed_orders = VALUES(failed_orders),
    purchase_count = VALUES(purchase_count),
    purchase_quantity = VALUES(purchase_quantity),
    unique_buyers = VALUES(unique_buyers),
    refund_count = VALUES(refund_count),
    updated_at = NOW();

-- name: AdminUpdateRefundStatus :execrows
UPDATE payment_refund
SET status = ?,
    reason = ?,
    updated_at = NOW()
WHERE id = ?;

-- name: AdminSetRefundProviderID :execrows
UPDATE payment_refund
SET provider_refund_id = ?,
    updated_at = NOW()
WHERE id = ?
  AND (provider_refund_id IS NULL OR provider_refund_id = ?);

-- name: AdminListProviderCursors :many
SELECT
    workspace_id,
    provider_code,
    network,
    source_key,
    cursor_value,
    cursor_sequence,
    updated_at
FROM payment_provider_cursor
WHERE workspace_id = ?
  AND (? = '' OR provider_code = ?)
  AND (? = '' OR network = ?)
ORDER BY provider_code, network, source_key
LIMIT ? OFFSET ?;

-- name: AdminListProviderTransactions :many
SELECT
    id,
    workspace_id,
    provider_code,
    network,
    source_key,
    asset_code,
    external_transaction_id,
    sequence_number,
    source_address,
    destination_address,
    amount_minor,
    payment_reference,
    sender_reference,
    order_id,
    attempt_id,
    status,
    error,
    occurred_at,
    created_at
FROM payment_provider_transaction
WHERE workspace_id = ?
  AND (? = '' OR provider_code = ?)
  AND (? = '' OR network = ?)
  AND (? = '' OR source_key = ?)
  AND (? = '' OR CAST(status AS CHAR) = ?)
ORDER BY sequence_number DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminGetProviderTransaction :one
SELECT
    id,
    workspace_id,
    provider_code,
    network,
    source_key,
    asset_code,
    external_transaction_id,
    sequence_number,
    source_address,
    destination_address,
    amount_minor,
    payment_reference,
    sender_reference,
    order_id,
    attempt_id,
    status,
    error,
    occurred_at,
    created_at
FROM payment_provider_transaction
WHERE workspace_id = ?
  AND id = ?
LIMIT 1;

-- name: AdminUpdateProviderTransactionStatus :execrows
UPDATE payment_provider_transaction
SET status = ?,
    error = ?
WHERE workspace_id = ?
  AND id = ?;

-- name: LockPaymentAttemptByProviderPaymentID :one
SELECT
    id,
    order_id,
    provider_code,
    asset_code,
    amount_minor,
    status,
    provider_payment_id,
    provider_invoice_id,
    provider_charge_id,
    provider_subscription_id,
    idempotency_key,
    confirmation_url,
    return_url,
    expires_at,
    created_at,
    updated_at
FROM payment_attempt
WHERE provider_code = ?
  AND provider_payment_id = ?
LIMIT 1
FOR UPDATE;

-- name: MarkOrderRefunded :execrows
UPDATE payment_order
SET status = 'refunded',
    updated_at = NOW()
WHERE id = ?
  AND status IN ('paid', 'fulfilled', 'refunded');

-- name: MarkFulfillmentRevokedForOrder :execrows
UPDATE payment_fulfillment
SET status = 'revoked',
    revoked_at = COALESCE(revoked_at, NOW()),
    updated_at = NOW()
WHERE order_id = ?
  AND status IN ('pending', 'succeeded', 'revoked');

-- name: GetFulfillmentForOrder :one
SELECT
    id,
    order_id,
    attempt_id,
    internal_user_id,
    status,
    error,
    created_at,
    updated_at,
    fulfilled_at,
    revoked_at
FROM payment_fulfillment
WHERE order_id = ?
LIMIT 1;

-- name: DecrementProductLimitCountersForRefund :execrows
UPDATE payment_product_limit_counter plc
JOIN payment_order po
  ON po.workspace_id = plc.workspace_id
 AND po.platform_id = plc.platform_id
 AND po.product_id = plc.product_id
SET plc.paid_count = GREATEST(plc.paid_count - po.quantity, 0),
    plc.updated_at = NOW()
WHERE po.id = ?
  AND po.paid_at IS NOT NULL
  AND po.paid_at >= plc.window_start
  AND po.paid_at < plc.window_end
  AND (
      (plc.counter_scope = 'global' AND plc.platform_user_id = '')
      OR
      (plc.counter_scope = 'user' AND plc.platform_user_id = po.platform_user_id)
  );

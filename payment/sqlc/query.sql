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
WHERE code = $1
  AND is_active = true
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
WHERE chain = $1
  AND network = $2
  AND contract_address = $3
  AND asset_kind = 'crypto_jetton'
  AND is_active = true
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (code) DO UPDATE SET
    title = EXCLUDED.title,
    asset_kind = EXCLUDED.asset_kind,
    scale = EXCLUDED.scale,
    chain = EXCLUDED.chain,
    network = EXCLUDED.network,
    contract_address = EXCLUDED.contract_address,
    is_active = EXCLUDED.is_active,
    updated_at = now();

-- name: DeleteAsset :execrows
DELETE FROM payment_asset
WHERE code = $1;

-- name: DeleteAssetRatesForAsset :execrows
DELETE FROM payment_asset_rate
WHERE asset_code = $1
   OR reference_asset_code = $2;

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
WHERE provider_code = $1
  AND asset_code = $2
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
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (provider_code, asset_code) DO UPDATE SET
    min_amount_minor = EXCLUDED.min_amount_minor,
    max_amount_minor = EXCLUDED.max_amount_minor,
    merchant_account = EXCLUDED.merchant_account,
    is_active = EXCLUDED.is_active,
    updated_at = now();

-- name: DeleteProviderAsset :execrows
DELETE FROM payment_provider_asset
WHERE provider_code = $1
  AND asset_code = $2;

-- name: UpsertProductGroup :exec
INSERT INTO payment_product_group (
    workspace_id,
    code,
    title_key,
    description_key,
    position,
    is_active
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (workspace_id, code) DO UPDATE SET
    title_key = EXCLUDED.title_key,
    description_key = EXCLUDED.description_key,
    position = EXCLUDED.position,
    is_active = EXCLUDED.is_active,
    updated_at = now();

-- name: DeleteProductGroup :execrows
DELETE FROM payment_product_group
WHERE workspace_id = $1
  AND code = $2;

-- name: UpsertLocalization :exec
INSERT INTO payment_localization (
    workspace_id,
    locale,
    localization_key,
    value
)
VALUES ($1, $2, $3, $4)
ON CONFLICT (workspace_id, locale, localization_key) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = now();

-- name: DeleteLocalization :execrows
DELETE FROM payment_localization
WHERE locale = $1
  AND localization_key = $2
  AND workspace_id = $3;

-- name: UpsertProduct :exec
INSERT INTO payment_product (
    workspace_id,
    id,
    group_code,
    title_key,
    description_key,
    target,
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
ON CONFLICT (workspace_id, id) DO UPDATE SET
    group_code = EXCLUDED.group_code,
    title_key = EXCLUDED.title_key,
    description_key = EXCLUDED.description_key,
    target = EXCLUDED.target,
    image_url = EXCLUDED.image_url,
    link_url = EXCLUDED.link_url,
    size_label = EXCLUDED.size_label,
    period_seconds = EXCLUDED.period_seconds,
    trial_duration_seconds = EXCLUDED.trial_duration_seconds,
    quantity_mode = EXCLUDED.quantity_mode,
    position = EXCLUDED.position,
    global_limit = EXCLUDED.global_limit,
    global_interval = EXCLUDED.global_interval,
    global_interval_count = EXCLUDED.global_interval_count,
    user_limit = EXCLUDED.user_limit,
    user_interval = EXCLUDED.user_interval,
    user_interval_count = EXCLUDED.user_interval_count,
    available_from = EXCLUDED.available_from,
    available_until = EXCLUDED.available_until,
    is_visible = EXCLUDED.is_visible,
    is_closed = EXCLUDED.is_closed,
    updated_at = now();

-- name: DeleteProduct :execrows
DELETE FROM payment_product
WHERE workspace_id = $1
  AND id = $2;

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
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (workspace_id, id) DO UPDATE SET
    item_type = EXCLUDED.item_type,
    title_key = EXCLUDED.title_key,
    description_key = EXCLUDED.description_key,
    rarity = EXCLUDED.rarity,
    position = EXCLUDED.position,
    updated_at = now();

-- name: DeleteItem :execrows
DELETE FROM payment_item
WHERE workspace_id = $1
  AND id = $2;

-- name: ListProductIDsForItem :many
SELECT product_id
FROM payment_product_item
WHERE workspace_id = $1
  AND item_id = $2;

-- name: UpsertProductItem :exec
INSERT INTO payment_product_item (
    workspace_id,
    product_id,
    item_id,
    reward_type,
    quantity,
    scale,
    duration_unit
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (workspace_id, product_id, item_id) DO UPDATE SET
    reward_type = EXCLUDED.reward_type,
    quantity = EXCLUDED.quantity,
    scale = EXCLUDED.scale,
    duration_unit = EXCLUDED.duration_unit,
    updated_at = now();

-- name: DeleteProductItem :execrows
DELETE FROM payment_product_item
WHERE product_id = $1
  AND item_id = $2
  AND workspace_id = $3;

-- name: CreateProductPrice :one
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;
-- name: CreateDynamicProductPrice :one
INSERT INTO payment_price (
    workspace_id, product_id, asset_code, list_amount_minor, discount_amount_minor,
    pricing_mode, reference_asset_code, reference_list_amount_minor,
    reference_discount_amount_minor, coefficient, is_promotion, starts_at, ends_at
)
VALUES ($1, $2, $3, $4, $5, 'dynamic', $6, $7, $8, $9, $10, $11, $12)
RETURNING id;
-- name: UpdateProductPrice :execrows
UPDATE payment_price
SET asset_code = $1,
    list_amount_minor = $2,
    discount_amount_minor = $3,
    pricing_mode = 'fixed',
    reference_asset_code = NULL,
    reference_list_amount_minor = NULL,
    reference_discount_amount_minor = NULL,
    coefficient = NULL,
    is_promotion = $4,
    starts_at = $5,
    ends_at = $6,
    updated_at = now()
WHERE workspace_id = $7
  AND id = $8;

-- name: UpdateDynamicProductPrice :execrows
UPDATE payment_price
SET asset_code = $1,
    list_amount_minor = $2,
    discount_amount_minor = $3,
    pricing_mode = 'dynamic',
    reference_asset_code = $4,
    reference_list_amount_minor = $5,
    reference_discount_amount_minor = $6,
    coefficient = $7,
    is_promotion = $8,
    starts_at = $9,
    ends_at = $10,
    updated_at = now()
WHERE workspace_id = $11
  AND id = $12;

-- name: GetAssetRateForPricing :one
SELECT r.reference_per_asset_minor, target.scale AS target_scale
FROM payment_asset_rate r
JOIN payment_asset target ON target.code = r.asset_code
WHERE r.asset_code = $1
  AND r.reference_asset_code = $2
LIMIT 1
FOR UPDATE;

-- name: GetAssetUSDTPrice :one
SELECT
    r.asset_code, a.title AS asset_title, a.scale, r.reference_asset_code,
    r.reference_per_asset_minor, r.source, r.observed_at, r.updated_at
FROM payment_asset_rate r
JOIN payment_asset a ON a.code = r.asset_code
WHERE r.asset_code = $1
  AND r.reference_asset_code = $2
  AND a.is_active = true
LIMIT 1;

-- name: ListAssetUSDTPrices :many
SELECT
    r.asset_code, a.title AS asset_title, a.scale, r.reference_asset_code,
    r.reference_per_asset_minor, r.source, r.observed_at, r.updated_at
FROM payment_asset_rate r
JOIN payment_asset a ON a.code = r.asset_code
WHERE r.reference_asset_code = $1
  AND a.is_active = true
ORDER BY r.asset_code;

-- name: UpsertAssetRate :exec
INSERT INTO payment_asset_rate (
    asset_code, reference_asset_code, reference_per_asset_minor, source, observed_at
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (asset_code, reference_asset_code) DO UPDATE SET
    reference_per_asset_minor = EXCLUDED.reference_per_asset_minor,
    source = EXCLUDED.source,
    observed_at = EXCLUDED.observed_at,
    updated_at = now();

-- name: SyncAutomaticAssetRates :execrows
INSERT INTO payment_asset_rate (
    asset_code,
    reference_asset_code,
    reference_per_asset_minor,
    source,
    observed_at,
    auto_update_enabled,
    auto_update_source,
    source_chain_id,
    source_token_address
)
SELECT
    a.code,
    $1,
    CASE WHEN a.code = $2 THEN 1000000 ELSE 1 END,
    CASE WHEN a.code = $3 THEN 'fixed' ELSE 'pending' END,
    now(),
    CASE WHEN a.code = $4 THEN false ELSE true END,
    CASE WHEN a.code = $5 THEN NULL ELSE 'dexscreener' END,
    CASE WHEN a.code = $6 THEN NULL ELSE a.chain END,
    CASE WHEN a.code = $7 THEN NULL ELSE a.contract_address END
FROM payment_asset a
WHERE a.is_active = true
  AND (
      a.code = $8
      OR (
          a.asset_kind IN ('crypto_native', 'crypto_jetton')
          AND a.chain IS NOT NULL
          AND a.contract_address IS NOT NULL
      )
  )
ON CONFLICT (asset_code, reference_asset_code) DO UPDATE SET
    reference_per_asset_minor = CASE
        WHEN payment_asset_rate.asset_code = payment_asset_rate.reference_asset_code
            THEN EXCLUDED.reference_per_asset_minor
        ELSE payment_asset_rate.reference_per_asset_minor
    END,
    source = CASE
        WHEN payment_asset_rate.asset_code = payment_asset_rate.reference_asset_code
            THEN EXCLUDED.source
        ELSE payment_asset_rate.source
    END,
    observed_at = CASE
        WHEN payment_asset_rate.asset_code = payment_asset_rate.reference_asset_code
            THEN EXCLUDED.observed_at
        ELSE payment_asset_rate.observed_at
    END,
    auto_update_enabled = EXCLUDED.auto_update_enabled,
    auto_update_source = EXCLUDED.auto_update_source,
    source_chain_id = EXCLUDED.source_chain_id,
    source_token_address = EXCLUDED.source_token_address,
    updated_at = now();

-- name: ConfigureAssetRateAutoUpdate :execrows
UPDATE payment_asset_rate
SET auto_update_enabled = $1,
    auto_update_source = $2,
    source_chain_id = $3,
    source_token_address = $4,
    last_error = NULL,
    lease_owner = NULL,
    lease_until = NULL,
    updated_at = now()
WHERE asset_code = $5
  AND reference_asset_code = $6;

-- name: ListDueAssetRateUpdates :many
SELECT
    r.asset_code, r.reference_asset_code, r.auto_update_source, r.source_chain_id,
    COALESCE(r.source_token_address, a.contract_address) AS source_token_address
FROM payment_asset_rate r
JOIN payment_asset a ON a.code = r.asset_code
WHERE r.auto_update_enabled = true
  AND a.is_active = true
  AND a.asset_kind IN ('crypto_native', 'crypto_jetton')
  AND COALESCE(r.source_token_address, a.contract_address) IS NOT NULL
  AND (r.lease_until IS NULL OR r.lease_until < now())
ORDER BY r.asset_code
LIMIT $1;

-- name: ClaimAssetRateUpdate :execrows
UPDATE payment_asset_rate
SET lease_owner = $1,
    lease_until = now() + make_interval(secs => $2::int),
    last_attempt_at = now(),
    updated_at = now()
WHERE asset_code = $3
  AND reference_asset_code = $4
  AND auto_update_enabled = true
  AND (lease_until IS NULL OR lease_until < now());

-- name: CompleteAssetRateUpdate :execrows
UPDATE payment_asset_rate
SET last_attempt_at = now(),
    last_error = NULL,
    lease_owner = NULL,
    lease_until = NULL,
    updated_at = now()
WHERE asset_code = $1
  AND reference_asset_code = $2
  AND lease_owner = $3;

-- name: FailAssetRateUpdate :execrows
UPDATE payment_asset_rate
SET last_attempt_at = now(),
    last_error = $1,
    lease_owner = NULL,
    lease_until = NULL,
    updated_at = now()
WHERE asset_code = $2
  AND reference_asset_code = $3
  AND lease_owner = $4;

-- name: ListDynamicPricesForRate :many
SELECT
    workspace_id, id, product_id, reference_list_amount_minor,
    reference_discount_amount_minor, coefficient
FROM payment_price
WHERE asset_code = $1
  AND reference_asset_code = $2
  AND pricing_mode = 'dynamic'
ORDER BY id
FOR UPDATE;

-- name: UpdateDynamicPriceAmounts :execrows
UPDATE payment_price
SET list_amount_minor = $1,
    discount_amount_minor = $2,
    updated_at = now()
WHERE workspace_id = $3
  AND id = $4
  AND pricing_mode = 'dynamic';

-- name: GetProductPriceProductID :one
SELECT product_id
FROM payment_price
WHERE workspace_id = $1
  AND id = $2
LIMIT 1;

-- name: DeleteProductPrice :execrows
DELETE FROM payment_price
WHERE workspace_id = $1
  AND id = $2;

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
WHERE pp.workspace_id = $1
  AND p.workspace_id = $2
  AND pp.product_id = $3
  AND pp.asset_code = $4
  AND p.is_visible = true
  AND p.is_closed = false
  AND now() BETWEEN p.available_from AND p.available_until
  AND now() BETWEEN pp.starts_at AND pp.ends_at
ORDER BY pp.is_promotion DESC, pp.starts_at DESC, pp.id DESC
LIMIT 1;

-- name: DeleteWorkspaceProductCache :execrows
DELETE FROM payment_product_cache
WHERE workspace_id = $1;

-- name: DeleteProductCache :execrows
DELETE FROM payment_product_cache
WHERE workspace_id = $1
  AND product_id = $2;

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
    target,
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
    item_scale,
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
    p.target,
    COALESCE(lp_title.value, p.title_key) AS product_title,
    COALESCE(lp_description.value, p.description_key, '') AS product_description,
    p.image_url,
    p.period_seconds,
    p.trial_duration_seconds,
    (p.quantity_mode::text)::payment_product_cache_quantity_mode,
    p.position AS product_position,
    p.global_limit,
    (p.global_interval::text)::payment_product_cache_global_interval,
    p.global_interval_count,
    p.user_limit,
    (p.user_interval::text)::payment_product_cache_user_interval,
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
    COALESCE(pi.scale, 0) AS item_scale,
    (COALESCE(pi.reward_type::text, 'quantity'))::payment_product_cache_reward_type AS reward_type,
    (pi.duration_unit::text)::payment_product_cache_duration_unit,
    i.item_type,
    COALESCE(li_title.value, i.title_key, '') AS item_title,
    COALESCE(li_description.value, i.description_key, '') AS item_description,
    i.rarity AS item_rarity,
    i.position AS item_position
FROM payment_product p
JOIN payment_price pp
    ON pp.workspace_id = p.workspace_id
   AND pp.product_id = p.id
CROSS JOIN (
    SELECT 'ru' AS locale
    UNION SELECT 'en' AS locale
    UNION SELECT 'tr' AS locale
    UNION SELECT 'es' AS locale
    UNION
    SELECT DISTINCT locale
    FROM payment_localization
    WHERE payment_localization.workspace_id = $1
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
WHERE p.workspace_id = $2;

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
    target,
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
    item_scale,
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
    p.target,
    COALESCE(lp_title.value, p.title_key) AS product_title,
    COALESCE(lp_description.value, p.description_key, '') AS product_description,
    p.image_url,
    p.period_seconds,
    p.trial_duration_seconds,
    (p.quantity_mode::text)::payment_product_cache_quantity_mode,
    p.position AS product_position,
    p.global_limit,
    (p.global_interval::text)::payment_product_cache_global_interval,
    p.global_interval_count,
    p.user_limit,
    (p.user_interval::text)::payment_product_cache_user_interval,
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
    COALESCE(pi.scale, 0) AS item_scale,
    (COALESCE(pi.reward_type::text, 'quantity'))::payment_product_cache_reward_type AS reward_type,
    (pi.duration_unit::text)::payment_product_cache_duration_unit,
    i.item_type,
    COALESCE(li_title.value, i.title_key, '') AS item_title,
    COALESCE(li_description.value, i.description_key, '') AS item_description,
    i.rarity AS item_rarity,
    i.position AS item_position
FROM payment_product p
JOIN payment_price pp
    ON pp.workspace_id = p.workspace_id
   AND pp.product_id = p.id
CROSS JOIN (
    SELECT 'ru' AS locale
    UNION SELECT 'en' AS locale
    UNION SELECT 'tr' AS locale
    UNION SELECT 'es' AS locale
    UNION
    SELECT DISTINCT locale
    FROM payment_localization
    WHERE payment_localization.workspace_id = $1
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
WHERE p.workspace_id = $2
  AND p.id = $3;

-- name: GetProductRows :many
SELECT
    pc.product_id,
    pc.workspace_id,
    pc.link_url,
    pc.size_label,
    pc.group_code,
    pc.target,
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
    pc.item_scale,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.product_id = $1
  AND pc.workspace_id = $2
  AND pc.asset_code = $3
  AND pc.locale = $4
  AND pc.is_visible = true
  AND pc.is_closed = false
  AND now() BETWEEN pc.available_from AND pc.available_until
  AND now() BETWEEN pc.price_starts_at AND pc.price_ends_at
  AND pc.price_id = (
      SELECT pc2.price_id
      FROM payment_product_cache pc2
      WHERE pc2.product_id = pc.product_id
        AND pc2.workspace_id = pc.workspace_id
        AND pc2.asset_code = pc.asset_code
        AND pc2.locale = pc.locale
        AND pc2.is_visible = true
        AND pc2.is_closed = false
        AND now() BETWEEN pc2.available_from AND pc2.available_until
        AND now() BETWEEN pc2.price_starts_at AND pc2.price_ends_at
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
    pc.target,
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
    pc.item_scale,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.product_id = $1
  AND pc.workspace_id = $2
  AND pc.asset_code = $3
  AND pc.locale = $4
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
    pc.target,
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
    pc.item_scale,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.workspace_id = $1
  AND pc.asset_code = $2
  AND pc.locale = $3
  AND ($4 = '' OR pc.group_code = $5)
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
    WHERE pc2.product_id = $1
      AND pc2.workspace_id = $2
      AND pc2.asset_code = $3
      AND pc2.locale = $4
      AND pc2.is_visible = true
      AND pc2.is_closed = false
      AND now() BETWEEN pc2.available_from AND pc2.available_until
      AND now() BETWEEN pc2.price_starts_at AND pc2.price_ends_at
    ORDER BY
        pc2.is_promotion DESC,
        pc2.price_starts_at DESC,
        pc2.price_id DESC
    LIMIT 1
) ap ON ap.price_id = pc.price_id
WHERE pc.product_id = $5
  AND pc.workspace_id = $6
  AND pc.asset_code = $7
  AND pc.locale = $8
  AND pc.is_visible = true
  AND pc.is_closed = false
  AND now() BETWEEN pc.available_from AND pc.available_until
  AND now() BETWEEN pc.price_starts_at AND pc.price_ends_at
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
    (p.quantity_mode::text)::payment_product_cache_quantity_mode,
    p.global_limit,
    (p.global_interval::text)::payment_product_cache_global_interval,
    p.global_interval_count,
    p.user_limit,
    (p.user_interval::text)::payment_product_cache_user_interval,
    p.user_interval_count,
    pp.id AS price_id,
    pp.asset_code,
    pp.list_amount_minor,
    pp.discount_amount_minor,
    pi.item_id,
    pi.quantity AS item_quantity,
    pi.scale AS item_scale,
    pi.reward_type,
    (pi.duration_unit::text)::payment_product_cache_duration_unit,
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
      AND pp2.asset_code = $1
      AND now() BETWEEN pp2.starts_at AND pp2.ends_at
    ORDER BY pp2.is_promotion DESC, pp2.starts_at DESC, pp2.id DESC
    LIMIT 1
)
LEFT JOIN payment_localization lp_title
    ON lp_title.localization_key = p.title_key
   AND lp_title.locale = $2
   AND lp_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization lp_description
    ON lp_description.localization_key = p.description_key
   AND lp_description.locale = $3
   AND lp_description.workspace_id = p.workspace_id
LEFT JOIN payment_product_item pi
    ON pi.product_id = p.id
   AND pi.workspace_id = p.workspace_id
LEFT JOIN payment_item i
    ON i.id = pi.item_id
   AND i.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_title
    ON li_title.localization_key = i.title_key
   AND li_title.locale = $4
   AND li_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_description
    ON li_description.localization_key = i.description_key
   AND li_description.locale = $5
   AND li_description.workspace_id = p.workspace_id
WHERE p.id = $6
  AND p.workspace_id = $7
  AND p.is_visible = true
  AND p.is_closed = false
  AND now() BETWEEN p.available_from AND p.available_until;

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
    pc.item_scale,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.product_id = $1
  AND pc.workspace_id = $2
  AND pc.locale = $3
  AND pc.is_visible = true
  AND pc.is_closed = false
  AND now() BETWEEN pc.available_from AND pc.available_until
  AND now() BETWEEN pc.price_starts_at AND pc.price_ends_at
  AND pc.price_id = (
      SELECT pc2.price_id
      FROM payment_product_cache pc2
      WHERE pc2.product_id = pc.product_id
        AND pc2.workspace_id = pc.workspace_id
        AND pc2.locale = pc.locale
        AND pc2.is_visible = true
        AND pc2.is_closed = false
        AND now() BETWEEN pc2.available_from AND pc2.available_until
        AND now() BETWEEN pc2.price_starts_at AND pc2.price_ends_at
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
    pc.item_scale,
    pc.reward_type,
    pc.duration_unit,
    pc.item_type,
    pc.item_title,
    pc.item_description,
    pc.item_rarity,
    pc.item_position
FROM payment_product_cache pc
WHERE pc.product_id = $1
  AND pc.workspace_id = $2
  AND pc.locale = $3
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
    (p.quantity_mode::text)::payment_product_cache_quantity_mode,
    p.global_limit,
    (p.global_interval::text)::payment_product_cache_global_interval,
    p.global_interval_count,
    p.user_limit,
    (p.user_interval::text)::payment_product_cache_user_interval,
    p.user_interval_count,
    pi.item_id,
    pi.quantity AS item_quantity,
    pi.scale AS item_scale,
    pi.reward_type,
    (pi.duration_unit::text)::payment_product_cache_duration_unit,
    i.item_type,
    COALESCE(li_title.value, i.title_key, '') AS item_title,
    COALESCE(li_description.value, i.description_key, '') AS item_description,
    i.rarity AS item_rarity,
    i.position AS item_position
FROM payment_product p
LEFT JOIN payment_localization lp_title
    ON lp_title.localization_key = p.title_key
   AND lp_title.locale = $1
   AND lp_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization lp_description
    ON lp_description.localization_key = p.description_key
   AND lp_description.locale = $2
   AND lp_description.workspace_id = p.workspace_id
LEFT JOIN payment_product_item pi
    ON pi.product_id = p.id
   AND pi.workspace_id = p.workspace_id
LEFT JOIN payment_item i
    ON i.id = pi.item_id
   AND i.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_title
    ON li_title.localization_key = i.title_key
   AND li_title.locale = $3
   AND li_title.workspace_id = p.workspace_id
LEFT JOIN payment_localization li_description
    ON li_description.localization_key = i.description_key
   AND li_description.locale = $4
   AND li_description.workspace_id = p.workspace_id
WHERE p.id = $5
  AND p.workspace_id = $6
  AND p.is_visible = true
  AND p.is_closed = false
  AND now() BETWEEN p.available_from AND p.available_until
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
    string_agg(pa.provider_code, ',' ORDER BY pa.provider_code) AS provider_codes
FROM payment_price pp
JOIN payment_asset a
    ON a.code = pp.asset_code
   AND a.is_active = true
JOIN payment_provider_asset pa
    ON pa.asset_code = pp.asset_code
   AND pa.is_active = true
JOIN payment_provider p
    ON p.code = pa.provider_code
   AND p.is_active = true
WHERE pp.workspace_id = $1
  AND pp.product_id = $2
  AND now() BETWEEN pp.starts_at AND pp.ends_at
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
    string_agg(pa.provider_code, ',' ORDER BY pa.provider_code) AS provider_codes
FROM payment_price pp
JOIN payment_asset a
    ON a.code = pp.asset_code
   AND a.is_active = true
JOIN payment_provider_asset pa
    ON pa.asset_code = pp.asset_code
   AND pa.is_active = true
JOIN payment_provider p
    ON p.code = pa.provider_code
   AND p.is_active = true
WHERE pp.workspace_id = $1
  AND pp.product_id = $2
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
WHERE pc.product_id = $1
  AND pc.workspace_id = $2
ORDER BY pc.locale;

-- name: GetProductLimitCounterCount :one
SELECT paid_count
FROM payment_product_limit_counter
WHERE workspace_id = $1
  AND platform_id = $2
  AND product_id = $3
  AND counter_scope = $4
  AND platform_user_id = $5
  AND window_start = $6
  AND window_end = $7
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
WHERE workspace_id = $1
  AND platform_id = $2
  AND platform_user_id IN ('', $3)
  AND window_start <= $4
  AND window_end > $5
ORDER BY product_id, counter_scope, platform_user_id;

-- name: EnsureProductLimitCounter :execrows
INSERT INTO payment_product_limit_counter (
    workspace_id,
    platform_id,
    product_id,
    counter_scope,
    platform_user_id,
    window_start,
    window_end,
    paid_count
)
VALUES ($1, $2, $3, $4, $5, $6, $7, 0)
ON CONFLICT (workspace_id, platform_id, product_id, counter_scope, platform_user_id, window_start, window_end) DO NOTHING;

-- name: IncrementProductLimitCounter :execrows
UPDATE payment_product_limit_counter
SET paid_count = paid_count + $1,
    updated_at = now()
WHERE workspace_id = $2
  AND platform_id = $3
  AND product_id = $4
  AND counter_scope = $5
  AND platform_user_id = $6
  AND window_start = $7
  AND window_end = $8
  AND paid_count + $9 <= $10;

-- name: GetProductLimitConfig :one
SELECT
    global_limit,
    global_interval,
    global_interval_count,
    user_limit,
    user_interval,
    user_interval_count
FROM payment_product
WHERE workspace_id = $1
  AND id = $2
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
WHERE key_hash = $1
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
WHERE key_hash = $1
LIMIT 1
FOR UPDATE;

-- name: CreatePurchaseKey :one
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;
-- name: IncrementPurchaseKeyUsage :execrows
UPDATE payment_purchase_key
SET used_count = used_count + 1,
    status = CASE WHEN used_count + 1 >= max_uses THEN 'used' ELSE status END,
    updated_at = now()
WHERE id = $1
  AND status = 'active'
  AND used_count < max_uses;

-- name: CreatePaymentOrder :one
WITH created_order AS (
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
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11,
        $12,
        $13,
        $14,
        $15,
        $16,
        $17,
        $18,
        $19,
        $20,
        $21
    )
    RETURNING id, workspace_id, product_id, quantity
),
snapshot_items AS (
    INSERT INTO payment_order_item (
        order_id,
        workspace_id,
        item_id,
        reward_type,
        quantity,
        scale,
        duration_unit
    )
    SELECT
        created_order.id,
        pi.workspace_id,
        pi.item_id,
        (pi.reward_type::text)::payment_order_item_reward_type,
        pi.quantity * created_order.quantity,
        pi.scale,
        (pi.duration_unit::text)::payment_order_item_duration_unit
    FROM created_order
    JOIN payment_product_item pi
      ON pi.workspace_id = created_order.workspace_id
     AND pi.product_id = created_order.product_id
    RETURNING order_id
)
SELECT id
FROM created_order;
-- name: SnapshotPaymentOrderItems :exec
INSERT INTO payment_order_item (
    order_id,
    workspace_id,
    item_id,
    reward_type,
    quantity,
    scale,
    duration_unit
)
SELECT
    $1,
    pi.workspace_id,
    pi.item_id,
    (pi.reward_type::text)::payment_order_item_reward_type,
    pi.quantity * $2,
    pi.scale,
    (pi.duration_unit::text)::payment_order_item_duration_unit
FROM payment_product_item pi
WHERE pi.workspace_id = $3
  AND pi.product_id = $4;

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
WHERE id = $1
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
WHERE public_id = $1
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
WHERE id = $1
LIMIT 1
FOR UPDATE;

-- name: CreatePaymentAttempt :one
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
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13
)
RETURNING id;
-- name: CreatePaymentAttemptFromOrder :one
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
    $1,
    po.asset_code,
    po.payable_amount_minor,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10
FROM payment_order po
JOIN payment_provider_asset ppa
  ON ppa.provider_code = $11
 AND ppa.asset_code = po.asset_code
 AND ppa.is_active = true
WHERE po.id = $12
  AND po.status IN ('draft', 'pending_payment')
RETURNING id;
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
WHERE provider_code = $1
  AND provider_payment_id = $2
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
WHERE workspace_id = $1
  AND provider_code = $2
  AND network = $3
  AND source_key = $4
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
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (workspace_id, provider_code, network, source_key) DO UPDATE SET
    cursor_value = CASE WHEN EXCLUDED.cursor_sequence >= payment_provider_cursor.cursor_sequence THEN EXCLUDED.cursor_value ELSE payment_provider_cursor.cursor_value END,
    cursor_sequence = GREATEST(payment_provider_cursor.cursor_sequence, EXCLUDED.cursor_sequence),
    updated_at = now();

-- name: UpsertTONWallet :exec
INSERT INTO payment_ton_wallet (
    workspace_id,
    network,
    wallet_address,
    network_config_url,
    is_enabled
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (workspace_id) DO UPDATE SET
    network = EXCLUDED.network,
    wallet_address = EXCLUDED.wallet_address,
    network_config_url = EXCLUDED.network_config_url,
    is_enabled = EXCLUDED.is_enabled,
    updated_at = now();

-- name: DeleteTONWallet :execrows
DELETE FROM payment_ton_wallet
WHERE workspace_id = $1;

-- name: AdminGetTONWallet :one
SELECT
    workspace_id,
    network,
    wallet_address,
    network_config_url,
    is_enabled,
    created_at,
    updated_at
FROM payment_ton_wallet
WHERE workspace_id = $1
LIMIT 1;

-- name: ListEnabledTONWallets :many
SELECT
    workspace_id,
    network,
    wallet_address,
    network_config_url,
    is_enabled,
    created_at,
    updated_at
FROM payment_ton_wallet
WHERE is_enabled = true
ORDER BY workspace_id, network, wallet_address;

-- name: GetEnabledTONWalletForWorkspace :one
SELECT
    workspace_id,
    network,
    wallet_address,
    network_config_url,
    is_enabled,
    created_at,
    updated_at
FROM payment_ton_wallet
WHERE workspace_id = $1
  AND is_enabled = true
LIMIT 1;

-- name: CreateProviderTransaction :one
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING id;
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
WHERE workspace_id = $1
  AND provider_code = $2
  AND network = $3
  AND source_key = $4
  AND external_transaction_id = $5
LIMIT 1;

-- name: UpsertPaymentSubscription :one
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (provider_code, provider_subscription_id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    app_id = EXCLUDED.app_id,
    platform_id = EXCLUDED.platform_id,
    platform_user_id = EXCLUDED.platform_user_id,
    internal_user_id = EXCLUDED.internal_user_id,
    product_id = EXCLUDED.product_id,
    order_id = EXCLUDED.order_id,
    attempt_id = EXCLUDED.attempt_id,
    status = EXCLUDED.status,
    cancel_reason = EXCLUDED.cancel_reason,
    started_at = EXCLUDED.started_at,
    ended_at = EXCLUDED.ended_at,
    updated_at = now()
RETURNING id;
-- name: UpdatePaymentSubscriptionStatus :execrows
UPDATE payment_subscription
SET status = $1,
    cancel_reason = $2,
    ended_at = $3,
    updated_at = now()
WHERE provider_code = $4
  AND provider_subscription_id = $5;

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
WHERE provider_code = $1
  AND provider_subscription_id = $2
LIMIT 1;

-- name: CountActivePaymentSubscriptionsAll :one
SELECT COUNT(*)
FROM payment_subscription
WHERE platform_id = $1
  AND platform_user_id = $2
  AND workspace_id = $3
  AND status = 'active'
  AND (ended_at IS NULL OR ended_at > $4);

-- name: CountActivePaymentSubscriptionsForProduct :one
SELECT COUNT(*)
FROM payment_subscription
WHERE platform_id = $1
  AND platform_user_id = $2
  AND workspace_id = $3
  AND product_id = $4
  AND status = 'active'
  AND (ended_at IS NULL OR ended_at > $5);

-- name: CountActivePaymentSubscriptionsForProvider :one
SELECT COUNT(*)
FROM payment_subscription
WHERE platform_id = $1
  AND platform_user_id = $2
  AND workspace_id = $3
  AND provider_code = $4
  AND status = 'active'
  AND (ended_at IS NULL OR ended_at > $5);

-- name: CountActivePaymentSubscriptionsForProductProvider :one
SELECT COUNT(*)
FROM payment_subscription
WHERE platform_id = $1
  AND platform_user_id = $2
  AND workspace_id = $3
  AND product_id = $4
  AND provider_code = $5
  AND status = 'active'
  AND (ended_at IS NULL OR ended_at > $6);

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
WHERE id = $1
LIMIT 1
FOR UPDATE;

-- name: GetFulfilledAttemptResult :one
SELECT
    pa.order_id,
    pa.id AS attempt_id
FROM payment_attempt pa
JOIN payment_order po
  ON po.id = pa.order_id
WHERE pa.id = $1
  AND po.status = 'fulfilled'
LIMIT 1;

-- name: UpdatePaymentAttemptStatus :exec
UPDATE payment_attempt
SET status = $1,
    updated_at = now()
WHERE id = $2;

-- name: SetPaymentAttemptProviderChargeID :execrows
UPDATE payment_attempt
SET provider_charge_id = $1,
    updated_at = now()
WHERE id = $2
  AND provider_code = $3
  AND (provider_charge_id IS NULL OR provider_charge_id = $4);

-- name: MarkOrderPaid :execrows
UPDATE payment_order
SET status = 'paid',
    paid_at = COALESCE(paid_at, now()),
    updated_at = now()
WHERE id = $1
  AND status IN ('draft', 'pending_payment');

-- name: InsertPaidOrderIndexFromOrder :execrows
INSERT INTO payment_paid_order_index (
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
    (CASE WHEN status = 'fulfilled' THEN 'fulfilled' ELSE 'paid' END)::payment_paid_order_index_status,
    COALESCE(paid_at, now()),
    fulfilled_at
FROM payment_order
WHERE id = $1
  AND status IN ('paid', 'fulfilled')
ON CONFLICT (order_id) DO NOTHING;

-- name: MarkOrderPaidAndIndex :one
WITH marked_order AS (
    UPDATE payment_order
    SET status = 'paid',
        paid_at = COALESCE(paid_at, now()),
        updated_at = now()
    WHERE payment_order.id = $1
      AND payment_order.status IN ('draft', 'pending_payment')
    RETURNING
        payment_order.id,
        payment_order.workspace_id,
        payment_order.app_id,
        payment_order.platform_id,
        payment_order.platform_user_id,
        payment_order.internal_user_id,
        payment_order.payer_platform_id,
        payment_order.payer_platform_user_id,
        payment_order.payer_internal_user_id,
        payment_order.purchase_key_id,
        payment_order.product_id,
        payment_order.quantity,
        payment_order.price_id,
        payment_order.asset_code,
        payment_order.locale,
        payment_order.list_amount_minor,
        payment_order.discount_amount_minor,
        payment_order.payable_amount_minor,
        payment_order.status,
        payment_order.paid_at,
        payment_order.fulfilled_at
),
source_order AS (
    SELECT * FROM marked_order
    UNION ALL
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
        status,
        paid_at,
        fulfilled_at
    FROM payment_order
    WHERE id = $1
      AND NOT EXISTS (SELECT 1 FROM marked_order)
      AND status IN ('paid', 'fulfilled')
),
inserted_index AS (
    INSERT INTO payment_paid_order_index (
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
        (CASE WHEN status = 'fulfilled' THEN 'fulfilled' ELSE 'paid' END)::payment_paid_order_index_status,
        COALESCE(paid_at, now()),
        fulfilled_at
    FROM source_order
    ON CONFLICT (order_id) DO NOTHING
    RETURNING order_id
)
SELECT
    EXISTS (SELECT 1 FROM inserted_index) AS inserted
FROM source_order
LIMIT 1;

-- name: MarkOrderPendingPayment :execrows
UPDATE payment_order
SET status = 'pending_payment',
    updated_at = now()
WHERE id = $1
  AND status = 'draft';

-- name: MarkOrderFulfilled :execrows
UPDATE payment_order
SET status = 'fulfilled',
    fulfilled_at = COALESCE(fulfilled_at, now()),
    updated_at = now()
WHERE id = $1
  AND status IN ('paid', 'fulfilled');

-- name: MarkPaidOrderIndexFulfilled :execrows
UPDATE payment_paid_order_index
SET status = 'fulfilled',
    fulfilled_at = COALESCE(fulfilled_at, now()),
    updated_at = now()
WHERE order_id = $1;

-- name: CreatePaymentEvent :one
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
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
)
RETURNING id;
-- name: MarkPaymentEventProcessed :exec
UPDATE payment_event
SET processing_status = $1,
    processing_error = $2,
    processed_at = now()
WHERE id = $3;

-- name: CreateFulfillment :one
INSERT INTO payment_fulfillment (
    order_id,
    attempt_id,
    internal_user_id,
    status
)
VALUES ($1, $2, $3, $4)
RETURNING id;
-- name: CompleteFulfillmentFromOrder :one
WITH created_fulfillment AS (
    INSERT INTO payment_fulfillment (
        order_id,
        attempt_id,
        internal_user_id,
        status
    )
    VALUES ($1, $2, $3, $4)
    RETURNING id, order_id
),
created_items AS (
    INSERT INTO payment_fulfillment_item (
        fulfillment_id,
        workspace_id,
        item_id,
        reward_type,
        quantity,
        scale,
        duration_unit
    )
    SELECT
        created_fulfillment.id,
        oi.workspace_id,
        oi.item_id,
        (oi.reward_type::text)::payment_fulfillment_item_reward_type,
        oi.quantity,
        oi.scale,
        (oi.duration_unit::text)::payment_fulfillment_item_duration_unit
    FROM created_fulfillment
    JOIN payment_order_item oi
      ON oi.order_id = created_fulfillment.order_id
    RETURNING fulfillment_id
),
marked_order AS (
    UPDATE payment_order
    SET status = 'fulfilled',
        fulfilled_at = COALESCE(fulfilled_at, now()),
        updated_at = now()
    WHERE id = $1
      AND status IN ('paid', 'fulfilled')
    RETURNING id
),
marked_index AS (
    UPDATE payment_paid_order_index
    SET status = 'fulfilled',
        fulfilled_at = COALESCE(fulfilled_at, now()),
        updated_at = now()
    WHERE order_id = $1
    RETURNING order_id
)
SELECT id
FROM created_fulfillment;
-- name: CreateFulfillmentItem :exec
INSERT INTO payment_fulfillment_item (
    fulfillment_id,
    workspace_id,
    item_id,
    reward_type,
    quantity,
    scale,
    duration_unit
)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetFulfillmentItemsForProduct :many
SELECT
    item_id,
    reward_type,
    quantity,
    scale,
    duration_unit
FROM payment_product_item
WHERE workspace_id = $1
  AND product_id = $2
ORDER BY item_id;

-- name: GetFulfillmentItemsForOrder :many
SELECT
    item_id,
    reward_type,
    quantity,
    scale,
    duration_unit
FROM payment_order_item
WHERE order_id = $1
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
WHERE code = $1
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (code) DO UPDATE SET
    title = EXCLUDED.title,
    provider_kind = EXCLUDED.provider_kind,
    supports_create = EXCLUDED.supports_create,
    supports_redirect = EXCLUDED.supports_redirect,
    supports_webhook = EXCLUDED.supports_webhook,
    supports_refund = EXCLUDED.supports_refund,
    is_active = EXCLUDED.is_active,
    updated_at = now();

-- name: AdminDeleteProvider :execrows
DELETE FROM payment_provider
WHERE code = $1;

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
WHERE code = $1
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
WHERE ($1 = '' OR provider_code = $2)
  AND ($3 = '' OR asset_code = $4)
ORDER BY provider_code, asset_code
LIMIT $5 OFFSET $6;

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
WHERE workspace_id = $1
  AND code = $2
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
WHERE workspace_id = $1
ORDER BY position, code
LIMIT $2 OFFSET $3;

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
WHERE workspace_id = $1
  AND locale = $2
  AND localization_key = $3
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
WHERE workspace_id = $1
  AND ($2 = '' OR locale = $3)
ORDER BY locale, localization_key
LIMIT $4 OFFSET $5;

-- name: AdminGetProduct :one
SELECT
    workspace_id,
    id,
    group_code,
    title_key,
    description_key,
    target,
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
WHERE workspace_id = $1
  AND id = $2
LIMIT 1;

-- name: AdminListProducts :many
SELECT
    workspace_id,
    id,
    group_code,
    title_key,
    description_key,
    target,
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
WHERE workspace_id = $1
  AND ($2 = '' OR group_code = $3)
  AND ($4 = '' OR CAST(quantity_mode AS TEXT) = $5)
ORDER BY position, id
LIMIT $6 OFFSET $7;

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
WHERE workspace_id = $1
  AND id = $2
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
WHERE workspace_id = $1
  AND ($2 = '' OR item_type = $3)
ORDER BY position, id
LIMIT $4 OFFSET $5;

-- name: AdminListProductItems :many
SELECT
    id,
    workspace_id,
    product_id,
    item_id,
    reward_type,
    quantity,
    scale,
    duration_unit,
    created_at,
    updated_at
FROM payment_product_item
WHERE workspace_id = $1
  AND ($2 = '' OR product_id = $3)
  AND ($4 = '' OR item_id = $5)
ORDER BY product_id, item_id
LIMIT $6 OFFSET $7;

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
WHERE workspace_id = $1
  AND id = $2
LIMIT 1;

-- name: AdminGetAssetRate :one
SELECT
    asset_code, reference_asset_code, reference_per_asset_minor, source, observed_at,
    auto_update_enabled, auto_update_source,
    source_chain_id, source_token_address, last_attempt_at,
    last_error, lease_owner, lease_until, created_at, updated_at
FROM payment_asset_rate
WHERE asset_code = $1
  AND reference_asset_code = $2
LIMIT 1;

-- name: AdminListAssetRates :many
SELECT
    asset_code, reference_asset_code, reference_per_asset_minor, source, observed_at,
    auto_update_enabled, auto_update_source,
    source_chain_id, source_token_address, last_attempt_at,
    last_error, lease_owner, lease_until, created_at, updated_at
FROM payment_asset_rate
WHERE ($1 = '' OR asset_code = $2)
  AND ($3 = '' OR reference_asset_code = $4)
ORDER BY asset_code, reference_asset_code
LIMIT $5 OFFSET $6;

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
WHERE workspace_id = $1
  AND ($2 = '' OR product_id = $3)
  AND ($4 = '' OR asset_code = $5)
ORDER BY product_id, asset_code, starts_at DESC, id DESC
LIMIT $6 OFFSET $7;

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
WHERE workspace_id = $1
  AND ($2 = '' OR product_id = $3)
  AND ($4 = 0 OR platform_id = $5)
  AND ($6 = '' OR platform_user_id = $7)
ORDER BY window_end DESC, product_id, counter_scope, platform_user_id
LIMIT $8 OFFSET $9;

-- name: AdminDeleteProductLimitCounter :execrows
DELETE FROM payment_product_limit_counter
WHERE workspace_id = $1
  AND platform_id = $2
  AND product_id = $3
  AND counter_scope = $4
  AND platform_user_id = $5
  AND window_start = $6
  AND window_end = $7;

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
WHERE workspace_id = $1
  AND id = $2
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
WHERE workspace_id = $1
  AND ($2 = '' OR product_id = $3)
  AND ($4 = '' OR CAST(status AS TEXT) = $5)
  AND ($6 = 0 OR platform_id = $7)
  AND ($8 = '' OR platform_user_id = $9)
ORDER BY created_at DESC, id DESC
LIMIT $10 OFFSET $11;

-- name: AdminUpdatePurchaseKeyStatus :execrows
UPDATE payment_purchase_key
SET status = $1,
    updated_at = now()
WHERE workspace_id = $2
  AND id = $3;

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
WHERE workspace_id = $1
  AND ($2 = '' OR CAST(status AS TEXT) = $3)
  AND ($4 = '' OR product_id = $5)
  AND ($6 = 0 OR platform_id = $7)
  AND ($8 = '' OR platform_user_id = $9)
ORDER BY created_at DESC, id DESC
LIMIT $10 OFFSET $11;

-- name: AdminUpdateOrderStatus :execrows
UPDATE payment_order
SET status = $1,
    paid_at = CASE WHEN $2 = 'paid' AND paid_at IS NULL THEN now() ELSE paid_at END,
    fulfilled_at = CASE WHEN $3 = 'fulfilled' AND fulfilled_at IS NULL THEN now() ELSE fulfilled_at END,
    canceled_at = CASE WHEN $4 = 'canceled' AND canceled_at IS NULL THEN now() ELSE canceled_at END,
    updated_at = now()
WHERE workspace_id = $5
  AND id = $6;

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
WHERE id = $1
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
WHERE po.workspace_id = $1
  AND ($2 = 0 OR pa.order_id = $3)
  AND ($4 = '' OR pa.provider_code = $5)
  AND ($6 = '' OR CAST(pa.status AS TEXT) = $7)
ORDER BY pa.created_at DESC, pa.id DESC
LIMIT $8 OFFSET $9;

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
WHERE (po.workspace_id = $1 OR pao.workspace_id = $2)
  AND ($3 = '' OR pe.provider_code = $4)
  AND ($5 = '' OR CAST(pe.processing_status AS TEXT) = $6)
ORDER BY pe.received_at DESC, pe.id DESC
LIMIT $7 OFFSET $8;

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
WHERE id = $1
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
WHERE workspace_id = $1
  AND id = $2
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
WHERE workspace_id = $1
  AND ($2 = '' OR provider_code = $3)
  AND ($4 = '' OR product_id = $5)
  AND ($6 = '' OR CAST(status AS TEXT) = $7)
  AND ($8 = 0 OR platform_id = $9)
  AND ($10 = '' OR platform_user_id = $11)
ORDER BY created_at DESC, id DESC
LIMIT $12 OFFSET $13;

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
WHERE id = $1
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
WHERE po.workspace_id = $1
  AND ($2 = '' OR CAST(pf.status AS TEXT) = $3)
  AND ($4 = 0 OR pf.order_id = $5)
ORDER BY pf.created_at DESC, pf.id DESC
LIMIT $6 OFFSET $7;

-- name: AdminUpdateFulfillmentStatus :execrows
UPDATE payment_fulfillment
SET status = $1,
    error = $2,
    fulfilled_at = CASE WHEN $3 = 'succeeded' AND fulfilled_at IS NULL THEN now() ELSE fulfilled_at END,
    revoked_at = CASE WHEN $4 = 'revoked' AND revoked_at IS NULL THEN now() ELSE revoked_at END,
    updated_at = now()
WHERE id = $5;

-- name: AdminListFulfillmentItems :many
SELECT
    id,
    fulfillment_id,
    workspace_id,
    item_id,
    reward_type,
    quantity,
    scale,
    duration_unit,
    created_at
FROM payment_fulfillment_item
WHERE workspace_id = $1
  AND ($2 = 0 OR fulfillment_id = $3)
ORDER BY fulfillment_id, item_id
LIMIT $4 OFFSET $5;

-- name: AdminCreateRefund :one
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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (provider_code, provider_refund_id) DO UPDATE SET
    status = EXCLUDED.status,
    reason = EXCLUDED.reason,
    updated_at = now()
RETURNING id;
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
WHERE id = $1
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
WHERE po.workspace_id = $1
  AND ($2 = 0 OR pr.order_id = $3)
  AND ($4 = '' OR pr.provider_code = $5)
  AND ($6 = '' OR CAST(pr.status AS TEXT) = $7)
ORDER BY pr.created_at DESC, pr.id DESC
LIMIT $8 OFFSET $9;

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
        CAST(COALESCE(SUM(CASE WHEN is_closed = FALSE AND available_from <= now() AND available_until > now() THEN 1 ELSE 0 END), 0) AS BIGINT) AS active_products,
        CAST(COALESCE(SUM(CASE WHEN is_visible = TRUE AND is_closed = FALSE AND available_from <= now() AND available_until > now() THEN 1 ELSE 0 END), 0) AS BIGINT) AS visible_products
    FROM payment_product product_rows
    WHERE product_rows.workspace_id = $1
) p
CROSS JOIN (
    SELECT
        COUNT(*) AS orders_total,
        CAST(COALESCE(SUM(CASE WHEN status IN ('draft', 'pending_payment', 'paid') THEN 1 ELSE 0 END), 0) AS BIGINT) AS pending_orders,
        CAST(COALESCE(SUM(CASE WHEN status = 'fulfilled' THEN 1 ELSE 0 END), 0) AS BIGINT) AS fulfilled_orders,
        CAST(COALESCE(SUM(CASE WHEN status = 'refunded' THEN 1 ELSE 0 END), 0) AS BIGINT) AS refunded_orders,
        CAST(COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS BIGINT) AS failed_orders,
        CAST(COALESCE(SUM(CASE WHEN status IN ('canceled', 'expired') THEN 1 ELSE 0 END), 0) AS BIGINT) AS canceled_orders
    FROM payment_order order_rows
    WHERE order_rows.workspace_id = $2
) o
CROSS JOIN (
    SELECT
        CAST(COALESCE(SUM(CASE WHEN event_type = 'purchase' THEN 1 ELSE 0 END), 0) AS BIGINT) AS purchase_count,
        CAST(COALESCE(SUM(CASE WHEN event_type = 'purchase' THEN quantity ELSE 0 END), 0) AS BIGINT) AS purchase_quantity,
        COUNT(DISTINCT CASE WHEN event_type = 'purchase' THEN CONCAT_WS(':', app_id, platform_id, platform_user_id) ELSE NULL END) AS unique_buyers
    FROM payment_stats_event event_rows
    WHERE event_rows.workspace_id = $3
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
        CAST(COALESCE(SUM(CASE WHEN status IN ('draft', 'pending_payment', 'paid') THEN 1 ELSE 0 END), 0) AS BIGINT) AS pending_orders,
        CAST(COALESCE(SUM(CASE WHEN status = 'fulfilled' THEN 1 ELSE 0 END), 0) AS BIGINT) AS fulfilled_orders,
        CAST(COALESCE(SUM(CASE WHEN status = 'refunded' THEN 1 ELSE 0 END), 0) AS BIGINT) AS refunded_orders,
        CAST(COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS BIGINT) AS failed_orders,
        CAST(COALESCE(SUM(CASE WHEN status IN ('canceled', 'expired') THEN 1 ELSE 0 END), 0) AS BIGINT) AS canceled_orders
    FROM payment_order order_rows
    WHERE order_rows.workspace_id = $1 AND order_rows.product_id = $2
    GROUP BY order_rows.workspace_id, order_rows.product_id
) o ON o.workspace_id = p.workspace_id AND o.product_id = p.id
LEFT JOIN (
    SELECT
        event_rows.workspace_id,
        event_rows.product_id,
        CAST(COALESCE(SUM(CASE WHEN event_type = 'purchase' THEN 1 ELSE 0 END), 0) AS BIGINT) AS purchase_count,
        CAST(COALESCE(SUM(CASE WHEN event_type = 'purchase' THEN quantity ELSE 0 END), 0) AS BIGINT) AS purchase_quantity,
        COUNT(DISTINCT CASE WHEN event_type = 'purchase' THEN CONCAT_WS(':', app_id, platform_id, platform_user_id) ELSE NULL END) AS unique_buyers
    FROM payment_stats_event event_rows
    WHERE event_rows.workspace_id = $3 AND event_rows.product_id = $4
    GROUP BY event_rows.workspace_id, event_rows.product_id
) e ON e.workspace_id = p.workspace_id AND e.product_id = p.id
WHERE p.workspace_id = $5 AND p.id = $6
LIMIT 1;

-- name: AdminListPaymentAssetStats :many
SELECT
    asset_code,
    CAST(SUM(CASE WHEN event_type = 'purchase' THEN 1 ELSE 0 END) AS BIGINT) AS purchase_count,
    CAST(SUM(CASE WHEN event_type = 'purchase' THEN quantity ELSE 0 END) AS BIGINT) AS purchase_quantity,
    CAST(SUM(CASE WHEN event_type = 'purchase' THEN amount_minor ELSE 0 END) AS BIGINT) AS gross_amount_minor,
    CAST(SUM(CASE WHEN event_type = 'refund' THEN 1 ELSE 0 END) AS BIGINT) AS refund_count,
    CAST(SUM(CASE WHEN event_type = 'refund' THEN amount_minor ELSE 0 END) AS BIGINT) AS refund_amount_minor
FROM payment_stats_event
WHERE workspace_id = $1
  AND ($2 = '' OR product_id = $3)
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
WHERE workspace_id = $1
  AND product_id = $2
  AND stats_date >= $3
  AND stats_date <= $4
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
WHERE stored_overview.workspace_id = $1
  AND stored_overview.stats_date >= $2
  AND stored_overview.stats_date <= $3
  AND stored_overview.stats_date < CURRENT_DATE
UNION ALL
SELECT
    $4 AS workspace_id,
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
    now() AS updated_at
FROM (
    SELECT
        COUNT(*) AS products_total,
        CAST(COALESCE(SUM(CASE WHEN is_closed = FALSE AND available_from <= now() AND available_until > now() THEN 1 ELSE 0 END), 0) AS BIGINT) AS active_products,
        CAST(COALESCE(SUM(CASE WHEN is_visible = TRUE AND is_closed = FALSE AND available_from <= now() AND available_until > now() THEN 1 ELSE 0 END), 0) AS BIGINT) AS visible_products
    FROM payment_product current_products
    WHERE current_products.workspace_id = $5
) products
CROSS JOIN (
    SELECT
        CAST(COALESCE(MAX(orders_created), 0) AS BIGINT) AS orders_created,
        CAST(COALESCE(MAX(draft_orders), 0) AS BIGINT) AS draft_orders,
        CAST(COALESCE(MAX(pending_payment_orders), 0) AS BIGINT) AS pending_payment_orders,
        CAST(COALESCE(MAX(paid_orders), 0) AS BIGINT) AS paid_orders,
        CAST(COALESCE(MAX(fulfilled_orders), 0) AS BIGINT) AS fulfilled_orders,
        CAST(COALESCE(MAX(canceled_orders), 0) AS BIGINT) AS canceled_orders,
        CAST(COALESCE(MAX(expired_orders), 0) AS BIGINT) AS expired_orders,
        CAST(COALESCE(MAX(refunded_orders), 0) AS BIGINT) AS refunded_orders,
        CAST(COALESCE(MAX(chargebacked_orders), 0) AS BIGINT) AS chargebacked_orders,
        CAST(COALESCE(MAX(failed_orders), 0) AS BIGINT) AS failed_orders,
        CAST(COALESCE(MAX(purchase_count), 0) AS BIGINT) AS purchase_count,
        CAST(COALESCE(MAX(purchase_quantity), 0) AS BIGINT) AS purchase_quantity,
        CAST(COALESCE(MAX(unique_buyers), 0) AS BIGINT) AS unique_buyers,
        CAST(COALESCE(MAX(refund_count), 0) AS BIGINT) AS refund_count
    FROM payment_stats_daily_overview current_overview
    WHERE current_overview.workspace_id = $6
      AND current_overview.stats_date = CURRENT_DATE
) overview
WHERE CURRENT_DATE >= $7
  AND CURRENT_DATE <= $8
ORDER BY stats_date;

-- name: RefreshPaymentDailyStats :exec
INSERT INTO payment_stats_daily (
    workspace_id, product_id, asset_code, stats_date,
    purchase_count, purchase_quantity, unique_buyers,
    gross_amount_minor, refund_count, refund_amount_minor
)
SELECT
    e.workspace_id,
    COALESCE(e.product_id, ''),
    e.asset_code,
    DATE(e.occurred_at),
    SUM(CASE WHEN e.event_type = 'purchase' THEN 1 ELSE 0 END),
    SUM(CASE WHEN e.event_type = 'purchase' THEN e.quantity ELSE 0 END),
    COUNT(DISTINCT CASE WHEN e.event_type = 'purchase' THEN CONCAT_WS(':', e.app_id, e.platform_id, e.platform_user_id) ELSE NULL END),
    SUM(CASE WHEN e.event_type = 'purchase' THEN e.amount_minor ELSE 0 END),
    SUM(CASE WHEN e.event_type = 'refund' THEN 1 ELSE 0 END),
    SUM(CASE WHEN e.event_type = 'refund' THEN e.amount_minor ELSE 0 END)
FROM payment_stats_event e
WHERE e.occurred_at >= $1 AND e.occurred_at < $2
GROUP BY GROUPING SETS (
    (e.workspace_id, e.product_id, e.asset_code, DATE(e.occurred_at)),
    (e.workspace_id, e.asset_code, DATE(e.occurred_at))
)
ON CONFLICT (workspace_id, product_id, asset_code, stats_date) DO UPDATE SET
    purchase_count = EXCLUDED.purchase_count,
    purchase_quantity = EXCLUDED.purchase_quantity,
    unique_buyers = EXCLUDED.unique_buyers,
    gross_amount_minor = EXCLUDED.gross_amount_minor,
    refund_count = EXCLUDED.refund_count,
    refund_amount_minor = EXCLUDED.refund_amount_minor,
    updated_at = now();

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
    WHERE order_dates.occurred_at >= $1 AND order_dates.occurred_at < $2
    UNION
    SELECT payment_dates.workspace_id, DATE(payment_dates.occurred_at) AS stats_date
    FROM payment_stats_event payment_dates
    WHERE payment_dates.occurred_at >= $3 AND payment_dates.occurred_at < $4
) dates
LEFT JOIN (
    SELECT
        workspace_id,
        COUNT(*) AS products_total,
        SUM(CASE WHEN is_closed = FALSE AND available_from <= now() AND available_until > now() THEN 1 ELSE 0 END) AS active_products,
        SUM(CASE WHEN is_visible = TRUE AND is_closed = FALSE AND available_from <= now() AND available_until > now() THEN 1 ELSE 0 END) AS visible_products
    FROM payment_product
    GROUP BY workspace_id
) products ON products.workspace_id = dates.workspace_id
LEFT JOIN (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        SUM(CASE WHEN event_type = 'created' THEN 1 ELSE 0 END) AS orders_created,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'draft' THEN 1 ELSE 0 END) AS draft_orders,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'pending_payment' THEN 1 ELSE 0 END) AS pending_payment_orders,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'paid' THEN 1 ELSE 0 END) AS paid_orders,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'fulfilled' THEN 1 ELSE 0 END) AS fulfilled_orders,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'canceled' THEN 1 ELSE 0 END) AS canceled_orders,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'expired' THEN 1 ELSE 0 END) AS expired_orders,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'refunded' THEN 1 ELSE 0 END) AS refunded_orders,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'chargebacked' THEN 1 ELSE 0 END) AS chargebacked_orders,
        SUM(CASE WHEN event_type = 'status' AND order_status = 'failed' THEN 1 ELSE 0 END) AS failed_orders
    FROM payment_stats_order_event overview_orders
    WHERE overview_orders.occurred_at >= $5 AND overview_orders.occurred_at < $6
    GROUP BY overview_orders.workspace_id, DATE(overview_orders.occurred_at)
) orders
    ON orders.workspace_id = dates.workspace_id
   AND orders.stats_date = dates.stats_date
LEFT JOIN (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        SUM(CASE WHEN event_type = 'purchase' THEN 1 ELSE 0 END) AS purchase_count,
        SUM(CASE WHEN event_type = 'purchase' THEN quantity ELSE 0 END) AS purchase_quantity,
        COUNT(DISTINCT CASE WHEN event_type = 'purchase' THEN CONCAT_WS(':', app_id, platform_id, platform_user_id) ELSE NULL END) AS unique_buyers,
        SUM(CASE WHEN event_type = 'refund' THEN 1 ELSE 0 END) AS refund_count
    FROM payment_stats_event overview_payments
    WHERE overview_payments.occurred_at >= $7 AND overview_payments.occurred_at < $8
    GROUP BY overview_payments.workspace_id, DATE(overview_payments.occurred_at)
) payments
    ON payments.workspace_id = dates.workspace_id
   AND payments.stats_date = dates.stats_date
WHERE TRUE
ON CONFLICT (workspace_id, stats_date) DO UPDATE SET
    orders_created = EXCLUDED.orders_created,
    draft_orders = EXCLUDED.draft_orders,
    pending_payment_orders = EXCLUDED.pending_payment_orders,
    paid_orders = EXCLUDED.paid_orders,
    fulfilled_orders = EXCLUDED.fulfilled_orders,
    canceled_orders = EXCLUDED.canceled_orders,
    expired_orders = EXCLUDED.expired_orders,
    refunded_orders = EXCLUDED.refunded_orders,
    chargebacked_orders = EXCLUDED.chargebacked_orders,
    failed_orders = EXCLUDED.failed_orders,
    purchase_count = EXCLUDED.purchase_count,
    purchase_quantity = EXCLUDED.purchase_quantity,
    unique_buyers = EXCLUDED.unique_buyers,
    refund_count = EXCLUDED.refund_count,
    updated_at = now();

-- name: AdminUpdateRefundStatus :execrows
UPDATE payment_refund
SET status = $1,
    reason = $2,
    updated_at = now()
WHERE id = $3;

-- name: AdminSetRefundProviderID :execrows
UPDATE payment_refund
SET provider_refund_id = $1,
    updated_at = now()
WHERE id = $2
  AND (provider_refund_id IS NULL OR provider_refund_id = $3);

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
WHERE workspace_id = $1
  AND ($2 = '' OR provider_code = $3)
  AND ($4 = '' OR network = $5)
ORDER BY provider_code, network, source_key
LIMIT $6 OFFSET $7;

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
WHERE workspace_id = $1
  AND ($2 = '' OR provider_code = $3)
  AND ($4 = '' OR network = $5)
  AND ($6 = '' OR source_key = $7)
  AND ($8 = '' OR CAST(status AS TEXT) = $9)
ORDER BY sequence_number DESC, id DESC
LIMIT $10 OFFSET $11;

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
WHERE workspace_id = $1
  AND id = $2
LIMIT 1;

-- name: AdminUpdateProviderTransactionStatus :execrows
UPDATE payment_provider_transaction
SET status = $1,
    error = $2
WHERE workspace_id = $3
  AND id = $4;

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
WHERE provider_code = $1
  AND provider_payment_id = $2
LIMIT 1
FOR UPDATE;

-- name: MarkOrderRefunded :execrows
UPDATE payment_order
SET status = 'refunded',
    updated_at = now()
WHERE id = $1
  AND status IN ('paid', 'fulfilled', 'refunded');

-- name: MarkFulfillmentRevokedForOrder :execrows
UPDATE payment_fulfillment
SET status = 'revoked',
    revoked_at = COALESCE(revoked_at, now()),
    updated_at = now()
WHERE order_id = $1
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
WHERE order_id = $1
LIMIT 1;

-- name: DecrementProductLimitCountersForRefund :execrows
UPDATE payment_product_limit_counter plc
SET paid_count = GREATEST(plc.paid_count - po.quantity, 0),
    updated_at = now()
FROM payment_order po
WHERE po.workspace_id = plc.workspace_id
  AND po.platform_id = plc.platform_id
  AND po.product_id = plc.product_id
  AND po.id = $1
  AND po.paid_at IS NOT NULL
  AND po.paid_at >= plc.window_start
  AND po.paid_at < plc.window_end
  AND (
      (plc.counter_scope = 'global' AND plc.platform_user_id = '')
      OR
      (plc.counter_scope = 'user' AND plc.platform_user_id = po.platform_user_id)
  );

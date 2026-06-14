CREATE TABLE IF NOT EXISTS payment_provider (
    code VARCHAR(32) NOT NULL PRIMARY KEY,
    title VARCHAR(128) NOT NULL,
    provider_kind ENUM('platform_internal', 'fiat_gateway', 'crypto_chain') NOT NULL,
    supports_create TINYINT(1) NOT NULL DEFAULT 0,
    supports_redirect TINYINT(1) NOT NULL DEFAULT 0,
    supports_webhook TINYINT(1) NOT NULL DEFAULT 0,
    supports_refund TINYINT(1) NOT NULL DEFAULT 0,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payment_asset (
    code VARCHAR(32) NOT NULL PRIMARY KEY,
    title VARCHAR(128) NOT NULL,
    asset_kind ENUM('fiat', 'platform_currency', 'crypto_native', 'crypto_jetton') NOT NULL,
    scale SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    chain VARCHAR(32) NULL,
    network VARCHAR(32) NULL,
    contract_address VARCHAR(128) NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_asset_chain_contract_uq (chain, network, contract_address)
);

CREATE TABLE IF NOT EXISTS payment_provider_asset (
    provider_code VARCHAR(32) NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    min_amount_minor BIGINT UNSIGNED NULL,
    max_amount_minor BIGINT UNSIGNED NULL,
    merchant_account VARCHAR(128) NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (provider_code, asset_code),
    KEY payment_provider_asset_asset_active_idx (asset_code, is_active, provider_code),
    CONSTRAINT payment_provider_asset_provider_fk
        FOREIGN KEY (provider_code) REFERENCES payment_provider (code),
    CONSTRAINT payment_provider_asset_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code)
);

CREATE TABLE IF NOT EXISTS payment_asset_rate (
    asset_code VARCHAR(32) NOT NULL,
    reference_asset_code VARCHAR(32) NOT NULL,
    reference_per_asset_minor BIGINT UNSIGNED NOT NULL,
    source VARCHAR(64) NOT NULL,
    observed_at DATETIME NOT NULL,
    auto_update_enabled TINYINT(1) NOT NULL DEFAULT 0,
    auto_update_source VARCHAR(32) NULL,
    source_chain_id VARCHAR(32) NULL,
    source_token_address VARCHAR(128) NULL,
    last_attempt_at DATETIME NULL,
    last_error TEXT NULL,
    lease_owner VARCHAR(64) NULL,
    lease_until DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (asset_code, reference_asset_code),
    KEY payment_asset_rate_reference_idx (reference_asset_code, asset_code),
    KEY payment_asset_rate_auto_lease_idx (auto_update_enabled, lease_until),
    CONSTRAINT payment_asset_rate_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_asset_rate_reference_asset_fk
        FOREIGN KEY (reference_asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_asset_rate_positive_chk CHECK (reference_per_asset_minor > 0)
);

CREATE TABLE IF NOT EXISTS payment_product_group (
    workspace_id CHAR(36) NOT NULL,
    code VARCHAR(64) NOT NULL,
    title_key VARCHAR(255) NULL,
    description_key VARCHAR(255) NULL,
    position INT NOT NULL DEFAULT 0,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, code)
);

CREATE TABLE IF NOT EXISTS payment_localization (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id CHAR(36) NOT NULL,
    locale VARCHAR(16) NOT NULL,
    localization_key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_localization_locale_key_uq (workspace_id, locale, localization_key)
);

CREATE TABLE IF NOT EXISTS payment_product (
    workspace_id CHAR(36) NOT NULL,
    id VARCHAR(64) NOT NULL,
    group_code VARCHAR(64) NULL,
    title_key VARCHAR(255) NOT NULL,
    description_key VARCHAR(255) NULL,
    image_url VARCHAR(512) NULL,
    link_url VARCHAR(512) NULL,
    size_label VARCHAR(64) NULL,
    period_seconds BIGINT NULL,
    trial_duration_seconds BIGINT NULL,
    quantity_mode ENUM('fixed', 'flexible') NOT NULL DEFAULT 'fixed',
    position INT NOT NULL DEFAULT 0,
    global_limit INT NOT NULL DEFAULT 0,
    global_interval ENUM('SECOND', 'MINUTE', 'HOUR', 'DAY', 'WEEK', 'MONTH', 'ONCE', 'UNLIMITED') NOT NULL DEFAULT 'UNLIMITED',
    global_interval_count INT NOT NULL DEFAULT 0,
    user_limit INT NOT NULL DEFAULT 0,
    user_interval ENUM('SECOND', 'MINUTE', 'HOUR', 'DAY', 'WEEK', 'MONTH', 'ONCE', 'UNLIMITED') NOT NULL DEFAULT 'UNLIMITED',
    user_interval_count INT NOT NULL DEFAULT 0,
    available_from DATETIME NOT NULL DEFAULT '2024-01-01 00:00:00',
    available_until DATETIME NOT NULL DEFAULT '2124-01-01 00:00:00',
    is_visible TINYINT(1) NOT NULL DEFAULT 1,
    is_closed TINYINT(1) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, id),
    KEY payment_product_group_idx (workspace_id, group_code),
    KEY payment_product_workspace_window_idx (workspace_id, is_visible, is_closed, available_from, available_until, position),
    KEY payment_product_window_idx (available_from, available_until, position),
    CONSTRAINT payment_product_group_fk
        FOREIGN KEY (workspace_id, group_code) REFERENCES payment_product_group (workspace_id, code)
);

CREATE TABLE IF NOT EXISTS payment_item (
    workspace_id CHAR(36) NOT NULL,
    id VARCHAR(64) NOT NULL,
    item_type VARCHAR(64) NULL,
    title_key VARCHAR(255) NOT NULL,
    description_key VARCHAR(255) NULL,
    rarity VARCHAR(64) NOT NULL DEFAULT 'common',
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, id),
    KEY payment_item_position_idx (position)
);

CREATE TABLE IF NOT EXISTS payment_product_item (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id CHAR(36) NOT NULL,
    product_id VARCHAR(64) NOT NULL,
    item_id VARCHAR(64) NOT NULL,
    reward_type ENUM('quantity','duration') NOT NULL DEFAULT 'quantity',
    quantity BIGINT NOT NULL DEFAULT 0,
    duration_unit ENUM('second','minute','hour','day','week','month','year') NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_product_item_uq (workspace_id, product_id, item_id),
    CONSTRAINT payment_product_item_product_fk
        FOREIGN KEY (workspace_id, product_id) REFERENCES payment_product (workspace_id, id)
            ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT payment_product_item_item_fk
        FOREIGN KEY (workspace_id, item_id) REFERENCES payment_item (workspace_id, id)
            ON UPDATE CASCADE,
    CONSTRAINT payment_product_item_quantity_chk CHECK (quantity > 0),
    CONSTRAINT payment_product_item_reward_chk CHECK (
        (reward_type = 'quantity' AND duration_unit IS NULL) OR
        (reward_type = 'duration' AND duration_unit IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS payment_price (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id CHAR(36) NOT NULL,
    product_id VARCHAR(64) NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    list_amount_minor BIGINT UNSIGNED NOT NULL,
    discount_amount_minor BIGINT UNSIGNED NOT NULL DEFAULT 0,
    pricing_mode ENUM('fixed', 'dynamic') NOT NULL DEFAULT 'fixed',
    reference_asset_code VARCHAR(32) NULL,
    reference_list_amount_minor BIGINT UNSIGNED NULL,
    reference_discount_amount_minor BIGINT UNSIGNED NULL,
    coefficient DECIMAL(24,12) NULL,
    is_promotion TINYINT(1) NOT NULL DEFAULT 0,
    starts_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ends_at DATETIME NOT NULL DEFAULT '2124-01-01 00:00:00',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_price_window_uq (workspace_id, product_id, asset_code, is_promotion, starts_at, ends_at),
    KEY payment_price_current_idx (workspace_id, product_id, asset_code, starts_at, ends_at, is_promotion, id),
    KEY payment_price_dynamic_idx (workspace_id, asset_code, reference_asset_code, pricing_mode),
    CONSTRAINT payment_price_product_fk
        FOREIGN KEY (workspace_id, product_id) REFERENCES payment_product (workspace_id, id)
            ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT payment_price_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_price_reference_asset_fk
        FOREIGN KEY (reference_asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_price_reference_discount_chk CHECK (
        reference_discount_amount_minor IS NULL
        OR reference_list_amount_minor IS NULL
        OR reference_discount_amount_minor <= reference_list_amount_minor
    ),
    CONSTRAINT payment_price_dynamic_chk CHECK (
        (pricing_mode = 'fixed'
            AND reference_asset_code IS NULL
            AND reference_list_amount_minor IS NULL
            AND reference_discount_amount_minor IS NULL
            AND coefficient IS NULL)
        OR
        (pricing_mode = 'dynamic'
            AND reference_asset_code IS NOT NULL
            AND reference_list_amount_minor IS NOT NULL
            AND reference_discount_amount_minor IS NOT NULL
            AND coefficient IS NOT NULL
            AND coefficient > 0)
    )
);

CREATE TABLE IF NOT EXISTS payment_product_cache (
    workspace_id CHAR(36) NOT NULL,
    product_id VARCHAR(64) NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    locale VARCHAR(16) NOT NULL,
    price_id BIGINT UNSIGNED NOT NULL,
    item_id VARCHAR(64) NOT NULL DEFAULT '',
    link_url VARCHAR(512) NULL,
    size_label VARCHAR(64) NULL,
    group_code VARCHAR(64) NULL,
    product_title TEXT NOT NULL,
    product_description TEXT NOT NULL,
    image_url VARCHAR(512) NULL,
    period_seconds BIGINT NULL,
    trial_duration_seconds BIGINT NULL,
    quantity_mode ENUM('fixed', 'flexible') NOT NULL DEFAULT 'fixed',
    product_position INT NOT NULL DEFAULT 0,
    global_limit INT NOT NULL DEFAULT 0,
    global_interval ENUM('SECOND', 'MINUTE', 'HOUR', 'DAY', 'WEEK', 'MONTH', 'ONCE', 'UNLIMITED') NOT NULL DEFAULT 'UNLIMITED',
    global_interval_count INT NOT NULL DEFAULT 0,
    user_limit INT NOT NULL DEFAULT 0,
    user_interval ENUM('SECOND', 'MINUTE', 'HOUR', 'DAY', 'WEEK', 'MONTH', 'ONCE', 'UNLIMITED') NOT NULL DEFAULT 'UNLIMITED',
    user_interval_count INT NOT NULL DEFAULT 0,
    is_visible TINYINT(1) NOT NULL DEFAULT 1,
    is_closed TINYINT(1) NOT NULL DEFAULT 0,
    available_from DATETIME NOT NULL,
    available_until DATETIME NOT NULL,
    list_amount_minor BIGINT UNSIGNED NOT NULL,
    discount_amount_minor BIGINT UNSIGNED NOT NULL DEFAULT 0,
    is_promotion TINYINT(1) NOT NULL DEFAULT 0,
    price_starts_at DATETIME NOT NULL,
    price_ends_at DATETIME NOT NULL,
    item_quantity BIGINT NOT NULL DEFAULT 0,
    reward_type ENUM('quantity','duration') NOT NULL DEFAULT 'quantity',
    duration_unit ENUM('second','minute','hour','day','week','month','year') NULL,
    item_type VARCHAR(64) NULL,
    item_title TEXT NOT NULL,
    item_description TEXT NOT NULL,
    item_rarity VARCHAR(64) NULL,
    item_position INT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, product_id, asset_code, locale, price_id, item_id),
    KEY payment_product_cache_get_idx (
        workspace_id,
        product_id,
        asset_code,
        locale,
        is_visible,
        is_closed,
        available_from,
        available_until,
        price_starts_at,
        price_ends_at
    ),
    KEY payment_product_cache_price_idx (
        workspace_id,
        product_id,
        asset_code,
        locale,
        is_promotion,
        price_starts_at,
        price_id
    ),
    CONSTRAINT payment_product_cache_product_fk
        FOREIGN KEY (workspace_id, product_id) REFERENCES payment_product (workspace_id, id)
            ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT payment_product_cache_price_fk
        FOREIGN KEY (price_id) REFERENCES payment_price (id)
            ON DELETE CASCADE,
    CONSTRAINT payment_product_cache_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code)
);

CREATE TABLE IF NOT EXISTS payment_purchase_key (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id CHAR(36) NOT NULL,
    key_hash CHAR(64) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL,
    internal_user_id BIGINT NULL,
    product_id VARCHAR(64) NOT NULL,
    status ENUM('active', 'used', 'canceled', 'expired') NOT NULL DEFAULT 'active',
    max_uses INT NOT NULL DEFAULT 1,
    used_count INT NOT NULL DEFAULT 0,
    expires_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_purchase_key_hash_uq (key_hash),
    KEY payment_purchase_key_product_status_idx (workspace_id, product_id, status),
    KEY payment_purchase_key_target_idx (app_id, platform_id, platform_user_id),
    CONSTRAINT payment_purchase_key_product_fk
        FOREIGN KEY (workspace_id, product_id) REFERENCES payment_product (workspace_id, id)
            ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT payment_purchase_key_uses_chk
        CHECK (max_uses > 0 AND used_count >= 0 AND used_count <= max_uses)
);

CREATE TABLE IF NOT EXISTS payment_order (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    public_id CHAR(36) NOT NULL,
    workspace_id CHAR(36) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL,
    internal_user_id BIGINT NULL,
    payer_platform_id BIGINT NULL,
    payer_platform_user_id VARCHAR(128) NULL,
    payer_internal_user_id BIGINT NULL,
    purchase_key_id BIGINT UNSIGNED NULL,
    product_id VARCHAR(64) NOT NULL,
    quantity BIGINT UNSIGNED NOT NULL DEFAULT 1,
    price_id BIGINT UNSIGNED NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    locale VARCHAR(16) NOT NULL DEFAULT 'ru',
    list_amount_minor BIGINT UNSIGNED NOT NULL,
    discount_amount_minor BIGINT UNSIGNED NOT NULL DEFAULT 0,
    payable_amount_minor BIGINT UNSIGNED NOT NULL,
    status ENUM('draft', 'pending_payment', 'paid', 'fulfilled', 'canceled', 'expired', 'refunded', 'chargebacked', 'failed') NOT NULL DEFAULT 'draft',
    reserved_until DATETIME NULL,
    paid_at DATETIME NULL,
    fulfilled_at DATETIME NULL,
    canceled_at DATETIME NULL,
    expires_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_order_public_id_uq (public_id),
    KEY payment_order_user_product_status_idx (workspace_id, platform_id, platform_user_id, product_id, status),
    KEY payment_order_payer_idx (app_id, payer_platform_id, payer_platform_user_id),
    KEY payment_order_purchase_key_idx (purchase_key_id),
    KEY payment_order_status_created_idx (status, created_at),
    CONSTRAINT payment_order_product_fk
        FOREIGN KEY (workspace_id, product_id) REFERENCES payment_product (workspace_id, id),
    CONSTRAINT payment_order_price_fk
        FOREIGN KEY (price_id) REFERENCES payment_price (id),
    CONSTRAINT payment_order_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_order_purchase_key_fk
        FOREIGN KEY (purchase_key_id) REFERENCES payment_purchase_key (id),
    CONSTRAINT payment_order_payable_chk
        CHECK (
            discount_amount_minor <= list_amount_minor
            AND payable_amount_minor = list_amount_minor - discount_amount_minor
        )
);

CREATE TABLE IF NOT EXISTS payment_paid_order_index (
    order_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    workspace_id CHAR(36) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL,
    internal_user_id BIGINT NULL,
    payer_platform_id BIGINT NULL,
    payer_platform_user_id VARCHAR(128) NULL,
    payer_internal_user_id BIGINT NULL,
    purchase_key_id BIGINT UNSIGNED NULL,
    product_id VARCHAR(64) NOT NULL,
    quantity BIGINT UNSIGNED NOT NULL DEFAULT 1,
    price_id BIGINT UNSIGNED NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    locale VARCHAR(16) NOT NULL,
    list_amount_minor BIGINT UNSIGNED NOT NULL,
    discount_amount_minor BIGINT UNSIGNED NOT NULL,
    payable_amount_minor BIGINT UNSIGNED NOT NULL,
    status ENUM('paid', 'fulfilled') NOT NULL DEFAULT 'paid',
    paid_at DATETIME NOT NULL,
    fulfilled_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY payment_paid_order_global_window_idx (workspace_id, platform_id, product_id, paid_at),
    KEY payment_paid_order_user_window_idx (workspace_id, platform_id, platform_user_id, product_id, paid_at),
    KEY payment_paid_order_purchase_key_idx (purchase_key_id),
    CONSTRAINT payment_paid_order_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id)
            ON DELETE CASCADE,
    CONSTRAINT payment_paid_order_product_fk
        FOREIGN KEY (workspace_id, product_id) REFERENCES payment_product (workspace_id, id),
    CONSTRAINT payment_paid_order_price_fk
        FOREIGN KEY (price_id) REFERENCES payment_price (id),
    CONSTRAINT payment_paid_order_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_paid_order_purchase_key_fk
        FOREIGN KEY (purchase_key_id) REFERENCES payment_purchase_key (id)
);

CREATE TABLE IF NOT EXISTS payment_order_item (
    order_id BIGINT UNSIGNED NOT NULL,
    workspace_id CHAR(36) NOT NULL,
    item_id VARCHAR(64) NOT NULL,
    reward_type ENUM('quantity','duration') NOT NULL DEFAULT 'quantity',
    quantity BIGINT NOT NULL,
    duration_unit ENUM('second','minute','hour','day','week','month','year') NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (order_id, workspace_id, item_id),
    KEY payment_order_item_item_idx (workspace_id, item_id),
    CONSTRAINT payment_order_item_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id)
            ON DELETE CASCADE,
    CONSTRAINT payment_order_item_item_fk
        FOREIGN KEY (workspace_id, item_id) REFERENCES payment_item (workspace_id, id),
    CONSTRAINT payment_order_item_quantity_chk CHECK (quantity > 0),
    CONSTRAINT payment_order_item_reward_chk CHECK (
        (reward_type = 'quantity' AND duration_unit IS NULL) OR
        (reward_type = 'duration' AND duration_unit IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS payment_product_limit_counter (
    workspace_id CHAR(36) NOT NULL,
    platform_id BIGINT NOT NULL,
    product_id VARCHAR(64) NOT NULL,
    counter_scope ENUM('global', 'user') NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL DEFAULT '',
    window_start DATETIME NOT NULL,
    window_end DATETIME NOT NULL,
    paid_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (
        workspace_id,
        platform_id,
        product_id,
        counter_scope,
        platform_user_id,
        window_start,
        window_end
    ),
    KEY payment_product_limit_counter_window_idx (window_end, workspace_id, platform_id, product_id),
    CONSTRAINT payment_product_limit_counter_product_fk
        FOREIGN KEY (workspace_id, product_id) REFERENCES payment_product (workspace_id, id)
            ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS payment_attempt (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    provider_code VARCHAR(32) NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    amount_minor BIGINT UNSIGNED NOT NULL,
    status ENUM('created', 'pending', 'requires_action', 'waiting_capture', 'succeeded', 'canceled', 'expired', 'refunded', 'chargebacked', 'failed') NOT NULL DEFAULT 'created',
    provider_payment_id VARCHAR(128) NULL,
    provider_invoice_id VARCHAR(128) NULL,
    provider_charge_id VARCHAR(128) NULL,
    provider_subscription_id VARCHAR(128) NULL,
    idempotency_key VARCHAR(128) NULL,
    confirmation_url TEXT NULL,
    return_url TEXT NULL,
    expires_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_attempt_idempotency_uq (idempotency_key),
    UNIQUE KEY payment_attempt_provider_payment_uq (provider_code, provider_payment_id),
    UNIQUE KEY payment_attempt_provider_charge_uq (provider_code, provider_charge_id),
    KEY payment_attempt_order_idx (order_id),
    KEY payment_attempt_provider_status_idx (provider_code, status, created_at),
    CONSTRAINT payment_attempt_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id)
            ON DELETE CASCADE,
    CONSTRAINT payment_attempt_provider_fk
        FOREIGN KEY (provider_code) REFERENCES payment_provider (code),
    CONSTRAINT payment_attempt_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code)
);

CREATE TABLE IF NOT EXISTS payment_event (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    provider_code VARCHAR(32) NOT NULL,
    attempt_id BIGINT UNSIGNED NULL,
    order_id BIGINT UNSIGNED NULL,
    provider_event_id VARCHAR(128) NULL,
    provider_payment_id VARCHAR(128) NULL,
    event_type VARCHAR(128) NOT NULL,
    event_status VARCHAR(64) NULL,
    payload_hash CHAR(64) NOT NULL,
    signature_valid TINYINT(1) NULL,
    processing_status ENUM('new', 'processed', 'ignored', 'failed') NOT NULL DEFAULT 'new',
    processing_error TEXT NULL,
    received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at DATETIME NULL,
    UNIQUE KEY payment_event_provider_event_uq (provider_code, provider_event_id),
    UNIQUE KEY payment_event_payload_hash_uq (provider_code, payload_hash),
    KEY payment_event_attempt_idx (attempt_id),
    KEY payment_event_order_idx (order_id),
    KEY payment_event_processing_idx (processing_status, received_at),
    CONSTRAINT payment_event_provider_fk
        FOREIGN KEY (provider_code) REFERENCES payment_provider (code),
    CONSTRAINT payment_event_attempt_fk
        FOREIGN KEY (attempt_id) REFERENCES payment_attempt (id),
    CONSTRAINT payment_event_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id)
);

CREATE TABLE IF NOT EXISTS payment_subscription (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id CHAR(36) NOT NULL,
    provider_code VARCHAR(32) NOT NULL,
    provider_subscription_id VARCHAR(128) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL,
    internal_user_id BIGINT NULL,
    product_id VARCHAR(64) NOT NULL,
    order_id BIGINT UNSIGNED NULL,
    attempt_id BIGINT UNSIGNED NULL,
    status ENUM('active', 'canceled', 'refunded', 'expired') NOT NULL DEFAULT 'active',
    cancel_reason VARCHAR(255) NULL,
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_subscription_provider_uq (provider_code, provider_subscription_id),
    KEY payment_subscription_user_idx (workspace_id, platform_id, platform_user_id, product_id, status),
    KEY payment_subscription_active_idx (workspace_id, platform_id, platform_user_id, status, ended_at),
    KEY payment_subscription_active_product_idx (workspace_id, platform_id, platform_user_id, product_id, status, ended_at),
    KEY payment_subscription_active_provider_idx (workspace_id, platform_id, platform_user_id, provider_code, status, ended_at),
    KEY payment_subscription_active_product_provider_idx (workspace_id, platform_id, platform_user_id, product_id, provider_code, status, ended_at),
    KEY payment_subscription_order_idx (order_id),
    CONSTRAINT payment_subscription_provider_fk
        FOREIGN KEY (provider_code) REFERENCES payment_provider (code),
    CONSTRAINT payment_subscription_product_fk
        FOREIGN KEY (workspace_id, product_id) REFERENCES payment_product (workspace_id, id),
    CONSTRAINT payment_subscription_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id),
    CONSTRAINT payment_subscription_attempt_fk
        FOREIGN KEY (attempt_id) REFERENCES payment_attempt (id)
);

CREATE TABLE IF NOT EXISTS payment_fulfillment (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    attempt_id BIGINT UNSIGNED NOT NULL,
    internal_user_id BIGINT NULL,
    status ENUM('pending', 'succeeded', 'revoked', 'failed') NOT NULL DEFAULT 'pending',
    error TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    fulfilled_at DATETIME NULL,
    revoked_at DATETIME NULL,
    UNIQUE KEY payment_fulfillment_order_uq (order_id),
    KEY payment_fulfillment_user_status_idx (internal_user_id, status),
    CONSTRAINT payment_fulfillment_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id),
    CONSTRAINT payment_fulfillment_attempt_fk
        FOREIGN KEY (attempt_id) REFERENCES payment_attempt (id)
);

CREATE TABLE IF NOT EXISTS payment_fulfillment_item (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    fulfillment_id BIGINT UNSIGNED NOT NULL,
    workspace_id CHAR(36) NOT NULL,
    item_id VARCHAR(64) NOT NULL,
    reward_type ENUM('quantity','duration') NOT NULL DEFAULT 'quantity',
    quantity BIGINT NOT NULL,
    duration_unit ENUM('second','minute','hour','day','week','month','year') NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY payment_fulfillment_item_uq (fulfillment_id, workspace_id, item_id),
    CONSTRAINT payment_fulfillment_item_fulfillment_fk
        FOREIGN KEY (fulfillment_id) REFERENCES payment_fulfillment (id)
            ON DELETE CASCADE,
    CONSTRAINT payment_fulfillment_item_item_fk
        FOREIGN KEY (workspace_id, item_id) REFERENCES payment_item (workspace_id, id),
    CONSTRAINT payment_fulfillment_item_quantity_chk CHECK (quantity > 0),
    CONSTRAINT payment_fulfillment_item_reward_chk CHECK (
        (reward_type = 'quantity' AND duration_unit IS NULL) OR
        (reward_type = 'duration' AND duration_unit IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS payment_refund (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    attempt_id BIGINT UNSIGNED NOT NULL,
    provider_code VARCHAR(32) NOT NULL,
    provider_refund_id VARCHAR(128) NULL,
    amount_minor BIGINT UNSIGNED NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    status ENUM('created', 'pending', 'succeeded', 'canceled', 'failed') NOT NULL DEFAULT 'created',
    reason VARCHAR(255) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY payment_refund_provider_uq (provider_code, provider_refund_id),
    KEY payment_refund_order_idx (order_id),
    CONSTRAINT payment_refund_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id),
    CONSTRAINT payment_refund_attempt_fk
        FOREIGN KEY (attempt_id) REFERENCES payment_attempt (id),
    CONSTRAINT payment_refund_provider_fk
        FOREIGN KEY (provider_code) REFERENCES payment_provider (code),
    CONSTRAINT payment_refund_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code)
);

CREATE TABLE IF NOT EXISTS payment_stats_event (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_type ENUM('purchase', 'refund') NOT NULL,
    source_id BIGINT UNSIGNED NOT NULL,
    workspace_id CHAR(36) NOT NULL,
    product_id VARCHAR(64) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL,
    quantity BIGINT UNSIGNED NOT NULL DEFAULT 0,
    asset_code VARCHAR(32) NOT NULL,
    amount_minor BIGINT UNSIGNED NOT NULL,
    occurred_at DATETIME NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY payment_stats_event_source_uq (event_type, source_id),
    KEY payment_stats_event_workspace_idx (
        workspace_id, occurred_at, event_type, asset_code
    ),
    KEY payment_stats_event_product_idx (
        workspace_id, product_id, occurred_at, event_type, asset_code
    ),
    KEY payment_stats_event_user_idx (
        workspace_id, platform_id, platform_user_id, occurred_at
    )
);

CREATE TABLE IF NOT EXISTS payment_stats_order_event (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_id BIGINT UNSIGNED NOT NULL,
    workspace_id CHAR(36) NOT NULL,
    product_id VARCHAR(64) NOT NULL,
    event_type ENUM('created', 'status') NOT NULL,
    order_status ENUM(
        'draft',
        'pending_payment',
        'paid',
        'fulfilled',
        'canceled',
        'expired',
        'refunded',
        'chargebacked',
        'failed'
    ) NOT NULL,
    occurred_at DATETIME NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY payment_stats_order_event_uq (order_id, event_type, order_status),
    KEY payment_stats_order_event_workspace_idx (
        workspace_id, occurred_at, order_status
    ),
    KEY payment_stats_order_event_product_idx (
        workspace_id, product_id, occurred_at, order_status
    ),
    CONSTRAINT payment_stats_order_event_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id)
            ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS payment_stats_daily (
    workspace_id CHAR(36) NOT NULL,
    product_id VARCHAR(64) NOT NULL DEFAULT '',
    asset_code VARCHAR(32) NOT NULL,
    stats_date DATE NOT NULL,
    purchase_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    purchase_quantity BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_buyers BIGINT UNSIGNED NOT NULL DEFAULT 0,
    gross_amount_minor BIGINT UNSIGNED NOT NULL DEFAULT 0,
    refund_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    refund_amount_minor BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, product_id, asset_code, stats_date),
    KEY payment_stats_daily_date_idx (workspace_id, stats_date, product_id)
);

CREATE TABLE IF NOT EXISTS payment_stats_daily_overview (
    workspace_id CHAR(36) NOT NULL,
    stats_date DATE NOT NULL,
    products_total BIGINT UNSIGNED NOT NULL DEFAULT 0,
    active_products BIGINT UNSIGNED NOT NULL DEFAULT 0,
    visible_products BIGINT UNSIGNED NOT NULL DEFAULT 0,
    orders_created BIGINT UNSIGNED NOT NULL DEFAULT 0,
    draft_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    pending_payment_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    paid_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    fulfilled_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    canceled_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    expired_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    refunded_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    chargebacked_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    failed_orders BIGINT UNSIGNED NOT NULL DEFAULT 0,
    purchase_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    purchase_quantity BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_buyers BIGINT UNSIGNED NOT NULL DEFAULT 0,
    refund_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, stats_date),
    KEY payment_stats_daily_overview_date_idx (stats_date, workspace_id)
);

CREATE TABLE IF NOT EXISTS payment_stats_daily_buyer (
    workspace_id CHAR(36) NOT NULL,
    stats_date DATE NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (
        workspace_id,
        stats_date,
        app_id,
        platform_id,
        platform_user_id
    ),
    KEY payment_stats_daily_buyer_date_idx (stats_date, workspace_id)
);

CREATE TABLE IF NOT EXISTS payment_provider_cursor (
    workspace_id CHAR(36) NOT NULL,
    provider_code VARCHAR(32) NOT NULL,
    network VARCHAR(32) NOT NULL,
    source_key VARCHAR(255) NOT NULL,
    cursor_value VARCHAR(255) NOT NULL DEFAULT '',
    cursor_sequence BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, provider_code, network, source_key),
    KEY payment_provider_cursor_provider_idx (
        provider_code, network, updated_at
    ),
    CONSTRAINT payment_provider_cursor_provider_fk
        FOREIGN KEY (provider_code) REFERENCES payment_provider (code)
);

CREATE TABLE IF NOT EXISTS payment_provider_transaction (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id CHAR(36) NOT NULL,
    provider_code VARCHAR(32) NOT NULL,
    network VARCHAR(32) NOT NULL,
    source_key VARCHAR(255) NOT NULL,
    asset_code VARCHAR(32) NOT NULL,
    external_transaction_id VARCHAR(255) NOT NULL,
    sequence_number BIGINT UNSIGNED NOT NULL DEFAULT 0,
    source_address VARCHAR(255) NOT NULL DEFAULT '',
    destination_address VARCHAR(255) NOT NULL DEFAULT '',
    amount_minor BIGINT UNSIGNED NOT NULL,
    payment_reference VARCHAR(255) NOT NULL DEFAULT '',
    sender_reference VARCHAR(255) NULL,
    order_id BIGINT UNSIGNED NULL,
    attempt_id BIGINT UNSIGNED NULL,
    status ENUM('new', 'matched', 'ignored', 'failed') NOT NULL DEFAULT 'new',
    error TEXT NULL,
    occurred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY payment_provider_transaction_external_uq (
        workspace_id, provider_code, network, source_key, external_transaction_id
    ),
    KEY payment_provider_transaction_sequence_idx (
        workspace_id, provider_code, network, source_key, sequence_number
    ),
    KEY payment_provider_transaction_reference_idx (
        workspace_id, provider_code, asset_code, payment_reference
    ),
    KEY payment_provider_transaction_order_idx (order_id),
    KEY payment_provider_transaction_attempt_idx (attempt_id),
    CONSTRAINT payment_provider_transaction_provider_fk
        FOREIGN KEY (provider_code) REFERENCES payment_provider (code),
    CONSTRAINT payment_provider_transaction_asset_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_provider_transaction_order_fk
        FOREIGN KEY (order_id) REFERENCES payment_order (id),
    CONSTRAINT payment_provider_transaction_attempt_fk
        FOREIGN KEY (attempt_id) REFERENCES payment_attempt (id)
);

INSERT IGNORE INTO payment_provider (code, title, provider_kind, supports_create, supports_redirect, supports_webhook, supports_refund)
VALUES
    ('vkma', 'VK Mini Apps', 'platform_internal', 0, 0, 1, 1),
    ('telegram_stars', 'Telegram Stars', 'platform_internal', 1, 0, 1, 1),
    ('ton', 'TON blockchain', 'crypto_chain', 1, 0, 0, 0),
    ('yookassa', 'YooKassa', 'fiat_gateway', 1, 1, 1, 1),
    ('platega', 'Platega', 'fiat_gateway', 1, 1, 1, 1);

INSERT INTO payment_asset (code, title, asset_kind, scale, chain, network, contract_address, is_active)
VALUES
    ('VOTE', 'VK Votes', 'platform_currency', 0, NULL, NULL, NULL, 1),
    ('XTR', 'Telegram Stars', 'platform_currency', 0, NULL, NULL, NULL, 1),
    ('RUB', 'Russian Ruble', 'fiat', 2, NULL, NULL, NULL, 1),
    ('TON', 'Toncoin', 'crypto_native', 9, 'ton', 'mainnet', 'EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c', 1),
    ('USDT_TON', 'Tether USD on TON', 'crypto_jetton', 6, 'ton', 'mainnet', 'EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs', 1),
    ('TSTON_TON', 'Tonstakers TON', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQC98_qAmNEptUtPc7W6xdHh_ZHrBUFpw5Ft_IzNU20QAJav', 1),
    ('UTYA_TON', 'Utya', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQBaCgUwOoc6gHCNln_oJzb0mVs79YG7wYoavh-o1ItaneLA', 1),
    ('STON_TON', 'STON', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQA2kCVNwVsil2EM2mB0SkXytxCqQjS4mttjDpnXmwG9T6bO', 1),
    ('REDO_TON', 'Resistance Dog', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQBZ_cafPyDr5KUTs0aNxh0ZTDhkpEZONmLJA2SNGlLm4Cko', 1),
    ('STORM_TON', 'STORM', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQBsosmcZrD6FHijA7qWGLw5wo_aH8UN435hi935jJ_STORM', 1),
    ('GEMSTON_TON', 'GEMSTON', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQBX6K9aXVl3nXINCyPPL86C4ONVmQ8vK360u6dykFKXpHCa', 1),
    ('NOT_TON', 'Notcoin', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQAvlWFDxGF2lXm67y4yzC17wYKD9A0guwPkMs1gOsM__NOT', 1),
    ('JETTON_TON', 'JetTon', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQAQXlWJvGbbFfE8F3oS8s87lIgdovS455IsWFaRdmJetTon', 1),
    ('MAJOR_TON', 'Major', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQCuPm01HldiduQ55xaBF_1kaW_WAUy5DHey8suqzU_MAJOR', 1),
    ('DOGS_TON', 'Dogs', 'crypto_jetton', 9, 'ton', 'mainnet', 'EQCvxJy4eG8hyHBFsZ7eePxrRsUQSFE_jpptRAYBmcG_DOGS', 1),
    ('MEMCOIN_TON', 'Memcoin Jetton on TON', 'crypto_jetton', 9, 'ton', 'mainnet', NULL, 1)
ON DUPLICATE KEY UPDATE
    title = VALUES(title),
    asset_kind = VALUES(asset_kind),
    scale = VALUES(scale),
    chain = VALUES(chain),
    network = VALUES(network),
    contract_address = VALUES(contract_address),
    is_active = VALUES(is_active),
    updated_at = NOW();

INSERT INTO payment_provider_asset (provider_code, asset_code, is_active)
VALUES
    ('vkma', 'VOTE', 1),
    ('telegram_stars', 'XTR', 1),
    ('ton', 'TON', 1),
    ('ton', 'USDT_TON', 1),
    ('ton', 'TSTON_TON', 1),
    ('ton', 'UTYA_TON', 1),
    ('ton', 'STON_TON', 1),
    ('ton', 'REDO_TON', 1),
    ('ton', 'STORM_TON', 1),
    ('ton', 'GEMSTON_TON', 1),
    ('ton', 'NOT_TON', 1),
    ('ton', 'JETTON_TON', 1),
    ('ton', 'MAJOR_TON', 1),
    ('ton', 'DOGS_TON', 1),
    ('ton', 'MEMCOIN_TON', 1),
    ('yookassa', 'RUB', 1),
    ('platega', 'RUB', 1)
ON DUPLICATE KEY UPDATE
    is_active = VALUES(is_active),
    updated_at = NOW();

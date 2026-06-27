CREATE TABLE IF NOT EXISTS promo_offer (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    code VARCHAR(255) NOT NULL,
    code_normalized VARCHAR(255) NOT NULL,
    payload JSON NOT NULL,
    target JSON NULL,
    max_activations BIGINT UNSIGNED NOT NULL DEFAULT 0,
    activation_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    start_at DATETIME NULL,
    end_at DATETIME NULL,
    deleted_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY promo_offer_code_uq (workspace_id, code_normalized),
    UNIQUE KEY promo_offer_workspace_id_uq (workspace_id, id),
    KEY promo_offer_list_idx (workspace_id, created_at DESC, id),
    CONSTRAINT promo_offer_window_chk CHECK (
        start_at IS NULL OR end_at IS NULL OR start_at < end_at
    )
);

CREATE TABLE IF NOT EXISTS promo_localization (
    workspace_id VARCHAR(64) NOT NULL,
    promo_id BIGINT UNSIGNED NOT NULL,
    locale VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, promo_id, locale),
    CONSTRAINT promo_localization_offer_fk
        FOREIGN KEY (workspace_id, promo_id)
        REFERENCES promo_offer (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS promo_reward (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    promo_id BIGINT UNSIGNED NOT NULL,
    reward_key VARCHAR(128) NOT NULL,
    reward_type ENUM('quantity', 'duration') NOT NULL DEFAULT 'quantity',
    quantity BIGINT NOT NULL,
    scale SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    duration_unit ENUM('second', 'minute', 'hour', 'day', 'week', 'month', 'year') NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY promo_reward_uq (workspace_id, promo_id, reward_key),
    KEY promo_reward_list_idx (workspace_id, promo_id, id),
    CONSTRAINT promo_reward_offer_fk
        FOREIGN KEY (workspace_id, promo_id)
        REFERENCES promo_offer (workspace_id, id) ON DELETE RESTRICT,
    CONSTRAINT promo_reward_quantity_chk CHECK (quantity > 0),
    CONSTRAINT promo_reward_type_chk CHECK (
        (reward_type = 'quantity' AND duration_unit IS NULL)
        OR (reward_type = 'duration' AND duration_unit IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS promo_redemption (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    promo_id BIGINT UNSIGNED NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    reward_snapshot JSON NOT NULL,
    redeemed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY promo_redemption_user_uq (
        workspace_id, promo_id, app_id, platform_id, platform_user_id
    ),
    UNIQUE KEY promo_redemption_workspace_id_uq (workspace_id, promo_id, id),
    KEY promo_redemption_stats_idx (workspace_id, promo_id, redeemed_at),
    KEY promo_redemption_user_idx (
        workspace_id, app_id, platform_id, platform_user_id, redeemed_at
    ),
    CONSTRAINT promo_redemption_offer_fk
        FOREIGN KEY (workspace_id, promo_id)
        REFERENCES promo_offer (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS promo_redemption_event (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    promo_id BIGINT UNSIGNED NOT NULL,
    redemption_id BIGINT UNSIGNED NOT NULL,
    occurred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY promo_redemption_event_uq (redemption_id),
    KEY promo_redemption_event_stats_idx (workspace_id, promo_id, occurred_at),
    CONSTRAINT promo_redemption_event_redemption_fk
        FOREIGN KEY (workspace_id, promo_id, redemption_id)
        REFERENCES promo_redemption (workspace_id, promo_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS promo_stats_daily (
    workspace_id VARCHAR(64) NOT NULL,
    promo_id BIGINT UNSIGNED NOT NULL,
    stats_date DATE NOT NULL,
    redemption_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_users BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, promo_id, stats_date)
);

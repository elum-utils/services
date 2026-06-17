CREATE TABLE IF NOT EXISTS cpa_offer (
    workspace_id VARCHAR(64) NOT NULL,
    id VARCHAR(128) NOT NULL,
    payload JSON NOT NULL,
    target JSON NULL,
    code_mode ENUM('shared_code', 'personal_code') NOT NULL,
    code_source ENUM('generated', 'pool') NULL,
    shared_code VARCHAR(512) NULL,
    generated_length SMALLINT UNSIGNED NULL,
    generated_alphabet VARCHAR(512) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    start_at DATETIME NULL,
    end_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, id),
    KEY cpa_offer_active_idx (workspace_id, is_active, start_at, end_at),
    KEY cpa_offer_list_idx (workspace_id, created_at DESC, id),
    KEY cpa_offer_active_list_idx (workspace_id, is_active, created_at DESC, id),
    CONSTRAINT cpa_offer_code_config_chk CHECK (
        (
            code_mode = 'shared_code'
            AND shared_code IS NOT NULL
            AND shared_code <> ''
            AND code_source IS NULL
            AND generated_length IS NULL
            AND generated_alphabet IS NULL
        )
        OR
        (
            code_mode = 'personal_code'
            AND shared_code IS NULL
            AND (
                (
                    code_source = 'pool'
                    AND generated_length IS NULL
                    AND generated_alphabet IS NULL
                )
                OR
                (
                    code_source = 'generated'
                    AND generated_length > 0
                    AND CHAR_LENGTH(generated_alphabet) >= 2
                )
            )
        )
    ),
    CONSTRAINT cpa_offer_window_chk CHECK (
        start_at IS NULL OR end_at IS NULL OR start_at < end_at
    )
);

CREATE TABLE IF NOT EXISTS cpa_localization (
    workspace_id VARCHAR(64) NOT NULL,
    cpa_id VARCHAR(128) NOT NULL,
    locale VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, cpa_id, locale),
    CONSTRAINT cpa_localization_offer_fk
        FOREIGN KEY (workspace_id, cpa_id)
        REFERENCES cpa_offer (workspace_id, id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cpa_reward (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    cpa_id VARCHAR(128) NOT NULL,
    reward_key VARCHAR(128) NOT NULL,
    reward_type ENUM('quantity', 'duration') NOT NULL DEFAULT 'quantity',
    quantity BIGINT NOT NULL DEFAULT 1,
    duration_unit ENUM('second', 'minute', 'hour', 'day', 'week', 'month', 'year') NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY cpa_reward_uq (workspace_id, cpa_id, reward_key),
    KEY cpa_reward_list_idx (workspace_id, cpa_id, id),
    CONSTRAINT cpa_reward_offer_fk
        FOREIGN KEY (workspace_id, cpa_id)
        REFERENCES cpa_offer (workspace_id, id)
        ON DELETE CASCADE,
    CONSTRAINT cpa_reward_quantity_chk CHECK (quantity > 0),
    CONSTRAINT cpa_reward_type_chk CHECK (
        (reward_type = 'quantity' AND duration_unit IS NULL)
        OR (reward_type = 'duration' AND duration_unit IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS cpa_code (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    cpa_id VARCHAR(128) NOT NULL,
    code VARCHAR(512) NOT NULL,
    source ENUM('generated', 'pool') NOT NULL,
    status ENUM('available', 'issued', 'completed', 'deleted') NOT NULL DEFAULT 'available',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    PRIMARY KEY (id),
    UNIQUE KEY cpa_code_uq (workspace_id, cpa_id, code),
    KEY cpa_code_available_idx (workspace_id, cpa_id, status, id),
    CONSTRAINT cpa_code_offer_fk
        FOREIGN KEY (workspace_id, cpa_id)
        REFERENCES cpa_offer (workspace_id, id)
        ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cpa_assignment (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    cpa_id VARCHAR(128) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    code_id BIGINT UNSIGNED NULL,
    code VARCHAR(512) NOT NULL,
    code_mode ENUM('shared_code', 'personal_code') NOT NULL,
    status ENUM('issued', 'completed') NOT NULL DEFAULT 'issued',
    issued_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME NULL,
    deleted_at DATETIME NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY cpa_assignment_user_uq (
        workspace_id,
        cpa_id,
        app_id,
        platform_id,
        platform_user_id
    ),
    UNIQUE KEY cpa_assignment_code_uq (code_id),
    KEY cpa_assignment_status_idx (workspace_id, cpa_id, status, issued_at),
    KEY cpa_assignment_user_idx (
        workspace_id,
        app_id,
        platform_id,
        platform_user_id,
        issued_at
    ),
    CONSTRAINT cpa_assignment_offer_fk
        FOREIGN KEY (workspace_id, cpa_id)
        REFERENCES cpa_offer (workspace_id, id)
        ON DELETE RESTRICT,
    CONSTRAINT cpa_assignment_code_fk
        FOREIGN KEY (code_id)
        REFERENCES cpa_code (id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS cpa_assignment_event (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    cpa_id VARCHAR(128) NOT NULL,
    assignment_id BIGINT UNSIGNED NOT NULL,
    event_type ENUM('issued', 'completed') NOT NULL,
    occurred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY cpa_assignment_event_uq (assignment_id, event_type),
    KEY cpa_assignment_event_stats_idx (workspace_id, cpa_id, occurred_at, event_type),
    CONSTRAINT cpa_assignment_event_assignment_fk
        FOREIGN KEY (assignment_id)
        REFERENCES cpa_assignment (id)
        ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS cpa_stats_daily (
    workspace_id VARCHAR(64) NOT NULL,
    cpa_id VARCHAR(128) NOT NULL,
    stats_date DATE NOT NULL,
    issued_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    completed_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_users BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, cpa_id, stats_date),
    KEY cpa_stats_daily_date_idx (workspace_id, stats_date, cpa_id)
);

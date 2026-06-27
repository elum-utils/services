CREATE TABLE IF NOT EXISTS calendar_definition (
    id CHAR(36) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    type VARCHAR(64) NOT NULL,
    mode ENUM('interval', 'sequential', 'sequential_reset') NOT NULL,
    interval_type ENUM('calendar', 'floating') NOT NULL,
    interval_unit ENUM('second', 'minute', 'hour', 'day', 'week', 'month') NOT NULL,
    interval_count INT UNSIGNED NOT NULL DEFAULT 1,
    reset_after_intervals INT UNSIGNED NOT NULL DEFAULT 1,
    end_behavior ENUM('restart', 'repeat_last', 'stop') NOT NULL DEFAULT 'stop',
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    hide_future_rewards BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    start_at DATETIME NULL,
    end_at DATETIME NULL,
    deleted_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY calendar_definition_type_uq (workspace_id, type),
    UNIQUE KEY calendar_definition_workspace_id_uq (workspace_id, id),
    KEY calendar_definition_active_idx (
        workspace_id, is_active, deleted_at, start_at, end_at, created_at
    ),
    CONSTRAINT calendar_definition_interval_count_chk CHECK (interval_count > 0),
    CONSTRAINT calendar_definition_reset_count_chk CHECK (reset_after_intervals > 0),
    CONSTRAINT calendar_definition_window_chk CHECK (
        start_at IS NULL OR end_at IS NULL OR start_at < end_at
    )
);

CREATE TABLE IF NOT EXISTS calendar_localization (
    workspace_id VARCHAR(64) NOT NULL,
    calendar_id CHAR(36) NOT NULL,
    locale VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, calendar_id, locale),
    CONSTRAINT calendar_localization_definition_fk
        FOREIGN KEY (workspace_id, calendar_id)
        REFERENCES calendar_definition (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS calendar_step (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    calendar_id CHAR(36) NOT NULL,
    position INT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY calendar_step_position_uq (workspace_id, calendar_id, position),
    UNIQUE KEY calendar_step_workspace_id_uq (workspace_id, calendar_id, id),
    CONSTRAINT calendar_step_position_chk CHECK (position > 0),
    CONSTRAINT calendar_step_definition_fk
        FOREIGN KEY (workspace_id, calendar_id)
        REFERENCES calendar_definition (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS calendar_reward (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    calendar_id CHAR(36) NOT NULL,
    step_id BIGINT UNSIGNED NOT NULL,
    item_key VARCHAR(128) NOT NULL,
    reward_type ENUM('quantity', 'duration') NOT NULL DEFAULT 'quantity',
    item_count BIGINT NOT NULL,
    scale SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    duration_unit ENUM('second', 'minute', 'hour', 'day', 'week', 'month', 'year') NULL,
    position INT UNSIGNED NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY calendar_reward_key_uq (
        workspace_id, calendar_id, step_id, item_key
    ),
    KEY calendar_reward_list_idx (
        workspace_id, calendar_id, step_id, position, id
    ),
    CONSTRAINT calendar_reward_count_chk CHECK (item_count > 0),
    CONSTRAINT calendar_reward_type_chk CHECK (
        (reward_type = 'quantity' AND duration_unit IS NULL)
        OR (reward_type = 'duration' AND duration_unit IS NOT NULL)
    ),
    CONSTRAINT calendar_reward_position_chk CHECK (position > 0),
    CONSTRAINT calendar_reward_step_fk
        FOREIGN KEY (workspace_id, calendar_id, step_id)
        REFERENCES calendar_step (workspace_id, calendar_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS calendar_progress (
    workspace_id VARCHAR(64) NOT NULL,
    calendar_id CHAR(36) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    current_position INT UNSIGNED NOT NULL DEFAULT 0,
    claim_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    last_claim_position INT UNSIGNED NULL,
    last_claim_at DATETIME NULL,
    next_claim_at DATETIME NULL,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    reset_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    last_was_reset BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (
        workspace_id, calendar_id, app_id, platform_id, platform_user_id
    ),
    KEY calendar_progress_user_idx (
        workspace_id, app_id, platform_id, platform_user_id, updated_at
    ),
    CONSTRAINT calendar_progress_definition_fk
        FOREIGN KEY (workspace_id, calendar_id)
        REFERENCES calendar_definition (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS calendar_operation (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    calendar_id CHAR(36) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    operation_id VARCHAR(128) NOT NULL,
    granted BOOLEAN NOT NULL,
    status VARCHAR(32) NOT NULL,
    position INT UNSIGNED NULL,
    rewards_snapshot JSON NOT NULL,
    current_position INT UNSIGNED NOT NULL,
    claim_count BIGINT UNSIGNED NOT NULL,
    last_claim_position INT UNSIGNED NULL,
    last_claim_at DATETIME NULL,
    next_claim_at DATETIME NULL,
    is_completed BOOLEAN NOT NULL,
    reset_count BIGINT UNSIGNED NOT NULL,
    was_reset BOOLEAN NOT NULL,
    occurred_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY calendar_operation_idempotency_uq (
        workspace_id, calendar_id, app_id, platform_id,
        platform_user_id, operation_id
    ),
    KEY calendar_operation_stats_idx (
        workspace_id, calendar_id, occurred_at, granted
    ),
    KEY calendar_operation_user_idx (
        workspace_id, app_id, platform_id, platform_user_id, occurred_at
    ),
    CONSTRAINT calendar_operation_definition_fk
        FOREIGN KEY (workspace_id, calendar_id)
        REFERENCES calendar_definition (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS calendar_stats_daily (
    workspace_id VARCHAR(64) NOT NULL,
    calendar_id CHAR(36) NOT NULL,
    stats_date DATE NOT NULL,
    operation_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    grant_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_users BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, calendar_id, stats_date)
);

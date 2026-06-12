CREATE TABLE IF NOT EXISTS task_group (
    workspace_id VARCHAR(64) NOT NULL,
    `key` VARCHAR(100) NOT NULL,
    position INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, `key`),
    KEY task_group_list_idx (workspace_id, is_active, deleted_at, position, `key`)
);

CREATE TABLE IF NOT EXISTS task_group_localization (
    workspace_id VARCHAR(64) NOT NULL,
    group_key VARCHAR(100) NOT NULL,
    locale VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, group_key, locale),
    CONSTRAINT task_group_localization_group_fk
        FOREIGN KEY (workspace_id, group_key)
        REFERENCES task_group (workspace_id, `key`) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS task_sequence (
    workspace_id VARCHAR(64) NOT NULL,
    `key` VARCHAR(100) NOT NULL,
    position INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, `key`),
    KEY task_sequence_list_idx (workspace_id, is_active, deleted_at, position, `key`)
);

CREATE TABLE IF NOT EXISTS task_definition (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    `key` VARCHAR(100) NOT NULL,
    group_key VARCHAR(100) NOT NULL,
    sequence_key VARCHAR(100) NULL,
    sequence_position INT NULL,
    action_key VARCHAR(150) NOT NULL,
    action_kind ENUM('app_action', 'amount_action', 'channel_subscribe', 'advertisement_view', 'external') NOT NULL,
    claim_mode ENUM('manual', 'auto') NOT NULL DEFAULT 'manual',
    target_count BIGINT UNSIGNED NOT NULL DEFAULT 1,
    reset_unit ENUM('never', 'second', 'minute', 'hour', 'day', 'year') NOT NULL DEFAULT 'never',
    reset_every INT UNSIGNED NOT NULL DEFAULT 1,
    position INT NOT NULL DEFAULT 0,
    payload JSON NULL,
    image_url VARCHAR(1024) NULL,
    is_visible BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    start_at DATETIME NULL,
    end_at DATETIME NULL,
    deleted_at DATETIME NULL,
    branch_sort_key VARCHAR(101)
        GENERATED ALWAYS AS (COALESCE(sequence_key, CONCAT('~', `key`))) STORED,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY task_definition_workspace_id_uq (workspace_id, id),
    UNIQUE KEY task_definition_key_uq (workspace_id, `key`),
    UNIQUE KEY task_definition_sequence_position_uq (workspace_id, sequence_key, sequence_position),
    KEY task_definition_action_idx (workspace_id, action_key, is_active, deleted_at, start_at, end_at, position, id),
    KEY task_definition_admin_list_idx (workspace_id, deleted_at, position, id),
    KEY task_definition_admin_group_list_idx (workspace_id, group_key, deleted_at, position, id),
    KEY task_definition_active_branch_idx (workspace_id, is_active, deleted_at, branch_sort_key, sequence_position, position, id),
    KEY task_definition_visible_list_idx (workspace_id, is_visible, is_active, deleted_at, position, id),
    KEY task_definition_visible_user_list_idx (workspace_id, is_visible, is_active, deleted_at, position, id, `key`, group_key),
    KEY task_definition_group_idx (workspace_id, group_key, is_visible, is_active, deleted_at, position, id),
    KEY task_definition_sequence_idx (workspace_id, sequence_key, sequence_position, id),
    CONSTRAINT task_definition_group_fk
        FOREIGN KEY (workspace_id, group_key)
        REFERENCES task_group (workspace_id, `key`) ON DELETE RESTRICT,
    CONSTRAINT task_definition_sequence_fk
        FOREIGN KEY (workspace_id, sequence_key)
        REFERENCES task_sequence (workspace_id, `key`) ON DELETE RESTRICT,
    CONSTRAINT task_definition_sequence_chk CHECK (
        (sequence_key IS NULL AND sequence_position IS NULL) OR
        (sequence_key IS NOT NULL AND sequence_position IS NOT NULL AND sequence_position > 0)
    ),
    CONSTRAINT task_definition_target_chk CHECK (target_count > 0),
    CONSTRAINT task_definition_reset_chk CHECK (reset_every > 0),
    CONSTRAINT task_definition_window_chk CHECK (start_at IS NULL OR end_at IS NULL OR start_at < end_at)
);

CREATE TABLE IF NOT EXISTS task_localization (
    workspace_id VARCHAR(64) NOT NULL,
    task_id BIGINT UNSIGNED NOT NULL,
    locale VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, task_id, locale),
    CONSTRAINT task_localization_definition_fk
        FOREIGN KEY (workspace_id, task_id)
        REFERENCES task_definition (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS task_reward (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    task_id BIGINT UNSIGNED NOT NULL,
    reward_key VARCHAR(150) NOT NULL,
    reward_type ENUM('quantity', 'duration') NOT NULL DEFAULT 'quantity',
    quantity BIGINT NOT NULL,
    duration_unit ENUM('second', 'minute', 'hour', 'day', 'week', 'month', 'year') NULL,
    position INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY task_reward_workspace_id_uq (workspace_id, task_id, id),
    UNIQUE KEY task_reward_key_uq (workspace_id, task_id, reward_key),
    KEY task_reward_list_idx (workspace_id, task_id, position, id),
    CONSTRAINT task_reward_definition_fk
        FOREIGN KEY (workspace_id, task_id)
        REFERENCES task_definition (workspace_id, id) ON DELETE RESTRICT,
    CONSTRAINT task_reward_quantity_chk CHECK (quantity > 0),
    CONSTRAINT task_reward_type_chk CHECK (
        (reward_type = 'quantity' AND duration_unit IS NULL)
        OR (reward_type = 'duration' AND duration_unit IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS task_progress (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    task_id BIGINT UNSIGNED NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    period_start_at DATETIME NOT NULL,
    period_end_at DATETIME NOT NULL,
    progress BIGINT UNSIGNED NOT NULL DEFAULT 0,
    status ENUM('open', 'ready', 'claimed') NOT NULL DEFAULT 'open',
    ready_at DATETIME NULL,
    claimed_at DATETIME NULL,
    operation_id VARCHAR(128) NULL,
    rewards_snapshot JSON NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY task_progress_identity_uq (
        workspace_id, task_id, app_id, platform_id, platform_user_id, period_start_at
    ),
    KEY task_progress_user_idx (workspace_id, app_id, platform_id, platform_user_id, period_start_at, task_id),
    KEY task_progress_current_user_idx (workspace_id, app_id, platform_id, platform_user_id, period_start_at, period_end_at, task_id),
    KEY task_progress_task_idx (workspace_id, task_id, period_start_at, status),
    CONSTRAINT task_progress_definition_fk
        FOREIGN KEY (workspace_id, task_id)
        REFERENCES task_definition (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS task_sequence_state (
    workspace_id VARCHAR(64) NOT NULL,
    sequence_key VARCHAR(100) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    current_task_id BIGINT UNSIGNED NULL,
    status ENUM('active', 'completed') NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, sequence_key, app_id, platform_id, platform_user_id),
    KEY task_sequence_state_current_idx (
        workspace_id, app_id, platform_id, platform_user_id, status, current_task_id
    ),
    CONSTRAINT task_sequence_state_sequence_fk
        FOREIGN KEY (workspace_id, sequence_key)
        REFERENCES task_sequence (workspace_id, `key`) ON DELETE RESTRICT,
    CONSTRAINT task_sequence_state_current_task_fk
        FOREIGN KEY (workspace_id, current_task_id)
        REFERENCES task_definition (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS task_progress_event (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(64) NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    source VARCHAR(64) NOT NULL,
    external_event_key VARCHAR(255) NOT NULL,
    action_key VARCHAR(150) NOT NULL,
    amount BIGINT UNSIGNED NOT NULL DEFAULT 1,
    payload JSON NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY task_progress_event_uq (
        workspace_id, source, external_event_key, app_id, platform_id, platform_user_id
    ),
    KEY task_progress_event_external_user_idx (
        workspace_id, app_id, platform_id, platform_user_id, source, external_event_key
    )
);

CREATE TABLE IF NOT EXISTS task_stats_event (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    workspace_id VARCHAR(64) NOT NULL,
    task_id BIGINT UNSIGNED NOT NULL,
    progress_id BIGINT UNSIGNED NOT NULL,
    app_id BIGINT NOT NULL,
    platform_id BIGINT NOT NULL,
    platform_user_id VARCHAR(255) NOT NULL,
    event_type ENUM('progress_created', 'progress_added', 'ready', 'claimed') NOT NULL,
    claim_mode ENUM('manual', 'auto') NULL,
    amount BIGINT UNSIGNED NOT NULL DEFAULT 0,
    occurred_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY task_stats_event_workspace_idx (workspace_id, occurred_at, event_type),
    KEY task_stats_event_task_idx (workspace_id, task_id, occurred_at, event_type),
    KEY task_stats_event_user_idx (
        workspace_id, app_id, platform_id, platform_user_id, occurred_at
    ),
    CONSTRAINT task_stats_event_definition_fk
        FOREIGN KEY (workspace_id, task_id)
        REFERENCES task_definition (workspace_id, id) ON DELETE RESTRICT,
    CONSTRAINT task_stats_event_progress_fk
        FOREIGN KEY (progress_id)
        REFERENCES task_progress (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS task_stats_daily (
    workspace_id VARCHAR(64) NOT NULL,
    task_id BIGINT UNSIGNED NOT NULL,
    stats_date DATE NOT NULL,
    progress_created BIGINT UNSIGNED NOT NULL DEFAULT 0,
    progress_amount BIGINT UNSIGNED NOT NULL DEFAULT 0,
    ready_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    claimed_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    manual_claimed_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    auto_claimed_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_participants BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_claimers BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, task_id, stats_date),
    KEY task_stats_daily_date_idx (workspace_id, stats_date, task_id),
    CONSTRAINT task_stats_daily_definition_fk
        FOREIGN KEY (workspace_id, task_id)
        REFERENCES task_definition (workspace_id, id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS task_stats_daily_overview (
    workspace_id VARCHAR(64) NOT NULL,
    stats_date DATE NOT NULL,
    tasks_total BIGINT UNSIGNED NOT NULL DEFAULT 0,
    active_tasks BIGINT UNSIGNED NOT NULL DEFAULT 0,
    visible_tasks BIGINT UNSIGNED NOT NULL DEFAULT 0,
    progress_created BIGINT UNSIGNED NOT NULL DEFAULT 0,
    progress_amount BIGINT UNSIGNED NOT NULL DEFAULT 0,
    ready_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    claimed_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    manual_claimed_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    auto_claimed_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_participants BIGINT UNSIGNED NOT NULL DEFAULT 0,
    unique_claimers BIGINT UNSIGNED NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, stats_date),
    KEY task_stats_daily_overview_date_idx (stats_date, workspace_id)
);

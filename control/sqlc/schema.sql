CREATE TABLE IF NOT EXISTS control_account (
    id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    status ENUM('active', 'disabled') NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS control_identity (
    account_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    provider VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    provider_subject VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    payload JSON NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, provider),
    UNIQUE KEY control_identity_provider_subject_uq (provider, provider_subject),
    CONSTRAINT control_identity_account_fk FOREIGN KEY (account_id)
        REFERENCES control_account (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS control_session (
    id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    account_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    ip VARCHAR(45) NOT NULL DEFAULT '',
    user_agent VARCHAR(255) NOT NULL DEFAULT '',
    bind_to_ip BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME NULL,
    last_used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY control_session_token_hash_uq (token_hash),
    KEY control_session_account_idx (account_id, revoked_at, expires_at),
    KEY control_session_account_created_idx (account_id, created_at),
    CONSTRAINT control_session_account_fk FOREIGN KEY (account_id)
        REFERENCES control_account (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS control_workspace (
    id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    slug VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    title VARCHAR(255) NOT NULL,
    status ENUM('active', 'archived') NOT NULL DEFAULT 'active',
    created_by VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY control_workspace_slug_uq (slug),
    CONSTRAINT control_workspace_creator_fk FOREIGN KEY (created_by)
        REFERENCES control_account (id)
);

CREATE TABLE IF NOT EXISTS control_workspace_member (
    workspace_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    account_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    status ENUM('active', 'removed') NOT NULL DEFAULT 'active',
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, account_id),
    KEY control_member_account_idx (account_id, status),
    CONSTRAINT control_member_workspace_fk FOREIGN KEY (workspace_id)
        REFERENCES control_workspace (id) ON DELETE CASCADE,
    CONSTRAINT control_member_account_fk FOREIGN KEY (account_id)
        REFERENCES control_account (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS control_workspace_invite (
    id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    workspace_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    created_by VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    max_uses INT UNSIGNED NULL,
    used_count INT UNSIGNED NOT NULL DEFAULT 0,
    expires_at DATETIME NULL,
    revoked_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY control_invite_token_hash_uq (token_hash),
    KEY control_invite_workspace_idx (workspace_id, revoked_at, expires_at),
    KEY control_invite_workspace_created_idx (workspace_id, created_at),
    CONSTRAINT control_invite_workspace_fk FOREIGN KEY (workspace_id)
        REFERENCES control_workspace (id) ON DELETE CASCADE,
    CONSTRAINT control_invite_creator_fk FOREIGN KEY (created_by)
        REFERENCES control_account (id)
);

CREATE TABLE IF NOT EXISTS control_role (
    id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    workspace_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    code VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    position INT NOT NULL,
    is_owner BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY control_role_workspace_code_uq (workspace_id, code),
    KEY control_role_workspace_position_idx (workspace_id, deleted_at, position),
    CONSTRAINT control_role_workspace_fk FOREIGN KEY (workspace_id)
        REFERENCES control_workspace (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS control_role_member (
    role_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    account_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, account_id),
    KEY control_role_member_account_idx (account_id),
    CONSTRAINT control_role_member_role_fk FOREIGN KEY (role_id)
        REFERENCES control_role (id) ON DELETE CASCADE,
    CONSTRAINT control_role_member_account_fk FOREIGN KEY (account_id)
        REFERENCES control_account (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS control_workspace_invite_role (
    invite_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    role_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    PRIMARY KEY (invite_id, role_id),
    CONSTRAINT control_invite_role_invite_fk FOREIGN KEY (invite_id)
        REFERENCES control_workspace_invite (id) ON DELETE CASCADE,
    CONSTRAINT control_invite_role_role_fk FOREIGN KEY (role_id)
        REFERENCES control_role (id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS control_method (
    method_key VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    group_key VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (method_key),
    KEY control_method_service_idx (service, group_key)
);

CREATE TABLE IF NOT EXISTS control_method_group (
    service VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    group_key VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (service, group_key)
);

CREATE TABLE IF NOT EXISTS control_access_service (
    service VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (service)
);

CREATE TABLE IF NOT EXISTS control_localization (
    localization_key VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    locale VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (localization_key, locale)
);

CREATE TABLE IF NOT EXISTS control_role_permission (
    role_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    method_key VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, method_key),
    KEY control_permission_method_idx (method_key),
    CONSTRAINT control_permission_role_fk FOREIGN KEY (role_id)
        REFERENCES control_role (id) ON DELETE CASCADE,
    CONSTRAINT control_permission_method_fk FOREIGN KEY (method_key)
        REFERENCES control_method (method_key) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS control_two_factor (
    account_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    secret VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    backup_hashes JSON NOT NULL,
    activated_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id),
    CONSTRAINT control_two_factor_account_fk FOREIGN KEY (account_id)
        REFERENCES control_account (id) ON DELETE CASCADE,
    CONSTRAINT control_two_factor_backup_hashes_chk CHECK (JSON_VALID(backup_hashes))
);

CREATE TABLE IF NOT EXISTS control_two_factor_challenge (
    id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    account_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    token_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    ip VARCHAR(45) NOT NULL DEFAULT '',
    user_agent VARCHAR(255) NOT NULL DEFAULT '',
    bind_to_ip BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at DATETIME NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY control_two_factor_challenge_token_uq (token_hash),
    KEY control_two_factor_challenge_account_idx (account_id, expires_at),
    CONSTRAINT control_two_factor_challenge_account_fk FOREIGN KEY (account_id)
        REFERENCES control_account (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS control_audit_event (
    id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    workspace_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    actor_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    method_key VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    target_type VARCHAR(64) NOT NULL DEFAULT '',
    target_id VARCHAR(128) NOT NULL DEFAULT '',
    before_data JSON NULL,
    after_data JSON NULL,
    result ENUM('succeeded', 'failed') NOT NULL,
    request_id VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
    occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY control_audit_workspace_idx (workspace_id, occurred_at),
    KEY control_audit_actor_idx (actor_id, occurred_at)
);

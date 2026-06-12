CREATE TABLE IF NOT EXISTS reference_item (
    workspace_id VARCHAR(64) NOT NULL,
    `key` VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    item_type ENUM('quantity', 'duration') NOT NULL,
    payload JSON NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    deleted_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, `key`),
    KEY reference_item_list_idx (workspace_id, deleted_at, is_active, item_type, `key`),
    CONSTRAINT reference_item_payload_chk CHECK (JSON_VALID(payload))
);

CREATE TABLE IF NOT EXISTS reference_localization (
    workspace_id VARCHAR(64) NOT NULL,
    item_key VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    locale VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, item_key, locale),
    KEY reference_localization_locale_idx (workspace_id, locale, item_key),
    CONSTRAINT reference_localization_item_fk
        FOREIGN KEY (workspace_id, item_key)
        REFERENCES reference_item (workspace_id, `key`)
        ON UPDATE RESTRICT ON DELETE RESTRICT
);

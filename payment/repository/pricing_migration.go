package repository

import (
	"context"
	"fmt"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

func (r *PaymentRepository) migrateDynamicPricing(ctx context.Context) error {
	if err := r.migrateGlobalAssetRates(ctx); err != nil {
		return err
	}

	rateColumns := []struct {
		name       string
		definition string
	}{
		{"auto_update_enabled", "TINYINT(1) NOT NULL DEFAULT 0 AFTER observed_at"},
		{"auto_update_source", "VARCHAR(32) NULL AFTER auto_update_enabled"},
		{"source_chain_id", "VARCHAR(32) NULL AFTER auto_update_source"},
		{"source_token_address", "VARCHAR(128) NULL AFTER source_chain_id"},
		{"last_attempt_at", "DATETIME NULL AFTER source_token_address"},
		{"last_error", "TEXT NULL AFTER last_attempt_at"},
		{"lease_owner", "VARCHAR(64) NULL AFTER last_error"},
		{"lease_until", "DATETIME NULL AFTER lease_owner"},
	}
	for _, column := range rateColumns {
		exists, err := r.columnExists(ctx, "payment_asset_rate", column.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		statement := "ALTER TABLE payment_asset_rate ADD COLUMN `" + column.name + "` " + column.definition
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, statement)
			return err
		}); err != nil {
			return fmt.Errorf("payment: add asset rate auto-update column %s: %w", column.name, err)
		}
	}

	rateIndexExists, err := r.indexExists(ctx, "payment_asset_rate", "payment_asset_rate_auto_lease_idx")
	if err != nil {
		return err
	}
	if !rateIndexExists {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, `
ALTER TABLE payment_asset_rate
ADD KEY payment_asset_rate_auto_lease_idx (
    auto_update_enabled,
    lease_until
)`)
			return err
		}); err != nil {
			return fmt.Errorf("payment: add asset rate auto-update index: %w", err)
		}
	}

	columns := []struct {
		name       string
		definition string
	}{
		{"pricing_mode", "ENUM('fixed', 'dynamic') NOT NULL DEFAULT 'fixed' AFTER discount_amount_minor"},
		{"reference_asset_code", "VARCHAR(32) NULL AFTER pricing_mode"},
		{"reference_list_amount_minor", "BIGINT UNSIGNED NULL AFTER reference_asset_code"},
		{"reference_discount_amount_minor", "BIGINT UNSIGNED NULL AFTER reference_list_amount_minor"},
		{"coefficient", "DECIMAL(24,12) NULL AFTER reference_discount_amount_minor"},
	}
	for _, column := range columns {
		exists, err := r.columnExists(ctx, "payment_price", column.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		statement := "ALTER TABLE payment_price ADD COLUMN `" + column.name + "` " + column.definition
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, statement)
			return err
		}); err != nil {
			return fmt.Errorf("payment: add dynamic pricing column %s: %w", column.name, err)
		}
	}

	exists, err := r.indexExists(ctx, "payment_price", "payment_price_dynamic_idx")
	if err != nil {
		return err
	}
	if !exists {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, `
ALTER TABLE payment_price
ADD KEY payment_price_dynamic_idx (
    workspace_id,
    asset_code,
    reference_asset_code,
    pricing_mode
)`)
			return err
		}); err != nil {
			return fmt.Errorf("payment: add dynamic pricing index: %w", err)
		}
	}
	return nil
}

func (r *PaymentRepository) migrateGlobalAssetRates(ctx context.Context) error {
	hasWorkspace, err := r.columnExists(ctx, "payment_asset_rate", "workspace_id")
	if err != nil {
		return err
	}
	hasMinorRate, err := r.columnExists(ctx, "payment_asset_rate", "reference_per_asset_minor")
	if err != nil {
		return err
	}
	if !hasWorkspace && hasMinorRate {
		return nil
	}

	if hasWorkspace {
		statements := []string{
			"DROP TABLE IF EXISTS payment_asset_rate_legacy_workspace",
			"RENAME TABLE payment_asset_rate TO payment_asset_rate_legacy_workspace",
			`CREATE TABLE payment_asset_rate (
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
    CONSTRAINT payment_asset_rate_asset_global_fk
        FOREIGN KEY (asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_asset_rate_reference_global_fk
        FOREIGN KEY (reference_asset_code) REFERENCES payment_asset (code),
    CONSTRAINT payment_asset_rate_positive_global_chk CHECK (reference_per_asset_minor > 0)
)`,
			`INSERT INTO payment_asset_rate (
    asset_code,
    reference_asset_code,
    reference_per_asset_minor,
    source,
    observed_at,
    auto_update_enabled,
    auto_update_source,
    source_chain_id,
    source_token_address,
    last_attempt_at,
    last_error,
    created_at,
    updated_at
)
SELECT
    asset_code,
    reference_asset_code,
    GREATEST(1, CAST(CEIL(MAX(reference_per_asset) * 1000000) AS UNSIGNED)),
    MAX(source),
    MAX(observed_at),
    MAX(auto_update_enabled),
    MAX(auto_update_source),
    MAX(source_chain_id),
    MAX(source_token_address),
    MAX(last_attempt_at),
    MAX(last_error),
    MIN(created_at),
    MAX(updated_at)
FROM payment_asset_rate_legacy_workspace
GROUP BY asset_code, reference_asset_code`,
			"DROP TABLE payment_asset_rate_legacy_workspace",
		}
		for _, statement := range statements {
			if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
				_, err := r.db.DB().ExecContext(ctx, statement)
				return err
			}); err != nil {
				return fmt.Errorf("payment: migrate global asset rates: %w", err)
			}
		}
		return nil
	}

	if !hasMinorRate {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, `
ALTER TABLE payment_asset_rate
CHANGE COLUMN reference_per_asset reference_per_asset_minor BIGINT UNSIGNED NOT NULL`)
			return err
		}); err != nil {
			return fmt.Errorf("payment: convert asset rate to minor units: %w", err)
		}
	}
	return nil
}

func (r *PaymentRepository) columnExists(ctx context.Context, tableName string, columnName string) (bool, error) {
	var count int
	err := r.db.DB().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND column_name = ?`, tableName, columnName).Scan(&count)
	return count > 0, err
}

func (r *PaymentRepository) indexExists(ctx context.Context, tableName string, indexName string) (bool, error) {
	var count int
	err := r.db.DB().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.statistics
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND index_name = ?`, tableName, indexName).Scan(&count)
	return count > 0, err
}

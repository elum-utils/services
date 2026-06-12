package repository

import (
	"context"
	"fmt"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

func (r *PaymentRepository) migrateLegacyTONStorage(ctx context.Context) error {
	cursorExists, err := r.tableExists(ctx, "payment_ton_wallet_cursor")
	if err != nil {
		return err
	}
	transactionExists, err := r.tableExists(ctx, "payment_ton_transaction")
	if err != nil {
		return err
	}

	if cursorExists {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, `
INSERT IGNORE INTO payment_provider_cursor (
    workspace_id, provider_code, network, source_key,
    cursor_value, cursor_sequence, updated_at
)
SELECT
    workspace_id, 'ton', network, wallet_address,
    CAST(last_lt AS CHAR), last_lt, updated_at
FROM payment_ton_wallet_cursor`)
			return err
		}); err != nil {
			return fmt.Errorf("payment: migrate TON cursors: %w", err)
		}
	}

	if transactionExists {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, `
INSERT IGNORE INTO payment_provider_transaction (
    workspace_id, provider_code, network, source_key,
    asset_code, external_transaction_id, sequence_number,
    source_address, destination_address, amount_minor,
    payment_reference, sender_reference, order_id, attempt_id,
    status, error, occurred_at, created_at
)
SELECT
    workspace_id, 'ton', network, wallet_address,
    asset_code, tx_hash, logical_time,
    source_address, destination_address, amount_minor,
    comment, jetton_sender, order_id, attempt_id,
    status, error, created_at, created_at
FROM payment_ton_transaction`)
			return err
		}); err != nil {
			return fmt.Errorf("payment: migrate TON transactions: %w", err)
		}
	}

	if transactionExists {
		if err := r.dropTable(ctx, "payment_ton_transaction"); err != nil {
			return err
		}
	}
	if cursorExists {
		if err := r.dropTable(ctx, "payment_ton_wallet_cursor"); err != nil {
			return err
		}
	}
	return nil
}

func (r *PaymentRepository) tableExists(ctx context.Context, tableName string) (bool, error) {
	var count int
	err := r.db.DB().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = ?`, tableName).Scan(&count)
	return count > 0, err
}

func (r *PaymentRepository) dropTable(ctx context.Context, tableName string) error {
	if tableName != "payment_ton_transaction" && tableName != "payment_ton_wallet_cursor" {
		return fmt.Errorf("payment: unsupported legacy table %q", tableName)
	}
	return sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
		_, err := r.db.DB().ExecContext(ctx, "DROP TABLE `"+tableName+"`")
		return err
	})
}

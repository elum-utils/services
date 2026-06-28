package sql

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

var sqlIdentifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// EnsureColumn adds a missing column to an existing table.
func EnsureColumn(ctx context.Context, db *Client, timeout time.Duration, table, column, definition string) error {
	if db == nil || db.db == nil {
		return ErrNilDB
	}
	if !sqlIdentifierRe.MatchString(table) {
		return fmt.Errorf("invalid table identifier %q", table)
	}
	if !sqlIdentifierRe.MatchString(column) {
		return fmt.Errorf("invalid column identifier %q", column)
	}

	qctx, cancel := db.queryContext(ctx, timeout)
	defer cancel()

	var count int
	if err := db.db.QueryRowContext(qctx, `
SELECT COUNT(*)
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
  AND COLUMN_NAME = ?
`, table, column).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := db.db.ExecContext(qctx, fmt.Sprintf(
		"ALTER TABLE `%s` ADD COLUMN `%s` %s",
		table,
		column,
		definition,
	))
	return err
}

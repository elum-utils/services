package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
)

type Repository struct {
	db           *sqlwrap.Client
	q            *tasksqlc.Queries
	callbacks    *callbackutil.Store
	executor     tasksqlc.DBTX
	queryTimeout time.Duration
	cacheL1Delay time.Duration
	cacheL2Delay time.Duration
}

const DefaultQueryTimeout = time.Second
const bootstrapQueryTimeout = 30 * time.Second

type Options struct {
	QueryTimeout time.Duration
	CacheL1Delay time.Duration
	CacheL2Delay time.Duration
}

func New(db *sqlwrap.Client) *Repository {
	return NewWithOptions(db, Options{
		CacheL1Delay: 10 * time.Minute,
		CacheL2Delay: 10 * time.Minute,
	})
}

func NewWithOptions(db *sqlwrap.Client, options Options) *Repository {
	queryTimeout := options.QueryTimeout
	if queryTimeout <= 0 {
		queryTimeout = DefaultQueryTimeout
	}
	executor := db.WithQueryTimeout(queryTimeout)
	return &Repository{
		db: db, q: tasksqlc.New(executor), callbacks: callbackutil.NewWithTable(db.DB(), callbackutil.TasksTable),
		executor: executor, queryTimeout: queryTimeout,
		cacheL1Delay: options.CacheL1Delay, cacheL2Delay: options.CacheL2Delay,
	}
}

func NewPrepared(ctx context.Context, db *sqlwrap.Client) (*Repository, error) {
	return NewPreparedWithOptions(ctx, db, Options{})
}

func NewPreparedWithOptions(_ context.Context, db *sqlwrap.Client, options Options) (*Repository, error) {
	queryTimeout := options.QueryTimeout
	if queryTimeout <= 0 {
		queryTimeout = DefaultQueryTimeout
	}
	executor := db.WithQueryTimeout(queryTimeout)
	return &Repository{
		db: db, q: tasksqlc.New(executor), callbacks: callbackutil.NewWithTable(db.DB(), callbackutil.TasksTable),
		executor: executor, queryTimeout: queryTimeout,
		cacheL1Delay: options.CacheL1Delay, cacheL2Delay: options.CacheL2Delay,
	}, nil
}

func (r *Repository) Close() error {
	if r == nil {
		return nil
	}
	var err error
	if r.q != nil {
		err = errors.Join(err, r.q.Close())
	}
	if r.callbacks != nil {
		err = errors.Join(err, r.callbacks.Close())
	}
	return err
}

func (r *Repository) WithTx(ctx context.Context, fn func(*Repository) error) error {
	_, err := sqlwrap.Transaction(ctx, r.db, sqlwrap.Params{Timeout: r.queryTimeout}, func(ctx context.Context, tx *sql.Tx) (struct{}, error) {
		txRepo := &Repository{
			db: r.db, q: r.q.WithTx(tx), callbacks: r.callbacks.WithTx(tx),
			executor: tx, queryTimeout: r.queryTimeout,
			cacheL1Delay: r.cacheL1Delay, cacheL2Delay: r.cacheL2Delay,
		}
		return struct{}{}, fn(txRepo)
	})
	return err
}

func (r *Repository) Bootstrap(ctx context.Context) error {
	if err := r.migrateLegacySchema(ctx); err != nil {
		return err
	}
	if err := r.applySQL(ctx, tasksqlc.SchemaSQL, "schema"); err != nil {
		return err
	}
	if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
		return callbackutil.BootstrapTable(ctx, r.db.DB(), callbackutil.TasksTable)
	}); err != nil {
		return err
	}
	if err := r.applySQL(ctx, tasksqlc.TriggerSQL, "trigger"); err != nil {
		return err
	}
	return r.applySQL(ctx, tasksqlc.EventSQL, "event")
}

func (r *Repository) migrateLegacySchema(ctx context.Context) error {
	var workspaceColumnCount int
	if err := r.db.DB().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = 'task_group'
  AND column_name = 'workspace_id'`).Scan(&workspaceColumnCount); err != nil {
		return fmt.Errorf("tasks: inspect schema version: %w", err)
	}
	if workspaceColumnCount > 0 {
		return nil
	}

	legacyTables := []string{
		"task",
		"task_action_advertisement",
		"task_action_app",
		"task_action_channel",
		"task_group",
		"task_integration",
		"task_integration_attempt",
		"task_integration_field",
		"task_progress",
		"task_progress_event",
		"task_reward",
		"task_reward_delivery",
		"task_sequence",
	}
	renames := make([]string, 0, len(legacyTables))
	for _, table := range legacyTables {
		var sourceExists int
		if err := r.db.DB().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name = ?`, table).Scan(&sourceExists); err != nil {
			return fmt.Errorf("tasks: inspect legacy table %s: %w", table, err)
		}
		if sourceExists == 0 {
			continue
		}

		target := "legacy_" + table
		var targetExists int
		if err := r.db.DB().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name = ?`, target).Scan(&targetExists); err != nil {
			return fmt.Errorf("tasks: inspect legacy target %s: %w", target, err)
		}
		if targetExists > 0 {
			return fmt.Errorf("tasks: legacy migration target %s already exists", target)
		}
		renames = append(renames, quoteIdentifier(table)+" TO "+quoteIdentifier(target))
	}
	if len(renames) == 0 {
		return nil
	}
	if _, err := r.db.DB().ExecContext(ctx, "RENAME TABLE "+strings.Join(renames, ", ")); err != nil {
		return fmt.Errorf("tasks: rename legacy schema: %w", err)
	}
	return nil
}

func (r *Repository) applySQL(ctx context.Context, raw, source string) error {
	for _, statement := range splitSQLStatements(raw) {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, statement)
			return err
		}); err != nil {
			return fmt.Errorf("tasks %s SQL statement failed: %w\n%s", source, err, statement)
		}
	}
	return nil
}

func quoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func splitSQLStatements(raw string) []string {
	parts := strings.Split(raw, ";")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if stmt := strings.TrimSpace(part); stmt != "" {
			result = append(result, stmt)
		}
	}
	return result
}

func normalizePage(limit, offset int32) (int32, int32) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

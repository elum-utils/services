package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	calendarsqlc "github.com/elum-utils/services/calendar/sqlc"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

type Repository struct {
	db        *sqlwrap.Client
	q         *calendarsqlc.Queries
	callbacks *callbackutil.Store
	executor  calendarsqlc.DBTX
	timeout   time.Duration
	cacheL1   time.Duration
	cacheL2   time.Duration
}

type Options struct {
	QueryTimeout time.Duration
	CacheL1Delay time.Duration
	CacheL2Delay time.Duration
}

const bootstrapQueryTimeout = 30 * time.Second

func New(db *sqlwrap.Client) *Repository {
	return NewWithOptions(db, Options{
		CacheL1Delay: 10 * time.Minute,
		CacheL2Delay: 10 * time.Minute,
	})
}

func NewWithOptions(db *sqlwrap.Client, options Options) *Repository {
	timeout := queryTimeout(options.QueryTimeout)
	executor := db.WithQueryTimeout(timeout)
	q := calendarsqlc.New(executor)
	return &Repository{
		db: db, q: q, callbacks: callbackutil.NewWithTable(db.DB(), callbackutil.CalendarTable),
		executor: executor, timeout: timeout, cacheL1: options.CacheL1Delay, cacheL2: options.CacheL2Delay,
	}
}

func NewPrepared(ctx context.Context, db *sqlwrap.Client) (*Repository, error) {
	return NewPreparedWithOptions(ctx, db, Options{})
}

func NewPreparedWithOptions(_ context.Context, db *sqlwrap.Client, options Options) (*Repository, error) {
	return NewWithOptions(db, options), nil
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
	_, err := sqlwrap.Transaction(ctx, r.db, sqlwrap.Params{Timeout: r.timeout}, func(ctx context.Context, tx *sql.Tx) (struct{}, error) {
		txRepo := &Repository{
			db: r.db, q: r.q.WithTx(tx), callbacks: r.callbacks.WithTx(tx),
			executor: tx, timeout: r.timeout, cacheL1: r.cacheL1, cacheL2: r.cacheL2,
		}
		return struct{}{}, fn(txRepo)
	})
	return err
}

func (r *Repository) Bootstrap(ctx context.Context) error {
	if err := r.applySQL(ctx, calendarsqlc.SchemaSQL, "schema"); err != nil {
		return err
	}
	if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
		return callbackutil.BootstrapTable(ctx, r.db.DB(), callbackutil.CalendarTable)
	}); err != nil {
		return err
	}
	if err := r.applySQL(ctx, calendarsqlc.TriggerSQL, "trigger"); err != nil {
		return err
	}
	return r.applySQL(ctx, calendarsqlc.EventSQL, "event")
}

func (r *Repository) applySQL(ctx context.Context, raw, source string) error {
	for _, statement := range splitSQLStatements(raw) {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, statement)
			return err
		}); err != nil {
			return fmt.Errorf("calendar %s SQL statement failed: %w\n%s", source, err, statement)
		}
	}
	return nil
}

func queryTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return time.Second
	}
	return value
}

func splitSQLStatements(raw string) []string {
	parts := strings.Split(raw, ";")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if statement := strings.TrimSpace(part); statement != "" {
			result = append(result, statement)
		}
	}
	return result
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

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

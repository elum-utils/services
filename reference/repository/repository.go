package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	refsqlc "github.com/elum-utils/services/reference/sqlc"
)

var (
	ErrWorkspaceRequired = errors.New("reference: workspace is required")
	ErrItemNotFound      = errors.New("reference: item not found")
)

const bootstrapQueryTimeout = 30 * time.Second

type Options struct {
	QueryTimeout time.Duration
	CacheL1Delay time.Duration
	CacheL2Delay time.Duration
}

type Repository struct {
	db      *sqlwrap.Client
	q       *refsqlc.Queries
	timeout time.Duration
	cacheL1 time.Duration
	cacheL2 time.Duration
}

func New(db *sqlwrap.Client) *Repository {
	return NewWithOptions(db, Options{
		CacheL1Delay: 10 * time.Minute,
		CacheL2Delay: 10 * time.Minute,
	})
}

func NewWithOptions(db *sqlwrap.Client, options Options) *Repository {
	timeout := options.QueryTimeout
	if timeout <= 0 {
		timeout = time.Second
	}
	return &Repository{
		db: db, q: refsqlc.New(db.WithQueryTimeout(timeout)),
		timeout: timeout, cacheL1: options.CacheL1Delay, cacheL2: options.CacheL2Delay,
	}
}

func NewPreparedWithOptions(ctx context.Context, db *sqlwrap.Client, options Options) (*Repository, error) {
	repository := NewWithOptions(db, options)
	q, err := refsqlc.Prepare(ctx, db.WithQueryTimeout(repository.timeout))
	if err != nil {
		return nil, err
	}
	repository.q = q
	return repository, nil
}

func (r *Repository) Close() error {
	if r == nil || r.q == nil {
		return nil
	}
	return r.q.Close()
}

func (r *Repository) Bootstrap(ctx context.Context) error {
	for _, statement := range splitSQLStatements(refsqlc.SchemaSQL) {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, statement)
			return err
		}); err != nil {
			return fmt.Errorf("reference schema statement failed: %w\n%s", err, statement)
		}
	}
	for _, statement := range splitSQLStatements(refsqlc.TriggerSQL) {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, statement)
			return err
		}); err != nil {
			return fmt.Errorf("reference trigger statement failed: %w\n%s", err, statement)
		}
	}
	return nil
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

func requireWorkspace(workspaceID string) error {
	if strings.TrimSpace(workspaceID) == "" {
		return ErrWorkspaceRequired
	}
	return nil
}

func mapNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrItemNotFound
	}
	return err
}

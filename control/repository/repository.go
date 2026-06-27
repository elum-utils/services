package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	controlsqlc "github.com/elum-utils/services/control/sqlc"
	serviceerrors "github.com/elum-utils/services/errors"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

var (
	ErrNotFound          = serviceerrors.New(serviceerrors.CodeNotFound, "control entity not found")
	ErrInvalidScope      = serviceerrors.New(serviceerrors.CodeInvalidFields, "control workspace or account is required")
	ErrForbidden         = serviceerrors.New(serviceerrors.CodeForbidden, "control access denied")
	ErrRoleHierarchy     = serviceerrors.New(serviceerrors.CodeForbidden, "control role hierarchy denied")
	ErrMethodNotFound    = serviceerrors.New(serviceerrors.CodeNotFound, "control method not found")
	ErrMethodOwner       = serviceerrors.New(serviceerrors.CodeConflict, "control method belongs to another service")
	ErrRoleNotFound      = serviceerrors.New(serviceerrors.CodeNotFound, "control role not found")
	ErrAccountNotFound   = serviceerrors.New(serviceerrors.CodeNotFound, "control account not found")
	ErrWorkspaceNotFound = serviceerrors.New(serviceerrors.CodeNotFound, "control workspace not found")
)

const bootstrapQueryTimeout = 30 * time.Second

type Options struct {
	QueryTimeout time.Duration
	CacheL1Delay time.Duration
	CacheL2Delay time.Duration
}

type Repository struct {
	db      *sqlwrap.Client
	q       *controlsqlc.Queries
	timeout time.Duration
	cacheL1 time.Duration
	cacheL2 time.Duration
}

func New(db *sqlwrap.Client) *Repository { return NewWithOptions(db, Options{}) }

func NewWithOptions(db *sqlwrap.Client, options Options) *Repository {
	timeout := options.QueryTimeout
	if timeout <= 0 {
		timeout = time.Second
	}
	cacheL1, cacheL2 := options.CacheL1Delay, options.CacheL2Delay
	if cacheL1 <= 0 {
		cacheL1 = time.Second
	}
	if cacheL2 <= 0 {
		cacheL2 = time.Second
	}
	return &Repository{db: db, q: controlsqlc.New(db.WithQueryTimeout(timeout)), timeout: timeout, cacheL1: cacheL1, cacheL2: cacheL2}
}

func (r *Repository) Close() error {
	if r == nil || r.q == nil {
		return nil
	}
	return r.q.Close()
}

func (r *Repository) Bootstrap(ctx context.Context) error {
	if err := r.execBootstrapSQL(ctx, controlsqlc.SchemaSQL, "schema"); err != nil {
		return err
	}
	if err := r.execBootstrapSQL(ctx, controlsqlc.CatalogSQL, "catalog"); err != nil {
		return err
	}
	return r.db.BumpCacheVersion("control", "access-catalog")
}

func (r *Repository) execBootstrapSQL(ctx context.Context, raw, name string) error {
	for _, statement := range splitSQLStatements(raw) {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, statement)
			return err
		}); err != nil {
			return fmt.Errorf("control %s statement failed: %w\n%s", name, err, statement)
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

func normalizeID(value string) string { return strings.TrimSpace(value) }

func required(values ...string) error {
	for _, value := range values {
		if normalizeID(value) == "" {
			return ErrInvalidScope
		}
	}
	return nil
}

func noRows(err error, fallback error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fallback
	}
	return err
}

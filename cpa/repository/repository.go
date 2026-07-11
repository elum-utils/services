package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	cpasqlc "github.com/elum-utils/services/cpa/sqlc"
	serviceerrors "github.com/elum-utils/services/errors"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

var (
	ErrWorkspaceRequired = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa workspace id is required")
	ErrOfferRequired     = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa offer id is required")
	ErrLocaleRequired    = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa locale is required")
	ErrRewardKeyRequired = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa reward key is required")
	ErrCodeRequired      = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa code is required")
	ErrInvalidDateRange  = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa date range is invalid")
	ErrNoCodesAvailable  = serviceerrors.New(serviceerrors.CodeUnavailable, "cpa personal codes are not available")
	ErrInvalidCodeConfig = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa generated code configuration is invalid")
)

type Repository struct {
	db                       *sqlwrap.Client
	q                        *cpasqlc.Queries
	callbacks                *callbackutil.Store
	executor                 cpasqlc.DBTX
	timeout                  time.Duration
	cacheL1                  time.Duration
	cacheL2                  time.Duration
	onCacheInvalidationError func(error)
}

type Options struct {
	QueryTimeout             time.Duration
	CacheL1Delay             time.Duration
	CacheL2Delay             time.Duration
	OnCacheInvalidationError func(error)
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
	return &Repository{
		db:                       db,
		q:                        cpasqlc.New(executor),
		callbacks:                callbackutil.NewWithTable(db.DB(), callbackutil.CPATable),
		executor:                 executor,
		timeout:                  timeout,
		cacheL1:                  options.CacheL1Delay,
		cacheL2:                  options.CacheL2Delay,
		onCacheInvalidationError: options.OnCacheInvalidationError,
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
	_, err := sqlwrap.Transaction(
		ctx,
		r.db,
		sqlwrap.Params{
			Timeout: r.timeout,
		},
		func(ctx context.Context, tx *sql.Tx) (struct{}, error) {
			txRepo := &Repository{
				db:                       r.db,
				q:                        r.q.WithTx(tx),
				callbacks:                r.callbacks.WithTx(tx),
				executor:                 tx,
				timeout:                  r.timeout,
				cacheL1:                  r.cacheL1,
				cacheL2:                  r.cacheL2,
				onCacheInvalidationError: r.onCacheInvalidationError,
			}
			return struct{}{}, fn(txRepo)
		},
	)
	return err
}

func (r *Repository) Bootstrap(ctx context.Context) error {
	if err := r.applySQL(ctx, cpasqlc.SchemaSQL, "schema"); err != nil {
		return err
	}
	if err := r.applySchemaUpgrades(ctx); err != nil {
		return err
	}
	if err := sqlwrap.Exec(
		ctx,
		r.db,
		sqlwrap.Params{
			Timeout: bootstrapQueryTimeout,
		},
		func(ctx context.Context) error {
			return callbackutil.BootstrapTable(ctx, r.db.DB(), callbackutil.CPATable)
		},
	); err != nil {
		return err
	}
	return r.applySQL(ctx, cpasqlc.EventSQL, "event")
}

func (r *Repository) applySchemaUpgrades(ctx context.Context) error {
	return nil
}

func (r *Repository) applySQL(ctx context.Context, raw, source string) error {
	statements, err := sqlwrap.SplitStatements(raw)
	if err != nil {
		return fmt.Errorf("cpa %s SQL parse failed: %w", source, err)
	}
	for _, statement := range statements {
		if err := sqlwrap.Exec(
			ctx,
			r.db,
			sqlwrap.Params{
				Timeout: bootstrapQueryTimeout,
			},
			func(ctx context.Context) error {
				_, err := r.db.DB().ExecContext(ctx, statement)
				return err
			},
		); err != nil {
			if isCreateTypeAlreadyExists(statement, err) {
				continue
			}
			return fmt.Errorf("cpa %s SQL statement failed: %w\n%s", source, err, statement)
		}
	}
	return nil
}

func isCreateTypeAlreadyExists(statement string, err error) bool {
	var pgErr *pgconn.PgError
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(statement)), "CREATE TYPE ") &&
		errors.As(err, &pgErr) &&
		pgErr.Code == "42710"
}

func queryTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return time.Second
	}
	return value
}

func requireScope(workspaceID, cpaID string) error {
	if workspaceID == "" {
		return ErrWorkspaceRequired
	}
	if cpaID == "" {
		return ErrOfferRequired
	}
	return nil
}

func requireLocale(locale string) error {
	if strings.TrimSpace(locale) == "" {
		return ErrLocaleRequired
	}
	return nil
}

func requireRewardKey(rewardKey string) error {
	if strings.TrimSpace(rewardKey) == "" {
		return ErrRewardKeyRequired
	}
	return nil
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

type PaymentRepository struct {
	db        *sqlwrap.Client
	q         *paymentsqlc.Queries
	callbacks *callbackutil.Store
	inTx      bool
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

var ErrWorkspaceRequired = errors.New("payment: workspace id is required")

func NewPaymentRepository(db *sqlwrap.Client) *PaymentRepository {
	return NewPaymentRepositoryWithOptions(db, Options{
		CacheL1Delay: 10 * time.Minute,
		CacheL2Delay: 10 * time.Minute,
	})
}

func NewPaymentRepositoryWithOptions(db *sqlwrap.Client, options Options) *PaymentRepository {
	timeout := queryTimeout(options.QueryTimeout)
	return &PaymentRepository{
		db:        db,
		q:         paymentsqlc.New(db.WithQueryTimeout(timeout)),
		callbacks: callbackutil.NewWithTable(db.DB(), callbackutil.PaymentTable),
		timeout:   timeout,
		cacheL1:   options.CacheL1Delay,
		cacheL2:   options.CacheL2Delay,
	}
}

func NewPreparedPaymentRepository(ctx context.Context, db *sqlwrap.Client) (*PaymentRepository, error) {
	return NewPreparedPaymentRepositoryWithOptions(ctx, db, Options{})
}

func NewPreparedPaymentRepositoryWithOptions(_ context.Context, db *sqlwrap.Client, options Options) (*PaymentRepository, error) {
	return NewPaymentRepositoryWithOptions(db, options), nil
}

func (r *PaymentRepository) Close() error {
	if r == nil || r.q == nil {
		return nil
	}
	var callbackErr error
	if r.callbacks != nil {
		callbackErr = r.callbacks.Close()
	}
	return errors.Join(r.q.Close(), callbackErr)
}

func (r *PaymentRepository) WithTx(ctx context.Context, fn func(*PaymentRepository) error) error {
	_, err := sqlwrap.Transaction(ctx, r.db, sqlwrap.Params{Timeout: r.timeout}, func(ctx context.Context, tx *sql.Tx) (struct{}, error) {
		txRepo := &PaymentRepository{
			db:        r.db,
			q:         r.q.WithTx(tx),
			callbacks: r.callbacks.WithTx(tx),
			inTx:      true,
			timeout:   r.timeout,
			cacheL1:   r.cacheL1,
			cacheL2:   r.cacheL2,
		}
		return struct{}{}, fn(txRepo)
	})
	return err
}

func (r *PaymentRepository) inTransaction(ctx context.Context, fn func(*PaymentRepository) error) error {
	if r.inTx {
		return fn(r)
	}
	return r.WithTx(ctx, fn)
}

func (r *PaymentRepository) Bootstrap(ctx context.Context, schemaPath ...string) error {
	raw := paymentsqlc.SchemaSQL
	if len(schemaPath) > 0 && strings.TrimSpace(schemaPath[0]) != "" {
		data, err := os.ReadFile(schemaPath[0])
		if err != nil {
			return err
		}
		raw = string(data)
	}

	for _, stmt := range splitSQLStatements(raw) {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, stmt)
			return err
		}); err != nil {
			return fmt.Errorf("statement failed: %w\n%s", err, stmt)
		}
	}

	if err := r.migrateDynamicPricing(ctx); err != nil {
		return err
	}

	if err := r.migrateLegacyTONStorage(ctx); err != nil {
		return err
	}

	if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
		return callbackutil.BootstrapTable(ctx, r.db.DB(), callbackutil.PaymentTable)
	}); err != nil {
		return err
	}
	if err := r.applySQL(ctx, paymentsqlc.TriggerSQL, "trigger"); err != nil {
		return err
	}
	return r.applySQL(ctx, paymentsqlc.EventSQL, "event")
}

func (r *PaymentRepository) applySQL(ctx context.Context, raw, source string) error {
	for _, statement := range splitSQLStatements(raw) {
		if err := sqlwrap.Exec(ctx, r.db, sqlwrap.Params{Timeout: bootstrapQueryTimeout}, func(ctx context.Context) error {
			_, err := r.db.DB().ExecContext(ctx, statement)
			return err
		}); err != nil {
			return fmt.Errorf("payment %s SQL statement failed: %w\n%s", source, err, statement)
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

func (r *PaymentRepository) ListProviders(ctx context.Context) ([]paymentsqlc.PaymentProvider, error) {
	key := paymentCacheKey("providers")
	providers, err := queryPaymentCache(ctx, r, paymentGlobalCacheScope, key, func(ctx context.Context) ([]paymentsqlc.PaymentProvider, error) {
		return r.q.ListProviders(ctx)
	})
	if err != nil {
		return nil, err
	}
	return cloneSlice(providers), nil
}

func (r *PaymentRepository) ListAssets(ctx context.Context) ([]paymentsqlc.PaymentAsset, error) {
	key := paymentCacheKey("assets")
	assets, err := queryPaymentCache(ctx, r, paymentGlobalCacheScope, key, func(ctx context.Context) ([]paymentsqlc.PaymentAsset, error) {
		return r.q.ListAssets(ctx)
	})
	if err != nil {
		return nil, err
	}
	return cloneSlice(assets), nil
}

type AssetUpsertParams struct {
	Code            string
	Title           string
	AssetKind       paymentsqlc.PaymentAssetAssetKind
	Scale           uint16
	Chain           *string
	Network         *string
	ContractAddress *string
	IsActive        bool
}

func (r *PaymentRepository) UpsertAsset(ctx context.Context, params AssetUpsertParams) error {
	if err := r.q.UpsertAsset(ctx, paymentsqlc.UpsertAssetParams{
		Code:      params.Code,
		Title:     params.Title,
		AssetKind: params.AssetKind,
		Scale:     params.Scale,
		Chain: sqlwrap.NullFromPtr(params.Chain, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		Network: sqlwrap.NullFromPtr(params.Network, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		ContractAddress: sqlwrap.NullFromPtr(params.ContractAddress, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		IsActive: params.IsActive,
	}); err != nil {
		return err
	}
	return r.invalidateAllCache()
}

func (r *PaymentRepository) DeleteAsset(ctx context.Context, code string) (int64, error) {
	rows, err := r.q.DeleteAsset(ctx, code)
	if err != nil {
		return 0, err
	}
	return rows, r.invalidateAllCache()
}

func (r *PaymentRepository) GetProviderAsset(ctx context.Context, providerCode string, assetCode string) (paymentsqlc.PaymentProviderAsset, error) {
	key := paymentCacheKey("provider_asset", providerCode, assetCode)
	return queryPaymentCache(ctx, r, paymentGlobalCacheScope, key, func(ctx context.Context) (paymentsqlc.PaymentProviderAsset, error) {
		return r.q.GetProviderAsset(ctx, paymentsqlc.GetProviderAssetParams{
			ProviderCode: providerCode,
			AssetCode:    assetCode,
		})
	})
}

type ProviderAssetUpsertParams struct {
	ProviderCode    string
	AssetCode       string
	MinAmountMinor  *int64
	MaxAmountMinor  *int64
	MerchantAccount *string
	IsActive        bool
}

func (r *PaymentRepository) UpsertProviderAsset(ctx context.Context, params ProviderAssetUpsertParams) error {
	if err := r.q.UpsertProviderAsset(ctx, paymentsqlc.UpsertProviderAssetParams{
		ProviderCode: params.ProviderCode,
		AssetCode:    params.AssetCode,
		MinAmountMinor: sqlwrap.NullFromPtr(params.MinAmountMinor, func(v int64) sql.NullInt64 {
			return sql.NullInt64{Int64: v, Valid: true}
		}),
		MaxAmountMinor: sqlwrap.NullFromPtr(params.MaxAmountMinor, func(v int64) sql.NullInt64 {
			return sql.NullInt64{Int64: v, Valid: true}
		}),
		MerchantAccount: sqlwrap.NullFromPtr(params.MerchantAccount, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		IsActive: params.IsActive,
	}); err != nil {
		return err
	}
	return r.invalidateAllCache()
}

func (r *PaymentRepository) DeleteProviderAsset(ctx context.Context, providerCode string, assetCode string) (int64, error) {
	rows, err := r.q.DeleteProviderAsset(ctx, paymentsqlc.DeleteProviderAssetParams{
		ProviderCode: providerCode,
		AssetCode:    assetCode,
	})
	if err != nil {
		return 0, err
	}
	return rows, r.invalidateAllCache()
}

func splitSQLStatements(raw string) []string {
	parts := strings.Split(raw, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		statements = append(statements, stmt)
	}
	return statements
}

func requireWorkspaceID(workspaceID string) (string, error) {
	if workspaceID == "" {
		return "", ErrWorkspaceRequired
	}
	return workspaceID, nil
}

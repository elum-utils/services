package repository

import (
	"context"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (r *PaymentRepository) GetAsset(ctx context.Context, code string) (paymentsqlc.PaymentAsset, error) {
	key := paymentCacheKey("asset", code)
	return queryPaymentCache(ctx, r, paymentGlobalCacheScope, key, func(ctx context.Context) (paymentsqlc.PaymentAsset, error) {
		return r.q.GetAsset(ctx, code)
	})
}

func (r *PaymentRepository) GetAssetByChainContract(ctx context.Context, params paymentsqlc.GetAssetByChainContractParams) (paymentsqlc.PaymentAsset, error) {
	key := paymentCacheKey("asset_chain_contract", params.Chain, params.Network, params.ContractAddress)
	return queryPaymentCache(ctx, r, paymentGlobalCacheScope, key, func(ctx context.Context) (paymentsqlc.PaymentAsset, error) {
		return r.q.GetAssetByChainContract(ctx, params)
	})
}

func (r *PaymentRepository) GetProviderCursor(ctx context.Context, params paymentsqlc.GetProviderCursorParams) (paymentsqlc.PaymentProviderCursor, error) {
	return r.q.GetProviderCursor(ctx, params)
}

func (r *PaymentRepository) UpsertProviderCursor(ctx context.Context, params paymentsqlc.UpsertProviderCursorParams) (int64, error) {
	return r.q.UpsertProviderCursor(ctx, params)
}

func (r *PaymentRepository) GetProviderTransactionByExternalID(ctx context.Context, params paymentsqlc.GetProviderTransactionByExternalIDParams) (paymentsqlc.PaymentProviderTransaction, error) {
	return r.q.GetProviderTransactionByExternalID(ctx, params)
}

func (r *PaymentRepository) CreateProviderTransaction(ctx context.Context, params paymentsqlc.CreateProviderTransactionParams) (uint64, error) {
	id, err := r.q.CreateProviderTransaction(ctx, params)
	return uint64(id), err
}

func (r *PaymentRepository) StoreProviderTransaction(
	ctx context.Context,
	transaction paymentsqlc.CreateProviderTransactionParams,
	cursor paymentsqlc.UpsertProviderCursorParams,
) (uint64, error) {
	var id uint64
	err := r.WithTx(ctx, func(tx *PaymentRepository) error {
		var err error
		id, err = tx.CreateProviderTransaction(ctx, transaction)
		if err != nil {
			return err
		}
		_, err = tx.UpsertProviderCursor(ctx, cursor)
		return err
	})
	return id, err
}

func (r *PaymentRepository) AdminListProviderCursors(ctx context.Context, params paymentsqlc.AdminListProviderCursorsParams) ([]paymentsqlc.PaymentProviderCursor, error) {
	return r.q.AdminListProviderCursors(ctx, params)
}

func (r *PaymentRepository) AdminListProviderTransactions(ctx context.Context, params paymentsqlc.AdminListProviderTransactionsParams) ([]paymentsqlc.PaymentProviderTransaction, error) {
	return r.q.AdminListProviderTransactions(ctx, params)
}

func (r *PaymentRepository) AdminGetProviderTransaction(ctx context.Context, params paymentsqlc.AdminGetProviderTransactionParams) (paymentsqlc.PaymentProviderTransaction, error) {
	return r.q.AdminGetProviderTransaction(ctx, params)
}

func (r *PaymentRepository) AdminUpdateProviderTransactionStatus(ctx context.Context, params paymentsqlc.AdminUpdateProviderTransactionStatusParams) (int64, error) {
	return r.q.AdminUpdateProviderTransactionStatus(ctx, params)
}

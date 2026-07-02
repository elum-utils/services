package repository

import (
	"context"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (r *PaymentRepository) UpsertTONWallet(ctx context.Context, params paymentsqlc.UpsertTONWalletParams) error {
	return r.q.UpsertTONWallet(ctx, params)
}

func (r *PaymentRepository) DeleteTONWallet(ctx context.Context, params paymentsqlc.DeleteTONWalletParams) (int64, error) {
	return r.q.DeleteTONWallet(ctx, params)
}

func (r *PaymentRepository) AdminGetTONWallet(ctx context.Context, params paymentsqlc.AdminGetTONWalletParams) (paymentsqlc.PaymentTonWallet, error) {
	return r.q.AdminGetTONWallet(ctx, params)
}

func (r *PaymentRepository) AdminListTONWallets(ctx context.Context, params paymentsqlc.AdminListTONWalletsParams) ([]paymentsqlc.PaymentTonWallet, error) {
	return r.q.AdminListTONWallets(ctx, params)
}

func (r *PaymentRepository) ListEnabledTONWallets(ctx context.Context) ([]paymentsqlc.PaymentTonWallet, error) {
	return r.q.ListEnabledTONWallets(ctx)
}

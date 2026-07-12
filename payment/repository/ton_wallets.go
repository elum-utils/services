package repository

import (
	"context"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (r *PaymentRepository) UpsertTONWallet(ctx context.Context, params paymentsqlc.UpsertTONWalletParams) error {
	return r.q.UpsertTONWallet(ctx, params)
}

func (r *PaymentRepository) DeleteTONWallet(ctx context.Context, workspaceID string) (int64, error) {
	return r.q.DeleteTONWallet(ctx, workspaceID)
}

func (r *PaymentRepository) AdminGetTONWallet(
	ctx context.Context,
	workspaceID string,
) (paymentsqlc.PaymentTonWallet, error) {
	return r.q.AdminGetTONWallet(ctx, workspaceID)
}

func (r *PaymentRepository) ListEnabledTONWallets(ctx context.Context) ([]paymentsqlc.PaymentTonWallet, error) {
	return r.q.ListEnabledTONWallets(ctx)
}

func (r *PaymentRepository) GetEnabledTONWalletForWorkspace(
	ctx context.Context,
	workspaceID string,
) (paymentsqlc.PaymentTonWallet, error) {
	return r.q.GetEnabledTONWalletForWorkspace(ctx, workspaceID)
}

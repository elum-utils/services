package admin

import (
	"context"
	"database/sql"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (a *Admin) SaveTONWallet(ctx context.Context, params TONWalletUpsertParams) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	return a.repository.UpsertTONWallet(mergedCtx, paymentsqlc.UpsertTONWalletParams{
		WorkspaceID:   params.WorkspaceID,
		Network:       params.Network,
		WalletAddress: params.WalletAddress,
		NetworkConfigUrl: sqlwrap.NullFromPtr(params.NetworkConfigURL, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		IsEnabled: params.IsEnabled,
	})
}

func (a *Admin) DeleteTONWallet(ctx context.Context, workspaceID string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	return a.repository.DeleteTONWallet(mergedCtx, workspaceID)
}

func (a *Admin) GetTONWallet(ctx context.Context, workspaceID string) (paymentsqlc.PaymentTonWallet, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	return a.repository.AdminGetTONWallet(mergedCtx, workspaceID)
}

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

func (a *Admin) DeleteTONWallet(ctx context.Context, workspaceID string, network string, walletAddress string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	return a.repository.DeleteTONWallet(mergedCtx, paymentsqlc.DeleteTONWalletParams{
		WorkspaceID:   workspaceID,
		Network:       network,
		WalletAddress: walletAddress,
	})
}

func (a *Admin) GetTONWallet(ctx context.Context, workspaceID string, network string, walletAddress string) (paymentsqlc.PaymentTonWallet, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	return a.repository.AdminGetTONWallet(mergedCtx, paymentsqlc.AdminGetTONWalletParams{
		WorkspaceID:   workspaceID,
		Network:       network,
		WalletAddress: walletAddress,
	})
}

func (a *Admin) ListTONWallets(ctx context.Context, params TONWalletListParams) ([]paymentsqlc.PaymentTonWallet, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	limit, offset := normalizePage(params.Page)
	filterEnabled := params.IsEnabled != nil
	enabled := false
	if params.IsEnabled != nil {
		enabled = *params.IsEnabled
	}
	return a.repository.AdminListTONWallets(mergedCtx, paymentsqlc.AdminListTONWalletsParams{
		WorkspaceID: params.WorkspaceID,
		Column2:     params.Network,
		Network:     params.Network,
		Column4:     !filterEnabled,
		IsEnabled:   enabled,
		Limit:       limit,
		Offset:      offset,
	})
}

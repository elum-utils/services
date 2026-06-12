package admin

import (
	"context"
	"database/sql"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (a *Admin) ListProviderCursors(ctx context.Context, params ProviderCursorListParams) ([]paymentsqlc.PaymentProviderCursor, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListProviderCursors(mergedCtx, paymentsqlc.AdminListProviderCursorsParams{
		WorkspaceID:  params.WorkspaceID,
		Column2:      params.ProviderCode,
		ProviderCode: params.ProviderCode,
		Column4:      params.Network,
		Network:      params.Network,
		Limit:        limit,
		Offset:       offset,
	})
}

func (a *Admin) GetProviderCursor(ctx context.Context, workspaceID, providerCode, network, sourceKey string) (paymentsqlc.PaymentProviderCursor, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.GetProviderCursor(mergedCtx, paymentsqlc.GetProviderCursorParams{
		WorkspaceID: workspaceID, ProviderCode: providerCode,
		Network: network, SourceKey: sourceKey,
	})
}

func (a *Admin) UpsertProviderCursor(ctx context.Context, params paymentsqlc.UpsertProviderCursorParams) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.UpsertProviderCursor(mergedCtx, params)
}

func (a *Admin) ListProviderTransactions(ctx context.Context, params ProviderTransactionListParams) ([]paymentsqlc.PaymentProviderTransaction, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListProviderTransactions(mergedCtx, paymentsqlc.AdminListProviderTransactionsParams{
		WorkspaceID:  params.WorkspaceID,
		Column2:      params.ProviderCode,
		ProviderCode: params.ProviderCode,
		Column4:      params.Network,
		Network:      params.Network,
		Column6:      params.SourceKey,
		SourceKey:    params.SourceKey,
		Column8:      params.Status,
		Status:       paymentsqlc.PaymentProviderTransactionStatus(params.Status),
		Limit:        limit,
		Offset:       offset,
	})
}

func (a *Admin) GetProviderTransaction(ctx context.Context, workspaceID string, id uint64) (paymentsqlc.PaymentProviderTransaction, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.AdminGetProviderTransaction(mergedCtx, paymentsqlc.AdminGetProviderTransactionParams{
		WorkspaceID: workspaceID,
		ID:          id,
	})
}

func (a *Admin) GetProviderTransactionByExternalID(
	ctx context.Context,
	workspaceID, providerCode, network, sourceKey, externalTransactionID string,
) (paymentsqlc.PaymentProviderTransaction, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.GetProviderTransactionByExternalID(mergedCtx, paymentsqlc.GetProviderTransactionByExternalIDParams{
		WorkspaceID: workspaceID, ProviderCode: providerCode, Network: network,
		SourceKey: sourceKey, ExternalTransactionID: externalTransactionID,
	})
}

func (a *Admin) UpdateProviderTransactionStatus(ctx context.Context, workspaceID string, id uint64, status string, message string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.AdminUpdateProviderTransactionStatus(mergedCtx, paymentsqlc.AdminUpdateProviderTransactionStatusParams{
		WorkspaceID: workspaceID,
		ID:          id,
		Status:      paymentsqlc.PaymentProviderTransactionStatus(status),
		Error:       sql.NullString{String: message, Valid: message != ""},
	})
}

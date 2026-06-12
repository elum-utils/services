package admin

import (
	"context"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (a *Admin) ListProviders(ctx context.Context) ([]paymentsqlc.PaymentProvider, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.ListProviders(ctx)
}

func (a *Admin) GetProvider(ctx context.Context, code string) (paymentsqlc.PaymentProvider, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetProvider(ctx, code)
}

func (a *Admin) UpsertProvider(ctx context.Context, params ProviderUpsertParams) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminUpsertProvider(ctx, paymentsqlc.AdminUpsertProviderParams{
		Code:             params.Code,
		Title:            params.Title,
		ProviderKind:     params.ProviderKind,
		SupportsCreate:   params.SupportsCreate,
		SupportsRedirect: params.SupportsRedirect,
		SupportsWebhook:  params.SupportsWebhook,
		SupportsRefund:   params.SupportsRefund,
		IsActive:         params.IsActive,
	})
}

func (a *Admin) DeleteProvider(ctx context.Context, code string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminDeleteProvider(ctx, code)
}

func (a *Admin) ListAssets(ctx context.Context) ([]paymentsqlc.PaymentAsset, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.ListAssets(ctx)
}

func (a *Admin) GetAsset(ctx context.Context, code string) (paymentsqlc.PaymentAsset, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetAsset(ctx, code)
}

func (a *Admin) UpsertAsset(ctx context.Context, params paymentsqlc.UpsertAssetParams) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminUpsertAsset(ctx, params)
}

func (a *Admin) DeleteAsset(ctx context.Context, code string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminDeleteAsset(ctx, code)
}

func (a *Admin) ListProviderAssets(ctx context.Context, params ProviderAssetListParams) ([]paymentsqlc.PaymentProviderAsset, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListProviderAssets(ctx, paymentsqlc.AdminListProviderAssetsParams{
		Column1:      params.ProviderCode,
		ProviderCode: params.ProviderCode,
		Column3:      params.AssetCode,
		AssetCode:    params.AssetCode,
		Limit:        limit,
		Offset:       offset,
	})
}

func (a *Admin) GetProviderAsset(ctx context.Context, providerCode string, assetCode string) (paymentsqlc.PaymentProviderAsset, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.GetProviderAsset(ctx, providerCode, assetCode)
}

func (a *Admin) UpsertProviderAsset(ctx context.Context, params paymentsqlc.UpsertProviderAssetParams) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminUpsertProviderAsset(ctx, params)
}

func (a *Admin) DeleteProviderAsset(ctx context.Context, providerCode string, assetCode string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminDeleteProviderAsset(ctx, paymentsqlc.DeleteProviderAssetParams{
		ProviderCode: providerCode,
		AssetCode:    assetCode,
	})
}

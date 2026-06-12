package asset

import (
	"context"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (a *Asset) GetProvider(ctx context.Context, providerCode string, assetCode string) (paymentsqlc.PaymentProviderAsset, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	return a.repository.GetProviderAsset(ctx, providerCode, assetCode)
}

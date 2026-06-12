package asset

import (
	"context"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (a *Asset) List(ctx context.Context) ([]paymentsqlc.PaymentAsset, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	return a.repository.ListAssets(ctx)
}

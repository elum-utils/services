package product

import (
	"context"
	"time"

	"github.com/elum-utils/services/payment/repository"
)

type UpdatePriceParams struct {
	ID                  uint64
	WorkspaceID         string
	AssetCode           string
	ListAmountMinor     uint64
	DiscountAmountMinor uint64
	IsPromotion         bool
	StartsAt            *time.Time
	EndsAt              *time.Time
}

func (a *Product) UpdatePrice(ctx context.Context, params UpdatePriceParams) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	return a.repository.UpdateProductPrice(ctx, repository.ProductPriceUpdateParams{
		ID:                  params.ID,
		WorkspaceID:         params.WorkspaceID,
		AssetCode:           params.AssetCode,
		ListAmountMinor:     params.ListAmountMinor,
		DiscountAmountMinor: params.DiscountAmountMinor,
		IsPromotion:         params.IsPromotion,
		StartsAt:            params.StartsAt,
		EndsAt:              params.EndsAt,
	})
}

package product

import (
	"context"
	"time"

	"github.com/elum-utils/services/payment/repository"
)

type CreatePriceParams struct {
	WorkspaceID         string
	ProductID           string
	AssetCode           string
	ListAmountMinor     uint64
	DiscountAmountMinor uint64
	IsPromotion         bool
	StartsAt            *time.Time
	EndsAt              *time.Time
}

func (a *Product) CreatePrice(ctx context.Context, params CreatePriceParams) (uint64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	return a.repository.CreateProductPrice(ctx, repository.ProductPriceCreateParams{
		ProductID:           params.ProductID,
		WorkspaceID:         params.WorkspaceID,
		AssetCode:           params.AssetCode,
		ListAmountMinor:     params.ListAmountMinor,
		DiscountAmountMinor: params.DiscountAmountMinor,
		IsPromotion:         params.IsPromotion,
		StartsAt:            params.StartsAt,
		EndsAt:              params.EndsAt,
	})
}

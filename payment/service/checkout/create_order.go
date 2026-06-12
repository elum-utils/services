package checkout

import (
	"context"

	"github.com/elum-utils/services/payment/repository"
)

func (a *Checkout) CreateOrder(ctx context.Context, params CreateOrderParams) (*Order, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	order, err := a.repository.CreateOrder(ctx, repository.OrderCreateParams{
		AppID:               params.AppID,
		WorkspaceID:         params.WorkspaceID,
		PlatformID:          params.PlatformID,
		PlatformUserID:      params.PlatformUserID,
		InternalUserID:      params.InternalUserID,
		PayerPlatformID:     params.PayerPlatformID,
		PayerPlatformUserID: params.PayerPlatformUserID,
		PayerInternalUserID: params.PayerInternalUserID,
		ProductID:           params.ProductID,
		Quantity:            params.Quantity,
		AssetCode:           params.AssetCode,
		Locale:              params.Locale,
		ReservedUntil:       params.ReservedUntil,
		ExpiresAt:           params.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return mapOrder(order), nil
}

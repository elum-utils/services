package subscription

import (
	"context"

	"github.com/elum-utils/services/payment/repository"
)

func (a *Subscription) IsActive(ctx context.Context, params IsActiveParams) (bool, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	return a.repository.IsSubscriptionActive(ctx, repository.SubscriptionIsActiveParams{
		WorkspaceID:    params.Identity.WorkspaceID,
		PlatformID:     params.Identity.PlatformID,
		PlatformUserID: params.Identity.PlatformUserID,
		ProductID:      params.ProductID,
		ProviderCode:   params.ProviderCode,
	})
}

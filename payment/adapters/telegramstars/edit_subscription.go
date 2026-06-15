package telegramstars

import (
	"context"
)

func (a *TelegramStars) EditSubscription(ctx context.Context, params EditSubscriptionParams) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if a == nil {
		return ErrNotInitialized
	}
	return NewClient(params.Credentials).EditUserStarSubscription(ctx, editUserStarSubscriptionRequest{
		UserID:                  params.UserID,
		TelegramPaymentChargeID: params.TelegramPaymentChargeID,
		IsCanceled:              params.IsCanceled,
	})
}

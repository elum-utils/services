package telegramstars

import (
	"context"
	"errors"
)

func (a *TelegramStars) Execute(ctx context.Context, params RefundParams) (RefundResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if a == nil {
		return RefundResult{}, errors.New("telegram_stars: api is not initialized")
	}
	if err := NewClient(params.Credentials).RefundStarPayment(ctx, refundStarPaymentRequest{
		UserID:                  params.UserID,
		TelegramPaymentChargeID: params.TelegramPaymentChargeID,
	}); err != nil {
		return RefundResult{}, err
	}
	return RefundResult{Status: "succeeded"}, nil
}

package yookassa

import (
	"context"
	"encoding/json"
	"errors"
)

func (a *YooKassa) SyncPayment(ctx context.Context, params SyncPaymentParams) (*WebhookResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if a == nil || a.repository == nil {
		return nil, errors.New("yookassa: api is not initialized")
	}
	payment, err := NewClient(params.Credentials).GetPayment(ctx, params.PaymentID)
	if err != nil {
		return nil, err
	}
	webhook := webhookPayload{
		Type:  "poll",
		Event: "payment." + payment.Status,
		Object: webhookPaymentObject{
			ID:     payment.ID,
			Status: payment.Status,
			Paid:   payment.Paid,
			Amount: payment.Amount,
		},
	}
	raw, err := json.Marshal(webhook)
	if err != nil {
		return nil, err
	}
	return a.handlePayload(ctx, webhook, raw, false)
}

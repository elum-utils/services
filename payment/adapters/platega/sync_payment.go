package platega

import (
	"context"
	"encoding/json"
)

func (a *Platega) SyncPayment(ctx context.Context, params SyncPaymentParams) (*WebhookResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if a == nil || a.repository == nil {
		return nil, ErrNotInitialized
	}
	transaction, err := NewClient(params.Credentials).GetTransaction(ctx, params.TransactionID)
	if err != nil {
		return nil, err
	}
	payload := callbackPayload{
		ID:            transaction.ID,
		Amount:        transaction.PaymentDetails.Amount,
		Currency:      transaction.PaymentDetails.Currency,
		Status:        transaction.Status,
		PaymentMethod: PaymentMethodAny,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return a.handlePayload(ctx, payload, raw, false)
}

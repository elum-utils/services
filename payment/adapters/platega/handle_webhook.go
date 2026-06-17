package platega

import (
	"context"
	json "github.com/goccy/go-json"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"
)

func (a *Platega) HandleWebhook(ctx context.Context, request WebhookRequest) (*WebhookResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if a == nil || a.repository == nil {
		return nil, ErrNotInitialized
	}
	signatureValid := validateHeaders(request.Headers, request.Credentials)
	if !signatureValid {
		return nil, ErrWebhookCredentialsInvalid
	}

	var payload callbackPayload
	if err := json.Unmarshal(request.Raw, &payload); err != nil {
		return nil, err
	}
	return a.handlePayload(ctx, payload, request.Raw, signatureValid)
}

func (a *Platega) handlePayload(ctx context.Context, payload callbackPayload, raw []byte, signatureValid bool) (*WebhookResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if payload.ID == "" {
		return nil, ErrTransactionIDRequired
	}

	attempt, err := a.repository.GetAttemptByProviderPaymentID(ctx, ProviderCode, payload.ID)
	if err != nil {
		return nil, err
	}

	eventID := webhookEventID(payload)
	eventDBID, err := a.repository.CreateEvent(ctx, repository.EventCreateParams{
		ProviderCode:      ProviderCode,
		AttemptID:         utils.Ref(int64(attempt.ID)),
		OrderID:           utils.Ref(int64(attempt.OrderID)),
		ProviderEventID:   utils.Ref(eventID),
		ProviderPaymentID: utils.Ref(payload.ID),
		EventType:         "payment_status",
		EventStatus:       utils.Ref(string(payload.Status)),
		PayloadHash:       sha256Hex(raw),
		SignatureValid:    utils.Ref(signatureValid),
	})
	if err != nil && !isDuplicateEntry(err) {
		return nil, err
	}

	result := &WebhookResult{
		OrderID:   attempt.OrderID,
		AttemptID: attempt.ID,
		EventID:   eventDBID,
		Status:    payload.Status,
	}
	if payload.Status != StatusConfirmed {
		return result, nil
	}

	completed, err := a.repository.CompleteAttempt(ctx, repository.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      ProviderCode,
		ProviderPaymentID: utils.Ref(payload.ID),
		AmountMinor:       rubMinorFromMajor(payload.Amount),
		AssetCode:         payload.Currency,
	})
	if err != nil {
		return nil, err
	}
	result.AlreadyDone = completed.AlreadyDone
	result.FulfilledID = uint64Ptr(completed.FulfillmentID)
	return result, nil
}
func uint64Ptr(value *int64) *uint64 {
	if value == nil {
		return nil
	}
	v := uint64(*value)
	return utils.Ref(v)
}

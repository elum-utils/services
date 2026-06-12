package telegramstars

import (
	"context"
	"errors"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"
)

func (a *TelegramStars) HandlePreCheckoutQuery(ctx context.Context, query PreCheckoutQuery) (*PreCheckoutResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if a == nil || a.repository == nil {
		return nil, errors.New("telegram_stars: api is not initialized")
	}
	client := NewClient(query.Credentials)

	attempt, err := a.repository.GetAttemptByProviderPaymentID(ctx, ProviderCode, query.InvoicePayload)
	if err != nil {
		answerErr := client.AnswerPreCheckoutQuery(ctx, query.ID, false, "Payment order was not found")
		if answerErr != nil {
			return nil, answerErr
		}
		return &PreCheckoutResult{Accepted: false}, nil
	}

	eventID := query.ID
	if _, err := a.repository.CreateEvent(ctx, repository.EventCreateParams{
		ProviderCode:      ProviderCode,
		AttemptID:         utils.Ref(int64(attempt.ID)),
		OrderID:           utils.Ref(int64(attempt.OrderID)),
		ProviderEventID:   refIfNotEmpty(eventID),
		ProviderPaymentID: refIfNotEmpty(query.InvoicePayload),
		EventType:         "pre_checkout_query",
		EventStatus:       utils.Ref("received"),
		PayloadHash:       sha256Hex([]byte(eventID + ":" + query.InvoicePayload)),
		SignatureValid:    nil,
	}); err != nil && !isDuplicateEntry(err) {
		return nil, err
	}

	if query.Currency != AssetCode || query.TotalAmount != attempt.AmountMinor {
		if err := client.AnswerPreCheckoutQuery(ctx, query.ID, false, "Payment amount mismatch"); err != nil {
			return nil, err
		}
		return &PreCheckoutResult{AttemptID: attempt.ID, OrderID: attempt.OrderID, Accepted: false}, nil
	}

	if err := client.AnswerPreCheckoutQuery(ctx, query.ID, true, ""); err != nil {
		return nil, err
	}
	return &PreCheckoutResult{AttemptID: attempt.ID, OrderID: attempt.OrderID, Accepted: true}, nil
}

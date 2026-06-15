package telegramstars

import (
	"context"
	"time"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"
)

func (a *TelegramStars) HandleSuccessfulPayment(ctx context.Context, payment SuccessfulPayment) (*SuccessfulPaymentResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if payment.InvoicePayload == "" {
		return nil, ErrInvoicePayloadRequired
	}
	if payment.TelegramPaymentChargeID == "" {
		return nil, ErrTelegramPaymentChargeIDRequired
	}

	attempt, err := a.repository.GetAttemptByProviderPaymentID(ctx, ProviderCode, payment.InvoicePayload)
	if err != nil {
		return nil, err
	}

	eventID := "successful_payment:" + payment.TelegramPaymentChargeID
	eventDBID, err := a.repository.CreateEvent(ctx, repository.EventCreateParams{
		ProviderCode:      ProviderCode,
		AttemptID:         utils.Ref(int64(attempt.ID)),
		OrderID:           utils.Ref(int64(attempt.OrderID)),
		ProviderEventID:   utils.Ref(eventID),
		ProviderPaymentID: utils.Ref(payment.InvoicePayload),
		EventType:         "successful_payment",
		EventStatus:       utils.Ref("succeeded"),
		PayloadHash:       sha256Hex([]byte(eventID + ":" + payment.InvoicePayload)),
		SignatureValid:    nil,
	})
	if err != nil && !isDuplicateEntry(err) {
		return nil, err
	}

	if _, err := a.repository.SetAttemptProviderChargeID(ctx, attempt.ID, ProviderCode, payment.TelegramPaymentChargeID); err != nil {
		return nil, err
	}

	completed, err := a.repository.CompleteAttempt(ctx, repository.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      ProviderCode,
		ProviderPaymentID: utils.Ref(payment.InvoicePayload),
		AmountMinor:       payment.TotalAmount,
		AssetCode:         payment.Currency,
	})
	if err != nil {
		return nil, err
	}

	if payment.SubscriptionExpirationDate > 0 {
		order, err := a.repository.GetOrder(ctx, completed.OrderID)
		if err != nil {
			return nil, err
		}
		if _, err := a.repository.UpsertSubscription(ctx, repository.SubscriptionUpsertParams{
			WorkspaceID:            order.WorkspaceID,
			ProviderCode:           ProviderCode,
			ProviderSubscriptionID: payment.TelegramPaymentChargeID,
			AppID:                  order.AppID,
			PlatformID:             order.PlatformID,
			PlatformUserID:         order.PlatformUserID,
			InternalUserID:         nullInt64FromPtr(order.InternalUserID),
			ProductID:              order.ProductID,
			OrderID:                int64Null(int64(order.ID)),
			AttemptID:              int64Null(int64(attempt.ID)),
			Status:                 "active",
			StartedAt:              time.Now(),
			EndedAt:                timeNull(time.Unix(payment.SubscriptionExpirationDate, 0)),
		}); err != nil {
			return nil, err
		}
	}

	return &SuccessfulPaymentResult{
		OrderID:       completed.OrderID,
		AttemptID:     completed.AttemptID,
		EventID:       eventDBID,
		AlreadyDone:   completed.AlreadyDone,
		FulfillmentID: uint64Ptr(completed.FulfillmentID),
	}, nil
}

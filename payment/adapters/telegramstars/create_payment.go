package telegramstars

import (
	"context"
	"fmt"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"
)

func (a *TelegramStars) CreatePayment(ctx context.Context, params CreatePaymentParams) (*CreatePaymentResponse, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if a == nil || a.repository == nil {
		return nil, ErrNotInitialized
	}
	client := NewClient(params.Credentials)

	order, err := a.repository.CreateOrder(ctx, repository.OrderCreateParams{
		WorkspaceID:    params.WorkspaceID,
		AppID:          params.AppID,
		PlatformID:     params.PlatformID,
		PlatformUserID: params.PlatformUserID,
		InternalUserID: params.InternalUserID,
		ProductID:      params.ProductID,
		Quantity:       params.Quantity,
		AssetCode:      AssetCode,
		Locale:         normalizeLocale(params.Locale),
		ReservedUntil:  params.ReservedUntil,
		ExpiresAt:      params.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	payload := order.PublicID
	title := normalizeTitle(params.Title, order.ProductID)
	description := normalizeDescription(params.Description, order.PublicID)
	subscriptionPeriod := normalizeSubscriptionPeriod(params.SubscriptionPeriod)

	invoiceLink, err := client.CreateInvoiceLink(ctx, createInvoiceLinkRequest{
		Title:              title,
		Description:        description,
		Payload:            payload,
		ProviderToken:      "",
		Currency:           AssetCode,
		Prices:             []LabeledPrice{{Label: title, Amount: order.PayableAmountMinor}},
		SubscriptionPeriod: subscriptionPeriod,
	})
	if err != nil {
		return nil, err
	}

	idempotencyKey := params.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("%s:order:%d", ProviderCode, order.ID)
	}

	attempt, err := a.repository.CreateAttempt(ctx, repository.AttemptCreateParams{
		OrderID:           order.ID,
		ProviderCode:      ProviderCode,
		KnownAssetCode:    utils.Ref(order.AssetCode),
		KnownAmountMinor:  utils.Ref(order.PayableAmountMinor),
		ProviderPaymentID: utils.Ref(payload),
		ProviderInvoiceID: utils.Ref(payload),
		IdempotencyKey:    utils.Ref(idempotencyKey),
		ExpiresAt:         params.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	return &CreatePaymentResponse{
		OrderID:            order.ID,
		OrderPublicID:      order.PublicID,
		AttemptID:          attempt.ID,
		InvoiceLink:        invoiceLink,
		AmountMinor:        attempt.AmountMinor,
		AssetCode:          attempt.AssetCode,
		SubscriptionPeriod: subscriptionPeriod,
	}, nil
}

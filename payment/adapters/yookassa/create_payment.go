package yookassa

import (
	"context"
	"fmt"
	"strconv"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"
)

func (a *YooKassa) CreatePayment(ctx context.Context, params CreatePaymentParams) (*CreatePaymentResponse, error) {
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

	idempotencyKey := params.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("%s:order:%d", ProviderCode, order.ID)
	}
	description := params.Description
	if description == "" {
		description = fmt.Sprintf("Payment order %s", order.PublicID)
	}
	capture := true
	if params.Capture != nil {
		capture = *params.Capture
	}

	payment, err := client.CreatePayment(ctx, createPaymentRequest{
		Amount: Amount{
			Value:    formatRubMinor(order.PayableAmountMinor),
			Currency: AssetCode,
		},
		Capture:           capture,
		Description:       description,
		PaymentMethodData: paymentMethodData(params.PaymentMethodType),
		Receipt:           params.Receipt,
		Confirmation: yookassaConfirmation{
			Type:      "redirect",
			ReturnURL: params.ReturnURL,
		},
		Metadata: map[string]string{
			"order_id":        strconv.FormatUint(order.ID, 10),
			"order_public_id": order.PublicID,
			"workspace_id":    order.WorkspaceID,
			"product_id":      order.ProductID,
		},
	}, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if payment.ID == "" {
		return nil, ErrCreatePaymentEmptyID
	}

	attempt, err := a.repository.CreateAttempt(ctx, repository.AttemptCreateParams{
		OrderID:           order.ID,
		ProviderCode:      ProviderCode,
		KnownAssetCode:    utils.Ref(order.AssetCode),
		KnownAmountMinor:  utils.Ref(order.PayableAmountMinor),
		ProviderPaymentID: utils.Ref(payment.ID),
		IdempotencyKey:    utils.Ref(idempotencyKey),
		ConfirmationURL:   nilIfEmpty(payment.Confirmation.ConfirmationURL),
	})
	if err != nil {
		return nil, err
	}

	return &CreatePaymentResponse{
		OrderID:           order.ID,
		OrderPublicID:     order.PublicID,
		AttemptID:         attempt.ID,
		PaymentID:         payment.ID,
		Status:            payment.Status,
		ConfirmationURL:   payment.Confirmation.ConfirmationURL,
		AmountMinor:       attempt.AmountMinor,
		AssetCode:         attempt.AssetCode,
		PaymentMethodType: params.PaymentMethodType,
	}, nil
}

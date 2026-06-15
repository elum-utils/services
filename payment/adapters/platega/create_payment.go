package platega

import (
	"context"
	"fmt"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"
)

func (a *Platega) CreatePayment(ctx context.Context, params CreatePaymentParams) (*CreatePaymentResponse, error) {
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

	description := params.Description
	if description == "" {
		description = fmt.Sprintf("Payment order %s", order.PublicID)
	}

	var method *PaymentMethod
	if params.PaymentMethod != PaymentMethodAny {
		method = &params.PaymentMethod
	}
	transaction, err := client.CreateTransaction(ctx, createTransactionRequest{
		PaymentMethod: method,
		PaymentDetails: paymentDetails{
			Amount:   rubMajorFromMinor(order.PayableAmountMinor),
			Currency: AssetCode,
		},
		Description: description,
		ReturnURL:   params.ReturnURL,
		FailedURL:   params.FailedURL,
		Payload:     order.PublicID,
	})
	if err != nil {
		return nil, err
	}
	if transaction.TransactionID == "" {
		return nil, ErrTransactionResponseEmpty
	}

	idempotencyKey := params.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("%s:order:%d", ProviderCode, order.ID)
	}
	paymentURL := transaction.URL
	if paymentURL == "" {
		paymentURL = transaction.Redirect
	}

	attempt, err := a.repository.CreateAttempt(ctx, repository.AttemptCreateParams{
		OrderID:           order.ID,
		ProviderCode:      ProviderCode,
		KnownAssetCode:    utils.Ref(order.AssetCode),
		KnownAmountMinor:  utils.Ref(order.PayableAmountMinor),
		ProviderPaymentID: utils.Ref(transaction.TransactionID),
		ProviderInvoiceID: utils.Ref(order.PublicID),
		IdempotencyKey:    utils.Ref(idempotencyKey),
		ConfirmationURL:   nilIfEmpty(paymentURL),
		ReturnURL:         nilIfEmpty(params.ReturnURL),
		ExpiresAt:         params.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	return &CreatePaymentResponse{
		OrderID:        order.ID,
		OrderPublicID:  order.PublicID,
		AttemptID:      attempt.ID,
		TransactionID:  transaction.TransactionID,
		Status:         transaction.Status,
		PaymentURL:     paymentURL,
		RedirectURL:    transaction.Redirect,
		ReturnURL:      transaction.ReturnURL,
		ExpiresIn:      transaction.ExpiresIn,
		AmountMinor:    attempt.AmountMinor,
		AssetCode:      attempt.AssetCode,
		PaymentMethod:  params.PaymentMethod,
		ProviderMethod: transaction.PaymentMethod,
	}, nil
}

func (a *Platega) GetH2H(ctx context.Context, params GetH2HParams) (H2HResponse, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	if a == nil {
		return H2HResponse{}, ErrNotInitialized
	}
	return NewClient(params.Credentials).GetH2H(ctx, params.TransactionID)
}

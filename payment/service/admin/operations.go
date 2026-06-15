package admin

import (
	"context"

	"github.com/elum-utils/services/payment/service/checkout"
	"github.com/elum-utils/services/payment/service/product"
	"github.com/elum-utils/services/payment/service/refund"
)

type CreateProductKeyParams = product.CreateKeyParams
type ExecuteRefundParams = refund.Params
type ExecuteRefundResult = refund.Result
type CreatePaymentEventParams = checkout.CreateEventParams
type CompletePaymentAttemptParams = checkout.CompleteAttemptParams
type CompletePaymentAttemptResult = checkout.CompleteAttemptResult

func (a *Admin) CreateProductKey(ctx context.Context, params CreateProductKeyParams) (string, error) {
	if a == nil || a.products == nil {
		return "", ErrProductServiceNotInitialized
	}
	return a.products.CreateKey(ctx, params)
}

func (a *Admin) RebuildProductCache(ctx context.Context, workspaceID string) error {
	if a == nil || a.products == nil {
		return ErrProductServiceNotInitialized
	}
	return a.products.RebuildWorkspaceCache(ctx, workspaceID)
}

func (a *Admin) ExecuteRefund(ctx context.Context, params ExecuteRefundParams) (*ExecuteRefundResult, error) {
	if a == nil || a.refunds == nil {
		return nil, ErrRefundServiceNotInitialized
	}
	return a.refunds.Execute(ctx, params)
}

func (a *Admin) CreatePaymentEvent(ctx context.Context, params CreatePaymentEventParams) (uint64, error) {
	if a == nil || a.checkout == nil {
		return 0, ErrCheckoutServiceNotInitialized
	}
	return a.checkout.CreateEvent(ctx, params)
}

func (a *Admin) CompletePaymentAttempt(ctx context.Context, params CompletePaymentAttemptParams) (*CompletePaymentAttemptResult, error) {
	if a == nil || a.checkout == nil {
		return nil, ErrCheckoutServiceNotInitialized
	}
	return a.checkout.CompleteAttempt(ctx, params)
}

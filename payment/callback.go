package payment

import (
	"context"
	"encoding/json"
	"time"

	servicecallback "github.com/elum-utils/services/callback"
	serviceerrors "github.com/elum-utils/services/errors"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
)

const (
	CallbackEventPaymentOrderFulfilled = "payment.order.fulfilled"
	CallbackEventPaymentOrderRefunded  = "payment.order.refunded"
)

type Reward = servicecallback.Reward

type PaymentFulfilledCallbackPayload struct {
	OrderID           uint64   `json:"order_id"`
	AttemptID         uint64   `json:"attempt_id"`
	FulfillmentID     uint64   `json:"fulfillment_id"`
	WorkspaceID       string   `json:"workspace_id"`
	AppID             int64    `json:"app_id"`
	PlatformID        int64    `json:"platform_id"`
	PlatformUserID    string   `json:"platform_user_id"`
	ProductID         string   `json:"product_id"`
	Quantity          uint64   `json:"quantity"`
	ProviderCode      string   `json:"provider_code"`
	ProviderPaymentID string   `json:"provider_payment_id,omitempty"`
	AssetCode         string   `json:"asset_code"`
	AmountMinor       uint64   `json:"amount_minor"`
	Rewards           []Reward `json:"rewards"`
}

type PaymentRefundedCallbackPayload struct {
	OrderID           uint64   `json:"order_id"`
	AttemptID         uint64   `json:"attempt_id"`
	FulfillmentID     uint64   `json:"fulfillment_id"`
	RefundID          uint64   `json:"refund_id"`
	WorkspaceID       string   `json:"workspace_id"`
	AppID             int64    `json:"app_id"`
	PlatformID        int64    `json:"platform_id"`
	PlatformUserID    string   `json:"platform_user_id"`
	ProductID         string   `json:"product_id"`
	Quantity          uint64   `json:"quantity"`
	ProviderCode      string   `json:"provider_code"`
	ProviderPaymentID string   `json:"provider_payment_id,omitempty"`
	AssetCode         string   `json:"asset_code"`
	AmountMinor       uint64   `json:"amount_minor"`
	Reason            string   `json:"reason,omitempty"`
	Rewards           []Reward `json:"rewards"`
}

type Context struct {
	callbackutil.Context

	Payload          *servicecallback.RewardPayload
	PaymentFulfilled *PaymentFulfilledCallbackPayload
	PaymentRefunded  *PaymentRefundedCallbackPayload
}

type CallbackHandler func(Context) error
type CallbackOption = callbackutil.Option

type callbackRegistration struct {
	ctx     context.Context
	handler CallbackHandler
	options []CallbackOption
}

var ErrCallbackAlreadyMarked = callbackutil.ErrAlreadyMarked

func WithCallbackWorkerID(workerID string) CallbackOption {
	return callbackutil.WithWorkerID(workerID)
}

func WithCallbackBatchSize(batchSize int32) CallbackOption {
	return callbackutil.WithBatchSize(batchSize)
}

func WithCallbackLeaseTimeout(timeout time.Duration) CallbackOption {
	return callbackutil.WithLeaseTimeout(timeout)
}

func WithCallbackIdleDelay(delay time.Duration) CallbackOption {
	return callbackutil.WithIdleDelay(delay)
}

func (a *Payment) OnCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if handler == nil {
		return ErrCallbackHandlerNil
	}
	if a == nil {
		return ErrServiceNil
	}
	a.lifecycleMu.Lock()
	if a.running {
		a.lifecycleMu.Unlock()
		return ErrCallbacksRegistrationClosed
	}
	if a.callbacks != nil {
		a.lifecycleMu.Unlock()
		return a.runCallback(ctx, handler, opts...)
	}
	a.callbacksToRun = append(a.callbacksToRun, callbackRegistration{
		ctx: ctx, handler: handler, options: append([]CallbackOption(nil), opts...),
	})
	a.lifecycleMu.Unlock()
	return nil
}

func (a *Payment) runCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if a == nil || a.callbacks == nil {
		return ErrCallbacksNotConfigured
	}
	runCtx, cancel := a.bindContext(ctx)
	defer cancel()

	opts = append(opts, callbackutil.WithSourceService("payment"))
	return a.callbacks.On(runCtx, func(callbackCtx callbackutil.Context) error {
		paymentCtx, err := newCallbackContext(callbackCtx)
		if err != nil {
			return serviceerrors.Wrap(serviceerrors.CodeInternalError, "payment callback payload decode failed", err)
		}
		return handler(paymentCtx)
	}, opts...)
}

func newCallbackContext(callbackCtx callbackutil.Context) (Context, error) {
	ctx := Context{Context: callbackCtx}
	switch callbackCtx.EventType {
	case CallbackEventPaymentOrderFulfilled:
		var payload PaymentFulfilledCallbackPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return Context{}, serviceerrors.Wrap(serviceerrors.CodeInternalError, "payment callback payload decode failed", err)
		}
		ctx.Payload = &servicecallback.RewardPayload{
			Identity: servicecallback.Identity{
				WorkspaceID: payload.WorkspaceID,
				AppID:       payload.AppID, PlatformID: payload.PlatformID,
				PlatformUserID: payload.PlatformUserID,
			},
			Rewards: payload.Rewards,
		}
		ctx.PaymentFulfilled = &payload
	case CallbackEventPaymentOrderRefunded:
		var payload PaymentRefundedCallbackPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return Context{}, serviceerrors.Wrap(serviceerrors.CodeInternalError, "payment callback payload decode failed", err)
		}
		ctx.Payload = &servicecallback.RewardPayload{
			Identity: servicecallback.Identity{
				WorkspaceID: payload.WorkspaceID,
				AppID:       payload.AppID, PlatformID: payload.PlatformID,
				PlatformUserID: payload.PlatformUserID,
			},
			Rewards: payload.Rewards,
		}
		ctx.PaymentRefunded = &payload
	}
	return ctx, nil
}

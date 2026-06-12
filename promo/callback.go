package promo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
)

const CallbackEventApplied = "promo.applied"

type CallbackReward struct {
	Key      string `json:"key"`
	Quantity int64  `json:"quantity"`
}

type CallbackPayload struct {
	RedemptionID   uint64           `json:"redemption_id"`
	WorkspaceID    string           `json:"workspace_id"`
	PromoID        uint64           `json:"promo_id"`
	Code           string           `json:"code"`
	AppID          int64            `json:"app_id"`
	PlatformID     int64            `json:"platform_id"`
	PlatformUserID string           `json:"platform_user_id"`
	Rewards        []CallbackReward `json:"rewards"`
}

type Context struct {
	callbackutil.Context
	Applied *CallbackPayload
}

type CallbackHandler func(Context) error
type CallbackOption = callbackutil.Option
type callbackRegistration struct {
	ctx     context.Context
	handler CallbackHandler
	options []CallbackOption
}

func WithCallbackWorkerID(value string) CallbackOption { return callbackutil.WithWorkerID(value) }
func WithCallbackBatchSize(value int32) CallbackOption { return callbackutil.WithBatchSize(value) }
func WithCallbackLeaseTimeout(value time.Duration) CallbackOption {
	return callbackutil.WithLeaseTimeout(value)
}
func WithCallbackIdleDelay(value time.Duration) CallbackOption {
	return callbackutil.WithIdleDelay(value)
}

func (p *Promo) OnCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if handler == nil {
		return errors.New("promo: callback handler is nil")
	}
	if p == nil {
		return errors.New("promo: nil service")
	}
	p.lifecycleMu.Lock()
	if p.running {
		p.lifecycleMu.Unlock()
		return errors.New("promo: callbacks must be registered before Run")
	}
	if p.callbacks != nil {
		p.lifecycleMu.Unlock()
		return p.runCallback(ctx, handler, opts...)
	}
	p.callbacksToRun = append(p.callbacksToRun, callbackRegistration{
		ctx: ctx, handler: handler, options: append([]CallbackOption(nil), opts...),
	})
	p.lifecycleMu.Unlock()
	return nil
}

func (p *Promo) runCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if p == nil || p.callbacks == nil {
		return callbackutil.ErrStoreNotConfigured
	}
	runCtx, cancel := p.bindContext(ctx)
	defer cancel()
	opts = append(opts, callbackutil.WithSourceService("promo"))
	return p.callbacks.On(runCtx, func(callbackCtx callbackutil.Context) error {
		var payload CallbackPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return err
		}
		return handler(Context{Context: callbackCtx, Applied: &payload})
	}, opts...)
}

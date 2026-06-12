package tasks

import (
	"context"
	"encoding/json"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/tasks/repository"
)

type Context struct {
	callbackutil.Context
	Claimed *repository.CallbackPayload
}

type CallbackHandler func(Context) error
type CallbackOption = callbackutil.Option

func WithCallbackWorkerID(value string) CallbackOption { return callbackutil.WithWorkerID(value) }
func WithCallbackBatchSize(value int32) CallbackOption { return callbackutil.WithBatchSize(value) }
func WithCallbackLeaseTimeout(value time.Duration) CallbackOption {
	return callbackutil.WithLeaseTimeout(value)
}
func WithCallbackIdleDelay(value time.Duration) CallbackOption {
	return callbackutil.WithIdleDelay(value)
}

func (t *Tasks) OnCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if t == nil || t.callbacks == nil {
		return callbackutil.ErrStoreNotConfigured
	}
	if handler == nil {
		return t.callbacks.On(ctx, nil, opts...)
	}
	runCtx, cancel := t.bindContext(ctx)
	defer cancel()
	t.background.Add(1)
	defer t.background.Done()
	opts = append(opts, callbackutil.WithSourceService("tasks"))
	return t.callbacks.On(runCtx, func(callbackCtx callbackutil.Context) error {
		var payload repository.CallbackPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return err
		}
		return handler(Context{Context: callbackCtx, Claimed: &payload})
	}, opts...)
}

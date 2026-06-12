package tasks

import (
	"context"
	"encoding/json"
	"errors"
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

func (t *Tasks) OnCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if handler == nil {
		return errors.New("tasks: callback handler is nil")
	}
	if t == nil {
		return errors.New("tasks: nil service")
	}
	t.lifecycleMu.Lock()
	if t.running {
		t.lifecycleMu.Unlock()
		return errors.New("tasks: callbacks must be registered before Run")
	}
	if t.callbacks != nil {
		t.lifecycleMu.Unlock()
		return t.runCallback(ctx, handler, opts...)
	}
	t.callbacksToRun = append(t.callbacksToRun, callbackRegistration{
		ctx: ctx, handler: handler, options: append([]CallbackOption(nil), opts...),
	})
	t.lifecycleMu.Unlock()
	return nil
}

func (t *Tasks) runCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if t == nil || t.callbacks == nil {
		return callbackutil.ErrStoreNotConfigured
	}
	runCtx, cancel := t.bindContext(ctx)
	defer cancel()
	opts = append(opts, callbackutil.WithSourceService("tasks"))
	return t.callbacks.On(runCtx, func(callbackCtx callbackutil.Context) error {
		var payload repository.CallbackPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return err
		}
		return handler(Context{Context: callbackCtx, Claimed: &payload})
	}, opts...)
}

package tasks

import (
	"context"
	"encoding/json"
	"time"

	servicecallback "github.com/elum-utils/services/callback"
	serviceerrors "github.com/elum-utils/services/errors"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
)

type CallbackPayload struct {
	WorkspaceID    string                   `json:"workspace_id"`
	AppID          int64                    `json:"app_id"`
	PlatformID     int64                    `json:"platform_id"`
	PlatformUserID string                   `json:"platform_user_id"`
	TaskID         uint64                   `json:"task_id"`
	TaskKey        string                   `json:"task_key"`
	OperationID    string                   `json:"operation_id"`
	PeriodStartAt  time.Time                `json:"period_start_at"`
	PeriodEndAt    time.Time                `json:"period_end_at"`
	Rewards        []servicecallback.Reward `json:"rewards"`
	Payload        json.RawMessage          `json:"payload"`
}

type Context struct {
	callbackutil.Context
	Payload *servicecallback.RewardPayload
	Claimed *CallbackPayload
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
		return ErrCallbackHandlerNil
	}
	if t == nil {
		return ErrServiceNil
	}
	t.lifecycleMu.Lock()
	if t.running {
		t.lifecycleMu.Unlock()
		return ErrCallbacksRegistrationClosed
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
		return ErrCallbacksNotConfigured
	}
	runCtx, cancel := t.bindContext(ctx)
	defer cancel()
	opts = append(opts, callbackutil.WithSourceService("tasks"))
	return t.callbacks.On(runCtx, func(callbackCtx callbackutil.Context) error {
		var payload CallbackPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return serviceerrors.Wrap(serviceerrors.CodeInternalError, "tasks callback payload decode failed", err)
		}
		return handler(Context{
			Context: callbackCtx,
			Payload: &servicecallback.RewardPayload{
				Identity: servicecallback.Identity{
					WorkspaceID: payload.WorkspaceID,
					AppID:       payload.AppID, PlatformID: payload.PlatformID,
					PlatformUserID: payload.PlatformUserID,
				},
				Rewards: payload.Rewards,
			},
			Claimed: &payload,
		})
	}, opts...)
}

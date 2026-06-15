package calendar

import (
	"context"
	"encoding/json"
	"time"

	servicecallback "github.com/elum-utils/services/callback"
	serviceerrors "github.com/elum-utils/services/errors"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
)

const CallbackEventRewardGranted = "calendar.reward_granted"

type CallbackReward = servicecallback.Reward

type RewardGrantedPayload struct {
	OperationRowID uint64           `json:"operation_row_id"`
	OperationID    string           `json:"operation_id"`
	WorkspaceID    string           `json:"workspace_id"`
	CalendarID     string           `json:"calendar_id"`
	AppID          int64            `json:"app_id"`
	PlatformID     int64            `json:"platform_id"`
	PlatformUserID string           `json:"platform_user_id"`
	Position       uint32           `json:"position"`
	Rewards        []CallbackReward `json:"rewards"`
	OccurredAt     time.Time        `json:"occurred_at"`
}

type Context struct {
	callbackutil.Context
	Payload       *servicecallback.RewardPayload
	RewardGranted *RewardGrantedPayload
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

func (c *Calendar) OnCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if handler == nil {
		return ErrCallbackHandlerNil
	}
	if c == nil {
		return ErrServiceNil
	}
	c.lifecycleMu.Lock()
	if c.running {
		c.lifecycleMu.Unlock()
		return ErrCallbacksRegistrationClosed
	}
	if c.callbacks != nil {
		c.lifecycleMu.Unlock()
		return c.runCallback(ctx, handler, opts...)
	}
	c.callbacksToRun = append(c.callbacksToRun, callbackRegistration{
		ctx: ctx, handler: handler, options: append([]CallbackOption(nil), opts...),
	})
	c.lifecycleMu.Unlock()
	return nil
}

func (c *Calendar) runCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if c == nil || c.callbacks == nil {
		return ErrCallbacksNotConfigured
	}
	runCtx, cancel := c.bindContext(ctx)
	defer cancel()
	opts = append(opts, callbackutil.WithSourceService("calendar"))
	return c.callbacks.On(runCtx, func(callbackCtx callbackutil.Context) error {
		var payload RewardGrantedPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return serviceerrors.Wrap(serviceerrors.CodeInternalError, "calendar callback payload decode failed", err)
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
			RewardGranted: &payload,
		})
	}, opts...)
}

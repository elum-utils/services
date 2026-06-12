package calendar

import (
	"context"
	"encoding/json"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
)

const CallbackEventRewardGranted = "calendar.reward_granted"

type CallbackReward struct {
	Key      string `json:"key"`
	Quantity int64  `json:"quantity"`
}

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
	RewardGranted *RewardGrantedPayload
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

func (c *Calendar) OnCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
	if c == nil || c.callbacks == nil {
		return callbackutil.ErrStoreNotConfigured
	}
	if handler == nil {
		return c.callbacks.On(ctx, nil, opts...)
	}
	runCtx, cancel := c.bindContext(ctx)
	defer cancel()
	c.background.Add(1)
	defer c.background.Done()
	opts = append(opts, callbackutil.WithSourceService("calendar"))
	return c.callbacks.On(runCtx, func(callbackCtx callbackutil.Context) error {
		var payload RewardGrantedPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return err
		}
		return handler(Context{Context: callbackCtx, RewardGranted: &payload})
	}, opts...)
}

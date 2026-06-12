package cpa

import (
	"context"
	"encoding/json"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
)

const (
	CallbackEventIssued    = "cpa.issued"
	CallbackEventCompleted = "cpa.completed"
)

type CallbackReward struct {
	Key      string `json:"key"`
	Quantity int64  `json:"quantity"`
}

type CallbackPayload struct {
	AssignmentID   uint64           `json:"assignment_id"`
	WorkspaceID    string           `json:"workspace_id"`
	CPAID          string           `json:"cpa_id"`
	AppID          int64            `json:"app_id"`
	PlatformID     int64            `json:"platform_id"`
	PlatformUserID string           `json:"platform_user_id"`
	Code           string           `json:"code"`
	CodeMode       string           `json:"code_mode"`
	Status         string           `json:"status"`
	Rewards        []CallbackReward `json:"rewards,omitempty"`
}

type Context struct {
	callbackutil.Context
	Issued    *CallbackPayload
	Completed *CallbackPayload
}

type CallbackHandler func(Context) error
type CallbackOption = callbackutil.Option

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

func (c *CPA) OnCallback(ctx context.Context, handler CallbackHandler, opts ...CallbackOption) error {
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

	opts = append(opts, callbackutil.WithSourceService("cpa"))
	return c.callbacks.On(runCtx, func(callbackCtx callbackutil.Context) error {
		value := Context{Context: callbackCtx}
		var payload CallbackPayload
		if err := json.Unmarshal(callbackCtx.Payload, &payload); err != nil {
			return err
		}
		switch callbackCtx.EventType {
		case CallbackEventIssued:
			value.Issued = &payload
		case CallbackEventCompleted:
			value.Completed = &payload
		}
		return handler(value)
	}, opts...)
}

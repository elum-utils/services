package admin

import (
	"context"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
)

type CallbackEventListParams struct {
	Status string
	Page   Page
}

func (a *Admin) ListCallbackEvents(ctx context.Context, params CallbackEventListParams) ([]callbackutil.Event, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	limit, offset := normalizePage(params.Page)
	return a.callbacks.AdminListEvents(mergedCtx, callbackutil.AdminListEventsParams{
		SourceService: "promo", EventType: "promo.applied", Status: params.Status,
		Limit: limit, Offset: offset,
	})
}

func (a *Admin) GetCallbackEvent(ctx context.Context, id uint64) (callbackutil.Event, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.callbacks.GetEvent(mergedCtx, id)
}

func (a *Admin) RetryCallbackEventNow(ctx context.Context, id uint64) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.callbacks.AdminRetryEventNow(mergedCtx, id)
}

func (a *Admin) MarkCallbackEventOK(ctx context.Context, id uint64) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.callbacks.AdminMarkEventOK(mergedCtx, id)
}

func (a *Admin) MarkCallbackEventReject(ctx context.Context, id uint64, reason string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.callbacks.AdminMarkEventReject(mergedCtx, id, reason)
}

func (a *Admin) ResetExpiredCallbackProcessing(ctx context.Context) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.callbacks.AdminResetExpiredProcessing(mergedCtx)
}

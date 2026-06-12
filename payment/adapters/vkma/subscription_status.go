package vkma

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"

	"github.com/elum-utils/sign/vkmashop"
)

func (a *VKMA) Active(ctx context.Context, params vkmashop.Params) (*SubscriptionStatusResponse, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.updateSubscriptionStatus(ctx, params, "active", sql.NullTime{})
}

func (a *VKMA) Canceled(ctx context.Context, params vkmashop.Params) (*SubscriptionStatusResponse, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.updateSubscriptionStatus(ctx, params, "canceled", sql.NullTime{Time: time.Now(), Valid: true})
}

func (a *VKMA) Refunded(ctx context.Context, params vkmashop.Params) (*SubscriptionStatusResponse, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.updateSubscriptionStatus(ctx, params, "refunded", sql.NullTime{Time: time.Now(), Valid: true})
}

func (a *VKMA) updateSubscriptionStatus(ctx context.Context, params vkmashop.Params, status string, endedAt sql.NullTime) (*SubscriptionStatusResponse, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	rows, err := a.repository.UpdateSubscriptionStatus(ctx, repository.SubscriptionStatusUpdateParams{
		ProviderCode:           ProviderCode,
		ProviderSubscriptionID: strconv.Itoa(params.SubscriptionID),
		Status:                 status,
		CancelReason:           sql.NullString{String: string(params.CancelReason), Valid: params.CancelReason != ""},
		EndedAt:                endedAt,
	})
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, sql.ErrNoRows
	}

	if _, err := a.repository.CreateEvent(ctx, repository.EventCreateParams{
		ProviderCode:      ProviderCode,
		ProviderEventID:   eventID(params),
		ProviderPaymentID: positiveString(params.OrderID),
		EventType:         string(params.NotificationType),
		EventStatus:       utils.Ref(string(params.Status)),
		PayloadHash:       payloadHash(params),
		SignatureValid:    utils.Ref(true),
	}); err != nil {
		return nil, err
	}

	return &SubscriptionStatusResponse{
		SubscriptionID: params.SubscriptionID,
		Status:         status,
	}, nil
}

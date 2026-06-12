package repository

import (
	"context"
	"database/sql"
	"time"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

type SubscriptionUpsertParams struct {
	WorkspaceID            string
	ProviderCode           string
	ProviderSubscriptionID string
	AppID                  int64
	PlatformID             int64
	PlatformUserID         string
	InternalUserID         sql.NullInt64
	ProductID              string
	OrderID                sql.NullInt64
	AttemptID              sql.NullInt64
	Status                 string
	CancelReason           sql.NullString
	StartedAt              time.Time
	EndedAt                sql.NullTime
}

type SubscriptionStatusUpdateParams struct {
	ProviderCode           string
	ProviderSubscriptionID string
	Status                 string
	CancelReason           sql.NullString
	EndedAt                sql.NullTime
}

type SubscriptionIsActiveParams struct {
	WorkspaceID    string
	PlatformID     int64
	PlatformUserID string
	ProductID      string
	ProviderCode   string
	Now            time.Time
}

func (r *PaymentRepository) UpsertSubscription(ctx context.Context, params SubscriptionUpsertParams) (uint64, error) {
	workspaceID, err := requireWorkspaceID(params.WorkspaceID)
	if err != nil {
		return 0, err
	}
	startedAt := params.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	status := params.Status
	if status == "" {
		status = string(paymentsqlc.PaymentSubscriptionStatusActive)
	}

	id, err := r.q.UpsertPaymentSubscription(ctx, paymentsqlc.UpsertPaymentSubscriptionParams{
		ProviderCode:           params.ProviderCode,
		WorkspaceID:            workspaceID,
		ProviderSubscriptionID: params.ProviderSubscriptionID,
		AppID:                  params.AppID,
		PlatformID:             params.PlatformID,
		PlatformUserID:         params.PlatformUserID,
		InternalUserID:         params.InternalUserID,
		ProductID:              params.ProductID,
		OrderID:                params.OrderID,
		AttemptID:              params.AttemptID,
		Status:                 paymentsqlc.PaymentSubscriptionStatus(status),
		CancelReason:           params.CancelReason,
		StartedAt:              startedAt,
		EndedAt:                params.EndedAt,
	})
	if err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func (r *PaymentRepository) UpdateSubscriptionStatus(ctx context.Context, params SubscriptionStatusUpdateParams) (int64, error) {
	return r.q.UpdatePaymentSubscriptionStatus(ctx, paymentsqlc.UpdatePaymentSubscriptionStatusParams{
		Status:                 paymentsqlc.PaymentSubscriptionStatus(params.Status),
		CancelReason:           params.CancelReason,
		EndedAt:                params.EndedAt,
		ProviderCode:           params.ProviderCode,
		ProviderSubscriptionID: params.ProviderSubscriptionID,
	})
}

func (r *PaymentRepository) IsSubscriptionActive(ctx context.Context, params SubscriptionIsActiveParams) (bool, error) {
	workspaceID, err := requireWorkspaceID(params.WorkspaceID)
	if err != nil {
		return false, err
	}
	now := params.Now
	if now.IsZero() {
		now = time.Now()
	}

	endedAt := sql.NullTime{Time: now, Valid: true}
	var count int64
	if params.ProductID != "" && params.ProviderCode != "" {
		count, err = r.q.CountActivePaymentSubscriptionsForProductProvider(ctx, paymentsqlc.CountActivePaymentSubscriptionsForProductProviderParams{
			PlatformID:     params.PlatformID,
			PlatformUserID: params.PlatformUserID,
			WorkspaceID:    workspaceID,
			ProductID:      params.ProductID,
			ProviderCode:   params.ProviderCode,
			EndedAt:        endedAt,
		})
	} else if params.ProductID != "" {
		count, err = r.q.CountActivePaymentSubscriptionsForProduct(ctx, paymentsqlc.CountActivePaymentSubscriptionsForProductParams{
			PlatformID:     params.PlatformID,
			PlatformUserID: params.PlatformUserID,
			WorkspaceID:    workspaceID,
			ProductID:      params.ProductID,
			EndedAt:        endedAt,
		})
	} else if params.ProviderCode != "" {
		count, err = r.q.CountActivePaymentSubscriptionsForProvider(ctx, paymentsqlc.CountActivePaymentSubscriptionsForProviderParams{
			PlatformID:     params.PlatformID,
			PlatformUserID: params.PlatformUserID,
			WorkspaceID:    workspaceID,
			ProviderCode:   params.ProviderCode,
			EndedAt:        endedAt,
		})
	} else {
		count, err = r.q.CountActivePaymentSubscriptionsAll(ctx, paymentsqlc.CountActivePaymentSubscriptionsAllParams{
			PlatformID:     params.PlatformID,
			PlatformUserID: params.PlatformUserID,
			WorkspaceID:    workspaceID,
			EndedAt:        endedAt,
		})
	}
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

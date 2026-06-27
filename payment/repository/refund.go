package repository

import (
	"context"
	"database/sql"
	"fmt"
	json "github.com/goccy/go-json"

	utils "github.com/elum-utils/services/internal/utils"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	sqlc "github.com/elum-utils/services/payment/sqlc"
)

type RefundCreateParams struct {
	OrderID          uint64
	AttemptID        uint64
	ProviderCode     string
	ProviderRefundID *string
	AmountMinor      uint64
	AssetCode        string
	Status           string
	Reason           *string
}

type ProviderRefundParams struct {
	WorkspaceID       string
	ProviderCode      string
	ProviderPaymentID string
	ProviderRefundID  string
	Reason            *string
	Event             EventCreateParams
}

type ProviderRefundResult struct {
	RefundID    uint64
	OrderID     uint64
	AttemptID   uint64
	AlreadyDone bool
}

type paymentRefundedCallbackPayload struct {
	OrderID           uint64                  `json:"order_id"`
	AttemptID         uint64                  `json:"attempt_id"`
	FulfillmentID     uint64                  `json:"fulfillment_id"`
	RefundID          uint64                  `json:"refund_id"`
	WorkspaceID       string                  `json:"workspace_id"`
	AppID             int64                   `json:"app_id"`
	PlatformID        int64                   `json:"platform_id"`
	PlatformUserID    string                  `json:"platform_user_id"`
	ProductID         string                  `json:"product_id"`
	Quantity          uint64                  `json:"quantity"`
	ProviderCode      string                  `json:"provider_code"`
	ProviderPaymentID string                  `json:"provider_payment_id,omitempty"`
	AssetCode         string                  `json:"asset_code"`
	AmountMinor       uint64                  `json:"amount_minor"`
	Reason            string                  `json:"reason,omitempty"`
	Rewards           []paymentCallbackReward `json:"rewards"`
}

func (r *PaymentRepository) ApplyProviderRefund(ctx context.Context, params ProviderRefundParams) (ProviderRefundResult, error) {
	var result ProviderRefundResult

	err := r.inTransaction(ctx, func(txRepo *PaymentRepository) error {
		attempt, err := txRepo.q.LockPaymentAttemptByProviderPaymentID(ctx, sqlc.LockPaymentAttemptByProviderPaymentIDParams{
			ProviderCode:      params.ProviderCode,
			ProviderPaymentID: sql.NullString{String: params.ProviderPaymentID, Valid: params.ProviderPaymentID != ""},
		})
		if err != nil {
			return err
		}

		order, err := txRepo.q.LockPaymentOrder(ctx, attempt.OrderID)
		if err != nil {
			return err
		}
		if params.WorkspaceID != "" && order.WorkspaceID != params.WorkspaceID {
			return sql.ErrNoRows
		}

		result.OrderID = order.ID
		result.AttemptID = attempt.ID

		refundID, err := txRepo.q.AdminCreateRefund(ctx, sqlc.AdminCreateRefundParams{
			OrderID:          order.ID,
			AttemptID:        attempt.ID,
			ProviderCode:     params.ProviderCode,
			ProviderRefundID: sql.NullString{String: params.ProviderRefundID, Valid: params.ProviderRefundID != ""},
			AmountMinor:      attempt.AmountMinor,
			AssetCode:        attempt.AssetCode,
			Status:           sqlc.PaymentRefundStatusSucceeded,
			Reason: sqlwrap.NullFromPtr(params.Reason, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
		})
		if err != nil {
			return err
		}
		result.RefundID = uint64(refundID)

		if order.Status == sqlc.PaymentOrderStatusRefunded {
			result.AlreadyDone = true
			return nil
		}
		if order.Status != sqlc.PaymentOrderStatusPaid && order.Status != sqlc.PaymentOrderStatusFulfilled {
			return ErrOrderStateInvalid
		}

		if err := txRepo.q.UpdatePaymentAttemptStatus(ctx, sqlc.UpdatePaymentAttemptStatusParams{
			Status: sqlc.PaymentAttemptStatusRefunded,
			ID:     attempt.ID,
		}); err != nil {
			return err
		}
		if rows, err := txRepo.q.MarkOrderRefunded(ctx, order.ID); err != nil {
			return err
		} else if rows == 0 {
			return ErrOrderStateInvalid
		}
		if _, err := txRepo.q.MarkFulfillmentRevokedForOrder(ctx, order.ID); err != nil {
			return err
		}
		fulfillment, err := txRepo.q.GetFulfillmentForOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		if _, err := txRepo.q.DecrementProductLimitCountersForRefund(ctx, order.ID); err != nil {
			return err
		}

		event := params.Event
		event.AttemptID = utils.Ref(int64(attempt.ID))
		event.OrderID = utils.Ref(int64(order.ID))
		if _, err := txRepo.CreateEvent(ctx, event); err != nil {
			return err
		}
		return txRepo.enqueuePaymentRefundedCallback(ctx, order, attempt, fulfillment.ID, result.RefundID, params.Reason)
	})

	return result, err
}

func (r *PaymentRepository) enqueuePaymentRefundedCallback(
	ctx context.Context,
	order sqlc.PaymentOrder,
	attempt sqlc.PaymentAttempt,
	fulfillmentID uint64,
	refundID uint64,
	reason *string,
) error {
	items, err := r.q.GetFulfillmentItemsForOrder(ctx, order.ID)
	if err != nil {
		return err
	}
	payload := paymentRefundedCallbackPayload{
		OrderID:        order.ID,
		AttemptID:      attempt.ID,
		FulfillmentID:  fulfillmentID,
		RefundID:       refundID,
		WorkspaceID:    order.WorkspaceID,
		AppID:          order.AppID,
		PlatformID:     order.PlatformID,
		PlatformUserID: order.PlatformUserID,
		ProductID:      order.ProductID,
		Quantity:       order.Quantity,
		ProviderCode:   attempt.ProviderCode,
		AssetCode:      attempt.AssetCode,
		AmountMinor:    attempt.AmountMinor,
		Rewards:        make([]paymentCallbackReward, 0, len(items)),
	}
	for _, item := range items {
		payload.Rewards = append(payload.Rewards, paymentCallbackReward{
			Key: item.ItemID, Type: string(item.RewardType), Quantity: item.Quantity,
			Scale: item.Scale,
			Unit:  orderDurationUnitPtr(item.DurationUnit),
		})
	}
	if attempt.ProviderPaymentID.Valid {
		payload.ProviderPaymentID = attempt.ProviderPaymentID.String
	}
	if reason != nil {
		payload.Reason = *reason
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	eventKey := fmt.Sprintf("payment.order.refunded:%d", order.ID)
	_, err = r.callbacks.CreateEvent(ctx, callbackutil.CreateParams{
		SourceService:      "payment",
		EventType:          "payment.order.refunded",
		EventKey:           eventKey,
		IdempotencyKey:     eventKey,
		Payload:            raw,
		PayloadContentType: callbackutil.JSONContentType,
	})
	return err
}

func (r *PaymentRepository) GetAttempt(ctx context.Context, id uint64) (Attempt, error) {
	attempt, err := r.q.AdminGetPaymentAttempt(ctx, id)
	if err != nil {
		return Attempt{}, err
	}
	return mapAttempt(attempt), nil
}

func (r *PaymentRepository) GetRefundAttempt(ctx context.Context, workspaceID string, orderID uint64) (Attempt, error) {
	attempts, err := r.q.AdminListPaymentAttempts(ctx, sqlc.AdminListPaymentAttemptsParams{
		WorkspaceID: workspaceID,
		Column2:     orderID,
		OrderID:     orderID,
		Column6:     string(sqlc.PaymentAttemptStatusSucceeded),
		Status:      sqlc.PaymentAttemptStatusSucceeded,
		Limit:       1,
	})
	if err != nil {
		return Attempt{}, err
	}
	if len(attempts) == 0 {
		return Attempt{}, sql.ErrNoRows
	}
	return mapAttempt(attempts[0]), nil
}

func (r *PaymentRepository) CreateRefund(ctx context.Context, params RefundCreateParams) (uint64, error) {
	status := params.Status
	if status == "" {
		status = string(sqlc.PaymentRefundStatusCreated)
	}
	id, err := r.q.AdminCreateRefund(ctx, sqlc.AdminCreateRefundParams{
		OrderID:          params.OrderID,
		AttemptID:        params.AttemptID,
		ProviderCode:     params.ProviderCode,
		ProviderRefundID: sqlwrap.NullFromPtr(params.ProviderRefundID, func(v string) sql.NullString { return sql.NullString{String: v, Valid: true} }),
		AmountMinor:      params.AmountMinor,
		AssetCode:        params.AssetCode,
		Status:           sqlc.PaymentRefundStatus(status),
		Reason:           sqlwrap.NullFromPtr(params.Reason, func(v string) sql.NullString { return sql.NullString{String: v, Valid: true} }),
	})
	return uint64(id), err
}

func (r *PaymentRepository) UpdateRefundStatus(ctx context.Context, id uint64, status string, reason string) (int64, error) {
	return r.q.AdminUpdateRefundStatus(ctx, sqlc.AdminUpdateRefundStatusParams{
		ID:     id,
		Status: sqlc.PaymentRefundStatus(status),
		Reason: sqlwrap.NullFromPtr(nilIfEmpty(reason), func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
	})
}

func (r *PaymentRepository) SetRefundProviderID(ctx context.Context, id uint64, providerRefundID string) (int64, error) {
	return r.q.AdminSetRefundProviderID(ctx, sqlc.AdminSetRefundProviderIDParams{
		ID: id,
		ProviderRefundID: sqlwrap.NullFromPtr(nilIfEmpty(providerRefundID), func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
	})
}

func (r *PaymentRepository) UpdateAttemptStatus(ctx context.Context, id uint64, status string) error {
	return r.q.UpdatePaymentAttemptStatus(ctx, sqlc.UpdatePaymentAttemptStatusParams{
		ID:     id,
		Status: sqlc.PaymentAttemptStatus(status),
	})
}

func (r *PaymentRepository) UpdateOrderStatus(ctx context.Context, workspaceID string, id uint64, status string) (int64, error) {
	return r.q.AdminUpdateOrderStatus(ctx, sqlc.AdminUpdateOrderStatusParams{
		WorkspaceID: workspaceID,
		ID:          id,
		Status:      sqlc.PaymentOrderStatus(status),
		Column2:     status,
		Column3:     status,
		Column4:     status,
	})
}

func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return utils.Ref(value)
}

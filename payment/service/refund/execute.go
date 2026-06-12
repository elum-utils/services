package refund

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/elum-utils/services/payment/repository"
)

func (a *Refund) Execute(ctx context.Context, params Params) (*Result, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	order, err := a.repository.GetOrder(ctx, params.OrderID)
	if err != nil {
		return nil, err
	}
	if params.WorkspaceID != "" && params.WorkspaceID != order.WorkspaceID {
		return nil, sql.ErrNoRows
	}

	attempt, err := a.refundAttempt(ctx, order, params.AttemptID)
	if err != nil {
		return nil, err
	}
	amount := params.AmountMinor
	if amount == 0 {
		amount = attempt.AmountMinor
	}
	if amount == 0 || amount > attempt.AmountMinor {
		return nil, ErrAmountInvalid
	}

	providerRefund, ok := a.providers[attempt.ProviderCode]
	if !ok || providerRefund == nil {
		return nil, fmt.Errorf("%w: %s", ErrProviderUnsupported, attempt.ProviderCode)
	}

	id, err := a.repository.CreateRefund(ctx, repository.RefundCreateParams{
		OrderID:      order.ID,
		AttemptID:    attempt.ID,
		ProviderCode: attempt.ProviderCode,
		AmountMinor:  amount,
		AssetCode:    attempt.AssetCode,
		Status:       "pending",
		Reason:       refIfNotEmpty(params.Reason),
	})
	if err != nil {
		return nil, err
	}

	providerResult, providerErr := providerRefund(ctx, ProviderRefundParams{
		Order: ProviderRefundOrder{
			ID:             order.ID,
			WorkspaceID:    order.WorkspaceID,
			AppID:          order.AppID,
			PlatformID:     order.PlatformID,
			PlatformUserID: order.PlatformUserID,
			ProductID:      order.ProductID,
		},
		Attempt: ProviderRefundAttempt{
			ID:                attempt.ID,
			ProviderCode:      attempt.ProviderCode,
			AssetCode:         attempt.AssetCode,
			AmountMinor:       attempt.AmountMinor,
			ProviderPaymentID: attempt.ProviderPaymentID,
			ProviderChargeID:  attempt.ProviderChargeID,
		},
		RefundID:       id,
		AmountMinor:    amount,
		Reason:         params.Reason,
		ProviderParams: params.ProviderParams,
	})
	if providerErr != nil {
		_, _ = a.repository.UpdateRefundStatus(ctx, id, "failed", providerErr.Error())
		return nil, providerErr
	}

	if providerResult.ProviderRefundID != "" {
		if _, err := a.repository.SetRefundProviderID(ctx, id, providerResult.ProviderRefundID); err != nil {
			return nil, err
		}
	}
	status := providerResult.Status
	if status == "" {
		status = "succeeded"
	}
	if _, err := a.repository.UpdateRefundStatus(ctx, id, status, params.Reason); err != nil {
		return nil, err
	}
	if status == "succeeded" && amount == attempt.AmountMinor {
		if err := a.repository.UpdateAttemptStatus(ctx, attempt.ID, "refunded"); err != nil {
			return nil, err
		}
		if _, err := a.repository.UpdateOrderStatus(ctx, order.WorkspaceID, order.ID, "refunded"); err != nil {
			return nil, err
		}
	}

	return &Result{
		RefundID:         id,
		OrderID:          order.ID,
		AttemptID:        attempt.ID,
		ProviderCode:     attempt.ProviderCode,
		ProviderRefundID: refIfNotEmpty(providerResult.ProviderRefundID),
		AmountMinor:      amount,
		AssetCode:        attempt.AssetCode,
		Status:           status,
	}, nil
}

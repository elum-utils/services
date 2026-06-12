package admin

import (
	"context"
	"database/sql"

	"github.com/elum-utils/services/payment/repository"
	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (a *Admin) ListPurchaseKeys(ctx context.Context, params PurchaseKeyListParams) ([]paymentsqlc.PaymentPurchaseKey, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListPurchaseKeys(ctx, paymentsqlc.AdminListPurchaseKeysParams{
		WorkspaceID:    params.WorkspaceID,
		Column2:        params.ProductID,
		ProductID:      params.ProductID,
		Column4:        params.Status,
		Status:         paymentsqlc.PaymentPurchaseKeyStatus(params.Status),
		Column6:        params.PlatformID,
		PlatformID:     params.PlatformID,
		Column8:        params.PlatformUserID,
		PlatformUserID: params.PlatformUserID,
		Limit:          limit,
		Offset:         offset,
	})
}

func (a *Admin) GetPurchaseKey(ctx context.Context, workspaceID string, id uint64) (paymentsqlc.PaymentPurchaseKey, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetPurchaseKey(ctx, paymentsqlc.AdminGetPurchaseKeyParams{WorkspaceID: workspaceID, ID: id})
}

func (a *Admin) UpdatePurchaseKeyStatus(ctx context.Context, workspaceID string, id uint64, status string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminUpdatePurchaseKeyStatus(ctx, paymentsqlc.AdminUpdatePurchaseKeyStatusParams{
		WorkspaceID: workspaceID,
		ID:          id,
		Status:      paymentsqlc.PaymentPurchaseKeyStatus(status),
	})
}

func (a *Admin) ListOrders(ctx context.Context, params OrderListParams) ([]paymentsqlc.PaymentOrder, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListOrders(ctx, paymentsqlc.AdminListOrdersParams{
		WorkspaceID:    params.WorkspaceID,
		Column2:        params.Status,
		Status:         paymentsqlc.PaymentOrderStatus(params.Status),
		Column4:        params.ProductID,
		ProductID:      params.ProductID,
		Column6:        params.PlatformID,
		PlatformID:     params.PlatformID,
		Column8:        params.PlatformUserID,
		PlatformUserID: params.PlatformUserID,
		Limit:          limit,
		Offset:         offset,
	})
}

func (a *Admin) GetOrder(ctx context.Context, id uint64) (paymentsqlc.PaymentOrder, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetOrder(ctx, id)
}

func (a *Admin) GetOrderByPublicID(ctx context.Context, publicID string) (paymentsqlc.PaymentOrder, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetOrderByPublicID(ctx, publicID)
}

func (a *Admin) UpdateOrderStatus(ctx context.Context, workspaceID string, id uint64, status string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.UpdateOrderStatus(ctx, workspaceID, id, status)
}

func (a *Admin) ListPaymentAttempts(ctx context.Context, params AttemptListParams) ([]paymentsqlc.PaymentAttempt, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListPaymentAttempts(ctx, paymentsqlc.AdminListPaymentAttemptsParams{
		WorkspaceID:  params.WorkspaceID,
		Column2:      params.OrderID,
		OrderID:      params.OrderID,
		Column4:      params.ProviderCode,
		ProviderCode: params.ProviderCode,
		Column6:      params.Status,
		Status:       paymentsqlc.PaymentAttemptStatus(params.Status),
		Limit:        limit,
		Offset:       offset,
	})
}

func (a *Admin) GetPaymentAttempt(ctx context.Context, id uint64) (paymentsqlc.PaymentAttempt, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetPaymentAttempt(ctx, id)
}

func (a *Admin) UpdatePaymentAttemptStatus(ctx context.Context, id uint64, status string) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.UpdateAttemptStatus(ctx, id, status)
}

func (a *Admin) ListPaymentEvents(ctx context.Context, params EventListParams) ([]paymentsqlc.PaymentEvent, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListPaymentEvents(ctx, paymentsqlc.AdminListPaymentEventsParams{
		WorkspaceID:      params.WorkspaceID,
		WorkspaceID_2:    params.WorkspaceID,
		Column3:          params.ProviderCode,
		ProviderCode:     params.ProviderCode,
		Column5:          params.ProcessingStatus,
		ProcessingStatus: paymentsqlc.PaymentEventProcessingStatus(params.ProcessingStatus),
		Limit:            limit,
		Offset:           offset,
	})
}

func (a *Admin) GetPaymentEvent(ctx context.Context, id uint64) (paymentsqlc.PaymentEvent, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetPaymentEvent(ctx, id)
}

func (a *Admin) UpdatePaymentEventProcessingStatus(ctx context.Context, id uint64, status string, message string) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminUpdatePaymentEventProcessingStatus(ctx, paymentsqlc.MarkPaymentEventProcessedParams{
		ID:               id,
		ProcessingStatus: paymentsqlc.PaymentEventProcessingStatus(status),
		ProcessingError:  sql.NullString{String: message, Valid: message != ""},
	})
}

func (a *Admin) ListSubscriptions(ctx context.Context, params SubscriptionListParams) ([]paymentsqlc.PaymentSubscription, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListSubscriptions(ctx, paymentsqlc.AdminListSubscriptionsParams{
		WorkspaceID:    params.WorkspaceID,
		Column2:        params.ProviderCode,
		ProviderCode:   params.ProviderCode,
		Column4:        params.ProductID,
		ProductID:      params.ProductID,
		Column6:        params.Status,
		Status:         paymentsqlc.PaymentSubscriptionStatus(params.Status),
		Column8:        params.PlatformID,
		PlatformID:     params.PlatformID,
		Column10:       params.PlatformUserID,
		PlatformUserID: params.PlatformUserID,
		Limit:          limit,
		Offset:         offset,
	})
}

func (a *Admin) GetSubscription(ctx context.Context, workspaceID string, id uint64) (paymentsqlc.PaymentSubscription, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetSubscription(ctx, paymentsqlc.AdminGetSubscriptionParams{WorkspaceID: workspaceID, ID: id})
}

func (a *Admin) GetSubscriptionByProviderID(ctx context.Context, providerCode string, providerSubscriptionID string) (paymentsqlc.PaymentSubscription, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetSubscriptionByProviderID(ctx, paymentsqlc.GetPaymentSubscriptionByProviderIDParams{
		ProviderCode:           providerCode,
		ProviderSubscriptionID: providerSubscriptionID,
	})
}

func (a *Admin) UpsertSubscription(ctx context.Context, params paymentsqlc.UpsertPaymentSubscriptionParams) (uint64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminUpsertSubscription(ctx, params)
}

func (a *Admin) UpdateSubscriptionStatus(ctx context.Context, params paymentsqlc.UpdatePaymentSubscriptionStatusParams) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminUpdateSubscriptionStatus(ctx, params)
}

func (a *Admin) ListFulfillments(ctx context.Context, params FulfillmentListParams) ([]paymentsqlc.PaymentFulfillment, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListFulfillments(ctx, paymentsqlc.AdminListFulfillmentsParams{
		WorkspaceID: params.WorkspaceID,
		Column2:     params.Status,
		Status:      paymentsqlc.PaymentFulfillmentStatus(params.Status),
		Column4:     params.OrderID,
		OrderID:     params.OrderID,
		Limit:       limit,
		Offset:      offset,
	})
}

func (a *Admin) GetFulfillment(ctx context.Context, id uint64) (paymentsqlc.PaymentFulfillment, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetFulfillment(ctx, id)
}

func (a *Admin) UpdateFulfillmentStatus(ctx context.Context, id uint64, status string, message string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminUpdateFulfillmentStatus(ctx, paymentsqlc.AdminUpdateFulfillmentStatusParams{
		ID:      id,
		Status:  paymentsqlc.PaymentFulfillmentStatus(status),
		Error:   sql.NullString{String: message, Valid: message != ""},
		Column3: status,
		Column4: status,
	})
}

func (a *Admin) ListFulfillmentItems(ctx context.Context, params FulfillmentItemListParams) ([]paymentsqlc.PaymentFulfillmentItem, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListFulfillmentItems(ctx, paymentsqlc.AdminListFulfillmentItemsParams{
		WorkspaceID:   params.WorkspaceID,
		Column2:       params.FulfillmentID,
		FulfillmentID: params.FulfillmentID,
		Limit:         limit,
		Offset:        offset,
	})
}

func (a *Admin) CreateRefund(ctx context.Context, params RefundCreateParams) (uint64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	status := params.Status
	if status == "" {
		status = string(paymentsqlc.PaymentRefundStatusCreated)
	}
	return a.repository.CreateRefund(ctx, repository.RefundCreateParams{
		OrderID:          params.OrderID,
		AttemptID:        params.AttemptID,
		ProviderCode:     params.ProviderCode,
		ProviderRefundID: params.ProviderRefundID,
		AmountMinor:      params.AmountMinor,
		AssetCode:        params.AssetCode,
		Status:           status,
		Reason:           params.Reason,
	})
}

func (a *Admin) ListRefunds(ctx context.Context, params RefundListParams) ([]paymentsqlc.PaymentRefund, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	limit, offset := normalizePage(params.Page)
	return a.repository.AdminListRefunds(ctx, paymentsqlc.AdminListRefundsParams{
		WorkspaceID:  params.WorkspaceID,
		Column2:      params.OrderID,
		OrderID:      params.OrderID,
		Column4:      params.ProviderCode,
		ProviderCode: params.ProviderCode,
		Column6:      params.Status,
		Status:       paymentsqlc.PaymentRefundStatus(params.Status),
		Limit:        limit,
		Offset:       offset,
	})
}

func (a *Admin) GetRefund(ctx context.Context, id uint64) (paymentsqlc.PaymentRefund, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.AdminGetRefund(ctx, id)
}

func (a *Admin) UpdateRefundStatus(ctx context.Context, id uint64, status string, reason string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	return a.repository.UpdateRefundStatus(ctx, id, status, reason)
}

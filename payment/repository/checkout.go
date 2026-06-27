package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	json "github.com/goccy/go-json"
	"time"

	serviceerrors "github.com/elum-utils/services/errors"
	utils "github.com/elum-utils/services/internal/utils"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/payment/sqlc"

	"github.com/google/uuid"
)

var (
	ErrProductLocked         = serviceerrors.New(serviceerrors.CodeFailedPrecondition, "payment product limit is locked")
	ErrPaymentMismatch       = serviceerrors.New(serviceerrors.CodeConflict, "payment data mismatch")
	ErrOrderStateInvalid     = serviceerrors.New(serviceerrors.CodeFailedPrecondition, "payment order state is invalid")
	ErrPaymentAmountOverflow = serviceerrors.New(serviceerrors.CodeInvalidFields, "payment amount overflow")
	ErrProductQuantityFixed  = serviceerrors.New(serviceerrors.CodeFailedPrecondition, "payment product quantity is fixed")
)

type OrderCreateParams struct {
	WorkspaceID         string
	AppID               int64
	PlatformID          int64
	PlatformUserID      string
	InternalUserID      *int64
	PayerPlatformID     *int64
	PayerPlatformUserID *string
	PayerInternalUserID *int64
	PurchaseKeyID       *int64
	ProductID           string
	Quantity            uint64
	AssetCode           string
	Locale              string
	ReservedUntil       *time.Time
	ExpiresAt           *time.Time
}

type OrderCreateByKeyParams struct {
	Key                 string
	PayerPlatformID     *int64
	PayerPlatformUserID *string
	PayerInternalUserID *int64
	AssetCode           string
	Locale              string
	Quantity            uint64
	ReservedUntil       *time.Time
	ExpiresAt           *time.Time
	Now                 time.Time
}

type Order struct {
	ID                  uint64
	PublicID            string
	WorkspaceID         string
	AppID               int64
	PlatformID          int64
	PlatformUserID      string
	InternalUserID      *int64
	PayerPlatformID     *int64
	PayerPlatformUserID *string
	PayerInternalUserID *int64
	PurchaseKeyID       *int64
	ProductID           string
	Quantity            uint64
	PriceID             uint64
	AssetCode           string
	Locale              string
	ListAmountMinor     uint64
	DiscountAmountMinor uint64
	PayableAmountMinor  uint64
	Status              string
}

type AttemptCreateParams struct {
	OrderID                uint64
	ProviderCode           string
	ProviderPaymentID      *string
	ProviderInvoiceID      *string
	ProviderChargeID       *string
	ProviderSubscriptionID *string
	IdempotencyKey         *string
	ConfirmationURL        *string
	ReturnURL              *string
	ExpiresAt              *time.Time
	KnownAssetCode         *string
	KnownAmountMinor       *uint64
}

type Attempt struct {
	ID                     uint64
	OrderID                uint64
	ProviderCode           string
	AssetCode              string
	AmountMinor            uint64
	Status                 string
	ProviderPaymentID      *string
	ProviderInvoiceID      *string
	ProviderChargeID       *string
	ProviderSubscriptionID *string
}

type EventCreateParams struct {
	ProviderCode      string
	AttemptID         *int64
	OrderID           *int64
	ProviderEventID   *string
	ProviderPaymentID *string
	EventType         string
	EventStatus       *string
	PayloadHash       string
	SignatureValid    *bool
}

type CompleteAttemptParams struct {
	AttemptID         uint64
	ProviderCode      string
	ProviderPaymentID *string
	AmountMinor       uint64
	AssetCode         string
}

type CompleteAttemptResult struct {
	OrderID       uint64
	AttemptID     uint64
	FulfillmentID *int64
	AlreadyDone   bool
}

type paymentFulfilledCallbackPayload struct {
	OrderID           uint64                  `json:"order_id"`
	AttemptID         uint64                  `json:"attempt_id"`
	FulfillmentID     uint64                  `json:"fulfillment_id"`
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
	Rewards           []paymentCallbackReward `json:"rewards"`
}

type paymentCallbackReward struct {
	Key      string  `json:"key"`
	Type     string  `json:"type"`
	Quantity int64   `json:"quantity"`
	Scale    uint16  `json:"scale"`
	Unit     *string `json:"unit,omitempty"`
}

func (r *PaymentRepository) CreateOrder(ctx context.Context, params OrderCreateParams) (Order, error) {
	workspaceID, err := requireWorkspaceID(params.WorkspaceID)
	if err != nil {
		return Order{}, err
	}
	var order Order
	err = r.inTransaction(ctx, func(txRepo *PaymentRepository) error {
		product, err := txRepo.GetProduct(ctx, ProductGetParams{
			AppID:          params.AppID,
			WorkspaceID:    workspaceID,
			PlatformID:     params.PlatformID,
			PlatformUserID: params.PlatformUserID,
			ProductID:      params.ProductID,
			AssetCode:      params.AssetCode,
			Locale:         params.Locale,
		})
		if err != nil {
			return err
		}
		if product.Limit.Global.LockUntil.Valid || product.Limit.User.LockUntil.Valid {
			return ErrProductLocked
		}
		quantity := normalizeOrderQuantity(params.Quantity)
		if product.QuantityMode != string(sqlc.PaymentProductCacheQuantityModeFlexible) && quantity != 1 {
			return ErrProductQuantityFixed
		}
		if err := txRepo.ensureProductLimitAvailable(ctx, product, params.PlatformID, params.PlatformUserID, quantity); err != nil {
			return err
		}
		listAmountMinor, err := multiplyMinorAmount(product.Price.ListAmountMinor, quantity)
		if err != nil {
			return err
		}
		discountAmountMinor, err := multiplyMinorAmount(product.Price.DiscountAmountMinor, quantity)
		if err != nil {
			return err
		}
		payableAmountMinor, err := multiplyMinorAmount(product.Price.PayableAmountMinor, quantity)
		if err != nil {
			return err
		}
		publicID := uuid.NewString()
		id, err := txRepo.q.CreatePaymentOrder(ctx, sqlc.CreatePaymentOrderParams{
			PublicID:       publicID,
			WorkspaceID:    product.WorkspaceID,
			AppID:          params.AppID,
			PlatformID:     params.PlatformID,
			PlatformUserID: params.PlatformUserID,
			InternalUserID: sqlwrap.NullFromPtr(params.InternalUserID, func(v int64) sql.NullInt64 {
				return sql.NullInt64{Int64: v, Valid: true}
			}),
			PayerPlatformID: sqlwrap.NullFromPtr(params.PayerPlatformID, func(v int64) sql.NullInt64 {
				return sql.NullInt64{Int64: v, Valid: true}
			}),
			PayerPlatformUserID: sqlwrap.NullFromPtr(params.PayerPlatformUserID, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
			PayerInternalUserID: sqlwrap.NullFromPtr(params.PayerInternalUserID, func(v int64) sql.NullInt64 {
				return sql.NullInt64{Int64: v, Valid: true}
			}),
			PurchaseKeyID: sqlwrap.NullFromPtr(params.PurchaseKeyID, func(v int64) sql.NullInt64 {
				return sql.NullInt64{Int64: v, Valid: true}
			}),
			ProductID:           product.ID,
			Quantity:            quantity,
			PriceID:             product.Price.ID,
			AssetCode:           product.Price.AssetCode,
			Locale:              normalizedLocale(params.Locale),
			ListAmountMinor:     listAmountMinor,
			DiscountAmountMinor: discountAmountMinor,
			PayableAmountMinor:  payableAmountMinor,
			Status:              sqlc.PaymentOrderStatusDraft,
			ReservedUntil:       sqlwrap.NullTimeFromPtr(params.ReservedUntil),
			ExpiresAt:           sqlwrap.NullTimeFromPtr(params.ExpiresAt),
		})
		if err != nil {
			return err
		}
		orderID := uint64(id)
		if err := txRepo.q.SnapshotPaymentOrderItems(ctx, sqlc.SnapshotPaymentOrderItemsParams{
			OrderID:     orderID,
			Quantity:    int64(quantity),
			WorkspaceID: product.WorkspaceID,
			ProductID:   product.ID,
		}); err != nil {
			return err
		}

		order = Order{
			ID:                  orderID,
			PublicID:            publicID,
			WorkspaceID:         product.WorkspaceID,
			AppID:               params.AppID,
			PlatformID:          params.PlatformID,
			PlatformUserID:      params.PlatformUserID,
			InternalUserID:      params.InternalUserID,
			PayerPlatformID:     params.PayerPlatformID,
			PayerPlatformUserID: params.PayerPlatformUserID,
			PayerInternalUserID: params.PayerInternalUserID,
			PurchaseKeyID:       params.PurchaseKeyID,
			ProductID:           product.ID,
			Quantity:            quantity,
			PriceID:             product.Price.ID,
			AssetCode:           product.Price.AssetCode,
			Locale:              normalizedLocale(params.Locale),
			ListAmountMinor:     listAmountMinor,
			DiscountAmountMinor: discountAmountMinor,
			PayableAmountMinor:  payableAmountMinor,
			Status:              string(sqlc.PaymentOrderStatusDraft),
		}
		return nil
	})
	return order, err
}

func (r *PaymentRepository) CreateOrderByKey(ctx context.Context, params OrderCreateByKeyParams) (Order, error) {
	var order Order
	err := r.WithTx(ctx, func(txRepo *PaymentRepository) error {
		now := params.Now
		if now.IsZero() {
			now = time.Now()
		}

		key, err := txRepo.q.LockPurchaseKeyByHash(ctx, hashPurchaseKey(params.Key))
		if err != nil {
			return err
		}
		if !isPurchaseKeyUsable(key, now) {
			return sql.ErrNoRows
		}

		order, err = txRepo.CreateOrder(ctx, OrderCreateParams{
			AppID:               key.AppID,
			WorkspaceID:         key.WorkspaceID,
			PlatformID:          key.PlatformID,
			PlatformUserID:      key.PlatformUserID,
			InternalUserID:      nullInt64Ptr(key.InternalUserID),
			PayerPlatformID:     params.PayerPlatformID,
			PayerPlatformUserID: params.PayerPlatformUserID,
			PayerInternalUserID: params.PayerInternalUserID,
			PurchaseKeyID:       utils.Ref(int64(key.ID)),
			ProductID:           key.ProductID,
			AssetCode:           params.AssetCode,
			Locale:              params.Locale,
			Quantity:            params.Quantity,
			ReservedUntil:       params.ReservedUntil,
			ExpiresAt:           params.ExpiresAt,
		})
		return err
	})
	return order, err
}

func normalizeOrderQuantity(quantity uint64) uint64 {
	if quantity == 0 {
		return 1
	}
	return quantity
}

func multiplyMinorAmount(amount uint64, quantity uint64) (uint64, error) {
	quantity = normalizeOrderQuantity(quantity)
	if quantity > uint64(1<<63-1) {
		return 0, ErrPaymentAmountOverflow
	}
	if amount != 0 && quantity > ^uint64(0)/amount {
		return 0, ErrPaymentAmountOverflow
	}
	return amount * quantity, nil
}

func (r *PaymentRepository) ensureProductLimitAvailable(ctx context.Context, product Product, platformID int64, platformUserID string, quantity uint64) error {
	globalLock, err := r.getProductLimitLock(ctx, productLimitQuery{
		workspaceID:    product.WorkspaceID,
		platformID:     platformID,
		platformUserID: "",
		productID:      product.ID,
		limit:          product.Limit.Global.Limit,
		interval:       product.Limit.Global.Interval,
		intervalCount:  product.Limit.Global.IntervalCount,
		amount:         quantity,
	})
	if err != nil {
		return err
	}
	if globalLock.Valid {
		return ErrProductLocked
	}

	userLock, err := r.getProductLimitLock(ctx, productLimitQuery{
		workspaceID:    product.WorkspaceID,
		platformID:     platformID,
		platformUserID: platformUserID,
		productID:      product.ID,
		limit:          product.Limit.User.Limit,
		interval:       product.Limit.User.Interval,
		intervalCount:  product.Limit.User.IntervalCount,
		amount:         quantity,
	})
	if err != nil {
		return err
	}
	if userLock.Valid {
		return ErrProductLocked
	}
	return nil
}

func (r *PaymentRepository) GetOrder(ctx context.Context, id uint64) (Order, error) {
	order, err := r.q.GetPaymentOrder(ctx, id)
	if err != nil {
		return Order{}, err
	}
	return mapOrder(order), nil
}

func (r *PaymentRepository) GetAttemptByProviderPaymentID(ctx context.Context, providerCode string, providerPaymentID string) (Attempt, error) {
	attempt, err := r.q.GetPaymentAttemptByProviderPaymentID(ctx, sqlc.GetPaymentAttemptByProviderPaymentIDParams{
		ProviderCode:      providerCode,
		ProviderPaymentID: sql.NullString{String: providerPaymentID, Valid: true},
	})
	if err != nil {
		return Attempt{}, err
	}
	return mapAttempt(attempt), nil
}

func (r *PaymentRepository) CreateAttempt(ctx context.Context, params AttemptCreateParams) (Attempt, error) {
	var attempt Attempt
	err := r.WithTx(ctx, func(txRepo *PaymentRepository) error {
		useKnownOrderData := params.KnownAssetCode != nil && params.KnownAmountMinor != nil

		id, err := txRepo.q.CreatePaymentAttemptFromOrder(ctx, sqlc.CreatePaymentAttemptFromOrderParams{
			ProviderCode: params.ProviderCode,
			Status:       sqlc.PaymentAttemptStatusPending,
			ProviderPaymentID: sqlwrap.NullFromPtr(params.ProviderPaymentID, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
			ProviderInvoiceID: sqlwrap.NullFromPtr(params.ProviderInvoiceID, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
			ProviderChargeID: sqlwrap.NullFromPtr(params.ProviderChargeID, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
			ProviderSubscriptionID: sqlwrap.NullFromPtr(params.ProviderSubscriptionID, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
			IdempotencyKey: sqlwrap.NullFromPtr(params.IdempotencyKey, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
			ConfirmationUrl: sqlwrap.NullFromPtr(params.ConfirmationURL, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
			ReturnUrl: sqlwrap.NullFromPtr(params.ReturnURL, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			}),
			ExpiresAt:      sqlwrap.NullTimeFromPtr(params.ExpiresAt),
			ProviderCode_2: params.ProviderCode,
			ID:             params.OrderID,
		})
		if err != nil {
			return err
		}
		if id == 0 {
			order, orderErr := txRepo.q.GetPaymentOrder(ctx, params.OrderID)
			if orderErr != nil {
				return orderErr
			}
			if order.Status != sqlc.PaymentOrderStatusDraft && order.Status != sqlc.PaymentOrderStatusPendingPayment {
				return ErrOrderStateInvalid
			}
			return sql.ErrNoRows
		}

		if _, err := txRepo.q.MarkOrderPendingPayment(ctx, params.OrderID); err != nil {
			return err
		}

		assetCode := sqlwrap.ValueFromPtr(params.KnownAssetCode)
		amountMinor := sqlwrap.ValueFromPtr(params.KnownAmountMinor)
		if !useKnownOrderData {
			order, orderErr := txRepo.q.GetPaymentOrder(ctx, params.OrderID)
			if orderErr != nil {
				return orderErr
			}
			assetCode = order.AssetCode
			amountMinor = order.PayableAmountMinor
		}

		attempt = Attempt{
			ID:                uint64(id),
			OrderID:           params.OrderID,
			ProviderCode:      params.ProviderCode,
			AssetCode:         assetCode,
			AmountMinor:       amountMinor,
			Status:            string(sqlc.PaymentAttemptStatusPending),
			ProviderPaymentID: params.ProviderPaymentID,
		}
		return nil
	})
	return attempt, err
}

func (r *PaymentRepository) CreateEvent(ctx context.Context, params EventCreateParams) (uint64, error) {
	id, err := r.q.CreatePaymentEvent(ctx, sqlc.CreatePaymentEventParams{
		ProviderCode: params.ProviderCode,
		AttemptID: sqlwrap.NullFromPtr(params.AttemptID, func(v int64) sql.NullInt64 {
			return sql.NullInt64{Int64: v, Valid: true}
		}),
		OrderID: sqlwrap.NullFromPtr(params.OrderID, func(v int64) sql.NullInt64 {
			return sql.NullInt64{Int64: v, Valid: true}
		}),
		ProviderEventID: sqlwrap.NullFromPtr(params.ProviderEventID, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		ProviderPaymentID: sqlwrap.NullFromPtr(params.ProviderPaymentID, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		EventType: params.EventType,
		EventStatus: sqlwrap.NullFromPtr(params.EventStatus, func(v string) sql.NullString {
			return sql.NullString{String: v, Valid: true}
		}),
		PayloadHash: params.PayloadHash,
		SignatureValid: sqlwrap.NullFromPtr(params.SignatureValid, func(v bool) sql.NullBool {
			return sql.NullBool{Bool: v, Valid: true}
		}),
	})
	if err != nil {
		return 0, err
	}
	return uint64(id), nil
}

func (r *PaymentRepository) SetAttemptProviderChargeID(ctx context.Context, attemptID uint64, providerCode string, chargeID string) (int64, error) {
	return r.q.SetPaymentAttemptProviderChargeID(ctx, sqlc.SetPaymentAttemptProviderChargeIDParams{
		ProviderChargeID: sql.NullString{String: chargeID, Valid: chargeID != ""},
		ID:               attemptID,
		ProviderCode:     providerCode,
		ProviderChargeID_2: sql.NullString{
			String: chargeID,
			Valid:  chargeID != "",
		},
	})
}

func (r *PaymentRepository) CompleteAttempt(ctx context.Context, params CompleteAttemptParams) (CompleteAttemptResult, error) {
	fulfilled, err := r.q.GetFulfilledAttemptResult(ctx, params.AttemptID)
	if err == nil {
		return CompleteAttemptResult{
			OrderID:     fulfilled.OrderID,
			AttemptID:   fulfilled.AttemptID,
			AlreadyDone: true,
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CompleteAttemptResult{}, err
	}

	var result CompleteAttemptResult
	err = r.WithTx(ctx, func(txRepo *PaymentRepository) error {
		attempt, err := txRepo.q.LockPaymentAttempt(ctx, params.AttemptID)
		if err != nil {
			return err
		}
		order, err := txRepo.q.LockPaymentOrder(ctx, attempt.OrderID)
		if err != nil {
			return err
		}

		result.OrderID = order.ID
		result.AttemptID = attempt.ID

		if order.Status == sqlc.PaymentOrderStatusFulfilled {
			result.AlreadyDone = true
			return nil
		}

		if attempt.ProviderCode != params.ProviderCode ||
			attempt.AssetCode != params.AssetCode ||
			attempt.AmountMinor != params.AmountMinor ||
			!sameProviderPaymentID(attempt.ProviderPaymentID, sqlwrap.NullFromPtr(params.ProviderPaymentID, func(v string) sql.NullString {
				return sql.NullString{String: v, Valid: true}
			})) {
			return ErrPaymentMismatch
		}

		if err := txRepo.q.UpdatePaymentAttemptStatus(ctx, sqlc.UpdatePaymentAttemptStatusParams{
			Status: sqlc.PaymentAttemptStatusSucceeded,
			ID:     attempt.ID,
		}); err != nil {
			return err
		}

		if order.Status == sqlc.PaymentOrderStatusDraft || order.Status == sqlc.PaymentOrderStatusPendingPayment {
			if _, err := txRepo.q.MarkOrderPaid(ctx, order.ID); err != nil {
				return err
			}
		} else if order.Status != sqlc.PaymentOrderStatusPaid {
			return ErrOrderStateInvalid
		}
		if err := txRepo.indexPaidOrderAndIncrementLimits(ctx, order); err != nil {
			return err
		}

		fulfillmentID, err := txRepo.q.CreateFulfillment(ctx, sqlc.CreateFulfillmentParams{
			OrderID:        order.ID,
			AttemptID:      attempt.ID,
			InternalUserID: order.InternalUserID,
			Status:         sqlc.PaymentFulfillmentStatusSucceeded,
		})
		if err != nil {
			return err
		}
		result.FulfillmentID = utils.Ref(fulfillmentID)

		items, err := txRepo.q.GetFulfillmentItemsForOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := txRepo.q.CreateFulfillmentItem(ctx, sqlc.CreateFulfillmentItemParams{
				FulfillmentID: uint64(fulfillmentID),
				WorkspaceID:   order.WorkspaceID,
				ItemID:        item.ItemID,
				RewardType:    sqlc.PaymentFulfillmentItemRewardType(item.RewardType),
				Quantity:      item.Quantity,
				Scale:         item.Scale,
				DurationUnit: sqlc.NullPaymentFulfillmentItemDurationUnit{
					PaymentFulfillmentItemDurationUnit: sqlc.PaymentFulfillmentItemDurationUnit(
						orderDurationUnitValue(item.DurationUnit),
					),
					Valid: item.DurationUnit.Valid,
				},
			}); err != nil {
				return err
			}
		}

		if _, err := txRepo.q.MarkOrderFulfilled(ctx, order.ID); err != nil {
			return err
		}
		if _, err := txRepo.q.MarkPaidOrderIndexFulfilled(ctx, order.ID); err != nil {
			return err
		}

		if order.PurchaseKeyID.Valid {
			if _, err := txRepo.q.IncrementPurchaseKeyUsage(ctx, uint64(order.PurchaseKeyID.Int64)); err != nil {
				return err
			}
		}

		if err := txRepo.enqueuePaymentFulfilledCallback(ctx, order, attempt, uint64(fulfillmentID), items); err != nil {
			return err
		}

		return nil
	})
	return result, err
}

func (r *PaymentRepository) enqueuePaymentFulfilledCallback(
	ctx context.Context,
	order sqlc.PaymentOrder,
	attempt sqlc.PaymentAttempt,
	fulfillmentID uint64,
	items []sqlc.GetFulfillmentItemsForOrderRow,
) error {
	payload := paymentFulfilledCallbackPayload{
		OrderID:        order.ID,
		AttemptID:      attempt.ID,
		FulfillmentID:  fulfillmentID,
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
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	eventKey := fmt.Sprintf("payment.order.fulfilled:%d", order.ID)
	_, err = r.callbacks.CreateEvent(ctx, callbackutil.CreateParams{
		SourceService:      "payment",
		EventType:          "payment.order.fulfilled",
		EventKey:           eventKey,
		IdempotencyKey:     eventKey,
		Payload:            raw,
		PayloadContentType: callbackutil.JSONContentType,
	})
	return err
}

func orderDurationUnitValue(value sqlc.NullPaymentOrderItemDurationUnit) string {
	if !value.Valid {
		return ""
	}
	return string(value.PaymentOrderItemDurationUnit)
}

func orderDurationUnitPtr(value sqlc.NullPaymentOrderItemDurationUnit) *string {
	if !value.Valid {
		return nil
	}
	unit := string(value.PaymentOrderItemDurationUnit)
	return &unit
}

func (r *PaymentRepository) indexPaidOrderAndIncrementLimits(ctx context.Context, order sqlc.PaymentOrder) error {
	rows, err := r.q.InsertPaidOrderIndexFromOrder(ctx, order.ID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return nil
	}

	config, err := r.q.GetProductLimitConfig(ctx, sqlc.GetProductLimitConfigParams{
		WorkspaceID: order.WorkspaceID,
		ID:          order.ProductID,
	})
	if err != nil {
		return err
	}

	if err := r.incrementProductLimitCounter(ctx, productLimitQuery{
		workspaceID:    order.WorkspaceID,
		platformID:     order.PlatformID,
		platformUserID: "",
		productID:      order.ProductID,
		limit:          config.GlobalLimit,
		interval:       string(config.GlobalInterval),
		intervalCount:  config.GlobalIntervalCount,
		amount:         order.Quantity,
	}); err != nil {
		return err
	}

	return r.incrementProductLimitCounter(ctx, productLimitQuery{
		workspaceID:    order.WorkspaceID,
		platformID:     order.PlatformID,
		platformUserID: order.PlatformUserID,
		productID:      order.ProductID,
		limit:          config.UserLimit,
		interval:       string(config.UserInterval),
		intervalCount:  config.UserIntervalCount,
		amount:         order.Quantity,
	})
}

func (r *PaymentRepository) incrementProductLimitCounter(ctx context.Context, query productLimitQuery) error {
	if query.limit <= 0 || query.interval == "UNLIMITED" {
		return nil
	}

	now, err := r.databaseNow(ctx)
	if err != nil {
		return err
	}

	start, end, ok := limitWindow(query.interval, query.intervalCount, now)
	if !ok {
		return nil
	}

	scope := sqlc.PaymentProductLimitCounterCounterScopeGlobal
	platformUserID := ""
	if query.platformUserID != "" {
		scope = sqlc.PaymentProductLimitCounterCounterScopeUser
		platformUserID = query.platformUserID
	}

	ensureParams := sqlc.EnsureProductLimitCounterParams{
		WorkspaceID:    query.workspaceID,
		PlatformID:     query.platformID,
		ProductID:      query.productID,
		CounterScope:   scope,
		PlatformUserID: platformUserID,
		WindowStart:    start,
		WindowEnd:      end,
	}
	if _, err := r.q.EnsureProductLimitCounter(ctx, ensureParams); err != nil {
		return err
	}

	rows, err := r.q.IncrementProductLimitCounter(ctx, sqlc.IncrementProductLimitCounterParams{
		PaidCount:      normalizeLimitAmount(query.amount),
		WorkspaceID:    ensureParams.WorkspaceID,
		PlatformID:     ensureParams.PlatformID,
		ProductID:      ensureParams.ProductID,
		CounterScope:   ensureParams.CounterScope,
		PlatformUserID: ensureParams.PlatformUserID,
		WindowStart:    ensureParams.WindowStart,
		WindowEnd:      ensureParams.WindowEnd,
		PaidCount_2:    normalizeLimitAmount(query.amount),
		PaidCount_3:    uint64(query.limit),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrProductLocked
	}
	return nil
}

func sameProviderPaymentID(stored sql.NullString, received sql.NullString) bool {
	if stored.Valid != received.Valid {
		return false
	}
	if !stored.Valid {
		return true
	}
	return stored.String == received.String
}

func mapOrder(order sqlc.PaymentOrder) Order {
	return Order{
		ID:                  order.ID,
		PublicID:            order.PublicID,
		WorkspaceID:         order.WorkspaceID,
		AppID:               order.AppID,
		PlatformID:          order.PlatformID,
		PlatformUserID:      order.PlatformUserID,
		InternalUserID:      nullInt64Ptr(order.InternalUserID),
		PayerPlatformID:     nullInt64Ptr(order.PayerPlatformID),
		PayerPlatformUserID: sqlwrap.NullStringPtr(order.PayerPlatformUserID),
		PayerInternalUserID: nullInt64Ptr(order.PayerInternalUserID),
		PurchaseKeyID:       nullInt64Ptr(order.PurchaseKeyID),
		ProductID:           order.ProductID,
		Quantity:            order.Quantity,
		PriceID:             order.PriceID,
		AssetCode:           order.AssetCode,
		Locale:              order.Locale,
		ListAmountMinor:     order.ListAmountMinor,
		DiscountAmountMinor: order.DiscountAmountMinor,
		PayableAmountMinor:  order.PayableAmountMinor,
		Status:              string(order.Status),
	}
}

func mapAttempt(attempt sqlc.PaymentAttempt) Attempt {
	return Attempt{
		ID:                     attempt.ID,
		OrderID:                attempt.OrderID,
		ProviderCode:           attempt.ProviderCode,
		AssetCode:              attempt.AssetCode,
		AmountMinor:            attempt.AmountMinor,
		Status:                 string(attempt.Status),
		ProviderPaymentID:      sqlwrap.NullStringPtr(attempt.ProviderPaymentID),
		ProviderInvoiceID:      sqlwrap.NullStringPtr(attempt.ProviderInvoiceID),
		ProviderChargeID:       sqlwrap.NullStringPtr(attempt.ProviderChargeID),
		ProviderSubscriptionID: sqlwrap.NullStringPtr(attempt.ProviderSubscriptionID),
	}
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func normalizedLocale(locale string) string {
	if locale == "" {
		return "ru"
	}
	return locale
}

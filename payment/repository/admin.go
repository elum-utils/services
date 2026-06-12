package repository

import (
	"context"

	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (r *PaymentRepository) AdminGetProvider(ctx context.Context, code string) (paymentsqlc.PaymentProvider, error) {
	return r.q.AdminGetProvider(ctx, code)
}

func (r *PaymentRepository) AdminUpsertProvider(ctx context.Context, params paymentsqlc.AdminUpsertProviderParams) error {
	if err := r.q.AdminUpsertProvider(ctx, params); err != nil {
		return err
	}
	return r.invalidateAllCache()
}

func (r *PaymentRepository) AdminDeleteProvider(ctx context.Context, code string) (int64, error) {
	rows, err := r.q.AdminDeleteProvider(ctx, code)
	if err != nil {
		return 0, err
	}
	return rows, r.invalidateAllCache()
}

func (r *PaymentRepository) AdminGetAsset(ctx context.Context, code string) (paymentsqlc.PaymentAsset, error) {
	return r.q.AdminGetAsset(ctx, code)
}

func (r *PaymentRepository) AdminUpsertAsset(ctx context.Context, params paymentsqlc.UpsertAssetParams) error {
	if err := r.q.UpsertAsset(ctx, params); err != nil {
		return err
	}
	return r.invalidateAllCache()
}

func (r *PaymentRepository) AdminDeleteAsset(ctx context.Context, code string) (int64, error) {
	rows, err := r.q.DeleteAsset(ctx, code)
	if err != nil {
		return 0, err
	}
	return rows, r.invalidateAllCache()
}

func (r *PaymentRepository) AdminListProviderAssets(ctx context.Context, params paymentsqlc.AdminListProviderAssetsParams) ([]paymentsqlc.PaymentProviderAsset, error) {
	return r.q.AdminListProviderAssets(ctx, params)
}

func (r *PaymentRepository) AdminUpsertProviderAsset(ctx context.Context, params paymentsqlc.UpsertProviderAssetParams) error {
	if err := r.q.UpsertProviderAsset(ctx, params); err != nil {
		return err
	}
	return r.invalidateAllCache()
}

func (r *PaymentRepository) AdminDeleteProviderAsset(ctx context.Context, params paymentsqlc.DeleteProviderAssetParams) (int64, error) {
	rows, err := r.q.DeleteProviderAsset(ctx, params)
	if err != nil {
		return 0, err
	}
	return rows, r.invalidateAllCache()
}

func (r *PaymentRepository) AdminListProductGroups(ctx context.Context, params paymentsqlc.AdminListProductGroupsParams) ([]paymentsqlc.PaymentProductGroup, error) {
	return r.q.AdminListProductGroups(ctx, params)
}

func (r *PaymentRepository) AdminGetProductGroup(ctx context.Context, params paymentsqlc.AdminGetProductGroupParams) (paymentsqlc.PaymentProductGroup, error) {
	return r.q.AdminGetProductGroup(ctx, params)
}

func (r *PaymentRepository) AdminListLocalizations(ctx context.Context, params paymentsqlc.AdminListLocalizationsParams) ([]paymentsqlc.PaymentLocalization, error) {
	return r.q.AdminListLocalizations(ctx, params)
}

func (r *PaymentRepository) AdminGetLocalization(ctx context.Context, params paymentsqlc.AdminGetLocalizationParams) (paymentsqlc.PaymentLocalization, error) {
	return r.q.AdminGetLocalization(ctx, params)
}

func (r *PaymentRepository) AdminListProducts(ctx context.Context, params paymentsqlc.AdminListProductsParams) ([]paymentsqlc.PaymentProduct, error) {
	return r.q.AdminListProducts(ctx, params)
}

func (r *PaymentRepository) AdminGetProduct(ctx context.Context, params paymentsqlc.AdminGetProductParams) (paymentsqlc.PaymentProduct, error) {
	return r.q.AdminGetProduct(ctx, params)
}

func (r *PaymentRepository) AdminListItems(ctx context.Context, params paymentsqlc.AdminListItemsParams) ([]paymentsqlc.PaymentItem, error) {
	return r.q.AdminListItems(ctx, params)
}

func (r *PaymentRepository) AdminGetItem(ctx context.Context, params paymentsqlc.AdminGetItemParams) (paymentsqlc.PaymentItem, error) {
	return r.q.AdminGetItem(ctx, params)
}

func (r *PaymentRepository) AdminListProductItems(ctx context.Context, params paymentsqlc.AdminListProductItemsParams) ([]paymentsqlc.PaymentProductItem, error) {
	return r.q.AdminListProductItems(ctx, params)
}

func (r *PaymentRepository) AdminListPrices(ctx context.Context, params paymentsqlc.AdminListPricesParams) ([]paymentsqlc.PaymentPrice, error) {
	return r.q.AdminListPrices(ctx, params)
}

func (r *PaymentRepository) AdminGetPrice(ctx context.Context, params paymentsqlc.AdminGetPriceParams) (paymentsqlc.PaymentPrice, error) {
	return r.q.AdminGetPrice(ctx, params)
}

func (r *PaymentRepository) AdminListProductLimitCounters(ctx context.Context, params paymentsqlc.AdminListProductLimitCountersParams) ([]paymentsqlc.PaymentProductLimitCounter, error) {
	return r.q.AdminListProductLimitCounters(ctx, params)
}

func (r *PaymentRepository) AdminDeleteProductLimitCounter(ctx context.Context, params paymentsqlc.AdminDeleteProductLimitCounterParams) (int64, error) {
	return r.q.AdminDeleteProductLimitCounter(ctx, params)
}

func (r *PaymentRepository) AdminListPurchaseKeys(ctx context.Context, params paymentsqlc.AdminListPurchaseKeysParams) ([]paymentsqlc.PaymentPurchaseKey, error) {
	return r.q.AdminListPurchaseKeys(ctx, params)
}

func (r *PaymentRepository) AdminGetPurchaseKey(ctx context.Context, params paymentsqlc.AdminGetPurchaseKeyParams) (paymentsqlc.PaymentPurchaseKey, error) {
	return r.q.AdminGetPurchaseKey(ctx, params)
}

func (r *PaymentRepository) AdminUpdatePurchaseKeyStatus(ctx context.Context, params paymentsqlc.AdminUpdatePurchaseKeyStatusParams) (int64, error) {
	return r.q.AdminUpdatePurchaseKeyStatus(ctx, params)
}

func (r *PaymentRepository) AdminListOrders(ctx context.Context, params paymentsqlc.AdminListOrdersParams) ([]paymentsqlc.PaymentOrder, error) {
	return r.q.AdminListOrders(ctx, params)
}

func (r *PaymentRepository) AdminGetOrder(ctx context.Context, id uint64) (paymentsqlc.PaymentOrder, error) {
	return r.q.GetPaymentOrder(ctx, id)
}

func (r *PaymentRepository) AdminGetOrderByPublicID(ctx context.Context, publicID string) (paymentsqlc.PaymentOrder, error) {
	return r.q.GetPaymentOrderByPublicID(ctx, publicID)
}

func (r *PaymentRepository) AdminListPaymentAttempts(ctx context.Context, params paymentsqlc.AdminListPaymentAttemptsParams) ([]paymentsqlc.PaymentAttempt, error) {
	return r.q.AdminListPaymentAttempts(ctx, params)
}

func (r *PaymentRepository) AdminGetPaymentAttempt(ctx context.Context, id uint64) (paymentsqlc.PaymentAttempt, error) {
	return r.q.AdminGetPaymentAttempt(ctx, id)
}

func (r *PaymentRepository) AdminListPaymentEvents(ctx context.Context, params paymentsqlc.AdminListPaymentEventsParams) ([]paymentsqlc.PaymentEvent, error) {
	return r.q.AdminListPaymentEvents(ctx, params)
}

func (r *PaymentRepository) AdminGetPaymentEvent(ctx context.Context, id uint64) (paymentsqlc.PaymentEvent, error) {
	return r.q.AdminGetPaymentEvent(ctx, id)
}

func (r *PaymentRepository) AdminUpdatePaymentEventProcessingStatus(ctx context.Context, params paymentsqlc.MarkPaymentEventProcessedParams) error {
	return r.q.MarkPaymentEventProcessed(ctx, params)
}

func (r *PaymentRepository) AdminListSubscriptions(ctx context.Context, params paymentsqlc.AdminListSubscriptionsParams) ([]paymentsqlc.PaymentSubscription, error) {
	return r.q.AdminListSubscriptions(ctx, params)
}

func (r *PaymentRepository) AdminGetSubscription(ctx context.Context, params paymentsqlc.AdminGetSubscriptionParams) (paymentsqlc.PaymentSubscription, error) {
	return r.q.AdminGetSubscription(ctx, params)
}

func (r *PaymentRepository) AdminGetSubscriptionByProviderID(ctx context.Context, params paymentsqlc.GetPaymentSubscriptionByProviderIDParams) (paymentsqlc.PaymentSubscription, error) {
	return r.q.GetPaymentSubscriptionByProviderID(ctx, params)
}

func (r *PaymentRepository) AdminUpsertSubscription(ctx context.Context, params paymentsqlc.UpsertPaymentSubscriptionParams) (uint64, error) {
	id, err := r.q.UpsertPaymentSubscription(ctx, params)
	return uint64(id), err
}

func (r *PaymentRepository) AdminUpdateSubscriptionStatus(ctx context.Context, params paymentsqlc.UpdatePaymentSubscriptionStatusParams) (int64, error) {
	return r.q.UpdatePaymentSubscriptionStatus(ctx, params)
}

func (r *PaymentRepository) AdminListFulfillments(ctx context.Context, params paymentsqlc.AdminListFulfillmentsParams) ([]paymentsqlc.PaymentFulfillment, error) {
	return r.q.AdminListFulfillments(ctx, params)
}

func (r *PaymentRepository) AdminGetFulfillment(ctx context.Context, id uint64) (paymentsqlc.PaymentFulfillment, error) {
	return r.q.AdminGetFulfillment(ctx, id)
}

func (r *PaymentRepository) AdminUpdateFulfillmentStatus(ctx context.Context, params paymentsqlc.AdminUpdateFulfillmentStatusParams) (int64, error) {
	return r.q.AdminUpdateFulfillmentStatus(ctx, params)
}

func (r *PaymentRepository) AdminListFulfillmentItems(ctx context.Context, params paymentsqlc.AdminListFulfillmentItemsParams) ([]paymentsqlc.PaymentFulfillmentItem, error) {
	return r.q.AdminListFulfillmentItems(ctx, params)
}

func (r *PaymentRepository) AdminListRefunds(ctx context.Context, params paymentsqlc.AdminListRefundsParams) ([]paymentsqlc.PaymentRefund, error) {
	return r.q.AdminListRefunds(ctx, params)
}

func (r *PaymentRepository) AdminGetRefund(ctx context.Context, id uint64) (paymentsqlc.PaymentRefund, error) {
	return r.q.AdminGetRefund(ctx, id)
}

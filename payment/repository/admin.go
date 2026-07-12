package repository

import (
	"context"
	"strings"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

func (r *PaymentRepository) AdminGetProvider(ctx context.Context, code string) (paymentsqlc.PaymentProvider, error) {
	return r.q.AdminGetProvider(ctx, code)
}

func (r *PaymentRepository) AdminUpsertProvider(
	ctx context.Context,
	params paymentsqlc.AdminUpsertProviderParams,
) error {
	if strings.TrimSpace(params.Code) == "" || strings.TrimSpace(params.Title) == "" ||
		!validProviderKind(params.ProviderKind) {
		return ErrInvalidProvider
	}
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
	return r.UpsertAsset(ctx, AssetUpsertParams{
		Code:            params.Code,
		Title:           params.Title,
		AssetKind:       params.AssetKind,
		Scale:           uint16(params.Scale),
		Chain:           sqlwrap.NullStringPtr(params.Chain),
		Network:         sqlwrap.NullStringPtr(params.Network),
		ContractAddress: sqlwrap.NullStringPtr(params.ContractAddress),
		IsActive:        params.IsActive,
	})
}

func (r *PaymentRepository) AdminDeleteAsset(ctx context.Context, code string) (int64, error) {
	var rows int64
	err := r.inTransaction(ctx, func(tx *PaymentRepository) error {
		if _, err := tx.q.DeleteAssetRatesForAsset(ctx, paymentsqlc.DeleteAssetRatesForAssetParams{
			AssetCode:          code,
			ReferenceAssetCode: code,
		}); err != nil {
			return err
		}
		deleted, err := tx.q.DeleteAsset(ctx, code)
		if err != nil {
			return err
		}
		rows = deleted
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rows, r.invalidateAllCache()
}

func (r *PaymentRepository) AdminListProviderAssets(
	ctx context.Context,
	params paymentsqlc.AdminListProviderAssetsParams,
) ([]paymentsqlc.PaymentProviderAsset, error) {
	return r.q.AdminListProviderAssets(ctx, params)
}

func (r *PaymentRepository) AdminUpsertProviderAsset(
	ctx context.Context,
	params paymentsqlc.UpsertProviderAssetParams,
) error {
	return r.UpsertProviderAsset(ctx, ProviderAssetUpsertParams{
		ProviderCode:    params.ProviderCode,
		AssetCode:       params.AssetCode,
		MinAmountMinor:  nullInt64Ptr(params.MinAmountMinor),
		MaxAmountMinor:  nullInt64Ptr(params.MaxAmountMinor),
		MerchantAccount: sqlwrap.NullStringPtr(params.MerchantAccount),
		IsActive:        params.IsActive,
	})
}

func validProviderKind(value paymentsqlc.PaymentProviderProviderKind) bool {
	switch value {
	case paymentsqlc.PaymentProviderProviderKindPlatformInternal,
		paymentsqlc.PaymentProviderProviderKindFiatGateway,
		paymentsqlc.PaymentProviderProviderKindCryptoChain:
		return true
	default:
		return false
	}
}

func (r *PaymentRepository) AdminDeleteProviderAsset(
	ctx context.Context,
	params paymentsqlc.DeleteProviderAssetParams,
) (int64, error) {
	rows, err := r.q.DeleteProviderAsset(ctx, params)
	if err != nil {
		return 0, err
	}
	return rows, r.invalidateAllCache()
}

func (r *PaymentRepository) AdminListProductGroups(
	ctx context.Context,
	params paymentsqlc.AdminListProductGroupsParams,
) ([]paymentsqlc.PaymentProductGroup, error) {
	return r.q.AdminListProductGroups(ctx, params)
}

func (r *PaymentRepository) AdminGetProductGroup(
	ctx context.Context,
	params paymentsqlc.AdminGetProductGroupParams,
) (paymentsqlc.PaymentProductGroup, error) {
	return r.q.AdminGetProductGroup(ctx, params)
}

func (r *PaymentRepository) AdminListLocalizations(
	ctx context.Context,
	params paymentsqlc.AdminListLocalizationsParams,
) ([]paymentsqlc.PaymentLocalization, error) {
	return r.q.AdminListLocalizations(ctx, params)
}

func (r *PaymentRepository) AdminGetLocalization(
	ctx context.Context,
	params paymentsqlc.AdminGetLocalizationParams,
) (paymentsqlc.PaymentLocalization, error) {
	return r.q.AdminGetLocalization(ctx, params)
}

func (r *PaymentRepository) AdminListProducts(
	ctx context.Context,
	params paymentsqlc.AdminListProductsParams,
) ([]paymentsqlc.PaymentProduct, error) {
	return r.q.AdminListProducts(ctx, params)
}

func (r *PaymentRepository) AdminGetProduct(
	ctx context.Context,
	params paymentsqlc.AdminGetProductParams,
) (paymentsqlc.PaymentProduct, error) {
	return r.q.AdminGetProduct(ctx, params)
}

func (r *PaymentRepository) AdminListProductItems(
	ctx context.Context,
	params paymentsqlc.AdminListProductItemsParams,
) ([]paymentsqlc.PaymentProductItem, error) {
	return r.q.AdminListProductItems(ctx, params)
}

func (r *PaymentRepository) AdminListPrices(
	ctx context.Context,
	params paymentsqlc.AdminListPricesParams,
) ([]paymentsqlc.PaymentPrice, error) {
	return r.q.AdminListPrices(ctx, params)
}

func (r *PaymentRepository) AdminGetPrice(
	ctx context.Context,
	params paymentsqlc.AdminGetPriceParams,
) (paymentsqlc.PaymentPrice, error) {
	return r.q.AdminGetPrice(ctx, params)
}

func (r *PaymentRepository) AdminGetAssetRate(
	ctx context.Context,
	params paymentsqlc.AdminGetAssetRateParams,
) (paymentsqlc.PaymentAssetRate, error) {
	return r.q.AdminGetAssetRate(ctx, params)
}

func (r *PaymentRepository) AdminListAssetRates(
	ctx context.Context,
	params paymentsqlc.AdminListAssetRatesParams,
) ([]paymentsqlc.PaymentAssetRate, error) {
	return r.q.AdminListAssetRates(ctx, params)
}

func (r *PaymentRepository) AdminListProductLimitCounters(
	ctx context.Context,
	params paymentsqlc.AdminListProductLimitCountersParams,
) ([]paymentsqlc.PaymentProductLimitCounter, error) {
	return r.q.AdminListProductLimitCounters(ctx, params)
}

func (r *PaymentRepository) AdminDeleteProductLimitCounter(
	ctx context.Context,
	params paymentsqlc.AdminDeleteProductLimitCounterParams,
) (int64, error) {
	return r.q.AdminDeleteProductLimitCounter(ctx, params)
}

func (r *PaymentRepository) AdminListPurchaseKeys(
	ctx context.Context,
	params paymentsqlc.AdminListPurchaseKeysParams,
) ([]paymentsqlc.PaymentPurchaseKey, error) {
	return r.q.AdminListPurchaseKeys(ctx, params)
}

func (r *PaymentRepository) AdminGetPurchaseKey(
	ctx context.Context,
	params paymentsqlc.AdminGetPurchaseKeyParams,
) (paymentsqlc.PaymentPurchaseKey, error) {
	return r.q.AdminGetPurchaseKey(ctx, params)
}

func (r *PaymentRepository) AdminUpdatePurchaseKeyStatus(
	ctx context.Context,
	params paymentsqlc.AdminUpdatePurchaseKeyStatusParams,
) (int64, error) {
	return r.q.AdminUpdatePurchaseKeyStatus(ctx, params)
}

func (r *PaymentRepository) AdminListOrders(
	ctx context.Context,
	params paymentsqlc.AdminListOrdersParams,
) ([]paymentsqlc.PaymentOrder, error) {
	return r.q.AdminListOrders(ctx, params)
}

func (r *PaymentRepository) AdminGetOrder(
	ctx context.Context,
	workspaceID string,
	id uint64,
) (paymentsqlc.PaymentOrder, error) {
	return r.q.AdminGetOrderForWorkspace(ctx, paymentsqlc.AdminGetOrderForWorkspaceParams{
		WorkspaceID: workspaceID,
		ID:          int64(id),
	})
}

func (r *PaymentRepository) AdminGetOrderByPublicID(
	ctx context.Context,
	workspaceID string,
	publicID string,
) (paymentsqlc.PaymentOrder, error) {
	return r.q.AdminGetOrderByPublicIDForWorkspace(ctx, paymentsqlc.AdminGetOrderByPublicIDForWorkspaceParams{
		WorkspaceID: workspaceID,
		PublicID:    publicID,
	})
}

func (r *PaymentRepository) AdminListPaymentAttempts(
	ctx context.Context,
	params paymentsqlc.AdminListPaymentAttemptsParams,
) ([]paymentsqlc.PaymentAttempt, error) {
	return r.q.AdminListPaymentAttempts(ctx, params)
}

func (r *PaymentRepository) AdminGetPaymentAttempt(
	ctx context.Context,
	workspaceID string,
	id uint64,
) (paymentsqlc.PaymentAttempt, error) {
	return r.q.AdminGetPaymentAttemptForWorkspace(ctx, paymentsqlc.AdminGetPaymentAttemptForWorkspaceParams{
		WorkspaceID: workspaceID,
		ID:          int64(id),
	})
}

func (r *PaymentRepository) AdminUpdatePaymentAttemptStatus(
	ctx context.Context,
	workspaceID string,
	id uint64,
	status paymentsqlc.PaymentAttemptStatus,
) (int64, error) {
	return r.q.AdminUpdatePaymentAttemptStatusForWorkspace(
		ctx,
		paymentsqlc.AdminUpdatePaymentAttemptStatusForWorkspaceParams{
			Status:      status,
			WorkspaceID: workspaceID,
			ID:          int64(id),
		},
	)
}

func (r *PaymentRepository) AdminListPaymentEvents(
	ctx context.Context,
	params paymentsqlc.AdminListPaymentEventsParams,
) ([]paymentsqlc.PaymentEvent, error) {
	return r.q.AdminListPaymentEvents(ctx, params)
}

func (r *PaymentRepository) AdminGetPaymentEvent(
	ctx context.Context,
	workspaceID string,
	id uint64,
) (paymentsqlc.PaymentEvent, error) {
	return r.q.AdminGetPaymentEventForWorkspace(ctx, paymentsqlc.AdminGetPaymentEventForWorkspaceParams{
		WorkspaceID: workspaceID,
		ID:          int64(id),
	})
}

func (r *PaymentRepository) AdminUpdatePaymentEventProcessingStatus(
	ctx context.Context,
	params paymentsqlc.AdminUpdatePaymentEventStatusForWorkspaceParams,
) (int64, error) {
	return r.q.AdminUpdatePaymentEventStatusForWorkspace(ctx, params)
}

func (r *PaymentRepository) AdminListSubscriptions(
	ctx context.Context,
	params paymentsqlc.AdminListSubscriptionsParams,
) ([]paymentsqlc.PaymentSubscription, error) {
	return r.q.AdminListSubscriptions(ctx, params)
}

func (r *PaymentRepository) AdminGetSubscription(
	ctx context.Context,
	params paymentsqlc.AdminGetSubscriptionParams,
) (paymentsqlc.PaymentSubscription, error) {
	return r.q.AdminGetSubscription(ctx, params)
}

func (r *PaymentRepository) AdminGetSubscriptionByProviderID(
	ctx context.Context,
	params paymentsqlc.AdminGetSubscriptionByProviderIDForWorkspaceParams,
) (paymentsqlc.PaymentSubscription, error) {
	return r.q.AdminGetSubscriptionByProviderIDForWorkspace(ctx, params)
}

func (r *PaymentRepository) AdminUpsertSubscription(
	ctx context.Context,
	params paymentsqlc.UpsertPaymentSubscriptionParams,
) (uint64, error) {
	id, err := r.q.UpsertPaymentSubscription(ctx, params)
	return uint64(id), err
}

func (r *PaymentRepository) AdminUpdateSubscriptionStatus(
	ctx context.Context,
	params paymentsqlc.UpdatePaymentSubscriptionStatusParams,
) (int64, error) {
	return r.q.UpdatePaymentSubscriptionStatus(ctx, params)
}

func (r *PaymentRepository) AdminListFulfillments(
	ctx context.Context,
	params paymentsqlc.AdminListFulfillmentsParams,
) ([]paymentsqlc.PaymentFulfillment, error) {
	return r.q.AdminListFulfillments(ctx, params)
}

func (r *PaymentRepository) AdminGetFulfillment(
	ctx context.Context,
	workspaceID string,
	id uint64,
) (paymentsqlc.PaymentFulfillment, error) {
	return r.q.AdminGetFulfillmentForWorkspace(ctx, paymentsqlc.AdminGetFulfillmentForWorkspaceParams{
		WorkspaceID: workspaceID,
		ID:          int64(id),
	})
}

func (r *PaymentRepository) AdminUpdateFulfillmentStatus(
	ctx context.Context,
	params paymentsqlc.AdminUpdateFulfillmentStatusForWorkspaceParams,
) (int64, error) {
	return r.q.AdminUpdateFulfillmentStatusForWorkspace(ctx, params)
}

func (r *PaymentRepository) AdminListFulfillmentItems(
	ctx context.Context,
	params paymentsqlc.AdminListFulfillmentItemsParams,
) ([]paymentsqlc.PaymentFulfillmentItem, error) {
	return r.q.AdminListFulfillmentItems(ctx, params)
}

func (r *PaymentRepository) AdminListRefunds(
	ctx context.Context,
	params paymentsqlc.AdminListRefundsParams,
) ([]paymentsqlc.PaymentRefund, error) {
	return r.q.AdminListRefunds(ctx, params)
}

func (r *PaymentRepository) AdminGetRefund(
	ctx context.Context,
	workspaceID string,
	id uint64,
) (paymentsqlc.PaymentRefund, error) {
	return r.q.AdminGetRefundForWorkspace(ctx, paymentsqlc.AdminGetRefundForWorkspaceParams{
		WorkspaceID: workspaceID,
		ID:          int64(id),
	})
}

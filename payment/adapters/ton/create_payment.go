package ton

import (
	"context"
	"fmt"
	"strings"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/elum-utils/services/payment/repository"
)

func (a *TON) CreatePayment(ctx context.Context, params CreatePaymentParams) (*CreatePaymentResponse, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx
	assetCode := normalizeAsset(params.AssetCode)
	network, err := validateNetwork(params.Network)
	if err != nil {
		return nil, err
	}
	walletAddress := strings.TrimSpace(params.WalletAddress)

	asset, err := a.repository.GetAsset(ctx, assetCode)
	if err != nil {
		return nil, err
	}
	if asset.Chain.Valid && asset.Chain.String != "" && !strings.EqualFold(asset.Chain.String, "ton") {
		return nil, ErrAssetChainMismatch
	}
	if asset.Network.Valid && asset.Network.String != "" && normalizeNetwork(asset.Network.String) != network {
		return nil, ErrAssetNetworkMismatch
	}
	if walletAddress == "" {
		return nil, ErrWalletAddressRequired
	}

	order, err := a.repository.CreateOrder(ctx, repository.OrderCreateParams{
		WorkspaceID:    params.WorkspaceID,
		AppID:          params.AppID,
		PlatformID:     params.PlatformID,
		PlatformUserID: params.PlatformUserID,
		InternalUserID: params.InternalUserID,
		ProductID:      params.ProductID,
		Quantity:       params.Quantity,
		AssetCode:      assetCode,
		Locale:         normalizeLocale(params.Locale),
		ReservedUntil:  params.ReservedUntil,
		ExpiresAt:      params.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	comment := order.PublicID
	attempt, err := a.repository.CreateAttempt(ctx, repository.AttemptCreateParams{
		OrderID:           order.ID,
		ProviderCode:      ProviderCode,
		KnownAssetCode:    utils.Ref(order.AssetCode),
		KnownAmountMinor:  utils.Ref(order.PayableAmountMinor),
		ProviderPaymentID: utils.Ref(comment),
		IdempotencyKey:    utils.Ref(fmt.Sprintf("%s:%s", ProviderCode, comment)),
	})
	if err != nil {
		return nil, err
	}

	return &CreatePaymentResponse{
		OrderID:        order.ID,
		OrderPublicID:  order.PublicID,
		AttemptID:      attempt.ID,
		WalletAddress:  walletAddress,
		Network:        network,
		AssetCode:      attempt.AssetCode,
		AmountMinor:    attempt.AmountMinor,
		Comment:        comment,
		Decimals:       asset.Scale,
		ProviderStatus: attempt.Status,
	}, nil
}

func normalizeAsset(assetCode string) string {
	assetCode = strings.ToUpper(strings.TrimSpace(assetCode))
	if assetCode == "" {
		return AssetTON
	}
	return assetCode
}

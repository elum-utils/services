package ton

import (
	"context"
	"errors"
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
		return nil, fmt.Errorf("ton: asset %s belongs to chain %s", assetCode, asset.Chain.String)
	}
	if asset.Network.Valid && asset.Network.String != "" && normalizeNetwork(asset.Network.String) != network {
		return nil, fmt.Errorf("ton: asset %s belongs to network %s", assetCode, asset.Network.String)
	}
	if walletAddress == "" {
		return nil, errors.New("ton: wallet address is required")
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

package product

import (
	"context"

	services "github.com/elum-utils/services"
	"github.com/elum-utils/services/payment/repository"
)

type GetParams struct {
	Identity  services.Identity
	ProductID string
	AssetCode string
	Locale    string
}

func (a *Product) Get(ctx context.Context, params GetParams) (*ProductModel, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	product, err := a.repository.GetProduct(ctx, repository.ProductGetParams{
		AppID:          params.Identity.AppID,
		WorkspaceID:    params.Identity.WorkspaceID,
		PlatformID:     params.Identity.PlatformID,
		Platform:       params.Identity.Platform,
		PlatformUserID: params.Identity.PlatformUserID,
		IsPremium:      params.Identity.IsPremium,
		Sex:            params.Identity.Sex,
		Country:        params.Identity.Country,
		ProductID:      params.ProductID,
		AssetCode:      params.AssetCode,
		Locale:         params.Locale,
	})
	if err != nil {
		return nil, err
	}

	return mapProduct(product), nil
}

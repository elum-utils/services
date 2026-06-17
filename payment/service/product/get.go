package product

import (
	"context"

	"github.com/elum-utils/services/payment/repository"
)

type GetParams struct {
	WorkspaceID    string
	AppID          int64
	PlatformID     int64
	Platform       string
	PlatformUserID string
	IsPremium      bool
	Sex            string
	Country        string
	ProductID      string
	AssetCode      string
	Locale         string
}

func (a *Product) Get(ctx context.Context, params GetParams) (*ProductModel, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	product, err := a.repository.GetProduct(ctx, repository.ProductGetParams{
		AppID:          params.AppID,
		WorkspaceID:    params.WorkspaceID,
		PlatformID:     params.PlatformID,
		Platform:       params.Platform,
		PlatformUserID: params.PlatformUserID,
		IsPremium:      params.IsPremium,
		Sex:            params.Sex,
		Country:        params.Country,
		ProductID:      params.ProductID,
		AssetCode:      params.AssetCode,
		Locale:         params.Locale,
	})
	if err != nil {
		return nil, err
	}

	return mapProduct(product), nil
}

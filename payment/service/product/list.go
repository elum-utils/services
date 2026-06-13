package product

import (
	"context"

	"github.com/elum-utils/services/payment/repository"
)

func (a *Product) List(ctx context.Context, params ListParams) ([]ProductModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()

	products, err := a.repository.ListProducts(mergedCtx, repository.ProductListParams{
		WorkspaceID:    params.WorkspaceID,
		AppID:          params.AppID,
		PlatformID:     params.PlatformID,
		PlatformUserID: params.PlatformUserID,
		AssetCode:      params.AssetCode,
		Locale:         params.Locale,
	})
	if err != nil {
		return nil, err
	}

	result := make([]ProductModel, 0, len(products))
	for _, item := range products {
		result = append(result, *mapProduct(item))
	}
	return result, nil
}

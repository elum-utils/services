package asset

import "context"

func (a *Asset) GetUSDTPrice(
	ctx context.Context,
	assetCode string,
) (*USDTPriceModel, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()

	row, err := a.repository.GetAssetUSDTPrice(mergedCtx, assetCode)
	if err != nil {
		return nil, err
	}
	return &USDTPriceModel{
		AssetCode:          row.AssetCode,
		AssetTitle:         row.AssetTitle,
		Scale:              row.Scale,
		ReferenceAssetCode: row.ReferenceAssetCode,
		USDTPerAssetMinor:  row.ReferencePerAssetMinor,
		Source:             row.Source,
		ObservedAt:         row.ObservedAt,
		UpdatedAt:          row.UpdatedAt,
	}, nil
}

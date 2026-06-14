package admin

import (
	"context"

	"github.com/elum-utils/services/payment/repository"
)

func (a *Admin) UpdateAssetRate(
	ctx context.Context,
	params UpdateAssetRateParams,
) (UpdateAssetRateResult, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()

	result, err := a.repository.UpdateAssetRate(mergedCtx, repository.AssetRateUpdateParams{
		AssetCode:              params.AssetCode,
		ReferenceAssetCode:     params.ReferenceAssetCode,
		ReferencePerAssetMinor: params.ReferencePerAssetMinor,
		Source:                 params.Source,
		ObservedAt:             params.ObservedAt,
	})
	if err != nil {
		return UpdateAssetRateResult{}, err
	}
	return UpdateAssetRateResult{
		UpdatedPrices:      result.UpdatedPrices,
		AffectedProducts:   result.AffectedProducts,
		AffectedWorkspaces: result.AffectedWorkspaces,
	}, nil
}

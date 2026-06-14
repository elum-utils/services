package admin

import (
	"context"

	"github.com/elum-utils/services/payment/repository"
)

func (a *Admin) ConfigureAssetRateAutoUpdate(
	ctx context.Context,
	params ConfigureAssetRateAutoUpdateParams,
) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()

	return a.repository.ConfigureAssetRateAutoUpdate(mergedCtx, repository.AssetRateAutoUpdateParams{
		AssetCode:          params.AssetCode,
		ReferenceAssetCode: params.ReferenceAssetCode,
		Enabled:            params.Enabled,
		Source:             params.Source,
		SourceChainID:      params.SourceChainID,
		SourceTokenAddress: params.SourceTokenAddress,
	})
}

package admin

import (
	"context"

	"github.com/elum-utils/services/cpa/repository"
)

type AddCodesParams struct {
	WorkspaceID string
	CPAID       string
	Codes       []string
}

func (a *Admin) AddCodes(ctx context.Context, params AddCodesParams) (int, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	offer, err := a.repository.GetOffer(mergedCtx, params.WorkspaceID, params.CPAID)
	if err != nil {
		return 0, err
	}
	if offer.CodeMode != repository.CodeModePersonal ||
		offer.CodeSource == nil ||
		*offer.CodeSource != repository.CodeSourcePool {
		return 0, ErrCodeUploadModeUnsupported
	}
	return a.repository.AddCodes(mergedCtx, params.WorkspaceID, params.CPAID, params.Codes)
}

func (a *Admin) DeleteAvailableCodes(ctx context.Context, workspaceID, cpaID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteAvailableCodes(mergedCtx, workspaceID, cpaID)
}

func (a *Admin) DeleteIssuedCodes(ctx context.Context, workspaceID, cpaID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteIssuedCodes(mergedCtx, workspaceID, cpaID)
}

func (a *Admin) DeleteCompletedCodes(ctx context.Context, workspaceID, cpaID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteCompletedCodes(mergedCtx, workspaceID, cpaID)
}

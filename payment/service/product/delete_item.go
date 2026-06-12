package product

import "context"

func (a *Product) DeleteItem(ctx context.Context, workspaceID string, id string) (int64, error) {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	return a.repository.DeleteItem(ctx, workspaceID, id)
}

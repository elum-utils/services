package product

import (
	"context"

	"github.com/elum-utils/services/payment/repository"
)

type UpsertItemParams struct {
	WorkspaceID    string
	ID             string
	ItemType       *string
	TitleKey       string
	DescriptionKey *string
	Rarity         string
	Position       int32
}

func (a *Product) UpsertItem(ctx context.Context, params UpsertItemParams) error {
	mergedCtx, paymentRequestCancel := a.withContext(ctx)
	defer paymentRequestCancel()
	ctx = mergedCtx

	return a.repository.UpsertItem(ctx, repository.ItemUpsertParams{
		ID:             params.ID,
		WorkspaceID:    params.WorkspaceID,
		ItemType:       params.ItemType,
		TitleKey:       params.TitleKey,
		DescriptionKey: params.DescriptionKey,
		Rarity:         params.Rarity,
		Position:       params.Position,
	})
}

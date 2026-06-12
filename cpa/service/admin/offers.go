package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/elum-utils/services/cpa/repository"
)

type UpsertOfferParams struct {
	WorkspaceID       string
	ID                string
	Payload           json.RawMessage
	CodeMode          string
	CodeSource        *string
	SharedCode        *string
	GeneratedLength   *int16
	GeneratedAlphabet *string
	IsActive          bool
	StartAt           *time.Time
	EndAt             *time.Time
}

func (a *Admin) UpsertOffer(ctx context.Context, params UpsertOfferParams) error {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	if err := validateOffer(params); err != nil {
		return err
	}
	normalizeOffer(&params)
	return a.repository.UpsertOffer(mergedCtx, repository.UpsertOfferParams(params))
}

func (a *Admin) GetOffer(ctx context.Context, workspaceID, cpaID string) (OfferModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	offer, err := a.repository.GetOffer(mergedCtx, workspaceID, cpaID)
	if err != nil {
		return OfferModel{}, err
	}
	localizations, err := a.repository.ListLocalizations(mergedCtx, workspaceID, cpaID)
	if err != nil {
		return OfferModel{}, err
	}
	rewards, err := a.repository.ListRewards(mergedCtx, workspaceID, cpaID)
	if err != nil {
		return OfferModel{}, err
	}
	return mapOffer(offer, localizations, rewards), nil
}

func (a *Admin) ListOffers(ctx context.Context, workspaceID string, page Page) ([]OfferModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	limit, offset := normalizePage(page)
	bundles, err := a.repository.ListOfferBundles(mergedCtx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]OfferModel, 0, len(bundles))
	for _, bundle := range bundles {
		result = append(result, mapOffer(bundle.Offer, bundle.Localizations, bundle.Rewards))
	}
	return result, nil
}

func (a *Admin) DeleteOffer(ctx context.Context, workspaceID, cpaID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteOffer(mergedCtx, workspaceID, cpaID)
}

func validateOffer(params UpsertOfferParams) error {
	if params.WorkspaceID == "" || params.ID == "" {
		return errors.New("cpa admin: workspace and offer id are required")
	}
	if len(params.Payload) == 0 || !json.Valid(params.Payload) {
		return errors.New("cpa admin: payload must be valid JSON")
	}
	if params.StartAt != nil && params.EndAt != nil && !params.StartAt.Before(*params.EndAt) {
		return errors.New("cpa admin: start_at must be before end_at")
	}
	switch params.CodeMode {
	case repository.CodeModeShared:
		if params.SharedCode == nil || strings.TrimSpace(*params.SharedCode) == "" {
			return errors.New("cpa admin: shared_code is required")
		}
	case repository.CodeModePersonal:
		if params.CodeSource == nil {
			return errors.New("cpa admin: personal code source is required")
		}
		switch *params.CodeSource {
		case repository.CodeSourcePool:
		case repository.CodeSourceGenerated:
			if params.GeneratedLength == nil || *params.GeneratedLength <= 0 {
				return errors.New("cpa admin: generated code length must be positive")
			}
			if params.GeneratedAlphabet == nil || len([]rune(*params.GeneratedAlphabet)) < 2 {
				return errors.New("cpa admin: generated alphabet needs at least two symbols")
			}
		default:
			return errors.New("cpa admin: unsupported personal code source")
		}
	default:
		return errors.New("cpa admin: unsupported code mode")
	}
	return nil
}

func normalizeOffer(params *UpsertOfferParams) {
	if params.CodeMode == repository.CodeModeShared {
		params.CodeSource = nil
		params.GeneratedLength = nil
		params.GeneratedAlphabet = nil
		return
	}
	params.SharedCode = nil
	if params.CodeSource != nil && *params.CodeSource == repository.CodeSourcePool {
		params.GeneratedLength = nil
		params.GeneratedAlphabet = nil
	}
}

package admin

import (
	"context"
	"encoding/json"
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
		return ErrOfferScopeRequired
	}
	if len(params.Payload) == 0 || !json.Valid(params.Payload) {
		return ErrOfferPayloadInvalid
	}
	if params.StartAt != nil && params.EndAt != nil && !params.StartAt.Before(*params.EndAt) {
		return ErrOfferRangeInvalid
	}
	switch params.CodeMode {
	case repository.CodeModeShared:
		if params.SharedCode == nil || strings.TrimSpace(*params.SharedCode) == "" {
			return ErrOfferSharedCodeRequired
		}
	case repository.CodeModePersonal:
		if params.CodeSource == nil {
			return ErrOfferCodeSourceRequired
		}
		switch *params.CodeSource {
		case repository.CodeSourcePool:
		case repository.CodeSourceGenerated:
			if params.GeneratedLength == nil || *params.GeneratedLength <= 0 {
				return ErrGeneratedCodeLengthInvalid
			}
			if params.GeneratedAlphabet == nil || len([]rune(*params.GeneratedAlphabet)) < 2 {
				return ErrGeneratedCodeAlphabetInvalid
			}
		default:
			return ErrPersonalCodeSourceUnsupported
		}
	default:
		return ErrCodeModeUnsupported
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

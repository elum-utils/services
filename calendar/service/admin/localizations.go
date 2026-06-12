package admin

import (
	"context"
	"errors"

	"github.com/elum-utils/services/calendar/repository"
)

type SaveLocalizationParams struct {
	WorkspaceID string
	CalendarID  string
	Locale      string
	Title       string
	Description string
}

func (a *Admin) UpsertLocalization(ctx context.Context, params SaveLocalizationParams) error {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	if params.WorkspaceID == "" || params.CalendarID == "" || params.Locale == "" || params.Title == "" {
		return errors.New("calendar admin: localization scope, locale and title are required")
	}
	return a.repository.UpsertLocalization(mergedCtx, repository.Localization(params))
}

func (a *Admin) GetLocalization(ctx context.Context, workspaceID, calendarID, locale string) (LocalizationModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	value, err := a.repository.GetLocalization(mergedCtx, workspaceID, calendarID, locale)
	if err != nil {
		return LocalizationModel{}, err
	}
	return LocalizationModel{Locale: value.Locale, Title: value.Title, Description: value.Description}, nil
}

func (a *Admin) ListLocalizations(ctx context.Context, workspaceID, calendarID string) ([]LocalizationModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	values, err := a.repository.ListLocalizations(mergedCtx, workspaceID, calendarID)
	if err != nil {
		return nil, err
	}
	result := make([]LocalizationModel, 0, len(values))
	for _, value := range values {
		result = append(result, LocalizationModel{
			Locale: value.Locale, Title: value.Title, Description: value.Description,
		})
	}
	return result, nil
}

func (a *Admin) DeleteLocalization(ctx context.Context, workspaceID, calendarID, locale string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteLocalization(mergedCtx, workspaceID, calendarID, locale)
}

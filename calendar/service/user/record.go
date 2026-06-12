package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/elum-utils/services/calendar/repository"
)

type RecordParams struct {
	Identity    Identity
	CalendarRef string
	OperationID string
	Now         time.Time
}

func (u *User) Record(ctx context.Context, params RecordParams) (RecordResult, error) {
	mergedCtx, cancel := u.withContext(ctx)
	defer cancel()
	if params.Identity.WorkspaceID == "" || params.Identity.PlatformUserID == "" ||
		strings.TrimSpace(params.CalendarRef) == "" || strings.TrimSpace(params.OperationID) == "" {
		return RecordResult{}, errors.New("calendar user: identity, calendar and operation are required")
	}
	value, err := u.repository.Record(mergedCtx, repository.RecordParams{
		Identity: repositoryIdentity(params.Identity), CalendarRef: params.CalendarRef,
		OperationID: params.OperationID, Now: params.Now,
	})
	if err != nil {
		return RecordResult{}, err
	}
	return mapRecord(value), nil
}

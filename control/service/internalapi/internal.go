package internalapi

import (
	"context"
	"strings"

	"github.com/elum-utils/services/control/repository"
	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

type Internal struct {
	rootCtx    context.Context
	repository *repository.Repository
}

type MethodManifest struct {
	Key, Service, GroupKey string
}

type AccessRequest struct {
	AccountID, WorkspaceID, MethodKey string
}

type AuthorizedMethod struct {
	Key, Service, GroupKey string
}

func NewWithOptions(ctx context.Context, db *sqlwrap.Client, options repository.Options) *Internal {
	return &Internal{rootCtx: contextutil.Normalize(ctx), repository: repository.NewWithOptions(db, options)}
}

func (i *Internal) Close() error {
	if i == nil || i.repository == nil {
		return nil
	}
	return i.repository.Close()
}

func (i *Internal) withContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return contextutil.Merge(i.rootCtx, ctx)
}

func (i *Internal) RegisterManifest(ctx context.Context, values []MethodManifest) error {
	mergedCtx, cancel := i.withContext(ctx)
	defer cancel()
	for _, value := range values {
		if err := i.repository.RegisterMethod(mergedCtx, repository.Method{
			Key: strings.TrimSpace(value.Key), Service: strings.TrimSpace(value.Service), GroupKey: strings.TrimSpace(value.GroupKey),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (i *Internal) CheckAccess(ctx context.Context, value AccessRequest) (bool, error) {
	mergedCtx, cancel := i.withContext(ctx)
	defer cancel()
	return i.repository.CheckAccess(mergedCtx, strings.TrimSpace(value.AccountID), strings.TrimSpace(value.WorkspaceID), strings.TrimSpace(value.MethodKey))
}

func (i *Internal) GetAuthorizedMethods(ctx context.Context, accountID, workspaceID string) ([]AuthorizedMethod, error) {
	mergedCtx, cancel := i.withContext(ctx)
	defer cancel()
	methods, err := i.repository.ListAuthorizedMethods(mergedCtx, strings.TrimSpace(accountID), strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, err
	}
	result := make([]AuthorizedMethod, 0, len(methods))
	for _, method := range methods {
		result = append(result, AuthorizedMethod{Key: method.Key, Service: method.Service, GroupKey: method.GroupKey})
	}
	return result, nil
}

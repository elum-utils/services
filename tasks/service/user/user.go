package user

import (
	"context"

	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/tasks/repository"
	taskruntime "github.com/elum-utils/services/tasks/runtime"
)

type User struct {
	rootCtx    context.Context
	repository *repository.Repository
	runtime    *taskruntime.Manager
	providers  map[string]PartnerProvider
}

type Options struct {
	RepositoryOptions repository.Options
	PartnerProviders  map[string]PartnerProvider
	Runtime           *taskruntime.Manager
}

func New(ctx context.Context, db *sqlwrap.Client) *User {
	return &User{rootCtx: contextutil.Normalize(ctx), repository: repository.New(db)}
}

func NewWithOptions(ctx context.Context, db *sqlwrap.Client, options repository.Options) *User {
	return &User{rootCtx: contextutil.Normalize(ctx), repository: repository.NewWithOptions(db, options)}
}

func NewWithServiceOptions(ctx context.Context, db *sqlwrap.Client, options Options) *User {
	return &User{
		rootCtx:    contextutil.Normalize(ctx),
		repository: repository.NewWithOptions(db, options.RepositoryOptions),
		runtime:    options.Runtime,
		providers:  defaultPartnerProviders(options.PartnerProviders),
	}
}

func (u *User) Close() error {
	if u == nil || u.repository == nil {
		return nil
	}
	return u.repository.Close()
}

func (u *User) withContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if u == nil {
		return contextutil.Merge(context.Background(), ctx)
	}
	return contextutil.Merge(u.rootCtx, ctx)
}

func clonePartnerProviders(values map[string]PartnerProvider) map[string]PartnerProvider {
	result := make(map[string]PartnerProvider, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func (u *User) partnerProvider(provider string) PartnerProvider {
	if u == nil {
		return nil
	}
	if value := u.providers[provider]; value != nil {
		return value
	}
	if u.runtime != nil {
		return LuaProvider{Runtime: u.runtime, Provider: provider}
	}
	return nil
}

func defaultPartnerProviders(overrides map[string]PartnerProvider) map[string]PartnerProvider {
	result := make(map[string]PartnerProvider, len(overrides))
	for key, value := range overrides {
		if value == nil {
			delete(result, key)
			continue
		}
		result[key] = value
	}
	return result
}

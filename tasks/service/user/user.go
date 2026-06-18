package user

import (
	"context"

	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/tasks/repository"
)

type User struct {
	rootCtx    context.Context
	repository *repository.Repository
	providers  map[string]PartnerProvider
}

type Options struct {
	RepositoryOptions repository.Options
	PartnerProviders  map[string]PartnerProvider
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

func defaultPartnerProviders(overrides map[string]PartnerProvider) map[string]PartnerProvider {
	result := map[string]PartnerProvider{
		"flyer":   FlyerProvider{},
		"subgram": SubGramProvider{},
		"tgrass":  TgrassProvider{},
	}
	for key, value := range overrides {
		if value == nil {
			delete(result, key)
			continue
		}
		result[key] = value
	}
	return result
}

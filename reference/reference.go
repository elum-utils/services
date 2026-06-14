package reference

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/elum-utils/services/internal/utils/contextutil"
	"github.com/elum-utils/services/internal/utils/mysqlutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/reference/repository"
	"github.com/elum-utils/services/reference/service/admin"
	"github.com/elum-utils/services/reference/service/user"
)

type Reference struct {
	Admin *admin.Admin
	User  *user.User

	client     *sqlwrap.Client
	ownsClient bool
	rootCtx    context.Context
	rootCancel context.CancelFunc

	lifecycleMu sync.Mutex
	params      DatabaseParams
	running     bool
}

func New(params DatabaseParams) *Reference {
	return &Reference{params: params}
}

func NewWithDatabase(ctx context.Context, db *sql.DB, options Options) (*Reference, error) {
	client, err := sqlwrap.New(db, toSQLWrapOptions(options))
	if err != nil {
		return nil, err
	}
	return newReference(ctx, client, false, options), nil
}

func (r *Reference) Run(ctx context.Context) error {
	if r == nil {
		return errors.New("reference: nil service")
	}
	r.lifecycleMu.Lock()
	if r.running {
		r.lifecycleMu.Unlock()
		return errors.New("reference: service is already running")
	}
	r.running = true
	params := r.params
	r.lifecycleMu.Unlock()

	running, err := open(ctx, params)
	if err != nil {
		r.lifecycleMu.Lock()
		r.running = false
		r.lifecycleMu.Unlock()
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return nil
		}
		return err
	}
	r.adopt(running)
	defer r.Close()
	<-r.rootCtx.Done()
	return nil
}

func open(ctx context.Context, params DatabaseParams) (*Reference, error) {
	if params.User == "" || params.Database == "" {
		return nil, errors.New("reference: database user and name are required")
	}
	db, err := mysqlutil.Open(ctx, mysqlutil.Config{
		User: params.User, Password: params.Password, Database: params.Database,
		Host: params.Host, Port: params.Port,
	})
	if err != nil {
		return nil, err
	}
	client, err := sqlwrap.New(db, toSQLWrapOptions(params.Options))
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	bootstrap := repository.NewWithOptions(client, repository.Options{
		QueryTimeout: params.Options.QueryTimeout,
		CacheL1Delay: params.Options.CacheL1Delay,
		CacheL2Delay: params.Options.CacheL2Delay,
	})
	if err := bootstrap.Bootstrap(contextutil.Normalize(ctx)); err != nil {
		_ = bootstrap.Close()
		_ = client.Close()
		return nil, err
	}
	if err := bootstrap.Close(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return newReference(ctx, client, true, params.Options), nil
}

func (r *Reference) adopt(running *Reference) {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.Admin, r.User = running.Admin, running.User
	r.client, r.ownsClient = running.client, running.ownsClient
	r.rootCtx, r.rootCancel = running.rootCtx, running.rootCancel
}

func newReference(ctx context.Context, db *sqlwrap.Client, ownsClient bool, options Options) *Reference {
	rootCtx, cancel := context.WithCancel(contextutil.Normalize(ctx))
	repositoryOptions := repository.Options{
		QueryTimeout: options.QueryTimeout,
		CacheL1Delay: options.CacheL1Delay,
		CacheL2Delay: options.CacheL2Delay,
	}
	return &Reference{
		Admin:  admin.NewWithRepositoryOptions(rootCtx, db, repositoryOptions),
		User:   user.NewWithRepositoryOptions(rootCtx, db, repositoryOptions),
		client: db, ownsClient: ownsClient, rootCtx: rootCtx, rootCancel: cancel,
	}
}

func (r *Reference) Close() error {
	if r == nil {
		return nil
	}
	if r.rootCancel != nil {
		r.rootCancel()
	}
	var err error
	if r.Admin != nil {
		err = errors.Join(err, r.Admin.Close())
	}
	if r.User != nil {
		err = errors.Join(err, r.User.Close())
	}
	if r.ownsClient && r.client != nil {
		err = errors.Join(err, r.client.Close())
	}
	return err
}

// IsReady reports whether the service is initialized and its lifecycle is active.
func (r *Reference) IsReady() bool {
	if r == nil {
		return false
	}
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	return r.rootCtx != nil && r.rootCtx.Err() == nil && r.Admin != nil && r.User != nil
}

package tasks

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	serviceerrors "github.com/elum-utils/services/errors"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/internal/utils/contextutil"
	"github.com/elum-utils/services/internal/utils/mysqlutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/tasks/repository"
	"github.com/elum-utils/services/tasks/service/admin"
	"github.com/elum-utils/services/tasks/service/integration"
	"github.com/elum-utils/services/tasks/service/internalapi"
	"github.com/elum-utils/services/tasks/service/user"
)

type Tasks struct {
	Admin       *admin.Admin
	Internal    *internalapi.Internal
	Integration *integration.Integration
	User        *user.User

	callbacks  *callbackutil.Store
	client     *sqlwrap.Client
	ownsClient bool
	rootCtx    context.Context
	rootCancel context.CancelFunc
	background sync.WaitGroup

	lifecycleMu    sync.Mutex
	params         DatabaseParams
	callbacksToRun []callbackRegistration
	running        bool
}

func New(params DatabaseParams) *Tasks {
	return &Tasks{params: params}
}

func NewWithDatabase(ctx context.Context, db *sql.DB, options Options) (*Tasks, error) {
	client, err := sqlwrap.New(db, toSQLWrapOptions(options))
	if err != nil {
		return nil, serviceerrors.Wrap(serviceerrors.CodeInternalError, "tasks sql client initialization failed", err)
	}
	return newTasks(ctx, client, false, options), nil
}

func (t *Tasks) Run(ctx context.Context) error {
	if t == nil {
		return ErrServiceNil
	}
	t.lifecycleMu.Lock()
	if t.running {
		t.lifecycleMu.Unlock()
		return ErrServiceRunning
	}
	t.running = true
	params := t.params
	registrations := append([]callbackRegistration(nil), t.callbacksToRun...)
	t.lifecycleMu.Unlock()

	running, err := open(ctx, params)
	if err != nil {
		t.lifecycleMu.Lock()
		t.running = false
		t.lifecycleMu.Unlock()
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return nil
		}
		return wrapLifecycleError(err)
	}
	t.adopt(running)
	defer t.Close()

	errCh := make(chan error, len(registrations))
	t.background.Add(len(registrations))
	for _, registration := range registrations {
		registration := registration
		go func() {
			defer t.background.Done()
			errCh <- t.runCallback(registration.ctx, registration.handler, registration.options...)
		}()
	}
	select {
	case <-t.rootCtx.Done():
		return nil
	case err := <-errCh:
		if errors.Is(err, context.Canceled) && t.rootCtx.Err() != nil {
			return nil
		}
		return wrapLifecycleError(err)
	}
}

func open(ctx context.Context, params DatabaseParams) (*Tasks, error) {
	if params.User == "" || params.Database == "" {
		return nil, ErrDatabaseConfigRequired
	}
	db, err := mysqlutil.Open(ctx, mysqlutil.Config{
		User: params.User, Password: params.Password, Database: params.Database,
		Host: params.Host, Port: params.Port,
	})
	if err != nil {
		return nil, serviceerrors.Wrap(serviceerrors.CodeUnavailable, "tasks database connection failed", err)
	}
	client, err := sqlwrap.New(db, toSQLWrapOptions(params.Options))
	if err != nil {
		_ = db.Close()
		return nil, serviceerrors.Wrap(serviceerrors.CodeInternalError, "tasks sql client initialization failed", err)
	}
	bootstrap := repository.NewWithOptions(client, repositoryOptions(params.Options))
	if err := bootstrap.Bootstrap(ctx); err != nil {
		_ = bootstrap.Close()
		_ = client.Close()
		return nil, serviceerrors.Wrap(serviceerrors.CodeInternalError, "tasks bootstrap failed", err)
	}
	if err := bootstrap.Close(); err != nil {
		_ = client.Close()
		return nil, serviceerrors.Wrap(serviceerrors.CodeInternalError, "tasks bootstrap shutdown failed", err)
	}
	return newTasks(ctx, client, true, params.Options), nil
}

func (t *Tasks) adopt(running *Tasks) {
	t.lifecycleMu.Lock()
	defer t.lifecycleMu.Unlock()
	t.Admin, t.Internal, t.Integration, t.User = running.Admin, running.Internal, running.Integration, running.User
	t.callbacks, t.client, t.ownsClient = running.callbacks, running.client, running.ownsClient
	t.rootCtx, t.rootCancel = running.rootCtx, running.rootCancel
}

func newTasks(ctx context.Context, db *sqlwrap.Client, ownsClient bool, options Options) *Tasks {
	rootCtx, cancel := context.WithCancel(contextutil.Normalize(ctx))
	repositoryOptions := repositoryOptions(options)
	return &Tasks{
		Admin: admin.NewWithOptions(rootCtx, db, repositoryOptions), Internal: internalapi.NewWithOptions(rootCtx, db, repositoryOptions),
		Integration: integration.NewWithOptions(rootCtx, db, integrationOptions(options, repositoryOptions)),
		User: user.NewWithServiceOptions(rootCtx, db, user.Options{
			RepositoryOptions: repositoryOptions,
			PartnerProviders:  options.PartnerProviders,
		}),
		callbacks: callbackutil.NewWithTable(db.DB(), callbackutil.TasksTable), client: db, ownsClient: ownsClient,
		rootCtx: rootCtx, rootCancel: cancel,
	}
}

func integrationOptions(options Options, repositoryOptions repository.Options) integration.Options {
	result := options.Integration
	result.RepositoryOptions = repositoryOptions
	return result
}

func repositoryOptions(options Options) repository.Options {
	return repository.Options{
		QueryTimeout: options.QueryTimeout,
		CacheL1Delay: options.CacheL1Delay,
		CacheL2Delay: options.CacheL2Delay,
	}
}

func (t *Tasks) Close() error {
	if t == nil {
		return nil
	}
	if t.rootCancel != nil {
		t.rootCancel()
	}
	t.background.Wait()
	var err error
	if t.Admin != nil {
		err = errors.Join(err, t.Admin.Close())
	}
	if t.Internal != nil {
		err = errors.Join(err, t.Internal.Close())
	}
	if t.Integration != nil {
		err = errors.Join(err, t.Integration.Close())
	}
	if t.User != nil {
		err = errors.Join(err, t.User.Close())
	}
	if t.callbacks != nil {
		err = errors.Join(err, t.callbacks.Close())
	}
	if t.ownsClient && t.client != nil {
		err = errors.Join(err, t.client.Close())
	}
	return err
}

// IsReady reports whether the service is initialized and its lifecycle is active.
func (t *Tasks) IsReady() bool {
	if t == nil {
		return false
	}
	t.lifecycleMu.Lock()
	defer t.lifecycleMu.Unlock()
	return t.rootCtx != nil && t.rootCtx.Err() == nil &&
		t.Admin != nil && t.Internal != nil && t.Integration != nil && t.User != nil
}

func (t *Tasks) bindContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if t == nil {
		return contextutil.Merge(context.Background(), ctx)
	}
	return contextutil.Merge(t.rootCtx, ctx)
}

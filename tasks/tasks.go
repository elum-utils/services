package tasks

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/internal/utils/contextutil"
	"github.com/elum-utils/services/internal/utils/mysqlutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/tasks/repository"
	"github.com/elum-utils/services/tasks/service/admin"
	"github.com/elum-utils/services/tasks/service/internalapi"
	"github.com/elum-utils/services/tasks/service/user"
)

type Tasks struct {
	Admin    *admin.Admin
	Internal *internalapi.Internal
	User     *user.User

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
		return nil, err
	}
	return newTasks(ctx, client, false, options), nil
}

func (t *Tasks) Run(ctx context.Context) error {
	if t == nil {
		return errors.New("tasks: nil service")
	}
	t.lifecycleMu.Lock()
	if t.running {
		t.lifecycleMu.Unlock()
		return errors.New("tasks: service is already running")
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
		return err
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
		return err
	}
}

func open(ctx context.Context, params DatabaseParams) (*Tasks, error) {
	if params.User == "" || params.Database == "" {
		return nil, errors.New("tasks: database user and name are required")
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
	bootstrap := repository.NewWithOptions(client, repositoryOptions(params.Options))
	if err := bootstrap.Bootstrap(ctx); err != nil {
		_ = bootstrap.Close()
		_ = client.Close()
		return nil, err
	}
	if err := bootstrap.Close(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return newTasks(ctx, client, true, params.Options), nil
}

func (t *Tasks) adopt(running *Tasks) {
	t.Admin, t.Internal, t.User = running.Admin, running.Internal, running.User
	t.callbacks, t.client, t.ownsClient = running.callbacks, running.client, running.ownsClient
	t.rootCtx, t.rootCancel = running.rootCtx, running.rootCancel
}

func newTasks(ctx context.Context, db *sqlwrap.Client, ownsClient bool, options Options) *Tasks {
	rootCtx, cancel := context.WithCancel(contextutil.Normalize(ctx))
	repositoryOptions := repositoryOptions(options)
	return &Tasks{
		Admin: admin.NewWithOptions(rootCtx, db, repositoryOptions), Internal: internalapi.NewWithOptions(rootCtx, db, repositoryOptions), User: user.NewWithOptions(rootCtx, db, repositoryOptions),
		callbacks: callbackutil.NewWithTable(db.DB(), callbackutil.TasksTable), client: db, ownsClient: ownsClient,
		rootCtx: rootCtx, rootCancel: cancel,
	}
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

func (t *Tasks) bindContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if t == nil {
		return contextutil.Merge(context.Background(), ctx)
	}
	return contextutil.Merge(t.rootCtx, ctx)
}

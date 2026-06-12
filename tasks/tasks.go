package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/tasks/repository"
	"github.com/elum-utils/services/tasks/service/admin"
	"github.com/elum-utils/services/tasks/service/internalapi"
	"github.com/elum-utils/services/tasks/service/user"
	"github.com/go-sql-driver/mysql"
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
}

func New(ctx context.Context, db *sql.DB) (*Tasks, error) {
	return NewWithOptions(ctx, db, Options{
		CacheL1Delay: defaultCacheDelay,
		CacheL2Delay: defaultCacheDelay,
	})
}

func NewWithOptions(ctx context.Context, db *sql.DB, options Options) (*Tasks, error) {
	client, err := sqlwrap.New(db, toSQLWrapOptions(options))
	if err != nil {
		return nil, err
	}
	return newTasks(ctx, client, false, options), nil
}

func Open(ctx context.Context, params DatabaseParams) (*Tasks, error) {
	if params.User == "" || params.Database == "" {
		return nil, errors.New("tasks: database user and name are required")
	}
	host := params.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := params.Port
	if port <= 0 {
		port = 3306
	}
	cfg := mysql.Config{
		User: params.User, Passwd: params.Password, Net: "tcp",
		Addr: fmt.Sprintf("%s:%d", host, port), DBName: params.Database,
		ParseTime: true, AllowNativePasswords: true,
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	client, err := sqlwrap.New(db, toSQLWrapOptions(params.Options))
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return newTasks(ctx, client, true, params.Options), nil
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

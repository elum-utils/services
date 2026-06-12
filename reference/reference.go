package reference

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/reference/repository"
	"github.com/elum-utils/services/reference/service/admin"
	"github.com/elum-utils/services/reference/service/user"
	"github.com/go-sql-driver/mysql"
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
	bootstrap := repository.NewWithOptions(client, repository.Options{QueryTimeout: params.Options.QueryTimeout})
	if err := bootstrap.Bootstrap(contextutil.Normalize(ctx)); err != nil {
		_ = bootstrap.Close()
		_ = client.Close()
		return nil, err
	}
	_ = bootstrap.Close()
	return newReference(ctx, client, true, params.Options), nil
}

func (r *Reference) adopt(running *Reference) {
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

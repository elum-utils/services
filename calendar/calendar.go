package calendar

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/elum-utils/services/calendar/repository"
	"github.com/elum-utils/services/calendar/service/admin"
	"github.com/elum-utils/services/calendar/service/user"
	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/go-sql-driver/mysql"
)

type Calendar struct {
	Admin *admin.Admin
	User  *user.User

	callbacks  *callbackutil.Store
	client     *sqlwrap.Client
	ownsClient bool
	rootCtx    context.Context
	rootCancel context.CancelFunc
	background sync.WaitGroup
}

func New(ctx context.Context, db *sql.DB) (*Calendar, error) {
	return NewWithOptions(ctx, db, Options{
		CacheL1Delay: defaultCacheDelay,
		CacheL2Delay: defaultCacheDelay,
	})
}

func NewWithOptions(ctx context.Context, db *sql.DB, options Options) (*Calendar, error) {
	client, err := sqlwrap.New(db, toSQLWrapOptions(options))
	if err != nil {
		return nil, err
	}
	return newCalendar(ctx, client, false, options), nil
}

func Open(ctx context.Context, params DatabaseParams) (*Calendar, error) {
	if params.User == "" || params.Database == "" {
		return nil, errors.New("calendar: database user and name are required")
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
	return newCalendar(ctx, client, true, params.Options), nil
}

func newCalendar(ctx context.Context, db *sqlwrap.Client, ownsClient bool, options Options) *Calendar {
	rootCtx, cancel := context.WithCancel(contextutil.Normalize(ctx))
	repositoryOptions := repository.Options{
		QueryTimeout: options.QueryTimeout,
		CacheL1Delay: options.CacheL1Delay,
		CacheL2Delay: options.CacheL2Delay,
	}
	return &Calendar{
		Admin:     admin.NewWithRepositoryOptions(rootCtx, db, repositoryOptions),
		User:      user.NewWithRepositoryOptions(rootCtx, db, repositoryOptions),
		callbacks: callbackutil.NewWithTable(db.DB(), callbackutil.CalendarTable), client: db, ownsClient: ownsClient,
		rootCtx: rootCtx, rootCancel: cancel,
	}
}

func (c *Calendar) Close() error {
	if c == nil {
		return nil
	}
	if c.rootCancel != nil {
		c.rootCancel()
	}
	c.background.Wait()
	var err error
	if c.Admin != nil {
		err = errors.Join(err, c.Admin.Close())
	}
	if c.User != nil {
		err = errors.Join(err, c.User.Close())
	}
	if c.callbacks != nil {
		err = errors.Join(err, c.callbacks.Close())
	}
	if c.ownsClient && c.client != nil {
		err = errors.Join(err, c.client.Close())
	}
	return err
}

func (c *Calendar) bindContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c == nil {
		return contextutil.Merge(context.Background(), ctx)
	}
	return contextutil.Merge(c.rootCtx, ctx)
}

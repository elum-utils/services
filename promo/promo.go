package promo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/promo/repository"
	"github.com/elum-utils/services/promo/service/admin"
	"github.com/elum-utils/services/promo/service/user"
	"github.com/go-sql-driver/mysql"
)

type Promo struct {
	Admin *admin.Admin
	User  *user.User

	callbacks  *callbackutil.Store
	client     *sqlwrap.Client
	ownsClient bool
	rootCtx    context.Context
	rootCancel context.CancelFunc
	background sync.WaitGroup
}

func New(ctx context.Context, db *sql.DB) (*Promo, error) {
	return NewWithOptions(ctx, db, Options{
		CacheL1Delay: defaultCacheDelay,
		CacheL2Delay: defaultCacheDelay,
	})
}

func NewWithOptions(ctx context.Context, db *sql.DB, options Options) (*Promo, error) {
	client, err := sqlwrap.New(db, toSQLWrapOptions(options))
	if err != nil {
		return nil, err
	}
	return newPromo(ctx, client, false, options), nil
}

func Open(ctx context.Context, params DatabaseParams) (*Promo, error) {
	if params.User == "" || params.Database == "" {
		return nil, errors.New("promo: database user and name are required")
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
	return newPromo(ctx, client, true, params.Options), nil
}

func newPromo(ctx context.Context, db *sqlwrap.Client, ownsClient bool, options Options) *Promo {
	rootCtx, cancel := context.WithCancel(contextutil.Normalize(ctx))
	repositoryOptions := repository.Options{
		QueryTimeout: options.QueryTimeout,
		CacheL1Delay: options.CacheL1Delay,
		CacheL2Delay: options.CacheL2Delay,
	}
	return &Promo{
		Admin:     admin.NewWithRepositoryOptions(rootCtx, db, repositoryOptions),
		User:      user.NewWithRepositoryOptions(rootCtx, db, repositoryOptions),
		callbacks: callbackutil.NewWithTable(db.DB(), callbackutil.PromoTable), client: db, ownsClient: ownsClient,
		rootCtx: rootCtx, rootCancel: cancel,
	}
}

func (p *Promo) Close() error {
	if p == nil {
		return nil
	}
	if p.rootCancel != nil {
		p.rootCancel()
	}
	p.background.Wait()
	var err error
	if p.Admin != nil {
		err = errors.Join(err, p.Admin.Close())
	}
	if p.User != nil {
		err = errors.Join(err, p.User.Close())
	}
	if p.callbacks != nil {
		err = errors.Join(err, p.callbacks.Close())
	}
	if p.ownsClient && p.client != nil {
		err = errors.Join(err, p.client.Close())
	}
	return err
}

func (p *Promo) bindContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if p == nil {
		return contextutil.Merge(context.Background(), ctx)
	}
	return contextutil.Merge(p.rootCtx, ctx)
}

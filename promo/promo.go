package promo

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/internal/utils/contextutil"
	"github.com/elum-utils/services/internal/utils/mysqlutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/promo/repository"
	"github.com/elum-utils/services/promo/service/admin"
	"github.com/elum-utils/services/promo/service/user"
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

	lifecycleMu    sync.Mutex
	params         DatabaseParams
	callbacksToRun []callbackRegistration
	running        bool
}

func New(params DatabaseParams) *Promo {
	return &Promo{params: params}
}

func NewWithDatabase(ctx context.Context, db *sql.DB, options Options) (*Promo, error) {
	client, err := sqlwrap.New(db, toSQLWrapOptions(options))
	if err != nil {
		return nil, err
	}
	return newPromo(ctx, client, false, options), nil
}

func (p *Promo) Run(ctx context.Context) error {
	if p == nil {
		return errors.New("promo: nil service")
	}
	p.lifecycleMu.Lock()
	if p.running {
		p.lifecycleMu.Unlock()
		return errors.New("promo: service is already running")
	}
	p.running = true
	params := p.params
	registrations := append([]callbackRegistration(nil), p.callbacksToRun...)
	p.lifecycleMu.Unlock()

	running, err := open(ctx, params)
	if err != nil {
		p.lifecycleMu.Lock()
		p.running = false
		p.lifecycleMu.Unlock()
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return nil
		}
		return err
	}
	p.adopt(running)
	defer p.Close()

	errCh := make(chan error, len(registrations))
	p.background.Add(len(registrations))
	for _, registration := range registrations {
		registration := registration
		go func() {
			defer p.background.Done()
			errCh <- p.runCallback(registration.ctx, registration.handler, registration.options...)
		}()
	}
	select {
	case <-p.rootCtx.Done():
		return nil
	case err := <-errCh:
		if errors.Is(err, context.Canceled) && p.rootCtx.Err() != nil {
			return nil
		}
		return err
	}
}

func open(ctx context.Context, params DatabaseParams) (*Promo, error) {
	if params.User == "" || params.Database == "" {
		return nil, errors.New("promo: database user and name are required")
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
	if err := bootstrap.Bootstrap(ctx); err != nil {
		_ = bootstrap.Close()
		_ = client.Close()
		return nil, err
	}
	if err := bootstrap.Close(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return newPromo(ctx, client, true, params.Options), nil
}

func (p *Promo) adopt(running *Promo) {
	p.Admin, p.User = running.Admin, running.User
	p.callbacks, p.client, p.ownsClient = running.callbacks, running.client, running.ownsClient
	p.rootCtx, p.rootCancel = running.rootCtx, running.rootCancel
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

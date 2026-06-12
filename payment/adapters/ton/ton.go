package ton

import (
	"context"
	"database/sql"
	"sync"

	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"

	"github.com/elum-utils/services/payment/repository"
)

const (
	ProviderCode = "ton"
	AssetTON     = "TON"

	NetworkMainnet = "mainnet"
	NetworkTestnet = "testnet"

	NetworkConfigURLMainnet = "https://ton.org/global.config.json"
	NetworkConfigURLTestnet = "https://ton.org/testnet-global.config.json"
)

type TON struct {
	repository  *repository.PaymentRepository
	rootCtx     context.Context
	subscribers map[*Sub]struct{}
	mu          sync.Mutex
}

func New(ctx context.Context, db *sqlwrap.Client) *TON {
	return NewWithOptions(ctx, db, repository.Options{})
}

func NewWithOptions(ctx context.Context, db *sqlwrap.Client, options repository.Options) *TON {
	repo, err := repository.NewPreparedPaymentRepositoryWithOptions(context.Background(), db, options)
	if err != nil {
		repo = repository.NewPaymentRepositoryWithOptions(db, options)
	}
	return &TON{
		repository:  repo,
		rootCtx:     ctx,
		subscribers: make(map[*Sub]struct{}),
	}
}

func (a *TON) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	subs := make([]*Sub, 0, len(a.subscribers))
	for sub := range a.subscribers {
		subs = append(subs, sub)
	}
	a.mu.Unlock()

	for _, sub := range subs {
		_ = sub.Close()
	}

	if a.repository == nil {
		return nil
	}
	return a.repository.Close()
}

func (a *TON) bindContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if a == nil {
		return contextutil.Merge(context.Background(), ctx)
	}
	return contextutil.Merge(a.rootCtx, ctx)
}

func (a *TON) withContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return contextutil.Merge(a.rootCtx, ctx)
}

func (a *TON) registerSubscriber(sub *Sub) {
	if a == nil || sub == nil {
		return
	}
	a.mu.Lock()
	a.subscribers[sub] = struct{}{}
	a.mu.Unlock()
}

func (a *TON) unregisterSubscriber(sub *Sub) {
	if a == nil || sub == nil {
		return
	}
	a.mu.Lock()
	delete(a.subscribers, sub)
	a.mu.Unlock()
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

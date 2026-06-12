package admin

import (
	"context"
	"errors"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/internal/utils/contextutil"
	"github.com/elum-utils/services/payment/repository"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

type Admin struct {
	repository *repository.PaymentRepository
	callbacks  *callbackutil.Store
	rootCtx    context.Context
}

func New(ctx context.Context, db *sqlwrap.Client) *Admin {
	return NewWithOptions(ctx, db, repository.Options{})
}

func NewWithOptions(ctx context.Context, db *sqlwrap.Client, options repository.Options) *Admin {
	return &Admin{
		repository: repository.NewPaymentRepositoryWithOptions(db, options),
		callbacks:  callbackutil.NewWithTable(db.DB(), callbackutil.PaymentTable),
		rootCtx:    contextutil.Normalize(ctx),
	}
}

func (a *Admin) Close() error {
	if a == nil {
		return nil
	}
	var err error
	if a.repository != nil {
		err = errors.Join(err, a.repository.Close())
	}
	if a.callbacks != nil {
		err = errors.Join(err, a.callbacks.Close())
	}
	return err
}

func (a *Admin) withContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return contextutil.Merge(a.rootCtx, ctx)
}

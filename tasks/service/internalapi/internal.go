package internalapi

import (
	"context"

	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/tasks/repository"
)

type Internal struct {
	rootCtx    context.Context
	repository *repository.Repository
}

func New(ctx context.Context, db *sqlwrap.Client) *Internal {
	return &Internal{rootCtx: contextutil.Normalize(ctx), repository: repository.New(db)}
}

func NewWithOptions(ctx context.Context, db *sqlwrap.Client, options repository.Options) *Internal {
	return &Internal{rootCtx: contextutil.Normalize(ctx), repository: repository.NewWithOptions(db, options)}
}

func (i *Internal) Close() error {
	if i == nil || i.repository == nil {
		return nil
	}
	return i.repository.Close()
}

func (i *Internal) withContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if i == nil {
		return contextutil.Merge(context.Background(), ctx)
	}
	return contextutil.Merge(i.rootCtx, ctx)
}

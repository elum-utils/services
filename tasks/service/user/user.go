package user

import (
	"context"

	"github.com/elum-utils/services/internal/utils/contextutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/tasks/repository"
)

type User struct {
	rootCtx    context.Context
	repository *repository.Repository
}

func New(ctx context.Context, db *sqlwrap.Client) *User {
	return &User{rootCtx: contextutil.Normalize(ctx), repository: repository.New(db)}
}

func NewWithOptions(ctx context.Context, db *sqlwrap.Client, options repository.Options) *User {
	return &User{rootCtx: contextutil.Normalize(ctx), repository: repository.NewWithOptions(db, options)}
}

func (u *User) Close() error {
	if u == nil || u.repository == nil {
		return nil
	}
	return u.repository.Close()
}

func (u *User) withContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if u == nil {
		return contextutil.Merge(context.Background(), ctx)
	}
	return contextutil.Merge(u.rootCtx, ctx)
}

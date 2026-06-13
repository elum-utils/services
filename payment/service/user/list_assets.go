package user

import (
	"context"
	"errors"
)

func (u *User) ListAssets(ctx context.Context) ([]AssetModel, error) {
	if u == nil || u.assets == nil {
		return nil, errors.New("payment user: asset service is not initialized")
	}
	return u.assets.List(ctx)
}

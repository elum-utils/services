package user

import (
	"context"
)

func (u *User) ListAssets(ctx context.Context) ([]AssetModel, error) {
	if u == nil || u.assets == nil {
		return nil, ErrAssetNotInitialized
	}
	return u.assets.List(ctx)
}

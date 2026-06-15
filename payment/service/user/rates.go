package user

import (
	"context"
)

func (u *User) GetUSDTPrice(ctx context.Context, assetCode string) (*USDTPriceModel, error) {
	if u == nil || u.assets == nil {
		return nil, ErrAssetNotInitialized
	}
	return u.assets.GetUSDTPrice(ctx, assetCode)
}

func (u *User) ListUSDTPrices(ctx context.Context) ([]USDTPriceModel, error) {
	if u == nil || u.assets == nil {
		return nil, ErrAssetNotInitialized
	}
	return u.assets.ListUSDTPrices(ctx)
}

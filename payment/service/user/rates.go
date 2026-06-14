package user

import (
	"context"
	"errors"
)

func (u *User) GetUSDTPrice(ctx context.Context, assetCode string) (*USDTPriceModel, error) {
	if u == nil || u.assets == nil {
		return nil, errors.New("payment user: asset service is not initialized")
	}
	return u.assets.GetUSDTPrice(ctx, assetCode)
}

func (u *User) ListUSDTPrices(ctx context.Context) ([]USDTPriceModel, error) {
	if u == nil || u.assets == nil {
		return nil, errors.New("payment user: asset service is not initialized")
	}
	return u.assets.ListUSDTPrices(ctx)
}

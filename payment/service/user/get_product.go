package user

import (
	"context"
	"errors"
)

func (u *User) GetProduct(ctx context.Context, params GetProductParams) (*ProductModel, error) {
	if u == nil || u.products == nil {
		return nil, errors.New("payment user: product service is not initialized")
	}
	return u.products.Get(ctx, params)
}

func (u *User) GetProductByKey(ctx context.Context, params GetProductByKeyParams) (*ProductModel, error) {
	if u == nil || u.products == nil {
		return nil, errors.New("payment user: product service is not initialized")
	}
	return u.products.GetByKey(ctx, params)
}

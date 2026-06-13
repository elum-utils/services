package user

import (
	"context"
	"errors"
)

func (u *User) ListProducts(ctx context.Context, params ListProductsParams) ([]ProductModel, error) {
	if u == nil || u.products == nil {
		return nil, errors.New("payment user: service is not initialized")
	}
	return u.products.List(ctx, params)
}

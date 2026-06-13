package user

import "github.com/elum-utils/services/payment/service/product"

type User struct {
	products *product.Product
}

func New(products *product.Product) *User {
	return &User{products: products}
}

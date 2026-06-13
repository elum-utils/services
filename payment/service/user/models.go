package user

import (
	"github.com/elum-utils/services/payment/service/checkout"
	"github.com/elum-utils/services/payment/service/product"
	"github.com/elum-utils/services/payment/service/subscription"
	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

type ListProductsParams = product.ListParams
type GetProductParams = product.GetParams
type GetProductByKeyParams = product.GetByKeyParams
type ProductModel = product.ProductModel

type AssetModel = paymentsqlc.PaymentAsset

type CreateOrderParams = checkout.CreateOrderParams
type CreateOrderByKeyParams = checkout.CreateOrderByKeyParams
type OrderModel = checkout.Order
type CreateAttemptParams = checkout.CreateAttemptParams
type AttemptModel = checkout.Attempt

type IsSubscriptionActiveParams = subscription.IsActiveParams

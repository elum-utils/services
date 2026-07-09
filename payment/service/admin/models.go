package admin

import (
	"database/sql"
	"time"

	"github.com/elum-utils/services/payment/repository"
	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
)

const AssetRateSourceDexScreener = repository.AssetRateSourceDexScreener

type ExportRequest = repository.ExportRequest
type ExportPackage = repository.ExportPackage
type ExportProductGroup = repository.ExportProductGroup
type ExportText = repository.ExportText
type ExportProduct = repository.ExportProduct
type ExportItem = repository.ExportItem
type ExportProductItem = repository.ExportProductItem
type ExportPrice = repository.ExportPrice
type ExportTONWallet = repository.ExportTONWallet
type ImportRequest = repository.ImportRequest
type ImportPreview = repository.ImportPreview
type ImportCounts = repository.ImportCounts
type ImportConflict = repository.ImportConflict
type ImportResult = repository.ImportResult

type PageParams struct {
	Limit  int32
	Offset int32
}

type StatsModel struct {
	ProductsTotal    uint64            `json:"products_total"`
	ActiveProducts   uint64            `json:"active_products"`
	VisibleProducts  uint64            `json:"visible_products"`
	OrdersTotal      uint64            `json:"orders_total"`
	PendingOrders    uint64            `json:"pending_orders"`
	FulfilledOrders  uint64            `json:"fulfilled_orders"`
	RefundedOrders   uint64            `json:"refunded_orders"`
	FailedOrders     uint64            `json:"failed_orders"`
	CanceledOrders   uint64            `json:"canceled_orders"`
	PurchaseCount    uint64            `json:"purchase_count"`
	PurchaseQuantity uint64            `json:"purchase_quantity"`
	UniqueBuyers     uint64            `json:"unique_buyers"`
	Assets           []AssetStatsModel `json:"assets"`
}

type ProductStatsModel struct {
	ProductID        string            `json:"product_id"`
	OrdersTotal      uint64            `json:"orders_total"`
	PendingOrders    uint64            `json:"pending_orders"`
	FulfilledOrders  uint64            `json:"fulfilled_orders"`
	RefundedOrders   uint64            `json:"refunded_orders"`
	FailedOrders     uint64            `json:"failed_orders"`
	CanceledOrders   uint64            `json:"canceled_orders"`
	PurchaseCount    uint64            `json:"purchase_count"`
	PurchaseQuantity uint64            `json:"purchase_quantity"`
	UniqueBuyers     uint64            `json:"unique_buyers"`
	Assets           []AssetStatsModel `json:"assets"`
}

type AssetStatsModel struct {
	AssetCode         string `json:"asset_code"`
	PurchaseCount     uint64 `json:"purchase_count"`
	PurchaseQuantity  uint64 `json:"purchase_quantity"`
	GrossAmountMinor  uint64 `json:"gross_amount_minor"`
	RefundCount       uint64 `json:"refund_count"`
	RefundAmountMinor uint64 `json:"refund_amount_minor"`
}

type DailyStatsModel struct {
	Date              time.Time `json:"date"`
	ProductID         string    `json:"product_id,omitempty"`
	AssetCode         string    `json:"asset_code"`
	PurchaseCount     uint64    `json:"purchase_count"`
	PurchaseQuantity  uint64    `json:"purchase_quantity"`
	UniqueBuyers      uint64    `json:"unique_buyers"`
	GrossAmountMinor  uint64    `json:"gross_amount_minor"`
	RefundCount       uint64    `json:"refund_count"`
	RefundAmountMinor uint64    `json:"refund_amount_minor"`
}

type DailyOverviewModel struct {
	Date                 time.Time `json:"date"`
	ProductsTotal        uint64    `json:"products_total"`
	ActiveProducts       uint64    `json:"active_products"`
	VisibleProducts      uint64    `json:"visible_products"`
	OrdersCreated        uint64    `json:"orders_created"`
	DraftOrders          uint64    `json:"draft_orders"`
	PendingPaymentOrders uint64    `json:"pending_payment_orders"`
	PaidOrders           uint64    `json:"paid_orders"`
	FulfilledOrders      uint64    `json:"fulfilled_orders"`
	CanceledOrders       uint64    `json:"canceled_orders"`
	ExpiredOrders        uint64    `json:"expired_orders"`
	RefundedOrders       uint64    `json:"refunded_orders"`
	ChargebackedOrders   uint64    `json:"chargebacked_orders"`
	FailedOrders         uint64    `json:"failed_orders"`
	PurchaseCount        uint64    `json:"purchase_count"`
	PurchaseQuantity     uint64    `json:"purchase_quantity"`
	UniqueBuyers         uint64    `json:"unique_buyers"`
	RefundCount          uint64    `json:"refund_count"`
}

type ProviderUpsertParams struct {
	Code             string
	Title            string
	ProviderKind     paymentsqlc.PaymentProviderProviderKind
	SupportsCreate   bool
	SupportsRedirect bool
	SupportsWebhook  bool
	SupportsRefund   bool
	IsActive         bool
}

type ProviderAssetListParams struct {
	ProviderCode string
	AssetCode    string
	Page         PageParams
}

type TONWalletUpsertParams struct {
	WorkspaceID      string
	Network          string
	WalletAddress    string
	NetworkConfigURL *string
	IsEnabled        bool
}

type ProductGroupListParams struct {
	WorkspaceID string
	Page        PageParams
}

type LocalizationListParams struct {
	WorkspaceID string
	Locale      string
	Page        PageParams
}

type ProductListParams struct {
	WorkspaceID  string
	GroupCode    string
	QuantityMode string
	Page         PageParams
}

type ItemListParams struct {
	WorkspaceID string
	ItemType    string
	Page        PageParams
}

type ProductItemListParams struct {
	WorkspaceID string
	ProductID   string
	ItemID      string
	Page        PageParams
}

type PriceListParams struct {
	WorkspaceID string
	ProductID   string
	AssetCode   string
	Page        PageParams
}

type UpdateAssetRateParams struct {
	AssetCode              string
	ReferenceAssetCode     string
	ReferencePerAssetMinor uint64
	Source                 string
	ObservedAt             time.Time
}

type UpdateAssetRateResult struct {
	UpdatedPrices      uint64 `json:"updated_prices"`
	AffectedProducts   uint64 `json:"affected_products"`
	AffectedWorkspaces uint64 `json:"affected_workspaces"`
}

type ConfigureAssetRateAutoUpdateParams struct {
	AssetCode          string
	ReferenceAssetCode string
	Enabled            bool
	Source             string
	SourceChainID      string
	SourceTokenAddress *string
}

type AssetRateListParams struct {
	AssetCode          string
	ReferenceAssetCode string
	Page               PageParams
}

type ProductLimitCounterListParams struct {
	WorkspaceID    string
	ProductID      string
	PlatformID     int64
	PlatformUserID string
	Page           PageParams
}

type ProductLimitCounterDeleteParams struct {
	WorkspaceID    string
	PlatformID     int64
	ProductID      string
	CounterScope   paymentsqlc.PaymentProductLimitCounterCounterScope
	PlatformUserID string
	WindowStart    sql.NullTime
	WindowEnd      sql.NullTime
}

type PurchaseKeyListParams struct {
	WorkspaceID    string
	ProductID      string
	Status         string
	PlatformID     int64
	PlatformUserID string
	Page           PageParams
}

type OrderListParams struct {
	WorkspaceID    string
	Status         string
	ProductID      string
	PlatformID     int64
	PlatformUserID string
	Page           PageParams
}

type AttemptListParams struct {
	WorkspaceID  string
	OrderID      uint64
	ProviderCode string
	Status       string
	Page         PageParams
}

type EventListParams struct {
	WorkspaceID      string
	ProviderCode     string
	ProcessingStatus string
	Page             PageParams
}

type SubscriptionListParams struct {
	WorkspaceID    string
	ProviderCode   string
	ProductID      string
	Status         string
	PlatformID     int64
	PlatformUserID string
	Page           PageParams
}

type FulfillmentListParams struct {
	WorkspaceID string
	Status      string
	OrderID     uint64
	Page        PageParams
}

type FulfillmentItemListParams struct {
	WorkspaceID   string
	FulfillmentID uint64
	Page          PageParams
}

type RefundCreateParams struct {
	OrderID          uint64
	AttemptID        uint64
	ProviderCode     string
	ProviderRefundID *string
	AmountMinor      uint64
	AssetCode        string
	Status           string
	Reason           *string
}

type RefundListParams struct {
	WorkspaceID  string
	OrderID      uint64
	ProviderCode string
	Status       string
	Page         PageParams
}

type ProviderCursorListParams struct {
	WorkspaceID  string
	ProviderCode string
	Network      string
	Page         PageParams
}

type ProviderTransactionListParams struct {
	WorkspaceID  string
	ProviderCode string
	Network      string
	SourceKey    string
	Status       string
	Page         PageParams
}

type CallbackEventListParams struct {
	SourceService string
	EventType     string
	Status        string
	Page          PageParams
}

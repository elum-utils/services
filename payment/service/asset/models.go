package asset

import paymentsqlc "github.com/elum-utils/services/payment/sqlc"

type UpsertParams struct {
	Code            string
	Title           string
	AssetKind       paymentsqlc.PaymentAssetAssetKind
	Scale           uint16
	Chain           *string
	Network         *string
	ContractAddress *string
	IsActive        bool
}

type ProviderUpsertParams struct {
	ProviderCode    string
	AssetCode       string
	MinAmountMinor  *int64
	MaxAmountMinor  *int64
	MerchantAccount *string
	IsActive        bool
}

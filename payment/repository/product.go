package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/elum-utils/services/payment/sqlc"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
)

type ProductGetParams struct {
	WorkspaceID    string
	AppID          int64
	PlatformID     int64
	PlatformUserID string
	ProductID      string
	AssetCode      string
	Locale         string
	Now            time.Time
}

type ProductGetByKeyParams struct {
	Key       string
	AssetCode string
	Locale    string
	Now       time.Time
}

type ProductCreateKeyParams struct {
	WorkspaceID    string
	AppID          int64
	PlatformID     int64
	PlatformUserID string
	InternalUserID *int64
	ProductID      string
	MaxUses        int32
	ExpiresAt      *time.Time
}

type Product struct {
	WorkspaceID          string
	ID                   string
	LinkURL              sql.NullString
	SizeLabel            sql.NullString
	GroupCode            sql.NullString
	Title                string
	Description          string
	ImageURL             sql.NullString
	PeriodSeconds        sql.NullInt64
	TrialDurationSeconds sql.NullInt64
	QuantityMode         string
	Price                ProductPrice
	Limit                ProductLimit
	Items                []ProductItem
}

type ProductPurchaseKey struct {
	WorkspaceID    string
	AppID          int64
	PlatformID     int64
	PlatformUserID string
	InternalUserID sql.NullInt64
	ProductID      string
}

type ProductPreview struct {
	WorkspaceID          string
	ID                   string
	LinkURL              sql.NullString
	SizeLabel            sql.NullString
	GroupCode            sql.NullString
	Title                string
	Description          string
	ImageURL             sql.NullString
	PeriodSeconds        sql.NullInt64
	TrialDurationSeconds sql.NullInt64
	QuantityMode         string
	Limit                ProductLimit
	Items                []ProductItem
}

type ProductPriceOption struct {
	PriceID             uint64
	ProductID           string
	AssetCode           string
	AssetTitle          string
	AssetKind           string
	Scale               uint16
	Chain               sql.NullString
	Network             sql.NullString
	ContractAddress     sql.NullString
	ListAmountMinor     uint64
	DiscountAmountMinor uint64
	PayableAmountMinor  uint64
	ProviderCodes       []string
}

type ProductPrice struct {
	ID                  uint64
	AssetCode           string
	ListAmountMinor     uint64
	DiscountAmountMinor uint64
	PayableAmountMinor  uint64
}

type ProductLimit struct {
	Global ProductLimitRule
	User   ProductLimitRule
}

type ProductLimitRule struct {
	Limit         int32
	Interval      string
	IntervalCount int32
	LockUntil     sql.NullTime
}

type ProductItem struct {
	ID           string
	Quantity     int64
	RewardType   string
	DurationUnit *string
	Type         sql.NullString
	Title        string
	Description  string
	Rarity       sql.NullString
	Position     sql.NullInt32
}

func (r *PaymentRepository) GetProduct(ctx context.Context, params ProductGetParams) (Product, error) {
	workspaceID, err := requireWorkspaceID(params.WorkspaceID)
	if err != nil {
		return Product{}, err
	}
	locale := params.Locale
	if locale == "" {
		locale = "ru"
	}

	now, err := r.catalogNow(ctx, params.Now)
	if err != nil {
		return Product{}, err
	}

	product, err := r.getProductCatalog(ctx, workspaceID, params.ProductID, params.AssetCode, locale, now)
	if err != nil {
		return Product{}, err
	}

	if err := r.attachProductLimitLocks(ctx, &product, params.PlatformID, params.PlatformUserID); err != nil {
		return Product{}, err
	}

	return product, nil
}

func (r *PaymentRepository) getProductCatalog(
	ctx context.Context,
	workspaceID string,
	productID string,
	assetCode string,
	locale string,
	now time.Time,
) (Product, error) {
	key := paymentCacheKey("product_catalog", workspaceID, productID, assetCode, locale)
	rows, err := queryPaymentCache(ctx, r, workspaceID, key, func(ctx context.Context) ([]sqlc.ListProductCatalogCacheRowsRow, error) {
		return r.q.ListProductCatalogCacheRows(ctx, sqlc.ListProductCatalogCacheRowsParams{
			ProductID:   productID,
			WorkspaceID: workspaceID,
			AssetCode:   assetCode,
			Locale:      locale,
		})
	})
	if err != nil {
		return Product{}, err
	}
	return mapProductCatalogRows(rows, now)
}

func (r *PaymentRepository) attachProductLimitLocks(ctx context.Context, product *Product, platformID int64, platformUserID string) error {
	var err error
	product.Limit.Global.LockUntil, err = r.getProductLimitLock(ctx, productLimitQuery{
		workspaceID:    product.WorkspaceID,
		platformID:     platformID,
		platformUserID: "",
		productID:      product.ID,
		limit:          product.Limit.Global.Limit,
		interval:       product.Limit.Global.Interval,
		intervalCount:  product.Limit.Global.IntervalCount,
	})
	if err != nil {
		return err
	}

	product.Limit.User.LockUntil, err = r.getProductLimitLock(ctx, productLimitQuery{
		workspaceID:    product.WorkspaceID,
		platformID:     platformID,
		platformUserID: platformUserID,
		productID:      product.ID,
		limit:          product.Limit.User.Limit,
		interval:       product.Limit.User.Interval,
		intervalCount:  product.Limit.User.IntervalCount,
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *PaymentRepository) getCheckoutProduct(ctx context.Context, params ProductGetParams) (Product, error) {
	workspaceID, err := requireWorkspaceID(params.WorkspaceID)
	if err != nil {
		return Product{}, err
	}
	locale := normalizedLocale(params.Locale)

	now, err := r.catalogNow(ctx, params.Now)
	if err != nil {
		return Product{}, err
	}
	product, err := r.getProductCatalog(ctx, workspaceID, params.ProductID, params.AssetCode, locale, now)
	if err != nil {
		return Product{}, err
	}
	product.Items = nil
	if err := r.attachProductLimitLocks(ctx, &product, params.PlatformID, params.PlatformUserID); err != nil {
		return Product{}, err
	}

	return product, nil
}

func (r *PaymentRepository) GetProductByKey(ctx context.Context, params ProductGetByKeyParams) (Product, error) {
	now := params.Now
	if now.IsZero() {
		now = time.Now()
	}

	key, err := r.q.GetPurchaseKeyByHash(ctx, hashPurchaseKey(params.Key))
	if err != nil {
		return Product{}, err
	}
	if !isPurchaseKeyUsable(key, now) {
		return Product{}, sql.ErrNoRows
	}

	return r.GetProduct(ctx, ProductGetParams{
		WorkspaceID:    key.WorkspaceID,
		AppID:          key.AppID,
		PlatformID:     key.PlatformID,
		PlatformUserID: key.PlatformUserID,
		ProductID:      key.ProductID,
		AssetCode:      params.AssetCode,
		Locale:         params.Locale,
		Now:            now,
	})
}

type ProductPreviewParams struct {
	WorkspaceID    string
	AppID          int64
	PlatformID     int64
	PlatformUserID string
	ProductID      string
	Locale         string
	Now            time.Time
}

func (r *PaymentRepository) GetProductPreview(ctx context.Context, params ProductPreviewParams) (ProductPreview, error) {
	workspaceID, err := requireWorkspaceID(params.WorkspaceID)
	if err != nil {
		return ProductPreview{}, err
	}
	locale := params.Locale
	if locale == "" {
		locale = "ru"
	}

	now, err := r.catalogNow(ctx, params.Now)
	if err != nil {
		return ProductPreview{}, err
	}

	key := paymentCacheKey("product_preview_catalog", workspaceID, params.ProductID, locale)
	rows, err := queryPaymentCache(ctx, r, workspaceID, key, func(ctx context.Context) ([]sqlc.ListProductPreviewCatalogCacheRowsRow, error) {
		return r.q.ListProductPreviewCatalogCacheRows(ctx, sqlc.ListProductPreviewCatalogCacheRowsParams{
			ProductID:   params.ProductID,
			WorkspaceID: workspaceID,
			Locale:      locale,
		})
	})
	if err != nil {
		return ProductPreview{}, err
	}
	product, err := mapProductPreviewCatalogRows(rows, now)
	if err != nil {
		return ProductPreview{}, err
	}

	product.Limit.Global.LockUntil, err = r.getProductLimitLock(ctx, productLimitQuery{
		workspaceID:    product.WorkspaceID,
		platformID:     params.PlatformID,
		platformUserID: "",
		productID:      product.ID,
		limit:          product.Limit.Global.Limit,
		interval:       product.Limit.Global.Interval,
		intervalCount:  product.Limit.Global.IntervalCount,
	})
	if err != nil {
		return ProductPreview{}, err
	}

	product.Limit.User.LockUntil, err = r.getProductLimitLock(ctx, productLimitQuery{
		workspaceID:    product.WorkspaceID,
		platformID:     params.PlatformID,
		platformUserID: params.PlatformUserID,
		productID:      product.ID,
		limit:          product.Limit.User.Limit,
		interval:       product.Limit.User.Interval,
		intervalCount:  product.Limit.User.IntervalCount,
	})
	if err != nil {
		return ProductPreview{}, err
	}

	return product, nil
}

func (r *PaymentRepository) ListProductPriceOptions(ctx context.Context, workspaceID string, productID string) ([]ProductPriceOption, error) {
	workspaceID, err := requireWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}
	now, err := r.catalogNow(ctx, time.Time{})
	if err != nil {
		return nil, err
	}
	key := paymentCacheKey("product_price_options", workspaceID, productID)
	rows, err := queryPaymentCache(ctx, r, workspaceID, key, func(ctx context.Context) ([]sqlc.ListProductPriceOptionCatalogRowsRow, error) {
		return r.q.ListProductPriceOptionCatalogRows(ctx, sqlc.ListProductPriceOptionCatalogRowsParams{
			WorkspaceID: workspaceID,
			ProductID:   productID,
		})
	})
	if err != nil {
		return nil, err
	}
	options := make([]ProductPriceOption, 0, len(rows))
	for _, row := range rows {
		if now.Before(row.StartsAt) || now.After(row.EndsAt) {
			continue
		}
		options = append(options, ProductPriceOption{
			PriceID:             row.PriceID,
			ProductID:           row.ProductID,
			AssetCode:           row.AssetCode,
			AssetTitle:          row.AssetTitle,
			AssetKind:           string(row.AssetKind),
			Scale:               row.Scale,
			Chain:               row.Chain,
			Network:             row.Network,
			ContractAddress:     row.ContractAddress,
			ListAmountMinor:     row.ListAmountMinor,
			DiscountAmountMinor: row.DiscountAmountMinor,
			PayableAmountMinor:  row.ListAmountMinor - row.DiscountAmountMinor,
			ProviderCodes:       splitProviderCodes(row.ProviderCodes),
		})
	}
	return options, nil
}

func (r *PaymentRepository) catalogNow(ctx context.Context, value time.Time) (time.Time, error) {
	if !value.IsZero() {
		return value, nil
	}
	return r.databaseNow(ctx)
}

func mapProductCatalogRows(rows []sqlc.ListProductCatalogCacheRowsRow, now time.Time) (Product, error) {
	if len(rows) == 0 {
		return Product{}, sql.ErrNoRows
	}

	selected, ok := selectProductCatalogPrice(rows, now)
	if !ok {
		return Product{}, sql.ErrNoRows
	}

	product := Product{
		WorkspaceID:          selected.WorkspaceID,
		ID:                   selected.ProductID,
		LinkURL:              selected.LinkUrl,
		SizeLabel:            selected.SizeLabel,
		GroupCode:            selected.GroupCode,
		Title:                selected.ProductTitle,
		Description:          selected.ProductDescription,
		ImageURL:             selected.ImageUrl,
		PeriodSeconds:        selected.PeriodSeconds,
		TrialDurationSeconds: selected.TrialDurationSeconds,
		QuantityMode:         string(selected.QuantityMode),
		Price: ProductPrice{
			ID:                  selected.PriceID,
			AssetCode:           selected.AssetCode,
			ListAmountMinor:     selected.ListAmountMinor,
			DiscountAmountMinor: selected.DiscountAmountMinor,
			PayableAmountMinor:  selected.ListAmountMinor - selected.DiscountAmountMinor,
		},
		Limit: ProductLimit{
			Global: ProductLimitRule{
				Limit:         selected.GlobalLimit,
				Interval:      string(selected.GlobalInterval),
				IntervalCount: selected.GlobalIntervalCount,
			},
			User: ProductLimitRule{
				Limit:         selected.UserLimit,
				Interval:      string(selected.UserInterval),
				IntervalCount: selected.UserIntervalCount,
			},
		},
		Items: make([]ProductItem, 0, len(rows)),
	}

	for _, row := range rows {
		if row.PriceID != selected.PriceID || row.ItemID == "" {
			continue
		}
		product.Items = append(product.Items, ProductItem{
			ID:           row.ItemID,
			Quantity:     row.ItemQuantity,
			RewardType:   string(row.RewardType),
			DurationUnit: paymentCacheDurationUnitPtr(row.DurationUnit),
			Type:         row.ItemType,
			Title:        row.ItemTitle,
			Description:  row.ItemDescription,
			Rarity:       row.ItemRarity,
			Position:     row.ItemPosition,
		})
	}

	return product, nil
}

func selectProductCatalogPrice(rows []sqlc.ListProductCatalogCacheRowsRow, now time.Time) (sqlc.ListProductCatalogCacheRowsRow, bool) {
	for _, row := range rows {
		if productCatalogRowActive(row.IsVisible, row.IsClosed, row.AvailableFrom, row.AvailableUntil, row.PriceStartsAt, row.PriceEndsAt, now) {
			return row, true
		}
	}
	return sqlc.ListProductCatalogCacheRowsRow{}, false
}

func mapProductPreviewCatalogRows(rows []sqlc.ListProductPreviewCatalogCacheRowsRow, now time.Time) (ProductPreview, error) {
	if len(rows) == 0 {
		return ProductPreview{}, sql.ErrNoRows
	}

	selected, ok := selectProductPreviewCatalogPrice(rows, now)
	if !ok {
		return ProductPreview{}, sql.ErrNoRows
	}

	product := ProductPreview{
		WorkspaceID:          selected.WorkspaceID,
		ID:                   selected.ProductID,
		LinkURL:              selected.LinkUrl,
		SizeLabel:            selected.SizeLabel,
		GroupCode:            selected.GroupCode,
		Title:                selected.ProductTitle,
		Description:          selected.ProductDescription,
		ImageURL:             selected.ImageUrl,
		PeriodSeconds:        selected.PeriodSeconds,
		TrialDurationSeconds: selected.TrialDurationSeconds,
		QuantityMode:         string(selected.QuantityMode),
		Limit: ProductLimit{
			Global: ProductLimitRule{
				Limit:         selected.GlobalLimit,
				Interval:      string(selected.GlobalInterval),
				IntervalCount: selected.GlobalIntervalCount,
			},
			User: ProductLimitRule{
				Limit:         selected.UserLimit,
				Interval:      string(selected.UserInterval),
				IntervalCount: selected.UserIntervalCount,
			},
		},
		Items: make([]ProductItem, 0, len(rows)),
	}

	for _, row := range rows {
		if row.PriceID != selected.PriceID || row.ItemID == "" {
			continue
		}
		product.Items = append(product.Items, ProductItem{
			ID:           row.ItemID,
			Quantity:     row.ItemQuantity,
			RewardType:   string(row.RewardType),
			DurationUnit: paymentCacheDurationUnitPtr(row.DurationUnit),
			Type:         row.ItemType,
			Title:        row.ItemTitle,
			Description:  row.ItemDescription,
			Rarity:       row.ItemRarity,
			Position:     row.ItemPosition,
		})
	}

	return product, nil
}

func paymentCacheDurationUnitPtr(value sqlc.NullPaymentProductCacheDurationUnit) *string {
	if !value.Valid {
		return nil
	}
	unit := string(value.PaymentProductCacheDurationUnit)
	return &unit
}

func selectProductPreviewCatalogPrice(rows []sqlc.ListProductPreviewCatalogCacheRowsRow, now time.Time) (sqlc.ListProductPreviewCatalogCacheRowsRow, bool) {
	for _, row := range rows {
		if productCatalogRowActive(row.IsVisible, row.IsClosed, row.AvailableFrom, row.AvailableUntil, row.PriceStartsAt, row.PriceEndsAt, now) {
			return row, true
		}
	}
	return sqlc.ListProductPreviewCatalogCacheRowsRow{}, false
}

func productCatalogRowActive(isVisible bool, isClosed bool, availableFrom time.Time, availableUntil time.Time, priceStartsAt time.Time, priceEndsAt time.Time, now time.Time) bool {
	return isVisible &&
		!isClosed &&
		!now.Before(availableFrom) &&
		!now.After(availableUntil) &&
		!now.Before(priceStartsAt) &&
		!now.After(priceEndsAt)
}

func (r *PaymentRepository) CreateProductPurchaseKey(ctx context.Context, params ProductCreateKeyParams) (string, error) {
	workspaceID, err := requireWorkspaceID(params.WorkspaceID)
	if err != nil {
		return "", err
	}
	maxUses := params.MaxUses
	if maxUses <= 0 {
		maxUses = 1
	}

	key, err := newPurchaseKey()
	if err != nil {
		return "", err
	}

	_, err = r.q.CreatePurchaseKey(ctx, sqlc.CreatePurchaseKeyParams{
		KeyHash:        hashPurchaseKey(key),
		WorkspaceID:    workspaceID,
		AppID:          params.AppID,
		PlatformID:     params.PlatformID,
		PlatformUserID: params.PlatformUserID,
		InternalUserID: sqlwrap.NullFromPtr(params.InternalUserID, func(v int64) sql.NullInt64 {
			return sql.NullInt64{Int64: v, Valid: true}
		}),
		ProductID: params.ProductID,
		MaxUses:   maxUses,
		ExpiresAt: sqlwrap.NullTimeFromPtr(params.ExpiresAt),
	})
	if err != nil {
		return "", err
	}

	return key, nil
}

func hashPurchaseKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func newPurchaseKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func isPurchaseKeyUsable(key sqlc.PaymentPurchaseKey, now time.Time) bool {
	if key.Status != sqlc.PaymentPurchaseKeyStatusActive {
		return false
	}
	if key.ExpiresAt.Valid && !key.ExpiresAt.Time.After(now) {
		return false
	}
	return key.UsedCount < key.MaxUses
}

func splitProviderCodes(value sql.NullString) []string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return strings.Split(value.String, ",")
}

type productLimitQuery struct {
	workspaceID    string
	platformID     int64
	platformUserID string
	productID      string
	limit          int32
	interval       string
	intervalCount  int32
	amount         uint64
}

func (r *PaymentRepository) getProductLimitLock(ctx context.Context, query productLimitQuery) (sql.NullTime, error) {
	if query.limit <= 0 || query.interval == "UNLIMITED" {
		return sql.NullTime{}, nil
	}

	now, err := r.databaseNow(ctx)
	if err != nil {
		return sql.NullTime{}, err
	}

	start, end, ok := limitWindow(query.interval, query.intervalCount, now)
	if !ok {
		return sql.NullTime{}, nil
	}

	scope := sqlc.PaymentProductLimitCounterCounterScopeGlobal
	platformUserID := ""
	if query.platformUserID == "" {
		scope = sqlc.PaymentProductLimitCounterCounterScopeGlobal
	} else {
		scope = sqlc.PaymentProductLimitCounterCounterScopeUser
		platformUserID = query.platformUserID
	}

	total, err := r.q.GetProductLimitCounterCount(ctx, sqlc.GetProductLimitCounterCountParams{
		WorkspaceID:    query.workspaceID,
		PlatformID:     query.platformID,
		ProductID:      query.productID,
		CounterScope:   scope,
		PlatformUserID: platformUserID,
		WindowStart:    start,
		WindowEnd:      end,
	})
	if err == sql.ErrNoRows {
		return sql.NullTime{}, nil
	}
	if err != nil {
		return sql.NullTime{}, err
	}
	amount := normalizeLimitAmount(query.amount)
	limit := uint64(query.limit)
	if amount <= limit && total <= limit-amount {
		return sql.NullTime{}, nil
	}

	return sql.NullTime{Time: end, Valid: true}, nil
}

func normalizeLimitAmount(amount uint64) uint64 {
	if amount == 0 {
		return 1
	}
	return amount
}

func (r *PaymentRepository) databaseNow(ctx context.Context) (time.Time, error) {
	return sqlwrap.Query(ctx, r.db, sqlwrap.Params{Timeout: r.timeout}, func(ctx context.Context) (time.Time, error) {
		var value time.Time
		if err := r.db.QueryRowContext(ctx, "SELECT NOW()").Scan(&value); err != nil {
			return time.Time{}, err
		}
		return value, nil
	})
}

func limitWindow(interval string, intervalCount int32, now time.Time) (time.Time, time.Time, bool) {
	count := int(intervalCount)
	if count <= 0 {
		count = 1
	}

	anchor := time.Date(2024, 1, 1, 0, 0, 0, 0, now.Location())
	switch interval {
	case "SECOND":
		return fixedLimitWindow(anchor, now, time.Duration(count)*time.Second)
	case "MINUTE":
		return fixedLimitWindow(anchor, now, time.Duration(count)*time.Minute)
	case "HOUR":
		return fixedLimitWindow(anchor, now, time.Duration(count)*time.Hour)
	case "DAY":
		return fixedLimitWindow(anchor, now, time.Duration(count)*24*time.Hour)
	case "WEEK":
		return fixedLimitWindow(anchor, now, time.Duration(count)*7*24*time.Hour)
	case "MONTH":
		start := monthLimitWindow(anchor, now, count)
		return start, start.AddDate(0, count, 0), true
	case "ONCE":
		end := anchor.AddDate(100, 0, 0)
		return anchor, end, true
	default:
		return time.Time{}, time.Time{}, false
	}
}

func fixedLimitWindow(anchor time.Time, now time.Time, duration time.Duration) (time.Time, time.Time, bool) {
	if duration <= 0 {
		return time.Time{}, time.Time{}, false
	}
	if now.Before(anchor) {
		return anchor, anchor.Add(duration), true
	}
	elapsed := now.Sub(anchor)
	start := anchor.Add(time.Duration(int64(elapsed/duration)) * duration)
	return start, start.Add(duration), true
}

func monthLimitWindow(anchor time.Time, now time.Time, count int) time.Time {
	if now.Before(anchor) {
		return anchor
	}
	months := (now.Year()-anchor.Year())*12 + int(now.Month()) - int(anchor.Month())
	bucket := months / count * count
	return anchor.AddDate(0, bucket, 0)
}

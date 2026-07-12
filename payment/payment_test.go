package payment

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	services "github.com/elum-utils/services"
	utils "github.com/elum-utils/services/internal/utils"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/payment/adapters/platega"
	"github.com/elum-utils/services/payment/adapters/telegramstars"
	paymentton "github.com/elum-utils/services/payment/adapters/ton"
	paymentvkma "github.com/elum-utils/services/payment/adapters/vkma"
	"github.com/elum-utils/services/payment/adapters/yookassa"
	"github.com/elum-utils/services/payment/repository"
	"github.com/elum-utils/services/payment/service/admin"
	"github.com/elum-utils/services/payment/service/checkout"
	"github.com/elum-utils/services/payment/service/operational"
	"github.com/elum-utils/services/payment/service/product"
	paymentrefund "github.com/elum-utils/services/payment/service/refund"
	"github.com/elum-utils/services/payment/service/subscription"
	"github.com/elum-utils/services/payment/service/user"
	paymentsqlc "github.com/elum-utils/services/payment/sqlc"
	"github.com/elum-utils/sign/vkmashop"
	json "github.com/goccy/go-json"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/xssnick/tonutils-go/address"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchDexScreenerPricesBatchesAddressesAndSelectsLiquidity(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/tokens/v1/ton/token-a,token-b" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`[
			{"baseToken":{"address":"token-a"},"priceUsd":"1.25","liquidity":{"usd":100}},
			{"baseToken":{"address":"token-a"},"priceUsd":"1.30","liquidity":{"usd":500}},
			{"baseToken":{"address":"token-b"},"priceUsd":"0.004","liquidity":{"usd":250}},
			{"baseToken":{"address":"unexpected"},"priceUsd":"999","liquidity":{"usd":999999}}
		]`)),
			Request: r,
		}, nil
	})}

	prices, err := fetchDexScreenerPrices(
		context.Background(),
		client,
		"https://dex.example",
		"ton",
		[]repository.DueAssetRateUpdate{
			{SourceTokenAddress: "token-a"},
			{SourceTokenAddress: "token-b"},
			{SourceTokenAddress: "token-a"},
		},
	)
	if err != nil {
		t.Fatalf("fetch prices: %v", err)
	}
	if prices["token-a"] != 1_300_000 {
		t.Fatalf("unexpected token-a price: %d", prices["token-a"])
	}
	if prices["token-b"] != 4_000 {
		t.Fatalf("unexpected token-b price: %d", prices["token-b"])
	}
	if _, ok := prices["unexpected"]; ok {
		t.Fatal("unexpected token must not be returned")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestSelectDexScreenerPricesRejectsQuoteAndInvalidPrice(t *testing.T) {
	pairs := []dexScreenerPair{{PriceUSD: "0"}}
	pairs[0].BaseToken.Address = "token-a"

	prices := selectDexScreenerPrices(pairs, map[string]struct{}{"token-a": {}})
	if len(prices) != 0 {
		t.Fatalf("expected invalid price to be ignored: %#v", prices)
	}
}

func TestSelectDexScreenerPricesCalculatesQuoteTokenUSDPrice(t *testing.T) {
	pair := dexScreenerPair{
		PriceUSD:    "1.0019",
		PriceNative: "0.5788",
	}
	pair.BaseToken.Address = "usdt"
	pair.QuoteToken.Address = "ton"
	pair.Liquidity = &struct {
		USD float64 `json:"usd"`
	}{USD: 6_525_315}

	prices := selectDexScreenerPrices(
		[]dexScreenerPair{pair},
		map[string]struct{}{"ton": {}},
	)
	if prices["ton"] != 1_730_996 {
		t.Fatalf("unexpected TON price: %d", prices["ton"])
	}
}

func TestMergeContextsCancelsOnLifecycleDone(t *testing.T) {
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	methodCtx := context.Background()

	ctx, cancel := mergeContexts(lifecycleCtx, methodCtx)
	defer cancel()

	lifecycleCancel()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("expected merged context to be canceled by lifecycle context")
	}
}

func TestMergeContextsCancelsOnMethodDone(t *testing.T) {
	lifecycleCtx := context.Background()
	methodCtx, methodCancel := context.WithCancel(context.Background())

	ctx, cancel := mergeContexts(lifecycleCtx, methodCtx)
	defer cancel()

	methodCancel()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("expected merged context to be canceled by method context")
	}
}

func TestIsReady(t *testing.T) {
	var nilService *Payment
	if nilService.IsReady() {
		t.Fatal("nil payment must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized payment must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx, service.Admin, service.Operational, service.User = ctx, &admin.Admin{}, &operational.Operational{}, &user.User{}
	service.Adapters = &Adapters{}
	if !service.IsReady() {
		t.Fatal("initialized payment must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed payment must not be ready")
	}
}

const (
	paymentPostgresHost     = "localhost"
	paymentPostgresPort     = 5432
	paymentPostgresUsername = "postgres"
	paymentPostgresPassword = "RBTX0DXKbagvCy2XCAi4qHt0cjeSD6bU"

	mysqlControlHost     = paymentPostgresHost
	mysqlControlUsername = paymentPostgresUsername
	mysqlControlPassword = paymentPostgresPassword

	paymentTestDB = "payment_test"
)

func TestBootstrapRealPostgres(t *testing.T) {
	dbName := paymentTestDB

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminDB, err := openPaymentPostgres("postgres")
	if err != nil {
		t.Fatalf("open admin postgres connection: %v", err)
	}
	defer adminDB.Close()

	if err := recreatePaymentTestDatabase(ctx, adminDB, dbName); err != nil {
		t.Fatalf("recreate database %s: %v", dbName, err)
	}

	appDB, err := openPaymentPostgres(dbName)
	if err != nil {
		t.Fatalf("open payment postgres connection: %v", err)
	}
	defer appDB.Close()

	db, err := sqlwrap.New(appDB)
	if err != nil {
		t.Fatalf("create sql client: %v", err)
	}

	repo := repository.NewPaymentRepository(db)
	if err := repo.Bootstrap(ctx, filepath.Join("sqlc", "schema.sql")); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	payments, err := NewWithDatabase(ctx, appDB, paymentTestOptions())
	if err != nil {
		t.Fatalf("create payment service: %v", err)
	}
	defer payments.Close()
	if payments.User == nil {
		t.Fatal("payment user service is nil")
	}
	if payments.Admin == nil {
		t.Fatal("payment admin service is nil")
	}
	if payments.Adapters == nil {
		t.Fatal("payment adapters are nil")
	}
	if payments.Adapters.VKMA == nil {
		t.Fatal("payment vkma service is nil")
	}
	if payments.Adapters.YooKassa == nil {
		t.Fatal("payment yookassa service is nil")
	}
	if payments.Adapters.Platega == nil {
		t.Fatal("payment platega service is nil")
	}
	if payments.Adapters.TON == nil {
		t.Fatal("payment ton service is nil")
	}
	if payments.Adapters.TelegramStars == nil {
		t.Fatal("payment telegram stars service is nil")
	}

	providers, err := repo.ListProviders(ctx)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(providers) < 5 {
		t.Fatalf("expected seeded providers, got %d", len(providers))
	}

	assets, err := repo.ListAssets(ctx)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) < 6 {
		t.Fatalf("expected seeded assets, got %d", len(assets))
	}

	if _, err := repo.GetProviderAsset(ctx, "yookassa", "RUB"); err != nil {
		t.Fatalf("get yookassa/RUB provider asset: %v", err)
	}
	if _, err := repo.GetProviderAsset(ctx, "ton", "DOGS_TON"); err != nil {
		t.Fatalf("get ton/DOGS_TON provider asset: %v", err)
	}
	if _, err := repo.GetProviderAsset(ctx, "ton", "NOT_TON"); err != nil {
		t.Fatalf("get ton/NOT_TON provider asset: %v", err)
	}
	if _, err := repo.GetProviderAsset(ctx, "ton", "MAJOR_TON"); err != nil {
		t.Fatalf("get ton/MAJOR_TON provider asset: %v", err)
	}
}

func TestRunCreatesDatabaseSchemaAndCallbackTable(t *testing.T) {
	const database = "payment_run_bootstrap_test"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminDB, err := openPaymentPostgres("postgres")
	if err != nil {
		t.Fatalf("open admin postgres connection: %v", err)
	}
	defer adminDB.Close()
	if err := recreatePaymentTestDatabase(ctx, adminDB, database); err != nil {
		t.Fatalf("recreate test database: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+quoteIdentifier(database))
	})
	appDB, err := openPaymentPostgres(database)
	if err != nil {
		t.Fatalf("open payment postgres connection: %v", err)
	}
	defer appDB.Close()

	service := New(DatabaseParams{
		User:     paymentPostgresUsername,
		Password: paymentPostgresPassword,
		Database: database,
		Host:     paymentPostgresHost,
		Port:     paymentPostgresPort,
	})
	runCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		done <- service.Run(runCtx)
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		var tableCount int
		err := appDB.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('payment_product', 'payment_clb_event')`).Scan(&tableCount)
		if err == nil && tableCount == 2 {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("Run returned during bootstrap: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			stop()
			t.Fatalf("Run did not complete payment bootstrap: tables=%d err=%v", tableCount, err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Run must still be blocked after the complete schema is ready.
	select {
	case err := <-done:
		t.Fatalf("Run returned before cancellation: %v", err)
	default:
	}

	stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run after cancellation: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not stop after cancellation")
	}
}

func openPaymentPostgres(database string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		paymentPostgresHost,
		paymentPostgresPort,
		paymentPostgresUsername,
		paymentPostgresPassword,
		database,
	)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func openMySQL(_ string, dbName string) (*sql.DB, error) {
	if dbName == "" {
		dbName = "postgres"
	}
	return openPaymentPostgres(dbName)
}

func paymentTestDSN(t interface{ Helper() }) string {
	t.Helper()
	return ""
}

func recreatePaymentTestDatabase(ctx context.Context, db *sql.DB, dbName string) error {
	if _, err := db.ExecContext(ctx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", dbName); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+quoteIdentifier(dbName)); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, "CREATE DATABASE "+quoteIdentifier(dbName))
	return err
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func paymentTestIdentity(workspaceID string, appID int64, platformID int64, platformUserID string) services.Identity {
	return services.Identity{
		WorkspaceID:    workspaceID,
		AppID:          appID,
		PlatformID:     platformID,
		PlatformUserID: platformUserID,
	}
}

func TestPaymentAssetCRUD(t *testing.T) {
	env := setupPaymentIntegrationTest(t)

	chain := "ton"
	network := "mainnet"
	contract := "EQ_TEST_ASSET_CRUD"
	minAmount := int64(1)

	if err := env.api.Operational.UpsertAsset(env.ctx, operational.AssetUpsertParams{
		Code:            "CRUD_TON",
		Title:           "CRUD Token",
		AssetKind:       paymentsqlc.PaymentAssetAssetKindCryptoJetton,
		Scale:           9,
		Chain:           &chain,
		Network:         &network,
		ContractAddress: &contract,
		IsActive:        true,
	}); err != nil {
		t.Fatalf("upsert asset: %v", err)
	}

	if err := env.api.Operational.UpsertProviderAsset(env.ctx, operational.ProviderAssetUpsertParams{
		ProviderCode:   "ton",
		AssetCode:      "CRUD_TON",
		MinAmountMinor: &minAmount,
		IsActive:       true,
	}); err != nil {
		t.Fatalf("upsert provider asset: %v", err)
	}

	providerAsset, err := env.api.Admin.GetProviderAsset(env.ctx, "ton", "CRUD_TON")
	if err != nil {
		t.Fatalf("get provider asset: %v", err)
	}
	if !providerAsset.IsActive || !providerAsset.MinAmountMinor.Valid || providerAsset.MinAmountMinor.Int64 != minAmount {
		t.Fatalf("unexpected provider asset: %#v", providerAsset)
	}

	if err := env.api.Operational.UpsertAsset(env.ctx, operational.AssetUpsertParams{
		Code:            "CRUD_TON",
		Title:           "CRUD Token Updated",
		AssetKind:       paymentsqlc.PaymentAssetAssetKindCryptoJetton,
		Scale:           6,
		Chain:           &chain,
		Network:         &network,
		ContractAddress: &contract,
		IsActive:        true,
	}); err != nil {
		t.Fatalf("update asset: %v", err)
	}

	assets, err := env.api.User.ListAssets(env.ctx, user.ListAssetsParams{})
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	found := false
	for _, row := range assets {
		if row.Code == "CRUD_TON" {
			found = row.Title == "CRUD Token Updated" && row.Scale == 6
			break
		}
	}
	if !found {
		t.Fatal("expected updated CRUD_TON asset in list")
	}

	if rows, err := env.api.Operational.DeleteProviderAsset(env.ctx, "ton", "CRUD_TON"); err != nil || rows != 1 {
		t.Fatalf("delete provider asset rows=%d err=%v", rows, err)
	}
	if rows, err := env.api.Operational.DeleteAsset(env.ctx, "CRUD_TON"); err != nil || rows != 1 {
		t.Fatalf("delete asset rows=%d err=%v", rows, err)
	}
}

type paymentTestEnv struct {
	ctx    context.Context
	db     *sql.DB
	client *sqlwrap.Client
	api    *Payment
}

const testWorkspaceID = "00000000-0000-0000-0000-000000000001"

func TestPaymentRunBlocksUntilContextCanceled(t *testing.T) {
	setupPaymentIntegrationTest(t)

	service := New(DatabaseParams{
		User:     mysqlControlUsername,
		Password: mysqlControlPassword,
		Database: paymentTestDB,
		Host:     mysqlControlHost,
		Port:     paymentPostgresPort,
	})
	if err := service.OnCallback(context.Background(), func(ctx Context) error {
		return ctx.Successful()
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- service.Run(runCtx)
	}()

	select {
	case err := <-done:
		t.Fatalf("Run returned before cancellation: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if err := service.OnCallback(context.Background(), func(Context) error {
		return nil
	}); err == nil {
		t.Fatal("callback registration after Run must fail")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run after cancellation: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}
}

func TestPaymentCatalogCheckoutAndGiftCycle(t *testing.T) {
	// Test database setup.
	// Create the MySQL database connection and bootstrap the payment schema.
	// Verify every following block runs through the same initialized payment API.
	env := setupPaymentIntegrationTest(t)
	productID := fmt.Sprintf("test_product_%d", time.Now().UnixNano())
	groupCode := fmt.Sprintf("test_group_%d", time.Now().UnixNano())
	itemID := fmt.Sprintf("test_item_%d", time.Now().UnixNano())
	productTitleKey := productID + ".title"
	productDescriptionKey := productID + ".description"
	itemTitleKey := itemID + ".title"
	itemDescriptionKey := itemID + ".description"
	now := time.Now()
	availableFrom := now.Add(-time.Hour)
	availableUntil := now.Add(time.Hour)
	priceStartsAt := now.Add(-time.Hour)
	priceEndsAt := now.Add(time.Hour)
	periodSeconds := int64(86400)

	// Product creation.
	// Create a product group and product through the public Product CRUD API.
	// Verify checkout tests do not depend on direct SQL catalog seeding.
	if err := env.api.Admin.SaveProductGroup(env.ctx, product.UpsertGroupParams{
		WorkspaceID:    testWorkspaceID,
		Code:           groupCode,
		TitleKey:       utils.Ref(groupCode + ".title"),
		DescriptionKey: utils.Ref(groupCode + ".description"),
		Position:       1,
		IsActive:       true,
	}); err != nil {
		t.Fatalf("upsert product group: %v", err)
	}

	if err := env.api.Admin.SaveProduct(env.ctx, product.UpsertParams{
		WorkspaceID:    testWorkspaceID,
		ID:             productID,
		GroupCode:      utils.Ref(groupCode),
		TitleKey:       productTitleKey,
		DescriptionKey: utils.Ref(productDescriptionKey),
		ImageURL:       utils.Ref("https://example.com/product.png"),
		PeriodSeconds:  &periodSeconds,
		Position:       1,
		GlobalInterval: "UNLIMITED",
		UserInterval:   "UNLIMITED",
		AvailableFrom:  &availableFrom,
		AvailableUntil: &availableUntil,
		IsVisible:      true,
	}); err != nil {
		t.Fatalf("create product: %v", err)
	}
	// Localization creation.
	// Add product and item translations for the locales used by checkout.
	// Verify product previews resolve localized titles and descriptions.
	localizations := []product.UpsertLocalizationParams{
		{WorkspaceID: testWorkspaceID, Locale: "ru", LocalizationKey: productTitleKey, Value: "Тестовый товар"},
		{WorkspaceID: testWorkspaceID, Locale: "ru", LocalizationKey: productDescriptionKey, Value: "Описание тестового товара"},
		{WorkspaceID: testWorkspaceID, Locale: "ru", LocalizationKey: itemTitleKey, Value: "Премиум"},
		{WorkspaceID: testWorkspaceID, Locale: "ru", LocalizationKey: itemDescriptionKey, Value: "Премиум описание"},
		{WorkspaceID: testWorkspaceID, Locale: "en", LocalizationKey: productTitleKey, Value: "Test product"},
	}
	for _, localization := range localizations {
		if err := env.api.Admin.SaveLocalization(env.ctx, localization); err != nil {
			t.Fatalf("upsert localization %s/%s: %v", localization.Locale, localization.LocalizationKey, err)
		}
	}
	// Item and price setup.
	// Attach an item reward to the product and create the payable RUB price.
	// Verify fulfillment and price calculation use catalog data from the API.
	if err := env.api.Admin.AttachProductItem(env.ctx, product.AddItemParams{
		WorkspaceID: testWorkspaceID,
		ProductID:   productID,
		ItemID:      itemID,
		Quantity:    0,
	}); err == nil {
		t.Fatal("expected zero product item quantity to fail")
	}

	if err := env.api.Admin.AttachProductItem(env.ctx, product.AddItemParams{
		WorkspaceID: testWorkspaceID,
		ProductID:   productID,
		ItemID:      itemID,
		Quantity:    2,
	}); err != nil {
		t.Fatalf("add item to product: %v", err)
	}

	priceID, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
		WorkspaceID:         testWorkspaceID,
		ProductID:           productID,
		AssetCode:           "RUB",
		ListAmountMinor:     1000,
		DiscountAmountMinor: 100,
		StartsAt:            &priceStartsAt,
		EndsAt:              &priceEndsAt,
	})
	if err != nil {
		t.Fatalf("create product price: %v", err)
	}
	if priceID == 0 {
		t.Fatal("expected created price id")
	}

	updatedRows, err := env.api.Admin.UpdateCatalogPrice(env.ctx, product.UpdatePriceParams{
		ID:                  priceID,
		WorkspaceID:         testWorkspaceID,
		AssetCode:           "RUB",
		ListAmountMinor:     1100,
		DiscountAmountMinor: 200,
		StartsAt:            &priceStartsAt,
		EndsAt:              &priceEndsAt,
	})
	if err != nil {
		t.Fatalf("update product price: %v", err)
	}
	if updatedRows != 1 {
		t.Fatalf("unexpected updated price rows: %d", updatedRows)
	}
	// Regular payment flow.
	// Create an order, payment attempt, provider event, and complete fulfillment.
	// Verify a normal purchase becomes fulfilled and completion is idempotent.
	item, err := env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 1001, 1, "buyer-regular"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if item.Price.PayableAmountMinor != 900 {
		t.Fatalf("unexpected payable amount: %d", item.Price.PayableAmountMinor)
	}
	if len(item.Items) != 1 || item.Items[0].Quantity != 2 {
		t.Fatalf("unexpected product items: %#v", item.Items)
	}

	products, err := env.api.User.ListProducts(env.ctx, user.ListProductsParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 1001, 1, "buyer-regular"),
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("list user products: %v", err)
	}
	var listedProduct *user.ProductModel
	for index := range products {
		if products[index].ID == productID {
			listedProduct = &products[index]
			break
		}
	}
	if listedProduct == nil {
		t.Fatalf("created product %q is missing from user catalog", productID)
	}
	if listedProduct.Title != "Тестовый товар" || listedProduct.Price.PayableAmountMinor != 900 {
		t.Fatalf("unexpected listed product: %#v", listedProduct)
	}
	if len(listedProduct.Items) != 1 || listedProduct.Items[0].Quantity != 2 {
		t.Fatalf("unexpected listed product items: %#v", listedProduct.Items)
	}
	groupProducts, err := env.api.User.ListProducts(env.ctx, user.ListProductsParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 1001, 1, "buyer-regular"),
		GroupCode: groupCode,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("list user products by group: %v", err)
	}
	if len(groupProducts) != 1 || groupProducts[0].ID != productID {
		t.Fatalf("unexpected grouped products: %#v", groupProducts)
	}
	missingGroupProducts, err := env.api.User.ListProducts(env.ctx, user.ListProductsParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 1001, 1, "buyer-regular"),
		GroupCode: "missing_group",
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("list user products by missing group: %v", err)
	}
	if len(missingGroupProducts) != 0 {
		t.Fatalf("missing group products = %#v", missingGroupProducts)
	}

	internalUserID := int64(501)
	order, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
		Identity:       paymentTestIdentity(testWorkspaceID, 1001, 1, "buyer-regular"),
		InternalUserID: &internalUserID,
		ProductID:      productID,
		AssetCode:      "RUB",
		Locale:         "ru",
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.Status != "draft" {
		t.Fatalf("unexpected order status: %s", order.Status)
	}

	providerPaymentID := uniquePaymentID("regular")
	attempt, err := env.api.User.CreateAttempt(env.ctx, checkout.CreateAttemptParams{
		OrderID:           order.ID,
		ProviderCode:      "yookassa",
		ProviderPaymentID: &providerPaymentID,
	})
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	if attempt.AmountMinor != order.PayableAmountMinor {
		t.Fatalf("attempt amount mismatch: got %d want %d", attempt.AmountMinor, order.PayableAmountMinor)
	}

	eventID := fmt.Sprintf("evt_%s", providerPaymentID)
	if _, err := env.api.Operational.CreateEvent(env.ctx, checkout.CreateEventParams{
		ProviderCode:      "yookassa",
		AttemptID:         utils.Ref(int64(attempt.ID)),
		OrderID:           utils.Ref(int64(order.ID)),
		ProviderEventID:   &eventID,
		ProviderPaymentID: &providerPaymentID,
		EventType:         "succeeded",
		EventStatus:       utils.Ref("succeeded"),
		PayloadHash:       sha256Hex(providerPaymentID),
		SignatureValid:    utils.Ref(true),
	}); err != nil {
		t.Fatalf("create event: %v", err)
	}

	completed, err := env.api.Operational.CompleteAttempt(env.ctx, checkout.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      "yookassa",
		ProviderPaymentID: &providerPaymentID,
		AmountMinor:       attempt.AmountMinor,
		AssetCode:         "RUB",
	})
	if err != nil {
		t.Fatalf("complete attempt: %v", err)
	}
	if completed.FulfillmentID == nil {
		t.Fatal("expected fulfillment id")
	}

	assertOrderStatus(t, env.ctx, env.db, order.ID, "fulfilled")
	assertAttemptStatus(t, env.ctx, env.db, attempt.ID, "succeeded")
	assertFulfillmentItemCount(t, env.ctx, env.db, *completed.FulfillmentID, 2)
	assertCallbackEvent(t, env.ctx, env.db, CallbackEventPaymentOrderFulfilled, order.ID, 1)
	assertOnCallbackSuccessful(t, env, order, attempt, *completed.FulfillmentID, providerPaymentID, itemID)
	assertAdminPaymentReadMethods(t, env, productID, order.ID, attempt.ID)
	assertPaymentPurchaseStats(t, env, productID, 1, 1, 1, 900)

	again, err := env.api.Operational.CompleteAttempt(env.ctx, checkout.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      "yookassa",
		ProviderPaymentID: &providerPaymentID,
		AmountMinor:       attempt.AmountMinor,
		AssetCode:         "RUB",
	})
	if err != nil {
		t.Fatalf("complete attempt again: %v", err)
	}
	if !again.AlreadyDone {
		t.Fatal("expected second completion to be idempotent")
	}
	assertCallbackEvent(t, env.ctx, env.db, CallbackEventPaymentOrderFulfilled, order.ID, 1)
	assertCallbackStatus(t, env.ctx, env.db, CallbackEventPaymentOrderFulfilled, order.ID, "ok")
	assertPaymentPurchaseStats(t, env, productID, 1, 1, 1, 900)
	// Gift payment flow.
	// Create a hidden recipient key and let another user pay for that product.
	// Verify recipient privacy, payer tracking, key usage, and gift fulfillment.
	recipientInternalID := int64(701)
	key, err := env.api.Admin.CreateProductKey(env.ctx, product.CreateKeyParams{
		WorkspaceID:    testWorkspaceID,
		AppID:          2002,
		PlatformID:     1,
		PlatformUserID: "recipient-hidden",
		InternalUserID: &recipientInternalID,
		ProductID:      productID,
		MaxUses:        1,
	})
	if err != nil {
		t.Fatalf("create purchase key: %v", err)
	}

	giftProduct, err := env.api.User.GetProductByKey(env.ctx, product.GetByKeyParams{
		Key:       key,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("get product by key: %v", err)
	}
	if giftProduct.ID != productID {
		t.Fatalf("unexpected keyed product: %s", giftProduct.ID)
	}
	if giftProduct.Price.AssetCode != "RUB" {
		t.Fatalf("unexpected keyed product asset: %s", giftProduct.Price.AssetCode)
	}

	payerPlatformID := int64(1)
	payerPlatformUserID := "payer-visible"
	payerInternalID := int64(702)
	giftOrder, err := env.api.User.CreateOrderByKey(env.ctx, checkout.CreateOrderByKeyParams{
		Key: key,
		Payer: &user.Actor{
			PlatformID:     payerPlatformID,
			PlatformUserID: payerPlatformUserID,
			InternalUserID: &payerInternalID,
		},
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("create order by key: %v", err)
	}
	if giftOrder.PlatformUserID != "recipient-hidden" {
		t.Fatalf("expected hidden recipient on order, got %s", giftOrder.PlatformUserID)
	}
	if giftOrder.PayerPlatformUserID == nil || *giftOrder.PayerPlatformUserID != payerPlatformUserID {
		t.Fatalf("expected payer on order, got %#v", giftOrder.PayerPlatformUserID)
	}

	giftProviderPaymentID := uniquePaymentID("gift")
	giftAttempt, err := env.api.User.CreateAttempt(env.ctx, checkout.CreateAttemptParams{
		OrderID:           giftOrder.ID,
		ProviderCode:      "platega",
		ProviderPaymentID: &giftProviderPaymentID,
	})
	if err != nil {
		t.Fatalf("create gift attempt: %v", err)
	}

	giftCompleted, err := env.api.Operational.CompleteAttempt(env.ctx, checkout.CompleteAttemptParams{
		AttemptID:         giftAttempt.ID,
		ProviderCode:      "platega",
		ProviderPaymentID: &giftProviderPaymentID,
		AmountMinor:       giftAttempt.AmountMinor,
		AssetCode:         "RUB",
	})
	if err != nil {
		t.Fatalf("complete gift attempt: %v", err)
	}
	if giftCompleted.FulfillmentID == nil {
		t.Fatal("expected gift fulfillment id")
	}

	assertOrderStatus(t, env.ctx, env.db, giftOrder.ID, "fulfilled")
	assertPurchaseKeyUsed(t, env.ctx, env.db, key)
	assertFulfillmentItemCount(t, env.ctx, env.db, *giftCompleted.FulfillmentID, 2)
	assertPaymentPurchaseStats(t, env, productID, 2, 2, 2, 1800)

	for _, status := range []string{"canceled", "expired", "chargebacked", "failed", "refunded"} {
		updated, err := env.api.Admin.UpdateOrderStatus(env.ctx, testWorkspaceID, giftOrder.ID, status)
		if err != nil {
			t.Fatalf("update order to %s for daily stats: %v", status, err)
		}
		if updated != 1 {
			t.Fatalf("expected one order updated to %s, got %d", status, updated)
		}
	}
	overview, err := env.api.Admin.ListDailyOverview(
		env.ctx, testWorkspaceID, now.Add(-24*time.Hour), now.Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("list complete payment daily overview: %v", err)
	}
	if len(overview) != 1 {
		t.Fatalf("unexpected complete payment daily overview: %#v", overview)
	}
	today := overview[0]
	if today.OrdersCreated != 2 ||
		today.DraftOrders != 2 ||
		today.PendingPaymentOrders != 2 ||
		today.PaidOrders != 2 ||
		today.FulfilledOrders != 2 ||
		today.CanceledOrders != 1 ||
		today.ExpiredOrders != 1 ||
		today.RefundedOrders != 1 ||
		today.ChargebackedOrders != 1 ||
		today.FailedOrders != 1 {
		t.Fatalf("daily overview does not contain every order status: %#v", today)
	}
}

func TestPaymentImportExportCycle(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	sourceWorkspace := "00000000-0000-0000-0000-000000000101"
	targetWorkspace := "00000000-0000-0000-0000-000000000102"
	groupCode := "export_group_" + suffix
	productID := "export_product_" + suffix
	itemID := "export_item_" + suffix
	productTitleKey := productID + ".title"
	productDescriptionKey := productID + ".description"
	itemTitleKey := itemID + ".title"
	itemDescriptionKey := itemID + ".description"
	now := time.Now()
	availableFrom := now.Add(-time.Hour)
	availableUntil := now.Add(time.Hour)
	priceStartsAt := now.Add(-time.Hour)
	priceEndsAt := now.Add(time.Hour)
	walletAddress := "UQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJKZ"
	walletConfigURL := "https://example.com/payment-ton.config.json"
	expectedWalletAddress, err := paymentton.NormalizeWalletAddress(walletAddress, paymentton.NetworkMainnet)
	if err != nil {
		t.Fatalf("normalize wallet: %v", err)
	}

	if err := env.api.Admin.SaveProductGroup(env.ctx, product.UpsertGroupParams{
		WorkspaceID: sourceWorkspace, Code: groupCode, TitleKey: utils.Ref(groupCode + ".title"),
		DescriptionKey: utils.Ref(groupCode + ".description"), Position: 1, IsActive: true,
	}); err != nil {
		t.Fatalf("upsert product group: %v", err)
	}
	if err := env.api.Admin.SaveProduct(env.ctx, product.UpsertParams{
		WorkspaceID: sourceWorkspace, ID: productID, GroupCode: utils.Ref(groupCode),
		TitleKey: productTitleKey, DescriptionKey: utils.Ref(productDescriptionKey),
		QuantityMode: "fixed", Position: 1, GlobalInterval: "UNLIMITED", UserInterval: "UNLIMITED",
		AvailableFrom: &availableFrom, AvailableUntil: &availableUntil, IsVisible: true,
	}); err != nil {
		t.Fatalf("upsert product: %v", err)
	}
	for _, localization := range []product.UpsertLocalizationParams{
		{WorkspaceID: sourceWorkspace, Locale: "ru", LocalizationKey: groupCode + ".title", Value: "Группа"},
		{WorkspaceID: sourceWorkspace, Locale: "ru", LocalizationKey: groupCode + ".description", Value: "Описание группы"},
		{WorkspaceID: sourceWorkspace, Locale: "ru", LocalizationKey: productTitleKey, Value: "Товар"},
		{WorkspaceID: sourceWorkspace, Locale: "ru", LocalizationKey: productDescriptionKey, Value: "Описание товара"},
		{WorkspaceID: sourceWorkspace, Locale: "ru", LocalizationKey: itemTitleKey, Value: "Награда"},
		{WorkspaceID: sourceWorkspace, Locale: "ru", LocalizationKey: itemDescriptionKey, Value: "Описание награды"},
	} {
		if err := env.api.Admin.SaveLocalization(env.ctx, localization); err != nil {
			t.Fatalf("upsert localization: %v", err)
		}
	}
	if err := env.api.Admin.AttachProductItem(env.ctx, product.AddItemParams{
		WorkspaceID: sourceWorkspace, ProductID: productID, ItemID: itemID, Quantity: 25, Scale: 2,
	}); err != nil {
		t.Fatalf("attach product item: %v", err)
	}
	if _, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
		WorkspaceID: sourceWorkspace, ProductID: productID, AssetCode: "RUB",
		ListAmountMinor: 1000, DiscountAmountMinor: 100, StartsAt: &priceStartsAt, EndsAt: &priceEndsAt,
	}); err != nil {
		t.Fatalf("create price: %v", err)
	}
	if err := env.api.Admin.SaveTONWallet(env.ctx, admin.TONWalletUpsertParams{
		WorkspaceID:      sourceWorkspace,
		Network:          paymentton.NetworkMainnet,
		WalletAddress:    walletAddress,
		NetworkConfigURL: &walletConfigURL,
		IsEnabled:        true,
	}); err != nil {
		t.Fatalf("save ton wallet: %v", err)
	}

	pkg, err := env.api.Admin.Export(env.ctx, sourceWorkspace, admin.ExportRequest{})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(pkg.TONWallets) != 1 || pkg.TONWallets[0].WalletAddress != expectedWalletAddress {
		t.Fatalf("unexpected exported ton wallets: %+v", pkg.TONWallets)
	}
	if _, err := env.api.Admin.Import(env.ctx, targetWorkspace, admin.ImportRequest{
		Package: pkg, ConflictStrategy: repository.ImportConflictUpdate,
	}); err != nil {
		t.Fatalf("import: %v", err)
	}
	imported, err := env.api.Admin.Export(env.ctx, targetWorkspace, admin.ExportRequest{})
	if err != nil {
		t.Fatalf("export imported: %v", err)
	}
	if len(imported.Groups) != 1 || len(imported.Groups[0].Products) != 1 ||
		len(imported.Groups[0].Products[0].Items) != 1 || len(imported.Groups[0].Products[0].Prices) != 1 ||
		imported.Groups[0].Products[0].Items[0].Scale != 2 || len(imported.TONWallets) != 1 {
		t.Fatalf("unexpected imported package: %+v", imported)
	}
	importedWallet, err := env.api.Admin.GetTONWallet(env.ctx, targetWorkspace)
	if err != nil {
		t.Fatalf("get imported ton wallet: %v", err)
	}
	if importedWallet.Network != paymentton.NetworkMainnet || importedWallet.WalletAddress != expectedWalletAddress ||
		!importedWallet.NetworkConfigUrl.Valid || importedWallet.NetworkConfigUrl.String != walletConfigURL || !importedWallet.IsEnabled {
		t.Fatalf("unexpected imported ton wallet: %+v", importedWallet)
	}
}

func setupPaymentIntegrationTest(t testing.TB) paymentTestEnv {
	return setupPaymentIntegrationTestWithOptions(t, paymentTestOptions())
}

func setupPaymentIntegrationTestWithOptions(t testing.TB, options Options) paymentTestEnv {
	t.Helper()

	dsn := paymentTestDSN(t)
	dbName := paymentTestDB
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	adminDB, err := openMySQL(dsn, "")
	if err != nil {
		t.Fatalf("open admin mysql connection: %v", err)
	}
	t.Cleanup(func() { adminDB.Close() })

	if err := recreatePaymentTestDatabase(ctx, adminDB, dbName); err != nil {
		t.Fatalf("recreate database %s: %v", dbName, err)
	}

	appDB, err := openMySQL(dsn, dbName)
	if err != nil {
		t.Fatalf("open payment mysql connection: %v", err)
	}
	t.Cleanup(func() { appDB.Close() })

	client, err := sqlwrap.New(appDB, paymentTestSQLOptions())
	if err != nil {
		t.Fatalf("create sql client: %v", err)
	}

	repo := repository.NewPaymentRepository(client)
	if err := repo.Bootstrap(ctx, filepath.Join("sqlc", "schema.sql")); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	api, err := NewWithDatabase(ctx, appDB, options)
	if err != nil {
		t.Fatalf("create payment service: %v", err)
	}
	t.Cleanup(func() { _ = api.Close() })

	return paymentTestEnv{
		ctx:    ctx,
		db:     appDB,
		client: client,
		api:    api,
	}
}

func setupExistingPaymentIntegrationTest(t testing.TB) paymentTestEnv {
	t.Helper()

	dsn := paymentTestDSN(t)
	dbName := paymentTestDB
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	appDB, err := openMySQL(dsn, dbName)
	if err != nil {
		t.Fatalf("open existing payment mysql connection: %v", err)
	}
	t.Cleanup(func() { appDB.Close() })

	client, err := sqlwrap.New(appDB, paymentTestSQLOptions())
	if err != nil {
		t.Fatalf("create sql client: %v", err)
	}
	api, err := NewWithDatabase(ctx, appDB, paymentTestOptions())
	if err != nil {
		t.Fatalf("create payment service: %v", err)
	}
	t.Cleanup(func() { _ = api.Close() })

	return paymentTestEnv{
		ctx:    ctx,
		db:     appDB,
		client: client,
		api:    api,
	}
}

func paymentTestSQLOptions() sqlwrap.Options {
	return sqlwrap.Options{
		CacheEnabled:  true,
		CacheSize:     100_000,
		CacheTTLCheck: time.Minute,
	}
}

func paymentTestOptions() Options {
	return Options{
		CacheEnabled:        true,
		CacheSize:           100_000,
		CacheTTLCheck:       time.Minute,
		CacheL1Delay:        time.Minute,
		DisablePriceUpdater: true,
	}
}

func assertOrderStatus(t *testing.T, ctx context.Context, db *sql.DB, orderID uint64, want string) {
	t.Helper()
	var got string
	if err := db.QueryRowContext(ctx, "SELECT status FROM payment_order WHERE id = $1", orderID).Scan(&got); err != nil {
		t.Fatalf("select order status: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected order status: got %s want %s", got, want)
	}
}

func assertAttemptStatus(t *testing.T, ctx context.Context, db *sql.DB, attemptID uint64, want string) {
	t.Helper()
	var got string
	if err := db.QueryRowContext(ctx, "SELECT status FROM payment_attempt WHERE id = $1", attemptID).Scan(&got); err != nil {
		t.Fatalf("select attempt status: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected attempt status: got %s want %s", got, want)
	}
}

func assertFulfillmentItemCount(t *testing.T, ctx context.Context, db *sql.DB, fulfillmentID uint64, want int) {
	t.Helper()
	var got int
	if err := db.QueryRowContext(ctx, "SELECT COALESCE(SUM(quantity), 0) FROM payment_fulfillment_item WHERE fulfillment_id = $1", fulfillmentID).Scan(&got); err != nil {
		t.Fatalf("select fulfillment item count: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected fulfillment item count: got %d want %d", got, want)
	}
}

func assertCallbackEvent(t *testing.T, ctx context.Context, db *sql.DB, eventType string, orderID uint64, want int) {
	t.Helper()
	var got int
	var idempotencyKey string
	eventKey := fmt.Sprintf("%s:%d", eventType, orderID)
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*), COALESCE(MAX(idempotency_key), '')
FROM payment_clb_event
WHERE source_service = 'payment' AND event_type = $1 AND event_key = $2`,
		eventType, eventKey,
	).Scan(&got, &idempotencyKey); err != nil {
		t.Fatalf("select callback event: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected callback event count: got %d want %d", got, want)
	}
	if want > 0 && idempotencyKey != eventKey {
		t.Fatalf("unexpected callback idempotency key: got %s want %s", idempotencyKey, eventKey)
	}
}

func assertOnCallbackSuccessful(
	t *testing.T,
	env paymentTestEnv,
	order *checkout.Order,
	attempt *checkout.Attempt,
	fulfillmentID uint64,
	providerPaymentID string,
	itemID string,
) {
	t.Helper()
	ctx, cancel := context.WithCancel(env.ctx)
	handled := 0
	err := env.api.OnCallback(ctx, func(callback Context) error {
		handled++
		if callback.EventType != CallbackEventPaymentOrderFulfilled {
			t.Fatalf("unexpected callback event type: got %s want %s", callback.EventType, CallbackEventPaymentOrderFulfilled)
		}
		if callback.EventKey != fmt.Sprintf("%s:%d", CallbackEventPaymentOrderFulfilled, order.ID) {
			t.Fatalf("unexpected callback event key: %s", callback.EventKey)
		}
		if callback.IdempotencyKey != callback.EventKey {
			t.Fatalf("unexpected callback idempotency key: got %s want %s", callback.IdempotencyKey, callback.EventKey)
		}
		if callback.PaymentFulfilled == nil {
			t.Fatal("expected payment fulfilled callback payload")
		}
		payload := *callback.PaymentFulfilled
		if payload.OrderID != order.ID ||
			payload.AttemptID != attempt.ID ||
			payload.FulfillmentID != fulfillmentID ||
			payload.WorkspaceID != order.WorkspaceID ||
			payload.AppID != order.AppID ||
			payload.PlatformID != order.PlatformID ||
			payload.PlatformUserID != order.PlatformUserID ||
			payload.ProductID != order.ProductID ||
			payload.Quantity != order.Quantity ||
			payload.ProviderCode != attempt.ProviderCode ||
			payload.ProviderPaymentID != providerPaymentID ||
			payload.AssetCode != attempt.AssetCode ||
			payload.AmountMinor != attempt.AmountMinor {
			t.Fatalf("unexpected callback payload: %#v", payload)
		}
		if len(payload.Rewards) != 1 ||
			payload.Rewards[0].Key != itemID ||
			payload.Rewards[0].Type != "quantity" ||
			payload.Rewards[0].Quantity != 2*int64(order.Quantity) ||
			payload.Rewards[0].Unit != nil {
			t.Fatalf("unexpected callback rewards: %#v", payload.Rewards)
		}
		if err := callback.Successful(); err != nil {
			return err
		}
		cancel()
		return nil
	}, WithCallbackBatchSize(1), WithCallbackIdleDelay(time.Millisecond))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("on callback: %v", err)
	}
	if handled != 1 {
		t.Fatalf("unexpected callback handled count: got %d want 1", handled)
	}
}

func assertCallbackStatus(t *testing.T, ctx context.Context, db *sql.DB, eventType string, orderID uint64, want string) {
	t.Helper()
	var got string
	eventKey := fmt.Sprintf("%s:%d", eventType, orderID)
	if err := db.QueryRowContext(ctx, "SELECT status FROM payment_clb_event WHERE source_service = 'payment' AND event_type = $1 AND event_key = $2", eventType, eventKey).Scan(&got); err != nil {
		t.Fatalf("select callback status: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected callback status: got %s want %s", got, want)
	}
}

func assertPaymentPurchaseStats(t *testing.T, env paymentTestEnv, productID string, purchaseCount, purchaseQuantity, uniqueBuyers, grossAmount uint64) {
	t.Helper()
	stats, err := env.api.Admin.GetStats(env.ctx, testWorkspaceID)
	if err != nil {
		t.Fatalf("get payment stats: %v", err)
	}
	if stats.ProductsTotal != 1 || stats.PurchaseCount != purchaseCount ||
		stats.PurchaseQuantity != purchaseQuantity || stats.UniqueBuyers != uniqueBuyers {
		t.Fatalf("unexpected payment stats: %#v", stats)
	}
	if len(stats.Assets) != 1 || stats.Assets[0].AssetCode != "RUB" ||
		stats.Assets[0].GrossAmountMinor != grossAmount {
		t.Fatalf("unexpected payment asset stats: %#v", stats.Assets)
	}

	productStats, err := env.api.Admin.GetProductStats(env.ctx, testWorkspaceID, productID)
	if err != nil {
		t.Fatalf("get payment product stats: %v", err)
	}
	if productStats.PurchaseCount != purchaseCount || productStats.PurchaseQuantity != purchaseQuantity {
		t.Fatalf("unexpected payment product stats: %#v", productStats)
	}

	now := time.Now()
	if err := env.api.Admin.RefreshDailyStats(env.ctx, testWorkspaceID, now.Add(-24*time.Hour), now.Add(24*time.Hour)); err != nil {
		t.Fatalf("refresh payment daily stats: %v", err)
	}
	daily, err := env.api.Admin.ListDailyStats(
		env.ctx, testWorkspaceID, productID, now.Add(-24*time.Hour), now.Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("list payment daily stats: %v", err)
	}
	if len(daily) != 1 || daily[0].PurchaseCount != purchaseCount ||
		daily[0].PurchaseQuantity != purchaseQuantity || daily[0].GrossAmountMinor != grossAmount {
		t.Fatalf("unexpected payment daily stats: %#v", daily)
	}

	overview, err := env.api.Admin.ListDailyOverview(
		env.ctx, testWorkspaceID, now.Add(-24*time.Hour), now.Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("list payment daily overview: %v", err)
	}
	if len(overview) != 1 ||
		overview[0].ProductsTotal != 1 ||
		overview[0].VisibleProducts != 1 ||
		overview[0].PurchaseCount != purchaseCount ||
		overview[0].PurchaseQuantity != purchaseQuantity ||
		overview[0].UniqueBuyers != uniqueBuyers ||
		overview[0].FulfilledOrders != purchaseCount {
		t.Fatalf("unexpected payment daily overview: %#v", overview)
	}
}

func assertAdminPaymentReadMethods(t *testing.T, env paymentTestEnv, productID string, orderID uint64, attemptID uint64) {
	t.Helper()
	products, err := env.api.Admin.ListProducts(env.ctx, admin.ProductListParams{
		WorkspaceID: testWorkspaceID,
	})
	if err != nil {
		t.Fatalf("admin list products: %v", err)
	}
	if len(products) == 0 {
		t.Fatal("expected admin products")
	}

	orders, err := env.api.Admin.ListOrders(env.ctx, admin.OrderListParams{
		WorkspaceID: testWorkspaceID,
		ProductID:   productID,
	})
	if err != nil {
		t.Fatalf("admin list orders: %v", err)
	}
	if len(orders) == 0 || uint64(orders[0].ID) != orderID {
		t.Fatalf("unexpected admin orders: %#v", orders)
	}

	attempts, err := env.api.Admin.ListPaymentAttempts(env.ctx, admin.AttemptListParams{
		WorkspaceID: testWorkspaceID,
		OrderID:     orderID,
	})
	if err != nil {
		t.Fatalf("admin list attempts: %v", err)
	}
	if len(attempts) == 0 || uint64(attempts[0].ID) != attemptID {
		t.Fatalf("unexpected admin attempts: %#v", attempts)
	}

	events, err := env.api.Admin.ListPaymentEvents(env.ctx, admin.EventListParams{
		WorkspaceID: testWorkspaceID,
	})
	if err != nil {
		t.Fatalf("admin list payment events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected admin payment events")
	}

	callbacks, err := env.api.Admin.ListCallbackEvents(env.ctx, admin.CallbackEventListParams{
		WorkspaceID:   testWorkspaceID,
		SourceService: "payment",
	})
	if err != nil {
		t.Fatalf("admin list callback events: %v", err)
	}
	if len(callbacks) == 0 {
		t.Fatal("expected admin callback events")
	}
}

func assertPurchaseKeyUsed(t *testing.T, ctx context.Context, db *sql.DB, key string) {
	t.Helper()
	var status string
	var usedCount int
	if err := db.QueryRowContext(ctx, "SELECT status, used_count FROM payment_purchase_key WHERE key_hash = $1", sha256Hex(key)).Scan(&status, &usedCount); err != nil {
		t.Fatalf("select purchase key: %v", err)
	}
	if status != "used" || usedCount != 1 {
		t.Fatalf("unexpected purchase key state: status=%s used=%d", status, usedCount)
	}
}

func uniquePaymentID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func TestPaymentGlobalDynamicPricingAcrossWorkspaces(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	now := time.Now()
	secondWorkspaceID := "00000000-0000-0000-0000-000000000002"

	firstProductID := createPaymentProduct(t, env, testProductOptions{
		AssetCode:       "RUB",
		ListAmountMinor: 100,
	})
	secondProductID := createPaymentProduct(t, env, testProductOptions{
		WorkspaceID:     secondWorkspaceID,
		AssetCode:       "RUB",
		ListAmountMinor: 100,
	})
	fixedProductID := createPaymentProduct(t, env, testProductOptions{
		AssetCode:       "TON",
		ListAmountMinor: 1_000_000_000,
	})

	if _, err := env.api.Operational.UpdateAssetRate(env.ctx, operational.UpdateAssetRateParams{
		AssetCode:              "TON",
		ReferenceAssetCode:     repository.USDTAssetCode,
		ReferencePerAssetMinor: 2_000_000,
		Source:                 "integration-test",
		ObservedAt:             now,
	}); err != nil {
		t.Fatalf("seed global TON rate: %v", err)
	}

	referenceAsset := repository.USDTAssetCode
	referenceList := uint64(1_000_000)
	referenceDiscount := uint64(0)
	coefficient := "1"
	startsAt := now.Add(-time.Hour)
	endsAt := now.Add(time.Hour)
	for workspaceID, productID := range map[string]string{
		testWorkspaceID:   firstProductID,
		secondWorkspaceID: secondProductID,
	} {
		if _, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
			WorkspaceID:                  workspaceID,
			ProductID:                    productID,
			AssetCode:                    "TON",
			PricingMode:                  repository.PricingModeDynamic,
			ReferenceAssetCode:           &referenceAsset,
			ReferenceListAmountMinor:     &referenceList,
			ReferenceDiscountAmountMinor: &referenceDiscount,
			Coefficient:                  &coefficient,
			StartsAt:                     &startsAt,
			EndsAt:                       &endsAt,
		}); err != nil {
			t.Fatalf("create dynamic TON price for %s: %v", workspaceID, err)
		}
	}

	result, err := env.api.Operational.UpdateAssetRate(env.ctx, operational.UpdateAssetRateParams{
		AssetCode:              "TON",
		ReferenceAssetCode:     repository.USDTAssetCode,
		ReferencePerAssetMinor: 4_000_000,
		Source:                 "integration-test",
		ObservedAt:             now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("update global TON rate: %v", err)
	}
	if result.UpdatedPrices != 2 || result.AffectedProducts != 2 || result.AffectedWorkspaces != 2 {
		t.Fatalf("unexpected global update result: %#v", result)
	}

	for workspaceID, productID := range map[string]string{
		testWorkspaceID:   firstProductID,
		secondWorkspaceID: secondProductID,
	} {
		item, err := env.api.User.GetProduct(env.ctx, user.GetProductParams{
			Identity:  paymentTestIdentity(workspaceID, 1001, 1, "dynamic-user"),
			ProductID: productID,
			AssetCode: "TON",
			Locale:    "ru",
		})
		if err != nil {
			t.Fatalf("get dynamic product for %s: %v", workspaceID, err)
		}
		if item.Price.PayableAmountMinor != 250_000_000 {
			t.Fatalf("unexpected dynamic price for %s: %d", workspaceID, item.Price.PayableAmountMinor)
		}
	}

	fixed, err := env.api.User.GetProduct(env.ctx, user.GetProductParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 1001, 1, "fixed-user"),
		ProductID: fixedProductID,
		AssetCode: "TON",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("get fixed TON product: %v", err)
	}
	if fixed.Price.PayableAmountMinor != 1_000_000_000 {
		t.Fatalf("fixed TON price changed: %d", fixed.Price.PayableAmountMinor)
	}

	rate, err := env.api.User.GetUSDTPrice(env.ctx, user.GetUSDTPriceParams{AssetCode: "TON"})
	if err != nil {
		t.Fatalf("get global TON rate: %v", err)
	}
	if rate.USDTPerAssetMinor != 4_000_000 {
		t.Fatalf("unexpected global TON rate: %d", rate.USDTPerAssetMinor)
	}
}

func TestPaymentPriceUpdaterStopsWithService(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	if err := env.api.Close(); err != nil {
		t.Fatalf("close initial payment service: %v", err)
	}
	if _, err := env.db.ExecContext(env.ctx, "DELETE FROM payment_asset_rate"); err != nil {
		t.Fatalf("clear asset rates: %v", err)
	}

	var requests atomic.Int32
	httpClient := &http.Client{Transport: paymentTestRateRoundTrip(func(request *http.Request) (*http.Response, error) {
		requests.Add(1)
		tokenPath := request.URL.EscapedPath()
		tokenPath = tokenPath[strings.LastIndex(tokenPath, "/")+1:]
		tokenPath, _ = url.PathUnescape(tokenPath)
		addresses := strings.Split(tokenPath, ",")
		var body strings.Builder
		body.WriteByte('[')
		for index, address := range addresses {
			if index > 0 {
				body.WriteByte(',')
			}
			body.WriteString(`{"baseToken":{"address":`)
			body.WriteString(strconv.Quote(address))
			body.WriteString(`},"quoteToken":{"address":"EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c"},"priceNative":"2","priceUsd":"2","liquidity":{"usd":1000000}}`)
		}
		body.WriteByte(']')
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body.String())),
			Request:    request,
		}, nil
	})}
	service, err := NewWithDatabase(env.ctx, env.db, Options{
		PriceUpdateHTTPClient: httpClient,
		PriceUpdateInterval:   10 * time.Millisecond,
		PriceUpdateBaseURL:    "https://dex.example",
	})
	if err != nil {
		t.Fatalf("create payment service with updater: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for requests.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("price updater did not request rate")
		}
		time.Sleep(10 * time.Millisecond)
	}
	for {
		rate, rateErr := service.User.GetUSDTPrice(env.ctx, user.GetUSDTPriceParams{AssetCode: "DOGS_TON"})
		if rateErr == nil && rate.USDTPerAssetMinor == 2_000_000 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("automatic DOGS_TON rate was not stored: rate=%#v err=%v", rate, rateErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	usdtRate, err := service.User.GetUSDTPrice(env.ctx, user.GetUSDTPriceParams{AssetCode: repository.USDTAssetCode})
	if err != nil {
		t.Fatalf("get automatic USDT rate: %v", err)
	}
	if usdtRate.USDTPerAssetMinor != 1_000_000 {
		t.Fatalf("unexpected automatic USDT rate: %d", usdtRate.USDTPerAssetMinor)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("close payment service: %v", err)
	}
	requestCount := requests.Load()
	time.Sleep(50 * time.Millisecond)
	if requests.Load() != requestCount {
		t.Fatalf("price updater continued after Close: before=%d after=%d", requestCount, requests.Load())
	}
}

type paymentTestRateRoundTrip func(*http.Request) (*http.Response, error)

func (f paymentTestRateRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestPaymentStarsTopupExampleImportExport(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	req := loadPaymentImportExample(t, "stars_topup_import.json")

	preview, err := env.api.Admin.PreviewImport(env.ctx, testWorkspaceID, req.Package)
	if err != nil {
		t.Fatalf("preview import: %v", err)
	}
	if preview.Counts.Groups != 1 || preview.Counts.Products != 1 ||
		preview.Counts.ProductItems != 1 || preview.Counts.Prices != 7 || preview.Counts.Localizations != 4 {
		t.Fatalf("unexpected preview counts: %+v", preview.Counts)
	}

	result, err := env.api.Admin.Import(env.ctx, testWorkspaceID, req)
	if err != nil {
		t.Fatalf("import example: %v", err)
	}
	if result.Imported.Groups != 1 || result.Imported.Products != 1 ||
		result.Imported.ProductItems != 1 || result.Imported.Prices != 7 || result.Imported.Localizations != 8 {
		t.Fatalf("unexpected import counts: %+v", result.Imported)
	}

	exported, err := env.api.Admin.Export(env.ctx, testWorkspaceID, repository.ExportRequest{})
	if err != nil {
		t.Fatalf("export after import: %v", err)
	}
	group := findExportGroup(t, exported, "topup")
	if group.TitleKey == nil || *group.TitleKey != "payment.group.topup.title" {
		t.Fatalf("unexpected group title key: %#v", group.TitleKey)
	}
	if len(group.Products) != 1 {
		t.Fatalf("expected one product in group, got %d", len(group.Products))
	}
	product := group.Products[0]
	if product.ID != "topup.stars.flexible" {
		t.Fatalf("unexpected product id: %s", product.ID)
	}
	if product.QuantityMode != "flexible" {
		t.Fatalf("unexpected quantity mode: %s", product.QuantityMode)
	}
	if product.GlobalLimit != 0 || product.UserLimit != 0 ||
		product.GlobalInterval != "UNLIMITED" || product.UserInterval != "UNLIMITED" {
		t.Fatalf("unexpected limits: global=%d/%s user=%d/%s",
			product.GlobalLimit, product.GlobalInterval, product.UserLimit, product.UserInterval)
	}
	if len(product.Items) != 1 {
		t.Fatalf("expected one product item, got %d", len(product.Items))
	}
	if product.Items[0].ItemID != "stars" || product.Items[0].Quantity != 100 || product.Items[0].Scale != 2 {
		t.Fatalf("unexpected product item: %+v", product.Items[0])
	}

	prices := indexExportPrices(product.Prices)
	assertFixedExamplePrice(t, prices["XTR"], "XTR", 1)
	assertFixedExamplePrice(t, prices["USDT_TON"], "USDT_TON", 15000)
	for _, code := range []string{"TON", "NOT_TON", "DOGS_TON", "MAJOR_TON", "UTYA_TON"} {
		assertDynamicExamplePrice(t, prices[code], code)
	}

	exportedJSON, err := json.Marshal(exported)
	if err != nil {
		t.Fatalf("marshal exported package: %v", err)
	}
	var exportedRoot map[string]any
	if err := json.Unmarshal(exportedJSON, &exportedRoot); err != nil {
		t.Fatalf("unmarshal exported package: %v", err)
	}
	if _, ok := exportedRoot["items"]; ok {
		t.Fatalf("payment export must not expose root items, got: %#v", exportedRoot["items"])
	}
	if _, ok := exportedRoot["references"]; ok {
		t.Fatal("payment export must not expose root references")
	}
	exportedContent := string(exportedJSON)
	if strings.Contains(exportedContent, "title_key") || strings.Contains(exportedContent, "description_key") {
		t.Fatalf("payment export must not expose localization keys: %s", exportedContent)
	}

	var localItemTable *string
	if err := env.db.QueryRowContext(
		env.ctx,
		"SELECT to_regclass('payment_item')::text",
	).Scan(&localItemTable); err != nil {
		t.Fatalf("inspect local payment item table: %v", err)
	}
	if localItemTable != nil {
		t.Fatalf("payment must not own local item catalog, found table %q", *localItemTable)
	}

	order, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 7007, 2, "opaque-item-user"),
		ProductID: "topup.stars.flexible",
		Quantity:  3,
		AssetCode: "XTR",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("create order with opaque reward key: %v", err)
	}
	providerPaymentID := "opaque-item-payment"
	attempt, err := env.api.User.CreateAttempt(env.ctx, checkout.CreateAttemptParams{
		OrderID:           order.ID,
		ProviderCode:      "telegram_stars",
		ProviderPaymentID: &providerPaymentID,
	})
	if err != nil {
		t.Fatalf("create attempt with opaque reward key: %v", err)
	}
	completed, err := env.api.Operational.CompleteAttempt(env.ctx, checkout.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      attempt.ProviderCode,
		ProviderPaymentID: &providerPaymentID,
		AmountMinor:       attempt.AmountMinor,
		AssetCode:         attempt.AssetCode,
	})
	if err != nil {
		t.Fatalf("complete attempt with opaque reward key: %v", err)
	}
	if completed.FulfillmentID == nil {
		t.Fatal("expected fulfillment for opaque reward key")
	}

	var quantity int64
	var scale int16
	if err := env.db.QueryRowContext(
		env.ctx,
		"SELECT quantity, scale FROM payment_fulfillment_item WHERE fulfillment_id = $1 AND item_id = $2",
		*completed.FulfillmentID,
		"stars",
	).Scan(&quantity, &scale); err != nil {
		t.Fatalf("read opaque fulfillment reward: %v", err)
	}
	if quantity != 300 || scale != 2 {
		t.Fatalf("unexpected opaque fulfillment reward: quantity=%d scale=%d", quantity, scale)
	}
}

func loadPaymentImportExample(t *testing.T, name string) repository.ImportRequest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("examples", name))
	if err != nil {
		t.Fatalf("read payment example %s: %v", name, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw payment example %s: %v", name, err)
	}
	pkg, ok := raw["package"].(map[string]any)
	if !ok {
		t.Fatalf("payment example %s must contain package object", name)
	}
	if _, ok := pkg["items"]; ok {
		t.Fatalf("payment example %s must not contain root items", name)
	}
	if _, ok := pkg["references"]; ok {
		t.Fatalf("payment example %s must not contain root references", name)
	}
	content := string(data)
	if strings.Contains(content, "title_key") || strings.Contains(content, "description_key") {
		t.Fatalf("payment example %s must not expose localization keys", name)
	}
	var req repository.ImportRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal payment example %s: %v", name, err)
	}
	return req
}

func findExportGroup(t *testing.T, pkg repository.ExportPackage, code string) repository.ExportProductGroup {
	t.Helper()
	for _, group := range pkg.Groups {
		if group.Code == code {
			return group
		}
	}
	t.Fatalf("group %s not found in export", code)
	return repository.ExportProductGroup{}
}

func indexExportPrices(prices []repository.ExportPrice) map[string]repository.ExportPrice {
	result := make(map[string]repository.ExportPrice, len(prices))
	for _, price := range prices {
		result[price.AssetCode] = price
	}
	return result
}

func assertFixedExamplePrice(t *testing.T, price repository.ExportPrice, assetCode string, amount uint64) {
	t.Helper()
	if price.AssetCode != assetCode {
		t.Fatalf("price for %s not found", assetCode)
	}
	if price.PricingMode != "fixed" || price.ListAmountMinor != amount || price.DiscountAmountMinor != 0 {
		t.Fatalf("unexpected fixed price for %s: %+v", assetCode, price)
	}
	if price.ReferenceAssetCode != nil || price.ReferenceListAmountMinor != nil ||
		price.ReferenceDiscountAmountMinor != nil || price.Coefficient != nil {
		t.Fatalf("fixed price %s must not have reference fields: %+v", assetCode, price)
	}
}

func assertDynamicExamplePrice(t *testing.T, price repository.ExportPrice, assetCode string) {
	t.Helper()
	if price.AssetCode != assetCode {
		t.Fatalf("price for %s not found", assetCode)
	}
	if price.PricingMode != "dynamic" {
		t.Fatalf("unexpected pricing mode for %s: %+v", assetCode, price)
	}
	if price.ReferenceAssetCode == nil || *price.ReferenceAssetCode != "USDT_TON" {
		t.Fatalf("unexpected reference asset for %s: %+v", assetCode, price.ReferenceAssetCode)
	}
	if price.ReferenceListAmountMinor == nil || *price.ReferenceListAmountMinor != 15000 {
		t.Fatalf("unexpected reference list amount for %s: %+v", assetCode, price.ReferenceListAmountMinor)
	}
	if price.ReferenceDiscountAmountMinor == nil || *price.ReferenceDiscountAmountMinor != 0 {
		t.Fatalf("unexpected reference discount amount for %s: %+v", assetCode, price.ReferenceDiscountAmountMinor)
	}
	if price.Coefficient == nil || *price.Coefficient != "1.000000000000" {
		t.Fatalf("unexpected coefficient for %s: %+v", assetCode, price.Coefficient)
	}
}

func TestPlategaAdapterFullCycle(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "platega_product",
		AssetCode:       platega.AssetCode,
		ListAmountMinor: 1299,
	})

	var createPayload plategaTestCreateTransactionPayload
	var createPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MerchantId") != "merchant-1" || r.Header.Get("X-Secret") != "secret-1" {
			t.Fatalf("unexpected platega auth headers: merchant=%q secret=%q", r.Header.Get("X-MerchantId"), r.Header.Get("X-Secret"))
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/transaction/process":
			createPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&createPayload); err != nil {
				t.Fatalf("decode platega create payload: %v", err)
			}
			writeJSON(t, w, map[string]any{
				"paymentMethod":  "SBPQR",
				"transactionId":  "platega-tx-1",
				"redirect":       "https://pay.platega.io?qrsbp",
				"return":         "https://example.com/success",
				"paymentDetails": "12.99 RUB",
				"status":         "PENDING",
				"expiresIn":      "00:15:00",
				"merchantId":     "merchant-1",
				"usdtRate":       90.5,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/h2h/platega-tx-1":
			writeJSON(t, w, map[string]any{
				"amount": 12.99,
				"qr":     "https://qr.nspk.ru/test",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/transaction/platega-tx-1":
			writeJSON(t, w, map[string]any{
				"id":     "platega-tx-1",
				"status": "CONFIRMED",
				"paymentDetails": map[string]any{
					"amount":   12.99,
					"currency": "RUB",
				},
				"paymentMethod": "SBPQR",
				"payload":       createPayload.Payload,
				"description":   createPayload.Description,
			})
		default:
			t.Fatalf("unexpected platega api request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	credentials := platega.Credentials{
		MerchantID: "merchant-1",
		Secret:     "secret-1",
		APIBaseURL: server.URL,
	}

	payment, err := env.api.Adapters.Platega.CreatePayment(env.ctx, platega.CreatePaymentParams{
		Credentials:    credentials,
		WorkspaceID:    testWorkspaceID,
		AppID:          8008,
		PlatformID:     3,
		PlatformUserID: "platega-user",
		ProductID:      productID,
		Locale:         "ru",
		Description:    "Platega product",
		ReturnURL:      "https://example.com/success",
		FailedURL:      "https://example.com/fail",
		PaymentMethod:  platega.PaymentMethodSBPQR,
	})
	if err != nil {
		t.Fatalf("create platega payment: %v", err)
	}
	if createPath != "/transaction/process" {
		t.Fatalf("expected method-specific create path, got %s", createPath)
	}
	if createPayload.PaymentMethod == nil || *createPayload.PaymentMethod != int(platega.PaymentMethodSBPQR) {
		t.Fatalf("unexpected payment method payload: %#v", createPayload)
	}
	if createPayload.PaymentDetails.Amount != 12.99 || createPayload.PaymentDetails.Currency != "RUB" {
		t.Fatalf("unexpected payment details: %#v", createPayload.PaymentDetails)
	}
	if createPayload.Payload != payment.OrderPublicID {
		t.Fatalf("expected order public id as payload, got %q", createPayload.Payload)
	}
	if payment.TransactionID != "platega-tx-1" || payment.PaymentURL == "" || payment.AmountMinor != 1299 {
		t.Fatalf("unexpected platega payment response: %#v", payment)
	}
	assertOrderStatus(t, env.ctx, env.db, payment.OrderID, "pending_payment")
	assertAttemptStatus(t, env.ctx, env.db, payment.AttemptID, "pending")

	h2h, err := env.api.Adapters.Platega.GetH2H(env.ctx, platega.GetH2HParams{
		Credentials:   credentials,
		TransactionID: payment.TransactionID,
	})
	if err != nil {
		t.Fatalf("get platega h2h: %v", err)
	}
	if !strings.Contains(h2h.QR, "qr.nspk.ru") {
		t.Fatalf("unexpected h2h response: %#v", h2h)
	}

	raw := []byte(`{"id":"platega-tx-1","amount":12.99,"currency":"RUB","status":"CONFIRMED","paymentMethod":2}`)
	headers := http.Header{}
	headers.Set("X-MerchantId", "merchant-1")
	headers.Set("X-Secret", "secret-1")
	result, err := env.api.Adapters.Platega.HandleWebhook(env.ctx, platega.WebhookRequest{
		Credentials: credentials,
		Raw:         raw,
		Headers:     headers,
	})
	if err != nil {
		t.Fatalf("handle platega webhook: %v", err)
	}
	if result.OrderID != payment.OrderID || result.AttemptID != payment.AttemptID {
		t.Fatalf("unexpected platega webhook result: %#v", result)
	}
	assertOrderStatus(t, env.ctx, env.db, payment.OrderID, "fulfilled")
	assertAttemptStatus(t, env.ctx, env.db, payment.AttemptID, "succeeded")

	again, err := env.api.Adapters.Platega.SyncPayment(env.ctx, platega.SyncPaymentParams{
		Credentials:   credentials,
		TransactionID: payment.TransactionID,
	})
	if err != nil {
		t.Fatalf("sync platega payment: %v", err)
	}
	if !again.AlreadyDone {
		t.Fatalf("expected idempotent platega sync, got %#v", again)
	}
}

func TestPlategaAdapterRejectsInvalidWebhookCredentials(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	credentials := platega.Credentials{
		MerchantID: "merchant-1",
		Secret:     "secret-1",
		APIBaseURL: "https://example.com",
	}

	headers := http.Header{}
	headers.Set("X-MerchantId", "merchant-1")
	headers.Set("X-Secret", "wrong-secret")
	_, err := env.api.Adapters.Platega.HandleWebhook(env.ctx, platega.WebhookRequest{
		Credentials: credentials,
		Raw:         []byte(`{"id":"tx"}`),
		Headers:     headers,
	})
	if err == nil {
		t.Fatal("expected invalid platega webhook credentials error")
	}
}

type plategaTestCreateTransactionPayload struct {
	PaymentMethod  *int `json:"paymentMethod"`
	PaymentDetails struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	} `json:"paymentDetails"`
	Description string `json:"description"`
	ReturnURL   string `json:"return"`
	FailedURL   string `json:"failedUrl"`
	Payload     string `json:"payload"`
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json response: %v", err)
	}
}

func TestPaymentCacheVersionInvalidatesOtherNode(t *testing.T) {
	cache := newPaymentSharedTestCache()
	options := paymentTestOptions()
	options.Cache = cache
	options.CacheL1Delay = time.Minute
	options.CacheL2Delay = time.Minute

	env := setupPaymentIntegrationTestWithOptions(t, options)
	nodeB, err := NewWithDatabase(env.ctx, env.db, options)
	if err != nil {
		t.Fatalf("create second payment node: %v", err)
	}
	t.Cleanup(func() { _ = nodeB.Close() })

	now := time.Now().UTC()
	productID := "cache-version-product"
	groupCode := "cache-version-group"
	titleKey := productID + ".title"
	availableFrom := now.Add(-time.Hour)
	availableUntil := now.Add(time.Hour)

	if err := env.api.Admin.SaveProductGroup(env.ctx, product.UpsertGroupParams{
		WorkspaceID: testWorkspaceID,
		Code:        groupCode,
		IsActive:    true,
	}); err != nil {
		t.Fatalf("create cache test group: %v", err)
	}
	if err := env.api.Admin.SaveLocalization(env.ctx, product.UpsertLocalizationParams{
		WorkspaceID:     testWorkspaceID,
		Locale:          "ru",
		LocalizationKey: titleKey,
		Value:           "Old title",
	}); err != nil {
		t.Fatalf("create cache test localization: %v", err)
	}
	if err := env.api.Admin.SaveProduct(env.ctx, product.UpsertParams{
		WorkspaceID:    testWorkspaceID,
		ID:             productID,
		GroupCode:      &groupCode,
		TitleKey:       titleKey,
		QuantityMode:   "fixed",
		GlobalInterval: "UNLIMITED",
		UserLimit:      1,
		UserInterval:   "DAY",
		AvailableFrom:  &availableFrom,
		AvailableUntil: &availableUntil,
		IsVisible:      true,
	}); err != nil {
		t.Fatalf("create cache test product: %v", err)
	}
	if _, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
		WorkspaceID:     testWorkspaceID,
		ProductID:       productID,
		AssetCode:       "RUB",
		ListAmountMinor: 100,
		StartsAt:        &availableFrom,
		EndsAt:          &availableUntil,
	}); err != nil {
		t.Fatalf("create cache test price: %v", err)
	}

	warm, err := nodeB.User.GetProduct(env.ctx, paymentGetProductParams(productID, "cache-user"))
	if err != nil {
		t.Fatalf("warm product cache on node B: %v", err)
	}
	if warm.Title != "Old title" || warm.Limit.User.Limit != 1 {
		t.Fatalf("unexpected warm product: %+v", warm)
	}

	if err := env.api.Admin.SaveLocalization(env.ctx, product.UpsertLocalizationParams{
		WorkspaceID:     testWorkspaceID,
		Locale:          "ru",
		LocalizationKey: titleKey,
		Value:           "New title",
	}); err != nil {
		t.Fatalf("update cache test localization: %v", err)
	}
	if err := env.api.Admin.SaveProduct(env.ctx, product.UpsertParams{
		WorkspaceID:    testWorkspaceID,
		ID:             productID,
		GroupCode:      &groupCode,
		TitleKey:       titleKey,
		QuantityMode:   "fixed",
		GlobalInterval: "UNLIMITED",
		UserLimit:      3,
		UserInterval:   "DAY",
		AvailableFrom:  &availableFrom,
		AvailableUntil: &availableUntil,
		IsVisible:      true,
	}); err != nil {
		t.Fatalf("update cache test product: %v", err)
	}

	updated, err := nodeB.User.GetProduct(env.ctx, paymentGetProductParams(productID, "cache-user"))
	if err != nil {
		t.Fatalf("read invalidated product on node B: %v", err)
	}
	if updated.Title != "New title" || updated.Limit.User.Limit != 3 {
		t.Fatalf("node B returned stale product: %+v", updated)
	}
}

func TestPaymentImportBatchesMoreThanPostgresParameterLimit(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	const productCount = 3001

	products := make([]repository.ExportProduct, 0, productCount)
	for index := 0; index < productCount; index++ {
		productID := fmt.Sprintf("large-import-product-%04d", index)
		products = append(products, repository.ExportProduct{
			ID:             productID,
			TitleKey:       productID + ".title",
			QuantityMode:   "fixed",
			GlobalInterval: "UNLIMITED",
			UserInterval:   "UNLIMITED",
			IsVisible:      true,
		})
	}

	result, err := env.api.Admin.Import(env.ctx, "large-import-workspace", admin.ImportRequest{
		Package: admin.ExportPackage{
			Format:   repository.ExportFormat,
			Service:  "payment",
			Products: products,
		},
		ConflictStrategy: repository.ImportConflictUpdate,
	})
	if err != nil {
		t.Fatalf("import package larger than PostgreSQL parameter limit: %v", err)
	}
	if result.Imported.Products != productCount {
		t.Fatalf("imported products = %d, want %d", result.Imported.Products, productCount)
	}

	var stored int
	if err := env.db.QueryRowContext(
		env.ctx,
		"SELECT COUNT(*) FROM payment_product WHERE workspace_id = $1",
		"large-import-workspace",
	).Scan(&stored); err != nil {
		t.Fatalf("count imported products: %v", err)
	}
	if stored != productCount {
		t.Fatalf("stored products = %d, want %d", stored, productCount)
	}
}

func TestPaymentImportSerializesWithAdminWrite(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	workspaceID := "concurrent-import-workspace"

	transaction, err := env.db.BeginTx(env.ctx, nil)
	if err != nil {
		t.Fatalf("begin competing transaction: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback() })
	if _, err := transaction.ExecContext(
		env.ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		"payment:"+workspaceID,
	); err != nil {
		t.Fatalf("lock payment workspace: %v", err)
	}

	importResult := make(chan error, 1)
	go func() {
		_, err := env.api.Admin.Import(env.ctx, workspaceID, admin.ImportRequest{
			Package: admin.ExportPackage{
				Format:  repository.ExportFormat,
				Service: "payment",
				Products: []repository.ExportProduct{
					paymentImportTestProduct("concurrent-import-product"),
				},
			},
			ConflictStrategy: repository.ImportConflictUpdate,
		})
		importResult <- err
	}()

	waitForPaymentWorkspaceLock(t, env, 1)

	adminResult := make(chan error, 1)
	go func() {
		adminResult <- env.api.Admin.SaveProduct(env.ctx, product.UpsertParams{
			WorkspaceID:    workspaceID,
			ID:             "concurrent-admin-product",
			TitleKey:       "concurrent-admin-product.title",
			QuantityMode:   "fixed",
			GlobalInterval: "UNLIMITED",
			UserInterval:   "UNLIMITED",
			AvailableFrom:  timePointer(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			AvailableUntil: timePointer(time.Date(2124, 1, 1, 0, 0, 0, 0, time.UTC)),
			IsVisible:      true,
		})
	}()

	waitForPaymentWorkspaceLock(t, env, 2)

	if err := transaction.Commit(); err != nil {
		t.Fatalf("release payment workspace lock: %v", err)
	}
	if err := <-importResult; err != nil {
		t.Fatalf("concurrent import: %v", err)
	}
	if err := <-adminResult; err != nil {
		t.Fatalf("concurrent admin write: %v", err)
	}

	var stored int
	if err := env.db.QueryRowContext(
		env.ctx,
		`SELECT COUNT(*)
FROM payment_product
WHERE workspace_id = $1
  AND id IN ('concurrent-import-product', 'concurrent-admin-product')`,
		workspaceID,
	).Scan(&stored); err != nil {
		t.Fatalf("count concurrent payment products: %v", err)
	}
	if stored != 2 {
		t.Fatalf("concurrent operations stored %d products, want 2", stored)
	}
}

func waitForPaymentWorkspaceLock(t *testing.T, env paymentTestEnv, minimum int) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting int
		err := env.db.QueryRowContext(env.ctx, `
SELECT COUNT(*)
FROM pg_stat_activity
WHERE datname = current_database()
  AND wait_event_type = 'Lock'
  AND query LIKE '%pg_advisory_xact_lock%'`).Scan(&waiting)
		if err != nil {
			t.Fatalf("inspect payment workspace lock waiters: %v", err)
		}
		if waiting >= minimum {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("payment workspace lock waiters = %d, want at least %d", waiting, minimum)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func paymentImportTestProduct(id string) repository.ExportProduct {
	return repository.ExportProduct{
		ID:             id,
		TitleKey:       id + ".title",
		QuantityMode:   "fixed",
		GlobalInterval: "UNLIMITED",
		UserInterval:   "UNLIMITED",
		IsVisible:      true,
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func paymentGetProductParams(productID string, platformUserID string) product.GetParams {
	return product.GetParams{
		Identity: services.Identity{
			WorkspaceID:    testWorkspaceID,
			AppID:          1,
			PlatformID:     1,
			PlatformUserID: platformUserID,
		},
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	}
}

type paymentSharedTestCacheEntry struct {
	value     []byte
	expiresAt time.Time
}

type paymentSharedTestCache struct {
	mu      sync.Mutex
	entries map[string]paymentSharedTestCacheEntry
}

func newPaymentSharedTestCache() *paymentSharedTestCache {
	return &paymentSharedTestCache{
		entries: make(map[string]paymentSharedTestCacheEntry),
	}
}

func (c *paymentSharedTestCache) GetWithTTL(key string) ([]byte, time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, 0, nil
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		delete(c.entries, key)
		return nil, 0, nil
	}

	return append([]byte(nil), entry.value...), time.Until(entry.expiresAt), nil
}

func (c *paymentSharedTestCache) Set(key string, value []byte, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := paymentSharedTestCacheEntry{
		value: append([]byte(nil), value...),
	}
	if expiration > 0 {
		entry.expiresAt = time.Now().Add(expiration)
	}
	c.entries[key] = entry

	return nil
}

func (c *paymentSharedTestCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)

	return nil
}

func (c *paymentSharedTestCache) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.entries)

	return nil
}

func (c *paymentSharedTestCache) Close() error {
	return nil
}

var _ Storage = (*paymentSharedTestCache)(nil)

type testProductOptions struct {
	ProductID           string
	WorkspaceID         string
	AssetCode           string
	ListAmountMinor     uint64
	DiscountAmountMinor uint64
	GlobalLimit         int32
	GlobalInterval      string
	GlobalIntervalCount int32
	UserLimit           int32
	UserInterval        string
	UserIntervalCount   int32
	AvailableFrom       time.Time
	AvailableUntil      time.Time
	IsVisible           bool
	IsHidden            bool
	IsClosed            bool
}

func TestPaymentWorkspaceIsolation(t *testing.T) {
	// Test database setup.
	// Create the MySQL database connection and bootstrap the payment schema.
	// Verify workspace separates catalogs without binding products to AppID.
	env := setupPaymentIntegrationTest(t)
	workspaceA := fmt.Sprintf("00000000-0000-0000-0000-%012d", time.Now().UnixNano()%1_000_000_000_000)
	workspaceB := fmt.Sprintf("11111111-1111-1111-1111-%012d", time.Now().UnixNano()%1_000_000_000_000)

	// Workspace catalog isolation.
	// Create a product only inside workspace A and fetch it from two workspaces.
	// Verify the same AppID cannot see workspace A catalog entries through workspace B.
	productID := createPaymentProduct(t, env, testProductOptions{
		WorkspaceID:     workspaceA,
		AssetCode:       "RUB",
		ListAmountMinor: 1000,
	})
	if _, err := env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(workspaceA, 4500, 1, "workspace-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	}); err != nil {
		t.Fatalf("expected workspace A product to be visible, got %v", err)
	}
	if _, err := env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(workspaceB, 4500, 1, "workspace-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected workspace B lookup to be isolated, got %v", err)
	}

	sameProductID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       productID,
		WorkspaceID:     workspaceB,
		AssetCode:       "RUB",
		ListAmountMinor: 2500,
	})
	if sameProductID != productID {
		t.Fatalf("expected duplicate product id across workspaces, got %s", sameProductID)
	}
	workspaceBProduct, err := env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(workspaceB, 4500, 1, "workspace-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("expected workspace B product with duplicate id to be visible, got %v", err)
	}
	if workspaceBProduct.Price.ListAmountMinor != 2500 {
		t.Fatalf("expected workspace B duplicate product price 2500, got %d", workspaceBProduct.Price.ListAmountMinor)
	}

	// Workspace checkout isolation.
	// Create orders for the same product id after both workspaces define it.
	// Verify checkout resolves catalog data by workspace instead of global product ids.
	if _, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
		Identity:  paymentTestIdentity(workspaceA, 4500, 1, "workspace-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	}); err != nil {
		t.Fatalf("expected workspace A order to be created, got %v", err)
	}
	if _, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
		Identity:  paymentTestIdentity(workspaceB, 4500, 1, "workspace-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	}); err != nil {
		t.Fatalf("expected workspace B order for its duplicate product to be created, got %v", err)
	}
}

func TestPaymentLimitsAcrossAllIntervals(t *testing.T) {
	// Test database setup.
	// Create the MySQL database connection and bootstrap the payment schema.
	// Verify limit checks run against real persisted paid orders.
	env := setupPaymentIntegrationTest(t)

	intervals := []string{"SECOND", "MINUTE", "HOUR", "DAY", "WEEK", "MONTH", "ONCE"}
	for _, interval := range intervals {
		t.Run("user_"+interval, func(t *testing.T) {
			// User purchase limit.
			// Complete one purchase for a user-limited product in the selected interval.
			// Verify the same user is blocked while another user can still buy it.
			productID := createPaymentProduct(t, env, testProductOptions{
				AssetCode:           "RUB",
				ListAmountMinor:     1000,
				DiscountAmountMinor: 100,
				UserLimit:           1,
				UserInterval:        interval,
				UserIntervalCount:   1,
			})

			completeTestPayment(t, env, productID, "limit-user-"+interval, "buyer-a", "yookassa", "RUB")

			if _, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
				Identity:  paymentTestIdentity(testWorkspaceID, 4100, 1, "buyer-a"),
				ProductID: productID,
				AssetCode: "RUB",
				Locale:    "ru",
			}); !errors.Is(err, repository.ErrProductLocked) {
				t.Fatalf("expected user limit lock, got %v", err)
			}

			if _, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
				Identity:  paymentTestIdentity(testWorkspaceID, 4101, 1, "buyer-a"),
				ProductID: productID,
				AssetCode: "RUB",
				Locale:    "ru",
			}); !errors.Is(err, repository.ErrProductLocked) {
				t.Fatalf("expected user limit lock across AppID, got %v", err)
			}

			if _, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
				Identity:  paymentTestIdentity(testWorkspaceID, 4100, 1, "buyer-b"),
				ProductID: productID,
				AssetCode: "RUB",
				Locale:    "ru",
			}); err != nil {
				t.Fatalf("expected another user to bypass user limit, got %v", err)
			}
		})

		t.Run("global_"+interval, func(t *testing.T) {
			// Global purchase limit.
			// Complete one purchase for a globally limited product in the selected interval.
			// Verify every user is blocked after the global quota is consumed.
			productID := createPaymentProduct(t, env, testProductOptions{
				AssetCode:           "RUB",
				ListAmountMinor:     1000,
				GlobalLimit:         1,
				GlobalInterval:      interval,
				GlobalIntervalCount: 1,
				UserInterval:        "UNLIMITED",
				DiscountAmountMinor: 0,
			})

			completeTestPayment(t, env, productID, "limit-global-"+interval, "buyer-a", "yookassa", "RUB")

			if _, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
				Identity:  paymentTestIdentity(testWorkspaceID, 4101, 1, "buyer-b"),
				ProductID: productID,
				AssetCode: "RUB",
				Locale:    "ru",
			}); !errors.Is(err, repository.ErrProductLocked) {
				t.Fatalf("expected global limit lock, got %v", err)
			}
		})
	}

	t.Run("unlimited", func(t *testing.T) {
		// Unlimited purchase interval.
		// Complete one purchase for a product without global or user limits.
		// Verify the same user can create another order after a completed purchase.
		productID := createPaymentProduct(t, env, testProductOptions{
			AssetCode:       "RUB",
			ListAmountMinor: 1000,
			UserInterval:    "UNLIMITED",
			GlobalInterval:  "UNLIMITED",
		})

		completeTestPayment(t, env, productID, "limit-unlimited", "buyer-a", "yookassa", "RUB")

		if _, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
			Identity:  paymentTestIdentity(testWorkspaceID, 4100, 1, "buyer-a"),
			ProductID: productID,
			AssetCode: "RUB",
			Locale:    "ru",
		}); err != nil {
			t.Fatalf("expected unlimited product to remain available, got %v", err)
		}
	})
}

func TestPaymentCatalogAvailabilityAndPriceSafety(t *testing.T) {
	// Test database setup.
	// Create the MySQL database connection and bootstrap the payment schema.
	// Verify catalog visibility, time windows, localization fallback, and prices.
	env := setupPaymentIntegrationTest(t)

	// Product availability guards.
	// Create unavailable products for every visibility and availability state.
	// Verify hidden, closed, future, and expired products cannot be fetched.
	now := time.Now()
	unavailable := []struct {
		name string
		opt  testProductOptions
	}{
		{name: "hidden", opt: testProductOptions{IsHidden: true}},
		{name: "closed", opt: testProductOptions{IsVisible: true, IsClosed: true}},
		{name: "future", opt: testProductOptions{IsVisible: true, AvailableFrom: now.Add(time.Hour), AvailableUntil: now.Add(2 * time.Hour)}},
		{name: "expired", opt: testProductOptions{IsVisible: true, AvailableFrom: now.Add(-2 * time.Hour), AvailableUntil: now.Add(-time.Hour)}},
	}
	for _, tc := range unavailable {
		productID := createPaymentProduct(t, env, mergeProductOptions(testProductOptions{
			AssetCode:       "RUB",
			ListAmountMinor: 1000,
		}, tc.opt))
		if _, err := env.api.User.GetProduct(env.ctx, product.GetParams{
			Identity:  paymentTestIdentity(testWorkspaceID, 4200, 1, "buyer"),
			ProductID: productID,
			AssetCode: "RUB",
			Locale:    "ru",
		}); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("%s product should be unavailable, got %v", tc.name, err)
		}
	}

	// Price selection and localization fallback.
	// Create active regular, active promo, and expired prices for one product.
	// Verify the current promo price wins and missing locale falls back to keys.
	productID := createPaymentProduct(t, env, testProductOptions{
		AssetCode:       "RUB",
		ListAmountMinor: 1000,
	})
	priceStartsAt := now.Add(-time.Hour)
	priceEndsAt := now.Add(time.Hour)
	expiredStart := now.Add(-3 * time.Hour)
	expiredEnd := now.Add(-2 * time.Hour)
	if _, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
		WorkspaceID:         testWorkspaceID,
		ProductID:           productID,
		AssetCode:           "RUB",
		ListAmountMinor:     2000,
		DiscountAmountMinor: 0,
		StartsAt:            &expiredStart,
		EndsAt:              &expiredEnd,
	}); err != nil {
		t.Fatalf("create expired price: %v", err)
	}
	if _, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
		WorkspaceID:         testWorkspaceID,
		ProductID:           productID,
		AssetCode:           "RUB",
		ListAmountMinor:     900,
		DiscountAmountMinor: 200,
		IsPromotion:         true,
		StartsAt:            &priceStartsAt,
		EndsAt:              &priceEndsAt,
	}); err != nil {
		t.Fatalf("create promotion price: %v", err)
	}

	item, err := env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 4200, 1, "buyer"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "en",
	})
	if err != nil {
		t.Fatalf("get product with promo price: %v", err)
	}
	if item.Price.PayableAmountMinor != 700 {
		t.Fatalf("expected promo payable amount 700, got %d", item.Price.PayableAmountMinor)
	}
	if item.Description == "" {
		t.Fatal("expected localization fallback to keep a non-empty description")
	}

	// Invalid price guard.
	// Try to create and update prices with discounts greater than the list amount.
	// Verify catalog writes reject underflow-prone price definitions.
	if _, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
		WorkspaceID:         testWorkspaceID,
		ProductID:           productID,
		AssetCode:           "RUB",
		ListAmountMinor:     100,
		DiscountAmountMinor: 101,
		StartsAt:            &priceStartsAt,
		EndsAt:              &priceEndsAt,
	}); !errors.Is(err, repository.ErrInvalidPrice) {
		t.Fatalf("expected invalid price create rejection, got %v", err)
	}
	if _, err := env.api.Admin.UpdateCatalogPrice(env.ctx, product.UpdatePriceParams{
		ID:                  item.Price.ID,
		WorkspaceID:         testWorkspaceID,
		AssetCode:           "RUB",
		ListAmountMinor:     100,
		DiscountAmountMinor: 101,
		StartsAt:            &priceStartsAt,
		EndsAt:              &priceEndsAt,
	}); !errors.Is(err, repository.ErrInvalidPrice) {
		t.Fatalf("expected invalid price update rejection, got %v", err)
	}
}

func TestPaymentProductCacheConsistency(t *testing.T) {
	// Product cache synchronization.
	// Create a product, mutate localization and price, then rebuild the full workspace cache.
	// Verify Product.Get reflects source data changes through the cache read model.
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		AssetCode:       "RUB",
		ListAmountMinor: 1000,
	})

	item, err := env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 4600, 1, "cache-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("get cached product: %v", err)
	}
	if item.Title != "Security product" {
		t.Fatalf("expected initial cached title, got %q", item.Title)
	}

	if err := env.api.Admin.SaveLocalization(env.ctx, product.UpsertLocalizationParams{
		WorkspaceID:     testWorkspaceID,
		Locale:          "ru",
		LocalizationKey: productID + ".title",
		Value:           "Updated cached product",
	}); err != nil {
		t.Fatalf("update cached product title: %v", err)
	}
	item, err = env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 4600, 1, "cache-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("get product after localization update: %v", err)
	}
	if item.Title != "Updated cached product" {
		t.Fatalf("expected updated cached title, got %q", item.Title)
	}

	priceStart := time.Now().Add(-time.Hour)
	priceEnd := time.Now().Add(time.Hour)
	if _, err := env.api.Admin.UpdateCatalogPrice(env.ctx, product.UpdatePriceParams{
		ID:                  item.Price.ID,
		WorkspaceID:         testWorkspaceID,
		AssetCode:           "RUB",
		ListAmountMinor:     1500,
		DiscountAmountMinor: 250,
		StartsAt:            &priceStart,
		EndsAt:              &priceEnd,
	}); err != nil {
		t.Fatalf("update cached product price: %v", err)
	}
	item, err = env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 4600, 1, "cache-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("get product after price update: %v", err)
	}
	if item.Price.PayableAmountMinor != 1250 {
		t.Fatalf("expected updated cached price 1250, got %d", item.Price.PayableAmountMinor)
	}

	if _, err := env.db.ExecContext(env.ctx, "DELETE FROM payment_product_cache WHERE workspace_id = $1", testWorkspaceID); err != nil {
		t.Fatalf("clear product cache: %v", err)
	}
	if err := env.client.ResetCache(); err != nil {
		t.Fatalf("clear go product cache: %v", err)
	}
	freshAPI, err := NewWithDatabase(env.ctx, env.db, paymentTestOptions())
	if err != nil {
		t.Fatalf("create fresh payment service: %v", err)
	}
	defer freshAPI.Close()
	if _, err := freshAPI.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 4600, 1, "cache-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected empty cache lookup to miss, got %v", err)
	}
	if err := env.api.Admin.RebuildProductCache(env.ctx, testWorkspaceID); err != nil {
		t.Fatalf("rebuild workspace product cache: %v", err)
	}
	item, err = env.api.User.GetProduct(env.ctx, product.GetParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 4600, 1, "cache-user"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("get product after workspace cache rebuild: %v", err)
	}
	if item.Title != "Updated cached product" || item.Price.PayableAmountMinor != 1250 {
		t.Fatalf("unexpected rebuilt cache product: title=%q price=%d", item.Title, item.Price.PayableAmountMinor)
	}
}

func TestPaymentCheckoutNegativeSecurityCases(t *testing.T) {
	// Test database setup.
	// Create the MySQL database connection and bootstrap the payment schema.
	// Verify checkout rejects mismatch, duplicate, and incompatible provider cases.
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		AssetCode:       "RUB",
		ListAmountMinor: 1000,
	})

	// Provider and asset compatibility.
	// Try to create a VKMA attempt for a RUB order.
	// Verify providers cannot charge assets they are not configured to support.
	order, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 4300, 1, "buyer-provider"),
		ProductID: productID,
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("create provider guard order: %v", err)
	}
	badProviderPaymentID := uniquePaymentID("bad-provider")
	if _, err := env.api.User.CreateAttempt(env.ctx, checkout.CreateAttemptParams{
		OrderID:           order.ID,
		ProviderCode:      "vkma",
		ProviderPaymentID: &badProviderPaymentID,
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected provider asset rejection, got %v", err)
	}

	// Completion mismatch guard.
	// Complete the same attempt with wrong provider, amount, asset, and payment id values.
	// Verify every mismatch is rejected before fulfillment can be created.
	goodProviderPaymentID := uniquePaymentID("good")
	attempt, err := env.api.User.CreateAttempt(env.ctx, checkout.CreateAttemptParams{
		OrderID:           order.ID,
		ProviderCode:      "yookassa",
		ProviderPaymentID: &goodProviderPaymentID,
	})
	if err != nil {
		t.Fatalf("create mismatch attempt: %v", err)
	}
	wrongPaymentID := goodProviderPaymentID + "_wrong"
	mismatchCases := []struct {
		name   string
		params checkout.CompleteAttemptParams
	}{
		{name: "provider", params: checkout.CompleteAttemptParams{AttemptID: attempt.ID, ProviderCode: "platega", ProviderPaymentID: &goodProviderPaymentID, AmountMinor: attempt.AmountMinor, AssetCode: "RUB"}},
		{name: "payment_id", params: checkout.CompleteAttemptParams{AttemptID: attempt.ID, ProviderCode: "yookassa", ProviderPaymentID: &wrongPaymentID, AmountMinor: attempt.AmountMinor, AssetCode: "RUB"}},
		{name: "missing_payment_id", params: checkout.CompleteAttemptParams{AttemptID: attempt.ID, ProviderCode: "yookassa", AmountMinor: attempt.AmountMinor, AssetCode: "RUB"}},
		{name: "amount", params: checkout.CompleteAttemptParams{AttemptID: attempt.ID, ProviderCode: "yookassa", ProviderPaymentID: &goodProviderPaymentID, AmountMinor: attempt.AmountMinor + 1, AssetCode: "RUB"}},
		{name: "asset", params: checkout.CompleteAttemptParams{AttemptID: attempt.ID, ProviderCode: "yookassa", ProviderPaymentID: &goodProviderPaymentID, AmountMinor: attempt.AmountMinor, AssetCode: "VOTE"}},
	}
	for _, tc := range mismatchCases {
		if _, err := env.api.Operational.CompleteAttempt(env.ctx, tc.params); !errors.Is(err, repository.ErrPaymentMismatch) {
			t.Fatalf("%s mismatch should be rejected, got %v", tc.name, err)
		}
	}

	completed, err := env.api.Operational.CompleteAttempt(env.ctx, checkout.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      "yookassa",
		ProviderPaymentID: &goodProviderPaymentID,
		AmountMinor:       attempt.AmountMinor,
		AssetCode:         "RUB",
	})
	if err != nil {
		t.Fatalf("complete valid attempt after mismatch checks: %v", err)
	}
	if completed.FulfillmentID == nil {
		t.Fatal("expected fulfillment after valid completion")
	}

	// Duplicate and invalid state guards.
	// Reuse provider ids, event ids, payload hashes, and a fulfilled order.
	// Verify uniqueness and order-state checks stop replay attempts.
	if _, err := env.api.User.CreateAttempt(env.ctx, checkout.CreateAttemptParams{
		OrderID:           order.ID,
		ProviderCode:      "yookassa",
		ProviderPaymentID: &goodProviderPaymentID,
	}); !errors.Is(err, repository.ErrOrderStateInvalid) {
		t.Fatalf("expected fulfilled order to reject new attempt, got %v", err)
	}

	eventID := uniquePaymentID("event")
	if _, err := env.api.Operational.CreateEvent(env.ctx, checkout.CreateEventParams{
		ProviderCode:      "yookassa",
		AttemptID:         utils.Ref(int64(attempt.ID)),
		OrderID:           utils.Ref(int64(order.ID)),
		ProviderEventID:   &eventID,
		ProviderPaymentID: &goodProviderPaymentID,
		EventType:         "succeeded",
		EventStatus:       utils.Ref("succeeded"),
		PayloadHash:       sha256Hex(eventID),
	}); err != nil {
		t.Fatalf("create event: %v", err)
	}
	if _, err := env.api.Operational.CreateEvent(env.ctx, checkout.CreateEventParams{
		ProviderCode:      "yookassa",
		AttemptID:         utils.Ref(int64(attempt.ID)),
		OrderID:           utils.Ref(int64(order.ID)),
		ProviderEventID:   &eventID,
		ProviderPaymentID: &goodProviderPaymentID,
		EventType:         "succeeded",
		EventStatus:       utils.Ref("succeeded"),
		PayloadHash:       sha256Hex(eventID + "_different"),
	}); err == nil {
		t.Fatal("expected duplicate provider event id to be rejected")
	}
}

func TestPaymentGiftKeyNegativeCases(t *testing.T) {
	// Test database setup.
	// Create the MySQL database connection and bootstrap the payment schema.
	// Verify gift purchase keys cannot be reused, expired, or guessed.
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		AssetCode:       "RUB",
		ListAmountMinor: 1000,
	})

	// Expired and unknown gift keys.
	// Create an already expired key and also try a random unknown key.
	// Verify neither key can reveal or create a payable gift offer.
	expiredAt := time.Now().Add(-time.Minute)
	expiredKey, err := env.api.Admin.CreateProductKey(env.ctx, product.CreateKeyParams{
		WorkspaceID:    testWorkspaceID,
		AppID:          4400,
		PlatformID:     1,
		PlatformUserID: "recipient-expired",
		ProductID:      productID,
		MaxUses:        1,
		ExpiresAt:      &expiredAt,
	})
	if err != nil {
		t.Fatalf("create expired key: %v", err)
	}
	for _, key := range []string{expiredKey, "not-a-real-key"} {
		if _, err := env.api.User.GetProductByKey(env.ctx, product.GetByKeyParams{Key: key, AssetCode: "RUB", Locale: "ru"}); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected key %q to be hidden, got %v", key, err)
		}
		if _, err := env.api.User.CreateOrderByKey(env.ctx, checkout.CreateOrderByKeyParams{
			Key: key,
			Payer: &services.Actor{
				PlatformID:     1,
				PlatformUserID: "payer",
			},
			AssetCode: "RUB",
			Locale:    "ru",
		}); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected key %q to reject order creation, got %v", key, err)
		}
	}

	// Gift key max uses.
	// Complete one gift purchase from a max-use-one key and try using it again.
	// Verify used keys cannot create a second payable order.
	key, err := env.api.Admin.CreateProductKey(env.ctx, product.CreateKeyParams{
		WorkspaceID:    testWorkspaceID,
		AppID:          4400,
		PlatformID:     1,
		PlatformUserID: "recipient-once",
		ProductID:      productID,
		MaxUses:        1,
	})
	if err != nil {
		t.Fatalf("create one-use key: %v", err)
	}
	payerPlatformID := int64(1)
	payerUserID := "payer-once"
	order, err := env.api.User.CreateOrderByKey(env.ctx, checkout.CreateOrderByKeyParams{
		Key: key,
		Payer: &services.Actor{
			PlatformID:     payerPlatformID,
			PlatformUserID: payerUserID,
		},
		AssetCode: "RUB",
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("create gift order: %v", err)
	}
	providerPaymentID := uniquePaymentID("gift-once")
	attempt, err := env.api.User.CreateAttempt(env.ctx, checkout.CreateAttemptParams{
		OrderID:           order.ID,
		ProviderCode:      "yookassa",
		ProviderPaymentID: &providerPaymentID,
	})
	if err != nil {
		t.Fatalf("create gift attempt: %v", err)
	}
	if _, err := env.api.Operational.CompleteAttempt(env.ctx, checkout.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      "yookassa",
		ProviderPaymentID: &providerPaymentID,
		AmountMinor:       attempt.AmountMinor,
		AssetCode:         "RUB",
	}); err != nil {
		t.Fatalf("complete gift attempt: %v", err)
	}
	if _, err := env.api.User.CreateOrderByKey(env.ctx, checkout.CreateOrderByKeyParams{
		Key: key,
		Payer: &services.Actor{
			PlatformID:     payerPlatformID,
			PlatformUserID: payerUserID,
		},
		AssetCode: "RUB",
		Locale:    "ru",
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected used gift key to reject another order, got %v", err)
	}
}

func createPaymentProduct(t *testing.T, env paymentTestEnv, opt testProductOptions) string {
	t.Helper()

	now := time.Now()
	productID := opt.ProductID
	if productID == "" {
		productID = fmt.Sprintf("secure_product_%d", time.Now().UnixNano())
	}
	groupCode := fmt.Sprintf("secure_group_%d", time.Now().UnixNano())
	itemID := fmt.Sprintf("secure_item_%d", time.Now().UnixNano())
	productTitleKey := productID + ".title"
	productDescriptionKey := productID + ".description"
	itemTitleKey := itemID + ".title"
	itemDescriptionKey := itemID + ".description"
	availableFrom := opt.AvailableFrom
	if availableFrom.IsZero() {
		availableFrom = now.Add(-time.Hour)
	}
	availableUntil := opt.AvailableUntil
	if availableUntil.IsZero() {
		availableUntil = now.Add(time.Hour)
	}
	priceStartsAt := now.Add(-time.Hour)
	priceEndsAt := now.Add(time.Hour)
	assetCode := opt.AssetCode
	if assetCode == "" {
		assetCode = "RUB"
	}
	listAmount := opt.ListAmountMinor
	if listAmount == 0 {
		listAmount = 1000
	}
	globalInterval := opt.GlobalInterval
	if globalInterval == "" {
		globalInterval = "UNLIMITED"
	}
	userInterval := opt.UserInterval
	if userInterval == "" {
		userInterval = "UNLIMITED"
	}
	workspaceID := opt.WorkspaceID
	if workspaceID == "" {
		workspaceID = testWorkspaceID
	}

	if err := env.api.Admin.SaveProductGroup(env.ctx, product.UpsertGroupParams{
		Code:           groupCode,
		WorkspaceID:    workspaceID,
		TitleKey:       utils.Ref(groupCode + ".title"),
		DescriptionKey: utils.Ref(groupCode + ".description"),
		Position:       1,
		IsActive:       true,
	}); err != nil {
		t.Fatalf("upsert test product group: %v", err)
	}
	if err := env.api.Admin.SaveProduct(env.ctx, product.UpsertParams{
		ID:                  productID,
		WorkspaceID:         workspaceID,
		GroupCode:           utils.Ref(groupCode),
		TitleKey:            productTitleKey,
		DescriptionKey:      utils.Ref(productDescriptionKey),
		Position:            1,
		GlobalLimit:         opt.GlobalLimit,
		GlobalInterval:      globalInterval,
		GlobalIntervalCount: opt.GlobalIntervalCount,
		UserLimit:           opt.UserLimit,
		UserInterval:        userInterval,
		UserIntervalCount:   opt.UserIntervalCount,
		AvailableFrom:       &availableFrom,
		AvailableUntil:      &availableUntil,
		IsVisible:           !opt.IsHidden,
		IsClosed:            opt.IsClosed,
	}); err != nil {
		t.Fatalf("create test product: %v", err)
	}
	for _, localization := range []product.UpsertLocalizationParams{
		{Locale: "ru", LocalizationKey: productTitleKey, Value: "Security product"},
		{Locale: "ru", LocalizationKey: productDescriptionKey, Value: "Security product description"},
		{Locale: "ru", LocalizationKey: itemTitleKey, Value: "Security item"},
		{Locale: "ru", LocalizationKey: itemDescriptionKey, Value: "Security item description"},
	} {
		localization.WorkspaceID = workspaceID
		if err := env.api.Admin.SaveLocalization(env.ctx, localization); err != nil {
			t.Fatalf("upsert test localization: %v", err)
		}
	}
	if err := env.api.Admin.AttachProductItem(env.ctx, product.AddItemParams{
		ProductID:   productID,
		WorkspaceID: workspaceID,
		ItemID:      itemID,
		Quantity:    1,
	}); err != nil {
		t.Fatalf("add test item: %v", err)
	}
	if _, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
		ProductID:           productID,
		WorkspaceID:         workspaceID,
		AssetCode:           assetCode,
		ListAmountMinor:     listAmount,
		DiscountAmountMinor: opt.DiscountAmountMinor,
		StartsAt:            &priceStartsAt,
		EndsAt:              &priceEndsAt,
	}); err != nil {
		t.Fatalf("create test price: %v", err)
	}

	return productID
}

func mergeProductOptions(base testProductOptions, override testProductOptions) testProductOptions {
	if override.AssetCode != "" {
		base.AssetCode = override.AssetCode
	}
	if override.ListAmountMinor != 0 {
		base.ListAmountMinor = override.ListAmountMinor
	}
	if override.DiscountAmountMinor != 0 {
		base.DiscountAmountMinor = override.DiscountAmountMinor
	}
	if override.GlobalLimit != 0 {
		base.GlobalLimit = override.GlobalLimit
	}
	if override.GlobalInterval != "" {
		base.GlobalInterval = override.GlobalInterval
	}
	if override.GlobalIntervalCount != 0 {
		base.GlobalIntervalCount = override.GlobalIntervalCount
	}
	if override.UserLimit != 0 {
		base.UserLimit = override.UserLimit
	}
	if override.UserInterval != "" {
		base.UserInterval = override.UserInterval
	}
	if override.UserIntervalCount != 0 {
		base.UserIntervalCount = override.UserIntervalCount
	}
	if !override.AvailableFrom.IsZero() {
		base.AvailableFrom = override.AvailableFrom
	}
	if !override.AvailableUntil.IsZero() {
		base.AvailableUntil = override.AvailableUntil
	}
	base.IsVisible = override.IsVisible
	base.IsHidden = override.IsHidden
	base.IsClosed = override.IsClosed
	return base
}

func completeTestPayment(t *testing.T, env paymentTestEnv, productID string, prefix string, platformUserID string, providerCode string, assetCode string) {
	t.Helper()

	order, err := env.api.User.CreateOrder(env.ctx, checkout.CreateOrderParams{
		Identity:  paymentTestIdentity(testWorkspaceID, 4100, 1, platformUserID),
		ProductID: productID,
		AssetCode: assetCode,
		Locale:    "ru",
	})
	if err != nil {
		t.Fatalf("create order for %s: %v", prefix, err)
	}
	providerPaymentID := uniquePaymentID(prefix)
	attempt, err := env.api.User.CreateAttempt(env.ctx, checkout.CreateAttemptParams{
		OrderID:           order.ID,
		ProviderCode:      providerCode,
		ProviderPaymentID: &providerPaymentID,
	})
	if err != nil {
		t.Fatalf("create attempt for %s: %v", prefix, err)
	}
	if _, err := env.api.Operational.CompleteAttempt(env.ctx, checkout.CompleteAttemptParams{
		AttemptID:         attempt.ID,
		ProviderCode:      providerCode,
		ProviderPaymentID: &providerPaymentID,
		AmountMinor:       attempt.AmountMinor,
		AssetCode:         assetCode,
	}); err != nil {
		t.Fatalf("complete attempt for %s: %v", prefix, err)
	}
}

func TestTelegramStarsAdapterFullCycle(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "telegram_stars_product",
		AssetCode:       telegramstars.AssetCode,
		ListAmountMinor: 25,
	})

	var createInvoicePayload telegramStarsTestCreateInvoicePayload
	var answeredPreCheckout bool
	var refundCalled bool
	var editSubscriptionCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/createInvoiceLink"):
			if err := json.NewDecoder(r.Body).Decode(&createInvoicePayload); err != nil {
				t.Fatalf("decode createInvoiceLink payload: %v", err)
			}
			writeTelegramStarsResult(t, w, "https://t.me/invoice/test-link")
		case strings.HasSuffix(r.URL.Path, "/answerPreCheckoutQuery"):
			answeredPreCheckout = true
			writeTelegramStarsResult(t, w, true)
		case strings.HasSuffix(r.URL.Path, "/refundStarPayment"):
			refundCalled = true
			writeTelegramStarsResult(t, w, true)
		case strings.HasSuffix(r.URL.Path, "/editUserStarSubscription"):
			editSubscriptionCalled = true
			writeTelegramStarsResult(t, w, true)
		default:
			t.Fatalf("unexpected telegram bot api path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	credentials := telegramstars.Credentials{
		BotToken:   "test-token",
		APIBaseURL: server.URL,
	}

	payment, err := env.api.Adapters.TelegramStars.CreatePayment(env.ctx, telegramstars.CreatePaymentParams{
		Credentials:    credentials,
		WorkspaceID:    testWorkspaceID,
		AppID:          7007,
		PlatformID:     2,
		PlatformUserID: "12345",
		ProductID:      productID,
		Locale:         "ru",
		Title:          "Stars product",
		Description:    "Stars product description",
	})
	if err != nil {
		t.Fatalf("create telegram stars payment: %v", err)
	}
	if payment.InvoiceLink != "https://t.me/invoice/test-link" || payment.AmountMinor != 25 {
		t.Fatalf("unexpected telegram stars payment: %#v", payment)
	}
	if createInvoicePayload.Currency != "XTR" || createInvoicePayload.ProviderToken != "" || len(createInvoicePayload.Prices) != 1 {
		t.Fatalf("unexpected createInvoiceLink payload: %#v", createInvoicePayload)
	}
	if createInvoicePayload.Payload != payment.OrderPublicID || createInvoicePayload.Prices[0].Amount != 25 {
		t.Fatalf("unexpected invoice payload or amount: %#v", createInvoicePayload)
	}
	var storedInvoiceLink sql.NullString
	if err := env.db.QueryRowContext(env.ctx, `
SELECT confirmation_url
FROM payment_attempt
WHERE id = $1`, payment.AttemptID).Scan(&storedInvoiceLink); err != nil {
		t.Fatalf("select telegram stars confirmation url: %v", err)
	}
	if storedInvoiceLink.Valid {
		t.Fatalf("telegram stars invoice link must not be stored in payment_attempt: %s", storedInvoiceLink.String)
	}

	preCheckout, err := env.api.Adapters.TelegramStars.HandlePreCheckoutQuery(env.ctx, telegramstars.PreCheckoutQuery{
		Credentials:    credentials,
		ID:             "pre-checkout-1",
		FromID:         12345,
		Currency:       telegramstars.AssetCode,
		TotalAmount:    payment.AmountMinor,
		InvoicePayload: payment.OrderPublicID,
	})
	if err != nil {
		t.Fatalf("handle pre-checkout query: %v", err)
	}
	if !answeredPreCheckout || !preCheckout.Accepted {
		t.Fatalf("expected accepted pre-checkout, got %#v", preCheckout)
	}

	success, err := env.api.Adapters.TelegramStars.HandleSuccessfulPayment(env.ctx, telegramstars.SuccessfulPayment{
		Currency:                telegramstars.AssetCode,
		TotalAmount:             payment.AmountMinor,
		InvoicePayload:          payment.OrderPublicID,
		TelegramPaymentChargeID: "tg-charge-1",
	})
	if err != nil {
		t.Fatalf("handle successful payment: %v", err)
	}
	if success.OrderID != payment.OrderID || success.AttemptID != payment.AttemptID {
		t.Fatalf("unexpected successful payment result: %#v", success)
	}
	assertOrderStatus(t, env.ctx, env.db, payment.OrderID, "fulfilled")
	assertAttemptStatus(t, env.ctx, env.db, payment.AttemptID, "succeeded")

	var chargeID string
	if err := env.db.QueryRowContext(env.ctx, `
SELECT provider_charge_id
FROM payment_attempt
WHERE id = $1`, payment.AttemptID).Scan(&chargeID); err != nil {
		t.Fatalf("select telegram charge id: %v", err)
	}
	if chargeID != "tg-charge-1" {
		t.Fatalf("unexpected telegram charge id: %s", chargeID)
	}

	refunded, err := env.api.Admin.ExecuteRefund(env.ctx, paymentrefund.Params{
		WorkspaceID:    testWorkspaceID,
		OrderID:        payment.OrderID,
		AttemptID:      payment.AttemptID,
		Reason:         "test refund",
		ProviderParams: credentials,
	})
	if err != nil {
		t.Fatalf("orchestrate telegram stars refund: %v", err)
	}
	if !refundCalled {
		t.Fatal("expected refundStarPayment call")
	}
	if refunded.OrderID != payment.OrderID || refunded.AttemptID != payment.AttemptID || refunded.Status != "succeeded" {
		t.Fatalf("unexpected orchestrated refund result: %#v", refunded)
	}
	assertOrderStatus(t, env.ctx, env.db, payment.OrderID, "refunded")
	assertAttemptStatus(t, env.ctx, env.db, payment.AttemptID, "refunded")
	assertRefundStatus(t, env.ctx, env.db, refunded.RefundID, "succeeded")

	if err := env.api.Adapters.TelegramStars.EditSubscription(env.ctx, telegramstars.EditSubscriptionParams{
		Credentials:             credentials,
		UserID:                  12345,
		TelegramPaymentChargeID: "tg-charge-1",
		IsCanceled:              true,
	}); err != nil {
		t.Fatalf("edit telegram stars subscription: %v", err)
	}
	if !editSubscriptionCalled {
		t.Fatal("expected editUserStarSubscription call")
	}
}

func assertRefundStatus(t *testing.T, ctx context.Context, db *sql.DB, refundID uint64, want string) {
	t.Helper()
	var got string
	if err := db.QueryRowContext(ctx, "SELECT status FROM payment_refund WHERE id = $1", refundID).Scan(&got); err != nil {
		t.Fatalf("select refund status: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected refund status: got %s want %s", got, want)
	}
}

func TestTelegramStarsAdapterSubscriptionCycle(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "telegram_stars_subscription",
		AssetCode:       telegramstars.AssetCode,
		ListAmountMinor: 50,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTelegramStarsResult(t, w, "https://t.me/invoice/subscription-link")
	}))
	defer server.Close()

	credentials := telegramstars.Credentials{
		BotToken:   "test-token",
		APIBaseURL: server.URL,
	}

	payment, err := env.api.Adapters.TelegramStars.CreatePayment(env.ctx, telegramstars.CreatePaymentParams{
		Credentials:        credentials,
		WorkspaceID:        testWorkspaceID,
		AppID:              7007,
		PlatformID:         2,
		PlatformUserID:     "telegram-sub-user",
		ProductID:          productID,
		Locale:             "ru",
		Title:              "Stars subscription",
		Description:        "Stars subscription description",
		SubscriptionPeriod: 30 * 24 * 60 * 60,
	})
	if err != nil {
		t.Fatalf("create telegram stars subscription payment: %v", err)
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if _, err := env.api.Adapters.TelegramStars.HandleSuccessfulPayment(env.ctx, telegramstars.SuccessfulPayment{
		Currency:                   telegramstars.AssetCode,
		TotalAmount:                payment.AmountMinor,
		InvoicePayload:             payment.OrderPublicID,
		TelegramPaymentChargeID:    "tg-sub-charge-1",
		SubscriptionExpirationDate: expiresAt.Unix(),
		IsRecurring:                true,
		IsFirstRecurring:           true,
	}); err != nil {
		t.Fatalf("handle successful subscription payment: %v", err)
	}

	active, err := env.api.User.IsSubscriptionActive(env.ctx, subscription.IsActiveParams{
		Identity:     paymentTestIdentity(testWorkspaceID, 7007, 2, "telegram-sub-user"),
		ProductID:    productID,
		ProviderCode: telegramstars.ProviderCode,
	})
	if err != nil {
		t.Fatalf("check telegram stars subscription active: %v", err)
	}
	if !active {
		t.Fatal("expected telegram stars subscription to be active")
	}
}

type telegramStarsTestCreateInvoicePayload struct {
	Title              string                       `json:"title"`
	Description        string                       `json:"description"`
	Payload            string                       `json:"payload"`
	ProviderToken      string                       `json:"provider_token"`
	Currency           string                       `json:"currency"`
	Prices             []telegramstars.LabeledPrice `json:"prices"`
	SubscriptionPeriod int                          `json:"subscription_period"`
}

func writeTelegramStarsResult(t *testing.T, w http.ResponseWriter, result any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"result": result,
	}); err != nil {
		t.Fatalf("write telegram stars response: %v", err)
	}
}

func TestTONAdapterFullCycleAndCursor(t *testing.T) {

	// Test database setup.
	// Create a TON-priced product and a blockchain payment request.
	// Verify the adapter stores a pending attempt with the order public id as comment.
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "ton_product",
		AssetCode:       paymentton.AssetTON,
		ListAmountMinor: 1_000_000_000,
	})
	configureTONWallet(t, env, testWorkspaceID, paymentton.NetworkMainnet, "EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c")

	payment, err := env.api.Adapters.TON.CreatePayment(env.ctx, paymentton.CreatePaymentParams{
		WorkspaceID:    testWorkspaceID,
		AppID:          6006,
		PlatformID:     1,
		PlatformUserID: "buyer-ton",
		ProductID:      productID,
		AssetCode:      paymentton.AssetTON,
		Locale:         "ru",
	})
	if err != nil {
		t.Fatalf("create ton payment: %v", err)
	}
	if payment.Comment == "" || payment.AmountMinor != 1_000_000_000 {
		t.Fatalf("unexpected ton payment response: %#v", payment)
	}
	tx, err := env.api.Adapters.TON.CreateTransaction(env.ctx, paymentton.CreateTransactionParams{
		AssetCode:   payment.AssetCode,
		Network:     payment.Network,
		Destination: payment.WalletAddress,
		AmountMinor: payment.AmountMinor,
		Comment:     payment.Comment,
	})
	if err != nil {
		t.Fatalf("create ton transaction: %v", err)
	}
	if tonkeeper := paymentton.TonkeeperLink(tx); tonkeeper == "" {
		t.Fatalf("expected tonkeeper link for ton transaction: %#v", tx)
	}
	assertOrderStatus(t, env.ctx, env.db, payment.OrderID, "pending_payment")
	assertAttemptStatus(t, env.ctx, env.db, payment.AttemptID, "pending")

	// TON transfer processing.
	// Emulate an incoming TON transfer with the expected comment and amount.
	// Verify the shared checkout completion path fulfills the order and stores LT.
	result, err := env.api.Adapters.TON.ProcessTransfer(env.ctx, paymentton.IncomingTransfer{
		WorkspaceID:        testWorkspaceID,
		Network:            paymentton.NetworkMainnet,
		WalletAddress:      payment.WalletAddress,
		AssetCode:          paymentton.AssetTON,
		TxHash:             "ton_tx_hash_1",
		LogicalTime:        uint64(time.Now().UnixNano()),
		SourceAddress:      "EQ_SOURCE",
		DestinationAddress: payment.WalletAddress,
		AmountMinor:        payment.AmountMinor,
		Comment:            payment.Comment,
	})
	if err != nil {
		t.Fatalf("process ton transfer: %v", err)
	}
	if result.Transaction == 0 || result.OrderID != payment.OrderID || result.AttemptID != payment.AttemptID {
		t.Fatalf("unexpected ton process result: %#v", result)
	}
	assertOrderStatus(t, env.ctx, env.db, payment.OrderID, "fulfilled")
	assertAttemptStatus(t, env.ctx, env.db, payment.AttemptID, "succeeded")

	var lastLT uint64
	if err := env.db.QueryRowContext(env.ctx, `
SELECT cursor_sequence
FROM payment_provider_cursor
WHERE workspace_id = $1
  AND provider_code = 'ton'
  AND network = 'mainnet'
  AND source_key = $2`, testWorkspaceID, payment.WalletAddress).Scan(&lastLT); err != nil {
		t.Fatalf("select ton cursor: %v", err)
	}
	if lastLT == 0 {
		t.Fatal("expected ton cursor last_lt to be stored")
	}

	// TON transfer idempotency.
	// Process the same transaction hash again.
	// Verify duplicate blockchain events do not create another fulfillment.
	again, err := env.api.Adapters.TON.ProcessTransfer(env.ctx, paymentton.IncomingTransfer{
		WorkspaceID:        testWorkspaceID,
		Network:            paymentton.NetworkMainnet,
		WalletAddress:      payment.WalletAddress,
		AssetCode:          paymentton.AssetTON,
		TxHash:             "ton_tx_hash_1",
		LogicalTime:        lastLT,
		SourceAddress:      "EQ_SOURCE",
		DestinationAddress: payment.WalletAddress,
		AmountMinor:        payment.AmountMinor,
		Comment:            payment.Comment,
	})
	if err != nil {
		t.Fatalf("process duplicate ton transfer: %v", err)
	}
	if !again.AlreadyDone || again.Transaction != result.Transaction {
		t.Fatalf("expected duplicate ton transaction to be idempotent: %#v", again)
	}
}

func TestTONAdapterJettonTransfer(t *testing.T) {

	// Test database setup.
	// Create a USDT_TON-priced product and payment request.
	// Verify a Jetton transfer is matched by comment and asset code.
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "ton_jetton_product",
		AssetCode:       "USDT_TON",
		ListAmountMinor: 1_000_000,
	})
	configureTONWallet(t, env, testWorkspaceID, paymentton.NetworkMainnet, "EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM9c")

	payment, err := env.api.Adapters.TON.CreatePayment(env.ctx, paymentton.CreatePaymentParams{
		WorkspaceID:    testWorkspaceID,
		AppID:          6006,
		PlatformID:     1,
		PlatformUserID: "buyer-ton-jetton",
		ProductID:      productID,
		AssetCode:      "USDT_TON",
		Locale:         "ru",
	})
	if err != nil {
		t.Fatalf("create ton jetton payment: %v", err)
	}
	if payment.Decimals != 6 {
		t.Fatalf("expected USDT_TON decimals=6, got %d", payment.Decimals)
	}

	result, err := env.api.Adapters.TON.ProcessTransfer(env.ctx, paymentton.IncomingTransfer{
		WorkspaceID:        testWorkspaceID,
		Network:            paymentton.NetworkMainnet,
		WalletAddress:      payment.WalletAddress,
		AssetCode:          "USDT_TON",
		TxHash:             "ton_jetton_tx_hash_1",
		LogicalTime:        uint64(time.Now().UnixNano()),
		SourceAddress:      "JETTON_WALLET",
		DestinationAddress: payment.WalletAddress,
		AmountMinor:        payment.AmountMinor,
		Comment:            payment.Comment,
		JettonSender:       "EQ_JETTON_SENDER",
	})
	if err != nil {
		t.Fatalf("process ton jetton transfer: %v", err)
	}
	if result.Transaction == 0 || result.OrderID != payment.OrderID {
		t.Fatalf("unexpected ton jetton process result: %#v", result)
	}
	assertOrderStatus(t, env.ctx, env.db, payment.OrderID, "fulfilled")
	assertAttemptStatus(t, env.ctx, env.db, payment.AttemptID, "succeeded")
}

func TestTONAdapterResolvesMultipleJettonAssets(t *testing.T) {
	env := setupPaymentIntegrationTest(t)

	tests := []struct {
		master   string
		code     string
		decimals uint16
	}{
		{
			master:   "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs",
			code:     "USDT_TON",
			decimals: 6,
		},
		{
			master:   "EQC98_qAmNEptUtPc7W6xdHh_ZHrBUFpw5Ft_IzNU20QAJav",
			code:     "TSTON_TON",
			decimals: 9,
		},
	}

	for _, tt := range tests {
		master, err := address.ParseAddr(tt.master)
		if err != nil {
			t.Fatalf("parse %s master address: %v", tt.code, err)
		}
		resolved, err := env.api.Adapters.TON.ResolveJettonAsset(env.ctx, paymentton.NetworkMainnet, master.StringRaw())
		if err != nil {
			t.Fatalf("resolve %s by raw master address: %v", tt.code, err)
		}
		if resolved.Code != tt.code || resolved.Decimals != tt.decimals {
			t.Fatalf("unexpected resolved asset for %s: %#v", tt.code, resolved)
		}
	}
}

func TestTONAdapterUsesWorkspaceWalletForPayment(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "ton_configured_wallet_product",
		AssetCode:       paymentton.AssetTON,
		ListAmountMinor: 100_000_000,
	})
	rawWallet := "0:0000000000000000000000000000000000000000000000000000000000000000"
	expectedWallet, err := paymentton.NormalizeWalletAddress(rawWallet, paymentton.NetworkMainnet)
	if err != nil {
		t.Fatalf("normalize test wallet: %v", err)
	}
	configureTONWallet(t, env, testWorkspaceID, paymentton.NetworkMainnet, rawWallet)

	payment, err := env.api.Adapters.TON.CreatePayment(env.ctx, paymentton.CreatePaymentParams{
		WorkspaceID:    testWorkspaceID,
		AppID:          6006,
		PlatformID:     1,
		PlatformUserID: "buyer-ton-configured",
		ProductID:      productID,
		AssetCode:      paymentton.AssetTON,
		Locale:         "ru",
	})
	if err != nil {
		t.Fatalf("create ton payment with workspace wallet: %v", err)
	}
	if payment.WalletAddress != expectedWallet {
		t.Fatalf("expected workspace TON wallet in payment response: %#v", payment)
	}
}

func TestTONAdapterRequiresWorkspaceWallet(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "ton_missing_wallet_product",
		AssetCode:       paymentton.AssetTON,
		ListAmountMinor: 100_000_000,
	})

	_, err := env.api.Adapters.TON.CreatePayment(env.ctx, paymentton.CreatePaymentParams{
		WorkspaceID:    testWorkspaceID,
		AppID:          6006,
		PlatformID:     1,
		PlatformUserID: "buyer-ton-missing-wallet",
		ProductID:      productID,
		AssetCode:      paymentton.AssetTON,
		Locale:         "ru",
	})
	if err == nil {
		t.Fatal("expected missing workspace TON wallet to be rejected")
	}
}

func configureTONWallet(t *testing.T, env paymentTestEnv, workspaceID string, network string, wallet string) {
	t.Helper()
	if err := env.api.Admin.SaveTONWallet(env.ctx, admin.TONWalletUpsertParams{
		WorkspaceID:   workspaceID,
		Network:       network,
		WalletAddress: wallet,
		IsEnabled:     true,
	}); err != nil {
		t.Fatalf("configure ton wallet: %v", err)
	}
}

func TestPaymentTONWalletAdminConfig(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	wallet := "UQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJKZ"
	customConfigURL := "https://example.com/ton.config.json"
	expectedWallet, err := paymentton.NormalizeWalletAddress(wallet, paymentton.NetworkMainnet)
	if err != nil {
		t.Fatalf("normalize wallet: %v", err)
	}

	if err := env.api.Admin.SaveTONWallet(env.ctx, admin.TONWalletUpsertParams{
		WorkspaceID:      testWorkspaceID,
		Network:          paymentton.NetworkMainnet,
		WalletAddress:    wallet,
		NetworkConfigURL: &customConfigURL,
		IsEnabled:        true,
	}); err != nil {
		t.Fatalf("save enabled ton wallet: %v", err)
	}
	got, err := env.api.Admin.GetTONWallet(env.ctx, testWorkspaceID)
	if err != nil {
		t.Fatalf("get ton wallet: %v", err)
	}
	if got.WalletAddress != expectedWallet || !got.IsEnabled || !got.NetworkConfigUrl.Valid || got.NetworkConfigUrl.String != customConfigURL {
		t.Fatalf("unexpected ton wallet: %+v", got)
	}

	if err := env.api.Admin.SaveTONWallet(env.ctx, admin.TONWalletUpsertParams{
		WorkspaceID:   testWorkspaceID,
		Network:       paymentton.NetworkMainnet,
		WalletAddress: wallet,
		IsEnabled:     false,
	}); err != nil {
		t.Fatalf("disable ton wallet: %v", err)
	}
	if err := env.api.Adapters.TON.SyncManagedSubscribers(env.ctx); err != nil {
		t.Fatalf("sync managed subscribers with disabled wallet: %v", err)
	}

	got, err = env.api.Admin.GetTONWallet(env.ctx, testWorkspaceID)
	if err != nil {
		t.Fatalf("get disabled ton wallet: %v", err)
	}
	if got.IsEnabled || got.WalletAddress != expectedWallet {
		t.Fatalf("unexpected disabled ton wallet: %+v", got)
	}

	rows, err := env.api.Admin.DeleteTONWallet(env.ctx, testWorkspaceID)
	if err != nil {
		t.Fatalf("delete ton wallet: %v", err)
	}
	if rows != 1 {
		t.Fatalf("delete ton wallet rows = %d, want 1", rows)
	}

	replacementWallet := "0:1111111111111111111111111111111111111111111111111111111111111111"
	expectedReplacementWallet, err := paymentton.NormalizeWalletAddress(replacementWallet, paymentton.NetworkTestnet)
	if err != nil {
		t.Fatalf("normalize replacement wallet: %v", err)
	}
	if err := env.api.Admin.SaveTONWallet(env.ctx, admin.TONWalletUpsertParams{
		WorkspaceID:   testWorkspaceID,
		Network:       paymentton.NetworkMainnet,
		WalletAddress: wallet,
		IsEnabled:     true,
	}); err != nil {
		t.Fatalf("save first workspace ton wallet: %v", err)
	}
	if err := env.api.Admin.SaveTONWallet(env.ctx, admin.TONWalletUpsertParams{
		WorkspaceID:   testWorkspaceID,
		Network:       paymentton.NetworkTestnet,
		WalletAddress: replacementWallet,
		IsEnabled:     true,
	}); err != nil {
		t.Fatalf("replace workspace ton wallet: %v", err)
	}
	got, err = env.api.Admin.GetTONWallet(env.ctx, testWorkspaceID)
	if err != nil {
		t.Fatalf("get replaced ton wallet: %v", err)
	}
	if got.Network != paymentton.NetworkTestnet || got.WalletAddress != expectedReplacementWallet {
		t.Fatalf("expected replaced workspace ton wallet: %+v", got)
	}
}

func TestVKMAAdapterFullCycleWithSubscription(t *testing.T) {

	// Test database setup.
	// Create the MySQL database connection and bootstrap the payment schema.
	// Verify the VKMA adapter runs against the same initialized payment API.
	env := setupPaymentIntegrationTest(t)
	productID, itemID := createVKMAProduct(t, env)

	// VKMA item lookup.
	// Request product information through the VKMA get_item adapter method.
	// Verify VK receives the localized title, item id, and VOTE price.
	item, err := env.api.Adapters.VKMA.GetItemForWorkspace(env.ctx, testWorkspaceID, vkmashop.Params{
		NotificationType: vkmashop.GetItem,
		AppID:            3003,
		UserID:           8001,
		Item:             productID,
		Lang:             "ru",
	})
	if err != nil {
		t.Fatalf("vkma get item: %v", err)
	}
	if item.ItemID != productID || item.Price != 35 || item.Title != "VKMA подписка" {
		t.Fatalf("unexpected vkma item response: %#v", item)
	}

	// VKMA regular
	// Process a chargeable order_status_change notification as a one-time purchase.
	// Verify the adapter creates and fulfills the payment order.
	orderPaymentID := int(time.Now().UnixNano() % 1_000_000_000)
	regular, err := env.api.Adapters.VKMA.ChargeableForWorkspace(env.ctx, testWorkspaceID, vkmashop.Params{
		NotificationType: vkmashop.OrderStatusChange,
		Status:           vkmashop.Chargeable,
		AppID:            3003,
		UserID:           8001,
		Item:             productID,
		OrderID:          orderPaymentID,
		Lang:             "ru",
	})
	if err != nil {
		t.Fatalf("vkma regular chargeable: %v", err)
	}
	if regular.AppOrderID == 0 || regular.OrderID != orderPaymentID {
		t.Fatalf("unexpected regular vkma response: %#v", regular)
	}
	assertOrderStatus(t, env.ctx, env.db, regular.AppOrderID, "fulfilled")
	if _, err := env.db.ExecContext(env.ctx,
		"UPDATE payment_clb_event SET status = 'ok', delivered_at = now() WHERE source_service = 'payment' AND event_key = $1",
		fmt.Sprintf("%s:%d", CallbackEventPaymentOrderFulfilled, regular.AppOrderID),
	); err != nil {
		t.Fatalf("complete vkma fulfilled callback fixture: %v", err)
	}

	// VK initiates refunds through order_status_change; the application only
	// revokes the fulfilled purchase and acknowledges the notification.
	refundRequest := paymentvkma.Request{
		WorkspaceID: testWorkspaceID,
		Params: vkmashop.Params{
			NotificationType: vkmashop.OrderStatusChange,
			Status:           vkmashop.Refunded,
			AppID:            3003,
			UserID:           8001,
			Item:             productID,
			OrderID:          orderPaymentID,
			Lang:             "ru",
		},
	}
	refundResponse, err := env.api.Adapters.VKMA.HandleRequest(env.ctx, refundRequest)
	if err != nil {
		t.Fatalf("vkma regular refund: %v", err)
	}
	refunded, ok := refundResponse.(*paymentvkma.ChargeableResponse)
	if !ok || refunded.AppOrderID != regular.AppOrderID || refunded.OrderID != orderPaymentID {
		t.Fatalf("unexpected regular vkma refund response: %#v", refundResponse)
	}
	assertOrderStatus(t, env.ctx, env.db, regular.AppOrderID, "refunded")

	var attemptID uint64
	var attemptStatus string
	var fulfillmentID uint64
	var fulfillmentStatus string
	var refundID uint64
	var refundStatus string
	var refundCount int
	if err := env.db.QueryRowContext(env.ctx,
		"SELECT id, status FROM payment_attempt WHERE order_id = $1 AND provider_code = $2",
		regular.AppOrderID, paymentvkma.ProviderCode,
	).Scan(&attemptID, &attemptStatus); err != nil {
		t.Fatalf("select refunded vkma attempt: %v", err)
	}
	if attemptStatus != "refunded" {
		t.Fatalf("unexpected refunded vkma attempt status: %s", attemptStatus)
	}
	if err := env.db.QueryRowContext(env.ctx,
		"SELECT id, status FROM payment_fulfillment WHERE order_id = $1",
		regular.AppOrderID,
	).Scan(&fulfillmentID, &fulfillmentStatus); err != nil {
		t.Fatalf("select revoked vkma fulfillment: %v", err)
	}
	if fulfillmentStatus != "revoked" {
		t.Fatalf("unexpected vkma fulfillment status: %s", fulfillmentStatus)
	}
	if err := env.db.QueryRowContext(env.ctx,
		"SELECT COUNT(*), MAX(id), MAX(status) FROM payment_refund WHERE order_id = $1",
		regular.AppOrderID,
	).Scan(&refundCount, &refundID, &refundStatus); err != nil {
		t.Fatalf("select vkma refund: %v", err)
	}
	if refundCount != 1 || refundStatus != "succeeded" {
		t.Fatalf("unexpected vkma refund state: count=%d status=%s", refundCount, refundStatus)
	}
	productStats, err := env.api.Admin.GetProductStats(env.ctx, testWorkspaceID, productID)
	if err != nil {
		t.Fatalf("get refunded product stats: %v", err)
	}
	if len(productStats.Assets) != 1 || productStats.Assets[0].RefundCount != 1 ||
		productStats.Assets[0].RefundAmountMinor != 35 {
		t.Fatalf("unexpected refunded product stats: %#v", productStats)
	}
	assertVKMARefundedCallback(t, env, PaymentRefundedCallbackPayload{
		OrderID:           regular.AppOrderID,
		AttemptID:         attemptID,
		FulfillmentID:     fulfillmentID,
		RefundID:          refundID,
		WorkspaceID:       testWorkspaceID,
		AppID:             3003,
		PlatformID:        paymentvkma.PlatformID,
		PlatformUserID:    "8001",
		ProductID:         productID,
		Quantity:          1,
		ProviderCode:      paymentvkma.ProviderCode,
		ProviderPaymentID: fmt.Sprint(orderPaymentID),
		AssetCode:         paymentvkma.AssetCode,
		AmountMinor:       35,
		Rewards: []Reward{
			{Key: itemID, Type: "quantity", Quantity: 1},
		},
	})

	if _, err := env.api.Adapters.VKMA.HandleRequest(env.ctx, refundRequest); err != nil {
		t.Fatalf("vkma duplicate regular refund: %v", err)
	}
	if err := env.db.QueryRowContext(env.ctx,
		"SELECT COUNT(*) FROM payment_refund WHERE order_id = $1",
		regular.AppOrderID,
	).Scan(&refundCount); err != nil {
		t.Fatalf("count duplicate vkma refund: %v", err)
	}
	if refundCount != 1 {
		t.Fatalf("expected one idempotent vkma refund, got %d", refundCount)
	}
	productStats, err = env.api.Admin.GetProductStats(env.ctx, testWorkspaceID, productID)
	if err != nil {
		t.Fatalf("get duplicate refunded product stats: %v", err)
	}
	if len(productStats.Assets) != 1 || productStats.Assets[0].RefundCount != 1 {
		t.Fatalf("duplicate vkma refund created extra stats: %#v", productStats.Assets)
	}
	now := time.Now()
	overview, err := env.api.Admin.ListDailyOverview(
		env.ctx, testWorkspaceID, now.Add(-24*time.Hour), now.Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("list refunded payment daily overview: %v", err)
	}
	if len(overview) != 1 || overview[0].RefundedOrders != 1 || overview[0].RefundCount != 1 {
		t.Fatalf("unexpected refunded payment daily overview: %#v", overview)
	}

	// VKMA subscription lookup.
	// Request subscription product information through get_subscription.
	// Verify VK receives the subscription duration as expiration.
	subscriptionItem, err := env.api.Adapters.VKMA.GetSubscriptionForWorkspace(env.ctx, testWorkspaceID, vkmashop.Params{
		NotificationType: vkmashop.GetSubscription,
		AppID:            3003,
		UserID:           8002,
		Item:             productID,
		Lang:             "ru",
	})
	if err != nil {
		t.Fatalf("vkma get subscription: %v", err)
	}
	if subscriptionItem.Expiration != 2592000 {
		t.Fatalf("unexpected subscription expiration: %d", subscriptionItem.Expiration)
	}

	// VKMA subscription
	// Process a chargeable subscription_status_change notification and repeat it.
	// Verify the subscription is activated and duplicate callbacks stay idempotent.
	subscriptionOrderID := orderPaymentID + 1
	subscriptionID := orderPaymentID + 100
	created, err := env.api.Adapters.VKMA.ChargeableForWorkspace(env.ctx, testWorkspaceID, vkmashop.Params{
		NotificationType: vkmashop.SubscriptionStatusChange,
		Status:           vkmashop.Chargeable,
		AppID:            3003,
		UserID:           8002,
		Item:             productID,
		OrderID:          subscriptionOrderID,
		SubscriptionID:   subscriptionID,
		Lang:             "ru",
	})
	if err != nil {
		t.Fatalf("vkma subscription chargeable: %v", err)
	}
	if created.AppOrderID == 0 || created.OrderID != subscriptionOrderID {
		t.Fatalf("unexpected subscription vkma response: %#v", created)
	}
	assertOrderStatus(t, env.ctx, env.db, created.AppOrderID, "fulfilled")

	again, err := env.api.Adapters.VKMA.ChargeableForWorkspace(env.ctx, testWorkspaceID, vkmashop.Params{
		NotificationType: vkmashop.SubscriptionStatusChange,
		Status:           vkmashop.Chargeable,
		AppID:            3003,
		UserID:           8002,
		Item:             productID,
		OrderID:          subscriptionOrderID,
		SubscriptionID:   subscriptionID,
		Lang:             "ru",
	})
	if err != nil {
		t.Fatalf("vkma subscription chargeable again: %v", err)
	}
	if again.AppOrderID != created.AppOrderID {
		t.Fatalf("expected idempotent app order id: got %d want %d", again.AppOrderID, created.AppOrderID)
	}

	active, err := env.api.User.IsSubscriptionActive(env.ctx, subscription.IsActiveParams{
		Identity:     paymentTestIdentity(testWorkspaceID, 3003, paymentvkma.PlatformID, "8002"),
		ProductID:    productID,
		ProviderCode: paymentvkma.ProviderCode,
	})
	if err != nil {
		t.Fatalf("subscription is active: %v", err)
	}
	if !active {
		t.Fatal("expected active subscription after chargeable")
	}

	// VKMA subscription statuses.
	// Apply active, canceled, and refunded subscription notifications.
	// Verify the shared Subscription API reports active and inactive states correctly.
	if _, err := env.api.Adapters.VKMA.Active(env.ctx, vkmashop.Params{
		NotificationType: vkmashop.SubscriptionStatusChange,
		Status:           vkmashop.Active,
		AppID:            3003,
		UserID:           8002,
		SubscriptionID:   subscriptionID,
		CancelReason:     vkmashop.CancelUserDecision,
	}); err != nil {
		t.Fatalf("vkma subscription active status: %v", err)
	}

	active, err = env.api.User.IsSubscriptionActive(env.ctx, subscription.IsActiveParams{
		Identity:     paymentTestIdentity(testWorkspaceID, 3003, paymentvkma.PlatformID, "8002"),
		ProductID:    productID,
		ProviderCode: paymentvkma.ProviderCode,
	})
	if err != nil {
		t.Fatalf("subscription is active after active status: %v", err)
	}
	if !active {
		t.Fatal("expected active subscription after active status")
	}

	if _, err := env.api.Adapters.VKMA.Canceled(env.ctx, vkmashop.Params{
		NotificationType: vkmashop.SubscriptionStatusChange,
		Status:           vkmashop.Canceled,
		AppID:            3003,
		UserID:           8002,
		SubscriptionID:   subscriptionID,
		CancelReason:     vkmashop.CancelUserDecision,
	}); err != nil {
		t.Fatalf("vkma subscription canceled status: %v", err)
	}

	active, err = env.api.User.IsSubscriptionActive(env.ctx, subscription.IsActiveParams{
		Identity:     paymentTestIdentity(testWorkspaceID, 3003, paymentvkma.PlatformID, "8002"),
		ProductID:    productID,
		ProviderCode: paymentvkma.ProviderCode,
	})
	if err != nil {
		t.Fatalf("subscription is active after cancel: %v", err)
	}
	if active {
		t.Fatal("expected inactive subscription after cancel")
	}

	if _, err := env.api.Adapters.VKMA.Refunded(env.ctx, vkmashop.Params{
		NotificationType: vkmashop.SubscriptionStatusChange,
		Status:           vkmashop.Refunded,
		AppID:            3003,
		UserID:           8002,
		SubscriptionID:   subscriptionID,
	}); err != nil {
		t.Fatalf("vkma subscription refunded status: %v", err)
	}
}

func assertVKMARefundedCallback(t *testing.T, env paymentTestEnv, want PaymentRefundedCallbackPayload) {
	t.Helper()

	ctx, cancel := context.WithCancel(env.ctx)
	handled := 0
	err := env.api.OnCallback(ctx, func(callback Context) error {
		handled++
		if callback.EventType != CallbackEventPaymentOrderRefunded {
			t.Fatalf("unexpected callback event type: %s", callback.EventType)
		}
		if callback.EventKey != fmt.Sprintf("%s:%d", CallbackEventPaymentOrderRefunded, want.OrderID) {
			t.Fatalf("unexpected refunded callback event key: %s", callback.EventKey)
		}
		if callback.PaymentRefunded == nil {
			t.Fatal("expected payment refunded callback payload")
		}
		if got := *callback.PaymentRefunded; !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected payment refunded callback payload: got %#v want %#v", got, want)
		}
		if err := callback.Successful(); err != nil {
			return err
		}
		cancel()
		return nil
	}, WithCallbackBatchSize(1), WithCallbackIdleDelay(time.Millisecond))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("on refunded callback: %v", err)
	}
	if handled != 1 {
		t.Fatalf("unexpected refunded callback handled count: got %d want 1", handled)
	}
}

func TestVKMAAdapterUsesRequestWorkspace(t *testing.T) {
	env := setupPaymentIntegrationTest(t)
	productID, _ := createVKMAProduct(t, env)

	response, err := env.api.Adapters.VKMA.HandleRequest(env.ctx, paymentvkma.Request{
		WorkspaceID: testWorkspaceID,
		Params: vkmashop.Params{
			NotificationType: vkmashop.GetItem,
			AppID:            3003,
			UserID:           8001,
			Item:             productID,
			Lang:             "ru",
		},
	})
	if err != nil {
		t.Fatalf("vkma get item with request workspace: %v", err)
	}
	item, ok := response.(*paymentvkma.ItemResponse)
	if !ok {
		t.Fatalf("unexpected vkma response type: %T", response)
	}
	if item.ItemID != productID || item.Price != 35 {
		t.Fatalf("unexpected vkma item response: %#v", item)
	}
}

func createVKMAProduct(t *testing.T, env paymentTestEnv) (string, string) {
	t.Helper()

	productID := fmt.Sprintf("vkma_product_%d", time.Now().UnixNano())
	groupCode := fmt.Sprintf("vkma_group_%d", time.Now().UnixNano())
	itemID := fmt.Sprintf("vkma_item_%d", time.Now().UnixNano())
	productTitleKey := productID + ".title"
	productDescriptionKey := productID + ".description"
	itemTitleKey := itemID + ".title"
	itemDescriptionKey := itemID + ".description"
	now := time.Now()
	availableFrom := now.Add(-time.Hour)
	availableUntil := now.Add(time.Hour)
	priceStartsAt := now.Add(-time.Hour)
	priceEndsAt := now.Add(time.Hour)
	periodSeconds := int64(2592000)

	if err := env.api.Admin.SaveProductGroup(env.ctx, product.UpsertGroupParams{
		WorkspaceID:    testWorkspaceID,
		Code:           groupCode,
		TitleKey:       utils.Ref(groupCode + ".title"),
		DescriptionKey: utils.Ref(groupCode + ".description"),
		Position:       1,
		IsActive:       true,
	}); err != nil {
		t.Fatalf("upsert vkma product group: %v", err)
	}

	if err := env.api.Admin.SaveProduct(env.ctx, product.UpsertParams{
		WorkspaceID:    testWorkspaceID,
		ID:             productID,
		GroupCode:      utils.Ref(groupCode),
		TitleKey:       productTitleKey,
		DescriptionKey: utils.Ref(productDescriptionKey),
		PeriodSeconds:  &periodSeconds,
		Position:       1,
		GlobalInterval: "UNLIMITED",
		UserInterval:   "UNLIMITED",
		AvailableFrom:  &availableFrom,
		AvailableUntil: &availableUntil,
		IsVisible:      true,
	}); err != nil {
		t.Fatalf("create vkma product: %v", err)
	}

	for _, localization := range []product.UpsertLocalizationParams{
		{WorkspaceID: testWorkspaceID, Locale: "ru", LocalizationKey: productTitleKey, Value: "VKMA подписка"},
		{WorkspaceID: testWorkspaceID, Locale: "ru", LocalizationKey: productDescriptionKey, Value: "VKMA описание"},
		{WorkspaceID: testWorkspaceID, Locale: "ru", LocalizationKey: itemTitleKey, Value: "VKMA premium"},
		{WorkspaceID: testWorkspaceID, Locale: "ru", LocalizationKey: itemDescriptionKey, Value: "VKMA premium description"},
	} {
		if err := env.api.Admin.SaveLocalization(env.ctx, localization); err != nil {
			t.Fatalf("upsert vkma localization: %v", err)
		}
	}

	if err := env.api.Admin.AttachProductItem(env.ctx, product.AddItemParams{
		WorkspaceID: testWorkspaceID,
		ProductID:   productID,
		ItemID:      itemID,
		Quantity:    1,
	}); err != nil {
		t.Fatalf("add vkma product item: %v", err)
	}

	if _, err := env.api.Admin.CreateCatalogPrice(env.ctx, product.CreatePriceParams{
		WorkspaceID:     testWorkspaceID,
		ProductID:       productID,
		AssetCode:       paymentvkma.AssetCode,
		ListAmountMinor: 35,
		StartsAt:        &priceStartsAt,
		EndsAt:          &priceEndsAt,
	}); err != nil {
		t.Fatalf("create vkma product price: %v", err)
	}

	return productID, itemID
}

func TestYooKassaAdapterFullCycle(t *testing.T) {

	// Test database setup.
	// Create a RUB product and configure a fake YooKassa HTTP API.
	// Verify the adapter uses the shared payment database and provider catalog.
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "yookassa_product",
		AssetCode:       yookassa.AssetCode,
		ListAmountMinor: 1299,
	})

	var requestSeen bool
	var refundSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/refunds" {
			refundSeen = true
			if r.Header.Get("Idempotence-Key") == "" {
				t.Fatal("expected yookassa refund idempotence key")
			}
			var body struct {
				PaymentID string `json:"payment_id"`
				Amount    struct {
					Value    string `json:"value"`
					Currency string `json:"currency"`
				} `json:"amount"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode yookassa refund: %v", err)
			}
			if body.PaymentID != "yk_pay_1" || body.Amount.Value != "12.99" || body.Amount.Currency != "RUB" {
				t.Fatalf("unexpected yookassa refund body: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": "yk_refund_1",
				"status": "succeeded",
				"payment_id": "yk_pay_1",
				"amount": {"value": "12.99", "currency": "RUB"}
			}`))
			return
		}
		if r.URL.Path != "/v3/payments" {
			t.Fatalf("unexpected yookassa path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected yookassa method: %s", r.Method)
		}
		shopID, secret, ok := r.BasicAuth()
		if !ok || shopID != "shop_1" || secret != "secret_1" {
			t.Fatalf("unexpected yookassa auth: ok=%v shop=%s secret=%s", ok, shopID, secret)
		}
		if r.Header.Get("Idempotence-Key") != "idem-yookassa-1" {
			t.Fatalf("unexpected idempotence key: %s", r.Header.Get("Idempotence-Key"))
		}

		var body struct {
			Amount struct {
				Value    string `json:"value"`
				Currency string `json:"currency"`
			} `json:"amount"`
			Capture      bool `json:"capture"`
			Confirmation struct {
				Type      string `json:"type"`
				ReturnURL string `json:"return_url"`
			} `json:"confirmation"`
			PaymentMethodData struct {
				Type string `json:"type"`
			} `json:"payment_method_data"`
			Receipt struct {
				Customer struct {
					Email string `json:"email"`
				} `json:"customer"`
				Items []struct {
					Description string `json:"description"`
					Quantity    string `json:"quantity"`
					Amount      struct {
						Value    string `json:"value"`
						Currency string `json:"currency"`
					} `json:"amount"`
					VATCode        int    `json:"vat_code"`
					PaymentMode    string `json:"payment_mode"`
					PaymentSubject string `json:"payment_subject"`
				} `json:"items"`
			} `json:"receipt"`
			Metadata map[string]string `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode yookassa request: %v", err)
		}
		if body.Amount.Value != "12.99" || body.Amount.Currency != "RUB" {
			t.Fatalf("unexpected yookassa amount: %#v", body.Amount)
		}
		if !body.Capture {
			t.Fatal("expected yookassa capture=true")
		}
		if body.Confirmation.Type != "redirect" || body.Confirmation.ReturnURL != "https://example.com/return" {
			t.Fatalf("unexpected yookassa confirmation: %#v", body.Confirmation)
		}
		if body.PaymentMethodData.Type != string(yookassa.PaymentMethodSBP) {
			t.Fatalf("unexpected yookassa payment method: %#v", body.PaymentMethodData)
		}
		if body.Receipt.Customer.Email != "buyer@example.com" {
			t.Fatalf("unexpected yookassa receipt customer: %#v", body.Receipt.Customer)
		}
		if len(body.Receipt.Items) != 1 || body.Receipt.Items[0].Description != "Elum Love Premium" || body.Receipt.Items[0].Amount.Value != "12.99" {
			t.Fatalf("unexpected yookassa receipt items: %#v", body.Receipt.Items)
		}
		if body.Metadata["product_id"] != productID || body.Metadata["workspace_id"] != testWorkspaceID {
			t.Fatalf("unexpected yookassa metadata: %#v", body.Metadata)
		}

		requestSeen = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "yk_pay_1",
			"status": "pending",
			"paid": false,
			"amount": {"value": "12.99", "currency": "RUB"},
			"confirmation": {
				"type": "redirect",
				"confirmation_url": "https://yookassa.test/confirm/yk_pay_1"
			}
		}`))
	}))
	defer server.Close()

	credentials := yookassa.Credentials{
		ShopID:     "shop_1",
		SecretKey:  "secret_1",
		APIBaseURL: server.URL,
		HTTPClient: server.Client(),
	}

	// YooKassa payment creation.
	// Create a local order and remote YooKassa payment with redirect confirmation.
	// Verify provider identifiers and confirmation URL are stored on the attempt.
	response, err := env.api.Adapters.YooKassa.CreatePayment(env.ctx, yookassa.CreatePaymentParams{
		Credentials:       credentials,
		WorkspaceID:       testWorkspaceID,
		AppID:             5005,
		PlatformID:        1,
		PlatformUserID:    "buyer-yookassa",
		ProductID:         productID,
		Locale:            "ru",
		ReturnURL:         "https://example.com/return",
		IdempotencyKey:    "idem-yookassa-1",
		PaymentMethodType: yookassa.PaymentMethodSBP,
		Description:       "YooKassa adapter test",
		Receipt: &yookassa.Receipt{
			Customer: yookassa.ReceiptCustomer{Email: "buyer@example.com"},
			Items: []yookassa.ReceiptItem{
				{
					Description:    "Elum Love Premium",
					Quantity:       "1.00",
					Amount:         yookassa.Amount{Value: "12.99", Currency: yookassa.AssetCode},
					VATCode:        1,
					PaymentMode:    "full_payment",
					PaymentSubject: "service",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("create yookassa payment: %v", err)
	}
	if !requestSeen {
		t.Fatal("expected fake yookassa API to receive request")
	}
	if response.PaymentID != "yk_pay_1" || response.ConfirmationURL == "" {
		t.Fatalf("unexpected yookassa response: %#v", response)
	}
	if response.AmountMinor != 1299 || response.AssetCode != "RUB" {
		t.Fatalf("unexpected yookassa attempt amount: %#v", response)
	}
	if response.PaymentMethodType != yookassa.PaymentMethodSBP {
		t.Fatalf("unexpected yookassa response payment method: %#v", response)
	}
	assertOrderStatus(t, env.ctx, env.db, response.OrderID, "pending_payment")
	assertAttemptStatus(t, env.ctx, env.db, response.AttemptID, "pending")

	// YooKassa payment webhook.
	// Process payment.succeeded and repeat the same notification.
	// Verify fulfillment is created once and duplicate webhook remains idempotent.
	webhook := []byte(`{
		"type": "notification",
		"event": "payment.succeeded",
		"object": {
			"id": "yk_pay_1",
			"status": "succeeded",
			"paid": true,
			"amount": {"value": "12.99", "currency": "RUB"}
		}
	}`)
	completed, err := env.api.Adapters.YooKassa.HandleWebhook(env.ctx, webhook, true)
	if err != nil {
		t.Fatalf("handle yookassa webhook: %v", err)
	}
	if completed.FulfilledID == nil {
		t.Fatal("expected yookassa fulfillment id")
	}
	assertOrderStatus(t, env.ctx, env.db, response.OrderID, "fulfilled")
	assertAttemptStatus(t, env.ctx, env.db, response.AttemptID, "succeeded")
	assertFulfillmentItemCount(t, env.ctx, env.db, *completed.FulfilledID, 1)

	again, err := env.api.Adapters.YooKassa.HandleWebhook(env.ctx, webhook, true)
	if err != nil {
		t.Fatalf("handle yookassa webhook again: %v", err)
	}
	if !again.AlreadyDone {
		t.Fatal("expected duplicate yookassa webhook to be idempotent")
	}

	refunded, err := env.api.Admin.ExecuteRefund(env.ctx, paymentrefund.Params{
		WorkspaceID:    testWorkspaceID,
		OrderID:        response.OrderID,
		AttemptID:      response.AttemptID,
		Reason:         "test yookassa refund",
		ProviderParams: credentials,
	})
	if err != nil {
		t.Fatalf("execute yookassa refund: %v", err)
	}
	if !refundSeen {
		t.Fatal("expected yookassa refund request")
	}
	if refunded.ProviderRefundID == nil || *refunded.ProviderRefundID != "yk_refund_1" || refunded.Status != "succeeded" {
		t.Fatalf("unexpected yookassa refund result: %#v", refunded)
	}
	assertOrderStatus(t, env.ctx, env.db, response.OrderID, "refunded")
	assertAttemptStatus(t, env.ctx, env.db, response.AttemptID, "refunded")
	assertRefundStatus(t, env.ctx, env.db, refunded.RefundID, "succeeded")
}

func TestYooKassaAdapterRejectsWrongWebhookAmount(t *testing.T) {

	// Test database setup.
	// Create a YooKassa payment through a fake API.
	// Verify webhook amount mismatch cannot fulfill the order.
	env := setupPaymentIntegrationTest(t)
	productID := createPaymentProduct(t, env, testProductOptions{
		ProductID:       "yookassa_wrong_amount_product",
		AssetCode:       yookassa.AssetCode,
		ListAmountMinor: 500,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "yk_pay_wrong_amount",
			"status": "pending",
			"paid": false,
			"amount": {"value": "5.00", "currency": "RUB"},
			"confirmation": {
				"type": "redirect",
				"confirmation_url": "https://yookassa.test/confirm/yk_pay_wrong_amount"
			}
		}`))
	}))
	defer server.Close()

	credentials := yookassa.Credentials{
		ShopID:     "shop_1",
		SecretKey:  "secret_1",
		APIBaseURL: server.URL,
		HTTPClient: server.Client(),
	}

	response, err := env.api.Adapters.YooKassa.CreatePayment(env.ctx, yookassa.CreatePaymentParams{
		Credentials:    credentials,
		WorkspaceID:    testWorkspaceID,
		AppID:          5005,
		PlatformID:     1,
		PlatformUserID: "buyer-yookassa-wrong",
		ProductID:      productID,
		Locale:         "ru",
		ReturnURL:      "https://example.com/return",
		IdempotencyKey: "idem-yookassa-wrong-amount-" + time.Now().Format("150405.000000000"),
	})
	if err != nil {
		t.Fatalf("create yookassa payment: %v", err)
	}

	_, err = env.api.Adapters.YooKassa.HandleWebhook(env.ctx, []byte(`{
		"type": "notification",
		"event": "payment.succeeded",
		"object": {
			"id": "yk_pay_wrong_amount",
			"status": "succeeded",
			"paid": true,
			"amount": {"value": "5.01", "currency": "RUB"}
		}
	}`), true)
	if err == nil {
		t.Fatal("expected yookassa wrong amount webhook to fail")
	}
	assertOrderStatus(t, env.ctx, env.db, response.OrderID, "pending_payment")
}

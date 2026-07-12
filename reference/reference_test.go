package reference

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/reference/repository"
	"github.com/elum-utils/services/reference/service/admin"
	"github.com/elum-utils/services/reference/service/user"
	_ "github.com/jackc/pgx/v5/stdlib"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIsReady(t *testing.T) {
	var nilService *Reference
	if nilService.IsReady() {
		t.Fatal("nil reference must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized reference must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx, service.Admin, service.User = ctx, &admin.Admin{}, &user.User{}
	if !service.IsReady() {
		t.Fatal("initialized reference must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed reference must not be ready")
	}
}

func TestReferenceCacheVersionInvalidatesOtherNode(t *testing.T) {
	cache := newReferenceSharedCache()
	options := Options{
		Cache:        cache,
		CacheEnabled: true,
		CacheL1Delay: time.Minute,
		CacheL2Delay: time.Minute,
	}
	nodeA := newReferenceTestServiceWithOptions(t, referenceTestDB, options)
	db, err := openReferencePostgres(referenceTestDB)
	if err != nil {
		t.Fatalf("open second reference node database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	nodeB, err := NewWithDatabase(context.Background(), db, referenceTestOptions(options))
	if err != nil {
		t.Fatalf("create second reference node: %v", err)
	}
	t.Cleanup(func() { _ = nodeB.Close() })

	if err := nodeA.Admin.CreateItem(context.Background(), admin.SaveItemParams{
		WorkspaceID: "cache-workspace",
		Key:         "stars",
		Type:        repository.ItemTypeQuantity,
		Payload:     json.RawMessage(`{"version":1}`),
		IsActive:    true,
	}); err != nil {
		t.Fatalf("create cached reference item: %v", err)
	}
	if err := nodeA.Admin.UpsertLocalization(context.Background(), admin.SaveLocalizationParams{
		WorkspaceID: "cache-workspace",
		ItemKey:     "stars",
		Locale:      "ru",
		Title:       "Old title",
	}); err != nil {
		t.Fatalf("create cached reference localization: %v", err)
	}

	warmReferenceReads(t, nodeB, "Old title", 1)

	if _, err := nodeA.Admin.UpdateItem(context.Background(), admin.UpdateItemParams{
		WorkspaceID: "cache-workspace",
		Key:         "stars",
		Payload:     json.RawMessage(`{"version":2}`),
		IsActive:    true,
	}); err != nil {
		t.Fatalf("update cached reference item: %v", err)
	}
	if err := nodeA.Admin.UpsertLocalization(context.Background(), admin.SaveLocalizationParams{
		WorkspaceID: "cache-workspace",
		ItemKey:     "stars",
		Locale:      "ru",
		Title:       "New title",
	}); err != nil {
		t.Fatalf("update cached reference localization: %v", err)
	}

	warmReferenceReads(t, nodeB, "New title", 2)
}

func TestReferenceImportBatchesLargePackage(t *testing.T) {
	service := newReferenceTestService(t)
	const itemCount = 12001
	items := make([]repository.ExportItem, 0, itemCount)
	for index := 0; index < itemCount; index++ {
		items = append(items, repository.ExportItem{
			Key:      fmt.Sprintf("large.item.%05d", index),
			Type:     repository.ItemTypeQuantity,
			Payload:  json.RawMessage(`{}`),
			IsActive: true,
		})
	}

	result, err := service.Admin.Import(context.Background(), "large-workspace", admin.ImportRequest{
		Package: admin.ExportPackage{
			Format:  repository.ExportFormat,
			Service: "reference",
			Items:   items,
		},
		ConflictStrategy: repository.ImportConflictUpdate,
	})
	if err != nil {
		t.Fatalf("import large reference package: %v", err)
	}
	if result.Imported.Items != itemCount {
		t.Fatalf("imported items = %d, want %d", result.Imported.Items, itemCount)
	}
}

func TestReferenceImportSerializesWithAdminWrite(t *testing.T) {
	service := newReferenceTestService(t)
	db, err := openReferencePostgres(referenceTestDB)
	if err != nil {
		t.Fatalf("open reference lock database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	workspaceID := "concurrent-workspace"

	transaction, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin reference lock transaction: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback() })
	if _, err := transaction.ExecContext(
		ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		"reference:"+workspaceID,
	); err != nil {
		t.Fatalf("lock reference workspace: %v", err)
	}

	importResult := make(chan error, 1)
	go func() {
		_, err := service.Admin.Import(ctx, workspaceID, admin.ImportRequest{
			Package: admin.ExportPackage{
				Format:  repository.ExportFormat,
				Service: "reference",
				Items: []repository.ExportItem{
					{Key: "import.item", Type: repository.ItemTypeQuantity, Payload: json.RawMessage(`{}`), IsActive: true},
				},
			},
			ConflictStrategy: repository.ImportConflictUpdate,
		})
		importResult <- err
	}()
	waitForReferenceWorkspaceLock(t, db, 1)

	adminResult := make(chan error, 1)
	go func() {
		adminResult <- service.Admin.CreateItem(ctx, admin.SaveItemParams{
			WorkspaceID: workspaceID,
			Key:         "admin.item",
			Type:        repository.ItemTypeQuantity,
			Payload:     json.RawMessage(`{}`),
			IsActive:    true,
		})
	}()
	waitForReferenceWorkspaceLock(t, db, 2)

	if err := transaction.Commit(); err != nil {
		t.Fatalf("release reference workspace lock: %v", err)
	}
	if err := <-importResult; err != nil {
		t.Fatalf("concurrent reference import: %v", err)
	}
	if err := <-adminResult; err != nil {
		t.Fatalf("concurrent reference admin write: %v", err)
	}

	items, err := service.User.Resolve(ctx, user.ResolveParams{
		WorkspaceID: workspaceID,
		Keys:        []string{"import.item", "admin.item"},
	})
	if err != nil || len(items.Items) != 2 {
		t.Fatalf("concurrent reference result: items=%+v err=%v", items, err)
	}
}

func waitForReferenceWorkspaceLock(t *testing.T, db interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, minimum int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting int
		if err := db.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM pg_stat_activity
WHERE datname = current_database()
  AND wait_event_type = 'Lock'
  AND query LIKE '%pg_advisory_xact_lock%'`).Scan(&waiting); err != nil {
			t.Fatalf("inspect reference lock waiters: %v", err)
		}
		if waiting >= minimum {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("reference lock waiters = %d, want at least %d", waiting, minimum)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func warmReferenceReads(t *testing.T, service *Reference, title string, version int) {
	t.Helper()
	ctx := context.Background()
	item, err := service.User.Get(ctx, user.GetParams{
		WorkspaceID: "cache-workspace",
		Key:         "stars",
		Locale:      "ru",
	})
	if err != nil || item.Localization == nil || item.Localization.Title != title ||
		referencePayloadVersion(item.Payload) != version {
		t.Fatalf("reference Get returned stale data: item=%+v err=%v", item, err)
	}
	resolved, err := service.User.Resolve(ctx, user.ResolveParams{
		WorkspaceID: "cache-workspace",
		Keys:        []string{"stars"},
		Locale:      "ru",
	})
	if err != nil || len(resolved.Items) != 1 || resolved.Items[0].Localization == nil ||
		resolved.Items[0].Localization.Title != title || referencePayloadVersion(resolved.Items[0].Payload) != version {
		t.Fatalf("reference Resolve returned stale data: result=%+v err=%v", resolved, err)
	}
	adminItem, err := service.Admin.GetItem(ctx, "cache-workspace", "stars")
	if err != nil || adminItem.Localizations[0].Title != title || referencePayloadVersion(adminItem.Payload) != version {
		t.Fatalf("reference admin GetItem returned stale data: item=%+v err=%v", adminItem, err)
	}
}

func referencePayloadVersion(payload json.RawMessage) int {
	var value struct {
		Version int `json:"version"`
	}
	if json.Unmarshal(payload, &value) != nil {
		return 0
	}
	return value.Version
}

type referenceSharedCacheEntry struct {
	value     []byte
	expiresAt time.Time
}

type referenceSharedCache struct {
	mu      sync.Mutex
	entries map[string]referenceSharedCacheEntry
}

func newReferenceSharedCache() *referenceSharedCache {
	return &referenceSharedCache{entries: make(map[string]referenceSharedCacheEntry)}
}

func (c *referenceSharedCache) GetWithTTL(key string) ([]byte, time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, exists := c.entries[key]
	if !exists || (!entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt)) {
		delete(c.entries, key)
		return nil, 0, nil
	}
	return append([]byte(nil), entry.value...), time.Until(entry.expiresAt), nil
}

func (c *referenceSharedCache) Set(key string, value []byte, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := referenceSharedCacheEntry{value: append([]byte(nil), value...)}
	if expiration > 0 {
		entry.expiresAt = time.Now().Add(expiration)
	}
	c.entries[key] = entry
	return nil
}

func (c *referenceSharedCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
	return nil
}

func (c *referenceSharedCache) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.entries)
	return nil
}

func (c *referenceSharedCache) Close() error { return nil }

var _ Storage = (*referenceSharedCache)(nil)

const (
	referenceTestPGHost     = "localhost"
	referenceTestPGPort     = 5432
	referenceTestPGUser     = "postgres"
	referenceTestPGPassword = "RBTX0DXKbagvCy2XCAi4qHt0cjeSD6bU"
	referenceTestDB         = "reference_test"
)

func TestReferenceFullLifecycle(t *testing.T) {
	service := newReferenceTestService(t)
	ctx := context.Background()

	if err := service.Admin.CreateItem(ctx, admin.SaveItemParams{
		WorkspaceID: "workspace-a", Key: "Coin", Type: repository.ItemTypeQuantity,
		Payload: json.RawMessage(`{"icon":"coin.png","decimals":0}`), IsActive: true,
	}); err != nil {
		t.Fatalf("create quantity item: %v", err)
	}
	if err := service.Admin.CreateItem(ctx, admin.SaveItemParams{
		WorkspaceID: "workspace-a", Key: "premium", Type: repository.ItemTypeDuration,
		Payload: json.RawMessage(`{"icon":"premium.png"}`), IsActive: true,
	}); err != nil {
		t.Fatalf("create duration item: %v", err)
	}
	for _, localization := range []admin.SaveLocalizationParams{
		{WorkspaceID: "workspace-a", ItemKey: "coin", Locale: "ru", Title: "Монеты", Description: "Игровая валюта"},
		{WorkspaceID: "workspace-a", ItemKey: "coin", Locale: "en", Title: "Coins", Description: "Game currency"},
		{WorkspaceID: "workspace-a", ItemKey: "premium", Locale: "ru", Title: "Премиум", Description: "Премиум-доступ"},
	} {
		if err := service.Admin.UpsertLocalization(ctx, localization); err != nil {
			t.Fatalf("upsert localization: %v", err)
		}
	}

	coin, err := service.User.Get(ctx, user.GetParams{
		WorkspaceID: "workspace-a", Key: " COIN ", Locale: "ru",
	})
	if err != nil {
		t.Fatalf("get coin: %v", err)
	}
	if coin.Key != "coin" || coin.Type != repository.ItemTypeQuantity ||
		coin.Localization == nil || coin.Localization.Title != "Монеты" {
		t.Fatalf("unexpected coin: %#v", coin)
	}

	resolved, err := service.User.Resolve(ctx, user.ResolveParams{
		WorkspaceID: "workspace-a",
		Keys:        []string{"premium", "missing", "coin", "PREMIUM"},
		Locale:      "ru",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved.Items) != 2 || resolved.Items[0].Key != "premium" ||
		resolved.Items[1].Key != "coin" || len(resolved.MissingKeys) != 1 ||
		resolved.MissingKeys[0] != "missing" {
		t.Fatalf("unexpected resolve result: %#v", resolved)
	}

	if _, err := service.Admin.UpdateItem(ctx, admin.UpdateItemParams{
		WorkspaceID: "workspace-a", Key: "coin",
		Payload: json.RawMessage(`{"icon":"coin-v2.png","decimals":0}`), IsActive: true,
	}); err != nil {
		t.Fatalf("update item: %v", err)
	}
	updated, err := service.User.Get(ctx, user.GetParams{
		WorkspaceID: "workspace-a", Key: "coin", Locale: "ru",
	})
	if err != nil || !strings.Contains(string(updated.Payload), "coin-v2.png") {
		t.Fatalf("updated cached item: %#v err=%v", updated, err)
	}

	adminItem, err := service.Admin.GetItem(ctx, "workspace-a", "coin")
	if err != nil || len(adminItem.Localizations) != 2 {
		t.Fatalf("admin item: %#v err=%v", adminItem, err)
	}
	stats, err := service.Admin.GetStats(ctx, "workspace-a")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.ItemsTotal != 2 || stats.ActiveItems != 2 ||
		stats.QuantityItems != 1 || stats.DurationItems != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}

	if err := service.Admin.CreateItem(ctx, admin.SaveItemParams{
		WorkspaceID: "workspace-b", Key: "coin", Type: repository.ItemTypeDuration,
		Payload: json.RawMessage(`{"workspace":"b"}`), IsActive: true,
	}); err != nil {
		t.Fatalf("create isolated item: %v", err)
	}
	isolated, err := service.User.Get(ctx, user.GetParams{
		WorkspaceID: "workspace-b", Key: "coin", Locale: "ru",
	})
	if err != nil || isolated.Type != repository.ItemTypeDuration {
		t.Fatalf("workspace isolation: %#v err=%v", isolated, err)
	}

	if _, err := service.Admin.SoftDeleteItem(ctx, "workspace-a", "coin"); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if _, err := service.User.Get(ctx, user.GetParams{
		WorkspaceID: "workspace-a", Key: "coin", Locale: "ru",
	}); !errors.Is(err, repository.ErrItemNotFound) {
		t.Fatalf("deleted item must be hidden, err=%v", err)
	}
	if _, err := service.Admin.RestoreItem(ctx, "workspace-a", "coin", true); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := service.User.Get(ctx, user.GetParams{
		WorkspaceID: "workspace-a", Key: "coin", Locale: "ru",
	}); err != nil {
		t.Fatalf("restored item: %v", err)
	}
}

func TestReferenceImportExportCycle(t *testing.T) {
	service := newReferenceTestService(t)
	ctx := context.Background()
	if err := service.Admin.CreateItem(ctx, admin.SaveItemParams{
		WorkspaceID: "workspace-export", Key: "coin", Type: repository.ItemTypeQuantity,
		Payload: json.RawMessage(`{"icon":"coin.png","scale":2}`), IsActive: true,
	}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := service.Admin.UpsertLocalization(ctx, admin.SaveLocalizationParams{
		WorkspaceID: "workspace-export", ItemKey: "coin", Locale: "ru",
		Title: "Монеты", Description: "Игровая валюта",
	}); err != nil {
		t.Fatalf("upsert localization: %v", err)
	}
	pkg, err := service.Admin.Export(ctx, "workspace-export", admin.ExportRequest{})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	preview, err := service.Admin.PreviewImport(ctx, "workspace-import", pkg)
	if err != nil {
		t.Fatalf("preview import: %v", err)
	}
	if preview.Counts.Items != 1 || preview.Counts.Localizations != 1 || len(preview.Conflicts) != 0 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	result, err := service.Admin.Import(ctx, "workspace-import", admin.ImportRequest{
		Package: pkg, ConflictStrategy: repository.ImportConflictUpdate,
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.Imported.Items != 1 || result.Imported.Localizations != 1 {
		t.Fatalf("unexpected import result: %+v", result)
	}
	imported, err := service.Admin.Export(ctx, "workspace-import", admin.ExportRequest{})
	if err != nil {
		t.Fatalf("export imported: %v", err)
	}
	if len(imported.Items) != 1 || imported.Items[0].Key != "coin" ||
		imported.Items[0].Localization["ru"].Title != "Монеты" ||
		!strings.Contains(string(imported.Items[0].Payload), "coin.png") {
		t.Fatalf("unexpected imported package: %+v", imported)
	}
}

func TestReferenceImmutableKeyAndDangerousTypeChange(t *testing.T) {
	service := newReferenceTestService(t)
	ctx := context.Background()
	if err := service.Admin.CreateItem(ctx, admin.SaveItemParams{
		WorkspaceID: "workspace", Key: "fixed-key", Type: repository.ItemTypeQuantity,
		Payload: json.RawMessage(`{}`), IsActive: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Admin.UpsertLocalization(ctx, admin.SaveLocalizationParams{
		WorkspaceID: "workspace", ItemKey: "fixed-key", Locale: "en", Title: "Fixed",
	}); err != nil {
		t.Fatal(err)
	}

	params := admin.DangerousChangeTypeParams{
		WorkspaceID: "workspace", Key: "fixed-key",
		CurrentType: repository.ItemTypeQuantity, NewType: repository.ItemTypeDuration,
	}
	if _, err := service.Admin.DangerousChangeType(ctx, params); !errors.Is(err, admin.ErrTypeChangeNotConfirmed) {
		t.Fatalf("unconfirmed change: %v", err)
	}
	params.Confirmation = admin.DangerousTypeConfirmation
	rows, err := service.Admin.DangerousChangeType(ctx, params)
	if err != nil || rows != 1 {
		t.Fatalf("dangerous type change: rows=%d err=%v", rows, err)
	}
	item, err := service.User.Get(ctx, user.GetParams{
		WorkspaceID: "workspace", Key: "fixed-key", Locale: "en",
	})
	if err != nil || item.Type != repository.ItemTypeDuration ||
		item.Localization == nil || item.Localization.Title != "Fixed" {
		t.Fatalf("changed item: %#v err=%v", item, err)
	}

	params.CurrentType = repository.ItemTypeQuantity
	rows, err = service.Admin.DangerousChangeType(ctx, params)
	if err != nil || rows != 0 {
		t.Fatalf("stale expected type must not update: rows=%d err=%v", rows, err)
	}

}

func TestReferenceValidationAndContext(t *testing.T) {
	service := newReferenceTestService(t)
	ctx := context.Background()
	cases := []admin.SaveItemParams{
		{WorkspaceID: "", Key: "coin", Type: repository.ItemTypeQuantity, Payload: json.RawMessage(`{}`)},
		{WorkspaceID: "workspace", Key: "bad key", Type: repository.ItemTypeQuantity, Payload: json.RawMessage(`{}`)},
		{WorkspaceID: "workspace", Key: "coin", Type: "unknown", Payload: json.RawMessage(`{}`)},
		{WorkspaceID: "workspace", Key: "coin", Type: repository.ItemTypeQuantity, Payload: json.RawMessage(`{`)},
	}
	for _, params := range cases {
		if err := service.Admin.CreateItem(ctx, params); err == nil {
			t.Fatalf("expected validation error for %#v", params)
		}
	}
	if _, err := service.User.Resolve(ctx, user.ResolveParams{WorkspaceID: "workspace"}); !errors.Is(err, user.ErrKeysRequired) {
		t.Fatalf("empty resolve: %v", err)
	}
	tooMany := make([]string, 1001)
	for index := range tooMany {
		tooMany[index] = fmt.Sprintf("item.%d", index)
	}
	if _, err := service.User.Resolve(ctx, user.ResolveParams{
		WorkspaceID: "workspace", Keys: tooMany,
	}); !errors.Is(err, user.ErrTooManyKeys) {
		t.Fatalf("oversized resolve: %v", err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.User.List(canceled, user.ListParams{WorkspaceID: "workspace", Locale: "en", Page: user.Page{}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request: %v", err)
	}
}

func TestReferenceOpenBootstrapsSchema(t *testing.T) {
	const database = "reference_open_test"
	ctx := context.Background()
	adminDB, err := openReferencePostgres("")
	if err != nil {
		t.Fatalf("open admin postgres: %v", err)
	}
	terminateReferenceConnections(ctx, t, adminDB, database)
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", database)); err != nil {
		t.Fatalf("drop database: %v", err)
	}
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", database)); err != nil {
		t.Fatalf("create database: %v", err)
	}
	_ = adminDB.Close()
	t.Cleanup(func() {
		adminDB, err := openReferencePostgres("")
		if err == nil {
			terminateReferenceConnections(context.Background(), t, adminDB, database)
			_, _ = adminDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", database))
			_ = adminDB.Close()
		}
	})

	db, err := openReferencePostgres(database)
	if err != nil {
		t.Fatalf("open reference database: %v", err)
	}
	defer db.Close()
	client, err := sqlwrap.New(db)
	if err != nil {
		t.Fatalf("create reference sql client: %v", err)
	}
	repo := repository.New(client)
	if err := repo.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrap reference: %v", err)
	}
	defer repo.Close()
	service, err := NewWithDatabase(ctx, db, referenceTestOptions(Options{CacheSize: 100}))
	if err != nil {
		t.Fatalf("create reference: %v", err)
	}
	defer service.Close()
	if err := service.Admin.CreateItem(ctx, admin.SaveItemParams{
		WorkspaceID: "workspace", Key: "coin", Type: repository.ItemTypeQuantity,
		Payload: json.RawMessage(`{}`), IsActive: true,
	}); err != nil {
		t.Fatalf("schema was not bootstrapped: %v", err)
	}
}

func newReferenceTestService(t testing.TB) *Reference {
	return newReferenceTestServiceWithOptions(t, referenceTestDB, Options{})
}

func newReferenceTestServiceWithOptions(t testing.TB, database string, options Options) *Reference {
	t.Helper()
	ctx := context.Background()
	adminDB, err := openReferencePostgres("")
	if err != nil {
		t.Fatalf("open admin postgres: %v", err)
	}
	terminateReferenceConnections(ctx, t, adminDB, database)
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", database)); err != nil {
		t.Fatalf("drop database: %v", err)
	}
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", database)); err != nil {
		t.Fatalf("create database: %v", err)
	}
	_ = adminDB.Close()

	db, err := openReferencePostgres(database)
	if err != nil {
		t.Fatalf("open app postgres: %v", err)
	}
	client, err := sqlwrap.New(db, sqlwrap.Options{
		CacheEnabled: true, CacheSize: 10000, CacheTTLCheck: time.Minute,
	})
	if err != nil {
		t.Fatalf("create sql client: %v", err)
	}
	repo := repository.New(client)
	if err := repo.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrap reference: %v", err)
	}
	service, err := NewWithDatabase(ctx, db, referenceTestOptions(options))
	if err != nil {
		t.Fatalf("create reference service: %v", err)
	}
	t.Cleanup(func() {
		_ = service.Close()
		_ = repo.Close()
		_ = client.Close()
	})
	return service
}

func referenceTestOptions(options Options) Options {
	options.CacheEnabled = true
	if options.CacheSize == 0 {
		options.CacheSize = 10000
	}
	if options.CacheTTLCheck == 0 {
		options.CacheTTLCheck = time.Minute
	}
	if options.CacheL1Delay == 0 {
		options.CacheL1Delay = time.Minute
	}
	return options
}

func terminateReferenceConnections(ctx context.Context, t testing.TB, db *sql.DB, database string) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = $1 AND pid <> pg_backend_pid()`, database)
	if err != nil {
		t.Fatalf("terminate postgres connections: %v", err)
	}
}

func openReferencePostgres(database string) (*sql.DB, error) {
	if database == "" {
		database = "postgres"
	}
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		referenceTestPGHost,
		referenceTestPGPort,
		referenceTestPGUser,
		referenceTestPGPassword,
		database,
	)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

package promo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elum-utils/services/internal/testsupport"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/elum-utils/services/promo/repository"
	"github.com/elum-utils/services/promo/service/admin"
	"github.com/elum-utils/services/promo/service/user"
	_ "github.com/jackc/pgx/v5/stdlib"
	"sync"
	"testing"
	"time"
)

func TestIsReady(t *testing.T) {
	var nilService *Promo
	if nilService.IsReady() {
		t.Fatal("nil promo must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized promo must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx, service.Admin, service.User = ctx, &admin.Admin{}, &user.User{}
	if !service.IsReady() {
		t.Fatal("initialized promo must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed promo must not be ready")
	}
}

func TestPromoCacheVersionInvalidatesOtherNode(t *testing.T) {
	cache := testsupport.NewCache()
	options := promoTestOptions()
	options.Cache = cache
	options.CacheL2Delay = time.Minute
	nodeA := newPromoTestServiceWithOptions(t, options)
	db, err := openPromoPostgres(promoTestDB)
	if err != nil {
		t.Fatalf("open second promo node database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	nodeB, err := NewWithDatabase(context.Background(), db, options)
	if err != nil {
		t.Fatalf("create second promo node: %v", err)
	}
	t.Cleanup(func() { _ = nodeB.Close() })

	promoID, err := nodeA.Admin.CreatePromo(context.Background(), admin.SavePromoParams{
		WorkspaceID: "cache-workspace",
		Code:        "CACHE",
		Payload:     json.RawMessage(`{"version":1}`),
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("create cached promo: %v", err)
	}
	if err := nodeA.Admin.UpsertLocalization(context.Background(), admin.SaveLocalizationParams{
		WorkspaceID: "cache-workspace",
		PromoID:     promoID,
		Locale:      "ru",
		Title:       "Old title",
	}); err != nil {
		t.Fatalf("create cached promo localization: %v", err)
	}
	if err := nodeA.Admin.UpsertReward(context.Background(), admin.SaveRewardParams{
		WorkspaceID: "cache-workspace",
		PromoID:     promoID,
		Key:         "stars",
		Quantity:    1,
	}); err != nil {
		t.Fatalf("create cached promo reward: %v", err)
	}
	assertPromoCacheRead(t, nodeB, promoID, "Old title", 1)

	if _, err := nodeA.Admin.UpdatePromo(context.Background(), admin.SavePromoParams{
		ID:          promoID,
		WorkspaceID: "cache-workspace",
		Code:        "CACHE",
		Payload:     json.RawMessage(`{"version":2}`),
		IsActive:    true,
	}); err != nil {
		t.Fatalf("update cached promo: %v", err)
	}
	if err := nodeA.Admin.UpsertLocalization(context.Background(), admin.SaveLocalizationParams{
		WorkspaceID: "cache-workspace",
		PromoID:     promoID,
		Locale:      "ru",
		Title:       "New title",
	}); err != nil {
		t.Fatalf("update cached promo localization: %v", err)
	}
	if err := nodeA.Admin.UpsertReward(context.Background(), admin.SaveRewardParams{
		WorkspaceID: "cache-workspace",
		PromoID:     promoID,
		Key:         "stars",
		Quantity:    2,
	}); err != nil {
		t.Fatalf("update cached promo reward: %v", err)
	}
	assertPromoCacheRead(t, nodeB, promoID, "New title", 2)
}

func TestPromoImportBatchesMoreThanPostgresParameterLimit(t *testing.T) {
	service := newPromoTestService(t)
	const promoCount = 6667
	promos := make([]repository.ExportPromo, 0, promoCount)
	for index := 0; index < promoCount; index++ {
		promos = append(promos, repository.ExportPromo{
			Code:     fmt.Sprintf("LARGE%05d", index),
			Payload:  json.RawMessage(`{}`),
			IsActive: true,
		})
	}

	result, err := service.Admin.Import(context.Background(), "large-workspace", admin.ImportRequest{
		Package: admin.ExportPackage{
			Format:  repository.ExportFormat,
			Service: "promo",
			Promos:  promos,
		},
		ConflictStrategy: repository.ImportConflictUpdate,
	})
	if err != nil {
		t.Fatalf("import large promo package: %v", err)
	}
	if result.Imported.Promos != promoCount {
		t.Fatalf("imported promos = %d, want %d", result.Imported.Promos, promoCount)
	}
}

func TestPromoImportSerializesWithAdminWrite(t *testing.T) {
	service := newPromoTestService(t)
	db, err := openPromoPostgres(promoTestDB)
	if err != nil {
		t.Fatalf("open promo lock database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	workspaceID := "concurrent-workspace"

	transaction, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin promo lock transaction: %v", err)
	}
	t.Cleanup(func() { _ = transaction.Rollback() })
	if _, err := transaction.ExecContext(
		ctx,
		"SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
		"promo:"+workspaceID,
	); err != nil {
		t.Fatalf("lock promo workspace: %v", err)
	}

	importResult := make(chan error, 1)
	go func() {
		_, err := service.Admin.Import(ctx, workspaceID, admin.ImportRequest{
			Package: admin.ExportPackage{
				Format:  repository.ExportFormat,
				Service: "promo",
				Promos: []repository.ExportPromo{
					{Code: "IMPORT", Payload: json.RawMessage(`{}`), IsActive: true},
				},
			},
			ConflictStrategy: repository.ImportConflictUpdate,
		})
		importResult <- err
	}()
	waitForPromoWorkspaceLock(t, db, 1)

	adminResult := make(chan error, 1)
	go func() {
		_, err := service.Admin.CreatePromo(ctx, admin.SavePromoParams{
			WorkspaceID: workspaceID,
			Code:        "ADMIN",
			Payload:     json.RawMessage(`{}`),
			IsActive:    true,
		})
		adminResult <- err
	}()
	waitForPromoWorkspaceLock(t, db, 2)

	if err := transaction.Commit(); err != nil {
		t.Fatalf("release promo workspace lock: %v", err)
	}
	if err := <-importResult; err != nil {
		t.Fatalf("concurrent promo import: %v", err)
	}
	if err := <-adminResult; err != nil {
		t.Fatalf("concurrent promo admin write: %v", err)
	}

	values, err := service.Admin.ListPromos(ctx, workspaceID, admin.Page{Limit: 10})
	if err != nil || len(values) != 2 {
		t.Fatalf("concurrent promo result: promos=%+v err=%v", values, err)
	}
}

func waitForPromoWorkspaceLock(t *testing.T, db interface {
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
			t.Fatalf("inspect promo lock waiters: %v", err)
		}
		if waiting >= minimum {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("promo lock waiters = %d, want at least %d", waiting, minimum)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertPromoCacheRead(t *testing.T, service *Promo, promoID uint64, title string, quantity int64) {
	t.Helper()
	value, err := service.Admin.GetPromo(context.Background(), "cache-workspace", promoID)
	if err != nil || len(value.Localizations) != 1 || value.Localizations[0].Title != title ||
		len(value.Rewards) != 1 || value.Rewards[0].Quantity != quantity {
		t.Fatalf("promo node returned stale catalog: promo=%+v err=%v", value, err)
	}
}

const (
	promoTestPGHost     = "localhost"
	promoTestPGPort     = 5432
	promoTestPGUser     = "postgres"
	promoTestPGPassword = "RBTX0DXKbagvCy2XCAi4qHt0cjeSD6bU"
	promoTestDB         = "promo_test"
)

func TestPromoApplyLifecycleAndCallback(t *testing.T) {
	service := newPromoTestService(t)
	ctx := context.Background()
	promoID, err := service.Admin.CreatePromo(ctx, admin.SavePromoParams{
		WorkspaceID: "workspace-a", Code: "SUMMER2026",
		Payload: json.RawMessage(`{"image":"summer.png"}`), MaxActivations: 10, IsActive: true,
	})
	if err != nil {
		t.Fatalf("create promo: %v", err)
	}
	if err := service.Admin.UpsertLocalization(ctx, admin.SaveLocalizationParams{
		WorkspaceID: "workspace-a", PromoID: promoID, Locale: "ru",
		Title: "Летний промо", Description: "Описание",
	}); err != nil {
		t.Fatalf("upsert localization: %v", err)
	}
	if err := service.Admin.UpsertReward(ctx, admin.SaveRewardParams{
		WorkspaceID: "workspace-a", PromoID: promoID, Key: "coin", Quantity: 100,
	}); err != nil {
		t.Fatalf("upsert reward: %v", err)
	}
	reward, err := service.Admin.GetReward(ctx, "workspace-a", promoID, "coin")
	if err != nil || reward.Key != "coin" || reward.Quantity != 100 {
		t.Fatalf("get reward: %+v, err=%v", reward, err)
	}
	identity := user.Identity{
		WorkspaceID: "workspace-a", AppID: 1, PlatformID: 2, PlatformUserID: "player",
	}
	first, err := service.User.Apply(ctx, user.ApplyParams{Identity: identity, Code: " summer2026 ", Locale: "ru"})
	if err != nil {
		t.Fatalf("apply promo: %v", err)
	}
	if first.Status != repository.StatusSuccess || first.Promo.ID != promoID ||
		first.Promo.Title != "Летний промо" || len(first.Promo.Rewards) != 1 {
		t.Fatalf("unexpected successful result: %+v", first)
	}
	second, err := service.User.Apply(ctx, user.ApplyParams{Identity: identity, Code: "SUMMER2026", Locale: "ru"})
	if err != nil {
		t.Fatalf("apply promo again: %v", err)
	}
	if second.Status != repository.StatusAlreadyApplied ||
		second.Redemption == nil || first.Redemption.ID != second.Redemption.ID {
		t.Fatalf("unexpected repeated result: %+v", second)
	}
	if err := service.Admin.UpsertReward(ctx, admin.SaveRewardParams{
		WorkspaceID: "workspace-a", PromoID: promoID, Key: "coin", Quantity: 999,
	}); err != nil {
		t.Fatalf("update reward after redemption: %v", err)
	}
	redemption, err := service.Admin.GetUserRedemption(ctx, identity, promoID)
	if err != nil || redemption == nil || redemption.ID != first.Redemption.ID {
		t.Fatalf("get user redemption: %+v, err=%v", redemption, err)
	}
	if err := service.Admin.RefreshDailyStats(ctx, "workspace-a", time.Now().Add(-time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("refresh daily stats: %v", err)
	}
	daily, err := service.Admin.ListDailyStats(
		ctx, "workspace-a", promoID, time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour),
	)
	if err != nil || len(daily) != 1 || daily[0].RedemptionCount != 1 || daily[0].UniqueUsers != 1 {
		t.Fatalf("daily stats: %+v, err=%v", daily, err)
	}
	events, err := service.Admin.ListCallbackEvents(ctx, admin.CallbackEventListParams{
		WorkspaceID: "workspace-a",
		Page:        admin.Page{Limit: 10},
	})
	if err != nil {
		t.Fatalf("list callback events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("callback count = %d, want 1", len(events))
	}

	workerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err = service.OnCallback(workerCtx, func(callbackCtx Context) error {
		if callbackCtx.Applied == nil || callbackCtx.Applied.PromoID != promoID ||
			len(callbackCtx.Applied.Rewards) != 1 ||
			callbackCtx.Applied.Rewards[0].Key != "coin" ||
			callbackCtx.Applied.Rewards[0].Quantity != 100 {
			return errors.New("callback payload is incomplete")
		}
		if err := callbackCtx.Successful(); err != nil {
			return err
		}
		cancel()
		return nil
	}, WithCallbackIdleDelay(10*time.Millisecond))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("OnCallback error = %v", err)
	}
}

func TestPromoImportExportCycle(t *testing.T) {
	service := newPromoTestService(t)
	ctx := context.Background()
	promoID, err := service.Admin.CreatePromo(ctx, admin.SavePromoParams{
		WorkspaceID: "workspace-export", Code: "EXPORT2026",
		Payload: json.RawMessage(`{"source":"export"}`), MaxActivations: 5, IsActive: true,
	})
	if err != nil {
		t.Fatalf("create promo: %v", err)
	}
	if err := service.Admin.UpsertLocalization(ctx, admin.SaveLocalizationParams{
		WorkspaceID: "workspace-export", PromoID: promoID, Locale: "ru",
		Title: "Промо", Description: "Описание",
	}); err != nil {
		t.Fatalf("upsert localization: %v", err)
	}
	if err := service.Admin.UpsertReward(ctx, admin.SaveRewardParams{
		WorkspaceID: "workspace-export", PromoID: promoID, Key: "stars", Quantity: 25, Scale: 2,
	}); err != nil {
		t.Fatalf("upsert reward: %v", err)
	}
	pkg, err := service.Admin.Export(ctx, "workspace-export", admin.ExportRequest{})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if _, err := service.Admin.Import(ctx, "workspace-import", admin.ImportRequest{
		Package: pkg, ConflictStrategy: repository.ImportConflictUpdate,
	}); err != nil {
		t.Fatalf("import: %v", err)
	}
	imported, err := service.Admin.Export(ctx, "workspace-import", admin.ExportRequest{})
	if err != nil {
		t.Fatalf("export imported: %v", err)
	}
	if len(imported.Promos) != 1 || len(imported.Promos[0].Localization) != 1 ||
		len(imported.Promos[0].Rewards) != 1 || imported.Promos[0].Rewards[0].Scale != 2 {
		t.Fatalf("unexpected imported package: %+v", imported)
	}
}

func TestPromoStatusesAndAdminCRUD(t *testing.T) {
	service := newPromoTestService(t)
	ctx := context.Background()
	identity := user.Identity{WorkspaceID: "workspace-s", AppID: 1, PlatformID: 1, PlatformUserID: "user"}

	assertStatus := func(code string, expected string) {
		t.Helper()
		result, err := service.User.Apply(ctx, user.ApplyParams{Identity: identity, Code: code, Locale: "ru"})
		if err != nil {
			t.Fatalf("apply %s: %v", code, err)
		}
		if result.Status != expected {
			t.Fatalf("status for %s = %s, want %s", code, result.Status, expected)
		}
	}
	assertStatus("missing", repository.StatusNotFound)

	now := time.Now()
	cases := []struct {
		code     string
		active   bool
		start    *time.Time
		end      *time.Time
		expected string
	}{
		{"inactive", false, nil, nil, repository.StatusInactive},
		{"future", true, timePtr(now.Add(time.Hour)), nil, repository.StatusNotStarted},
		{"expired", true, nil, timePtr(now.Add(-time.Hour)), repository.StatusExpired},
	}
	for _, item := range cases {
		_, err := service.Admin.CreatePromo(ctx, admin.SavePromoParams{
			WorkspaceID: identity.WorkspaceID, Code: item.code, Payload: json.RawMessage(`{}`),
			IsActive: item.active, StartAt: item.start, EndAt: item.end,
		})
		if err != nil {
			t.Fatalf("create %s: %v", item.code, err)
		}
		assertStatus(item.code, item.expected)
	}

	promoID, err := service.Admin.CreatePromo(ctx, admin.SavePromoParams{
		WorkspaceID: identity.WorkspaceID, Code: "crud", Payload: json.RawMessage(`{"v":1}`),
		MaxActivations: 1, IsActive: true,
	})
	if err != nil {
		t.Fatalf("create CRUD promo: %v", err)
	}
	if _, err := service.Admin.UpdatePromo(ctx, admin.SavePromoParams{
		ID: promoID, WorkspaceID: identity.WorkspaceID, Code: "CRUD",
		Payload: json.RawMessage(`{"v":2}`), MaxActivations: 1, IsActive: true,
	}); err != nil {
		t.Fatalf("update promo: %v", err)
	}
	if err := service.Admin.UpsertLocalization(ctx, admin.SaveLocalizationParams{
		WorkspaceID: identity.WorkspaceID, PromoID: promoID, Locale: "ru", Title: "Title",
	}); err != nil {
		t.Fatalf("upsert localization: %v", err)
	}
	if _, err := service.Admin.GetLocalization(ctx, identity.WorkspaceID, promoID, "ru"); err != nil {
		t.Fatalf("get localization: %v", err)
	}
	if err := service.Admin.UpsertReward(ctx, admin.SaveRewardParams{
		WorkspaceID: identity.WorkspaceID, PromoID: promoID, Key: "gem", Quantity: 5,
	}); err != nil {
		t.Fatalf("upsert reward: %v", err)
	}
	if _, err := service.Admin.GetPromo(ctx, identity.WorkspaceID, promoID); err != nil {
		t.Fatalf("get promo: %v", err)
	}
	stats, err := service.Admin.GetStats(ctx, identity.WorkspaceID, promoID)
	if err != nil || stats.RemainingActivations == nil || *stats.RemainingActivations != 1 {
		t.Fatalf("get stats before activation: %+v, err=%v", stats, err)
	}
	if _, err := service.Admin.ListPromos(ctx, identity.WorkspaceID, admin.Page{Limit: 10}); err != nil {
		t.Fatalf("list promos: %v", err)
	}
	if _, err := service.Admin.DeleteReward(ctx, identity.WorkspaceID, promoID, "gem"); err != nil {
		t.Fatalf("delete reward: %v", err)
	}
	if _, err := service.Admin.DeleteLocalization(ctx, identity.WorkspaceID, promoID, "ru"); err != nil {
		t.Fatalf("delete localization: %v", err)
	}
	if _, err := service.Admin.DeletePromo(ctx, identity.WorkspaceID, promoID); err != nil {
		t.Fatalf("soft delete promo: %v", err)
	}
	assertStatus("crud", repository.StatusNotFound)
}

func TestPromoConcurrentLifetimeLimit(t *testing.T) {
	service := newPromoTestService(t)
	ctx := context.Background()
	promoID, err := service.Admin.CreatePromo(ctx, admin.SavePromoParams{
		WorkspaceID: "workspace-limit", Code: "LIMITED", Payload: json.RawMessage(`{}`),
		MaxActivations: 3, IsActive: true,
	})
	if err != nil {
		t.Fatalf("create limited promo: %v", err)
	}

	const workers = 12
	statuses := make(chan string, workers)
	errs := make(chan error, workers)
	var wait sync.WaitGroup
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			result, err := service.User.Apply(ctx, user.ApplyParams{
				Identity: user.Identity{
					WorkspaceID: "workspace-limit", AppID: 1, PlatformID: 1,
					PlatformUserID: fmt.Sprintf("user-%d", index),
				},
				Code: "limited",
			})
			statuses <- result.Status
			errs <- err
		}(i)
	}
	wait.Wait()
	close(statuses)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent apply: %v", err)
		}
	}
	success := 0
	for status := range statuses {
		if status == repository.StatusSuccess {
			success++
		} else if status != repository.StatusLimitReached {
			t.Fatalf("unexpected status: %s", status)
		}
	}
	if success != 3 {
		t.Fatalf("successful activations = %d, want 3", success)
	}
	stats, err := service.Admin.GetStats(ctx, "workspace-limit", promoID)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats.ActivationCount != 3 || stats.RemainingActivations == nil || *stats.RemainingActivations != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	redemptions, err := service.Admin.ListRedemptions(ctx, "workspace-limit", promoID, admin.Page{Limit: 20})
	if err != nil || len(redemptions) != 3 {
		t.Fatalf("redemptions = %d, err=%v", len(redemptions), err)
	}
}

func newPromoTestService(t testing.TB) *Promo {
	return newPromoTestServiceWithOptions(t, promoTestOptions())
}

func newPromoTestServiceWithOptions(t testing.TB, options Options) *Promo {
	t.Helper()
	ctx := context.Background()
	adminDB, err := openPromoPostgres("postgres")
	if err != nil {
		t.Fatalf("open postgres admin: %v", err)
	}
	_, _ = adminDB.ExecContext(ctx, "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()", promoTestDB)
	if _, err := adminDB.ExecContext(ctx, "DROP DATABASE IF EXISTS "+promoTestDB); err != nil {
		t.Fatalf("drop database: %v", err)
	}
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+promoTestDB); err != nil {
		t.Fatalf("create database: %v", err)
	}
	_ = adminDB.Close()
	db, err := openPromoPostgres(promoTestDB)
	if err != nil {
		t.Fatalf("open app postgres: %v", err)
	}
	client, err := sqlwrap.New(db, sqlwrap.Options{
		CacheEnabled:  true,
		CacheSize:     10000,
		CacheTTLCheck: time.Minute,
	})
	if err != nil {
		t.Fatalf("create sql client: %v", err)
	}
	repo := repository.New(client)
	if err := repo.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrap promo: %v", err)
	}
	service, err := NewWithDatabase(ctx, db, options)
	if err != nil {
		t.Fatalf("create promo service: %v", err)
	}
	t.Cleanup(func() {
		_ = service.Close()
		_ = repo.Close()
		_ = client.Close()
	})
	return service
}

func promoTestOptions() Options {
	return Options{
		CacheEnabled:  true,
		CacheSize:     10000,
		CacheTTLCheck: time.Minute,
		CacheL1Delay:  time.Minute,
	}
}

func openPromoPostgres(database string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", promoTestPGHost, promoTestPGPort, promoTestPGUser, promoTestPGPassword, database)
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

func timePtr(value time.Time) *time.Time { return &value }

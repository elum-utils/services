package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	json "github.com/goccy/go-json"

	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/go-sql-driver/mysql"
)

const (
	exportImportMySQLHost     = "localhost"
	exportImportMySQLUser     = "root"
	exportImportMySQLPassword = "RBTX0DXKbagvCy2XCAi4qHt0cjeSD6bU"
	exportImportDB            = "tasks_export_import_test"
)

func TestExportSectionsDefaultsToFullCatalog(t *testing.T) {
	sections := exportSections(nil)
	for _, key := range []string{
		ExportSectionGroups,
		ExportSectionTasks,
		ExportSectionSequences,
		ExportSectionLocalization,
		ExportSectionRewards,
		ExportSectionTarget,
		ExportSectionIntegration,
		ExportSectionPartnerConfigs,
		ExportSectionPartnerRewards,
	} {
		if !sections[key] {
			t.Fatalf("default export section %q disabled", key)
		}
	}
}

func TestValidateExportPackageRequiresSequencePair(t *testing.T) {
	sequenceKey := "chain"
	if err := validateExportPackage(ExportPackage{
		Format:  ExportFormat,
		Service: "tasks",
		Groups: []ExportGroup{{
			Key: "daily",
			Tasks: []ExportTask{{
				Key:         "task",
				SequenceKey: &sequenceKey,
			}},
		}},
	}); err == nil {
		t.Fatal("sequence_key without sequence_position must fail")
	}
}

func TestRequireImportSecrets(t *testing.T) {
	err := requireImportSecrets(ExportPackage{
		Format:  ExportFormat,
		Service: "tasks",
		Groups: []ExportGroup{{
			Key: "daily",
			PartnerConfigs: []ExportPartnerConfig{{
				Provider: "tgrass",
				Platform: "telegram",
				Secret:   &ExportSecret{Mode: "required", Key: "tasks.partner.tgrass.daily.telegram.secret"},
			}},
		}},
	}, map[string]string{"tasks.partner.tgrass.daily.telegram.secret": "token"})
	if err != nil {
		t.Fatalf("secret should satisfy import requirement: %v", err)
	}
}

func TestExportImportFullCycle(t *testing.T) {
	repo := newExportImportRepository(t)
	ctx := context.Background()
	sourceWorkspace := "source"
	targetWorkspace := "target"
	seedExportSource(t, repo, sourceWorkspace)

	pkg, err := repo.Export(ctx, sourceWorkspace, ExportRequest{Now: time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if pkg.Format != ExportFormat || pkg.Service != "tasks" || len(pkg.Groups) != 1 || len(pkg.Sequences) != 1 {
		t.Fatalf("unexpected export package: %+v", pkg)
	}
	raw, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("marshal package: %v", err)
	}
	if strings.Contains(string(raw), "source-token") {
		t.Fatalf("export must not contain secret value: %s", raw)
	}

	preview, err := repo.PreviewImport(ctx, targetWorkspace, pkg)
	if err != nil {
		t.Fatalf("preview import: %v", err)
	}
	if preview.Counts.Groups != 1 ||
		preview.Counts.Sequences != 1 ||
		preview.Counts.Tasks != 1 ||
		preview.Counts.GroupLocalizations != 2 ||
		preview.Counts.TaskLocalizations != 2 ||
		preview.Counts.Rewards != 1 ||
		preview.Counts.PartnerConfigs != 1 ||
		preview.Counts.PartnerRewards != 1 {
		t.Fatalf("bad preview counts: %+v", preview.Counts)
	}
	if len(preview.RequiredSecrets) != 1 {
		t.Fatalf("required secrets = %+v, want 1", preview.RequiredSecrets)
	}
	if _, err := repo.Import(ctx, targetWorkspace, ImportRequest{
		Package:          pkg,
		ConflictStrategy: ImportConflictFail,
	}); err == nil {
		t.Fatal("import without required secret must fail")
	}

	result, err := repo.Import(ctx, targetWorkspace, ImportRequest{
		Package:          pkg,
		ConflictStrategy: ImportConflictFail,
		Secrets:          map[string]string{preview.RequiredSecrets[0].Key: "target-token"},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if result.Imported.Tasks != 1 || result.Imported.Rewards != 1 || result.Imported.PartnerConfigs != 1 {
		t.Fatalf("bad import result: %+v", result)
	}

	imported, err := repo.Export(ctx, targetWorkspace, ExportRequest{})
	if err != nil {
		t.Fatalf("export imported workspace: %v", err)
	}
	assertImportedCatalog(t, imported)
	config, found, err := repo.GetPartnerConfig(ctx, targetWorkspace, "tgrass", "daily", "telegram")
	if err != nil || !found || config.Secret == nil || *config.Secret != "target-token" {
		t.Fatalf("bad imported partner config: found=%t config=%+v err=%v", found, config, err)
	}

	conflictPreview, err := repo.PreviewImport(ctx, targetWorkspace, pkg)
	if err != nil {
		t.Fatalf("conflict preview: %v", err)
	}
	if len(conflictPreview.Conflicts) == 0 {
		t.Fatal("preview after import must report conflicts")
	}
	if _, err := repo.Import(ctx, targetWorkspace, ImportRequest{
		Package:          pkg,
		ConflictStrategy: ImportConflictFail,
		Secrets:          map[string]string{preview.RequiredSecrets[0].Key: "target-token"},
	}); err == nil {
		t.Fatal("fail_on_conflict must reject existing catalog")
	}
	skipped, err := repo.Import(ctx, targetWorkspace, ImportRequest{
		Package:          pkg,
		ConflictStrategy: ImportConflictSkip,
		Secrets:          map[string]string{preview.RequiredSecrets[0].Key: "target-token"},
	})
	if err != nil {
		t.Fatalf("skip existing import: %v", err)
	}
	if skipped.Skipped.Tasks != 1 || skipped.Skipped.Groups != 1 || skipped.Skipped.PartnerConfigs != 1 {
		t.Fatalf("bad skipped result: %+v", skipped)
	}

	pkg.Groups[0].Localization["ru"] = ExportText{Title: "Обновленные", Description: "Обновленное описание"}
	pkg.Groups[0].Tasks[0].Rewards[0].Quantity = 777
	updated, err := repo.Import(ctx, targetWorkspace, ImportRequest{
		Package:          pkg,
		ConflictStrategy: ImportConflictUpdate,
		Secrets:          map[string]string{preview.RequiredSecrets[0].Key: "updated-token"},
	})
	if err != nil {
		t.Fatalf("update existing import: %v", err)
	}
	if updated.Imported.Tasks != 1 || updated.Imported.Rewards != 1 {
		t.Fatalf("bad update result: %+v", updated)
	}
	afterUpdate, err := repo.Export(ctx, targetWorkspace, ExportRequest{})
	if err != nil {
		t.Fatalf("export after update: %v", err)
	}
	if afterUpdate.Groups[0].Localization["ru"].Title != "Обновленные" {
		t.Fatalf("group localization was not updated: %+v", afterUpdate.Groups[0].Localization)
	}
	if afterUpdate.Groups[0].Tasks[0].Rewards[0].Quantity != 777 {
		t.Fatalf("reward was not updated: %+v", afterUpdate.Groups[0].Tasks[0].Rewards[0])
	}
}

func TestExportSectionsAndInvalidImportFormats(t *testing.T) {
	repo := newExportImportRepository(t)
	ctx := context.Background()
	workspaceID := "sections"
	seedExportSource(t, repo, workspaceID)

	pkg, err := repo.Export(ctx, workspaceID, ExportRequest{
		Sections: []string{ExportSectionGroups, ExportSectionTasks},
	})
	if err != nil {
		t.Fatalf("section export: %v", err)
	}
	if len(pkg.Sequences) != 0 {
		t.Fatalf("sequences must be omitted: %+v", pkg.Sequences)
	}
	if len(pkg.Groups) != 1 || len(pkg.Groups[0].Tasks) != 1 {
		t.Fatalf("tasks must be exported inside groups: %+v", pkg)
	}
	task := pkg.Groups[0].Tasks[0]
	if task.SequenceKey != nil || task.SequencePosition != nil {
		t.Fatalf("sequence fields must be omitted: %+v", task)
	}
	if len(task.Rewards) != 0 || len(task.Localization) != 0 || len(pkg.Groups[0].PartnerConfigs) != 0 {
		t.Fatalf("disabled sections leaked into export: %+v", pkg.Groups[0])
	}
	if len(task.Target) != 0 || len(task.Integration.Payload) != 0 || task.Integration.Provider != nil {
		t.Fatalf("target/integration must be omitted: %+v", task)
	}

	if _, err := repo.PreviewImport(ctx, workspaceID, ExportPackage{Format: "tasks.export.v2", Service: "tasks"}); err == nil {
		t.Fatal("unsupported format must fail")
	}
	if _, err := repo.PreviewImport(ctx, workspaceID, ExportPackage{Format: ExportFormat, Service: "cpa"}); err == nil {
		t.Fatal("wrong service must fail")
	}
	position := uint32(1)
	if _, err := repo.PreviewImport(ctx, workspaceID, ExportPackage{
		Format:  ExportFormat,
		Service: "tasks",
		Groups: []ExportGroup{{
			Key: "broken",
			Tasks: []ExportTask{{
				Key:              "broken_task",
				SequencePosition: &position,
			}},
		}},
	}); err == nil {
		t.Fatal("sequence_position without sequence_key must fail")
	}
}

func TestImportDailyTasksExampleAndExport(t *testing.T) {
	repo := newExportImportRepository(t)
	ctx := context.Background()
	workspaceID := "daily-example"

	raw, err := os.ReadFile(filepath.Join("..", "examples", "daily_tasks_import.json"))
	if err != nil {
		t.Fatalf("read daily example: %v", err)
	}
	var pkg ExportPackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("unmarshal daily example: %v", err)
	}

	preview, err := repo.PreviewImport(ctx, workspaceID, pkg)
	if err != nil {
		t.Fatalf("preview daily example: %v", err)
	}
	if preview.Counts.Groups != 1 ||
		preview.Counts.Sequences != 0 ||
		preview.Counts.Tasks != 19 ||
		preview.Counts.GroupLocalizations != 2 ||
		preview.Counts.TaskLocalizations != 38 ||
		preview.Counts.Rewards != 19 ||
		preview.Counts.PartnerConfigs != 0 ||
		preview.Counts.PartnerRewards != 0 {
		t.Fatalf("bad daily preview counts: %+v", preview.Counts)
	}
	if len(preview.Conflicts) != 0 || len(preview.Warnings) != 0 || len(preview.RequiredSecrets) != 0 {
		t.Fatalf("daily preview should be clean: %+v", preview)
	}

	result, err := repo.Import(ctx, workspaceID, ImportRequest{
		Package:          pkg,
		ConflictStrategy: ImportConflictFail,
	})
	if err != nil {
		t.Fatalf("import daily example: %v", err)
	}
	if result.Imported.Groups != 1 ||
		result.Imported.Tasks != 19 ||
		result.Imported.GroupLocalizations != 2 ||
		result.Imported.TaskLocalizations != 38 ||
		result.Imported.Rewards != 19 {
		t.Fatalf("bad daily import result: %+v", result)
	}

	exported, err := repo.Export(ctx, workspaceID, ExportRequest{})
	if err != nil {
		t.Fatalf("export daily example: %v", err)
	}
	assertDailyExampleExport(t, pkg, exported)
}

func seedExportSource(t *testing.T, repo *Repository, workspaceID string) {
	t.Helper()
	ctx := context.Background()
	if err := repo.UpsertGroup(ctx, workspaceID, "daily", 10, true); err != nil {
		t.Fatalf("upsert group: %v", err)
	}
	if err := repo.UpsertGroupLocalization(ctx, workspaceID, "daily", "ru", "Ежедневные", "Ежедневные задания"); err != nil {
		t.Fatalf("upsert group ru localization: %v", err)
	}
	if err := repo.UpsertGroupLocalization(ctx, workspaceID, "daily", "en", "Daily", "Daily tasks"); err != nil {
		t.Fatalf("upsert group en localization: %v", err)
	}
	if err := repo.UpsertSequence(ctx, workspaceID, "daily_chain", 10, true); err != nil {
		t.Fatalf("upsert sequence: %v", err)
	}
	position := uint32(1)
	provider := "http"
	taskID, err := repo.SaveTask(ctx, SaveTaskParams{
		WorkspaceID: workspaceID,
		Key:         "subscribe_tg",
		GroupKey:    "daily",
		SequenceKey: strPtr("daily_chain"), SequencePosition: &position,
		TaskKind:            TaskKindChannelSubscribe,
		ActionKey:           "telegram.subscribe",
		ActionKind:          ActionKindChannelSubscribe,
		ClaimMode:           ClaimModeManual,
		TargetCount:         1,
		ResetUnit:           ResetNever,
		ResetEvery:          1,
		Position:            10,
		Payload:             json.RawMessage(`{"channel_url":"https://t.me/example"}`),
		Target:              json.RawMessage(`{"platform":["tma",12],"loc":["ru"]}`),
		IntegrationKind:     strPtr("channel"),
		IntegrationProvider: &provider,
		IntegrationPayload:  json.RawMessage(`{"url":"https://partner.example/check","secret":"private"}`),
		ImageURL:            strPtr("https://example.com/image.png"),
		IsVisible:           true,
		IsActive:            true,
	})
	if err != nil {
		t.Fatalf("save task: %v", err)
	}
	if err := repo.UpsertTaskLocalization(ctx, workspaceID, taskID, "ru", "Подпишись", "Подпишись на канал"); err != nil {
		t.Fatalf("upsert task ru localization: %v", err)
	}
	if err := repo.UpsertTaskLocalization(ctx, workspaceID, taskID, "en", "Subscribe", "Subscribe to channel"); err != nil {
		t.Fatalf("upsert task en localization: %v", err)
	}
	if err := repo.UpsertReward(ctx, workspaceID, taskID, Reward{Key: "coin", Type: "quantity", Quantity: 100, Scale: 2}, 10); err != nil {
		t.Fatalf("upsert reward: %v", err)
	}
	secret := "source-token"
	if err := repo.SavePartnerConfig(ctx, SavePartnerConfigParams{
		WorkspaceID: workspaceID,
		Provider:    "tgrass",
		GroupKey:    "daily",
		Platform:    "telegram",
		IsEnabled:   true,
		Secret:      &secret,
		Target:      json.RawMessage(`{"platform":"tma"}`),
		Settings:    json.RawMessage(`{"limit":5}`),
	}); err != nil {
		t.Fatalf("save partner config: %v", err)
	}
	if err := repo.SavePartnerRewardRule(ctx, SavePartnerRewardRuleParams{
		WorkspaceID:  workspaceID,
		Provider:     "tgrass",
		GroupKey:     "daily",
		ExternalType: "*",
		Reward:       Reward{Key: "coin", Type: "quantity", Quantity: 50, Scale: 2},
		Position:     10,
		IsEnabled:    true,
	}); err != nil {
		t.Fatalf("save partner reward rule: %v", err)
	}
}

func assertDailyExampleExport(t *testing.T, imported, exported ExportPackage) {
	t.Helper()
	if exported.Format != ExportFormat || exported.Service != "tasks" {
		t.Fatalf("bad exported header: %+v", exported)
	}
	if len(exported.Sequences) != 0 {
		t.Fatalf("daily tasks must be standalone, got sequences: %+v", exported.Sequences)
	}
	if len(imported.Groups) != 1 || len(exported.Groups) != 1 {
		t.Fatalf("bad group counts: imported=%d exported=%d", len(imported.Groups), len(exported.Groups))
	}
	expectedGroup := imported.Groups[0]
	actualGroup := exported.Groups[0]
	if actualGroup.Key != expectedGroup.Key || actualGroup.Localization["ru"].Title != "Ежедневные задания" ||
		actualGroup.Localization["en"].Title != "Daily tasks" {
		t.Fatalf("bad exported daily group: %+v", actualGroup)
	}
	if len(actualGroup.Tasks) != 19 {
		t.Fatalf("exported tasks = %d, want 19", len(actualGroup.Tasks))
	}
	expectedByKey := make(map[string]ExportTask, len(expectedGroup.Tasks))
	for _, task := range expectedGroup.Tasks {
		expectedByKey[task.Key] = task
	}
	for _, task := range actualGroup.Tasks {
		expected, ok := expectedByKey[task.Key]
		if !ok {
			t.Fatalf("unexpected exported task: %+v", task)
		}
		if task.SequenceKey != nil || task.SequencePosition != nil {
			t.Fatalf("daily task must not be sequential: %+v", task)
		}
		if len(task.Target) != 0 {
			t.Fatalf("daily task must not have target: key=%s target=%s", task.Key, task.Target)
		}
		if task.Localization["ru"].Title != expected.Localization["ru"].Title ||
			task.Localization["en"].Title != expected.Localization["en"].Title {
			t.Fatalf("bad localization for %s: %+v", task.Key, task.Localization)
		}
		if len(task.Rewards) != 1 || len(expected.Rewards) != 1 {
			t.Fatalf("bad rewards for %s: actual=%+v expected=%+v", task.Key, task.Rewards, expected.Rewards)
		}
		if task.Rewards[0].Key != "stars" ||
			task.Rewards[0].Type != "quantity" ||
			task.Rewards[0].Quantity != expected.Rewards[0].Quantity ||
			task.Rewards[0].Scale != 2 {
			t.Fatalf("bad reward for %s: actual=%+v expected=%+v", task.Key, task.Rewards[0], expected.Rewards[0])
		}
		if task.Reset.Unit != ResetDay || task.Reset.Every != 1 {
			t.Fatalf("daily task must reset daily: key=%s reset=%+v", task.Key, task.Reset)
		}
	}
}

func assertImportedCatalog(t *testing.T, pkg ExportPackage) {
	t.Helper()
	if len(pkg.Groups) != 1 || pkg.Groups[0].Key != "daily" {
		t.Fatalf("bad imported groups: %+v", pkg.Groups)
	}
	group := pkg.Groups[0]
	if group.Localization["ru"].Title != "Ежедневные" || group.Localization["en"].Title != "Daily" {
		t.Fatalf("bad group localization: %+v", group.Localization)
	}
	if len(group.Tasks) != 1 || group.Tasks[0].Key != "subscribe_tg" {
		t.Fatalf("bad imported tasks: %+v", group.Tasks)
	}
	task := group.Tasks[0]
	if task.SequenceKey == nil || *task.SequenceKey != "daily_chain" || task.SequencePosition == nil || *task.SequencePosition != 1 {
		t.Fatalf("bad task sequence fields: %+v", task)
	}
	if task.Localization["ru"].Title != "Подпишись" || len(task.Rewards) != 1 ||
		task.Rewards[0].Quantity != 100 || task.Rewards[0].Scale != 2 {
		t.Fatalf("bad task localized rewards: %+v", task)
	}
	if len(group.PartnerConfigs) != 1 || group.PartnerConfigs[0].Secret == nil {
		t.Fatalf("bad partner configs: %+v", group.PartnerConfigs)
	}
	if len(group.PartnerRewardRules) != 1 || group.PartnerRewardRules[0].Reward.Quantity != 50 ||
		group.PartnerRewardRules[0].Reward.Scale != 2 {
		t.Fatalf("bad partner rewards: %+v", group.PartnerRewardRules)
	}
}

func newExportImportRepository(t *testing.T) *Repository {
	t.Helper()
	ctx := context.Background()
	adminDB, err := openExportImportMySQL("")
	if err != nil {
		t.Fatalf("open admin mysql: %v", err)
	}
	if _, err := adminDB.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+exportImportDB+"`"); err != nil {
		t.Fatalf("drop database: %v", err)
	}
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE `"+exportImportDB+"`"); err != nil {
		t.Fatalf("create database: %v", err)
	}
	_ = adminDB.Close()
	db, err := openExportImportMySQL(exportImportDB)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	client, err := sqlwrap.New(db, sqlwrap.Options{CacheEnabled: true, CacheSize: 1000, CacheTTLCheck: time.Minute})
	if err != nil {
		t.Fatalf("sqlwrap: %v", err)
	}
	repo := New(client)
	if err := repo.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
		_ = client.Close()
	})
	return repo
}

func openExportImportMySQL(database string) (*sql.DB, error) {
	addr := exportImportMySQLHost
	if !strings.Contains(addr, ":") {
		addr += ":3306"
	}
	cfg := mysql.Config{
		User: exportImportMySQLUser, Passwd: exportImportMySQLPassword, Net: "tcp", Addr: addr, DBName: database,
		AllowNativePasswords: true, InterpolateParams: true, ParseTime: true,
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func strPtr(value string) *string { return &value }

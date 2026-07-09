package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (r *PaymentRepository) PreviewImport(ctx context.Context, workspaceID string, pkg ExportPackage) (ImportPreview, error) {
	workspaceID, err := requireWorkspaceID(workspaceID)
	if err != nil {
		return ImportPreview{}, err
	}
	pkg = normalizeExportPackage(pkg)
	if err := validateExportPackage(pkg); err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{Format: pkg.Format, Service: pkg.Service, Counts: countPackage(pkg)}
	existing, err := r.importExistingKeys(ctx, workspaceID)
	if err != nil {
		return ImportPreview{}, err
	}
	for _, group := range pkg.Groups {
		if existing.groups[group.Code] {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "group", Key: group.Code})
		}
		for _, product := range group.Products {
			if existing.products[product.ID] {
				preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "product", Key: product.ID})
			}
		}
	}
	for _, product := range pkg.Products {
		if existing.products[product.ID] {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "product", Key: product.ID})
		}
	}
	for _, item := range pkg.Items {
		if existing.items[item.ID] {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "item", Key: item.ID})
		}
	}
	if existing.tonWallet && len(pkg.TONWallets) > 0 {
		preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "ton_wallet", Key: "default"})
	}
	return preview, nil
}

func (r *PaymentRepository) Import(ctx context.Context, workspaceID string, req ImportRequest) (ImportResult, error) {
	workspaceID, err := requireWorkspaceID(workspaceID)
	if err != nil {
		return ImportResult{}, err
	}
	req.Package = normalizeExportPackage(req.Package)
	if err := validateExportPackage(req.Package); err != nil {
		return ImportResult{}, err
	}
	strategy := req.ConflictStrategy
	if strategy == "" {
		strategy = ImportConflictFail
	}
	if strategy != ImportConflictFail && strategy != ImportConflictSkip && strategy != ImportConflictUpdate {
		return ImportResult{}, fmt.Errorf("unsupported import conflict strategy: %s", strategy)
	}
	preview, err := r.PreviewImport(ctx, workspaceID, req.Package)
	if err != nil {
		return ImportResult{}, err
	}
	if strategy == ImportConflictFail && len(preview.Conflicts) > 0 {
		return ImportResult{}, fmt.Errorf("import conflicts found: %d", len(preview.Conflicts))
	}
	result := ImportResult{}
	err = r.WithTx(ctx, func(txRepo *PaymentRepository) error {
		if err := txRepo.importBulk(ctx, workspaceID, req.Package, strategy, preview, &result); err != nil {
			return err
		}
		return txRepo.RebuildWorkspaceProductCache(ctx, workspaceID)
	})
	if err != nil {
		return ImportResult{}, err
	}
	return result, r.invalidateWorkspaceCache(workspaceID)
}

func (r *PaymentRepository) importBulk(ctx context.Context, workspaceID string, pkg ExportPackage, strategy string, preview ImportPreview, result *ImportResult) error {
	if err := r.importGroupsBulk(ctx, workspaceID, pkg.Groups, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importItemsBulk(ctx, workspaceID, pkg.Items, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importLocalizationsBulk(ctx, workspaceID, pkg, strategy, preview, result); err != nil {
		return err
	}
	products := flattenProducts(pkg)
	if err := r.importProductsBulk(ctx, workspaceID, products, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importProductItemsBulk(ctx, workspaceID, products, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importPricesBulk(ctx, workspaceID, products, strategy, preview, result); err != nil {
		return err
	}
	return r.importTONWalletsBulk(ctx, workspaceID, pkg.TONWallets, strategy, preview, result)
}

func (r *PaymentRepository) importGroupsBulk(ctx context.Context, workspaceID string, groups []ExportProductGroup, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(groups))
	for _, group := range groups {
		if previewHasConflict(preview, "group", group.Code) && strategy == ImportConflictSkip {
			result.Skipped.Groups++
			continue
		}
		rows = append(rows, []any{workspaceID, group.Code, nullableString(group.TitleKey), nullableString(group.DescriptionKey), group.Position, group.IsActive})
		result.Imported.Groups++
	}
	return r.execImportBulk(ctx, "payment_product_group",
		[]string{"workspace_id", "code", "title_key", "description_key", "position", "is_active"},
		rows,
		"(workspace_id, code)",
		"title_key = EXCLUDED.title_key, description_key = EXCLUDED.description_key, position = EXCLUDED.position, "+
			"is_active = EXCLUDED.is_active, updated_at = now()",
	)
}

func (r *PaymentRepository) importItemsBulk(ctx context.Context, workspaceID string, items []ExportItem, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		if previewHasConflict(preview, "item", item.ID) && strategy == ImportConflictSkip {
			result.Skipped.Items++
			continue
		}
		rows = append(rows, []any{
			workspaceID, item.ID, nullableString(item.ItemType), item.TitleKey,
			nullableString(item.DescriptionKey), defaultString(item.Rarity, "common"), item.Position,
		})
		result.Imported.Items++
	}
	return r.execImportBulk(ctx, "payment_item",
		[]string{"workspace_id", "id", "item_type", "title_key", "description_key", "rarity", "position"},
		rows,
		"(workspace_id, id)",
		"item_type = EXCLUDED.item_type, title_key = EXCLUDED.title_key, description_key = EXCLUDED.description_key, "+
			"rarity = EXCLUDED.rarity, position = EXCLUDED.position, updated_at = now()",
	)
}

func (r *PaymentRepository) importLocalizationsBulk(ctx context.Context, workspaceID string, pkg ExportPackage, strategy string, preview ImportPreview, result *ImportResult) error {
	rowsByKey := make(map[string][]any)
	addText := func(entityType, entityKey string, titleKey string, descriptionKey *string, localization map[string]ExportText) {
		if previewHasConflict(preview, entityType, entityKey) && strategy == ImportConflictSkip {
			return
		}
		for locale, text := range localization {
			if titleKey != "" {
				rowsByKey[locale+"\x00"+titleKey] = []any{workspaceID, locale, titleKey, text.Title}
			}
			if descriptionKey != nil {
				rowsByKey[locale+"\x00"+*descriptionKey] = []any{workspaceID, locale, *descriptionKey, text.Description}
			}
		}
	}
	for _, group := range pkg.Groups {
		titleKey := ""
		if group.TitleKey != nil {
			titleKey = *group.TitleKey
		}
		addText("group", group.Code, titleKey, group.DescriptionKey, group.Localization)
		for _, product := range group.Products {
			addText("product", product.ID, product.TitleKey, product.DescriptionKey, product.Localization)
		}
	}
	for _, product := range pkg.Products {
		addText("product", product.ID, product.TitleKey, product.DescriptionKey, product.Localization)
	}
	for _, item := range pkg.Items {
		addText("item", item.ID, item.TitleKey, item.DescriptionKey, item.Localization)
	}
	rows := make([][]any, 0, len(rowsByKey))
	for _, row := range rowsByKey {
		rows = append(rows, row)
		result.Imported.Localizations++
	}
	return r.execImportBulk(ctx, "payment_localization",
		[]string{"workspace_id", "locale", "localization_key", "value"},
		rows,
		"(workspace_id, locale, localization_key)",
		"value = EXCLUDED.value, updated_at = now()",
	)
}

func (r *PaymentRepository) importProductsBulk(ctx context.Context, workspaceID string, products []ExportProduct, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(products))
	for _, product := range products {
		if previewHasConflict(preview, "product", product.ID) && strategy == ImportConflictSkip {
			result.Skipped.Products++
			continue
		}
		rows = append(rows, []any{
			workspaceID, product.ID, nullableString(product.GroupCode), product.TitleKey,
			nullableString(product.DescriptionKey), defaultJSON(product.Target, "null"),
			nullableString(product.ImageURL), nullableString(product.LinkURL), nullableString(product.SizeLabel),
			nullableInt64(product.PeriodSeconds), nullableInt64(product.TrialDurationSeconds),
			defaultString(product.QuantityMode, "fixed"), product.Position, product.GlobalLimit,
			defaultString(product.GlobalInterval, "UNLIMITED"), product.GlobalIntervalCount,
			product.UserLimit, defaultString(product.UserInterval, "UNLIMITED"), product.UserIntervalCount,
			defaultTime(product.AvailableFrom, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			defaultTime(product.AvailableUntil, time.Date(2124, 1, 1, 0, 0, 0, 0, time.UTC)),
			product.IsVisible, product.IsClosed,
		})
		result.Imported.Products++
	}
	return r.execImportBulk(ctx, "payment_product",
		[]string{
			"workspace_id", "id", "group_code", "title_key", "description_key", "target", "image_url",
			"link_url", "size_label", "period_seconds", "trial_duration_seconds", "quantity_mode",
			"position", "global_limit", "global_interval", "global_interval_count", "user_limit",
			"user_interval", "user_interval_count", "available_from", "available_until", "is_visible", "is_closed",
		},
		rows,
		"(workspace_id, id)",
		"group_code = EXCLUDED.group_code, title_key = EXCLUDED.title_key, description_key = EXCLUDED.description_key, "+
			"target = EXCLUDED.target, image_url = EXCLUDED.image_url, link_url = EXCLUDED.link_url, size_label = EXCLUDED.size_label, "+
			"period_seconds = EXCLUDED.period_seconds, trial_duration_seconds = EXCLUDED.trial_duration_seconds, "+
			"quantity_mode = EXCLUDED.quantity_mode, position = EXCLUDED.position, global_limit = EXCLUDED.global_limit, "+
			"global_interval = EXCLUDED.global_interval, global_interval_count = EXCLUDED.global_interval_count, "+
			"user_limit = EXCLUDED.user_limit, user_interval = EXCLUDED.user_interval, user_interval_count = EXCLUDED.user_interval_count, "+
			"available_from = EXCLUDED.available_from, available_until = EXCLUDED.available_until, "+
			"is_visible = EXCLUDED.is_visible, is_closed = EXCLUDED.is_closed, updated_at = now()",
	)
}

func (r *PaymentRepository) importProductItemsBulk(ctx context.Context, workspaceID string, products []ExportProduct, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, product := range products {
		if previewHasConflict(preview, "product", product.ID) && strategy == ImportConflictSkip {
			continue
		}
		for _, item := range product.Items {
			rows = append(rows, []any{
				workspaceID, product.ID, item.ItemID, defaultString(item.RewardType, "quantity"),
				item.Quantity, item.Scale, nullableString(item.DurationUnit),
			})
			result.Imported.ProductItems++
		}
	}
	return r.execImportBulk(ctx, "payment_product_item",
		[]string{"workspace_id", "product_id", "item_id", "reward_type", "quantity", "scale", "duration_unit"},
		rows,
		"(workspace_id, product_id, item_id)",
		"reward_type = EXCLUDED.reward_type, quantity = EXCLUDED.quantity, scale = EXCLUDED.scale, "+
			"duration_unit = EXCLUDED.duration_unit, updated_at = now()",
	)
}

func (r *PaymentRepository) importPricesBulk(ctx context.Context, workspaceID string, products []ExportProduct, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, product := range products {
		if previewHasConflict(preview, "product", product.ID) && strategy == ImportConflictSkip {
			continue
		}
		for _, price := range product.Prices {
			rows = append(rows, []any{
				workspaceID, product.ID, price.AssetCode, price.ListAmountMinor, price.DiscountAmountMinor,
				defaultString(price.PricingMode, "fixed"), nullableString(price.ReferenceAssetCode),
				nullableUint64(price.ReferenceListAmountMinor), nullableUint64(price.ReferenceDiscountAmountMinor),
				nullableString(price.Coefficient), price.IsPromotion, price.StartsAt, price.EndsAt,
			})
			result.Imported.Prices++
		}
	}
	return r.execImportBulk(ctx, "payment_price",
		[]string{
			"workspace_id", "product_id", "asset_code", "list_amount_minor", "discount_amount_minor",
			"pricing_mode", "reference_asset_code", "reference_list_amount_minor",
			"reference_discount_amount_minor", "coefficient", "is_promotion", "starts_at", "ends_at",
		},
		rows,
		"(workspace_id, product_id, asset_code, is_promotion, starts_at, ends_at)",
		"list_amount_minor = EXCLUDED.list_amount_minor, discount_amount_minor = EXCLUDED.discount_amount_minor, "+
			"pricing_mode = EXCLUDED.pricing_mode, reference_asset_code = EXCLUDED.reference_asset_code, "+
			"reference_list_amount_minor = EXCLUDED.reference_list_amount_minor, "+
			"reference_discount_amount_minor = EXCLUDED.reference_discount_amount_minor, coefficient = EXCLUDED.coefficient, "+
			"updated_at = now()",
	)
}

func (r *PaymentRepository) importTONWalletsBulk(ctx context.Context, workspaceID string, wallets []ExportTONWallet, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(wallets))
	for _, wallet := range wallets {
		if previewHasConflict(preview, "ton_wallet", "default") && strategy == ImportConflictSkip {
			result.Skipped.TONWallets++
			continue
		}
		rows = append(rows, []any{
			workspaceID,
			defaultString(wallet.Network, "mainnet"),
			wallet.WalletAddress,
			nullableString(wallet.NetworkConfigURL),
			wallet.IsEnabled,
		})
		result.Imported.TONWallets++
	}
	return r.execImportBulk(ctx, "payment_ton_wallet",
		[]string{"workspace_id", "network", "wallet_address", "network_config_url", "is_enabled"},
		rows,
		"(workspace_id)",
		"network = EXCLUDED.network, wallet_address = EXCLUDED.wallet_address, "+
			"network_config_url = EXCLUDED.network_config_url, is_enabled = EXCLUDED.is_enabled, updated_at = now()",
	)
}

func (r *PaymentRepository) execImportBulk(ctx context.Context, table string, columns []string, rows [][]any, conflictTarget string, duplicateUpdate string) error {
	if len(rows) == 0 {
		return nil
	}
	query, args := compileImportBulkUpsert(table, columns, rows, conflictTarget, duplicateUpdate)
	_, err := r.executor.ExecContext(ctx, query, args...)
	return err
}

func compileImportBulkUpsert(table string, columns []string, rows [][]any, conflictTarget string, duplicateUpdate string) (string, []any) {
	var builder strings.Builder
	builder.WriteString("INSERT INTO ")
	builder.WriteString(table)
	builder.WriteString(" (")
	builder.WriteString(strings.Join(columns, ", "))
	builder.WriteString(") VALUES ")
	args := make([]any, 0, len(rows)*len(columns))
	for rowIndex, row := range rows {
		if rowIndex > 0 {
			builder.WriteString(", ")
		}
		builder.WriteByte('(')
		for columnIndex := range columns {
			if columnIndex > 0 {
				builder.WriteString(", ")
			}
			fmt.Fprintf(&builder, "$%d", len(args)+columnIndex+1)
		}
		builder.WriteByte(')')
		args = append(args, row...)
	}
	if duplicateUpdate != "" {
		builder.WriteString(" ON CONFLICT ")
		builder.WriteString(conflictTarget)
		builder.WriteString(" DO UPDATE SET ")
		builder.WriteString(duplicateUpdate)
	}
	return builder.String(), args
}

func validateExportPackage(pkg ExportPackage) error {
	if pkg.Format != ExportFormat {
		return fmt.Errorf("unsupported export format: %s", pkg.Format)
	}
	if pkg.Service != "payment" {
		return fmt.Errorf("unsupported export service: %s", pkg.Service)
	}
	return nil
}

func countPackage(pkg ExportPackage) ImportCounts {
	var counts ImportCounts
	counts.Groups = uint64(len(pkg.Groups))
	counts.Items = uint64(len(pkg.Items))
	counts.TONWallets = uint64(len(pkg.TONWallets))
	for _, group := range pkg.Groups {
		counts.Localizations += uint64(len(group.Localization))
		countProducts(&counts, group.Products)
	}
	countProducts(&counts, pkg.Products)
	for _, item := range pkg.Items {
		counts.Localizations += uint64(len(item.Localization))
	}
	return counts
}

func countProducts(counts *ImportCounts, products []ExportProduct) {
	counts.Products += uint64(len(products))
	for _, product := range products {
		counts.Localizations += uint64(len(product.Localization))
		counts.ProductItems += uint64(len(product.Items))
		counts.Prices += uint64(len(product.Prices))
	}
}

type importExisting struct {
	groups    map[string]bool
	products  map[string]bool
	items     map[string]bool
	tonWallet bool
}

func (r *PaymentRepository) importExistingKeys(ctx context.Context, workspaceID string) (importExisting, error) {
	existing := importExisting{groups: make(map[string]bool), products: make(map[string]bool), items: make(map[string]bool)}
	groupRows, err := r.executor.QueryContext(ctx, `SELECT code FROM payment_product_group WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return existing, err
	}
	for groupRows.Next() {
		var key string
		if err := groupRows.Scan(&key); err != nil {
			groupRows.Close()
			return existing, err
		}
		existing.groups[key] = true
	}
	if err := groupRows.Close(); err != nil {
		return existing, err
	}
	productRows, err := r.executor.QueryContext(ctx, `SELECT id FROM payment_product WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return existing, err
	}
	for productRows.Next() {
		var key string
		if err := productRows.Scan(&key); err != nil {
			productRows.Close()
			return existing, err
		}
		existing.products[key] = true
	}
	if err := productRows.Close(); err != nil {
		return existing, err
	}
	itemRows, err := r.executor.QueryContext(ctx, `SELECT id FROM payment_item WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return existing, err
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var key string
		if err := itemRows.Scan(&key); err != nil {
			return existing, err
		}
		existing.items[key] = true
	}
	if err := itemRows.Err(); err != nil {
		return existing, err
	}
	if err := r.executor.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM payment_ton_wallet WHERE workspace_id = $1)`, workspaceID).Scan(&existing.tonWallet); err != nil {
		return existing, err
	}
	return existing, nil
}

func previewHasConflict(preview ImportPreview, kind, key string) bool {
	for _, conflict := range preview.Conflicts {
		if conflict.Type == kind && conflict.Key == key {
			return true
		}
	}
	return false
}

func flattenProducts(pkg ExportPackage) []ExportProduct {
	total := len(pkg.Products)
	for _, group := range pkg.Groups {
		total += len(group.Products)
	}
	result := make([]ExportProduct, 0, total)
	for _, group := range pkg.Groups {
		for _, product := range group.Products {
			if product.GroupCode == nil {
				product.GroupCode = &group.Code
			}
			result = append(result, product)
		}
	}
	result = append(result, pkg.Products...)
	return result
}

func normalizeExportPackage(pkg ExportPackage) ExportPackage {
	for index := range pkg.Groups {
		group := &pkg.Groups[index]
		if group.TitleKey == nil && group.Code != "" {
			group.TitleKey = stringPtr("payment.group." + group.Code + ".title")
		}
		if group.DescriptionKey == nil && group.Code != "" {
			group.DescriptionKey = stringPtr("payment.group." + group.Code + ".description")
		}
		for productIndex := range group.Products {
			normalizeExportProduct(&group.Products[productIndex])
		}
	}
	for index := range pkg.Products {
		normalizeExportProduct(&pkg.Products[index])
	}
	for index := range pkg.Items {
		item := &pkg.Items[index]
		if item.TitleKey == "" && item.ID != "" {
			item.TitleKey = "payment.item." + item.ID + ".title"
		}
		if item.DescriptionKey == nil && item.ID != "" {
			item.DescriptionKey = stringPtr("payment.item." + item.ID + ".description")
		}
	}
	return pkg
}

func normalizeExportProduct(product *ExportProduct) {
	if product == nil || product.ID == "" {
		return
	}
	if product.TitleKey == "" {
		product.TitleKey = "payment.product." + product.ID + ".title"
	}
	if product.DescriptionKey == nil {
		product.DescriptionKey = stringPtr("payment.product." + product.ID + ".description")
	}
}

func stringPtr(value string) *string {
	return &value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultJSON(value []byte, fallback string) string {
	if len(value) == 0 {
		return fallback
	}
	return string(value)
}

func defaultTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}

func nullableString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullableInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func nullableUint64(value *uint64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

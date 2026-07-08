package repository

import (
	"context"
	"database/sql"
	"time"
)

func (r *PaymentRepository) Export(ctx context.Context, workspaceID string, req ExportRequest) (ExportPackage, error) {
	workspaceID, err := requireWorkspaceID(workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	localizations, err := r.exportLocalizations(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	groups, err := r.exportGroups(ctx, workspaceID, localizations)
	if err != nil {
		return ExportPackage{}, err
	}
	items, err := r.exportItems(ctx, workspaceID, localizations)
	if err != nil {
		return ExportPackage{}, err
	}
	products, err := r.exportProducts(ctx, workspaceID, localizations)
	if err != nil {
		return ExportPackage{}, err
	}
	productItems, err := r.exportProductItems(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	prices, err := r.exportPrices(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	groupIndex := make(map[string]int, len(groups))
	for index := range groups {
		groupIndex[groups[index].Code] = index
	}
	rootProducts := make([]ExportProduct, 0)
	for _, product := range products {
		product.Items = productItems[product.ID]
		product.Prices = prices[product.ID]
		if product.GroupCode != nil {
			if index, ok := groupIndex[*product.GroupCode]; ok {
				groups[index].Products = append(groups[index].Products, product)
				continue
			}
		}
		rootProducts = append(rootProducts, product)
	}
	return ExportPackage{
		Format: ExportFormat, Service: "payment", CreatedAt: now.UTC(),
		Groups: groups, Products: rootProducts, Items: items,
	}, nil
}

func (r *PaymentRepository) exportGroups(ctx context.Context, workspaceID string, localizations map[string]map[string]string) ([]ExportProductGroup, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT code, title_key, description_key, position, is_active
FROM payment_product_group
WHERE workspace_id = $1
ORDER BY position, code`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ExportProductGroup, 0)
	for rows.Next() {
		var group ExportProductGroup
		var titleKey, descriptionKey sql.NullString
		if err := rows.Scan(&group.Code, &titleKey, &descriptionKey, &group.Position, &group.IsActive); err != nil {
			return nil, err
		}
		group.TitleKey = exportNullStringPtr(titleKey)
		group.DescriptionKey = exportNullStringPtr(descriptionKey)
		group.Localization = exportText(localizations, group.TitleKey, group.DescriptionKey)
		result = append(result, group)
	}
	return result, rows.Err()
}

func (r *PaymentRepository) exportProducts(ctx context.Context, workspaceID string, localizations map[string]map[string]string) ([]ExportProduct, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT id, group_code, title_key, description_key, target, image_url, link_url, size_label,
       period_seconds, trial_duration_seconds, quantity_mode, position,
       global_limit, global_interval, global_interval_count, user_limit, user_interval,
       user_interval_count, available_from, available_until, is_visible, is_closed
FROM payment_product
WHERE workspace_id = $1
ORDER BY position, id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ExportProduct, 0)
	for rows.Next() {
		var product ExportProduct
		var groupCode, descriptionKey, imageURL, linkURL, sizeLabel sql.NullString
		var periodSeconds, trialDurationSeconds sql.NullInt64
		if err := rows.Scan(
			&product.ID, &groupCode, &product.TitleKey, &descriptionKey, &product.Target,
			&imageURL, &linkURL, &sizeLabel, &periodSeconds, &trialDurationSeconds,
			&product.QuantityMode, &product.Position, &product.GlobalLimit, &product.GlobalInterval,
			&product.GlobalIntervalCount, &product.UserLimit, &product.UserInterval,
			&product.UserIntervalCount, &product.AvailableFrom, &product.AvailableUntil,
			&product.IsVisible, &product.IsClosed,
		); err != nil {
			return nil, err
		}
		product.GroupCode = exportNullStringPtr(groupCode)
		product.DescriptionKey = exportNullStringPtr(descriptionKey)
		product.ImageURL = exportNullStringPtr(imageURL)
		product.LinkURL = exportNullStringPtr(linkURL)
		product.SizeLabel = exportNullStringPtr(sizeLabel)
		product.PeriodSeconds = exportNullInt64Ptr(periodSeconds)
		product.TrialDurationSeconds = exportNullInt64Ptr(trialDurationSeconds)
		product.Localization = exportText(localizations, &product.TitleKey, product.DescriptionKey)
		if len(product.Target) == 0 || string(product.Target) == "null" {
			product.Target = nil
		}
		result = append(result, product)
	}
	return result, rows.Err()
}

func (r *PaymentRepository) exportItems(ctx context.Context, workspaceID string, localizations map[string]map[string]string) ([]ExportItem, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT id, item_type, title_key, description_key, rarity, position
FROM payment_item
WHERE workspace_id = $1
ORDER BY position, id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ExportItem, 0)
	for rows.Next() {
		var item ExportItem
		var itemType, descriptionKey sql.NullString
		if err := rows.Scan(&item.ID, &itemType, &item.TitleKey, &descriptionKey, &item.Rarity, &item.Position); err != nil {
			return nil, err
		}
		item.ItemType = exportNullStringPtr(itemType)
		item.DescriptionKey = exportNullStringPtr(descriptionKey)
		item.Localization = exportText(localizations, &item.TitleKey, item.DescriptionKey)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *PaymentRepository) exportProductItems(ctx context.Context, workspaceID string) (map[string][]ExportProductItem, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT product_id, item_id, reward_type, quantity, scale, duration_unit
FROM payment_product_item
WHERE workspace_id = $1
ORDER BY product_id, item_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]ExportProductItem)
	for rows.Next() {
		var productID string
		var item ExportProductItem
		var durationUnit sql.NullString
		if err := rows.Scan(&productID, &item.ItemID, &item.RewardType, &item.Quantity, &item.Scale, &durationUnit); err != nil {
			return nil, err
		}
		item.DurationUnit = exportNullStringPtr(durationUnit)
		result[productID] = append(result[productID], item)
	}
	return result, rows.Err()
}

func (r *PaymentRepository) exportPrices(ctx context.Context, workspaceID string) (map[string][]ExportPrice, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT id, product_id, asset_code, list_amount_minor, discount_amount_minor, pricing_mode,
       reference_asset_code, reference_list_amount_minor, reference_discount_amount_minor,
       coefficient, is_promotion, starts_at, ends_at
FROM payment_price
WHERE workspace_id = $1
ORDER BY product_id, asset_code, starts_at, id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]ExportPrice)
	for rows.Next() {
		var productID string
		var price ExportPrice
		var referenceAssetCode, coefficient sql.NullString
		var referenceListAmountMinor, referenceDiscountAmountMinor sql.NullInt64
		if err := rows.Scan(
			&price.ID, &productID, &price.AssetCode, &price.ListAmountMinor,
			&price.DiscountAmountMinor, &price.PricingMode, &referenceAssetCode,
			&referenceListAmountMinor, &referenceDiscountAmountMinor, &coefficient,
			&price.IsPromotion, &price.StartsAt, &price.EndsAt,
		); err != nil {
			return nil, err
		}
		price.ReferenceAssetCode = exportNullStringPtr(referenceAssetCode)
		price.ReferenceListAmountMinor = exportNullUint64Ptr(referenceListAmountMinor)
		price.ReferenceDiscountAmountMinor = exportNullUint64Ptr(referenceDiscountAmountMinor)
		price.Coefficient = exportNullStringPtr(coefficient)
		result[productID] = append(result[productID], price)
	}
	return result, rows.Err()
}

func (r *PaymentRepository) exportLocalizations(ctx context.Context, workspaceID string) (map[string]map[string]string, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT localization_key, locale, value
FROM payment_localization
WHERE workspace_id = $1
ORDER BY localization_key, locale`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]map[string]string)
	for rows.Next() {
		var key, locale, value string
		if err := rows.Scan(&key, &locale, &value); err != nil {
			return nil, err
		}
		if result[key] == nil {
			result[key] = make(map[string]string)
		}
		result[key][locale] = value
	}
	return result, rows.Err()
}

func exportText(localizations map[string]map[string]string, titleKey *string, descriptionKey *string) map[string]ExportText {
	if titleKey == nil && descriptionKey == nil {
		return nil
	}
	result := make(map[string]ExportText)
	if titleKey != nil {
		for locale, value := range localizations[*titleKey] {
			text := result[locale]
			text.Title = value
			result[locale] = text
		}
	}
	if descriptionKey != nil {
		for locale, value := range localizations[*descriptionKey] {
			text := result[locale]
			text.Description = value
			result[locale] = text
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func exportNullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func exportNullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func exportNullUint64Ptr(value sql.NullInt64) *uint64 {
	if !value.Valid || value.Int64 < 0 {
		return nil
	}
	out := uint64(value.Int64)
	return &out
}

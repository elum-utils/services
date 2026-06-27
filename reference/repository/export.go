package repository

import (
	"context"
	"time"
)

func (r *Repository) Export(ctx context.Context, workspaceID string, req ExportRequest) (ExportPackage, error) {
	if err := requireWorkspace(workspaceID); err != nil {
		return ExportPackage{}, err
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	items, err := r.AdminListItems(ctx, ListItemsParams{
		WorkspaceID: workspaceID, OnlyNotDeleted: req.OnlyNotDeleted,
		Limit: 100000, Offset: 0,
	})
	if err != nil {
		return ExportPackage{}, err
	}
	localizations, err := r.exportLocalizations(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	out := ExportPackage{
		Format: ExportFormat, Service: "reference", CreatedAt: now.UTC(),
		Items: make([]ExportItem, 0, len(items)),
	}
	for _, item := range items {
		value := ExportItem{
			Key: item.Key, Type: item.Type, Payload: item.Payload,
			IsActive: item.IsActive, Deleted: item.DeletedAt != nil,
			Localization: localizations[item.Key],
		}
		out.Items = append(out.Items, value)
	}
	return out, nil
}

func (r *Repository) exportLocalizations(ctx context.Context, workspaceID string) (map[string]map[string]ExportText, error) {
	rows, err := r.executor.QueryContext(ctx, `
SELECT item_key, locale, title, description
FROM reference_localization
WHERE workspace_id = ?
ORDER BY item_key, locale`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]map[string]ExportText)
	for rows.Next() {
		var itemKey, locale string
		var text ExportText
		if err := rows.Scan(&itemKey, &locale, &text.Title, &text.Description); err != nil {
			return nil, err
		}
		if result[itemKey] == nil {
			result[itemKey] = make(map[string]ExportText)
		}
		result[itemKey][locale] = text
	}
	return result, rows.Err()
}

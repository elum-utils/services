package repository

import (
	"context"
	"sort"
	"time"
)

func (r *Repository) Export(ctx context.Context, workspaceID string, req ExportRequest) (ExportPackage, error) {
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	promos, err := r.ListPromos(ctx, workspaceID, 100000, 0)
	if err != nil {
		return ExportPackage{}, err
	}
	out := ExportPackage{Format: ExportFormat, Service: "promo", CreatedAt: now.UTC(), Promos: make([]ExportPromo, 0, len(promos))}
	items := make(exportItemCollector)
	for _, promo := range promos {
		localizations, err := r.ListLocalizations(ctx, workspaceID, promo.ID)
		if err != nil {
			return ExportPackage{}, err
		}
		rewards, err := r.ListRewards(ctx, workspaceID, promo.ID)
		if err != nil {
			return ExportPackage{}, err
		}
		item := ExportPromo{
			Code: promo.Code, Payload: promo.Payload, Target: nullableJSON(promo.Target),
			MaxActivations: promo.MaxActivations, IsActive: promo.IsActive,
			StartAt: promo.StartAt, EndAt: promo.EndAt,
			Localization: make(map[string]ExportText, len(localizations)),
			Rewards:      make([]ExportReward, 0, len(rewards)),
		}
		for _, localization := range localizations {
			item.Localization[localization.Locale] = ExportText{Title: localization.Title, Description: localization.Description}
		}
		for _, reward := range rewards {
			item.Rewards = append(item.Rewards, ExportReward{
				Key: reward.Key, Type: reward.Type, Quantity: reward.Quantity, Scale: reward.Scale, Unit: reward.Unit,
			})
			items.add(reward.Key)
		}
		out.Promos = append(out.Promos, item)
	}
	out.Items = items.list()
	return out, nil
}

func nullableJSON(value []byte) []byte {
	if len(value) == 0 || string(value) == "null" {
		return nil
	}
	return value
}

type exportItemCollector map[string]struct{}

func (c exportItemCollector) add(id string) {
	if id == "" {
		return
	}
	c[id] = struct{}{}
}

func (c exportItemCollector) list() []ExportItem {
	if len(c) == 0 {
		return nil
	}
	ids := make([]string, 0, len(c))
	for id := range c {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]ExportItem, 0, len(ids))
	for index, id := range ids {
		items = append(items, ExportItem{ID: id, Position: int32((index + 1) * 10)})
	}
	return items
}

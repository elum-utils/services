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
	promoRows, err := r.q.ListExportPromos(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	localizationRows, err := r.q.ListExportLocalizations(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	rewardRows, err := r.q.ListExportRewards(ctx, workspaceID)
	if err != nil {
		return ExportPackage{}, err
	}
	out := ExportPackage{
		Format:    ExportFormat,
		Service:   "promo",
		CreatedAt: now.UTC(),
		Promos:    make([]ExportPromo, 0, len(promoRows)),
	}
	items := make(exportItemCollector)

	promoIndexByID := make(map[int64]int, len(promoRows))
	for _, row := range promoRows {
		promo := mapPromo(row)
		item := ExportPromo{
			Code:           promo.Code,
			Payload:        promo.Payload,
			Target:         nullableJSON(promo.Target),
			MaxActivations: promo.MaxActivations,
			IsActive:       promo.IsActive,
			StartAt:        promo.StartAt,
			EndAt:          promo.EndAt,
			Localization:   make(map[string]ExportText),
			Rewards:        make([]ExportReward, 0),
		}
		promoIndexByID[row.ID] = len(out.Promos)
		out.Promos = append(out.Promos, item)
	}

	for _, localization := range localizationRows {
		index, ok := promoIndexByID[localization.PromoID]
		if !ok {
			continue
		}
		out.Promos[index].Localization[localization.Locale] = ExportText{
			Title:       localization.Title,
			Description: localization.Description,
		}
	}

	for _, reward := range rewardRows {
		index, ok := promoIndexByID[reward.PromoID]
		if !ok {
			continue
		}
		out.Promos[index].Rewards = append(out.Promos[index].Rewards, ExportReward{
			Key:      reward.RewardKey,
			Type:     string(reward.RewardType),
			Quantity: reward.Quantity,
			Scale:    uint16(reward.Scale),
			Unit:     promoDurationUnitPtr(reward.DurationUnit),
		})
		items.add(reward.RewardKey)
	}

	for index := range out.Promos {
		if len(out.Promos[index].Localization) == 0 {
			out.Promos[index].Localization = nil
		}
		if len(out.Promos[index].Rewards) == 0 {
			out.Promos[index].Rewards = nil
		}
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
		items = append(items, ExportItem{
			ID:       id,
			Position: int32((index + 1) * 10),
		})
	}
	return items
}

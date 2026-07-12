package admin

import (
	"context"
	"strings"

	"github.com/elum-utils/services/control/repository"
)

func (a *Admin) AppendAudit(ctx context.Context, params AuditEventParams) error {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.AppendAudit(mergedCtx, repository.AuditEvent{
		WorkspaceID: strings.TrimSpace(
			params.WorkspaceID,
		), ActorID: strings.TrimSpace(params.ActorID), MethodKey: strings.TrimSpace(params.MethodKey),
		TargetType: strings.TrimSpace(
			params.TargetType,
		), TargetID: strings.TrimSpace(params.TargetID), Result: strings.TrimSpace(params.Result), RequestID: strings.TrimSpace(params.RequestID),
		BeforeData: params.BeforeData, AfterData: params.AfterData,
	})
}

func (a *Admin) ListAudit(ctx context.Context, workspaceID string, page Page) ([]AuditEventModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	limit, offset := normalizePage(page)
	items, err := a.repository.ListAudit(mergedCtx, strings.TrimSpace(workspaceID), limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]AuditEventModel, 0, len(items))
	for _, item := range items {
		result = append(
			result,
			AuditEventModel{
				ID:          item.ID,
				WorkspaceID: item.WorkspaceID,
				ActorID:     item.ActorID,
				MethodKey:   item.MethodKey,
				TargetType:  item.TargetType,
				TargetID:    item.TargetID,
				Result:      item.Result,
				RequestID:   item.RequestID,
				BeforeData:  item.BeforeData,
				AfterData:   item.AfterData,
				OccurredAt:  item.OccurredAt,
			},
		)
	}
	return result, nil
}

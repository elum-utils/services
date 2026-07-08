package repository

import (
	"context"
	"database/sql"

	controlsqlc "github.com/elum-utils/services/control/sqlc"
	"github.com/google/uuid"
)

func (r *Repository) AppendAudit(ctx context.Context, event AuditEvent) error {
	if err := required(event.MethodKey, event.Result); err != nil {
		return err
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	return r.q.CreateAuditEvent(ctx, controlsqlc.CreateAuditEventParams{
		ID: event.ID, WorkspaceID: nullableString(event.WorkspaceID), ActorID: nullableString(event.ActorID), MethodKey: event.MethodKey,
		TargetType: event.TargetType, TargetID: event.TargetID, BeforeData: rawMessageParam(event.BeforeData), AfterData: rawMessageParam(event.AfterData),
		Result: event.Result, RequestID: event.RequestID,
	})
}

func (r *Repository) ListAudit(ctx context.Context, workspaceID string, limit, offset int32) ([]AuditEvent, error) {
	rows, err := r.q.ListAuditEvents(ctx, controlsqlc.ListAuditEventsParams{WorkspaceID: nullableString(workspaceID), Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	result := make([]AuditEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, AuditEvent{ID: row.ID, WorkspaceID: valueString(row.WorkspaceID), ActorID: valueString(row.ActorID), MethodKey: row.MethodKey, TargetType: row.TargetType, TargetID: row.TargetID, BeforeData: nullRawMessage(row.BeforeData), AfterData: nullRawMessage(row.AfterData), Result: row.Result, RequestID: row.RequestID, OccurredAt: row.OccurredAt})
	}
	return result, nil
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
func valueString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	json "github.com/goccy/go-json"
	"math/big"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	"github.com/jackc/pgx/v5/pgconn"

	cpasqlc "github.com/elum-utils/services/cpa/sqlc"
)

type UserScope struct {
	WorkspaceID    string
	CPAID          string
	AppID          int64
	PlatformID     int64
	PlatformUserID string
}

type IssueResult struct {
	Assignment    Assignment
	Rewards       []Reward
	AlreadyIssued bool
}

type CompleteResult struct {
	Assignment  Assignment
	Rewards     []Reward
	AlreadyDone bool
}

func (r *Repository) GetAssignment(ctx context.Context, scope UserScope) (Assignment, error) {
	row, err := r.q.GetAssignment(ctx, assignmentParams(scope))
	if err != nil {
		return Assignment{}, err
	}
	return mapAssignment(row), nil
}

func (r *Repository) FindAssignment(ctx context.Context, scope UserScope) (*Assignment, error) {
	value, err := r.GetAssignment(ctx, scope)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (r *Repository) ListUserAssignments(ctx context.Context, scope UserScope) ([]Assignment, error) {
	rows, err := r.q.ListUserAssignments(ctx, cpasqlc.ListUserAssignmentsParams{
		WorkspaceID:    scope.WorkspaceID,
		AppID:          scope.AppID,
		PlatformID:     scope.PlatformID,
		PlatformUserID: scope.PlatformUserID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapAssignment(row))
	}
	return result, nil
}

func (r *Repository) ListAssignments(ctx context.Context, workspaceID, cpaID, status string, limit, offset int32) ([]Assignment, error) {
	limit, offset = normalizePage(limit, offset)
	rows, err := r.q.AdminListAssignments(ctx, cpasqlc.AdminListAssignmentsParams{
		WorkspaceID: workspaceID,
		CpaID:       cpaID,
		Column3:     status,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapAssignment(row))
	}
	return result, nil
}

func (r *Repository) ListCodes(ctx context.Context, workspaceID, cpaID, status string, limit, offset int32) ([]Code, error) {
	limit, offset = normalizePage(limit, offset)
	rows, err := r.q.AdminListCodes(ctx, cpasqlc.AdminListCodesParams{
		WorkspaceID: workspaceID,
		CpaID:       cpaID,
		Column3:     status,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]Code, 0, len(rows))
	for _, row := range rows {
		result = append(result, Code{
			ID:          uint64(row.ID),
			WorkspaceID: row.WorkspaceID,
			CPAID:       row.CpaID,
			Code:        row.Code,
			Source:      string(row.Source),
			Status:      string(row.Status),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
			DeletedAt:   sqlwrap.NullTimePtr(row.DeletedAt),
		})
	}
	return result, nil
}

func (r *Repository) ListAssignmentEvents(ctx context.Context, workspaceID, cpaID, eventType string, limit, offset int32) ([]AssignmentEvent, error) {
	limit, offset = normalizePage(limit, offset)
	rows, err := r.q.AdminListAssignmentEvents(ctx, cpasqlc.AdminListAssignmentEventsParams{
		WorkspaceID: workspaceID,
		CpaID:       cpaID,
		Column3:     eventType,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]AssignmentEvent, 0, len(rows))
	for _, row := range rows {
		result = append(result, AssignmentEvent{
			ID:           uint64(row.ID),
			WorkspaceID:  row.WorkspaceID,
			CPAID:        row.CpaID,
			AssignmentID: uint64(row.AssignmentID),
			EventType:    string(row.EventType),
			OccurredAt:   row.OccurredAt,
		})
	}
	return result, nil
}

func (r *Repository) Issue(ctx context.Context, scope UserScope) (IssueResult, error) {
	if err := requireScope(scope.WorkspaceID, scope.CPAID); err != nil {
		return IssueResult{}, err
	}
	var result IssueResult
	err := r.WithTx(ctx, func(txRepo *Repository) error {
		existing, err := txRepo.q.GetAssignment(ctx, assignmentParams(scope))
		if err == nil {
			result.Assignment = mapAssignment(existing)
			result.AlreadyIssued = true
			result.Rewards, err = txRepo.ListRewards(ctx, scope.WorkspaceID, scope.CPAID)
			return err
		}
		if !isNoRows(err) {
			return err
		}

		offer, err := txRepo.q.GetActiveOfferForUpdate(ctx, cpasqlc.GetActiveOfferForUpdateParams{
			WorkspaceID: scope.WorkspaceID,
			ID:          scope.CPAID,
		})
		if err != nil {
			return err
		}
		existing, err = txRepo.q.GetAssignmentForUpdate(ctx, assignmentForUpdateParams(scope))
		if err == nil {
			result.Assignment = mapAssignment(existing)
			result.AlreadyIssued = true
			result.Rewards, err = txRepo.ListRewards(ctx, scope.WorkspaceID, scope.CPAID)
			return err
		}
		if !isNoRows(err) {
			return err
		}

		code, codeID, err := txRepo.allocateCode(ctx, offer)
		if err != nil {
			return err
		}
		id, err := txRepo.q.CreateAssignment(ctx, cpasqlc.CreateAssignmentParams{
			WorkspaceID:    scope.WorkspaceID,
			CpaID:          scope.CPAID,
			AppID:          scope.AppID,
			PlatformID:     scope.PlatformID,
			PlatformUserID: scope.PlatformUserID,
			CodeID: sqlwrap.NullFromPtr(codeID, func(v uint64) sql.NullInt64 {
				return sql.NullInt64{Int64: int64(v), Valid: true}
			}),
			Code:     code,
			CodeMode: offer.CodeMode,
		})
		if err != nil {
			return err
		}
		if codeID != nil {
			affected, err := txRepo.q.MarkCodeIssued(ctx, int64(*codeID))
			if err != nil {
				return err
			}
			if affected != 1 {
				return ErrNoCodesAvailable
			}
		}
		row, err := txRepo.q.GetAssignmentByID(ctx, cpasqlc.GetAssignmentByIDParams{
			WorkspaceID: scope.WorkspaceID,
			ID:          id,
		})
		if err != nil {
			return err
		}
		result.Assignment = mapAssignment(row)
		result.Rewards, err = txRepo.ListRewards(ctx, scope.WorkspaceID, scope.CPAID)
		if err != nil {
			return err
		}
		return txRepo.recordEvent(ctx, result.Assignment, result.Rewards, StatusIssued)
	})
	return result, err
}

func (r *Repository) Complete(ctx context.Context, scope UserScope) (CompleteResult, error) {
	if err := requireScope(scope.WorkspaceID, scope.CPAID); err != nil {
		return CompleteResult{}, err
	}
	existing, err := r.q.GetAssignment(ctx, assignmentParams(scope))
	if err == nil && existing.Status == cpasqlc.CpaAssignmentStatusCompleted {
		rewards, err := r.ListRewards(ctx, scope.WorkspaceID, scope.CPAID)
		if err != nil {
			return CompleteResult{}, err
		}
		return CompleteResult{
			Assignment:  mapAssignment(existing),
			Rewards:     rewards,
			AlreadyDone: true,
		}, nil
	}
	if err != nil && !isNoRows(err) {
		return CompleteResult{}, err
	}
	var result CompleteResult
	err = r.WithTx(ctx, func(txRepo *Repository) error {
		row, err := txRepo.q.GetAssignmentForUpdate(ctx, assignmentForUpdateParams(scope))
		if err != nil {
			return err
		}
		result.Assignment = mapAssignment(row)
		result.Rewards, err = txRepo.ListRewards(ctx, scope.WorkspaceID, scope.CPAID)
		if err != nil {
			return err
		}
		if result.Assignment.Status == StatusCompleted {
			result.AlreadyDone = true
			return nil
		}
		affected, err := txRepo.q.CompleteAssignment(ctx, cpasqlc.CompleteAssignmentParams{
			WorkspaceID: scope.WorkspaceID,
			ID:          int64(result.Assignment.ID),
		})
		if err != nil {
			return err
		}
		if affected != 1 {
			return errors.New("cpa: assignment completion conflict")
		}
		if result.Assignment.CodeID != nil {
			if _, err := txRepo.q.MarkCodeCompleted(ctx, int64(*result.Assignment.CodeID)); err != nil {
				return err
			}
		}
		now := time.Now()
		result.Assignment.Status = StatusCompleted
		result.Assignment.CompletedAt = &now
		return txRepo.recordEvent(ctx, result.Assignment, result.Rewards, StatusCompleted)
	})
	return result, err
}

func (r *Repository) AddCodes(ctx context.Context, workspaceID, cpaID string, codes []string) (int, error) {
	if err := requireScope(workspaceID, cpaID); err != nil {
		return 0, err
	}
	added := 0
	err := r.WithTx(ctx, func(txRepo *Repository) error {
		for _, code := range codes {
			if code == "" {
				continue
			}
			affected, err := txRepo.q.AdminAddCode(ctx, cpasqlc.AdminAddCodeParams{
				WorkspaceID: workspaceID,
				CpaID:       cpaID,
				Code:        code,
				Source:      cpasqlc.CpaCodeSourcePool,
			})
			if err != nil {
				return err
			}
			added += int(affected)
		}
		return nil
	})
	return added, err
}

func (r *Repository) DeleteAvailableCodes(ctx context.Context, workspaceID, cpaID string) (int64, error) {
	return r.q.AdminDeleteAvailableCodes(ctx, cpasqlc.AdminDeleteAvailableCodesParams{
		WorkspaceID: workspaceID,
		CpaID:       cpaID,
	})
}

func (r *Repository) DeleteIssuedCodes(ctx context.Context, workspaceID, cpaID string) (int64, error) {
	var affected int64
	err := r.WithTx(ctx, func(txRepo *Repository) error {
		var err error
		affected, err = txRepo.q.AdminDeleteIssuedCodes(ctx, cpasqlc.AdminDeleteIssuedCodesParams{
			WorkspaceID: workspaceID,
			CpaID:       cpaID,
		})
		if err != nil {
			return err
		}
		_, err = txRepo.q.AdminDeleteIssuedCodeRows(ctx, cpasqlc.AdminDeleteIssuedCodeRowsParams{
			WorkspaceID: workspaceID,
			CpaID:       cpaID,
		})
		return err
	})
	return affected, err
}

func (r *Repository) DeleteCompletedCodes(ctx context.Context, workspaceID, cpaID string) (int64, error) {
	var affected int64
	err := r.WithTx(ctx, func(txRepo *Repository) error {
		var err error
		affected, err = txRepo.q.AdminDeleteCompletedCodes(ctx, cpasqlc.AdminDeleteCompletedCodesParams{
			WorkspaceID: workspaceID,
			CpaID:       cpaID,
		})
		if err != nil {
			return err
		}
		_, err = txRepo.q.AdminDeleteCompletedCodeRows(ctx, cpasqlc.AdminDeleteCompletedCodeRowsParams{
			WorkspaceID: workspaceID,
			CpaID:       cpaID,
		})
		return err
	})
	return affected, err
}

func (r *Repository) allocateCode(ctx context.Context, offer cpasqlc.CpaOffer) (string, *uint64, error) {
	if offer.CodeMode == cpasqlc.CpaCodeModeSharedCode {
		if !offer.SharedCode.Valid || offer.SharedCode.String == "" {
			return "", nil, errors.New("cpa: shared code is empty")
		}
		return offer.SharedCode.String, nil, nil
	}
	if !offer.CodeSource.Valid {
		return "", nil, ErrInvalidCodeConfig
	}
	if offer.CodeSource.CpaCodeSource == cpasqlc.CpaCodeSourcePool {
		row, err := r.q.GetAvailableCodeForUpdate(ctx, cpasqlc.GetAvailableCodeForUpdateParams{
			WorkspaceID: offer.WorkspaceID,
			CpaID:       offer.ID,
		})
		if isNoRows(err) {
			return "", nil, ErrNoCodesAvailable
		}
		if err != nil {
			return "", nil, err
		}
		id := uint64(row.ID)
		return row.Code, &id, nil
	}
	if !offer.GeneratedLength.Valid || !offer.GeneratedAlphabet.Valid {
		return "", nil, ErrInvalidCodeConfig
	}
	for range 16 {
		code, err := randomCode(int(offer.GeneratedLength.Int16), offer.GeneratedAlphabet.String)
		if err != nil {
			return "", nil, err
		}
		id, err := r.q.CreateGeneratedCode(ctx, cpasqlc.CreateGeneratedCodeParams{
			WorkspaceID: offer.WorkspaceID,
			CpaID:       offer.ID,
			Code:        code,
		})
		if err == nil {
			value := uint64(id)
			return code, &value, nil
		}
		if !isUniqueViolation(err) {
			return "", nil, err
		}
	}
	return "", nil, errors.New("cpa: generated code collision limit reached")
}

func (r *Repository) recordEvent(ctx context.Context, assignment Assignment, rewards []Reward, eventType string) error {
	_, err := r.q.CreateAssignmentEvent(ctx, cpasqlc.CreateAssignmentEventParams{
		WorkspaceID:  assignment.WorkspaceID,
		CpaID:        assignment.CPAID,
		AssignmentID: int64(assignment.ID),
		EventType:    cpasqlc.CpaAssignmentEventType(eventType),
	})
	if err != nil {
		return err
	}
	payload := callbackPayload{
		AssignmentID:   assignment.ID,
		WorkspaceID:    assignment.WorkspaceID,
		CPAID:          assignment.CPAID,
		AppID:          assignment.AppID,
		PlatformID:     assignment.PlatformID,
		PlatformUserID: assignment.PlatformUserID,
		Code:           assignment.Code,
		CodeMode:       assignment.CodeMode,
		Status:         eventType,
		Rewards:        make([]callbackReward, 0, len(rewards)),
	}
	for _, reward := range rewards {
		payload.Rewards = append(payload.Rewards, callbackReward{
			Key:      reward.Key,
			Type:     reward.Type,
			Quantity: reward.Quantity,
			Scale:    reward.Scale,
			Unit:     reward.Unit,
		})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	eventKey := fmt.Sprintf("cpa.%s:%d", eventType, assignment.ID)
	_, err = r.callbacks.CreateEvent(ctx, callbackutil.CreateParams{
		SourceService:      "cpa",
		EventType:          "cpa." + eventType,
		EventKey:           eventKey,
		IdempotencyKey:     eventKey,
		Payload:            raw,
		PayloadContentType: callbackutil.JSONContentType,
	})
	return err
}

type callbackPayload struct {
	AssignmentID   uint64           `json:"assignment_id"`
	WorkspaceID    string           `json:"workspace_id"`
	CPAID          string           `json:"cpa_id"`
	AppID          int64            `json:"app_id"`
	PlatformID     int64            `json:"platform_id"`
	PlatformUserID string           `json:"platform_user_id"`
	Code           string           `json:"code"`
	CodeMode       string           `json:"code_mode"`
	Status         string           `json:"status"`
	Rewards        []callbackReward `json:"rewards"`
}

type callbackReward struct {
	Key      string  `json:"key"`
	Type     string  `json:"type"`
	Quantity int64   `json:"quantity"`
	Scale    uint16  `json:"scale"`
	Unit     *string `json:"unit,omitempty"`
}

func assignmentParams(scope UserScope) cpasqlc.GetAssignmentParams {
	return cpasqlc.GetAssignmentParams{
		WorkspaceID:    scope.WorkspaceID,
		CpaID:          scope.CPAID,
		AppID:          scope.AppID,
		PlatformID:     scope.PlatformID,
		PlatformUserID: scope.PlatformUserID,
	}
}

func assignmentForUpdateParams(scope UserScope) cpasqlc.GetAssignmentForUpdateParams {
	return cpasqlc.GetAssignmentForUpdateParams(assignmentParams(scope))
}

func mapAssignment(row cpasqlc.CpaAssignment) Assignment {
	var codeID *uint64
	if row.CodeID.Valid {
		value := uint64(row.CodeID.Int64)
		codeID = &value
	}
	return Assignment{
		ID:             uint64(row.ID),
		WorkspaceID:    row.WorkspaceID,
		CPAID:          row.CpaID,
		AppID:          row.AppID,
		PlatformID:     row.PlatformID,
		PlatformUserID: row.PlatformUserID,
		CodeID:         codeID,
		Code:           row.Code,
		CodeMode:       string(row.CodeMode),
		Status:         string(row.Status),
		IssuedAt:       row.IssuedAt,
		CompletedAt:    sqlwrap.NullTimePtr(row.CompletedAt),
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func randomCode(length int, alphabet string) (string, error) {
	runes := []rune(alphabet)
	if length <= 0 || len(runes) < 2 {
		return "", ErrInvalidCodeConfig
	}
	result := make([]rune, length)
	max := big.NewInt(int64(len(runes)))
	for index := range result {
		value, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[index] = runes[value.Int64()]
	}
	return string(result), nil
}

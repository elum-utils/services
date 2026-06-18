package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	json "github.com/goccy/go-json"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
)

func PartnerIssueKey(id uint64) string {
	return PartnerIssueKeyPrefix + strconv.FormatUint(id, 10)
}

func ParsePartnerIssueRef(value string) (uint64, bool) {
	if !strings.HasPrefix(value, PartnerIssueKeyPrefix) {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(value, PartnerIssueKeyPrefix), 10, 64)
	return id, err == nil && id > 0
}

func (r *Repository) SavePartnerConfig(ctx context.Context, params SavePartnerConfigParams) error {
	target := params.Target
	if len(target) == 0 {
		target = []byte("null")
	}
	settings := params.Settings
	if len(settings) == 0 {
		settings = []byte("{}")
	}
	return repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.AdminUpsertPartnerConfig(ctx, tasksqlc.AdminUpsertPartnerConfigParams{
			WorkspaceID: params.WorkspaceID,
			Provider:    params.Provider,
			GroupKey:    params.GroupKey,
			Platform:    params.Platform,
			IsEnabled:   params.IsEnabled,
			Secret:      nullString(params.Secret),
			Target:      target,
			Settings:    settings,
		})
	})
}

func (r *Repository) GetPartnerConfig(ctx context.Context, workspaceID, provider, groupKey, platform string) (PartnerConfig, bool, error) {
	row, err := repositoryValue(ctx, r, func(ctx context.Context) (tasksqlc.TaskPartnerConfig, error) {
		return r.q.AdminGetPartnerConfig(ctx, tasksqlc.AdminGetPartnerConfigParams{
			WorkspaceID: workspaceID, Provider: provider, GroupKey: groupKey, Platform: platform,
		})
	})
	if err != nil {
		if isNoRows(err) {
			return PartnerConfig{}, false, nil
		}
		return PartnerConfig{}, false, err
	}
	return mapPartnerConfig(row), true, nil
}

func (r *Repository) ListPartnerConfigs(ctx context.Context, workspaceID string) ([]PartnerConfig, error) {
	rows, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskPartnerConfig, error) {
		return r.q.AdminListPartnerConfigs(ctx, workspaceID)
	})
	if err != nil {
		return nil, err
	}
	result := make([]PartnerConfig, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapPartnerConfig(row))
	}
	return result, nil
}

func (r *Repository) SavePartnerRewardRule(ctx context.Context, params SavePartnerRewardRuleParams) error {
	externalType := params.ExternalType
	if externalType == "" {
		externalType = "*"
	}
	rewardType := params.Reward.Type
	if rewardType == "" {
		rewardType = "quantity"
	}
	return repositoryExec(ctx, r, func(ctx context.Context) error {
		return r.q.AdminUpsertPartnerRewardRule(ctx, tasksqlc.AdminUpsertPartnerRewardRuleParams{
			WorkspaceID:  params.WorkspaceID,
			Provider:     params.Provider,
			GroupKey:     params.GroupKey,
			ExternalType: externalType,
			RewardKey:    params.Reward.Key,
			RewardType:   tasksqlc.TaskPartnerRewardRuleRewardType(rewardType),
			Quantity:     params.Reward.Quantity,
			DurationUnit: tasksqlc.NullTaskPartnerRewardRuleDurationUnit{
				TaskPartnerRewardRuleDurationUnit: tasksqlc.TaskPartnerRewardRuleDurationUnit(taskStringValue(params.Reward.Unit)),
				Valid:                             params.Reward.Unit != nil,
			},
			Position:  params.Position,
			IsEnabled: params.IsEnabled,
		})
	})
}

func (r *Repository) DeletePartnerRewardRule(ctx context.Context, workspaceID, provider, groupKey, externalType, rewardKey string) (int64, error) {
	if externalType == "" {
		externalType = "*"
	}
	return repositoryValue(ctx, r, func(ctx context.Context) (int64, error) {
		return r.q.AdminDeletePartnerRewardRule(ctx, tasksqlc.AdminDeletePartnerRewardRuleParams{
			WorkspaceID: workspaceID, Provider: provider, GroupKey: groupKey,
			ExternalType: externalType, RewardKey: rewardKey,
		})
	})
}

func (r *Repository) PartnerRewards(ctx context.Context, workspaceID, provider, groupKey, externalType string) ([]Reward, error) {
	rows, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskPartnerRewardRule, error) {
		return r.q.ListPartnerRewardRules(ctx, tasksqlc.ListPartnerRewardRulesParams{
			WorkspaceID: workspaceID, Provider: provider, GroupKey: groupKey,
			ExternalType: externalType, ExternalType_2: externalType,
		})
	})
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(rows))
	rewards := make([]Reward, 0, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.RewardKey]; ok {
			continue
		}
		seen[row.RewardKey] = struct{}{}
		rewards = append(rewards, Reward{
			Key: row.RewardKey, Type: string(row.RewardType), Quantity: row.Quantity,
			Unit: nullPartnerDurationUnit(row.DurationUnit),
		})
	}
	return rewards, nil
}

func (r *Repository) CreatePartnerIssue(ctx context.Context, params CreatePartnerIssueParams) (PartnerIssue, bool, error) {
	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	publicPayload := params.PublicPayload
	if len(publicPayload) == 0 {
		publicPayload = []byte("{}")
	}
	privatePayload := params.PrivatePayload
	if len(privatePayload) == 0 {
		privatePayload = []byte("{}")
	}
	issueKey := params.IssueKey
	if issueKey == "" {
		issueKey = fmt.Sprintf("%s:%s:%s:%d:%d:%s:%s", params.Provider, params.GroupKey, params.ExternalID, params.Identity.AppID, params.Identity.PlatformID, params.Identity.PlatformUserID, now.Format("20060102"))
	}
	id, err := repositoryValue(ctx, r, func(ctx context.Context) (int64, error) {
		return r.q.CreatePartnerIssue(ctx, tasksqlc.CreatePartnerIssueParams{
			WorkspaceID: params.Identity.WorkspaceID, Provider: params.Provider, GroupKey: params.GroupKey,
			Platform: params.Platform, ExternalID: params.ExternalID, ExternalType: params.ExternalType, IssueKey: issueKey,
			AppID: params.Identity.AppID, PlatformID: params.Identity.PlatformID, PlatformUserID: params.Identity.PlatformUserID,
			PublicPayload: publicPayload, PrivatePayload: privatePayload, IssuedAt: now, ExpiresAt: nullTime(params.ExpiresAt),
		})
	})
	if err != nil {
		return PartnerIssue{}, false, err
	}
	issue, found, err := r.GetPartnerIssue(ctx, params.Identity.WorkspaceID, uint64(id))
	if err != nil || !found {
		return issue, false, err
	}
	eventKey := "partner.issue:" + issue.IssueKey
	inserted, err := r.recordPartnerStatsEvent(ctx, issue, PartnerStatsEventIssued, eventKey, "", issue.PublicPayload, now)
	return issue, inserted, err
}

func (r *Repository) GetPartnerIssue(ctx context.Context, workspaceID string, id uint64) (PartnerIssue, bool, error) {
	row, err := repositoryValue(ctx, r, func(ctx context.Context) (tasksqlc.TaskPartnerIssue, error) {
		return r.q.GetPartnerIssueByID(ctx, tasksqlc.GetPartnerIssueByIDParams{WorkspaceID: workspaceID, ID: id})
	})
	if err != nil {
		if isNoRows(err) {
			return PartnerIssue{}, false, nil
		}
		return PartnerIssue{}, false, err
	}
	return mapPartnerIssue(row), true, nil
}

func (r *Repository) ListPartnerIssuesForUser(ctx context.Context, identity Identity, provider, groupKey, platform string, now time.Time) ([]PartnerIssue, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskPartnerIssue, error) {
		return r.q.ListPartnerIssuesForUser(ctx, tasksqlc.ListPartnerIssuesForUserParams{
			WorkspaceID: identity.WorkspaceID, Provider: provider, GroupKey: groupKey, Platform: platform,
			AppID: identity.AppID, PlatformID: identity.PlatformID, PlatformUserID: identity.PlatformUserID,
			ExpiresAt: sql.NullTime{Time: now, Valid: true},
		})
	})
	if err != nil {
		return nil, err
	}
	result := make([]PartnerIssue, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapPartnerIssue(row))
	}
	return result, nil
}

func (r *Repository) CompletePartnerIssue(ctx context.Context, workspaceID string, id uint64, status string, payload json.RawMessage, now time.Time) (PartnerIssue, bool, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var issue PartnerIssue
	completed := false
	err := r.WithTx(ctx, func(txRepo *Repository) error {
		row, err := txRepo.q.GetPartnerIssueByIDForUpdate(ctx, tasksqlc.GetPartnerIssueByIDForUpdateParams{WorkspaceID: workspaceID, ID: id})
		if err != nil {
			return err
		}
		issue = mapPartnerIssue(row)
		if issue.Status == PartnerIssueStatusCompleted || issue.Status == PartnerIssueStatusClaimed {
			return nil
		}
		affected, err := txRepo.q.CompletePartnerIssue(ctx, tasksqlc.CompletePartnerIssueParams{
			CompletedAt: nullTime(&now), WorkspaceID: workspaceID, ID: id,
		})
		if err != nil {
			return err
		}
		completed = affected == 1
		if completed {
			issue.Status = PartnerIssueStatusCompleted
			issue.CompletedAt = &now
			eventKey := fmt.Sprintf("partner.completed:%d", issue.ID)
			_, err = txRepo.recordPartnerStatsEvent(ctx, issue, PartnerStatsEventCompleted, eventKey, status, payload, now)
			return err
		}
		return nil
	})
	if err != nil {
		if errorsIsNoRows(err) {
			return PartnerIssue{}, false, nil
		}
		return PartnerIssue{}, false, err
	}
	return issue, completed, nil
}

func (r *Repository) ClaimPartnerIssue(ctx context.Context, identity Identity, issueID uint64, operationID string, now time.Time) (PartnerClaimResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if operationID == "" {
		operationID = fmt.Sprintf("partner-%d-%d", issueID, now.UnixNano())
	}
	result := PartnerClaimResult{Status: ClaimStatusNotFound, OperationID: operationID}
	err := r.WithTx(ctx, func(txRepo *Repository) error {
		row, err := txRepo.q.GetPartnerIssueByIDForUpdate(ctx, tasksqlc.GetPartnerIssueByIDForUpdateParams{
			WorkspaceID: identity.WorkspaceID, ID: issueID,
		})
		if err != nil {
			if isNoRows(err) {
				return nil
			}
			return err
		}
		issue := mapPartnerIssue(row)
		result.Issue = issue
		if issue.AppID != identity.AppID || issue.PlatformID != identity.PlatformID || issue.PlatformUserID != identity.PlatformUserID {
			result.Status = ClaimStatusNotFound
			return nil
		}
		if issue.Status == PartnerIssueStatusClaimed {
			result.Status = ClaimStatusAlreadyDone
			return nil
		}
		if issue.Status != PartnerIssueStatusCompleted {
			result.Status = ClaimStatusNotReady
			return nil
		}
		rewards, err := txRepo.PartnerRewards(ctx, issue.WorkspaceID, issue.Provider, issue.GroupKey, issue.ExternalType)
		if err != nil {
			return err
		}
		rewardPayload, err := json.Marshal(rewards)
		if err != nil {
			return err
		}
		inserted, err := txRepo.q.InsertPartnerRewardGrant(ctx, tasksqlc.InsertPartnerRewardGrantParams{
			WorkspaceID: issue.WorkspaceID, IssueID: issue.ID, Provider: issue.Provider, GroupKey: issue.GroupKey,
			ExternalType: issue.ExternalType, AppID: issue.AppID, PlatformID: issue.PlatformID,
			PlatformUserID: issue.PlatformUserID, OperationID: operationID, RewardSnapshot: rewardPayload, ClaimedAt: now,
		})
		if err != nil {
			return err
		}
		if inserted == 0 {
			result.Status = ClaimStatusAlreadyDone
			return nil
		}
		affected, err := txRepo.q.ClaimPartnerIssue(ctx, tasksqlc.ClaimPartnerIssueParams{
			ClaimedAt: nullTime(&now), WorkspaceID: issue.WorkspaceID, ID: issue.ID,
		})
		if err != nil {
			return err
		}
		if affected == 0 {
			result.Status = ClaimStatusAlreadyDone
			return nil
		}
		issue.Status = PartnerIssueStatusClaimed
		issue.ClaimedAt = &now
		result.Issue = issue
		result.Rewards = rewards
		result.Status = ClaimStatusClaimed
		eventKey := fmt.Sprintf("partner.claimed:%d", issue.ID)
		if _, err = txRepo.recordPartnerStatsEvent(ctx, issue, PartnerStatsEventClaimed, eventKey, PartnerIssueStatusClaimed, rewardPayload, now); err != nil {
			return err
		}
		callbackPayload, err := json.Marshal(CallbackPayload{
			WorkspaceID: issue.WorkspaceID, AppID: issue.AppID, PlatformID: issue.PlatformID,
			PlatformUserID: issue.PlatformUserID, TaskID: 0, TaskKey: PartnerIssueKey(issue.ID),
			OperationID: operationID, PeriodStartAt: issue.IssuedAt, PeriodEndAt: partnerIssuePeriodEnd(issue, now),
			Rewards: rewards, Payload: issue.PublicPayload,
		})
		if err != nil {
			return err
		}
		callbackEventKey := fmt.Sprintf("tasks.partner.claimed:%d", issue.ID)
		_, err = txRepo.callbacks.CreateEvent(ctx, callbackutil.CreateParams{
			SourceService: "tasks", EventType: CallbackEventClaimed,
			EventKey: callbackEventKey, IdempotencyKey: callbackEventKey,
			Payload: callbackPayload, NextAttemptAt: now,
		})
		return err
	})
	return result, err
}

func (r *Repository) ListPartnerDailyStats(ctx context.Context, workspaceID, provider, groupKey string, from, until time.Time) ([]PartnerStatsDaily, error) {
	rows, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskPartnerStatsDaily, error) {
		return r.q.AdminListPartnerDailyStats(ctx, tasksqlc.AdminListPartnerDailyStatsParams{
			WorkspaceID: workspaceID, StatsDate: from, StatsDate_2: until,
			Column4: provider, Provider: provider, Column6: groupKey, GroupKey: groupKey,
		})
	})
	if err != nil {
		return nil, err
	}
	result := make([]PartnerStatsDaily, 0, len(rows))
	for _, row := range rows {
		result = append(result, PartnerStatsDaily{
			Date: row.StatsDate, Provider: row.Provider, GroupKey: row.GroupKey, ExternalType: row.ExternalType,
			IssuedCount: row.IssuedCount, CompletedCount: row.CompletedCount, ClaimedCount: row.ClaimedCount,
			FailedCount: row.FailedCount, FakeCount: row.FakeCount, ExpiredCount: row.ExpiredCount,
			UniqueIssuedUsers: row.UniqueIssuedUsers, UniqueCompletedUsers: row.UniqueCompletedUsers, UniqueClaimers: row.UniqueClaimers,
		})
	}
	return result, nil
}

func (r *Repository) recordPartnerStatsEvent(ctx context.Context, issue PartnerIssue, eventType, eventKey, status string, payload json.RawMessage, now time.Time) (bool, error) {
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	inserted, err := r.q.InsertPartnerStatsEvent(ctx, tasksqlc.InsertPartnerStatsEventParams{
		WorkspaceID: issue.WorkspaceID, Provider: issue.Provider, GroupKey: issue.GroupKey, ExternalType: issue.ExternalType,
		IssueID:    sql.NullInt64{Int64: int64(issue.ID), Valid: issue.ID != 0},
		ExternalID: sql.NullString{String: issue.ExternalID, Valid: issue.ExternalID != ""},
		AppID:      issue.AppID, PlatformID: issue.PlatformID, PlatformUserID: issue.PlatformUserID,
		EventType: eventType, EventKey: eventKey, Status: sql.NullString{String: status, Valid: status != ""},
		Payload: payload, OccurredAt: now,
	})
	if err != nil || inserted == 0 {
		return false, err
	}
	uniqueInserted, err := r.q.InsertPartnerStatsUniqueUser(ctx, tasksqlc.InsertPartnerStatsUniqueUserParams{
		WorkspaceID: issue.WorkspaceID, DATE: now, Provider: issue.Provider, GroupKey: issue.GroupKey,
		ExternalType: issue.ExternalType, EventType: eventType, AppID: issue.AppID, PlatformID: issue.PlatformID,
		PlatformUserID: issue.PlatformUserID,
	})
	if err != nil {
		return false, err
	}
	increment := partnerStatsIncrement(eventType, status)
	switch eventType {
	case PartnerStatsEventIssued:
		increment.UniqueIssuedUsers = uint64(uniqueInserted)
	case PartnerStatsEventCompleted:
		increment.UniqueCompletedUsers = uint64(uniqueInserted)
	case PartnerStatsEventClaimed:
		increment.UniqueClaimers = uint64(uniqueInserted)
	}
	err = r.q.IncrementPartnerStatsDaily(ctx, tasksqlc.IncrementPartnerStatsDailyParams{
		WorkspaceID: issue.WorkspaceID, DATE: now, Provider: issue.Provider, GroupKey: issue.GroupKey, ExternalType: issue.ExternalType,
		IssuedCount: increment.IssuedCount, CompletedCount: increment.CompletedCount, ClaimedCount: increment.ClaimedCount,
		FailedCount: increment.FailedCount, FakeCount: increment.FakeCount, ExpiredCount: increment.ExpiredCount,
		UniqueIssuedUsers: increment.UniqueIssuedUsers, UniqueCompletedUsers: increment.UniqueCompletedUsers, UniqueClaimers: increment.UniqueClaimers,
	})
	return true, err
}

func partnerStatsIncrement(eventType, status string) PartnerStatsDaily {
	var out PartnerStatsDaily
	switch eventType {
	case PartnerStatsEventIssued:
		out.IssuedCount = 1
	case PartnerStatsEventCompleted:
		out.CompletedCount = 1
	case PartnerStatsEventClaimed:
		out.ClaimedCount = 1
	case PartnerStatsEventFailed:
		out.FailedCount = 1
	case PartnerStatsEventFake:
		out.FakeCount = 1
	case PartnerStatsEventExpired:
		out.ExpiredCount = 1
	}
	switch status {
	case "fake", "fraud_suspected":
		out.FakeCount = 1
	case "expired", "offer_expired":
		out.ExpiredCount = 1
	}
	return out
}

func partnerIssuePeriodEnd(issue PartnerIssue, now time.Time) time.Time {
	if issue.ExpiresAt != nil {
		return *issue.ExpiresAt
	}
	return now
}

func mapPartnerConfig(row tasksqlc.TaskPartnerConfig) PartnerConfig {
	return PartnerConfig{
		WorkspaceID: row.WorkspaceID, Provider: row.Provider, GroupKey: row.GroupKey, Platform: row.Platform,
		IsEnabled: row.IsEnabled, Secret: stringPtrFromNull(row.Secret), Target: row.Target, Settings: row.Settings,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func mapPartnerIssue(row tasksqlc.TaskPartnerIssue) PartnerIssue {
	return PartnerIssue{
		ID: row.ID, WorkspaceID: row.WorkspaceID, Provider: row.Provider, GroupKey: row.GroupKey,
		Platform: row.Platform, ExternalID: row.ExternalID, ExternalType: row.ExternalType, IssueKey: row.IssueKey,
		AppID: row.AppID, PlatformID: row.PlatformID, PlatformUserID: row.PlatformUserID,
		PublicPayload: row.PublicPayload, PrivatePayload: row.PrivatePayload, Status: row.Status,
		IssuedAt: row.IssuedAt, CompletedAt: timePtrFromNull(row.CompletedAt), ClaimedAt: timePtrFromNull(row.ClaimedAt),
		ExpiresAt: timePtrFromNull(row.ExpiresAt), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func nullPartnerDurationUnit(value tasksqlc.NullTaskPartnerRewardRuleDurationUnit) *string {
	if !value.Valid {
		return nil
	}
	unit := string(value.TaskPartnerRewardRuleDurationUnit)
	return &unit
}

func stringPtrFromNull(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func timePtrFromNull(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func errorsIsNoRows(err error) bool {
	return err == sql.ErrNoRows
}

package user

import (
	"context"
	"time"

	"github.com/elum-utils/services/internal/utils/target"
	"github.com/elum-utils/services/tasks/repository"
)

const (
	PartnerStatusNotConfigured  = "not_configured"
	PartnerStatusDisabled       = "disabled"
	PartnerStatusTargetMismatch = "target_mismatch"
	PartnerStatusNoProvider     = "no_provider"
	PartnerStatusNotSupported   = "not_supported"
	PartnerStatusNotFound       = repository.ClaimStatusNotFound
	PartnerStatusNotCompleted   = "not_completed"
	PartnerStatusReady          = repository.StatusReady
	PartnerStatusStarted        = "started"
)

func (u *User) ListPartner(ctx context.Context, params PartnerListParams) ([]TaskModel, error) {
	mergedCtx, cancel := u.withContext(ctx)
	defer cancel()

	if err := params.Identity.Validate(); err != nil {
		return nil, err
	}

	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	platform := params.Platform
	if platform == "" {
		platform = params.Identity.Platform
	}
	config, found, err := u.repository.GetPartnerConfig(mergedCtx, params.Identity.WorkspaceID, params.Provider, params.GroupKey, platform)
	if err != nil || !found || !config.IsEnabled {
		return nil, err
	}
	if !target.Match(config.Target, target.Context{
		IsPremium: params.Identity.IsPremium, Sex: params.Identity.Sex, Country: params.Identity.Country,
		Locale: params.Locale, Platform: params.Identity.Platform, PlatformID: params.Identity.PlatformID,
	}) {
		return []TaskModel{}, nil
	}
	repoIdentity := repositoryIdentity(params.Identity)
	existing, err := u.repository.ListPartnerIssuesForUser(mergedCtx, repoIdentity, config.Provider, config.GroupKey, config.Platform, now)
	if err != nil {
		return nil, err
	}
	result := make([]TaskModel, 0, len(existing))
	seen := make(map[string]struct{}, len(existing))
	for _, issue := range existing {
		rewards, err := u.repository.PartnerRewards(mergedCtx, issue.WorkspaceID, issue.Provider, issue.GroupKey, issue.ExternalType)
		if err != nil {
			return nil, err
		}
		result = append(result, partnerIssueTask(issue, rewards, now))
		seen[issue.IssueKey] = struct{}{}
	}
	provider := u.partnerProvider(params.Provider)
	if provider == nil {
		return result, nil
	}
	externalTasks, err := provider.ListPartnerTasks(mergedCtx, PartnerListProviderParams{
		Identity: params.Identity, Config: config, Locale: params.Locale,
		Limit: params.Limit, Variables: params.Variables, Now: now,
	})
	if err != nil {
		return nil, err
	}
	for _, external := range externalTasks {
		issueKey := partnerIssueKey(config, params.Identity, external)
		if _, ok := seen[issueKey]; ok {
			continue
		}
		rewards, err := u.repository.PartnerRewards(mergedCtx, config.WorkspaceID, config.Provider, config.GroupKey, external.ExternalType)
		if err != nil {
			return nil, err
		}
		issue, _, err := u.repository.CreatePartnerIssue(mergedCtx, repository.CreatePartnerIssueParams{
			Identity: repoIdentity, Provider: config.Provider, GroupKey: config.GroupKey, Platform: config.Platform,
			ExternalID: external.ExternalID, ExternalType: external.ExternalType,
			IssueKey:      issueKey,
			PublicPayload: external.PublicPayload, PrivatePayload: external.PrivatePayload,
			ExpiresAt: external.ExpiresAt, StartMode: external.StartMode, Now: now,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, partnerIssueTask(issue, rewards, now))
		seen[issue.IssueKey] = struct{}{}
	}
	return result, nil
}

func (u *User) CheckPartner(ctx context.Context, params PartnerCheckParams) (PartnerCheckOutput, error) {
	mergedCtx, cancel := u.withContext(ctx)
	defer cancel()

	if err := params.Identity.Validate(); err != nil {
		return PartnerCheckOutput{}, err
	}

	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	issueID, ok := repository.ParsePartnerIssueRef(params.IssueRef)
	if !ok {
		return PartnerCheckOutput{Status: PartnerStatusNotFound}, nil
	}
	issue, found, err := u.repository.GetPartnerIssue(mergedCtx, params.Identity.WorkspaceID, issueID)
	if err != nil {
		return PartnerCheckOutput{}, err
	}
	if !found || issue.AppID != params.Identity.AppID || issue.PlatformID != params.Identity.PlatformID || issue.PlatformUserID != params.Identity.PlatformUserID {
		return PartnerCheckOutput{Status: PartnerStatusNotFound}, nil
	}
	rewards, err := u.repository.PartnerRewards(mergedCtx, issue.WorkspaceID, issue.Provider, issue.GroupKey, issue.ExternalType)
	if err != nil {
		return PartnerCheckOutput{}, err
	}
	task := partnerIssueTask(issue, rewards, now)
	if issue.Status == repository.PartnerIssueStatusCompleted || issue.Status == repository.PartnerIssueStatusClaimed {
		return PartnerCheckOutput{Status: task.Progress.Status, Completed: true, Task: &task}, nil
	}
	if issue.Status == repository.PartnerIssueStatusRevoked || issue.Status == repository.PartnerIssueStatusRevokedAfterClaim {
		return PartnerCheckOutput{Status: task.Progress.Status, Completed: false, Task: &task}, nil
	}
	if issue.StartMode == repository.StartModeRequired && issue.StartedAt == nil {
		return PartnerCheckOutput{Status: repository.ClaimStatusNotStarted, Completed: false, Task: &task}, nil
	}
	config, found, err := u.repository.GetPartnerConfig(mergedCtx, issue.WorkspaceID, issue.Provider, issue.GroupKey, issue.Platform)
	if err != nil {
		return PartnerCheckOutput{}, err
	}
	if !found || !config.IsEnabled {
		return PartnerCheckOutput{Status: PartnerStatusNotConfigured, Task: &task}, nil
	}
	provider := u.partnerProvider(issue.Provider)
	if provider == nil {
		return PartnerCheckOutput{Status: PartnerStatusNoProvider, Task: &task}, nil
	}
	check, err := provider.CheckPartnerTask(mergedCtx, PartnerCheckProviderParams{
		Identity: params.Identity, Config: config, Issue: issue, Variables: params.Variables, Now: now,
	})
	if err != nil {
		return PartnerCheckOutput{}, err
	}
	if !check.Completed {
		return PartnerCheckOutput{Status: PartnerStatusNotCompleted, Completed: false, Task: &task}, nil
	}
	issue, _, err = u.repository.CompletePartnerIssue(mergedCtx, issue.WorkspaceID, issue.ID, check.Status, check.Payload, now)
	if err != nil {
		return PartnerCheckOutput{}, err
	}
	task = partnerIssueTask(issue, rewards, now)
	return PartnerCheckOutput{Status: PartnerStatusReady, Completed: true, Task: &task}, nil
}

func (u *User) StartPartner(ctx context.Context, params PartnerStartParams) (PartnerStartOutput, error) {
	mergedCtx, cancel := u.withContext(ctx)
	defer cancel()

	if err := params.Identity.Validate(); err != nil {
		return PartnerStartOutput{}, err
	}

	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	issueID, ok := repository.ParsePartnerIssueRef(params.IssueRef)
	if !ok {
		return PartnerStartOutput{Status: PartnerStatusNotFound}, nil
	}
	issue, found, err := u.repository.GetPartnerIssue(mergedCtx, params.Identity.WorkspaceID, issueID)
	if err != nil {
		return PartnerStartOutput{}, err
	}
	if !found || issue.AppID != params.Identity.AppID || issue.PlatformID != params.Identity.PlatformID || issue.PlatformUserID != params.Identity.PlatformUserID {
		return PartnerStartOutput{Status: PartnerStatusNotFound}, nil
	}
	rewards, err := u.repository.PartnerRewards(mergedCtx, issue.WorkspaceID, issue.Provider, issue.GroupKey, issue.ExternalType)
	if err != nil {
		return PartnerStartOutput{}, err
	}
	task := partnerIssueTask(issue, rewards, now)
	config, found, err := u.repository.GetPartnerConfig(mergedCtx, issue.WorkspaceID, issue.Provider, issue.GroupKey, issue.Platform)
	if err != nil {
		return PartnerStartOutput{}, err
	}
	if !found || !config.IsEnabled {
		return PartnerStartOutput{Status: PartnerStatusNotConfigured, Task: &task}, nil
	}
	provider := u.partnerProvider(issue.Provider)
	starter, ok := provider.(PartnerStarter)
	if !ok || starter == nil {
		if issue.StartMode == repository.StartModeRequired {
			updated, changed, err := u.repository.UpdatePartnerIssueStart(
				mergedCtx, issue.WorkspaceID, issue.ID, "", nil, nil,
			)
			if err != nil {
				return PartnerStartOutput{}, err
			}
			if changed {
				issue = updated
				task = partnerIssueTask(issue, rewards, now)
			}
			return PartnerStartOutput{Status: PartnerStatusStarted, Started: true, Task: &task}, nil
		}
		return PartnerStartOutput{Status: PartnerStatusNotSupported, Task: &task}, nil
	}
	started, err := starter.StartPartnerTask(mergedCtx, PartnerStartProviderParams{
		Identity: params.Identity, Config: config, Issue: issue, Variables: params.Variables, Now: now,
	})
	if err != nil {
		return PartnerStartOutput{}, err
	}
	if !started.Started {
		return PartnerStartOutput{Status: started.Status, Started: false, Task: &task}, nil
	}
	updated, changed, err := u.repository.UpdatePartnerIssueStart(
		mergedCtx, issue.WorkspaceID, issue.ID, started.ExternalClickID,
		started.PublicPayloadPatch, started.PrivatePayloadPatch,
	)
	if err != nil {
		return PartnerStartOutput{}, err
	}
	if changed {
		issue = updated
		task = partnerIssueTask(issue, rewards, now)
	}
	return PartnerStartOutput{Status: PartnerStatusStarted, Started: true, ActionURL: started.ActionURL, Task: &task}, nil
}

func partnerIssueKey(config repository.PartnerConfig, identity Identity, external PartnerExternalTask) string {
	return config.Provider + ":" + config.GroupKey + ":" + config.Platform + ":" + external.ExternalID + ":" +
		identity.PlatformUserID + ":" + external.ExternalType
}

func partnerIssueTask(issue repository.PartnerIssue, rewards []repository.Reward, now time.Time) TaskModel {
	status := repository.StatusOpen
	progressValue := uint64(0)
	switch issue.Status {
	case repository.PartnerIssueStatusCompleted:
		status = repository.StatusReady
		progressValue = 1
	case repository.PartnerIssueStatusClaimed:
		status = repository.StatusClaimed
		progressValue = 1
	case repository.PartnerIssueStatusRevoked, repository.PartnerIssueStatusRevokedAfterClaim:
		status = issue.Status
	}
	periodEnd := now
	if issue.ExpiresAt != nil {
		periodEnd = *issue.ExpiresAt
	}
	return TaskModel{
		ID: issue.ID, Key: repository.PartnerIssueKey(issue.ID), GroupKey: issue.GroupKey,
		TaskKind: repository.TaskKindPartner, ActionKey: "partner:" + issue.Provider,
		ActionKind: repository.ActionKindExternal, ClaimMode: repository.ClaimModeManual,
		StartMode: issue.StartMode, TargetCount: 1, Payload: issue.PublicPayload, Rewards: rewards,
		Progress: &repository.ActiveProgress{
			Progress: progressValue, Status: status,
			PeriodStartAt: issue.IssuedAt, PeriodEndAt: periodEnd,
			ReadyAt: issue.CompletedAt, ClaimedAt: issue.ClaimedAt,
		},
	}
}

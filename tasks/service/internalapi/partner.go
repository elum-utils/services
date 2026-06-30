package internalapi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	json "github.com/goccy/go-json"

	"github.com/elum-utils/services/tasks/repository"
	taskruntime "github.com/elum-utils/services/tasks/runtime"
)

const PartnerCallbackStatusRevoked = "revoked"

type PartnerCallbackParams struct {
	WorkspaceID     string
	Provider        string
	GroupKey        string
	Platform        string
	IssueID         uint64
	IssueRef        string
	ExternalID      string
	ExternalClickID string
	PlatformUserID  string
	Status          string
	Payload         json.RawMessage
	Now             time.Time
}

type PartnerCallbackResult struct {
	Status string                   `json:"status"`
	Issue  *repository.PartnerIssue `json:"issue,omitempty"`
}

type PartnerWebhookParams struct {
	WorkspaceID string
	Secret      string
	Headers     map[string]string
	Query       map[string]string
	Body        json.RawMessage
	Now         time.Time
}

func (i *Internal) OnPartnerCallback(ctx context.Context, params PartnerCallbackParams) (PartnerCallbackResult, error) {
	mergedCtx, cancel := i.withContext(ctx)
	defer cancel()
	issueID := params.IssueID
	if issueID == 0 && params.IssueRef != "" {
		if parsed, ok := repository.ParsePartnerIssueRef(params.IssueRef); ok {
			issueID = parsed
		} else if parsed, err := strconv.ParseUint(strings.TrimSpace(params.IssueRef), 10, 64); err == nil {
			issueID = parsed
		}
	}
	if issueID == 0 || params.WorkspaceID == "" {
		if params.WorkspaceID == "" || params.Provider == "" {
			return PartnerCallbackResult{Status: repository.ClaimStatusNotFound}, nil
		}
		var issue repository.PartnerIssue
		var found bool
		var err error
		if params.ExternalClickID != "" {
			issue, found, err = i.repository.GetPartnerIssueByExternalClickID(mergedCtx, params.WorkspaceID, params.Provider, params.ExternalClickID)
		} else if params.ExternalID != "" && params.PlatformUserID != "" {
			issue, found, err = i.repository.GetPartnerIssueByExternalUser(
				mergedCtx, params.WorkspaceID, params.Provider, params.GroupKey, params.Platform,
				params.ExternalID, params.PlatformUserID,
			)
		} else {
			return PartnerCallbackResult{Status: repository.ClaimStatusNotFound}, nil
		}
		if err != nil {
			return PartnerCallbackResult{}, err
		}
		if !found {
			return PartnerCallbackResult{Status: repository.ClaimStatusNotFound}, nil
		}
		issueID = issue.ID
	}
	switch params.Status {
	case repository.PartnerIssueStatusCompleted, "complete", "step_completed", "subscribed":
		issue, changed, err := i.repository.CompletePartnerIssue(mergedCtx, params.WorkspaceID, issueID, params.Status, params.Payload, params.Now)
		if err != nil {
			return PartnerCallbackResult{}, err
		}
		if issue.ID == 0 {
			return PartnerCallbackResult{Status: repository.ClaimStatusNotFound}, nil
		}
		if !changed {
			return PartnerCallbackResult{Status: issue.Status, Issue: &issue}, nil
		}
		return PartnerCallbackResult{Status: issue.Status, Issue: &issue}, nil
	case PartnerCallbackStatusRevoked, repository.PartnerIssueStatusRevokedAfterClaim, "unsubscribe", "unsubscribed", "cancelled", "canceled":
		issue, changed, err := i.repository.RevokePartnerIssue(mergedCtx, params.WorkspaceID, issueID, params.Status, params.Payload, params.Now)
		if err != nil {
			return PartnerCallbackResult{}, err
		}
		if issue.ID == 0 {
			return PartnerCallbackResult{Status: repository.ClaimStatusNotFound}, nil
		}
		if !changed {
			return PartnerCallbackResult{Status: issue.Status, Issue: &issue}, nil
		}
		return PartnerCallbackResult{Status: issue.Status, Issue: &issue}, nil
	default:
		return PartnerCallbackResult{Status: "unsupported_status"}, nil
	}
}

func (i *Internal) HandlePartnerWebhook(ctx context.Context, params PartnerWebhookParams) (PartnerCallbackResult, error) {
	mergedCtx, cancel := i.withContext(ctx)
	defer cancel()
	if params.WorkspaceID == "" || params.Secret == "" {
		return PartnerCallbackResult{Status: repository.ClaimStatusNotFound}, nil
	}
	config, found, err := i.repository.GetPartnerConfigByWebhookSecret(mergedCtx, params.WorkspaceID, params.Secret)
	if err != nil {
		return PartnerCallbackResult{}, err
	}
	if !found || !config.IsEnabled {
		return PartnerCallbackResult{Status: repository.ClaimStatusNotFound}, nil
	}
	if i.runtime == nil {
		return PartnerCallbackResult{}, fmt.Errorf("tasks partner runtime is not configured")
	}
	bodyMap := map[string]any{}
	if len(params.Body) != 0 {
		if err := json.Unmarshal(params.Body, &bodyMap); err != nil {
			bodyMap = map[string]any{"raw": string(params.Body)}
		}
	}
	result, err := i.runtime.Handle(mergedCtx, config.Provider, taskruntime.Event{
		Action:   "callback",
		Provider: config.Provider,
		Config:   internalPartnerConfigMap(config),
		Request: map[string]any{
			"headers":  stringMapToAny(params.Headers),
			"query":    stringMapToAny(params.Query),
			"body":     bodyMap,
			"raw_body": string(params.Body),
		},
		Now: params.Now,
	})
	if err != nil {
		return PartnerCallbackResult{}, err
	}
	if ok, _ := result["ok"].(bool); !ok {
		return PartnerCallbackResult{Status: firstWebhookString(result["error"], "unsupported_callback")}, nil
	}
	status := firstWebhookString(result["status"], result["action"])
	if status == "complete" {
		status = repository.PartnerIssueStatusCompleted
	}
	if status == "" && webhookBool(result["completed"]) {
		status = repository.PartnerIssueStatusCompleted
	}
	payload := webhookRaw(result["payload"])
	if len(payload) == 0 {
		payload = params.Body
	}
	return i.OnPartnerCallback(mergedCtx, PartnerCallbackParams{
		WorkspaceID:     config.WorkspaceID,
		Provider:        config.Provider,
		GroupKey:        config.GroupKey,
		Platform:        config.Platform,
		IssueID:         webhookUint64(result["issue_id"]),
		IssueRef:        firstWebhookString(result["issue_ref"], result["task_ref"]),
		ExternalID:      firstWebhookString(result["external_id"], result["offer_id"], result["task_id"]),
		ExternalClickID: firstWebhookString(result["external_click_id"], result["click_id"]),
		PlatformUserID:  firstWebhookString(result["platform_user_id"], result["user_id"], result["tg_user_id"]),
		Status:          status,
		Payload:         payload,
		Now:             params.Now,
	})
}

func internalPartnerConfigMap(config repository.PartnerConfig) map[string]any {
	return map[string]any{
		"workspace_id": config.WorkspaceID, "provider": config.Provider, "group_key": config.GroupKey,
		"platform": config.Platform, "secret": stringPtrValue(config.Secret),
		"webhook_secret": stringPtrValue(config.WebhookSecret),
		"settings":       rawObject(config.Settings),
		"target":         rawObject(config.Target),
	}
}

func stringMapToAny(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func rawObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func webhookRaw(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return raw
}

func firstWebhookString(values ...any) string {
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			if typed != "" {
				return typed
			}
		case int64:
			if typed != 0 {
				return strconv.FormatInt(typed, 10)
			}
		case float64:
			if typed != 0 {
				return strconv.FormatInt(int64(typed), 10)
			}
		}
	}
	return ""
}

func webhookUint64(value any) uint64 {
	switch typed := value.(type) {
	case int64:
		if typed > 0 {
			return uint64(typed)
		}
	case float64:
		if typed > 0 {
			return uint64(typed)
		}
	case string:
		parsed, _ := strconv.ParseUint(strings.TrimSpace(typed), 10, 64)
		return parsed
	}
	return 0
}

func webhookBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "1"
	}
	return false
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

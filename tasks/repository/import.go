package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	tasksqlc "github.com/elum-utils/services/tasks/sqlc"
)

func (r *Repository) PreviewImport(ctx context.Context, workspaceID string, pkg ExportPackage) (ImportPreview, error) {
	if err := validateExportPackage(pkg); err != nil {
		return ImportPreview{}, err
	}
	preview := ImportPreview{Format: pkg.Format, Service: pkg.Service}
	preview.Counts = countPackage(pkg)
	existing, err := r.importExistingKeys(ctx, workspaceID)
	if err != nil {
		return ImportPreview{}, err
	}
	seenSequences := make(map[string]struct{})
	for _, sequence := range pkg.Sequences {
		seenSequences[sequence.Key] = struct{}{}
		if existing.sequences[sequence.Key] {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "sequence", Key: sequence.Key})
		}
	}
	for _, group := range pkg.Groups {
		if existing.groups[group.Key] {
			preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "group", Key: group.Key})
		}
		for _, config := range group.PartnerConfigs {
			for _, secret := range []*ExportSecret{config.Secret, config.WebhookSecret} {
				if secret != nil && !secretHasEmbeddedValue(secret) {
					preview.RequiredSecrets = append(preview.RequiredSecrets, *secret)
				}
			}
			key := partnerConfigImportKey(config.Provider, group.Key, config.Platform)
			if existing.partnerConfigs[key] {
				preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "partner_config", Key: key})
			}
		}
		for _, rule := range group.PartnerRewardRules {
			key := partnerRewardImportKey(rule.Provider, group.Key, rule.ExternalType, rule.Reward.Key)
			if existing.partnerRewards[key] {
				preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "partner_reward_rule", Key: key})
			}
		}
		positions := make(map[string]map[uint32]string)
		for _, task := range group.Tasks {
			if existing.tasks[task.Key] {
				preview.Conflicts = append(preview.Conflicts, ImportConflict{Type: "task", Key: task.Key})
			}
			if task.SequenceKey == nil {
				continue
			}
			if _, ok := seenSequences[*task.SequenceKey]; !ok && !existing.sequences[*task.SequenceKey] {
				preview.Warnings = append(preview.Warnings, "sequence is referenced but not present: "+*task.SequenceKey)
			}
			if task.SequencePosition != nil {
				if positions[*task.SequenceKey] == nil {
					positions[*task.SequenceKey] = make(map[uint32]string)
				}
				if prev := positions[*task.SequenceKey][*task.SequencePosition]; prev != "" {
					preview.Warnings = append(preview.Warnings, fmt.Sprintf("duplicate sequence position %s:%d used by %s and %s", *task.SequenceKey, *task.SequencePosition, prev, task.Key))
				}
				positions[*task.SequenceKey][*task.SequencePosition] = task.Key
			}
		}
	}
	return preview, nil
}

func (r *Repository) Import(ctx context.Context, workspaceID string, req ImportRequest) (ImportResult, error) {
	if err := validateExportPackage(req.Package); err != nil {
		return ImportResult{}, err
	}
	strategy := req.ConflictStrategy
	if strategy == "" {
		strategy = ImportConflictFail
	}
	if strategy != ImportConflictFail && strategy != ImportConflictSkip && strategy != ImportConflictUpdate {
		return ImportResult{}, fmt.Errorf("unsupported import conflict strategy: %s", strategy)
	}
	preview, err := r.PreviewImport(ctx, workspaceID, req.Package)
	if err != nil {
		return ImportResult{}, err
	}
	if strategy == ImportConflictFail && len(preview.Conflicts) > 0 {
		return ImportResult{}, fmt.Errorf("import conflicts found: %d", len(preview.Conflicts))
	}
	if err := requireImportSecrets(req.Package, req.Secrets); err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{}
	err = r.WithTx(ctx, func(txRepo *Repository) error {
		return txRepo.importBulk(ctx, workspaceID, req, strategy, preview, &result)
	})
	if err != nil {
		return ImportResult{}, err
	}
	return result, r.invalidateTaskCache(ctx, workspaceID)
}

func (r *Repository) importBulk(ctx context.Context, workspaceID string, req ImportRequest, strategy string, preview ImportPreview, result *ImportResult) error {
	if err := r.importSequencesBulk(ctx, workspaceID, req.Package.Sequences, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importGroupsBulk(ctx, workspaceID, req.Package.Groups, strategy, preview, result); err != nil {
		return err
	}
	taskIDs, err := r.importTasksBulk(ctx, workspaceID, req.Package.Groups, strategy, preview, result)
	if err != nil {
		return err
	}
	if err := r.importTaskLocalizationsBulk(ctx, workspaceID, req.Package.Groups, taskIDs, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importRewardsBulk(ctx, workspaceID, req.Package.Groups, taskIDs, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importComplexConditionsBulk(ctx, workspaceID, req.Package.Groups, taskIDs, strategy, preview, result); err != nil {
		return err
	}
	if err := r.importPartnerConfigsBulk(ctx, workspaceID, req.Package.Groups, strategy, req.Secrets, preview, result); err != nil {
		return err
	}
	return r.importPartnerRewardRulesBulk(ctx, workspaceID, req.Package.Groups, strategy, preview, result)
}

func (r *Repository) importSequencesBulk(ctx context.Context, workspaceID string, sequences []ExportSequence, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0, len(sequences))
	for _, sequence := range sequences {
		exists := previewHasConflict(preview, "sequence", sequence.Key)
		if exists && strategy == ImportConflictSkip {
			result.Skipped.Sequences++
			continue
		}
		rows = append(rows, []any{workspaceID, sequence.Key, sequence.Position, sequence.IsActive})
		result.Imported.Sequences++
	}
	return r.execImportBulk(ctx, "task_sequence",
		[]string{"workspace_id", "`key`", "position", "is_active"},
		rows,
		"position = VALUES(position), is_active = VALUES(is_active), deleted_at = NULL, updated_at = NOW()",
	)
}

func (r *Repository) importGroupsBulk(ctx context.Context, workspaceID string, groups []ExportGroup, strategy string, preview ImportPreview, result *ImportResult) error {
	groupRows := make([][]any, 0, len(groups))
	localizationRows := make([][]any, 0, len(groups)*2)
	for _, group := range groups {
		exists := previewHasConflict(preview, "group", group.Key)
		if exists && strategy == ImportConflictSkip {
			result.Skipped.Groups++
			continue
		}
		groupRows = append(groupRows, []any{workspaceID, group.Key, group.Position, group.IsActive})
		result.Imported.Groups++
		for locale, text := range group.Localization {
			localizationRows = append(localizationRows, []any{workspaceID, group.Key, locale, text.Title, text.Description})
			result.Imported.GroupLocalizations++
		}
	}
	if err := r.execImportBulk(ctx, "task_group",
		[]string{"workspace_id", "`key`", "position", "is_active"},
		groupRows,
		"position = VALUES(position), is_active = VALUES(is_active), deleted_at = NULL, updated_at = NOW()",
	); err != nil {
		return err
	}
	return r.execImportBulk(ctx, "task_group_localization",
		[]string{"workspace_id", "group_key", "locale", "title", "description"},
		localizationRows,
		"title = VALUES(title), description = VALUES(description), updated_at = NOW()",
	)
}

func (r *Repository) importTasksBulk(ctx context.Context, workspaceID string, groups []ExportGroup, strategy string, preview ImportPreview, result *ImportResult) (map[string]uint64, error) {
	rows := make([][]any, 0)
	needed := make(map[string]struct{})
	for _, group := range groups {
		for _, task := range group.Tasks {
			exists := previewHasConflict(preview, "task", task.Key)
			if exists && strategy == ImportConflictSkip {
				result.Skipped.Tasks++
				continue
			}
			needed[task.Key] = struct{}{}
			rows = append(rows, []any{
				workspaceID,
				task.Key,
				group.Key,
				nullString(task.SequenceKey),
				nullInt32FromUint32(task.SequencePosition),
				defaultString(task.TaskKind, TaskKindInternal),
				task.ActionKey,
				task.ActionKind,
				defaultString(task.ClaimMode, ClaimModeManual),
				defaultString(task.StartMode, StartModeNone),
				task.TargetCount,
				defaultString(task.Reset.Unit, ResetNever),
				defaultUint32(task.Reset.Every, 1),
				task.Position,
				defaultJSON(task.Payload, "{}"),
				defaultJSON(task.Target, "null"),
				nullString(task.Integration.Kind),
				nullString(task.Integration.Provider),
				defaultJSON(task.Integration.Payload, "null"),
				nullString(task.ImageURL),
				task.IsVisible,
				task.IsActive,
				nullTime(task.StartAt),
				nullTime(task.EndAt),
			})
			result.Imported.Tasks++
		}
	}
	if err := r.execImportBulk(ctx, "task_definition",
		[]string{
			"workspace_id", "`key`", "group_key", "sequence_key", "sequence_position",
			"task_kind", "action_key", "action_kind", "claim_mode", "start_mode", "target_count",
			"reset_unit", "reset_every", "position", "payload", "target",
			"integration_kind", "integration_provider", "integration_payload", "image_url",
			"is_visible", "is_active", "start_at", "end_at",
		},
		rows,
		"group_key = VALUES(group_key), sequence_key = VALUES(sequence_key), sequence_position = VALUES(sequence_position), "+
			"task_kind = VALUES(task_kind), action_key = VALUES(action_key), action_kind = VALUES(action_kind), "+
			"claim_mode = VALUES(claim_mode), start_mode = VALUES(start_mode), target_count = VALUES(target_count), reset_unit = VALUES(reset_unit), "+
			"reset_every = VALUES(reset_every), position = VALUES(position), payload = VALUES(payload), target = VALUES(target), "+
			"integration_kind = VALUES(integration_kind), integration_provider = VALUES(integration_provider), "+
			"integration_payload = VALUES(integration_payload), image_url = VALUES(image_url), is_visible = VALUES(is_visible), "+
			"is_active = VALUES(is_active), start_at = VALUES(start_at), end_at = VALUES(end_at), deleted_at = NULL, updated_at = NOW()",
	); err != nil {
		return nil, err
	}
	if len(needed) == 0 {
		return nil, nil
	}
	taskRows, err := r.q.ExportListTasks(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	resultIDs := make(map[string]uint64, len(needed))
	for _, task := range taskRows {
		if _, ok := needed[task.Key]; ok {
			resultIDs[task.Key] = task.ID
		}
	}
	return resultIDs, nil
}

func (r *Repository) importTaskLocalizationsBulk(ctx context.Context, workspaceID string, groups []ExportGroup, taskIDs map[string]uint64, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, group := range groups {
		for _, task := range group.Tasks {
			if previewHasConflict(preview, "task", task.Key) && strategy == ImportConflictSkip {
				continue
			}
			taskID, ok := taskIDs[task.Key]
			if !ok {
				continue
			}
			for locale, text := range task.Localization {
				rows = append(rows, []any{workspaceID, taskID, locale, text.Title, text.Description})
				result.Imported.TaskLocalizations++
			}
		}
	}
	return r.execImportBulk(ctx, "task_localization",
		[]string{"workspace_id", "task_id", "locale", "title", "description"},
		rows,
		"title = VALUES(title), description = VALUES(description), updated_at = NOW()",
	)
}

func (r *Repository) importRewardsBulk(ctx context.Context, workspaceID string, groups []ExportGroup, taskIDs map[string]uint64, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, group := range groups {
		for _, task := range group.Tasks {
			if previewHasConflict(preview, "task", task.Key) && strategy == ImportConflictSkip {
				continue
			}
			taskID, ok := taskIDs[task.Key]
			if !ok {
				continue
			}
			for _, reward := range task.Rewards {
				rows = append(rows, []any{
					workspaceID, taskID, reward.Key, defaultString(reward.Type, "quantity"),
					reward.Quantity, reward.Scale, nullRewardDurationUnit(reward.Unit), reward.Position,
				})
				result.Imported.Rewards++
			}
		}
	}
	return r.execImportBulk(ctx, "task_reward",
		[]string{"workspace_id", "task_id", "reward_key", "reward_type", "quantity", "scale", "duration_unit", "position"},
		rows,
		"reward_type = VALUES(reward_type), quantity = VALUES(quantity), scale = VALUES(scale), "+
			"duration_unit = VALUES(duration_unit), position = VALUES(position), updated_at = NOW()",
	)
}

func (r *Repository) importComplexConditionsBulk(ctx context.Context, workspaceID string, groups []ExportGroup, taskIDs map[string]uint64, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, group := range groups {
		for _, task := range group.Tasks {
			if previewHasConflict(preview, "task", task.Key) && strategy == ImportConflictSkip {
				continue
			}
			parentID, ok := taskIDs[task.Key]
			if !ok {
				continue
			}
			for _, condition := range task.Conditions {
				conditionID, ok := taskIDs[condition.TaskKey]
				if !ok {
					continue
				}
				rows = append(rows, []any{
					workspaceID,
					parentID,
					conditionID,
					defaultString(condition.RequiredStatus, ComplexRequiredStatusReady),
					condition.Position,
					condition.IsRequired,
				})
				result.Imported.Conditions++
			}
		}
	}
	return r.execImportBulk(ctx, "task_complex_condition",
		[]string{"workspace_id", "parent_task_id", "condition_task_id", "required_status", "position", "is_required"},
		rows,
		"required_status = VALUES(required_status), position = VALUES(position), "+
			"is_required = VALUES(is_required), updated_at = NOW()",
	)
}

func (r *Repository) importPartnerConfigsBulk(ctx context.Context, workspaceID string, groups []ExportGroup, strategy string, secrets map[string]string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, group := range groups {
		for _, config := range group.PartnerConfigs {
			key := partnerConfigImportKey(config.Provider, group.Key, config.Platform)
			exists := previewHasConflict(preview, "partner_config", key)
			if exists && strategy == ImportConflictSkip {
				result.Skipped.PartnerConfigs++
				continue
			}
			secret := importSecretValue(config.Secret, secrets)
			webhookSecret := importSecretValue(config.WebhookSecret, secrets)
			rows = append(rows, []any{
				workspaceID, config.Provider, group.Key, config.Platform, config.IsEnabled,
				secret, webhookSecret, defaultJSON(config.Target, "null"), defaultJSON(config.Settings, "null"),
			})
			result.Imported.PartnerConfigs++
		}
	}
	return r.execImportBulk(ctx, "task_partner_config",
		[]string{"workspace_id", "provider", "group_key", "platform", "is_enabled", "secret", "webhook_secret", "target", "settings"},
		rows,
		"is_enabled = VALUES(is_enabled), secret = VALUES(secret), webhook_secret = VALUES(webhook_secret), target = VALUES(target), settings = VALUES(settings), updated_at = NOW()",
	)
}

func (r *Repository) importPartnerRewardRulesBulk(ctx context.Context, workspaceID string, groups []ExportGroup, strategy string, preview ImportPreview, result *ImportResult) error {
	rows := make([][]any, 0)
	for _, group := range groups {
		for _, rule := range group.PartnerRewardRules {
			externalType := defaultString(rule.ExternalType, "*")
			key := partnerRewardImportKey(rule.Provider, group.Key, externalType, rule.Reward.Key)
			exists := previewHasConflict(preview, "partner_reward_rule", key)
			if exists && strategy == ImportConflictSkip {
				result.Skipped.PartnerRewards++
				continue
			}
			rewardType := defaultString(rule.Reward.Type, "quantity")
			rows = append(rows, []any{
				workspaceID, rule.Provider, group.Key, externalType,
				rule.Reward.Key, rewardType, rule.Reward.Quantity, rule.Reward.Scale,
				nullPartnerRewardDurationUnit(rule.Reward.Unit), rule.Position, rule.IsEnabled,
			})
			result.Imported.PartnerRewards++
		}
	}
	return r.execImportBulk(ctx, "task_partner_reward_rule",
		[]string{
			"workspace_id", "provider", "group_key", "external_type", "reward_key",
			"reward_type", "quantity", "scale", "duration_unit", "position", "is_enabled",
		},
		rows,
		"reward_type = VALUES(reward_type), quantity = VALUES(quantity), scale = VALUES(scale), "+
			"duration_unit = VALUES(duration_unit), position = VALUES(position), is_enabled = VALUES(is_enabled), updated_at = NOW()",
	)
}

func (r *Repository) execImportBulk(ctx context.Context, table string, columns []string, rows [][]any, duplicateUpdate string) error {
	if len(rows) == 0 {
		return nil
	}
	query, args := compileImportBulkUpsert(table, columns, rows, duplicateUpdate)
	return repositoryExec(ctx, r, func(ctx context.Context) error {
		_, err := r.executor.ExecContext(ctx, query, args...)
		return err
	})
}

func compileImportBulkUpsert(table string, columns []string, rows [][]any, duplicateUpdate string) (string, []any) {
	var builder strings.Builder
	builder.Grow(len(rows) * len(columns) * 4)
	builder.WriteString("INSERT INTO ")
	builder.WriteString(table)
	builder.WriteString(" (")
	builder.WriteString(strings.Join(columns, ", "))
	builder.WriteString(") VALUES ")
	args := make([]any, 0, len(rows)*len(columns))
	for rowIndex, row := range rows {
		if rowIndex > 0 {
			builder.WriteString(", ")
		}
		builder.WriteByte('(')
		for columnIndex := range columns {
			if columnIndex > 0 {
				builder.WriteString(", ")
			}
			builder.WriteByte('?')
		}
		builder.WriteByte(')')
		args = append(args, row...)
	}
	if duplicateUpdate != "" {
		builder.WriteString(" ON DUPLICATE KEY UPDATE ")
		builder.WriteString(duplicateUpdate)
	}
	return builder.String(), args
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultUint32(value uint32, fallback uint32) uint32 {
	if value == 0 {
		return fallback
	}
	return value
}

func defaultJSON(value []byte, fallback string) string {
	if len(value) == 0 {
		return fallback
	}
	return string(value)
}

func nullRewardDurationUnit(value *string) tasksqlc.NullTaskRewardDurationUnit {
	return tasksqlc.NullTaskRewardDurationUnit{
		TaskRewardDurationUnit: tasksqlc.TaskRewardDurationUnit(taskStringValue(value)),
		Valid:                  value != nil,
	}
}

func nullPartnerRewardDurationUnit(value *string) tasksqlc.NullTaskPartnerRewardRuleDurationUnit {
	return tasksqlc.NullTaskPartnerRewardRuleDurationUnit{
		TaskPartnerRewardRuleDurationUnit: tasksqlc.TaskPartnerRewardRuleDurationUnit(taskStringValue(value)),
		Valid:                             value != nil,
	}
}

type importExistingKeys struct {
	groups         map[string]bool
	sequences      map[string]bool
	tasks          map[string]bool
	partnerConfigs map[string]bool
	partnerRewards map[string]bool
}

func (r *Repository) importExistingKeys(ctx context.Context, workspaceID string) (importExistingKeys, error) {
	out := importExistingKeys{
		groups: make(map[string]bool), sequences: make(map[string]bool), tasks: make(map[string]bool),
		partnerConfigs: make(map[string]bool), partnerRewards: make(map[string]bool),
	}
	groups, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskGroup, error) {
		return r.q.AdminListGroups(ctx, workspaceID)
	})
	if err != nil {
		return out, err
	}
	for _, group := range groups {
		out.groups[group.Key] = true
	}
	sequences, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskSequence, error) {
		return r.q.AdminListSequences(ctx, workspaceID)
	})
	if err != nil {
		return out, err
	}
	for _, sequence := range sequences {
		out.sequences[sequence.Key] = true
	}
	tasks, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskDefinition, error) {
		return r.q.ExportListTasks(ctx, workspaceID)
	})
	if err != nil {
		return out, err
	}
	for _, task := range tasks {
		out.tasks[task.Key] = true
	}
	configs, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskPartnerConfig, error) {
		return r.q.AdminListPartnerConfigs(ctx, workspaceID)
	})
	if err != nil {
		return out, err
	}
	for _, config := range configs {
		out.partnerConfigs[partnerConfigImportKey(config.Provider, config.GroupKey, config.Platform)] = true
	}
	rewards, err := repositoryValue(ctx, r, func(ctx context.Context) ([]tasksqlc.TaskPartnerRewardRule, error) {
		return r.q.AdminListPartnerRewardRules(ctx, workspaceID)
	})
	if err != nil {
		return out, err
	}
	for _, reward := range rewards {
		out.partnerRewards[partnerRewardImportKey(reward.Provider, reward.GroupKey, reward.ExternalType, reward.RewardKey)] = true
	}
	return out, nil
}

func countPackage(pkg ExportPackage) ImportCounts {
	out := ImportCounts{Groups: len(pkg.Groups), Sequences: len(pkg.Sequences)}
	for _, group := range pkg.Groups {
		out.GroupLocalizations += len(group.Localization)
		out.Tasks += len(group.Tasks)
		out.PartnerConfigs += len(group.PartnerConfigs)
		out.PartnerRewards += len(group.PartnerRewardRules)
		for _, task := range group.Tasks {
			out.TaskLocalizations += len(task.Localization)
			out.Rewards += len(task.Rewards)
			out.Conditions += len(task.Conditions)
		}
	}
	return out
}

func validateExportPackage(pkg ExportPackage) error {
	if pkg.Format != ExportFormat {
		return fmt.Errorf("unsupported tasks export format: %s", pkg.Format)
	}
	if pkg.Service != "" && pkg.Service != "tasks" {
		return fmt.Errorf("unsupported export service: %s", pkg.Service)
	}
	for _, group := range pkg.Groups {
		if group.Key == "" {
			return fmt.Errorf("group key is required")
		}
		for _, task := range group.Tasks {
			if task.Key == "" {
				return fmt.Errorf("task key is required")
			}
			if task.SequenceKey == nil && task.SequencePosition != nil {
				return fmt.Errorf("task %s has sequence_position without sequence_key", task.Key)
			}
			if task.SequenceKey != nil && task.SequencePosition == nil {
				return fmt.Errorf("task %s has sequence_key without sequence_position", task.Key)
			}
			for _, condition := range task.Conditions {
				if condition.TaskKey == "" {
					return fmt.Errorf("task %s condition task_key is required", task.Key)
				}
				if condition.TaskKey == task.Key {
					return fmt.Errorf("task %s cannot reference itself as condition", task.Key)
				}
			}
		}
	}
	return nil
}

func requireImportSecrets(pkg ExportPackage, secrets map[string]string) error {
	for _, group := range pkg.Groups {
		for _, config := range group.PartnerConfigs {
			for _, secret := range []*ExportSecret{config.Secret, config.WebhookSecret} {
				if secret == nil {
					continue
				}
				if importSecretValue(secret, secrets).Valid {
					continue
				}
				if secret.Key == "" {
					return fmt.Errorf("required import secret is missing")
				}
				if secrets == nil || secrets[secret.Key] == "" {
					return fmt.Errorf("required import secret is missing: %s", secret.Key)
				}
			}
		}
	}
	return nil
}

func importSecretValue(secret *ExportSecret, secrets map[string]string) sql.NullString {
	if secret == nil {
		return sql.NullString{}
	}
	if secrets != nil && secret.Key != "" {
		if value := secrets[secret.Key]; value != "" {
			return sql.NullString{String: value, Valid: true}
		}
	}
	if secret.Value != nil && *secret.Value != "" {
		return sql.NullString{String: *secret.Value, Valid: true}
	}
	return sql.NullString{}
}

func secretHasEmbeddedValue(secret *ExportSecret) bool {
	return secret != nil && secret.Value != nil && *secret.Value != ""
}

func previewHasConflict(preview ImportPreview, kind, key string) bool {
	for _, conflict := range preview.Conflicts {
		if conflict.Type == kind && conflict.Key == key {
			return true
		}
	}
	return false
}

func partnerConfigImportKey(provider, groupKey, platform string) string {
	return provider + ":" + groupKey + ":" + platform
}

func partnerRewardImportKey(provider, groupKey, externalType, rewardKey string) string {
	if externalType == "" {
		externalType = "*"
	}
	return provider + ":" + groupKey + ":" + externalType + ":" + rewardKey
}

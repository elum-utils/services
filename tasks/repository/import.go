package repository

import (
	"context"
	"fmt"

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
			if config.Secret != nil {
				preview.RequiredSecrets = append(preview.RequiredSecrets, *config.Secret)
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
		for _, sequence := range req.Package.Sequences {
			exists := previewHasConflict(preview, "sequence", sequence.Key)
			if exists && strategy == ImportConflictSkip {
				result.Skipped.Sequences++
				continue
			}
			if err := txRepo.UpsertSequence(ctx, workspaceID, sequence.Key, sequence.Position, sequence.IsActive); err != nil {
				return err
			}
			result.Imported.Sequences++
		}
		for _, group := range req.Package.Groups {
			groupExists := previewHasConflict(preview, "group", group.Key)
			if groupExists && strategy == ImportConflictSkip {
				result.Skipped.Groups++
			} else {
				if err := txRepo.UpsertGroup(ctx, workspaceID, group.Key, group.Position, group.IsActive); err != nil {
					return err
				}
				result.Imported.Groups++
				for locale, text := range group.Localization {
					if err := txRepo.UpsertGroupLocalization(ctx, workspaceID, group.Key, locale, text.Title, text.Description); err != nil {
						return err
					}
					result.Imported.GroupLocalizations++
				}
			}
			if err := txRepo.importTasks(ctx, workspaceID, group, strategy, preview, &result); err != nil {
				return err
			}
			if err := txRepo.importPartnerSettings(ctx, workspaceID, group, strategy, req.Secrets, preview, &result); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ImportResult{}, err
	}
	return result, r.invalidateTaskCache(ctx, workspaceID)
}

func (r *Repository) importTasks(ctx context.Context, workspaceID string, group ExportGroup, strategy string, preview ImportPreview, result *ImportResult) error {
	for _, task := range group.Tasks {
		exists := previewHasConflict(preview, "task", task.Key)
		if exists && strategy == ImportConflictSkip {
			result.Skipped.Tasks++
			continue
		}
		var id uint64
		if exists {
			row, err := r.q.AdminGetTaskByKey(ctx, tasksqlc.AdminGetTaskByKeyParams{WorkspaceID: workspaceID, Key: task.Key})
			if err != nil {
				return err
			}
			id = row.ID
		}
		savedID, err := r.SaveTask(ctx, SaveTaskParams{
			ID: id, WorkspaceID: workspaceID, Key: task.Key, GroupKey: group.Key,
			SequenceKey: task.SequenceKey, SequencePosition: task.SequencePosition,
			TaskKind: task.TaskKind, ActionKey: task.ActionKey, ActionKind: task.ActionKind,
			ClaimMode: task.ClaimMode, TargetCount: task.TargetCount,
			ResetUnit: task.Reset.Unit, ResetEvery: task.Reset.Every, Position: task.Position,
			Payload: task.Payload, Target: task.Target,
			IntegrationKind: task.Integration.Kind, IntegrationProvider: task.Integration.Provider,
			IntegrationPayload: task.Integration.Payload,
			ImageURL:           task.ImageURL, IsVisible: task.IsVisible, IsActive: task.IsActive,
			StartAt: task.StartAt, EndAt: task.EndAt,
		})
		if err != nil {
			return err
		}
		result.Imported.Tasks++
		for locale, text := range task.Localization {
			if err := r.UpsertTaskLocalization(ctx, workspaceID, savedID, locale, text.Title, text.Description); err != nil {
				return err
			}
			result.Imported.TaskLocalizations++
		}
		for _, reward := range task.Rewards {
			if err := r.UpsertReward(ctx, workspaceID, savedID, Reward{
				Key: reward.Key, Type: reward.Type, Quantity: reward.Quantity, Scale: reward.Scale, Unit: reward.Unit,
			}, reward.Position); err != nil {
				return err
			}
			result.Imported.Rewards++
		}
	}
	return nil
}

func (r *Repository) importPartnerSettings(ctx context.Context, workspaceID string, group ExportGroup, strategy string, secrets map[string]string, preview ImportPreview, result *ImportResult) error {
	for _, config := range group.PartnerConfigs {
		key := partnerConfigImportKey(config.Provider, group.Key, config.Platform)
		exists := previewHasConflict(preview, "partner_config", key)
		if exists && strategy == ImportConflictSkip {
			result.Skipped.PartnerConfigs++
			continue
		}
		var secret *string
		if config.Secret != nil {
			value := secrets[config.Secret.Key]
			secret = &value
		}
		if err := r.SavePartnerConfig(ctx, SavePartnerConfigParams{
			WorkspaceID: workspaceID, Provider: config.Provider, GroupKey: group.Key, Platform: config.Platform,
			IsEnabled: config.IsEnabled, Secret: secret, Target: config.Target, Settings: config.Settings,
		}); err != nil {
			return err
		}
		result.Imported.PartnerConfigs++
	}
	for _, rule := range group.PartnerRewardRules {
		key := partnerRewardImportKey(rule.Provider, group.Key, rule.ExternalType, rule.Reward.Key)
		exists := previewHasConflict(preview, "partner_reward_rule", key)
		if exists && strategy == ImportConflictSkip {
			result.Skipped.PartnerRewards++
			continue
		}
		if err := r.SavePartnerRewardRule(ctx, SavePartnerRewardRuleParams{
			WorkspaceID: workspaceID, Provider: rule.Provider, GroupKey: group.Key, ExternalType: rule.ExternalType,
			Reward: Reward{
				Key: rule.Reward.Key, Type: rule.Reward.Type, Quantity: rule.Reward.Quantity,
				Scale: rule.Reward.Scale, Unit: rule.Reward.Unit,
			},
			Position: rule.Position, IsEnabled: rule.IsEnabled,
		}); err != nil {
			return err
		}
		result.Imported.PartnerRewards++
	}
	return nil
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
		}
	}
	return nil
}

func requireImportSecrets(pkg ExportPackage, secrets map[string]string) error {
	for _, group := range pkg.Groups {
		for _, config := range group.PartnerConfigs {
			if config.Secret == nil {
				continue
			}
			if secrets == nil || secrets[config.Secret.Key] == "" {
				return fmt.Errorf("required import secret is missing: %s", config.Secret.Key)
			}
		}
	}
	return nil
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

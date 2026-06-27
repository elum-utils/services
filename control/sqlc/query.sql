-- name: CreateAccount :exec
INSERT INTO control_account (id, display_name) VALUES (?, ?);

-- name: GetAccount :one
SELECT id, display_name, status, created_at, updated_at
FROM control_account WHERE id = ? LIMIT 1;

-- name: FindAccountByIdentity :one
SELECT a.id, a.display_name, a.status, a.created_at, a.updated_at
FROM control_identity i
JOIN control_account a ON a.id = i.account_id
WHERE i.provider = ? AND i.provider_subject = ?
LIMIT 1;

-- name: UpsertIdentity :exec
INSERT INTO control_identity (account_id, provider, provider_subject, payload)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE provider_subject = VALUES(provider_subject), payload = VALUES(payload);

-- name: ListIdentities :many
SELECT account_id, provider, provider_subject, COALESCE(payload, JSON_OBJECT()) AS payload, created_at, updated_at
FROM control_identity WHERE account_id = ? ORDER BY provider;

-- name: CountIdentities :one
SELECT COUNT(*) AS count FROM control_identity WHERE account_id = ?;

-- name: DeleteIdentity :execrows
DELETE FROM control_identity WHERE account_id = ? AND provider = ?;

-- name: CreateSession :exec
INSERT INTO control_session (id, account_id, token_hash, ip, user_agent, bind_to_ip, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetActiveSessionByHash :one
SELECT id, account_id, token_hash, ip, user_agent, bind_to_ip, expires_at, revoked_at, last_used_at, created_at
FROM control_session
WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > NOW()
LIMIT 1;

-- name: RevokeSession :execrows
UPDATE control_session SET revoked_at = NOW()
WHERE id = ? AND account_id = ? AND revoked_at IS NULL;

-- name: RevokeAllSessions :execrows
UPDATE control_session SET revoked_at = NOW()
WHERE account_id = ? AND revoked_at IS NULL AND (? = '' OR id <> ?);

-- name: TouchSession :execrows
UPDATE control_session SET last_used_at = NOW()
WHERE id = ? AND revoked_at IS NULL;

-- name: ListSessions :many
SELECT id, account_id, token_hash, ip, user_agent, bind_to_ip, expires_at, revoked_at, last_used_at, created_at
FROM control_session WHERE account_id = ? ORDER BY created_at DESC;

-- name: CreateWorkspace :exec
INSERT INTO control_workspace (id, slug, title, created_by) VALUES (?, ?, ?, ?);

-- name: GetWorkspace :one
SELECT id, slug, title, status, created_by, created_at, updated_at
FROM control_workspace WHERE id = ? LIMIT 1;

-- name: ListWorkspacesForAccount :many
SELECT w.id, w.slug, w.title, w.status, w.created_by, w.created_at, w.updated_at
FROM control_workspace w
JOIN control_workspace_member m ON m.workspace_id = w.id
WHERE m.account_id = ? AND m.status = 'active'
ORDER BY w.created_at DESC LIMIT ? OFFSET ?;

-- name: AddWorkspaceMember :exec
INSERT INTO control_workspace_member (workspace_id, account_id)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE status = 'active';

-- name: ListWorkspaceMembers :many
SELECT m.workspace_id, m.account_id, m.status, m.joined_at, m.updated_at,
       a.display_name, COALESCE(MIN(r.position), 2147483647) AS position
FROM control_workspace_member m
JOIN control_account a ON a.id = m.account_id
LEFT JOIN control_role_member rm ON rm.account_id = m.account_id
LEFT JOIN control_role r ON r.id = rm.role_id AND r.workspace_id = m.workspace_id AND r.deleted_at IS NULL
WHERE m.workspace_id = ? AND m.status = 'active'
GROUP BY m.workspace_id, m.account_id, m.status, m.joined_at, m.updated_at, a.display_name
ORDER BY position, m.joined_at
LIMIT ? OFFSET ?;

-- name: RemoveWorkspaceMember :execrows
UPDATE control_workspace_member SET status = 'removed'
WHERE workspace_id = ? AND account_id = ? AND status = 'active';

-- name: RemoveWorkspaceMemberRoles :execrows
DELETE rm
FROM control_role_member rm
JOIN control_role r ON r.id = rm.role_id
WHERE r.workspace_id = ? AND rm.account_id = ?;

-- name: UpdateWorkspace :execrows
UPDATE control_workspace SET slug = ?, title = ?, status = ? WHERE id = ?;

-- name: UpdateWorkspaceAsActiveMember :execrows
UPDATE control_workspace w
SET w.slug = ?, w.title = ?, w.status = ?
WHERE w.id = ?
  AND EXISTS (
      SELECT 1 FROM control_workspace_member m
      WHERE m.workspace_id = w.id AND m.account_id = ? AND m.status = 'active'
  );

-- name: CreateInvite :exec
INSERT INTO control_workspace_invite (id, workspace_id, created_by, token_hash, max_uses, expires_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: AddInviteRole :exec
INSERT INTO control_workspace_invite_role (invite_id, role_id) VALUES (?, ?);

-- name: GetActiveInviteByHashForUpdate :one
SELECT id, workspace_id, created_by, token_hash, max_uses, used_count, expires_at, revoked_at, created_at
FROM control_workspace_invite
WHERE token_hash = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())
  AND (max_uses IS NULL OR used_count < max_uses)
LIMIT 1 FOR UPDATE;

-- name: ListInviteRoles :many
SELECT role_id FROM control_workspace_invite_role WHERE invite_id = ? ORDER BY role_id;

-- name: IncrementInviteUse :execrows
UPDATE control_workspace_invite SET used_count = used_count + 1
WHERE id = ? AND revoked_at IS NULL AND (max_uses IS NULL OR used_count < max_uses);

-- name: ListInvites :many
SELECT id, workspace_id, created_by, token_hash, max_uses, used_count, expires_at, revoked_at, created_at
FROM control_workspace_invite WHERE workspace_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: RevokeInvite :execrows
UPDATE control_workspace_invite SET revoked_at = NOW()
WHERE id = ? AND workspace_id = ? AND revoked_at IS NULL;

-- name: RevokeInviteAsActiveMember :execrows
UPDATE control_workspace_invite i
SET i.revoked_at = NOW()
WHERE i.id = ? AND i.workspace_id = ? AND i.revoked_at IS NULL
  AND EXISTS (
      SELECT 1 FROM control_workspace_member m
      WHERE m.workspace_id = i.workspace_id AND m.account_id = ? AND m.status = 'active'
  );

-- name: CreateRole :exec
INSERT INTO control_role (id, workspace_id, code, title, description, position, is_owner)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: AddRoleMember :exec
INSERT IGNORE INTO control_role_member (role_id, account_id) VALUES (?, ?);

-- name: RemoveRoleMember :execrows
DELETE FROM control_role_member WHERE role_id = ? AND account_id = ?;

-- name: UpdateRole :execrows
UPDATE control_role SET title = ?, description = ?, position = ?
WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL AND is_owner = FALSE;

-- name: DeleteRole :execrows
UPDATE control_role SET deleted_at = NOW()
WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL AND is_owner = FALSE;

-- name: ListRoles :many
SELECT r.id, r.workspace_id, r.code, r.title, r.description, r.position, r.is_owner, r.deleted_at, r.created_at, r.updated_at,
       COUNT(rm.account_id) AS member_count
FROM control_role r
LEFT JOIN control_role_member rm ON rm.role_id = r.id
WHERE r.workspace_id = ? AND r.deleted_at IS NULL
GROUP BY r.id
ORDER BY r.position, r.id;

-- name: ListRolePermissions :many
SELECT p.role_id, p.method_key, p.created_at
FROM control_role_permission p
JOIN control_role r ON r.id = p.role_id
WHERE r.workspace_id = ? AND p.role_id = ? AND r.deleted_at IS NULL
ORDER BY p.method_key;

-- name: UpsertMethod :exec
INSERT INTO control_method (method_key, service, group_key, position)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE service = VALUES(service), group_key = VALUES(group_key), position = VALUES(position);

-- name: UpsertMethodGroup :exec
INSERT INTO control_method_group (service, group_key, position)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE position = VALUES(position);

-- name: ListMethodGroups :many
SELECT service, group_key, position, created_at, updated_at
FROM control_method_group
ORDER BY service, group_key;

-- name: ListAccessCatalog :many
SELECT g.service, service_catalog.position AS service_position, g.group_key, g.position AS group_position,
       COALESCE(service_title.value, g.service) AS service_title,
       COALESCE(service_description.value, '') AS service_description,
       COALESCE(group_loc.value, g.group_key) AS group_title,
       COALESCE(group_description.value, '') AS group_description,
       m.method_key, m.position,
       COALESCE(access_loc.value, m.method_key) AS access_title,
       COALESCE(access_description.value, '') AS access_description
FROM control_method_group g
JOIN control_method m ON m.service = g.service AND m.group_key = g.group_key
JOIN control_access_service service_catalog ON service_catalog.service = g.service
LEFT JOIN control_localization service_title
    ON service_title.localization_key = CONCAT('control.access_service.', g.service, '.title')
   AND service_title.locale = ?
LEFT JOIN control_localization service_description
    ON service_description.localization_key = CONCAT('control.access_service.', g.service, '.description')
   AND service_description.locale = ?
LEFT JOIN control_localization group_loc
    ON group_loc.localization_key = CONCAT('control.method_group.', g.service, '.', g.group_key)
   AND group_loc.locale = ?
LEFT JOIN control_localization group_description
    ON group_description.localization_key = CONCAT('control.method_group.', g.service, '.', g.group_key, '.description')
   AND group_description.locale = ?
LEFT JOIN control_localization access_loc
    ON access_loc.localization_key = CONCAT('control.method.', m.method_key)
   AND access_loc.locale = ?
LEFT JOIN control_localization access_description
    ON access_description.localization_key = CONCAT('control.method.', m.method_key, '.description')
   AND access_description.locale = ?
ORDER BY service_catalog.position, g.position, m.position, m.method_key;

-- name: GetMethod :one
SELECT method_key, service, group_key, position, created_at, updated_at
FROM control_method WHERE method_key = ? LIMIT 1;

-- name: ListMethods :many
SELECT method_key, service, group_key, position, created_at, updated_at
FROM control_method ORDER BY service, group_key, method_key;

-- name: SetRolePermission :exec
INSERT IGNORE INTO control_role_permission (role_id, method_key) VALUES (?, ?);

-- name: DeleteRolePermission :execrows
DELETE FROM control_role_permission WHERE role_id = ? AND method_key = ?;

-- name: ClearRolePermissions :execrows
DELETE FROM control_role_permission WHERE role_id = ?;

-- name: CheckAccess :one
SELECT EXISTS(
    SELECT 1
    FROM control_workspace_member m
    JOIN control_method cm ON cm.method_key = ?
    JOIN control_role_member rm ON rm.account_id = m.account_id
    JOIN control_role r ON r.id = rm.role_id
    LEFT JOIN control_role_permission p ON p.role_id = r.id AND p.method_key = cm.method_key
    WHERE m.workspace_id = ? AND m.account_id = ? AND m.status = 'active'
      AND r.workspace_id = m.workspace_id AND r.deleted_at IS NULL
      AND (r.is_owner = TRUE OR p.method_key IS NOT NULL)
) AS allowed;

-- name: ListAuthorizedMethods :many
SELECT DISTINCT cm.method_key, cm.service, cm.group_key, cm.position
FROM control_method cm
JOIN control_workspace_member m ON m.workspace_id = ? AND m.account_id = ? AND m.status = 'active'
JOIN control_role_member rm ON rm.account_id = m.account_id
JOIN control_role r ON r.id = rm.role_id AND r.workspace_id = m.workspace_id AND r.deleted_at IS NULL
LEFT JOIN control_role_permission p ON p.role_id = r.id AND p.method_key = cm.method_key
WHERE r.is_owner = TRUE OR p.method_key IS NOT NULL
ORDER BY cm.service, cm.group_key, cm.method_key;

-- name: GetAccountPosition :one
SELECT COALESCE((
    SELECT r.position
    FROM control_workspace_member m
    JOIN control_role_member rm ON rm.account_id = m.account_id
    JOIN control_role r ON r.id = rm.role_id
    WHERE m.workspace_id = ? AND m.account_id = ? AND m.status = 'active'
      AND r.workspace_id = m.workspace_id AND r.deleted_at IS NULL
    ORDER BY r.position
    LIMIT 1
), 2147483647) AS position;

-- name: IsActiveWorkspaceMember :one
SELECT EXISTS(
    SELECT 1 FROM control_workspace_member
    WHERE workspace_id = ? AND account_id = ? AND status = 'active'
) AS active;

-- name: GetRole :one
SELECT id, workspace_id, code, title, description, position, is_owner, deleted_at, created_at, updated_at
FROM control_role WHERE id = ? AND workspace_id = ? AND deleted_at IS NULL LIMIT 1;

-- name: CreateAuditEvent :exec
INSERT INTO control_audit_event (
    id, workspace_id, actor_id, method_key, target_type, target_id,
    before_data, after_data, result, request_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpsertTwoFactor :exec
INSERT INTO control_two_factor (account_id, secret, backup_hashes, activated_at)
VALUES (?, ?, ?, NULL)
ON DUPLICATE KEY UPDATE secret = VALUES(secret), backup_hashes = VALUES(backup_hashes), activated_at = NULL;

-- name: GetTwoFactor :one
SELECT account_id, secret, backup_hashes, activated_at, created_at, updated_at
FROM control_two_factor WHERE account_id = ? LIMIT 1;

-- name: GetTwoFactorForUpdate :one
SELECT account_id, secret, backup_hashes, activated_at, created_at, updated_at
FROM control_two_factor WHERE account_id = ? LIMIT 1 FOR UPDATE;

-- name: ActivateTwoFactor :execrows
UPDATE control_two_factor SET activated_at = NOW()
WHERE account_id = ? AND activated_at IS NULL;

-- name: DeleteTwoFactor :execrows
DELETE FROM control_two_factor WHERE account_id = ?;

-- name: HasActiveTwoFactor :one
SELECT EXISTS(
    SELECT 1 FROM control_two_factor WHERE account_id = ? AND activated_at IS NOT NULL
) AS active;

-- name: CreateTwoFactorChallenge :exec
INSERT INTO control_two_factor_challenge (id, account_id, token_hash, ip, user_agent, bind_to_ip, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetTwoFactorChallengeForUpdate :one
SELECT id, account_id, token_hash, ip, user_agent, bind_to_ip, expires_at, created_at
FROM control_two_factor_challenge
WHERE token_hash = ? AND expires_at > NOW()
LIMIT 1 FOR UPDATE;

-- name: GetTwoFactorChallengeWithFactorForUpdate :one
SELECT c.id AS challenge_id, c.account_id, c.ip, c.user_agent, c.bind_to_ip, c.expires_at,
       f.secret, f.backup_hashes, f.activated_at
FROM control_two_factor_challenge c
JOIN control_two_factor f ON f.account_id = c.account_id
WHERE c.token_hash = ? AND c.expires_at > NOW()
LIMIT 1 FOR UPDATE;

-- name: DeleteTwoFactorChallenge :execrows
DELETE FROM control_two_factor_challenge WHERE id = ?;

-- name: UpdateTwoFactorBackupHashes :execrows
UPDATE control_two_factor SET backup_hashes = ? WHERE account_id = ? AND activated_at IS NOT NULL;

-- name: UpdatePendingTwoFactorBackupHashes :execrows
UPDATE control_two_factor SET backup_hashes = ? WHERE account_id = ? AND activated_at IS NULL;

-- name: ListAuditEvents :many
SELECT id, workspace_id, actor_id, method_key, target_type, target_id,
       COALESCE(before_data, JSON_OBJECT()) AS before_data, COALESCE(after_data, JSON_OBJECT()) AS after_data, result, request_id, occurred_at
FROM control_audit_event
WHERE workspace_id = ?
ORDER BY occurred_at DESC, id DESC LIMIT ? OFFSET ?;

-- name: ListAuditEventsFiltered :many
SELECT id, workspace_id, actor_id, method_key, target_type, target_id,
       COALESCE(before_data, JSON_OBJECT()) AS before_data, COALESCE(after_data, JSON_OBJECT()) AS after_data, result, request_id, occurred_at
FROM control_audit_event
WHERE workspace_id = ?
  AND (? = '' OR method_key = ?)
  AND (? = '' OR actor_id = ?)
  AND (? IS NULL OR occurred_at >= ?)
  AND (? IS NULL OR occurred_at < ?)
ORDER BY occurred_at DESC, id DESC LIMIT ? OFFSET ?;

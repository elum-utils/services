package admin

import (
	"context"
	"strings"

	"github.com/elum-utils/services/control/repository"
	"github.com/google/uuid"
)

func (a *Admin) CreateAccount(ctx context.Context, id, displayName string) (AccountModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	account, err := a.repository.CreateAccount(mergedCtx, strings.TrimSpace(id), strings.TrimSpace(displayName))
	return mapAccount(account), err
}

func (a *Admin) CreateWorkspace(ctx context.Context, params CreateWorkspaceParams) (WorkspaceModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	if strings.TrimSpace(params.ID) == "" {
		params.ID = uuid.NewString()
	}
	workspace, err := a.repository.CreateWorkspace(mergedCtx, params.ID, strings.ToLower(strings.TrimSpace(params.Slug)), strings.TrimSpace(params.Title), strings.TrimSpace(params.ActorID))
	return mapWorkspace(workspace), err
}

func (a *Admin) GetWorkspace(ctx context.Context, workspaceID string) (WorkspaceModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	workspace, err := a.repository.GetWorkspace(mergedCtx, strings.TrimSpace(workspaceID))
	return mapWorkspace(workspace), err
}

func (a *Admin) UpdateWorkspace(ctx context.Context, params UpdateWorkspaceParams) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.UpdateWorkspace(mergedCtx, strings.TrimSpace(params.ActorID), strings.TrimSpace(params.WorkspaceID), strings.ToLower(strings.TrimSpace(params.Slug)), strings.TrimSpace(params.Title), strings.TrimSpace(params.Status))
}

func (a *Admin) ListWorkspaces(ctx context.Context, accountID string, page Page) ([]WorkspaceModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	limit, offset := normalizePage(page)
	items, err := a.repository.ListWorkspaces(mergedCtx, strings.TrimSpace(accountID), limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]WorkspaceModel, 0, len(items))
	for _, item := range items {
		result = append(result, mapWorkspace(item))
	}
	return result, nil
}

func (a *Admin) CreateRole(ctx context.Context, params CreateRoleParams) (RoleModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	if strings.TrimSpace(params.ID) == "" {
		params.ID = uuid.NewString()
	}
	role, err := a.repository.CreateRole(mergedCtx, strings.TrimSpace(params.ActorID), repository.Role{
		ID: params.ID, WorkspaceID: strings.TrimSpace(params.WorkspaceID), Code: strings.ToLower(strings.TrimSpace(params.Code)),
		Title: strings.TrimSpace(params.Title), Description: strings.TrimSpace(params.Description), Position: params.Position,
	})
	return mapRole(role), err
}

func (a *Admin) ListRoles(ctx context.Context, workspaceID string) ([]RoleModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	items, err := a.repository.ListRoles(mergedCtx, strings.TrimSpace(workspaceID))
	if err != nil {
		return nil, err
	}
	result := make([]RoleModel, 0, len(items))
	for _, item := range items {
		result = append(result, mapRole(item))
	}
	return result, nil
}

func (a *Admin) UpdateRole(ctx context.Context, params UpdateRoleParams) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.UpdateRole(mergedCtx, strings.TrimSpace(params.ActorID), repository.Role{ID: strings.TrimSpace(params.ID), WorkspaceID: strings.TrimSpace(params.WorkspaceID), Title: strings.TrimSpace(params.Title), Description: strings.TrimSpace(params.Description), Position: params.Position})
}

func (a *Admin) DeleteRole(ctx context.Context, actorID, workspaceID, roleID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.DeleteRole(mergedCtx, strings.TrimSpace(actorID), strings.TrimSpace(workspaceID), strings.TrimSpace(roleID))
}

func (a *Admin) SetRoleMember(ctx context.Context, params SetRoleMemberParams) error {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.AssignRole(mergedCtx, strings.TrimSpace(params.ActorID), strings.TrimSpace(params.WorkspaceID), strings.TrimSpace(params.AccountID), strings.TrimSpace(params.RoleID))
}

func (a *Admin) RemoveRoleMember(ctx context.Context, params SetRoleMemberParams) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.RemoveRole(mergedCtx, strings.TrimSpace(params.ActorID), strings.TrimSpace(params.WorkspaceID), strings.TrimSpace(params.AccountID), strings.TrimSpace(params.RoleID))
}

func (a *Admin) ListMembers(ctx context.Context, workspaceID string, page Page) ([]MemberModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	limit, offset := normalizePage(page)
	items, err := a.repository.ListMembers(mergedCtx, strings.TrimSpace(workspaceID), limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]MemberModel, 0, len(items))
	for _, item := range items {
		result = append(result, MemberModel{WorkspaceID: item.WorkspaceID, AccountID: item.AccountID, DisplayName: item.DisplayName, Position: item.Position, JoinedAt: item.JoinedAt, UpdatedAt: item.UpdatedAt})
	}
	return result, nil
}

func (a *Admin) RemoveMember(ctx context.Context, actorID, workspaceID, accountID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.RemoveMember(mergedCtx, strings.TrimSpace(actorID), strings.TrimSpace(workspaceID), strings.TrimSpace(accountID))
}

func (a *Admin) CreateInvite(ctx context.Context, params CreateInviteParams) (InviteModel, string, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	item, token, err := a.repository.CreateInvite(mergedCtx, strings.TrimSpace(params.ActorID), strings.TrimSpace(params.WorkspaceID), params.RoleIDs, params.ExpiresAt, params.MaxUses)
	return mapInvite(item), token, err
}

func (a *Admin) AcceptInvite(ctx context.Context, accountID, token string) (InviteModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	item, err := a.repository.AcceptInvite(mergedCtx, strings.TrimSpace(accountID), strings.TrimSpace(token))
	return mapInvite(item), err
}

func (a *Admin) ListInvites(ctx context.Context, workspaceID string, page Page) ([]InviteModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	limit, offset := normalizePage(page)
	items, err := a.repository.ListInvites(mergedCtx, strings.TrimSpace(workspaceID), limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]InviteModel, 0, len(items))
	for _, item := range items {
		result = append(result, mapInvite(item))
	}
	return result, nil
}

func (a *Admin) RevokeInvite(ctx context.Context, actorID, workspaceID, inviteID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.RevokeInvite(mergedCtx, strings.TrimSpace(actorID), strings.TrimSpace(workspaceID), strings.TrimSpace(inviteID))
}

func (a *Admin) SetRolePermission(ctx context.Context, params SetRolePermissionParams) error {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.SetPermission(mergedCtx, strings.TrimSpace(params.ActorID), strings.TrimSpace(params.WorkspaceID), strings.TrimSpace(params.RoleID), strings.TrimSpace(params.MethodKey), params.Enabled)
}

func (a *Admin) ListRolePermissions(ctx context.Context, workspaceID, roleID string) ([]string, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.ListPermissions(mergedCtx, strings.TrimSpace(workspaceID), strings.TrimSpace(roleID))
}

func (a *Admin) ClearRolePermissions(ctx context.Context, actorID, workspaceID, roleID string) (int64, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.ClearPermissions(mergedCtx, strings.TrimSpace(actorID), strings.TrimSpace(workspaceID), strings.TrimSpace(roleID))
}

func (a *Admin) RegisterMethod(ctx context.Context, params RegisterMethodParams) error {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	return a.repository.RegisterMethod(mergedCtx, repository.Method{
		Key: strings.TrimSpace(params.Key), Service: strings.TrimSpace(params.Service), GroupKey: strings.TrimSpace(params.GroupKey),
		Title: strings.TrimSpace(params.Title), WorkspaceScoped: params.WorkspaceScoped, Sensitive: params.Sensitive, SchemaRevision: params.SchemaRevision,
	})
}

func (a *Admin) ListMethods(ctx context.Context) ([]MethodModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	items, err := a.repository.ListMethods(mergedCtx)
	if err != nil {
		return nil, err
	}
	result := make([]MethodModel, 0, len(items))
	for _, item := range items {
		result = append(result, mapMethod(item))
	}
	return result, nil
}

func (a *Admin) GetMethod(ctx context.Context, methodKey string) (MethodModel, error) {
	mergedCtx, cancel := a.withContext(ctx)
	defer cancel()
	value, err := a.repository.GetMethod(mergedCtx, strings.TrimSpace(methodKey))
	return mapMethod(value), err
}

func normalizePage(page Page) (int32, int32) {
	if page.Limit <= 0 {
		page.Limit = 100
	}
	if page.Limit > 1000 {
		page.Limit = 1000
	}
	if page.Offset < 0 {
		page.Offset = 0
	}
	return page.Limit, page.Offset
}

func mapAccount(value repository.Account) AccountModel {
	return AccountModel{ID: value.ID, DisplayName: value.DisplayName, Status: value.Status, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func mapSession(value repository.Session) SessionModel {
	return SessionModel{ID: value.ID, AccountID: value.AccountID, IP: value.IP, UserAgent: value.UserAgent, BindToIP: value.BindToIP, ExpiresAt: value.ExpiresAt, RevokedAt: value.RevokedAt, LastUsedAt: value.LastUsedAt, CreatedAt: value.CreatedAt}
}

func mapWorkspace(value repository.Workspace) WorkspaceModel {
	return WorkspaceModel{ID: value.ID, Slug: value.Slug, Title: value.Title, Status: value.Status, CreatedBy: value.CreatedBy, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func mapRole(value repository.Role) RoleModel {
	return RoleModel{ID: value.ID, WorkspaceID: value.WorkspaceID, Code: value.Code, Title: value.Title, Description: value.Description, Position: value.Position, IsOwner: value.IsOwner, MemberCount: value.MemberCount, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func mapInvite(value repository.Invite) InviteModel {
	return InviteModel{ID: value.ID, WorkspaceID: value.WorkspaceID, CreatedBy: value.CreatedBy, MaxUses: value.MaxUses, UsedCount: value.UsedCount, ExpiresAt: value.ExpiresAt, RevokedAt: value.RevokedAt, CreatedAt: value.CreatedAt, RoleIDs: append([]string(nil), value.RoleIDs...)}
}

func mapMethod(value repository.Method) MethodModel {
	return MethodModel{Key: value.Key, Service: value.Service, GroupKey: value.GroupKey, Title: value.Title, WorkspaceScoped: value.WorkspaceScoped, Sensitive: value.Sensitive, SchemaRevision: value.SchemaRevision, Status: value.Status, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

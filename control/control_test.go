package control_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/elum-utils/services/control"
	"github.com/elum-utils/services/control/repository"
	"github.com/elum-utils/services/control/service/admin"
	"github.com/elum-utils/services/control/service/internalapi"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"testing"
	"time"
)

const (
	controlTestHost     = "localhost"
	controlTestPort     = 5432
	controlTestUser     = "postgres"
	controlTestPassword = "RBTX0DXKbagvCy2XCAi4qHt0cjeSD6bU"
	controlTestDatabase = "control_test"
)

func TestControlWorkspaceAccessAndInvite(t *testing.T) {
	service := newControlTestService(t)
	ctx := context.Background()
	for _, accountID := range []string{"owner", "moderator", "member", "invitee"} {
		if _, err := service.Admin.CreateAccount(ctx, accountID, accountID); err != nil {
			t.Fatalf("create account %s: %v", accountID, err)
		}
	}
	workspace, err := service.Admin.CreateWorkspace(ctx, admin.CreateWorkspaceParams{ID: "workspace", ActorID: "owner", Slug: "workspace", Title: "Workspace"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if allowed, err := service.Internal.CheckAccess(ctx, internalapi.AccessRequest{AccountID: "owner", WorkspaceID: workspace.ID, MethodKey: "unknown.method"}); err != nil || allowed {
		t.Fatalf("unregistered method must be denied: allowed=%v err=%v", allowed, err)
	}
	moderator, err := service.Admin.CreateRole(ctx, admin.CreateRoleParams{ActorID: "owner", ID: "moderator-role", WorkspaceID: workspace.ID, Code: "moderator", Title: "Moderator", Position: 10})
	if err != nil {
		t.Fatalf("create moderator role: %v", err)
	}
	member, err := service.Admin.CreateRole(ctx, admin.CreateRoleParams{ActorID: "owner", ID: "member-role", WorkspaceID: workspace.ID, Code: "member", Title: "Member", Position: 20})
	if err != nil {
		t.Fatalf("create member role: %v", err)
	}
	invite, token, err := service.Admin.CreateInvite(ctx, admin.CreateInviteParams{ActorID: "owner", WorkspaceID: workspace.ID, RoleIDs: []string{member.ID}})
	if err != nil || invite.ID == "" || token == "" {
		t.Fatalf("create invite: invite=%#v token=%q err=%v", invite, token, err)
	}
	if _, err := service.Admin.AcceptInvite(ctx, "member", token); err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	if err := service.Admin.SetRoleMember(ctx, admin.SetRoleMemberParams{ActorID: "owner", WorkspaceID: workspace.ID, AccountID: "invitee", RoleID: member.ID}); !errors.Is(err, repository.ErrForbidden) {
		t.Fatalf("non-member must not receive role: %v", err)
	}
	if _, err := service.Admin.AcceptInvite(ctx, "moderator", token); err != nil {
		t.Fatalf("accept moderator invite: %v", err)
	}
	if err := service.Admin.SetRoleMember(ctx, admin.SetRoleMemberParams{ActorID: "owner", WorkspaceID: workspace.ID, AccountID: "moderator", RoleID: moderator.ID}); err != nil {
		t.Fatalf("assign moderator role: %v", err)
	}
	if allowed, err := service.Internal.CheckAccess(ctx, internalapi.AccessRequest{AccountID: "moderator", WorkspaceID: workspace.ID, MethodKey: "control.role_member.set"}); err != nil || allowed {
		t.Fatalf("permission must be denied before grant: allowed=%v err=%v", allowed, err)
	}
	if err := service.Admin.SetRolePermission(ctx, admin.SetRolePermissionParams{ActorID: "owner", WorkspaceID: workspace.ID, RoleID: moderator.ID, MethodKey: "control.role_member.set", Enabled: true}); err != nil {
		t.Fatalf("set permission: %v", err)
	}
	allowed, err := service.Internal.CheckAccess(ctx, internalapi.AccessRequest{AccountID: "moderator", WorkspaceID: workspace.ID, MethodKey: "control.role_member.set"})
	if err != nil || !allowed {
		t.Fatalf("moderator access: allowed=%v err=%v", allowed, err)
	}
	authorizedMethods, err := service.Internal.GetAuthorizedMethods(ctx, "moderator", workspace.ID)
	if err != nil || len(authorizedMethods) != 1 || authorizedMethods[0].Key != "control.role_member.set" {
		t.Fatalf("authorized methods: methods=%#v err=%v", authorizedMethods, err)
	}
	if err := service.Admin.SetRolePermission(ctx, admin.SetRolePermissionParams{ActorID: "owner", WorkspaceID: workspace.ID, RoleID: moderator.ID, MethodKey: "control.role_member.set", Enabled: false}); err != nil {
		t.Fatalf("remove permission: %v", err)
	}
	authorizedMethods, err = service.Internal.GetAuthorizedMethods(ctx, "moderator", workspace.ID)
	if err != nil || len(authorizedMethods) != 0 {
		t.Fatalf("authorization cache invalidation: methods=%#v err=%v", authorizedMethods, err)
	}
	if err := service.Admin.SetRolePermission(ctx, admin.SetRolePermissionParams{ActorID: "owner", WorkspaceID: workspace.ID, RoleID: moderator.ID, MethodKey: "control.role_member.set", Enabled: true}); err != nil {
		t.Fatalf("restore permission: %v", err)
	}
	if err := service.Admin.SetRoleMember(ctx, admin.SetRoleMemberParams{ActorID: "moderator", WorkspaceID: workspace.ID, AccountID: "member", RoleID: moderator.ID}); !errors.Is(err, repository.ErrRoleHierarchy) {
		t.Fatalf("moderator must not grant equal role: %v", err)
	}
	if err := service.Admin.SetRoleMember(ctx, admin.SetRoleMemberParams{ActorID: "moderator", WorkspaceID: workspace.ID, AccountID: "member", RoleID: member.ID}); err != nil {
		t.Fatalf("moderator may grant lower role: %v", err)
	}
}

func TestControlAdminMutationWritesAuditInTransaction(t *testing.T) {
	service := newControlTestService(t)
	ctx := context.Background()

	if _, err := service.Admin.CreateAccount(ctx, "audit-owner", "Audit owner"); err != nil {
		t.Fatalf("create account: %v", err)
	}

	workspace, err := service.Admin.CreateWorkspace(ctx, admin.CreateWorkspaceParams{
		ID:      "audit-workspace",
		ActorID: "audit-owner",
		Slug:    "audit-workspace",
		Title:   "Audit workspace",
	})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	if _, err := service.Admin.UpdateWorkspace(ctx, admin.UpdateWorkspaceParams{
		ActorID:     "audit-owner",
		WorkspaceID: workspace.ID,
		Slug:        "audit-workspace",
		Title:       "Updated audit workspace",
		Status:      "active",
	}); err != nil {
		t.Fatalf("update workspace: %v", err)
	}

	events, err := service.Admin.ListAudit(ctx, workspace.ID, admin.Page{Limit: 20})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}

	keys := make(map[string]bool, len(events))
	for _, event := range events {
		keys[event.MethodKey] = true
	}
	if !keys["control.workspace.create"] || !keys["control.workspace.update"] {
		t.Fatalf("automatic audit events = %#v", events)
	}
}

func TestControlRegisterManifestIsAtomic(t *testing.T) {
	service := newControlTestService(t)
	ctx := context.Background()

	if err := service.Internal.RegisterManifest(ctx, []internalapi.MethodManifest{
		{
			Key:      "atomic.owner",
			Service:  "owner-a",
			GroupKey: "test",
		},
	}); err != nil {
		t.Fatalf("register owner method: %v", err)
	}

	err := service.Internal.RegisterManifest(ctx, []internalapi.MethodManifest{
		{
			Key:      "atomic.must_rollback",
			Service:  "owner-b",
			GroupKey: "test",
		},
		{
			Key:      "atomic.owner",
			Service:  "owner-b",
			GroupKey: "test",
		},
	})
	if !errors.Is(err, repository.ErrMethodOwner) {
		t.Fatalf("register conflicting manifest error = %v, want ErrMethodOwner", err)
	}
	if _, err := service.Admin.GetMethod(ctx, "atomic.must_rollback"); !errors.Is(err, repository.ErrMethodNotFound) {
		t.Fatalf("partial manifest row survived rollback: %v", err)
	}
}

func TestControlRegisterManifestSerializesConflictingOwners(t *testing.T) {
	service := newControlTestService(t)
	ctx := context.Background()
	start := make(chan struct{})
	type result struct {
		service string
		err     error
	}
	results := make(chan result, 2)

	for _, serviceName := range []string{"concurrent-owner-a", "concurrent-owner-b"} {
		serviceName := serviceName
		go func() {
			<-start
			err := service.Internal.RegisterManifest(ctx, []internalapi.MethodManifest{
				{
					Key:      "concurrent.owner",
					Service:  serviceName,
					GroupKey: "test",
				},
			})
			results <- result{service: serviceName, err: err}
		}()
	}

	close(start)
	first := <-results
	second := <-results
	winners := make([]string, 0, 1)
	losers := 0
	for _, value := range []result{first, second} {
		switch {
		case value.err == nil:
			winners = append(winners, value.service)
		case errors.Is(value.err, repository.ErrMethodOwner):
			losers++
		default:
			t.Fatalf("unexpected concurrent manifest error for %s: %v", value.service, value.err)
		}
	}
	if len(winners) != 1 || losers != 1 {
		t.Fatalf("concurrent manifest results: winners=%v losers=%d", winners, losers)
	}

	method, err := service.Admin.GetMethod(ctx, "concurrent.owner")
	if err != nil {
		t.Fatalf("get concurrent manifest method: %v", err)
	}
	if method.Service != winners[0] {
		t.Fatalf("stored method owner = %q, want %q", method.Service, winners[0])
	}
}

func newControlTestService(t testing.TB) *control.Control {
	t.Helper()
	adminDB, err := sql.Open("pgx", controlPostgresDSN("postgres"))
	if err != nil {
		t.Fatalf("open postgres admin: %v", err)
	}
	defer adminDB.Close()
	if _, err := adminDB.Exec(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()",
		controlTestDatabase,
	); err != nil {
		t.Fatalf("terminate test database connections: %v", err)
	}
	if _, err := adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", controlTestDatabase)); err != nil {
		t.Fatalf("drop test database: %v", err)
	}
	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", controlTestDatabase)); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	db, err := sql.Open("pgx", controlPostgresDSN(controlTestDatabase))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	db.SetConnMaxLifetime(time.Minute)
	client, err := sqlwrap.New(db)
	if err != nil {
		t.Fatalf("new sql client: %v", err)
	}
	repo := repository.New(client)
	if err := repo.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	service, err := control.NewWithDatabase(context.Background(), db, control.Options{})
	if err != nil {
		t.Fatalf("new control: %v", err)
	}
	t.Cleanup(func() {
		_ = service.Close()
		_ = repo.Close()
		_ = client.Close()
	})
	return service
}

func controlPostgresDSN(database string) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		controlTestUser,
		controlTestPassword,
		controlTestHost,
		controlTestPort,
		database,
	)
}

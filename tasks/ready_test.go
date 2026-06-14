package tasks

import (
	"context"
	"testing"

	"github.com/elum-utils/services/tasks/service/admin"
	"github.com/elum-utils/services/tasks/service/internalapi"
	"github.com/elum-utils/services/tasks/service/user"
)

func TestIsReady(t *testing.T) {
	var nilService *Tasks
	if nilService.IsReady() {
		t.Fatal("nil tasks must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized tasks must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx = ctx
	service.Admin, service.Internal, service.User = &admin.Admin{}, &internalapi.Internal{}, &user.User{}
	if !service.IsReady() {
		t.Fatal("initialized tasks must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed tasks must not be ready")
	}
}

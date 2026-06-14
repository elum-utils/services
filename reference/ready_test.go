package reference

import (
	"context"
	"testing"

	"github.com/elum-utils/services/reference/service/admin"
	"github.com/elum-utils/services/reference/service/user"
)

func TestIsReady(t *testing.T) {
	var nilService *Reference
	if nilService.IsReady() {
		t.Fatal("nil reference must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized reference must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx, service.Admin, service.User = ctx, &admin.Admin{}, &user.User{}
	if !service.IsReady() {
		t.Fatal("initialized reference must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed reference must not be ready")
	}
}

package calendar

import (
	"context"
	"testing"

	"github.com/elum-utils/services/calendar/service/admin"
	"github.com/elum-utils/services/calendar/service/user"
)

func TestIsReady(t *testing.T) {
	var nilService *Calendar
	if nilService.IsReady() {
		t.Fatal("nil calendar must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized calendar must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx, service.Admin, service.User = ctx, &admin.Admin{}, &user.User{}
	if !service.IsReady() {
		t.Fatal("initialized calendar must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed calendar must not be ready")
	}
}

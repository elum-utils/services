package promo

import (
	"context"
	"testing"

	"github.com/elum-utils/services/promo/service/admin"
	"github.com/elum-utils/services/promo/service/user"
)

func TestIsReady(t *testing.T) {
	var nilService *Promo
	if nilService.IsReady() {
		t.Fatal("nil promo must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized promo must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx, service.Admin, service.User = ctx, &admin.Admin{}, &user.User{}
	if !service.IsReady() {
		t.Fatal("initialized promo must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed promo must not be ready")
	}
}

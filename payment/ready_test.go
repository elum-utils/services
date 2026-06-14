package payment

import (
	"context"
	"testing"

	"github.com/elum-utils/services/payment/service/admin"
	"github.com/elum-utils/services/payment/service/user"
)

func TestIsReady(t *testing.T) {
	var nilService *Payment
	if nilService.IsReady() {
		t.Fatal("nil payment must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized payment must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx, service.Admin, service.User = ctx, &admin.Admin{}, &user.User{}
	service.Adapters = &Adapters{}
	if !service.IsReady() {
		t.Fatal("initialized payment must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed payment must not be ready")
	}
}

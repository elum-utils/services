package cpa

import (
	"context"
	"testing"

	"github.com/elum-utils/services/cpa/service/admin"
	"github.com/elum-utils/services/cpa/service/user"
)

func TestIsReady(t *testing.T) {
	var nilService *CPA
	if nilService.IsReady() {
		t.Fatal("nil cpa must not be ready")
	}
	service := New(DatabaseParams{})
	if service.IsReady() {
		t.Fatal("uninitialized cpa must not be ready")
	}
	ctx, cancel := context.WithCancel(context.Background())
	service.rootCtx, service.Admin, service.User = ctx, &admin.Admin{}, &user.User{}
	if !service.IsReady() {
		t.Fatal("initialized cpa must be ready")
	}
	cancel()
	if service.IsReady() {
		t.Fatal("closed cpa must not be ready")
	}
}

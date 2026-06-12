package yookassa

import (
	"context"
	"strings"
	"testing"
)

func TestHandleWebhookRejectsInvalidSignatureBeforePayloadParsing(t *testing.T) {
	adapter := &YooKassa{}

	_, err := adapter.HandleWebhook(context.Background(), []byte(`not-json`), false)
	if err == nil {
		t.Fatal("expected invalid signature to fail")
	}
	if !strings.Contains(err.Error(), "invalid webhook signature") {
		t.Fatalf("unexpected error: %v", err)
	}
}

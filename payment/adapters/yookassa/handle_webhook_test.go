package yookassa

import (
	"context"
	"errors"
	"testing"
)

func TestHandleWebhookRejectsInvalidSignatureBeforePayloadParsing(t *testing.T) {
	adapter := &YooKassa{}

	_, err := adapter.HandleWebhook(context.Background(), []byte(`not-json`), false)
	if err == nil {
		t.Fatal("expected invalid signature to fail")
	}
	if !errors.Is(err, ErrWebhookSignatureInvalid) {
		t.Fatalf("unexpected error: %v", err)
	}
}

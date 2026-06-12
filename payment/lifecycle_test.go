package payment

import (
	"context"
	"testing"
	"time"
)

func TestMergeContextsCancelsOnLifecycleDone(t *testing.T) {
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	methodCtx := context.Background()

	ctx, cancel := mergeContexts(lifecycleCtx, methodCtx)
	defer cancel()

	lifecycleCancel()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("expected merged context to be canceled by lifecycle context")
	}
}

func TestMergeContextsCancelsOnMethodDone(t *testing.T) {
	lifecycleCtx := context.Background()
	methodCtx, methodCancel := context.WithCancel(context.Background())

	ctx, cancel := mergeContexts(lifecycleCtx, methodCtx)
	defer cancel()

	methodCancel()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("expected merged context to be canceled by method context")
	}
}

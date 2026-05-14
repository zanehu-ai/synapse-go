package outbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreEnqueueAndPending(t *testing.T) {
	store := NewMemoryStore()
	rec := &OutboxRecord{
		ID:          "out-1",
		TenantID:    "t-1",
		AggregateID: "agg-1",
		EventType:   "tenant.created",
		CreatedAt:   time.Now().UTC(),
	}
	if err := store.Enqueue(context.Background(), rec); err != nil {
		t.Fatalf("Enqueue returned error: %v", err)
	}
	pending := store.Pending(context.Background())
	if len(pending) != 1 || pending[0].ID != "out-1" {
		t.Fatalf("pending = %+v, want out-1", pending)
	}
}

func TestMemoryStoreRejectsDuplicate(t *testing.T) {
	store := NewMemoryStore()
	rec := &OutboxRecord{ID: "out-1", TenantID: "t-1", EventType: "tenant.created"}
	if err := store.Enqueue(context.Background(), rec); err != nil {
		t.Fatalf("Enqueue returned error: %v", err)
	}
	if err := store.Enqueue(context.Background(), rec); !errors.Is(err, ErrDuplicateRecord) {
		t.Fatalf("duplicate error = %v, want ErrDuplicateRecord", err)
	}
}

func TestMemoryStoreMarkDispatched(t *testing.T) {
	store := NewMemoryStore()
	rec := &OutboxRecord{ID: "out-1", TenantID: "t-1", EventType: "tenant.created"}
	if err := store.Enqueue(context.Background(), rec); err != nil {
		t.Fatalf("Enqueue returned error: %v", err)
	}
	if err := store.MarkDispatched(context.Background(), "out-1"); err != nil {
		t.Fatalf("MarkDispatched returned error: %v", err)
	}
	if got := store.Pending(context.Background()); len(got) != 0 {
		t.Fatalf("pending len = %d, want 0", len(got))
	}
}

func TestDispatcherWorkerLoop(t *testing.T) {
	t.Skip("Phase 1 W7-8 implements dispatcher worker loop with retry")
}

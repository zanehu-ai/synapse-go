package outbox

import (
	"context"
	"sync"

	"gorm.io/gorm"
)

// Inserter writes a single outbox row using the caller's transaction handle.
//
// Implementations MUST insert atomically with the caller's other DML — i.e.,
// if the caller's tx commits, the outbox row commits; if it rolls back, the
// outbox row is gone. This is the platform-side fix for funding_* services
// that currently build outbox events post-commit (best-effort), violating
// CLAUDE.md hard constraint #7 (audit + idempotency + reliable_events three-set).
//
// Usage pattern (transactional outbox):
//
//	evt, err := eventsSvc.BuildOutbox(input)
//	db.Transaction(func(tx *gorm.DB) error {
//	    // ... business DML ...
//	    return inserter.InsertTx(ctx, tx, evt)
//	})
//
// frozen reference
type Inserter interface {
	InsertTx(ctx context.Context, tx *gorm.DB, evt any) error
}

// RepoInserter is the production implementation of Inserter. It calls
// tx.WithContext(ctx).Create(evt) so the outbox row commits or rolls back
// together with whatever transaction the caller controls.
//
// The evt argument must be a pointer to a GORM-mapped struct (e.g.
// *reliable_events.Outbox). RepoInserter imposes no opinion on which
// struct is used — the caller's model determines the target table and
// column mappings.
type RepoInserter struct{}

// NewRepoInserter returns a zero-allocation RepoInserter ready to use.
func NewRepoInserter() *RepoInserter { return &RepoInserter{} }

// InsertTx creates evt inside the supplied GORM transaction.
// It propagates the context so deadline / cancellation is respected.
// Returns the underlying GORM error unchanged so callers can inspect it
// (e.g. duplicate-key detection).
func (r *RepoInserter) InsertTx(ctx context.Context, tx *gorm.DB, evt any) error {
	return tx.WithContext(ctx).Create(evt).Error
}

// MockInserter is a test helper that records every call to InsertTx.
// It is safe for concurrent use; inject a non-nil Err to simulate failures.
//
// Example:
//
//	m := outbox.NewMockInserter()
//	// ...wire m into the service under test...
//	m.Err = errors.New("forced failure")   // optional failure injection
//	assert.Len(t, m.Events(), 1)
type MockInserter struct {
	Err    error // if non-nil, InsertTx returns this instead of recording
	mu     sync.Mutex
	events []any
}

// NewMockInserter returns a ready-to-use MockInserter with no pre-set error.
func NewMockInserter() *MockInserter { return &MockInserter{} }

// InsertTx records evt in the in-memory slice (or returns Err if set).
// The tx and ctx arguments are accepted but not used — the mock is
// intentionally stateless with respect to DB transactions.
func (m *MockInserter) InsertTx(_ context.Context, _ *gorm.DB, evt any) error {
	if m.Err != nil {
		return m.Err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, evt)
	return nil
}

// Events returns a snapshot of all events recorded so far (oldest first).
func (m *MockInserter) Events() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]any, len(m.events))
	copy(out, m.events)
	return out
}

// Reset clears all recorded events and unsets the injected error.
func (m *MockInserter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = nil
	m.Err = nil
}

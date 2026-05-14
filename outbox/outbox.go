package outbox

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidRecord   = errors.New("outbox: invalid record")
	ErrDuplicateRecord = errors.New("outbox: duplicate record")
	ErrRecordNotFound  = errors.New("outbox: record not found")
)

// ErrNotImplemented is kept for compatibility with old skeleton callers.
var ErrNotImplemented = ErrInvalidRecord

// OutboxRecord 是 reliable_events 表的一条记录。
type OutboxRecord struct {
	ID           string
	TenantID     string
	AggregateID  string
	EventType    string
	Payload      []byte // JSON-encoded event envelope
	CreatedAt    time.Time
	DispatchedAt *time.Time
	RetryCount   int
}

var defaultStore = NewMemoryStore()

type MemoryStore struct {
	mu      sync.Mutex
	records map[string]*OutboxRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{records: map[string]*OutboxRecord{}}
}

// Enqueue records an outbox event in the process-local store. Application
// services with a DB should use their transactional repository instead; this
// helper is for tests and lightweight in-process producers.
func Enqueue(ctx context.Context, rec *OutboxRecord) error {
	return defaultStore.Enqueue(ctx, rec)
}

func (s *MemoryStore) Enqueue(_ context.Context, rec *OutboxRecord) error {
	if rec == nil || rec.ID == "" || rec.TenantID == "" || rec.EventType == "" {
		return ErrInvalidRecord
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.records[rec.ID]; exists {
		return ErrDuplicateRecord
	}
	cp := *rec
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	s.records[cp.ID] = &cp
	return nil
}

func (s *MemoryStore) Pending(_ context.Context) []*OutboxRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*OutboxRecord, 0, len(s.records))
	for _, rec := range s.records {
		if rec.DispatchedAt == nil {
			cp := *rec
			out = append(out, &cp)
		}
	}
	return out
}

func (s *MemoryStore) MarkDispatched(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[id]
	if !ok {
		return ErrRecordNotFound
	}
	now := time.Now()
	rec.DispatchedAt = &now
	return nil
}

// Dispatch marks all pending in-process records as dispatched.
func Dispatch(ctx context.Context) error {
	for _, rec := range defaultStore.Pending(ctx) {
		if err := defaultStore.MarkDispatched(ctx, rec.ID); err != nil {
			return err
		}
	}
	return nil
}

package reliableevents

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeEventsRepo struct {
	inserted        *Outbox
	pending         []*Outbox
	list            []*Outbox
	sentID          uint64
	failedID        uint64
	processed       map[string]bool
	markedProcessed *EventProcessed
	insertErr       error
}

func (f *fakeEventsRepo) InsertOutbox(_ context.Context, e *Outbox) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.inserted = e
	e.ID = 1
	return nil
}

func (f *fakeEventsRepo) FetchPending(_ context.Context, limit int) ([]*Outbox, error) {
	if limit > len(f.pending) {
		limit = len(f.pending)
	}
	return f.pending[:limit], nil
}

func (f *fakeEventsRepo) List(_ context.Context, limit, offset int) ([]*Outbox, error) {
	if offset >= len(f.list) {
		return []*Outbox{}, nil
	}
	end := offset + limit
	if end > len(f.list) {
		end = len(f.list)
	}
	return f.list[offset:end], nil
}

func (f *fakeEventsRepo) MarkSent(_ context.Context, id uint64) error {
	f.sentID = id
	return nil
}

func (f *fakeEventsRepo) MarkFailed(_ context.Context, id uint64) error {
	f.failedID = id
	return nil
}

func (f *fakeEventsRepo) MarkProcessed(_ context.Context, ep *EventProcessed) error {
	f.markedProcessed = ep
	if f.processed == nil {
		f.processed = map[string]bool{}
	}
	f.processed[ep.ConsumerName+":"+ep.EventID] = true
	return nil
}

func (f *fakeEventsRepo) IsProcessed(_ context.Context, consumerName, eventID string) (bool, error) {
	return f.processed[consumerName+":"+eventID], nil
}

func TestServicePublishValidatesAndEnqueues(t *testing.T) {
	repo := &fakeEventsRepo{}
	svc := NewService(repo)
	svc.now = func() time.Time { return time.Unix(100, 0) }

	evt, err := svc.Publish(context.Background(), PublishInput{
		TenantID:      9,
		EventType:     "funding.payment.created",
		Source:        "test",
		CorrelationID: "corr-1",
		Actor:         map[string]any{"id": 1},
		Payload:       map[string]any{"amount": "10.00"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.inserted != evt || evt.ID != 1 {
		t.Fatalf("inserted = %+v evt=%+v", repo.inserted, evt)
	}
	if evt.EventID == "" || len(evt.EventID) != 32 {
		t.Fatalf("event_id = %q, want generated 32 chars", evt.EventID)
	}
	if evt.CorrelationID == nil || *evt.CorrelationID != "corr-1" {
		t.Fatalf("correlation_id = %v", evt.CorrelationID)
	}
	var payload map[string]any
	if err := json.Unmarshal(evt.Payload, &payload); err != nil || payload["amount"] != "10.00" {
		t.Fatalf("payload = %s err=%v", string(evt.Payload), err)
	}
}

func TestServiceRejectsInvalidPublish(t *testing.T) {
	svc := NewService(&fakeEventsRepo{})
	if _, err := svc.Publish(context.Background(), PublishInput{TenantID: 0, EventType: "x", Source: "s"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

func TestMarkProcessedIdempotent(t *testing.T) {
	repo := &fakeEventsRepo{processed: map[string]bool{}}
	svc := NewService(repo)
	event := &Outbox{EventID: "evt-1", TenantID: 2}
	if err := svc.MarkProcessed(context.Background(), "consumer-a", event); err != nil {
		t.Fatal(err)
	}
	if repo.markedProcessed == nil {
		t.Fatal("expected processed row")
	}
	repo.markedProcessed = nil
	if err := svc.MarkProcessed(context.Background(), "consumer-a", event); err != nil {
		t.Fatal(err)
	}
	if repo.markedProcessed != nil {
		t.Fatal("second MarkProcessed should be idempotent")
	}
}

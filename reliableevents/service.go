package reliableevents

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidInput   = errors.New("reliableevents: invalid input")
	ErrDuplicateEvent = errors.New("reliableevents: duplicate event_id")
)

// PublishInput is the input for publishing a reliable event to the outbox.
type PublishInput struct {
	EventID       string
	TenantID      uint64
	EventType     string
	Source        string
	CorrelationID string
	Actor         map[string]any
	Payload       map[string]any
}

type repository interface {
	InsertOutbox(ctx context.Context, e *Outbox) error
	FetchPending(ctx context.Context, limit int) ([]*Outbox, error)
	List(ctx context.Context, limit, offset int) ([]*Outbox, error)
	MarkSent(ctx context.Context, id uint64) error
	MarkFailed(ctx context.Context, id uint64) error
	MarkProcessed(ctx context.Context, ep *EventProcessed) error
	IsProcessed(ctx context.Context, consumerName, eventID string) (bool, error)
}

// Service holds reliable events logic.
type Service struct {
	repo repository
	now  func() time.Time
}

// NewService creates a Service.
func NewService(repo repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Publish writes an event to the outbox table using the service's repository.
func (s *Service) Publish(ctx context.Context, in PublishInput) (*Outbox, error) {
	e, err := s.BuildOutbox(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.InsertOutbox(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// BuildOutbox validates and constructs an Outbox row without persisting it.
func (s *Service) BuildOutbox(in PublishInput) (*Outbox, error) {
	in.EventID = strings.TrimSpace(in.EventID)
	in.EventType = strings.TrimSpace(in.EventType)
	in.Source = strings.TrimSpace(in.Source)
	if in.TenantID == 0 || in.EventType == "" || in.Source == "" || len(in.EventID) > 32 {
		return nil, ErrInvalidInput
	}
	actor, _ := json.Marshal(in.Actor)
	payload, _ := json.Marshal(in.Payload)
	eventID := in.EventID
	if eventID == "" {
		eventID = newEventID()
	}
	now := time.Now()
	if s != nil && s.now != nil {
		now = s.now()
	}
	e := &Outbox{
		EventID:    eventID,
		TenantID:   in.TenantID,
		EventType:  in.EventType,
		Version:    1,
		Actor:      actor,
		Source:     in.Source,
		Payload:    payload,
		Status:     0,
		OccurredAt: now,
	}
	if correlationID := strings.TrimSpace(in.CorrelationID); correlationID != "" {
		e.CorrelationID = &correlationID
	}
	return e, nil
}

// IsProcessed checks if the consumer+eventID pair was already processed.
func (s *Service) IsProcessed(ctx context.Context, consumerName, eventID string) (bool, error) {
	if strings.TrimSpace(consumerName) == "" || strings.TrimSpace(eventID) == "" {
		return false, ErrInvalidInput
	}
	return s.repo.IsProcessed(ctx, consumerName, eventID)
}

func (s *Service) Pending(ctx context.Context, limit int) ([]*Outbox, error) {
	return s.repo.FetchPending(ctx, normalizeLimit(limit))
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]*Outbox, error) {
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, normalizeLimit(limit), offset)
}

func (s *Service) MarkSent(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidInput
	}
	return s.repo.MarkSent(ctx, id)
}

func (s *Service) MarkFailed(ctx context.Context, id uint64) error {
	if id == 0 {
		return ErrInvalidInput
	}
	return s.repo.MarkFailed(ctx, id)
}

func (s *Service) MarkProcessed(ctx context.Context, consumerName string, event *Outbox) error {
	consumerName = strings.TrimSpace(consumerName)
	if consumerName == "" || event == nil || event.EventID == "" || event.TenantID == 0 {
		return ErrInvalidInput
	}
	processed, err := s.repo.IsProcessed(ctx, consumerName, event.EventID)
	if err != nil {
		return err
	}
	if processed {
		return nil
	}
	now := time.Now()
	if s != nil && s.now != nil {
		now = s.now()
	}
	return s.repo.MarkProcessed(ctx, &EventProcessed{
		ConsumerName: consumerName,
		EventID:      event.EventID,
		TenantID:     event.TenantID,
		ProcessedAt:  now,
	})
}

func newEventID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}

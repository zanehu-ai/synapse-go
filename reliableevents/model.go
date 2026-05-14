package reliableevents

import "time"

// Outbox maps to an outbox table for reliable event delivery.
type Outbox struct {
	ID            uint64  `gorm:"primaryKey;autoIncrement"`
	EventID       string  `gorm:"size:32;not null;uniqueIndex:uk_event_id"`
	TenantID      uint64  `gorm:"not null"`
	EventType     string  `gorm:"size:64;not null"`
	Version       int     `gorm:"not null;default:1"`
	Actor         []byte  `gorm:"type:json"`
	CorrelationID *string `gorm:"size:64"`
	Source        string  `gorm:"size:64;not null"`
	Payload       []byte  `gorm:"type:json;not null"`
	Status        int8    `gorm:"not null;default:0"` // 0=pending 1=sent 2=failed
	RetryCount    int     `gorm:"not null;default:0"`
	NextRetryAt   *time.Time
	OccurredAt    time.Time `gorm:"not null"`
	SentAt        *time.Time
}

func (Outbox) TableName() string { return "outbox" }

// EventProcessed maps to a consumer idempotency table.
type EventProcessed struct {
	ConsumerName string    `gorm:"primaryKey;size:64"`
	EventID      string    `gorm:"primaryKey;size:32"`
	TenantID     uint64    `gorm:"not null"`
	ProcessedAt  time.Time `gorm:"not null"`
}

func (EventProcessed) TableName() string { return "event_processed" }

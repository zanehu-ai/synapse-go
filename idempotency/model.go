package idempotency

import "time"

const (
	StatusInProgress         = "in_progress"
	StatusCompleted          = "completed"
	StatusCompletedSensitive = "completed_sensitive"
)

// Record maps to a tenant-scoped idempotency record.
type Record struct {
	ID               uint64     `gorm:"primaryKey;autoIncrement"`
	TenantID         uint64     `gorm:"not null;uniqueIndex:uk_idempotency_scope"`
	PrincipalID      uint64     `gorm:"not null;uniqueIndex:uk_idempotency_scope"`
	Key              string     `gorm:"column:idempotency_key;size:255;not null;uniqueIndex:uk_idempotency_scope"`
	Method           string     `gorm:"size:16;not null;uniqueIndex:uk_idempotency_scope"`
	Path             string     `gorm:"size:255;not null;uniqueIndex:uk_idempotency_scope"`
	RequestHash      string     `gorm:"size:80;not null"`
	Status           string     `gorm:"size:32;not null"`
	ResponseStatus   *int       `gorm:"column:response_status"`
	ResponseBody     *string    `gorm:"column:response_body;type:mediumtext"`
	CreatedAt        time.Time  `gorm:"not null"`
	UpdatedAt        time.Time  `gorm:"not null"`
	ExpiresAt        time.Time  `gorm:"not null;index:idx_idempotency_expiry"`
	RunningExpiresAt *time.Time `gorm:"column:running_expires_at;index:idx_idempotency_running_expiry"`
}

func (Record) TableName() string { return "idempotency_records" }

// BeginResult is the decision from checking an Idempotency-Key.
type BeginResult struct {
	Record         *Record
	Started        bool
	Replay         bool
	ResponseStatus int
	ResponseBody   string
}

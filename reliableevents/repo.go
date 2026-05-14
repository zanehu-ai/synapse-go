package reliableevents

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repo handles outbox persistence.
type Repo struct {
	db *gorm.DB
}

// NewRepo creates a Repo.
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) InsertOutbox(ctx context.Context, e *Outbox) error {
	err := r.db.WithContext(ctx).Create(e).Error
	if isDuplicateKey(err) {
		return ErrDuplicateEvent
	}
	return err
}

// InsertOutboxTx inserts an outbox row inside a caller-supplied transaction.
func (r *Repo) InsertOutboxTx(tx *gorm.DB, e *Outbox) error {
	err := tx.Create(e).Error
	if isDuplicateKey(err) {
		return ErrDuplicateEvent
	}
	return err
}

// MarkProcessed records idempotency for a consumer+event pair.
func (r *Repo) MarkProcessed(ctx context.Context, ep *EventProcessed) error {
	err := r.db.WithContext(ctx).Create(ep).Error
	if isDuplicateKey(err) {
		return nil
	}
	return err
}

// IsProcessed returns true if consumer+eventID has already been processed.
func (r *Repo) IsProcessed(ctx context.Context, consumerName, eventID string) (bool, error) {
	var count int64
	res := r.db.WithContext(ctx).Model(&EventProcessed{}).
		Where("consumer_name = ? AND event_id = ?", consumerName, eventID).
		Count(&count)
	return count > 0, res.Error
}

// FetchPending returns pending rows ordered oldest-first.
func (r *Repo) FetchPending(ctx context.Context, limit int) ([]*Outbox, error) {
	var rows []*Outbox
	res := r.db.WithContext(ctx).
		Where("status = 0 AND (next_retry_at IS NULL OR next_retry_at <= ?)", time.Now()).
		Order("occurred_at").
		Limit(limit).
		Find(&rows)
	return rows, res.Error
}

func (r *Repo) List(ctx context.Context, limit, offset int) ([]*Outbox, error) {
	var rows []*Outbox
	res := r.db.WithContext(ctx).
		Order("occurred_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows)
	return rows, res.Error
}

// MarkSent sets status=1 and sent_at=now for an outbox row.
func (r *Repo) MarkSent(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&Outbox{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": 1, "sent_at": now}).Error
}

// MarkFailed increments retry_count and sets next_retry_at with exponential backoff.
func (r *Repo) MarkFailed(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE outbox
		SET status       = CASE WHEN retry_count >= 5 THEN 2 ELSE 0 END,
		    retry_count  = retry_count + 1,
		    next_retry_at = DATE_ADD(NOW(6), INTERVAL POWER(2, LEAST(retry_count, 5)) * 10 SECOND)
		WHERE id = ?`, id).Error
}

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "1062") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint")
}

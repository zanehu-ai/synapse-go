package idempotency

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repo handles idempotency_records persistence.
type Repo struct {
	db *gorm.DB
}

// NewRepo creates a Repo.
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(ctx context.Context, rec *Record) error {
	err := r.db.WithContext(ctx).Create(rec).Error
	if isDuplicateKey(err) {
		return ErrDuplicate
	}
	return err
}

func (r *Repo) Get(ctx context.Context, tenantID, principalID uint64, method, path, key string) (*Record, error) {
	var rec Record
	res := r.db.WithContext(ctx).
		Where("tenant_id = ? AND principal_id = ? AND method = ? AND path = ? AND idempotency_key = ?",
			tenantID, principalID, method, path, key).
		First(&rec)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &rec, res.Error
}

func (r *Repo) Complete(ctx context.Context, id uint64, status string, responseStatus int, responseBody string) error {
	if status == "" {
		status = StatusCompleted
	}
	res := r.db.WithContext(ctx).Model(&Record{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":             status,
			"response_status":    responseStatus,
			"response_body":      responseBody,
			"running_expires_at": nil,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Restart resets a stale in-progress record while keeping its unique scope.
func (r *Repo) Restart(ctx context.Context, id uint64, requestHash string, runningExpiresAt, updatedAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&Record{}).
		Where("id = ? AND status = ?", id, StatusInProgress).
		Updates(map[string]any{
			"request_hash":       requestHash,
			"running_expires_at": runningExpiresAt,
			"updated_at":         updatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) DeleteExpired(ctx context.Context, before time.Time, limit int) (int64, error) {
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	res := r.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Limit(limit).
		Delete(&Record{})
	return res.RowsAffected, res.Error
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

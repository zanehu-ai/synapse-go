package runtimeguard

import (
	"strings"
	"sync"
	"time"
)

type lockEntry struct {
	owner     string
	expiresAt time.Time
}

// LockManager provides short-lived best-effort process-local locks.
type LockManager struct {
	mu    sync.Mutex
	locks map[string]lockEntry
	now   func() time.Time
}

// NewLockManager creates an in-process lock manager.
func NewLockManager() *LockManager {
	return &LockManager{locks: map[string]lockEntry{}, now: time.Now}
}

// TryLock attempts to hold key for ttl. The returned unlock is idempotent.
func (m *LockManager) TryLock(key, owner string, ttl time.Duration) (func(), bool) {
	key = strings.TrimSpace(key)
	owner = strings.TrimSpace(owner)
	if m == nil || key == "" || owner == "" || ttl <= 0 {
		return func() {}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	if current, ok := m.locks[key]; ok && current.expiresAt.After(now) && current.owner != owner {
		return func() {}, false
	}
	m.locks[key] = lockEntry{owner: owner, expiresAt: now.Add(ttl)}
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if current, ok := m.locks[key]; ok && current.owner == owner {
			delete(m.locks, key)
		}
	}, true
}

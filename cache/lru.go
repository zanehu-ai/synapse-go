// Package cache provides minimal in-process caches for hot-path lookups
// (e.g. feature flags, tenant status) where each request would otherwise
// hit the database.
//
// The implementation is intentionally small: no external deps, no metrics,
// no eviction callbacks. For multi-node invalidation, use reliable_events
// or Redis pub/sub on top.
package cache

import (
	"container/list"
	"sync"
	"time"
)

// LRU is a goroutine-safe LRU cache with a per-entry TTL.
//
// Zero capacity disables the cache (every Get returns a miss).
// Zero ttl means entries never expire from age — only LRU eviction applies.
type LRU[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	now      func() time.Time
	ll       *list.List
	items    map[K]*list.Element
}

type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time // zero means no expiry
}

// New constructs an LRU. capacity <= 0 disables caching.
func New[K comparable, V any](capacity int, ttl time.Duration) *LRU[K, V] {
	return &LRU[K, V]{
		capacity: capacity,
		ttl:      ttl,
		now:      time.Now,
		ll:       list.New(),
		items:    make(map[K]*list.Element),
	}
}

// Get returns the cached value and true if present and not expired.
func (c *LRU[K, V]) Get(key K) (V, bool) {
	var zero V
	if c.capacity <= 0 {
		return zero, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return zero, false
	}
	en := el.Value.(*entry[K, V])
	if !en.expiresAt.IsZero() && c.now().After(en.expiresAt) {
		c.removeElement(el)
		return zero, false
	}
	c.ll.MoveToFront(el)
	return en.value, true
}

// Set inserts or updates a value, marking it most-recently-used.
func (c *LRU[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var expiresAt time.Time
	if c.ttl > 0 {
		expiresAt = c.now().Add(c.ttl)
	}
	if el, ok := c.items[key]; ok {
		en := el.Value.(*entry[K, V])
		en.value = value
		en.expiresAt = expiresAt
		c.ll.MoveToFront(el)
		return
	}
	en := &entry[K, V]{key: key, value: value, expiresAt: expiresAt}
	el := c.ll.PushFront(en)
	c.items[key] = el
	if c.ll.Len() > c.capacity {
		c.removeElement(c.ll.Back())
	}
}

// Delete drops a key. No-op if absent.
func (c *LRU[K, V]) Delete(key K) {
	if c.capacity <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.removeElement(el)
	}
}

// Purge clears all entries.
func (c *LRU[K, V]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ll.Init()
	c.items = make(map[K]*list.Element)
}

// Len returns the current number of cached entries (including not-yet-expired).
func (c *LRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

func (c *LRU[K, V]) removeElement(el *list.Element) {
	en := el.Value.(*entry[K, V])
	c.ll.Remove(el)
	delete(c.items, en.key)
}

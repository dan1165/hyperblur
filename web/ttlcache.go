package web

import (
	"sync"
	"time"
)

// ttlCache is a size-bounded map whose entries expire: enough for poll results,
// which only need to survive a few seconds of a reader watching a poll.
type ttlCache struct {
	mu      sync.Mutex
	entries map[string]ttlEntry
	max     int
	ttl     time.Duration
}

type ttlEntry struct {
	value   []byte
	expires time.Time
}

func newTTLCache(max int, ttl time.Duration) *ttlCache {
	return &ttlCache{entries: make(map[string]ttlEntry), max: max, ttl: ttl}
}

// Get returns a live entry, dropping it when expired.
func (c *ttlCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expires) {
		delete(c.entries, key)
		return nil, false
	}
	return entry.value, true
}

// Set stores a value, dropping expired entries when the cache is full.
func (c *ttlCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.max {
		c.sweepLocked()
	}
	c.entries[key] = ttlEntry{value: value, expires: time.Now().Add(c.ttl)}
}

// sweepLocked drops expired entries. If none are expired it drops the entry
// closest to expiring, so memory stays bounded without keeping an order.
func (c *ttlCache) sweepLocked() {
	now := time.Now()
	oldestKey := ""
	var oldest time.Time
	for key, entry := range c.entries {
		if now.After(entry.expires) {
			delete(c.entries, key)
			continue
		}
		// Compare against the running minimum, not against "now + ttl": the
		// clock can return the same instant twice, which would hide the entry.
		if oldestKey == "" || entry.expires.Before(oldest) {
			oldestKey, oldest = key, entry.expires
		}
	}
	if len(c.entries) >= c.max && oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

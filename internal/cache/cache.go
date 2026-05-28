// Package cache provides an in-memory LRU+TTL cache for GraphQL query responses.
package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"
	"time"
)

// Cache is a thread-safe LRU cache with per-entry TTL expiry.
type Cache struct {
	mu         sync.Mutex
	items      map[string]*list.Element
	lru        *list.List
	maxEntries int
	defaultTTL time.Duration
}

type entry struct {
	key       string
	data      []byte
	expiresAt time.Time
}

// New creates a Cache. maxEntries ≤ 0 defaults to 1000.
func New(maxEntries int, defaultTTL time.Duration) *Cache {
	if maxEntries <= 0 {
		maxEntries = 1000
	}
	if defaultTTL <= 0 {
		defaultTTL = 60 * time.Second
	}
	return &Cache{
		items:      make(map[string]*list.Element),
		lru:        list.New(),
		maxEntries: maxEntries,
		defaultTTL: defaultTTL,
	}
}

// Get returns the cached bytes for key, and whether it was found and not expired.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return nil, false
	}

	e := el.Value.(*entry)
	if time.Now().After(e.expiresAt) {
		c.lru.Remove(el)
		delete(c.items, key)
		return nil, false
	}

	c.lru.MoveToFront(el)
	return e.data, true
}

// Set stores data under key with the default TTL.
func (c *Cache) Set(key string, data []byte) {
	c.SetTTL(key, data, c.defaultTTL)
}

// SetTTL stores data with an explicit TTL.
func (c *Cache) SetTTL(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.lru.MoveToFront(el)
		el.Value.(*entry).data = data
		el.Value.(*entry).expiresAt = time.Now().Add(ttl)
		return
	}

	// Evict LRU entry when at capacity.
	if c.lru.Len() >= c.maxEntries {
		oldest := c.lru.Back()
		if oldest != nil {
			c.lru.Remove(oldest)
			delete(c.items, oldest.Value.(*entry).key)
		}
	}

	e := &entry{key: key, data: data, expiresAt: time.Now().Add(ttl)}
	el := c.lru.PushFront(e)
	c.items[key] = el
}

// Flush removes all entries from the cache.
func (c *Cache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.lru.Init()
}

// Len returns the number of entries currently in the cache (including expired).
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Len()
}

// ── Cache key ─────────────────────────────────────────────────────────────────

// QueryKey returns the cache key for a GraphQL query execution.
// Variables map is serialised with sorted keys for determinism.
func QueryKey(query, operationName, role string, variables map[string]any) string {
	varsJSON := "{}"
	if len(variables) > 0 {
		// Sort keys so identical variable maps produce the same key regardless
		// of insertion order.
		keys := make([]string, 0, len(variables))
		for k := range variables {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := make([]any, 0, len(keys)*2)
		for _, k := range keys {
			ordered = append(ordered, k, variables[k])
		}
		if b, err := json.Marshal(ordered); err == nil {
			varsJSON = string(b)
		}
	}

	raw := query + "\x00" + operationName + "\x00" + role + "\x00" + varsJSON
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

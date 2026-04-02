package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Key        string
	Value     interface{}
	Expiration time.Time
	Hits       int64
}

// Cache provides in-memory caching with TTL
type Cache struct {
	mu         sync.RWMutex
	items      map[string]*CacheItem
	maxSize    int
	hitCount   int64
	missCount  int64
	expiration time.Duration
}

// New creates a new Cache instance
func New(maxSize int, expiration time.Duration) *Cache {
	c := &Cache{
		items:      make(map[string]*CacheItem),
		maxSize:    maxSize,
		expiration: expiration,
	}
	// Start cleanup goroutine
	go c.cleanup()
	return c
}

// cleanup periodically removes expired items
func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.Expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// generateKey creates a cache key from input
func generateKey(prefix string, args ...interface{}) string {
	h := sha256.New()
	for _, arg := range args {
		fmt.Fprint(h, arg)
	}
	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(h.Sum(nil))[:16])
}

// Get retrieves an item from cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		c.missCount++
		return nil, false
	}

	if time.Now().After(item.Expiration) {
		delete(c.items, key)
		c.missCount++
		return nil, false
	}

	item.Hits++
	c.hitCount++
	return item.Value, true
}

// Set stores an item in cache
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.expiration)
}

// SetWithTTL stores an item with custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &CacheItem{
		Key:        key,
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Delete removes an item from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*CacheItem)
}

// evictOldest removes the least recently used item
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	now := time.Now()

	for key, item := range c.items {
		if oldestTime.IsZero() || item.Expiration.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.Expiration
		}
	}

	if now.After(oldestTime) {
		delete(c.items, oldestKey)
	}
}

// HitRate returns the cache hit rate
func (c *Cache) HitRate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	total := c.hitCount + c.missCount
	if total == 0 {
		return 0
	}
	return float64(c.hitCount) / float64(total)
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	total := c.hitCount + c.missCount
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hitCount) / float64(total)
	}

	return map[string]interface{}{
		"size":       len(c.items),
		"max_size":   c.maxSize,
		"hit_count":  c.hitCount,
		"miss_count": c.missCount,
		"hit_rate":   hitRate,
	}
}

// PetCache is a specialized cache for pet data
type PetCache struct {
	*Cache
}

// NewPetCache creates a new pet cache
func NewPetCache(maxSize int, expiration time.Duration) *PetCache {
	return &PetCache{
		Cache: New(maxSize, expiration),
	}
}

// GetPetKey generates a cache key for pet operations
func GetPetKey(id int64) string {
	return generateKey("pet", id)
}

// GetPetsListKey generates a cache key for pets list
func GetPetsListKey(page, pageSize int, filter string) string {
	return generateKey("pets_list", page, pageSize, filter)
}

// SessionCache handles user session caching
type SessionCache struct {
	*Cache
}

// NewSessionCache creates a new session cache
func NewSessionCache() *SessionCache {
	return &SessionCache{
		Cache: New(10000, 30*time.Minute),
	}
}

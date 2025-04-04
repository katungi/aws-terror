package cache

import (
	"sync"
	"time"
)

type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
}

type Cache struct {
	data  map[string]CacheEntry
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewCache creates a new cache with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		data: make(map[string]CacheEntry),
		ttl:  ttl,
	}
}

// Set adds a value to the cache with the specified key
func (c *Cache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}
}

// Get retrieves a value from the cache by key
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.Expiration) {
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.data, key)
		c.mutex.Unlock()
		c.mutex.RLock()
		return nil, false
	}

	return entry.Value, true
}

// Delete removes a value from the cache by key
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]CacheEntry)
}

// Cleanup removes expired entries from the cache
func (c *Cache) Cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	for key, entry := range c.data {
		if now.After(entry.Expiration) {
			delete(c.data, key)
		}
	}
}
package cache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	// Create a new cache with a 100ms TTL
	cache := NewCache(100 * time.Millisecond)

	// Test Set and Get
	t.Run("Set and Get", func(t *testing.T) {
		cache.Set("key1", "value1")
		value, exists := cache.Get("key1")
		if !exists {
			t.Error("expected key1 to exist")
		}
		if value != "value1" {
			t.Errorf("expected value1, got %v", value)
		}
	})

	// Test expiration
	t.Run("Expiration", func(t *testing.T) {
		cache.Set("key2", "value2")
		time.Sleep(200 * time.Millisecond) // Wait for expiration
		_, exists := cache.Get("key2")
		if exists {
			t.Error("expected key2 to be expired")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		cache.Set("key3", "value3")
		cache.Delete("key3")
		_, exists := cache.Get("key3")
		if exists {
			t.Error("expected key3 to be deleted")
		}
	})

	// Test Clear
	t.Run("Clear", func(t *testing.T) {
		cache.Set("key4", "value4")
		cache.Set("key5", "value5")
		cache.Clear()
		_, exists1 := cache.Get("key4")
		_, exists2 := cache.Get("key5")
		if exists1 || exists2 {
			t.Error("expected cache to be empty")
		}
	})

	// Test Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		cache.Set("key6", "value6")
		cache.Set("key7", "value7")
		time.Sleep(200 * time.Millisecond) // Wait for expiration
		cache.Cleanup()
		_, exists1 := cache.Get("key6")
		_, exists2 := cache.Get("key7")
		if exists1 || exists2 {
			t.Error("expected expired entries to be cleaned up")
		}
	})

	// Test concurrent access
	t.Run("Concurrent Access", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			go func() {
				cache.Set("concurrent", "value")
				cache.Get("concurrent")
			}()
		}
	})
}
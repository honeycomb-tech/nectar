package processors

import (
	"fmt"
	"sync"
	"testing"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache(3)

	// Test Put and Get
	cache.Put("key1", []byte("value1"))
	cache.Put("key2", []byte("value2"))
	cache.Put("key3", []byte("value3"))

	// Test Get existing keys
	if val, ok := cache.Get("key1"); !ok || string(val) != "value1" {
		t.Errorf("Expected value1, got %s", string(val))
	}

	if val, ok := cache.Get("key2"); !ok || string(val) != "value2" {
		t.Errorf("Expected value2, got %s", string(val))
	}

	if val, ok := cache.Get("key3"); !ok || string(val) != "value3" {
		t.Errorf("Expected value3, got %s", string(val))
	}

	// Test Get non-existing key
	if _, ok := cache.Get("key4"); ok {
		t.Error("Expected key4 to not exist")
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := NewLRUCache(2)

	// Fill cache
	cache.Put("key1", []byte("value1"))
	cache.Put("key2", []byte("value2"))

	// Access key1 to make it recently used
	cache.Get("key1")

	// Add key3, should evict key2 (least recently used)
	cache.Put("key3", []byte("value3"))

	// key1 should still exist
	if _, ok := cache.Get("key1"); !ok {
		t.Error("Expected key1 to still exist")
	}

	// key2 should be evicted
	if _, ok := cache.Get("key2"); ok {
		t.Error("Expected key2 to be evicted")
	}

	// key3 should exist
	if _, ok := cache.Get("key3"); !ok {
		t.Error("Expected key3 to exist")
	}
}

func TestLRUCache_UpdateExisting(t *testing.T) {
	cache := NewLRUCache(2)

	// Add initial values
	cache.Put("key1", []byte("value1"))
	cache.Put("key2", []byte("value2"))

	// Update key1
	cache.Put("key1", []byte("updated1"))

	// Check updated value
	if val, ok := cache.Get("key1"); !ok || string(val) != "updated1" {
		t.Errorf("Expected updated1, got %s", string(val))
	}

	// Add key3, should evict key2 since key1 was recently updated
	cache.Put("key3", []byte("value3"))

	// key1 should still exist
	if _, ok := cache.Get("key1"); !ok {
		t.Error("Expected key1 to still exist after update")
	}

	// key2 should be evicted
	if _, ok := cache.Get("key2"); ok {
		t.Error("Expected key2 to be evicted")
	}
}

func TestLRUCache_Concurrent(t *testing.T) {
	cache := NewLRUCache(100)
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				cache.Put(key, []byte(value))
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// Verify some entries exist (cache should have evicted some)
	found := 0
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numOperations; j++ {
			key := fmt.Sprintf("key-%d-%d", i, j)
			if _, ok := cache.Get(key); ok {
				found++
			}
		}
	}

	// Should have at most 100 entries (cache capacity)
	if found > 100 {
		t.Errorf("Found %d entries, expected at most 100", found)
	}

	// Should have some entries (not all evicted)
	if found == 0 {
		t.Error("Expected some entries to remain in cache")
	}
}

func TestLRUCache_ZeroCapacity(t *testing.T) {
	cache := NewLRUCache(0)

	// Should not panic
	cache.Put("key1", []byte("value1"))

	// Should not store anything
	if _, ok := cache.Get("key1"); ok {
		t.Error("Expected cache with 0 capacity to not store anything")
	}
}

func BenchmarkLRUCache_Put(b *testing.B) {
	cache := NewLRUCache(1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		cache.Put(key, []byte(value))
	}
}

func BenchmarkLRUCache_Get(b *testing.B) {
	cache := NewLRUCache(1000)
	
	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		cache.Put(key, []byte(value))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkLRUCache_ConcurrentMixed(b *testing.B) {
	cache := NewLRUCache(1000)
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				key := fmt.Sprintf("key%d", i)
				value := fmt.Sprintf("value%d", i)
				cache.Put(key, []byte(value))
			} else {
				key := fmt.Sprintf("key%d", i%1000)
				cache.Get(key)
			}
			i++
		}
	})
}
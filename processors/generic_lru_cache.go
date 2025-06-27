package processors

import (
	"container/list"
	"sync"
)

// GenericLRUCache is a thread-safe LRU cache for any type
type GenericLRUCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mutex    sync.RWMutex
}

type genericCacheEntry struct {
	key   string
	value interface{}
}

// NewGenericLRUCache creates a new generic LRU cache
func NewGenericLRUCache(capacity int) *GenericLRUCache {
	return &GenericLRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

// Get retrieves a value from the cache
func (c *GenericLRUCache) Get(key string) (interface{}, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.list.MoveToFront(elem)
		return elem.Value.(*genericCacheEntry).value, true
	}
	return nil, false
}

// Put adds or updates a value in the cache
func (c *GenericLRUCache) Put(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.list.MoveToFront(elem)
		elem.Value.(*genericCacheEntry).value = value
		return
	}

	// Add new entry
	entry := &genericCacheEntry{key: key, value: value}
	elem := c.list.PushFront(entry)
	c.cache[key] = elem

	// Remove oldest if over capacity
	if c.list.Len() > c.capacity {
		oldest := c.list.Back()
		if oldest != nil {
			c.list.Remove(oldest)
			delete(c.cache, oldest.Value.(*genericCacheEntry).key)
		}
	}
}

// Size returns the current size of the cache
func (c *GenericLRUCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.cache)
}

// Clear removes all entries from the cache
func (c *GenericLRUCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = make(map[string]*list.Element)
	c.list.Init()
}
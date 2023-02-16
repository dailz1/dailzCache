package dailzLRU

import (
	"dailzLRU/lru"
	"sync"
)

const (
	// DefaultEvictedBufferSize defines the default buffer size to store evicted key/val
	DefaultEvictedBufferSize = 16
)

// Cache is a thread-safe fixed size LRU cache.
type Cache[K comparable, V any] struct {
	lru         *lru.LRU[K, V]
	evictedKeys []K
	evictedVals []V
	onEvictedCB func(k K, v V)
	lock        sync.RWMutex
}

func New[K comparable, V any](size int) (*Cache[K, V], error) {
	return NewWithEvict[K, V](size, nil)
}

func NewWithEvict[K comparable, V any](size int, onEvicted func(key K, value V)) (c *Cache[K, V], err error) {
	c = &Cache[K, V]{
		onEvictedCB: onEvicted,
	}
	if onEvicted != nil {
		c.initEvictBuffers()
		onEvicted = c.onEvicted
	}
	c.lru, err = lru.NewLRU(size, onEvicted)
	return
}

func (c *Cache[K, V]) initEvictBuffers() {
	c.evictedKeys = make([]K, 0, DefaultEvictedBufferSize)
	c.evictedVals = make([]V, 0, DefaultEvictedBufferSize)
}

// onEvicted save evicted key/val and sent in externally registered callback
// outside of critical section
func (c *Cache[K, V]) onEvicted(k K, v V) {
	c.evictedKeys = append(c.evictedKeys, k)
	c.evictedVals = append(c.evictedVals, v)
}

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.Lock()
	value, ok = c.lru.Get(key)
	c.lock.Unlock()
	return
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *Cache[K, V]) Add(key K, value V) (evicted bool) {
	var k K
	var v V
	c.lock.Lock()
	evicted = c.lru.Add(key, value)
	if c.onEvictedCB != nil && evicted {
		k = c.evictedKeys[0]
		v = c.evictedVals[0]
		c.evictedKeys = c.evictedKeys[:0]
		c.evictedVals = c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && evicted {
		c.onEvictedCB(k, v)
	}
	return
}

func (c *Cache[K, V]) Contains(key K) (containKey bool) {
	c.lock.RLock()
	containKey = c.lru.Contains(key)
	c.lock.RUnlock()
	return
}

func (c *Cache[K, V]) ContainsOrAdd(key K, value V) (ok, evicted bool) {
	var k K
	var v V
	c.lock.Lock()
	if c.lru.Contains(key) {
		c.lock.Unlock()
		return true, false
	}
	evicted = c.lru.Add(key, value)
	if c.onEvictedCB != nil && evicted {
		k = c.evictedKeys[0]
		v = c.evictedVals[0]
		c.evictedKeys = c.evictedKeys[:0]
		c.evictedVals = c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && evicted {
		c.onEvictedCB(k, v)
	}
	return false, evicted
}

func (c *Cache[K, V]) PeekOrAdd(key K, value V) (previous V, ok, evicted bool) {
	var k K
	var v V
	c.lock.Lock()
	previous, ok = c.lru.Peek(key)
	if ok {
		c.lock.Unlock()
		return previous, true, false
	}
	evicted = c.lru.Add(key, value)
	if c.onEvictedCB != nil && evicted {
		k = c.evictedKeys[0]
		v = c.evictedVals[0]
		c.evictedKeys = c.evictedKeys[:0]
		c.evictedVals = c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && evicted {
		c.onEvictedCB(k, v)
	}
	return
}

func (c *Cache[K, V]) Remove(key K) (present bool) {
	var k K
	var v V
	c.lock.Lock()
	present = c.lru.Remove(key)
	if c.onEvictedCB != nil && present {
		k = c.evictedKeys[0]
		v = c.evictedVals[0]
		c.evictedKeys = c.evictedKeys[:0]
		c.evictedVals = c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && present {
		c.onEvictedCB(k, v)
	}
	return
}

func (c *Cache[K, V]) Resize(size int) (evicted int) {
	var ks []K
	var vs []V
	c.lock.Lock()
	evicted = c.lru.Resize(size)
	if c.onEvictedCB != nil && evicted > 0 {
		ks = c.evictedKeys
		vs = c.evictedVals
		c.initEvictBuffers()
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && evicted > 0 {
		for i := 0; i < len(ks); i++ {
			c.onEvictedCB(ks[i], vs[i])
		}
	}
	return evicted
}

func (c *Cache[K, V]) RemoveOldest() (key K, value V, ok bool) {
	var k K
	var v V
	c.lock.Lock()
	key, value, ok = c.lru.RemoveOldest()
	if c.onEvictedCB != nil && ok {
		k = c.evictedKeys[0]
		v = c.evictedVals[0]
		c.evictedKeys = c.evictedKeys[:0]
		c.evictedVals = c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && ok {
		c.onEvictedCB(k, v)
	}
	return
}

func (c *Cache[K, V]) GetOldest() (key K, value V, ok bool) {
	c.lock.RLock()
	key, value, ok = c.lru.GetOldest()
	c.lock.RUnlock()
	return
}

func (c *Cache[K, V]) Keys() []K {
	c.lock.RLock()
	keys := c.lru.Keys()
	c.lock.RUnlock()
	return keys
}

func (c *Cache[K, V]) Len() int {
	c.lock.RLock()
	length := c.lru.Len()
	c.lock.RUnlock()
	return length
}

// Purge is used to completely clear the cache.
func (c *Cache[K, V]) Purge() {
	var ks []K
	var vs []V
	c.lock.Lock()
	c.lru.Purge()
	if c.onEvictedCB != nil && len(c.evictedKeys) > 0 {
		ks = c.evictedKeys
		vs = c.evictedVals
		c.initEvictBuffers()
	}
	c.lock.Unlock()

	if c.onEvictedCB != nil {
		for i := 0; i < len(ks); i++ {
			c.onEvictedCB(ks[i], vs[i])
		}
	}
}

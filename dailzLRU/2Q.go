package dailzLRU

import (
	"dailzLRU/lru"
	"errors"
	"sync"
)

const (
	Default2QRecentRatio  = 0.25
	Default2QGhostEntries = 0.5
)

type TwoQueueCache[K comparable, V any] struct {
	size       int
	recentSize int

	recent      *lru.LRU[K, V]
	frequent    *lru.LRU[K, V]
	recentEvict *lru.LRU[K, V]
	lock        sync.RWMutex
}

func New2Q[K comparable, V any](size int) (*TwoQueueCache[K, V], error) {
	return New2QWithParam[K, V](size, Default2QRecentRatio, Default2QGhostEntries)
}

func New2QWithParam[K comparable, V any](size int, recentRatio, ghostRatio float64) (*TwoQueueCache[K, V], error) {
	if size <= 0 {
		return nil, errors.New("invalid size")
	}

	if recentRatio < 0.0 || recentRatio > 1.0 {
		return nil, errors.New("invalid recent ratio")
	}

	if ghostRatio < 0.0 || ghostRatio > 1.0 {
		return nil, errors.New("invalid ghost ratio")
	}

	recentSize := int(float64(size) * recentRatio)
	evictSize := int(float64(size) * ghostRatio)

	recent, err := lru.NewLRU[K, V](size, nil)
	if err != nil {
		return nil, err
	}

	frequent, err := lru.NewLRU[K, V](size, nil)
	if err != nil {
		return nil, err
	}

	recentEvict, err := lru.NewLRU[K, V](evictSize, nil)
	if err != nil {
		return nil, err
	}

	c := &TwoQueueCache[K, V]{
		size:        size,
		recentSize:  recentSize,
		recent:      recent,
		frequent:    frequent,
		recentEvict: recentEvict,
	}
	return c, nil
}

func (c *TwoQueueCache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if value, ok = c.frequent.Get(key); ok {
		return value, ok
	}

	if value, ok := c.recent.Peek(key); ok {
		c.recent.Remove(key)
		c.frequent.Add(key, value)
		return value, ok
	}
	return
}

func (c *TwoQueueCache[K, V]) Add(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.frequent.Contains(key) {
		c.frequent.Add(key, value)
		return
	}

	if c.recent.Contains(key) {
		c.recent.Remove(key)
		c.frequent.Add(key, value)
		return
	}

	if c.recentEvict.Contains(key) {
		c.ensureSpace(true)
		c.recentEvict.Remove(key)
		c.frequent.Add(key, value)
		return
	}
	c.ensureSpace(false)
	c.recent.Add(key, value)
}

func (c *TwoQueueCache[K, V]) ensureSpace(recentEvict bool) {
	recentLen := c.recent.Len()
	freqLen := c.frequent.Len()
	if recentLen+freqLen < c.size {
		return
	}

	if recentLen > 0 && (recentLen > c.recentSize || recentLen == c.recentSize && !recentEvict) {
		k, _, _ := c.recent.RemoveOldest()
		var empty V
		c.recentEvict.Add(k, empty)
		return
	}
	c.frequent.RemoveOldest()
}

func (c *TwoQueueCache[K, V]) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.recent.Len() + c.frequent.Len()
}

func (c *TwoQueueCache[K, V]) Keys() []K {
	c.lock.RLock()
	defer c.lock.RUnlock()
	k1 := c.frequent.Keys()
	k2 := c.recent.Keys()
	return append(k1, k2...)
}

func (c *TwoQueueCache[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.frequent.Remove(key) {
		return
	}
	if c.recent.Remove(key) {
		return
	}
	if c.recentEvict.Remove(key) {
		return
	}
}

func (c *TwoQueueCache[K, V]) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.recent.Purge()
	c.frequent.Purge()
	c.recentEvict.Purge()
}

func (c *TwoQueueCache[K, V]) Contains(key K) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.frequent.Contains(key) || c.recent.Contains(key)
}

func (c *TwoQueueCache[K, V]) Peek(key K) (value V, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if value, ok := c.frequent.Peek(key); ok {
		return value, ok
	}
	return c.recent.Peek(key)
}
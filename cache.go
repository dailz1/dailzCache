package main

import (
	"dailzCache/lru"
	"sync"
)

type cache struct {
	mu         sync.Mutex
	usedBytes  int64 // number of all keys and values
	lru        *lru.Cache
	maxEntries int
	hitNum     int64
	getNum     int64
	evictNum   int64 // number of evictions
}

// CacheStats are returned by stats accessors on Group.
type CacheStats struct {
	Bytes     int64
	Items     int64
	Gets      int64
	Hits      int64
	Evictions int64
}

func (c *cache) stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{
		Bytes:     c.usedBytes,
		Items:     c.itemsLocked(),
		Gets:      c.getNum,
		Hits:      c.hitNum,
		Evictions: c.evictNum,
	}
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.maxEntries, func(key lru.Key, value lru.Value) {
			val := value.(ByteView)
			c.usedBytes -= int64(len(key.(string))) + int64(val.Len())
		})
	}
	c.lru.Add(key, value)
	c.usedBytes += int64(len(key)) + int64(value.Len())
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getNum++
	if c.lru == nil {
		return
	}

	if value, ok := c.lru.Get(key); ok {
		c.hitNum++
		return value.(ByteView), ok
	}

	return
}

func (c *cache) removeOldest() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru != nil {
		c.lru.RemoveOldest()
	}
}

func (c *cache) bytes() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.usedBytes
}

func (c *cache) itemsLocked() int64 {
	if c.lru == nil {
		return 0
	}
	return int64(c.lru.Len())
}

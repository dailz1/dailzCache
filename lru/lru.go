package lru

import (
	"container/list"
)

// Cache is an LRU cache. It is not safe for concurrent access.
type Cache struct {
	// maxEntries is the maxinum number of cache entries before
	// an item is evicted. Zero means no limit.
	maxEntries int

	list  *list.List
	cache map[Key]*list.Element

	// onEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	onEvicted func(key Key, value Value)
}

// A Key may be any value.
type Key interface {
	//KeyLen() int64
}

// A Value may be any value/
type Value interface {
	//ValueLen() int64
}

// entry represents the entity stored in the Cache
type entry struct {
	key   Key
	value Value
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func New(maxEntries int, onEvicted func(Key, Value)) *Cache {
	return &Cache{
		maxEntries: maxEntries,
		list:       list.New(),
		cache:      make(map[Key]*list.Element),
		onEvicted:  onEvicted,
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key Key, value Value) {
	if c.cache == nil {
		c.cache = make(map[Key]*list.Element)
		c.list = list.New()
	}

	if element, ok := c.cache[key]; ok {
		c.list.MoveToBack(element)
		kv := element.Value.(*entry)
		kv.value = value
	} else {
		element := c.list.PushBack(&entry{key: key, value: value})
		c.cache[key] = element
	}
	if c.maxEntries != 0 && c.Len() > c.maxEntries {
		c.RemoveOldest()
	}
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key Key) (value Value, ok bool) {
	if c.cache == nil {
		return
	}

	if element, ok := c.cache[key]; ok {
		c.list.MoveToBack(element)
		kv := element.Value.(*entry)
		return kv.value, true
	}
	return
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key Key) {
	if c.cache == nil {
		return
	}

	if element, ok := c.cache[key]; ok {
		c.removeElement(element)
	}
}

// RemoveOldest removes the oldest item from the cache
func (c *Cache) RemoveOldest() {
	if c.cache == nil {
		return
	}

	element := c.list.Front()
	if element != nil {
		c.removeElement(element)
	}
}

func (c *Cache) removeElement(element *list.Element) {
	c.list.Remove(element)
	kv := element.Value.(*entry)
	delete(c.cache, kv.key)
	if c.onEvicted != nil {
		c.onEvicted(kv.key, kv.value)
	}
}

// Len the number of cache entries, not calculate the kv's len
func (c *Cache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.list.Len()
}

// Clear purges all stored items from the cache.
func (c *Cache) Clear() {
	if c.onEvicted != nil {
		for _, element := range c.cache {
			kv := element.Value.(*entry)
			c.onEvicted(kv.key, kv.value)
		}
	}
	c.list = nil
	c.cache = nil
}

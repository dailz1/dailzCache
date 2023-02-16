package lru

import "container/list"

type historyCache struct {
	cache    Cache
	countMap map[Key]int
	k        int
}

type LRUKCache struct {
	historyCache *historyCache
	maxEntries   int
	list         *list.List
	cache        map[Key]*list.Element
}

func NewLRUK(LRUKMaxEntries int, LRUMaxEntries int, onEvicted func(Key, Value)) *LRUKCache {
	historyCache := &historyCache{
		cache:    *New(LRUMaxEntries, onEvicted),
		countMap: nil,
		k:        0,
	}
	return &LRUKCache{
		historyCache: historyCache,
		maxEntries:   LRUKMaxEntries,
		list:         list.New(),
		cache:        make(map[Key]*list.Element),
	}
}

func (kc *LRUKCache) Add(key Key, value Value) {
	if kc.historyCache.cache.cache == nil {
		kc.historyCache.cache.cache = make(map[Key]*list.Element)

	}
}

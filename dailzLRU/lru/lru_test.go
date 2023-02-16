package lru

import "testing"

func TestLRU(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k int, v int) {
		if k != v {
			t.Fatalf("Evict values not equal (%v != %v)", k, v)
		}
		evictCounter++
	}

	//test NewLRU
	l, err := NewLRU(128, onEvicted)
	if err != nil {
		t.Fatalf("NewLRU error: %v", err)
	}

	// test Add()
	for i := 0; i < 256; i++ {
		l.Add(i, i)
	}
	// test Len()
	if l.Len() != 128 {
		t.Fatalf("LRU error: bad Len = %v", l.Len())
	}

	// test onEvicted
	if evictCounter != 128 {
		t.Fatalf("LRU error: bad evict count = %v", evictCounter)
	}

	// test Get()
	for i, k := range l.Keys() {
		if v, ok := l.Get(k); !ok || v != k || v != i+128 {
			t.Fatalf("LRU error: bad key = %v", k)
		}
	}

	for i := 0; i < 128; i++ {
		if _, ok := l.Get(i); ok {
			t.Fatalf("LRU error: key = %v should be evicted", i)
		}
	}

	for i := 128; i < 256; i++ {
		if _, ok := l.Get(i); !ok {
			t.Fatalf("LRU error: key = %v should not be evicted", i)
		}
	}

	for i := 128; i < 192; i++ {
		if ok := l.Remove(i); !ok {
			t.Fatalf("LRU error: key = %v should be contained", i)
		}

		if ok := l.Remove(i); ok {
			t.Fatalf("LRU error: key = %v should not be contained", i)
		}
		if _, ok := l.Get(i); ok {
			t.Fatalf("LRU error: key = %v should be deleted", i)
		}
	}

	l.Get(192)

	for i, k := range l.Keys() {
		if (i < 63 && k != i+193) || (i == 63 && k != 192) {
			t.Fatalf("LRU error: out of order key: %v", k)
		}
	}

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("LRU error: bad len: %v", l.Len())
	}

	if _, ok := l.Get(200); ok {
		t.Fatalf("LRU error: should contain nothing")
	}
}



package dailzLRU

import (
	"crypto/rand"
	"math"
	"math/big"
	"testing"
)

func TestLRU(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k int, v int) {
		if k != v {
			t.Fatalf("Evict values not equal (%v != %v)", k, v)
		}
		evictCounter++
	}

	cache, err := NewWithEvict(128, onEvicted)
	if err != nil {
		t.Fatalf("LRU error: %v", err)
	}
	for i := 0; i < 256; i++ {
		cache.Add(i, i)
	}

	if cache.Len() != 128 {
		t.Fatalf("LRU error: bad Len = %v", cache.Len())
	}
	for i := 128; i < 256; i++ {
		if _, ok := cache.Get(i); !ok {
			t.Fatalf("LRU error: key = %v should not be evicted", i)
		}
	}

	for i := 128; i < 192; i++ {
		if ok := cache.Remove(i); !ok {
			t.Fatalf("LRU error: key = %v should be contained", i)
		}

		if ok := cache.Remove(i); ok {
			t.Fatalf("LRU error: key = %v should not be contained", i)
		}
		if _, ok := cache.Get(i); ok {
			t.Fatalf("LRU error: key = %v should be deleted", i)
		}
	}

	cache.Get(192)

	for i, k := range cache.Keys() {
		if (i < 63 && k != i+193) || (i == 63 && k != 192) {
			t.Fatalf("LRU error: out of order key: %v", k)
		}
	}

	cache.Purge()
	if cache.Len() != 0 {
		t.Fatalf("LRU error: bad len: %v", cache.Len())
	}

	if _, ok := cache.Get(200); ok {
		t.Fatalf("LRU error: should contain nothing")
	}
}

func BenchmarkLRU_Rand(b *testing.B) {
	l, err := New[int64, int64](8192)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = getRand(b) % 32768
	}
	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			if _, ok := l.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func BenchmarkLRU_Freq(b *testing.B) {
	l, err := New[int64, int64](8192)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = getRand(b) % 16384
		} else {
			trace[i] = getRand(b) % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		if _, ok := l.Get(trace[i]); ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func getRand(tb testing.TB) int64 {
	out, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		tb.Fatal(err)
	}
	return out.Int64()
}

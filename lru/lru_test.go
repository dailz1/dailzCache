package lru

import (
	"fmt"
	"log"
	"testing"
)

type simpleStruct struct {
	para1 int
	para2 string
}

type complexStruct struct {
	para1 int
	para2 simpleStruct
}

var TestArgs = []struct {
	name       string
	keyToAdd   Key
	keyToGet   Key
	expectedOk bool
}{
	{"string_hit", "myKey1", "myKey1", true},
	{"string_miss", "myKey2", "nonsense", false},
	{"simple_struct_hit", simpleStruct{1, "one"}, simpleStruct{1, "one"}, true},
	{"simple_struct_miss", simpleStruct{1, "two"}, simpleStruct{0, "noway"}, false},
	{"complex_struct_hit", complexStruct{1, simpleStruct{2, "three"}},
		complexStruct{1, simpleStruct{2, "three"}}, true},
}

func TestGet(t *testing.T) {
	lru := New(0, nil)
	for _, tt := range TestArgs {
		lru.Add(tt.keyToAdd, 123)
		value, ok := lru.Get(tt.keyToGet)
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && value != 123 {
			t.Fatalf("%s expected get to return 123 but got %v", tt.name, value)
		}
	}
}

func TestRemove(t *testing.T) {
	lru := New(0, nil)
	for _, tt := range TestArgs {
		lru.Add(tt.keyToAdd, 123)
		value, ok := lru.Get(tt.keyToGet)
		if ok != tt.expectedOk {
			t.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && value != 123 {
			t.Fatalf("%s expected get to return 123 but got %v", tt.name, value)
		}

		lru.Remove(tt.keyToGet)
		if _, ok := lru.Get(tt.keyToGet); ok {
			t.Fatalf("TestRemove returned a removed entry")
		}
	}
}

func TestRemoveOldest(t *testing.T) {
	cap := 5

	lru := New(cap, nil)
	for _, tt := range TestArgs {
		lru.Add(tt.keyToAdd, 123)
		value, ok := lru.Get(tt.keyToGet)
		if ok != tt.expectedOk {
			log.Fatalf("%s: cache hit = %v; want %v", tt.name, ok, !ok)
		} else if ok && value != 123 {
			log.Fatalf("%s expected get to return 123 but got %v", tt.name, value)
		}
	}
	lru.Add(123, 123)
	if _, ok := lru.Get(TestArgs[0].keyToGet); ok || lru.Len() != 5 {
		t.Fatalf("RemoveOldest myKey1 failed")
	}

}

func TestEvict(t *testing.T) {
	keys := make([]Key, 0)
	callback := func(key Key, value Value) {
		keys = append(keys, key)
	}

	lru := New(9, callback)

	for i := 0; i < 10; i++ {
		lru.Add(fmt.Sprintf("myKey%d", i), 123)
	}

	if len(keys) != 1 {
		t.Fatalf("got %d evicted keys; want 1", len(keys))
	}
	if keys[0] != "myKey0" {
		t.Fatalf("got %v in first evicted key; want %v", keys[0], "myKey0")
	}
}

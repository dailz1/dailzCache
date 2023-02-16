package lru

// entry is an LRU entry
type entry[K comparable, V any] struct {
	next, prev *entry[K, V]
	list       *lruList[K, V] // The list to which this element belongs
	key        K              // The LRU key of this element
	value      V              // The LRU value of this element
}

// prevEntry returns lruList element or nil
func (e *entry[K, V]) prevEntry() *entry[K, V] {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

type lruList[K comparable, V any] struct {
	root entry[K, V]
	len  int
}

func (l *lruList[K, V]) init() *lruList[K, V] {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

// newList returns an initialized list
func newList[K comparable, V any]() *lruList[K, V] {
	return new(lruList[K, V]).init()
}

// length returns the number of elements of lruList
func (l *lruList[K, V]) length() int {
	return l.len
}

// back returns the last element of lruList or nil if the lruList is empty
func (l *lruList[K, V]) back() *entry[K, V] {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

// lazyInit lazily initializes a zero lruList value
func (l *lruList[K, V]) lazyInit() {
	if l.root.next == nil {
		l.init()
	}
}

// insert inserts e after at, increments lruList.len, and returns e
func (l *lruList[K, V]) insert(e *entry[K, V], at *entry[K, V]) *entry[K, V] {
	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
	e.list = l
	l.len++
	return e
}

// insertValue is a wrapper for insert
func (l *lruList[K, V]) insertValue(k K, v V, at *entry[K, V]) *entry[K, V] {
	return l.insert(&entry[K, V]{key: k, value: v}, at)
}

// remove removes e from its lruList, decrements lruList.len
func (l *lruList[K, V]) remove(e *entry[K, V]) V {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
	return e.value
}

// move moves e to next to at
func (l *lruList[K, V]) move(e, at *entry[K, V]) {
	if e == at {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	e.prev = at
	e.next = at.next
	e.prev.next = e
	e.next.prev = e
}

// pushFront inserts a new element e with value v at the front of lruList and returns e
func (l *lruList[K, V]) pushFront(k K, v V) *entry[K, V] {
	l.lazyInit()
	return l.insertValue(k, v, &l.root)
}

// moveToFront moves element e to the front of lruList.
// If e is not an element of lruList, the lruList is not modified.
// The element must not be nil.
func (l *lruList[K, V]) moveToFront(e *entry[K, V]) {
	if e.list != l || l.root.next == e {
		return
	}
	l.move(e, &l.root)
}

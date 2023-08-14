package arraymap

import (
	"golang.org/x/exp/slices"
)

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

// Entry stores a key-value pair in an ArrayMap.
type Entry[K Ordered, V any] struct {
	Key   K
	Value V
}

func entry[K Ordered, V any](k K, v V) Entry[K, V] {
	return Entry[K, V]{k, v}
}

type chunk[K Ordered, V any] struct {
	entries []Entry[K, V]
	min     K
	max     K
	end     int
	empty   bool
}

func newChunk[K Ordered, V any](chunkSize int) chunk[K, V] {
	return chunk[K, V]{
		entries: make([]Entry[K, V], chunkSize),
		empty:   true,
	}
}

func (c *chunk[K, V]) slice() []Entry[K, V] {
	if c.len() == 0 {
		return c.entries[0:0]
	}
	return c.entries[:c.end+1]
}

func (c *chunk[K, V]) insertAt(i int, e Entry[K, V]) {
	copy(c.entries[i+1:], c.entries[i:c.end+1])
	c.entries[i] = e
	c.end += 1
}

// pop removes the element at the given index in the chunk, shifting all elements to
// its right to the left by 1.
func (c *chunk[K, V]) pop(index int) {
	copy(c.entries[index:c.end+1], c.entries[index+1:c.end+1])
	if index == 0 {
		c.min = c.entries[0].Key
	}
	var zero Entry[K, V]
	c.entries[c.end] = zero
	c.end -= 1
	c.max = c.entries[c.end].Key
}

func (c *chunk[K, V]) len() int {
	if c.empty {
		return 0
	}
	return c.end + 1
}

func (c *chunk[K, V]) isFull() bool {
	return c.len() == len(c.entries)
}

// merge another chunk into the current chunk. Assumes all values in the second chunk
// are greater than the values in the current chunk, and that the current chunk has enough
// capacity.
func (c *chunk[K, V]) merge(other *chunk[K, V]) {
	copy(c.entries[c.end+1:], other.entries[:other.end+1])
	c.end += other.len()
	c.max = other.max
}

// split creates a new chunk, leaving the first half of the entries in the current chunk, and moving
// the remaining entries to a new chunk.
func (c *chunk[K, V]) split() chunk[K, V] {
	n := c.len() / 2

	// copy the right side half of the current chunk over to the new chunk
	newC := newChunk[K, V](len(c.entries))
	newEntries := c.entries[n : c.end+1]
	copy(newC.entries, newEntries)
	newC.end = len(newEntries) - 1
	newC.min = newEntries[0].Key
	newC.max = newEntries[newC.end].Key
	newC.empty = false

	// zero the right side half of the current chunk
	c.end = n - 1
	c.max = c.entries[c.end].Key
	var zero Entry[K, V]
	for i := c.end + 1; i < len(c.entries); i++ {
		c.entries[i] = zero
	}

	return newC
}

func cmpEntry[K Ordered, V any](e Entry[K, V], target K) int {
	if e.Key < target {
		return -1
	} else if e.Key == target {
		return 0
	} else {
		return 1
	}
}

// Insert or update an entry in the chunk. If the chunk does not have the capacity for
// a new entry, it splits the chunk in two, keeping the left half of entries in this
// chunk, and moves the right half of entries to a new chunk. It returns a pointer
// to the new chunk, if it created one, otherwise the pointer is nil.
func (c *chunk[K, V]) insert(k K, v V) *chunk[K, V] {
	if c.len() == 0 {
		c.min = k
		c.max = k
		c.entries[0] = entry(k, v)
		c.empty = false
		return nil
	}
	if k < c.min {
		c.min = k
		if c.isFull() {
			newChunk := c.split()
			c.insertAt(0, entry(k, v))
			c.min = k
			return &newChunk
		} else {
			c.insertAt(0, entry(k, v))
			return nil
		}
	} else if k > c.max {
		if c.isFull() {
			newChunk := c.split()
			newChunk.end += 1
			newChunk.entries[newChunk.end] = entry(k, v)
			newChunk.max = k
			return &newChunk
		} else {
			c.max = k
			c.end += 1
			c.entries[c.end] = entry(k, v)
			return nil
		}
	}

	// We're doing a linear search of the chunk here. This is faster for smaller chunk
	// sizes (at least up to 64) compared to a binary search. We could switch decide
	// between linear and binary searches based on the chunk size, but leaving it at
	// linear for now.
	for i := 0; i <= c.end; i++ {
		e := c.entries[i]
		if e.Key == k {
			c.entries[i] = entry(k, v)
			return nil
		}
		if e.Key > k {
			if c.isFull() {
				newChunk := c.split()
				if k > newChunk.min {
					newChunk.insert(k, v)
				} else {
					c.insert(k, v)
				}
				return &newChunk
			} else {
				c.insertAt(i, entry(k, v))
				return nil
			}
		}
	}
	panic("unreachable")
}

// Delete an entry with the given key from the chunk. Returns true if the entry
// exists, otherwise returns false.
func (c *chunk[K, V]) delete(k K) bool {
	n := c.len()
	if n == 0 || k < c.min || k > c.max {
		return false
	}
	if k == c.min {
		if n == 1 {
			var zero K
			c.empty = true
			c.end = 0
			c.min = zero
			c.max = zero
		} else {
			c.pop(0)
		}
		return true
	} else if k == c.max {
		c.pop(c.end)
		return true
	}
	for i := 1; i <= c.end-1; i++ {
		e := c.entries[i]
		if k == e.Key {
			c.pop(i)
			return true
		}
	}
	return false
}

// Get an entry from the chunk. Returns (entry, true) if an entry with the given key
// exists in the chunk, otherwise, it returns (_, false).
func (c *chunk[K, V]) get(k K) (V, bool) {
	var zero V
	if k < c.min || k > c.max {
		return zero, false
	}
	for _, entry := range c.entries[:c.end+1] {
		if entry.Key == k {
			return entry.Value, true
		}
	}
	return zero, false
}

func (c *chunk[K, V]) first() (Entry[K, V], bool) {
	if c.len() == 0 {
		var zero Entry[K, V]
		return zero, false
	}
	return c.entries[0], true
}

// An ArrayMap is an ordered map. It balances CPU cache-efficiency and insert/update/delete
// performance by storing entries in contiguous chunked arrays.
type ArrayMap[K Ordered, V any] struct {
	chunkSize int
	nmut      int
	chunks    []*chunk[K, V]
}

// Create a new ArrayMap with a given chunk size. The chunk size is a tuning parameter to
// adjust the tradeoff between cache-efficiency and insert/delete performance.
func New[K Ordered, V any](chunkSize int) *ArrayMap[K, V] {
	chunks := make([]*chunk[K, V], 0)
	return &ArrayMap[K, V]{chunkSize: chunkSize, chunks: chunks}
}

// Insert or update an entry in the ArrayMap.
func (m *ArrayMap[K, V]) Insert(k K, v V) {
	m.nmut += 1
	if m.nmut%100 == 0 {
		m.compact()
	}

	n := len(m.chunks)
	if n == 0 {
		chunk := newChunk[K, V](m.chunkSize)
		chunk.insert(k, v)
		m.chunks = append(m.chunks, &chunk)
		return
	}

	for i, chunk := range m.chunks {
		if i == n-1 {
			newChunk := chunk.insert(k, v)
			if newChunk != nil {
				m.chunks = append(m.chunks, newChunk)
			}
			return
		} else {
			nextC := m.chunks[i+1]
			if k >= chunk.min && k < nextC.min {
				newChunk := chunk.insert(k, v)
				if newChunk != nil {
					m.chunks = slices.Insert(m.chunks, i+1, newChunk)
				}
				return
			}
		}
	}
	panic("unreachable")
}

// Get a value with a given key from the ArrayMap. If the key does not exist in the map,
// it returns the empty entry and false.
func (m *ArrayMap[K, V]) Get(k K) (V, bool) {
	for _, chunk := range m.chunks {
		if k < chunk.min {
			break
		}
		v, ok := chunk.get(k)
		if ok {
			return v, ok
		}
	}
	var zero V
	return zero, false
}

func (m *ArrayMap[K, V]) removeChunk(i int) {
	m.chunks[i] = nil // zero the pointer so that the chunk can be GC'd
	m.chunks = slices.Delete(m.chunks, i, i+1)
}

// Delete an entry from the array map. Returns false if the key does not exist in the
// map.
func (m *ArrayMap[K, V]) Delete(k K) bool {
	m.nmut += 1
	if m.nmut%100 == 0 {
		m.compact()
	}

	for i, chunk := range m.chunks {
		if k < chunk.min {
			break
		}
		if chunk.delete(k) {
			if chunk.len() == 0 {
				m.removeChunk(i)
			}
			return true
		}
	}
	return false
}

func (m *ArrayMap[K, V]) compact() {
	i := 0
	for {
		if i >= len(m.chunks)-1 {
			break
		}
		cur := m.chunks[i]
		next := m.chunks[i+1]
		if float64(cur.len()+next.len()) < 0.75*float64(m.chunkSize) {
			cur.merge(next)
			m.removeChunk(i + 1)
		}
		i += 1
	}
}

// Len returns the number of entries in the map.
func (m *ArrayMap[K, V]) Len() int {
	length := 0
	for _, chunk := range m.chunks {
		length += chunk.len()
	}
	return length
}

// First returns the first element in the array map. If the map is empty, it returns
// false in the bool flag.
func (m *ArrayMap[K, V]) First() (Entry[K, V], bool) {
	if len(m.chunks) > 0 {
		return m.chunks[0].first()
	}
	var zero Entry[K, V]
	return zero, false
}

// Iter returns an IterMap object which may be used iterate over the map in sorted order.
func (m *ArrayMap[K, V]) Iter() *IterMap[K, V] {
	if len(m.chunks) == 0 {
		return &IterMap[K, V]{m: m, done: true}
	}
	return &IterMap[K, V]{
		m:        m,
		curChunk: m.chunks[0],
		chunkIdx: 0,
		entryIdx: 0,
	}
}

type IterMap[K Ordered, V any] struct {
	m        *ArrayMap[K, V]
	curChunk *chunk[K, V]
	chunkIdx int
	entryIdx int
	done     bool
}

// Returns the next entry in the map iterator. Returns the empty entry and false if the
// iterator after it reaches the end of the map.
func (it *IterMap[K, V]) Next() (Entry[K, V], bool) {
	if it.done {
		var zero Entry[K, V]
		return zero, false
	}

	entry := it.curChunk.entries[it.entryIdx]

	if it.entryIdx >= it.curChunk.end {
		if it.chunkIdx >= len(it.m.chunks)-1 {
			it.done = true
		} else {
			it.chunkIdx += 1
			it.curChunk = it.m.chunks[it.chunkIdx]
			it.entryIdx = 0
		}
	} else {
		it.entryIdx += 1
	}

	return entry, true
}

// Collect exhausts the map iterator, collecting all entries in a slice.
func (it *IterMap[K, V]) Collect() []Entry[K, V] {
	entries := make([]Entry[K, V], 0, it.m.Len())
	for {
		e, ok := it.Next()
		if !ok {
			break
		}
		entries = append(entries, e)
	}
	return entries
}

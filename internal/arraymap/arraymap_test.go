package arraymap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type E = Entry[int64, string]

func TestChunk(t *testing.T) {
	size := 4
	chunk := newChunk[int64, string](size)
	assert.Equal(t, chunk.len(), 0)

	assert.Nil(t, chunk.insert(1, "a"))
	assert.Equal(t, []E{{1, "a"}}, chunk.slice())
	assert.Equal(t, int64(1), chunk.max)
	assert.Equal(t, int64(1), chunk.min)
	assert.Equal(t, chunk.len(), 1)

	assert.Nil(t, chunk.insert(3, "c"))
	assert.Equal(t, 2, chunk.len())
	assert.Equal(t, []E{{1, "a"}, {3, "c"}}, chunk.slice())

	assert.Nil(t, chunk.insert(2, "b"))
	assert.Equal(t, int64(3), chunk.max)
	assert.Equal(t, int64(1), chunk.min)
	assert.Equal(t, 3, chunk.len())
	assert.Equal(t, []E{{1, "a"}, {2, "b"}, {3, "c"}}, chunk.slice())

	assert.Nil(t, chunk.insert(0, "x"))
	assert.Equal(t, int64(3), chunk.max)
	assert.Equal(t, int64(0), chunk.min)
	assert.Equal(t, 4, chunk.len())
	assert.Equal(t, []E{{0, "x"}, {1, "a"}, {2, "b"}, {3, "c"}}, chunk.slice())

	assert.Nil(t, chunk.insert(2, "bb"))
	assert.Equal(t, int64(3), chunk.max)
	assert.Equal(t, int64(0), chunk.min)
	assert.Equal(t, 4, chunk.len())
	assert.Equal(t, []E{{0, "x"}, {1, "a"}, {2, "bb"}, {3, "c"}}, chunk.slice())

	chunk.delete(1)
	assert.Equal(t, 3, chunk.len())
	assert.Equal(t, int64(0), chunk.min)
	assert.Equal(t, int64(3), chunk.max)
	assert.Equal(t, []E{{0, "x"}, {2, "bb"}, {3, "c"}}, chunk.slice())

	chunk.delete(0)
	assert.Equal(t, []E{{2, "bb"}, {3, "c"}}, chunk.slice())
	assert.Equal(t, 2, chunk.len())
	assert.Equal(t, int64(2), chunk.min)
	assert.Equal(t, int64(3), chunk.max)

	assert.Nil(t, chunk.insert(4, "d"))
	assert.Equal(t, 3, chunk.len())
	assert.Equal(t, []E{{2, "bb"}, {3, "c"}, {4, "d"}}, chunk.slice())

	assert.Nil(t, chunk.insert(6, "f"))
	assert.Equal(t, 4, chunk.len())
	assert.Equal(t, int64(2), chunk.min)
	assert.Equal(t, int64(6), chunk.max)
	assert.True(t, chunk.isFull())
	assert.Equal(t, []E{{2, "bb"}, {3, "c"}, {4, "d"}, {6, "f"}}, chunk.slice())

	chunk.delete(999)
	assert.Equal(t, []E{{2, "bb"}, {3, "c"}, {4, "d"}, {6, "f"}}, chunk.slice())

	newChunk := chunk.insert(5, "e")
	assert.NotNil(t, newChunk)
	assert.Equal(t, []E{{2, "bb"}, {3, "c"}}, chunk.slice())
	assert.Equal(t, 2, chunk.len())
	assert.Equal(t, int64(2), chunk.min)
	assert.Equal(t, int64(3), chunk.max)
	assert.Equal(t, []E{{4, "d"}, {5, "e"}, {6, "f"}}, newChunk.slice())
	assert.Equal(t, int64(4), newChunk.min)
	assert.Equal(t, int64(6), newChunk.max)

	assert.Nil(t, newChunk.insert(7, "g"))
	assert.Equal(t, []E{{4, "d"}, {5, "e"}, {6, "f"}, {7, "g"}}, newChunk.slice())

	newChunk2 := newChunk.insert(0, "xx")
	assert.NotNil(t, newChunk2)
	assert.Equal(t, []E{{0, "xx"}, {4, "d"}, {5, "e"}}, newChunk.slice())
	assert.Equal(t, int64(0), newChunk.min)
	assert.Equal(t, int64(5), newChunk.max)
	assert.Equal(t, []E{{6, "f"}, {7, "g"}}, newChunk2.slice())

	assert.Nil(t, newChunk.insert(8, "h"))
	assert.Equal(t, []E{{0, "xx"}, {4, "d"}, {5, "e"}, {8, "h"}}, newChunk.slice())
	newChunk.delete(0)
	assert.Equal(t, int64(4), newChunk.min)
	assert.Equal(t, int64(8), newChunk.max)
	assert.Equal(t, []E{{4, "d"}, {5, "e"}, {8, "h"}}, newChunk.slice())
	assert.Nil(t, newChunk.insert(9, "i"))
	assert.Equal(t, []E{{4, "d"}, {5, "e"}, {8, "h"}, {9, "i"}}, newChunk.slice())

	newChunk3 := newChunk.insert(10, "j")
	assert.NotNil(t, newChunk3)
	assert.Equal(t, []E{{4, "d"}, {5, "e"}}, newChunk.slice())
	assert.Equal(t, []E{{8, "h"}, {9, "i"}, {10, "j"}}, newChunk3.slice())
	assert.Equal(t, 2, newChunk.len())
	assert.Equal(t, int64(4), newChunk.min)
	assert.Equal(t, int64(5), newChunk.max)
	assert.Equal(t, 3, newChunk3.len())
	assert.Equal(t, int64(8), newChunk3.min)
	assert.Equal(t, int64(10), newChunk3.max)

	newChunk.delete(4)
	assert.Nil(t, newChunk.insert(4, "d"))
	assert.Equal(t, []E{{4, "d"}, {5, "e"}}, newChunk.slice())

	assert.Nil(t, newChunk.insert(1, "a"))
	assert.Nil(t, newChunk.insert(2, "b"))
	assert.Equal(t, []E{{1, "a"}, {2, "b"}, {4, "d"}, {5, "e"}}, newChunk.slice())
	newChunk4 := newChunk.insert(3, "c")
	assert.NotNil(t, newChunk4)
	assert.Equal(t, []E{{1, "a"}, {2, "b"}, {3, "c"}}, newChunk.slice())
	assert.Equal(t, []E{{4, "d"}, {5, "e"}}, newChunk4.slice())

	newChunk.delete(1)
	newChunk.delete(2)
	assert.Equal(t, []E{{3, "c"}}, newChunk.slice())
	assert.Equal(t, int64(3), newChunk.min)
	assert.Equal(t, int64(3), newChunk.max)
	assert.Equal(t, 1, newChunk.len())
	newChunk.delete(3)
	assert.Equal(t, []E{}, newChunk.slice())
	assert.Equal(t, 0, newChunk.len())
	assert.Equal(t, int64(0), newChunk.min)
	assert.Equal(t, int64(0), newChunk.max)

	newChunk.insert(1, "a")
	newChunk.insert(2, "b")
	newChunk.insert(3, "c")
	newChunk.insert(4, "d")
	assert.Equal(t, []E{{1, "a"}, {2, "b"}, {3, "c"}, {4, "d"}}, newChunk.slice())
	newChunk.delete(4)
	assert.Equal(t, []E{{1, "a"}, {2, "b"}, {3, "c"}}, newChunk.slice())
	assert.Equal(t, int64(1), newChunk.min)
	assert.Equal(t, int64(3), newChunk.max)
}

func TestChunkMerge(t *testing.T) {
	size := 6
	c1 := newChunk[int64, string](size)
	c2 := newChunk[int64, string](size)
	c1.insert(1, "a")
	c1.insert(2, "b")
	c2.insert(3, "c")
	c2.insert(4, "d")

	c1.merge(&c2)
	assert.Equal(t, []E{{1, "a"}, {2, "b"}, {3, "c"}, {4, "d"}}, c1.slice())
	assert.Equal(t, int64(1), c1.min)
	assert.Equal(t, int64(4), c1.max)
	assert.Equal(t, 4, c1.len())
}

func TestArrayMap(t *testing.T) {
	size := 4
	m := New[int64, string](size)

	m.Insert(1, "a")
	v, ok := m.Get(1)
	assert.True(t, ok)
	assert.Equal(t, "a", v)

	v, ok = m.Get(2)
	assert.False(t, ok)

	m.Insert(2, "b")
	entries := m.Iter().Collect()
	assert.Equal(t, []E{{1, "a"}, {2, "b"}}, entries)

	m.Insert(3, "c")
	m.Insert(4, "d")
	m.Insert(5, "e")
	assert.Equal(t, []E{{1, "a"}, {2, "b"}, {3, "c"}, {4, "d"}, {5, "e"}}, m.Iter().Collect())
	assert.Equal(t, 5, m.Len())

	m.Delete(2)
	assert.Equal(t, []E{{1, "a"}, {3, "c"}, {4, "d"}, {5, "e"}}, m.Iter().Collect())
	assert.Equal(t, 4, m.Len())
	assert.Equal(t, 2, len(m.chunks))

	m.Insert(99, "z")
	m.Insert(2, "b")
	assert.Equal(t, []E{{1, "a"}, {2, "b"}, {3, "c"}, {4, "d"}, {5, "e"}, {99, "z"}}, m.Iter().Collect())

	m.Insert(6, "f")
	m.Insert(7, "g")
	m.Insert(8, "h")
	m.Delete(99)
	m.Insert(9, "i")
	m.Insert(10, "j")
	m.Insert(11, "k")
	m.Insert(12, "l")
	m.Delete(1)
	assert.Equal(t,
		[]E{
			{2, "b"}, {3, "c"}, {4, "d"}, {5, "e"}, {6, "f"}, {7, "g"},
			{8, "h"}, {9, "i"}, {10, "j"}, {11, "k"}, {12, "l"},
		},
		m.Iter().Collect(),
	)

	m.Delete(4)
	m.Delete(5)
	m.Delete(6)
	m.Delete(7)
	m.Delete(8)
	m.Delete(9)
	assert.Equal(t,
		[]E{{2, "b"}, {3, "c"}, {10, "j"}, {11, "k"}, {12, "l"}},
		m.Iter().Collect(),
	)

	v, ok = m.Get(9)
	assert.False(t, ok)
	assert.False(t, m.Delete(9))

	m.Insert(5, "e")
	m.Insert(6, "f")
	m.Insert(7, "g")
	m.Insert(8, "h")
	assert.Equal(t,
		[]E{{2, "b"}, {3, "c"}, {5, "e"}, {6, "f"}, {7, "g"}, {8, "h"}, {10, "j"}, {11, "k"}, {12, "l"}},
		m.Iter().Collect(),
	)
	first, ok := m.First()
	assert.True(t, ok)
	assert.Equal(t, E{2, "b"}, first)

	empty := New[int64, string](size)
	assert.Equal(t, []E{}, empty.Iter().Collect())
	first, ok = empty.First()
	assert.False(t, ok)
}

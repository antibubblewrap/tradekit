package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunk(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}}, chunk(items, 2))
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}}, chunk(items, 5))
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}}, chunk(items, 33))
	assert.Equal(t, [][]int{{1}, {2}, {3}, {4}, {5}}, chunk(items, 1))
	assert.Equal(t, [][]int{{1, 2, 3}, {4, 5}}, chunk(items, 3))

	assert.Equal(t, [][]int{}, chunk([]int{}, 2))
}

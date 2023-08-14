package websocket

import (
	"bytes"
)

type bufferPool struct {
	buffers chan *bytes.Buffer
	size    int
	cap     int
}

// newBufferPool creates a new bufferPool with a given size. The size determines the
// maximum number of buffers that the pool can hold and bufCap defines the initial
// capacity of each buffer.
func newBufferPool(size int, bufCap int) *bufferPool {
	buffers := make(chan *bytes.Buffer, size)
	return &bufferPool{buffers: buffers, size: size, cap: bufCap}
}

// get gets a buffer from the pool, or creates a new buffer if the pool is currently
// empty.
func (bp *bufferPool) get() *bytes.Buffer {
	select {
	case buf := <-bp.buffers:
		return buf
	default:
		return bytes.NewBuffer(make([]byte, 0, bp.cap))
	}
}

// put releases a buffer back to the pool.
func (bp *bufferPool) put(buf *bytes.Buffer) {
	buf.Reset()
	if cap(buf.Bytes()) > bp.cap {
		buf = bytes.NewBuffer(make([]byte, 0, bp.cap))
	}

	select {
	case bp.buffers <- buf:
	default:
		// Discard the buffer if the pool is full
	}
}

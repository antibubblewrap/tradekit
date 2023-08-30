package tradekit

import (
	"time"
)

// StreamOptions is used to specify parameters for the underlying websocket connection.
type StreamOptions struct {
	// Whether compression should be enabled on the websocket. Defaults to false.
	EnableCompression bool

	// PingInterval is the interval to send ping messages on the websocket. Defaults to 15s
	PingInterval time.Duration

	// ResetInterval, if set, will reset the websocket connection on a given interval.
	ResetInterval time.Duration

	// BufferPoolSize sets the number of reusable buffers for a websocket connection.
	// Defaults to 32.
	BufferPoolSize int

	// BufferCapacity sets the capacity of each reusable buffer in the the websocket's
	// buffer pool. Defaults to 2048.
	BufferCapacity int
}

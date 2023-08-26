package tradekit

import "time"

// StreamOptions is used to specify parameters for the underlying websocket connection.
type StreamOptions struct {
	// PingInterval is the interval to send ping messages on the websocket. If not
	// specified, it defaults to 15 seconds.
	PingInterval time.Duration

	// ResetInterval, if set, will reset the websocket connection on a given interval.
	ResetInterval time.Duration

	// BufferPoolSize sets the number of reusable buffers for a websocket connection.
	// If not specified, it defaults to 32.
	BufferPoolSize uint

	// BufferCapacity sets the capacity of each reusable buffer in the the websocket's
	// buffer pool. If not specified, it defaults to 2048.
	BufferCapacity uint
}

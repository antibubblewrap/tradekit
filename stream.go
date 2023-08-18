package tradekit

import "time"

// StreamOptions is used to specify parameters for the underlying websocket connection for
// a Bybit or Deribit stream.
type StreamOptions struct {
	// ResetInterval, if set, will reset the websocket connection on a given interval.
	ResetInterval time.Duration
}

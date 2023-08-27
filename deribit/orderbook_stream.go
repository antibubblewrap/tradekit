package deribit

import (
	"fmt"
)

// OrderbookSub represents a request to subscribe to a [NewOrderbookStream] channel.
// For details see:
//   - https://docs.deribit.com/#book-instrument_name-interval
type OrderbookSub struct {
	Instrument string

	// Optional message frequency. Defaults to "raw" if not set.
	Freq UpdateFrequency
}

func (sub OrderbookSub) channel() string {
	if sub.Freq == "" {
		return fmt.Sprintf("book.%s.raw", sub.Instrument)
	} else {
		return fmt.Sprintf("book.%s.%s", sub.Instrument, sub.Freq)
	}
}

// NewOrderbookStream creates a new [Stream] which produces a stream of incremental
// orderbook updates. For a stream of orderbook depth snapshots, see [NewOrderbookDepthStream]
// For details see:
//   - https://docs.deribit.com/#book-instrument_name-interval
func NewOrderbookStream(wsUrl string, subscriptions ...OrderbookSub) Stream[OrderbookUpdate, OrderbookSub] {
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	p := streamParams[OrderbookUpdate, OrderbookSub]{
		name:         "OrderbookStream",
		wsUrl:        wsUrl,
		isPrivate:    false,
		parseMessage: parseOrderbookUpdate,
		subs:         subscriptions,
	}
	s := newStream[OrderbookUpdate](p)
	return s
}

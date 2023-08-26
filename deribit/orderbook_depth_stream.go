package deribit

import (
	"context"
	"fmt"

	"github.com/antibubblewrap/tradekit"
)

// OrderbookDepthSub represents a subscription to a deribit orderbook depth stream created
// using [NewOrderbookDepthStream]. The update interval is automatically set to 100ms.
// For details see:
//   - https://docs.deribit.com/#book-instrument_name-group-depth-interval
type OrderbookDepthSub struct {
	Instrument string
	Depth      int

	// Optional price grouping. A value of 0 means no grouping. Defaults to no grouping.
	Group int
}

func (sub OrderbookDepthSub) channel() string {
	if sub.Group == 0 {
		return fmt.Sprintf("book.%s.none.%d.100ms", sub.Instrument, sub.Depth)
	} else {
		return fmt.Sprintf("book.%s.%d.%d.100ms", sub.Instrument, sub.Group, sub.Depth)
	}
}

type bookDepthStream struct {
	s *stream[OrderbookDepth]
}

// NewOrderbookDepthStream creates a new [Stream] which produces a stream of orderbook
// depth snapshots. For a realtime stream of incremental orderbook updates, see
// [NewOrderbookStream]. For details see:
//   - https://docs.deribit.com/#book-instrument_name-group-depth-interval
func NewOrderbookDepthStream(wsUrl string, subscriptions ...OrderbookDepthSub) Stream[OrderbookDepth, OrderbookDepthSub] {
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	p := streamParams[OrderbookDepth]{
		name:         "OrderbookDepthStream",
		wsUrl:        wsUrl,
		isPrivate:    false,
		parseMessage: parseOrderbookDepth,
		subs:         subs,
	}
	s := newStream[OrderbookDepth](p)
	return &bookDepthStream{s: s}
}

func (s *bookDepthStream) Start(ctx context.Context) error {
	return s.s.Start(ctx)
}

func (s *bookDepthStream) SetCredentials(c *Credentials) {
	s.s.SetCredentials(c)
}

func (s *bookDepthStream) SetStreamOptions(opts *tradekit.StreamOptions) {
	s.s.SetStreamOptions(opts)
}

func (s *bookDepthStream) Subscribe(subscriptions ...OrderbookDepthSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *bookDepthStream) Unsubscribe(subscriptions ...OrderbookDepthSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *bookDepthStream) Err() <-chan error {
	return s.s.Err()
}

func (s *bookDepthStream) Messages() <-chan OrderbookDepth {
	return s.s.Messages()
}

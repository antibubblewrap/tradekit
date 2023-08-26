package deribit

import (
	"context"
	"fmt"

	"github.com/antibubblewrap/tradekit"
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

type orderbookStream struct {
	s *stream[OrderbookUpdate]
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
	p := streamParams[OrderbookUpdate]{
		name:         "OrderbookStream",
		wsUrl:        wsUrl,
		isPrivate:    false,
		parseMessage: parseOrderbookUpdate,
		subs:         subs,
	}
	s := newStream[OrderbookUpdate](p)
	return &orderbookStream{s: s}
}

func (s *orderbookStream) Start(ctx context.Context) error {
	return s.s.Start(ctx)
}

func (s *orderbookStream) SetCredentials(c *Credentials) {
	s.s.SetCredentials(c)
}

func (s *orderbookStream) SetStreamOptions(opts *tradekit.StreamOptions) {
	s.s.SetStreamOptions(opts)
}

func (s *orderbookStream) Subscribe(subscriptions ...OrderbookSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *orderbookStream) Unsubscribe(subscriptions ...OrderbookSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *orderbookStream) Err() <-chan error {
	return s.s.Err()
}

func (s *orderbookStream) Messages() <-chan OrderbookUpdate {
	return s.s.Messages()
}

package deribit

import (
	"context"
	"fmt"

	"github.com/antibubblewrap/tradekit"
)

// TradesSub represents a request to subscribe to a [NewTradeStream] channel. Either
// Instrument or Currency & Kind must be specified. If Instrument is specified then
// Currency and Kind are ignored. For details see:
//   - https://docs.deribit.com/#trades-instrument_name-interval
//   - https://docs.deribit.com/#trades-kind-currency-interval
type TradesSub struct {
	Instrument string

	Currency string
	Kind     InstrumentKind

	// Optional message frequency. Defaults to "raw" if not set.
	Interval UpdateFrequency
}

func (s TradesSub) channel() string {
	if s.Instrument != "" {
		if s.Interval == "" {
			return fmt.Sprintf("trades.%s.raw", s.Instrument)
		}
		return fmt.Sprintf("trades.%s.%s", s.Instrument, s.Interval)
	} else {
		if s.Interval == "" {
			return fmt.Sprintf("trades.%s.%s.raw", s.Kind, s.Currency)
		}
		return fmt.Sprintf("trades.%s.%s.%s", s.Kind, s.Currency, s.Interval)
	}
}

type tradesStream struct {
	s *stream[[]PublicTrade]
}

// NewTradesStream creates a new [Stream] which produces a stream of public trades.
func NewTradesStream(wsUrl string, subscriptions ...TradesSub) Stream[[]PublicTrade, TradesSub] {
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	p := streamParams[[]PublicTrade]{
		name:         "TradesStream",
		wsUrl:        wsUrl,
		isPrivate:    false,
		parseMessage: parsePublicTrades,
		subs:         subs,
	}
	s := newStream[[]PublicTrade](p)
	return &tradesStream{s: s}
}

func (s *tradesStream) SetStreamOptions(opts *tradekit.StreamOptions) {
	s.s.SetStreamOptions(opts)
}

func (s *tradesStream) SetCredentials(c *Credentials) {
	s.s.SetCredentials(c)
}

func (s *tradesStream) Start(ctx context.Context) error {
	return s.s.Start(ctx)
}

func (s *tradesStream) Subscribe(subscriptions ...TradesSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *tradesStream) Unsubscribe(subscriptions ...TradesSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *tradesStream) Err() <-chan error {
	return s.s.Err()
}

func (s *tradesStream) Messages() <-chan []PublicTrade {
	return s.s.Messages()
}

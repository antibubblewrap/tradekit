package deribit

import (
	"context"
	"fmt"

	"github.com/antibubblewrap/tradekit"
)

// UserTradesSub defines a subscription channel for a user trades stream created using
// [NewUserTradesStream]. Either Instrument or Kind & Currency must be specified.
type UserTradesSub struct {
	Instrument string

	Kind     InstrumentKind
	Currency string
}

func (s UserTradesSub) channel() string {
	if s.Instrument != "" {
		return fmt.Sprintf("user.trades.%s.raw", s.Instrument)
	} else {
		return fmt.Sprintf("user.trades.%s.%s.raw", s.Kind, s.Currency)
	}
}

type userTradesStream struct {
	s *stream[[]TradeExecution]
}

// NewUserTradesStream creates a new [Stream] returning user trade executions. Deribit
// credentials are required for this stream. For details see:
//   - https://docs.deribit.com/#user-trades-kind-currency-interval
//   - https://docs.deribit.com/#user-trades-instrument_name-interval
func NewUserTradesStream(wsUrl string, c Credentials, subscriptions ...UserOrdersSub) Stream[[]TradeExecution, UserTradesSub] {
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	p := streamParams[[]TradeExecution]{
		name:         "UserTradesStream",
		wsUrl:        wsUrl,
		isPrivate:    true,
		parseMessage: parseTradeExecutions,
		subs:         subs,
	}
	s := newStream[[]TradeExecution](p)
	s.SetCredentials(&c)
	return &userTradesStream{s: s}

}

func (s *userTradesStream) SetCredentials(c *Credentials) {
	s.s.SetCredentials(c)
}

func (s *userTradesStream) SetStreamOptions(opts *tradekit.StreamOptions) {
	s.s.SetStreamOptions(opts)
}

func (s *userTradesStream) Start(ctx context.Context) error {
	return s.s.Start(ctx)
}

func (s *userTradesStream) Subscribe(subscriptions ...UserTradesSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *userTradesStream) Unsubscribe(subscriptions ...UserTradesSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *userTradesStream) Err() <-chan error {
	return s.s.Err()
}

func (s *userTradesStream) Messages() <-chan []TradeExecution {
	return s.s.Messages()
}

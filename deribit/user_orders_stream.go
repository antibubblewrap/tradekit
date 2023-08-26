package deribit

import (
	"context"
	"fmt"

	"github.com/antibubblewrap/tradekit"
)

// UserOrdersSub defines a subscription channel for a user ordders stream created using
// [NewUserOrdersStream]. Either Instrument or Kind & Currency must be specified.
type UserOrdersSub struct {
	Instrument string

	Currency string
	Kind     InstrumentKind
}

func (s UserOrdersSub) channel() string {
	if s.Instrument != "" {
		return fmt.Sprintf("user.orders.%s.raw", s.Instrument)
	} else {
		return fmt.Sprintf("user.orders.%s.%s.raw", s.Kind, s.Currency)
	}
}

type userOrdersStream struct {
	s *stream[Order]
}

// NewUserOrdersStream creates a new [Stream] returning updates to order's in a user's
// account. Credentials are required for this stream. For details see:
//   - https://docs.deribit.com/#user-orders-instrument_name-raw
//   - https://docs.deribit.com/#user-orders-kind-currency-interval
func NewUserOrdersStream(wsUrl string, c Credentials, subscriptions ...UserOrdersSub) Stream[Order, UserOrdersSub] {
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	p := streamParams[Order]{
		name:         "UserOrdersStream",
		wsUrl:        wsUrl,
		isPrivate:    true,
		parseMessage: parseOrder,
		subs:         subs,
	}
	s := newStream[Order](p)
	s.SetCredentials(&c)
	return &userOrdersStream{s: s}

}

func (s *userOrdersStream) SetStreamOptions(opts *tradekit.StreamOptions) {
	s.s.SetStreamOptions(opts)
}

func (s *userOrdersStream) SetCredentials(c *Credentials) {
	s.s.SetCredentials(c)
}

func (s *userOrdersStream) Start(ctx context.Context) error {
	return s.s.Start(ctx)
}

func (s *userOrdersStream) Subscribe(subscriptions ...UserOrdersSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *userOrdersStream) Unsubscribe(subscriptions ...UserOrdersSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *userOrdersStream) Err() <-chan error {
	return s.s.Err()
}

func (s *userOrdersStream) Messages() <-chan Order {
	return s.s.Messages()
}

package deribit

import (
	"fmt"
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

// NewUserOrdersStream creates a new [Stream] returning updates to order's in a user's
// account. Credentials are required for this stream. For details see:
//   - https://docs.deribit.com/#user-orders-instrument_name-raw
//   - https://docs.deribit.com/#user-orders-kind-currency-interval
func NewUserOrdersStream(wsUrl string, c Credentials, subscriptions ...UserOrdersSub) Stream[Order, UserOrdersSub] {
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	p := streamParams[Order, UserOrdersSub]{
		name:         "UserOrdersStream",
		wsUrl:        wsUrl,
		isPrivate:    true,
		parseMessage: parseOrder,
		subs:         subscriptions,
	}
	s := newStream[Order](p)
	s.SetCredentials(&c)
	return s
}

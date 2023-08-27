package deribit

import (
	"fmt"
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

// NewUserTradesStream creates a new [Stream] returning user trade executions. Deribit
// credentials are required for this stream. For details see:
//   - https://docs.deribit.com/#user-trades-kind-currency-interval
//   - https://docs.deribit.com/#user-trades-instrument_name-interval
func NewUserTradesStream(wsUrl string, c Credentials, subscriptions ...UserTradesSub) Stream[[]TradeExecution, UserTradesSub] {
	p := streamParams[[]TradeExecution, UserTradesSub]{
		name:         "UserTradesStream",
		wsUrl:        wsUrl,
		isPrivate:    true,
		parseMessage: parseTradeExecutions,
		subs:         subscriptions,
	}
	s := newStream[[]TradeExecution](p)
	s.SetCredentials(&c)
	return s
}

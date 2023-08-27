package deribit

import (
	"fmt"
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

// NewTradesStream creates a new [Stream] which produces a stream of public trades.
func NewTradesStream(wsUrl string, subscriptions ...TradesSub) Stream[[]PublicTrade, TradesSub] {
	p := streamParams[[]PublicTrade, TradesSub]{
		name:         "TradesStream",
		wsUrl:        wsUrl,
		isPrivate:    false,
		parseMessage: parsePublicTrades,
		subs:         subscriptions,
	}
	return newStream[[]PublicTrade](p)
}

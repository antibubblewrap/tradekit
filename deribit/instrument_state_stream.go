package deribit

import (
	"fmt"
)

// InstrumentStateSub represents a subscription to a [NewInstrumentStateStream].
// For details see:
//   - https://docs.deribit.com/#instrument-state-kind-currency
type InstrumentStateSub struct {
	Kind     InstrumentKind
	Currency string
}

func (sub InstrumentStateSub) channel() string {
	return fmt.Sprintf("instrument.state.%s.%s", sub.Kind, sub.Currency)
}

// NewInstrumentStateStream creates a new [Stream] which produces a stream of notifications
// regarding new or terminated instruments.
//   - https://docs.deribit.com/#instrument-state-kind-currency
func NewInstrumentStateStream(wsUrl string, subscriptions ...InstrumentStateSub) Stream[InstrumentState, InstrumentStateSub] {
	p := streamParams[InstrumentState, InstrumentStateSub]{
		name:         "InstrumentStateStream",
		wsUrl:        wsUrl,
		isPrivate:    false,
		parseMessage: parseInstrumentState,
		subs:         subscriptions,
	}
	s := newStream[InstrumentState](p)
	return s
}

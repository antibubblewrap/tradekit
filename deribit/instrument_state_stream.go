package deribit

import (
	"context"
	"fmt"

	"github.com/antibubblewrap/tradekit"
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

type stateStream struct {
	s *stream[InstrumentState]
}

// NewInstrumentStateStream creates a new [Stream] which produces a stream of notifications
// regarding new or terminated instruments.
//   - https://docs.deribit.com/#instrument-state-kind-currency
func NewInstrumentStateStream(wsUrl string, subscriptions ...InstrumentStateSub) Stream[InstrumentState, InstrumentStateSub] {
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	p := streamParams[InstrumentState]{
		name:         "InstrumentStateStream",
		wsUrl:        wsUrl,
		isPrivate:    false,
		parseMessage: parseInstrumentState,
		subs:         subs,
	}
	s := newStream[InstrumentState](p)
	return &stateStream{s: s}
}

func (s *stateStream) Start(ctx context.Context) error {
	return s.s.Start(ctx)
}

func (s *stateStream) SetCredentials(c *Credentials) {
	s.s.SetCredentials(c)
}

func (s *stateStream) SetStreamOptions(opts *tradekit.StreamOptions) {
	s.s.SetStreamOptions(opts)
}

func (s *stateStream) Subscribe(subscriptions ...InstrumentStateSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *stateStream) Unsubscribe(subscriptions ...InstrumentStateSub) {
	if len(subscriptions) == 0 {
		return
	}
	subs := make([]subscription, len(subscriptions))
	for i, sub := range subscriptions {
		subs[i] = sub
	}
	s.s.Subscribe(subs...)
}

func (s *stateStream) Err() <-chan error {
	return s.s.Err()
}

func (s *stateStream) Messages() <-chan InstrumentState {
	return s.s.Messages()
}

package deribit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

// InstrumentStateStream streams messages from the instrument.state.{kind}.{currency}
// Deribit channel.
type InstrumentStateStream struct {
	ws   *websocket.Websocket
	msgs chan InstrumentState
	errc chan error
	p    fastjson.Parser
}

// InstrumentState is the type of message produced by the InstrumentStateStream.
// For details, see: https://docs.deribit.com/#instrument-state-kind-currency
type InstrumentState struct {
	Timestamp  int64  `json:"timestamp"`
	State      string `json:"state"`
	Instrument string `json:"instrument_name"`
}

type InstrumentStateSub struct {
	Kind     InstrumentKind
	Currency string
}

func (sub InstrumentStateSub) channel() string {
	return fmt.Sprintf("instrument.state.%s.%s", sub.Kind, sub.Currency)
}

// NewInstrumentStateStream creates a new InstrumentStateStream to the channel specified
// by the given InstrumentKind and Currency.
func NewInstrumentStateStream(t ConnectionType, subs ...InstrumentStateSub) (*InstrumentStateStream, error) {
	url, err := wsUrl(t)
	if err != nil {
		return nil, err
	}

	ws := websocket.New(url, nil)
	ws.PingInterval = 15 * time.Second
	ws.OnConnect = func() error {
		id := genId()
		channels := make([]string, len(subs))
		for i, sub := range subs {
			channels[i] = sub.channel()
		}
		subMsg, err := rpcSubscribeMsg(id, channels)
		if err != nil {
			return err
		}
		ws.Send(subMsg)
		return nil
	}

	msgs := make(chan InstrumentState)
	errc := make(chan error)

	return &InstrumentStateStream{ws: &ws, msgs: msgs, errc: errc}, nil
}

func (s *InstrumentStateStream) parseInstrumentStateMsg(msg websocket.Message) (InstrumentState, error) {
	defer msg.Release()

	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return InstrumentState{}, fmt.Errorf("invalid Deribit instrument state message: %s", string(msg.Data()))
	}

	method := v.GetStringBytes("method")
	if len(method) == 0 {
		return InstrumentState{}, nil
	}

	params := v.Get("params")
	channel := string(params.GetStringBytes("channel"))
	if !strings.HasPrefix(channel, "instrument.state.") {
		return InstrumentState{}, fmt.Errorf("invalid Deribit instrument state message: %s", string(msg.Data()))
	}

	v = params.Get("data")
	return InstrumentState{
		Timestamp:  v.GetInt64("timestamp"),
		State:      string(v.GetStringBytes("state")),
		Instrument: string(v.GetStringBytes("instrument_name")),
	}, nil
}

// Start the connection the the Deribit instrument state stream websocket. You must start a
// stream before any messages can be received. The connection will be automatically
// retried on failure with exponential backoff.
func (s *InstrumentStateStream) Start(ctx context.Context) error {
	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to Deribit InstrumentStateStream websocket: %w", err)
	}

	go func() {
		defer close(s.msgs)
		defer s.ws.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-s.ws.Messages():
				msg, err := s.parseInstrumentStateMsg(data)
				if err != nil {
					s.errc <- err
					return
				}
				if msg.State != "" {
					s.msgs <- msg
				}
			case err := <-s.ws.Err():
				s.errc <- err
				return
			}
		}
	}()

	return nil
}

// Messages returns a channel producing messages received from the instrument state
// stream. It should be read concurrently with the Err stream.
func (s *InstrumentStateStream) Messages() <-chan InstrumentState {
	return s.msgs
}

// The error stream should be read concurrently with the messages stream. If
// this channel produces an error, the messages stream will be closed.
func (s *InstrumentStateStream) Err() <-chan error {
	return s.errc
}

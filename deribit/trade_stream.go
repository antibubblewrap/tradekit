package deribit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

type TradeStream struct {
	ws   *websocket.Websocket
	msgs chan []TradeMsg
	errc chan error
	p    fastjson.Parser
}

// TradeMsg is the message type streamed from the Deribit trade channel.
// See https://docs.deribit.com/#trades-instrument_name-interval for more info.
type TradeMsg struct {
	Instrument    string  `json:"instrument_name"`
	TradeSeq      int64   `json:"trade_seq"`
	Timestamp     int64   `json:"timestamp"`
	TickDirection int64   `json:"tick_direction"`
	Price         float64 `json:"price"`
	Amount        float64 `json:"amount"`
	Direction     string  `json:"direction"`
	IndexPrice    float64 `json:"index_price"`
	MarkPrice     float64 `json:"mark_price"`
}

func NewTradeStream(t ConnectionType, instrument string, freq UpdateFrequency) (*TradeStream, error) {
	url, err := wsUrl(t)
	if err != nil {
		return nil, err
	}

	ws := websocket.New(url, nil)
	ws.PingInterval = 15 * time.Second
	ws.OnConnect = func() error {
		subMsg, err := newSubscribeMsg([]string{fmt.Sprintf("trades.%s.%s", instrument, freq)})
		if err != nil {
			return err
		}
		ws.Send(subMsg)
		return nil
	}

	msgs := make(chan []TradeMsg)
	errc := make(chan error)

	return &TradeStream{ws: &ws, msgs: msgs, errc: errc}, nil
}

func parseTrade(v *fastjson.Value) TradeMsg {
	return TradeMsg{
		Instrument:    string(v.GetStringBytes("instrument_name")),
		TradeSeq:      v.GetInt64("trade_seq"),
		Timestamp:     v.GetInt64("timestamp"),
		TickDirection: v.GetInt64("tick_direction"),
		Price:         v.GetFloat64("price"),
		Amount:        v.GetFloat64("amount"),
		Direction:     string(v.GetStringBytes("direction")),
		IndexPrice:    v.GetFloat64("index_price"),
		MarkPrice:     v.GetFloat64("mark_price"),
	}
}

// Parse an trade message received from the websocket stream. If the message is a
// subscription response, it returns an nil slice of TradeMsg and a nil error.
func (s *TradeStream) parseTradeMsg(msg websocket.Message) ([]TradeMsg, error) {
	defer msg.Release()
	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return nil, fmt.Errorf("invalid Deribit trade message: %s", string(msg.Data()))
	}

	method := v.GetStringBytes("method")
	if len(method) == 0 {
		return nil, nil
	}

	params := v.Get("params")
	channel := string(params.GetStringBytes("channel"))
	if !strings.HasPrefix(channel, "trades.") {
		return nil, fmt.Errorf("invalid Deribit trade message: %s", string(msg.Data()))
	}

	items := params.GetArray("data")
	trades := make([]TradeMsg, len(items))
	for i, v := range items {
		trades[i] = parseTrade(v)
	}
	return trades, nil
}

// Start the connection the the Deribit trade stream websocket. You must start a
// stream before any messages can be received. The connection will be automatically
// retried on failure with exponential backoff.
func (s *TradeStream) Start(ctx context.Context) error {
	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to Deribit TradeStream websocket: %w", err)
	}

	go func() {
		defer close(s.msgs)
		for {
			select {
			case data := <-s.ws.Messages():
				msg, err := s.parseTradeMsg(data)
				if err != nil {
					s.errc <- err
					return
				}
				if msg != nil {
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

// Messages returns a channel producing messages received from the trade stream. It
// should be read concurrently with the Err stream.
func (s *TradeStream) Messages() <-chan []TradeMsg {
	return s.msgs
}

// The error stream should be read concurrently with the  messages stream. If
// this channel produces an error, the messages stream will be closed.
func (s *TradeStream) Err() <-chan error {
	return s.errc
}

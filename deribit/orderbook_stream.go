package deribit

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

type OrderbookStream struct {
	ws   *websocket.Websocket
	msgs chan OrderbookMsg
	errc chan error
	p    fastjson.Parser
}

func NewOrderbookStream(t ConnectionType, instrument string, freq UpdateFrequency) (*OrderbookStream, error) {
	url, err := wsUrl(t)
	if err != nil {
		return nil, err
	}

	ws := websocket.New(url, nil)
	ws.PingInterval = 15 * time.Second
	ws.OnConnect = func() error {
		subMsg, err := newSubscribeMsg([]string{fmt.Sprintf("book.%s.%s", instrument, freq)})
		if err != nil {
			return err
		}
		ws.Send(subMsg)
		return nil
	}

	msgs := make(chan OrderbookMsg)
	errc := make(chan error)

	return &OrderbookStream{ws: &ws, msgs: msgs, errc: errc}, nil
}

// OrderbookMsg is the message type streamed from the Deribit book channel.
// For more info see: https://docs.deribit.com/#book-instrument_name-interval
type OrderbookMsg struct {
	// The type of orderbook update. Either "snapshot" or "change".
	Type         string
	Timestamp    int64
	Instrument   string `json:"instrument_name"`
	ChangeID     int64
	PrevChangeID int64
	Bids         []tradekit.Level
	Asks         []tradekit.Level
}

func parseOrderbookLevel(v *fastjson.Value) tradekit.Level {
	action := v.GetStringBytes("0")
	price := v.GetFloat64("1")
	amount := v.GetFloat64("2")
	if bytes.Equal(action, []byte("delete")) {
		amount = 0
	}
	return tradekit.Level{Price: price, Amount: amount}
}

func parseOrderbookLevels(items []*fastjson.Value) []tradekit.Level {
	levels := make([]tradekit.Level, len(items))
	for i, item := range items {
		levels[i] = parseOrderbookLevel(item)
	}
	return levels
}

// Parse an orderbook message received from the websocket stream. If the message is a
// subscription response, it returns an empty OrderbookMsg and a nil error.
func (s *OrderbookStream) parseOrderbookMsg(msg websocket.Message) (OrderbookMsg, error) {
	defer msg.Release()
	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return OrderbookMsg{}, fmt.Errorf("invalid Deribit orderbook message: %s", string(msg.Data()))
	}

	method := v.GetStringBytes("method")
	if len(method) == 0 {
		return OrderbookMsg{}, nil
	}

	params := v.Get("params")
	channel := string(params.GetStringBytes("channel"))
	if !strings.HasPrefix(channel, "book.") {
		return OrderbookMsg{}, fmt.Errorf("invalid Deribit orderbook message: %s", string(msg.Data()))
	}

	v = params.Get("data")
	return OrderbookMsg{
		Type:         string(v.GetStringBytes("type")),
		Timestamp:    v.GetInt64("timestamp"),
		Instrument:   string(v.GetStringBytes("instrument_name")),
		ChangeID:     v.GetInt64("change_id"),
		PrevChangeID: v.GetInt64("prev_change_id"),
		Bids:         parseOrderbookLevels(v.GetArray("bids")),
		Asks:         parseOrderbookLevels(v.GetArray("asks")),
	}, nil
}

// Start the connection the the Deribit orderbook stream websocket. You must start a
// stream before any messages can be received. The connection will be automatically
// retried on failure with exponential backoff.
func (s *OrderbookStream) Start(ctx context.Context) error {
	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to Deribit OrderbookStream websocket: %w", err)
	}

	go func() {
		defer close(s.msgs)
		for {
			select {
			case data := <-s.ws.Messages():
				msg, err := s.parseOrderbookMsg(data)
				if err != nil {
					s.errc <- err
					return
				}
				if msg.Type != "" {
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

// Messages returns a channel producing messages received from the orderbook stream. It
// should be read concurrently with the Err stream.
func (s *OrderbookStream) Messages() <-chan OrderbookMsg {
	return s.msgs
}

// The error stream should be read concurrently with the  messages stream. If
// this channel produces an error, the messages stream will be closed.
func (s *OrderbookStream) Err() <-chan error {
	return s.errc
}

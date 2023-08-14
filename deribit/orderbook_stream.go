package deribit

import (
	"context"
	"fmt"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

type OrderbookStream struct {
	ws   *websocket.Websocket
	msgs chan OrderbookMsg
	errc chan error
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
	Type         string
	Timestamp    int64
	Instrument   string
	ChangeID     int64
	PrevChangeID int64
	Bids         []tradekit.Level
	Asks         []tradekit.Level
}

func parseOrderbookLevel(v *fastjson.Value) tradekit.Level {
	action := v.GetStringBytes("0")
	price := v.GetFloat64("1")
	amount := v.GetFloat64("2")
	if string(action) == "delete" {
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

func parseOrderbookMsg(data []byte) (OrderbookMsg, error) {
	var p fastjson.Parser
	v, err := p.ParseBytes(data)
	if err != nil {
		return OrderbookMsg{}, err
	}

	updateType := v.GetStringBytes("type")
	timestamp := v.GetInt64("timestamp")
	instrument := v.GetStringBytes("instrument_name")
	changeId := v.GetInt64("change_id")
	prevChangeId := v.GetInt64("prev_change_id")

	bids := parseOrderbookLevels(v.GetArray("bids"))
	asks := parseOrderbookLevels(v.GetArray("asks"))

	return OrderbookMsg{
		Type:         string(updateType),
		Timestamp:    timestamp,
		Instrument:   string(instrument),
		ChangeID:     changeId,
		PrevChangeID: prevChangeId,
		Bids:         bids,
		Asks:         asks,
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
				subMsg, err := parseSubscriptionMsg(data.Data())
				if err != nil {
					s.errc <- err
					data.Release()
					return
				}
				if subMsg.Method != "" {
					msg, err := parseOrderbookMsg(subMsg.Params.Data)
					if err != nil {
						s.errc <- fmt.Errorf("deserializing Deribit order book message: %w (%s)", err, string(data.Data()))
						data.Release()
						return
					}
					data.Release()
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

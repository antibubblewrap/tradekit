package deribit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
)

type TradeStream struct {
	ws   *websocket.Websocket
	msgs chan []TradeMsg
	errc chan error
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
				subMsg, err := parseSubscriptionMsg(data.Data())
				if err != nil {
					s.errc <- err
					data.Release()
					return
				}
				if subMsg.Method != "" {
					var msg []TradeMsg
					if err := json.Unmarshal(subMsg.Params.Data, &msg); err != nil {
						s.errc <- fmt.Errorf("deserializing Deribit trade message: %w (%s)", err, string(data.Data()))
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

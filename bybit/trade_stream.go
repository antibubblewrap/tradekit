package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
)

type TradeMsgData struct {
	FilledTimestamp int64   `json:"T"`
	Symbol          string  `json:"s"`
	Side            string  `json:"S"`
	Amount          float64 `json:"v,string"`
	Price           float64 `json:"p,string"`
}

type TradeMsg struct {
	Topic     string         `json:"topic"`
	Timestamp int64          `json:"ts"`
	Data      []TradeMsgData `json:"data"`
}

type TradeStream struct {
	ws   *websocket.Websocket
	msgs chan TradeMsg
	errc chan error
}

func NewTradeStream(market Market, symbol string) (*TradeStream, error) {
	url, err := channelUrl(market)
	if err != nil {
		return nil, fmt.Errorf("creating Bybit TradeStream: %w", err)
	}

	ws := websocket.New(url, nil)
	ws.PingInterval = 15 * time.Second
	ws.OnConnect = func() error {
		channel := fmt.Sprintf("publicTrade.%s", symbol)
		msg := make(map[string]any)
		msg["op"] = "subscribe"
		msg["args"] = []string{channel}
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		ws.Send(data)
		return nil
	}

	msgs := make(chan TradeMsg)
	errc := make(chan error)

	return &TradeStream{ws: &ws, msgs: msgs, errc: errc}, nil
}

// Start the connection the the trade stream websocket. You must start a stream before
// any messages can be received. It reconnects automatically on failure with exponential
// backoff.
func (s *TradeStream) Start(ctx context.Context) error {
	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to trade stream websocket: %w", err)
	}

	go func() {
		defer close(s.msgs)
		defer s.ws.Close()
		ticker := time.NewTicker(20 * time.Second)
		for {
			select {
			case data := <-s.ws.Messages():
				var msg TradeMsg
				if err := json.Unmarshal(data.Data(), &msg); err != nil {
					data.Release()
					s.errc <- fmt.Errorf("deserializing Bybit trade message: %w (%s)", err, string(data.Data()))
					return
				}
				if msg.Topic == "" {
					if !isPingResponseMsg(data.Data()) {
						// It should be the subscription response
						if err := checkSubscribeResponse(data.Data()); err != nil {
							data.Release()
							s.errc <- err
							return
						}
					}
					data.Release()
				} else {
					data.Release()
					s.msgs <- msg
				}
			case <-ticker.C:
				s.ws.Send(heartbeatMsg())
			case err := <-s.ws.Err():
				s.errc <- err
				return
			}
		}
	}()

	return nil
}

// The error stream should be read concurrently with the orderbook messages stream. If
// this channel produces an error, the messages stream will be closed.
func (s *TradeStream) Err() <-chan error {
	return s.errc
}

// Messages returns a channel producing messages received from the trades stream. It
// should be read concurrently with the Err stream.
func (s *TradeStream) Messages() <-chan TradeMsg {
	return s.msgs
}

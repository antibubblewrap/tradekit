package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/internal/websocket"
)

type OrderbookStream struct {
	ws   *websocket.Websocket
	msgs chan OrderbookMsg
	errc chan error
}

// Create a new stream to the Bybit orderbook websocket stream for the given symbol and
// market depth.
func NewOrderbookStream(market Market, depth int, symbol string) (*OrderbookStream, error) {
	url, err := channelUrl(market)
	if err != nil {
		return nil, fmt.Errorf("creating Bybit OrderbookStream: %w", err)
	}

	ws := websocket.New(url, nil)
	ws.PingInterval = 15 * time.Second
	ws.OnConnect = func() error {
		channel := fmt.Sprintf("orderbook.%d.%s", depth, symbol)
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

	msgs := make(chan OrderbookMsg)
	errc := make(chan error)

	return &OrderbookStream{ws: &ws, msgs: msgs, errc: errc}, nil
}

type OrderbookMsgData struct {
	// Symbol name
	Symbol string `json:"s"`
	// Bids (price, amount) levels.
	Bids []tradekit.Level `json:"b"`
	// Asks (price, amount) levels.
	Asks []tradekit.Level `json:"a"`
	// Update Id. Increment sequentially. Snapshots have update ID 1.
	UpdateId int64 `json:"u"`
	// Cross sequence.
	Seq int64 `json:"seq"`
}

// OrderbookMsg is of the messages produced by the OrderbookStream.
type OrderbookMsg struct {
	// Topic name
	Topic string `json:"topic"`
	// Message type. Either "snapshot" or "delta"
	Type string `json:"type"`
	// The timestamp (ms) that the system generates the data
	Timestamp int64 `json:"ts"`
	// Main data
	Data OrderbookMsgData `json:"data"`
}

// Start the connection the the orderbook stream websocket. You must start a stream before
// any messages can be received. The connection will be automatically retried on failure
// with exponential backoff.
func (s *OrderbookStream) Start(ctx context.Context) error {

	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to OrderbookStream websocket: %w", err)
	}

	go func() {
		defer close(s.msgs)
		defer s.ws.Close()
		ticker := time.NewTicker(20 * time.Second)
		for {
			select {
			case data := <-s.ws.Messages():
				var msg OrderbookMsg
				if err := json.Unmarshal(data.Data(), &msg); err != nil {
					s.errc <- fmt.Errorf("deserializing Bybit orderbook message: %w (%s)", err, string(data.Data()))
					data.Release()
					return
				}
				if msg.Topic == "" {
					if !isPingResponseMsg(data.Data()) {
						// It should be the subscription response
						if err := checkSubscribeResponse(data.Data()); err != nil {
							s.errc <- err
							data.Release()
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
func (s *OrderbookStream) Err() <-chan error {
	return s.errc
}

// Messages is a channel producing messages received from the orderbook stream. It
// should be read concurrently with the Err stream.
func (s *OrderbookStream) Messages() <-chan OrderbookMsg {
	return s.msgs
}

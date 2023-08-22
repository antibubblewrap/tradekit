package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

// Create a new stream to the Bybit orderbook websocket stream for the given symbol and
// market depth.
func NewOrderbookStream(wsUrl string, depth int, symbol string) *OrderbookStream {
	ws := websocket.New(wsUrl, nil)
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

	return &OrderbookStream{ws: &ws, msgs: msgs, errc: errc}
}

type OrderbookMsgData struct {
	// Symbol name
	Symbol string `json:"s"`
	// Bids level updates. If the amount is zero it means the level is deleted.
	Bids []tradekit.Level `json:"b"`
	// Asks level updates. If the amount is zero it means the level is deleted.
	Asks []tradekit.Level `json:"a"`
	// The update ID should be the update ID of the previous message + 1, except when
	// a new snapshot is received when the sequence restarts.
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

func parsePriceLevel(v *fastjson.Value) (tradekit.Level, error) {
	priceS := string(v.GetStringBytes("0"))
	price, err := strconv.ParseFloat(priceS, 64)
	if err != nil {
		return tradekit.Level{}, fmt.Errorf("invalid level price %q", priceS)

	}
	amountS := string(v.GetStringBytes("1"))
	amount, err := strconv.ParseFloat(amountS, 64)
	if err != nil {
		return tradekit.Level{}, fmt.Errorf("invalid level amount %q", amountS)
	}
	return tradekit.Level{Price: price, Amount: amount}, nil
}

func parsePriceLevels(items []*fastjson.Value) ([]tradekit.Level, error) {
	levels := make([]tradekit.Level, len(items))
	for i, v := range items {
		level, err := parsePriceLevel(v)
		if err != nil {
			return nil, err
		}
		levels[i] = level
	}
	return levels, nil
}

func parseOrderbookMsgData(v *fastjson.Value) (OrderbookMsgData, error) {
	bids, err := parsePriceLevels(v.GetArray("b"))
	if err != nil {
		return OrderbookMsgData{}, err
	}
	asks, err := parsePriceLevels(v.GetArray("a"))
	if err != nil {
		return OrderbookMsgData{}, err
	}
	return OrderbookMsgData{
		Symbol:   string(v.GetStringBytes("s")),
		UpdateId: v.GetInt64("u"),
		Bids:     bids,
		Asks:     asks,
		Seq:      v.GetInt64("seq"),
	}, nil
}

// Parse a message received from the orderbook stream. If the message is a ping or
// subscription response, then it will return an empty OrderbookMsg and a nil error.
func (s *OrderbookStream) parseOrderbookMsg(msg websocket.Message) (OrderbookMsg, error) {
	defer msg.Release()
	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return OrderbookMsg{}, err
	}
	if isPingOrSubscribeMsg(v) {
		return OrderbookMsg{}, nil
	}

	// It should be an orderbook message
	topic := string(v.GetStringBytes("topic"))
	if !strings.HasPrefix(topic, "orderbook.") {
		return OrderbookMsg{}, fmt.Errorf("invalid Bybit orderbook message: %s", string(msg.Data()))
	}

	data, err := parseOrderbookMsgData(v.Get("data"))
	if err != nil {
		return OrderbookMsg{}, fmt.Errorf("invalid Bybit orderbook message: %w (%s)", err, string(msg.Data()))
	}
	return OrderbookMsg{
		Topic:     topic,
		Type:      string(v.GetStringBytes("type")),
		Timestamp: v.GetInt64("ts"),
		Data:      data,
	}, nil
}

// Start the connection the the orderbook stream websocket. You must start a stream before
// any messages can be received. The connection will be automatically retried on failure
// with exponential backoff.
func (s *OrderbookStream) Start(ctx context.Context) error {
	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to OrderbookStream websocket: %w", err)
	}

	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		defer close(s.msgs)
		defer s.ws.Close()
		for {
			select {
			case data := <-s.ws.Messages():
				msg, err := s.parseOrderbookMsg(data)
				if err != nil {
					s.errc <- err
					return
				}
				if msg.Topic != "" {
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

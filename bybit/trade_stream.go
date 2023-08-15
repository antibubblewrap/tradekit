package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
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
	p    fastjson.Parser
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

func parseTradeMsgData(v *fastjson.Value) (TradeMsgData, error) {
	amountS := string(v.GetStringBytes("v"))
	amount, err := strconv.ParseFloat(amountS, 64)
	if err != nil {
		return TradeMsgData{}, fmt.Errorf("invalid trade amount %q", amountS)
	}
	priceS := string(v.GetStringBytes("p"))
	price, err := strconv.ParseFloat(priceS, 64)
	if err != nil {
		return TradeMsgData{}, fmt.Errorf("invalid trade price %q", priceS)
	}
	return TradeMsgData{
		FilledTimestamp: v.GetInt64("T"),
		Symbol:          string(v.GetStringBytes("s")),
		Side:            string(v.GetStringBytes("S")),
		Price:           price,
		Amount:          amount,
	}, nil
}

// Parse a message received from the trade stream. If the message is a ping or
// subscription response, then it will return an empty OrderbookMsg and a nil error.
func (s *TradeStream) parseTradeMsg(msg websocket.Message) (TradeMsg, error) {
	defer msg.Release()
	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return TradeMsg{}, err
	}
	if isPingOrSubscribeMsg(v) {
		return TradeMsg{}, nil
	}

	// It should be a trade message
	topic := string(v.GetStringBytes("topic"))
	if !strings.HasPrefix(topic, "publicTrade.") {
		return TradeMsg{}, fmt.Errorf("invalid Bybit trade message: %s", string(msg.Data()))
	}

	items := v.GetArray("data")
	trades := make([]TradeMsgData, len(items))
	for i, v := range items {
		trade, err := parseTradeMsgData(v)
		if err != nil {
			return TradeMsg{}, fmt.Errorf("invalid Bybit trade message: %w (%s)", err, string(msg.Data()))
		}
		trades[i] = trade
	}

	return TradeMsg{
		Topic:     topic,
		Timestamp: v.GetInt64("ts"),
		Data:      trades,
	}, nil

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
				msg, err := s.parseTradeMsg(data)
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
func (s *TradeStream) Err() <-chan error {
	return s.errc
}

// Messages returns a channel producing messages received from the trades stream. It
// should be read concurrently with the Err stream.
func (s *TradeStream) Messages() <-chan TradeMsg {
	return s.msgs
}

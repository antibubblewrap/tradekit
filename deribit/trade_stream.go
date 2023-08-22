package deribit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

type TradeStream struct {
	ws              *websocket.Websocket
	msgs            chan []TradeMsg
	errc            chan error
	p               fastjson.Parser
	subs            map[subscription]struct{}
	subRequestIds   map[int64]struct{}
	unsubRequestIds map[int64]struct{}
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

type TradeSub struct {
	Instrument string
	Freq       UpdateFrequency
}

func (sub TradeSub) channel() string {
	return fmt.Sprintf("trades.%s.%s", sub.Instrument, sub.Freq)
}

func NewTradeStream(wsUrl string, opts *tradekit.StreamOptions, subs ...TradeSub) (*TradeStream, error) {
	ws := websocket.New(wsUrl, nil)
	ws.PingInterval = 15 * time.Second

	if opts != nil {
		if opts.ResetInterval != 0 {
			ws.ResetInterval = opts.ResetInterval
		}
	}

	msgs := make(chan []TradeMsg)
	errc := make(chan error)

	subscriptions := make(map[subscription]struct{})
	for _, sub := range subs {
		subscriptions[sub] = struct{}{}
	}

	stream := &TradeStream{
		ws:              &ws,
		msgs:            msgs,
		errc:            errc,
		subs:            subscriptions,
		subRequestIds:   make(map[int64]struct{}),
		unsubRequestIds: make(map[int64]struct{}),
	}
	stream.ws.OnConnect = func() error { return subscribeAll[[]TradeMsg](stream) }

	return stream, nil

}

func (s *TradeStream) subscriptions() map[subscription]struct{} {
	return s.subs
}

func (s *TradeStream) subscribeRequestIds() map[int64]struct{} {
	return s.subRequestIds
}

func (s *TradeStream) unsubscribRequestIds() map[int64]struct{} {
	return s.unsubRequestIds
}

func (s *TradeStream) parseChannel(channel string) (subscription, error) {
	sp := strings.Split(channel, ".")
	if len(sp) != 3 {
		return TradeSub{}, fmt.Errorf("invalid trade channel %q", channel)
	}
	if sp[0] != "trades" {
		return TradeSub{}, fmt.Errorf("invalid trade channel %q", channel)
	}
	instrument := sp[1]
	freq, err := updateFreqFromString(sp[2])
	if err != nil {
		return TradeSub{}, fmt.Errorf("invalid trade channel %q: %w", channel, err)
	}
	return TradeSub{instrument, freq}, nil
}

func (s *TradeStream) parseMessage(v *fastjson.Value) ([]TradeMsg, error) {
	items, err := v.Array()
	if err != nil {
		return nil, err
	}
	trades := make([]TradeMsg, len(items))
	for i, v := range items {
		trades[i] = parseTrade(v)
	}
	return trades, nil
}

func (s *TradeStream) websocket() *websocket.Websocket {
	return s.ws
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

// Subscribe adds 1 or more trade subscriptions to the current stream.
func (s *TradeStream) Subscribe(subs ...TradeSub) error {
	// Note: we send the subscribe request here, but wait until the response is
	// received in parseOrderBookMsg below to update the set of active subscriptions.
	newChannels := make([]string, 0)
	for _, sub := range subs {
		if _, ok := s.subs[sub]; !ok {
			newChannels = append(newChannels, sub.channel())
		}
	}
	if len(newChannels) > 0 {
		id := genId()
		subMsg, err := rpcSubscribeMsg(id, newChannels)
		if err != nil {
			return err
		}
		s.ws.Send(subMsg)
		s.subRequestIds[id] = struct{}{}
	}
	return nil
}

// Unsubscribe removes 1 or more trade subscriptions from the current stream.
func (s *TradeStream) Unsubscribe(subs ...TradeSub) error {
	// Note: we send the unsubscribe request here, but wait until the response is
	// received in parseOrderBookMsg below to update the set of active subscriptions.
	removeChannels := make([]string, 0)
	for _, sub := range subs {
		if _, ok := s.subs[sub]; ok {
			removeChannels = append(removeChannels, sub.channel())
		}
	}
	if len(removeChannels) > 0 {
		id := genId()
		unsubMsg, err := rpcUnsubscribeMsg(id, removeChannels)
		s.unsubRequestIds[id] = struct{}{}
		if err != nil {
			return err
		}
		s.ws.Send(unsubMsg)
		s.unsubRequestIds[id] = struct{}{}
	}
	return nil
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
		defer s.ws.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-s.ws.Messages():
				msg, err := parseStreamMsg[[]TradeMsg](s, data, s.p)
				if err != nil {
					s.errc <- fmt.Errorf("invalid Deribit trade message: %w", err)
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

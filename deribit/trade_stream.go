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
	ws              *websocket.Websocket
	msgs            chan []TradeMsg
	errc            chan error
	p               fastjson.Parser
	initSubs        []string
	subs            map[TradeSub]struct{}
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

func NewTradeStream(t ConnectionType, subs ...TradeSub) (*TradeStream, error) {
	url, err := wsUrl(t)
	if err != nil {
		return nil, err
	}
	ws := websocket.New(url, nil)
	ws.PingInterval = 15 * time.Second

	msgs := make(chan []TradeMsg)
	errc := make(chan error)

	initSubs := make([]string, len(subs))
	for i, sub := range subs {
		initSubs[i] = sub.channel()
	}

	return &TradeStream{
		ws:              &ws,
		msgs:            msgs,
		errc:            errc,
		initSubs:        initSubs,
		subs:            make(map[TradeSub]struct{}),
		subRequestIds:   make(map[int64]struct{}),
		unsubRequestIds: make(map[int64]struct{}),
	}, nil
}

func (s *TradeStream) subscriptions() map[TradeSub]struct{} {
	return s.subs
}

func (s *TradeStream) subscribeRequestIds() map[int64]struct{} {
	return s.subRequestIds
}

func (s *TradeStream) unsubscribRequestIds() map[int64]struct{} {
	return s.unsubRequestIds
}

func (s *TradeStream) parseChannel(channel string) (TradeSub, error) {
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

// onConnect makes the channel subscriptions after the websocket connection is made,
// either on the initial connection, or after any re-connections.
func (s *TradeStream) onConnect() error {
	// If it's the first connection, then subscribe to the set of channels specified when
	// the stream was created, otherwise re-subscribe to the set of existing channels.
	var channels []string
	if len(s.initSubs) > 0 {
		channels = s.initSubs
	} else {
		channels = make([]string, 0, len(s.subs))
		for sub := range s.subs {
			channels = append(channels, sub.channel())
		}
	}
	id := genId()
	subMsg, err := newSubscribeMsg(id, channels)
	if err != nil {
		return err
	}
	s.ws.Send(subMsg)
	s.subRequestIds[id] = struct{}{}
	s.initSubs = nil
	return nil
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
		subMsg, err := newSubscribeMsg(id, newChannels)
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
		unsubMsg, err := newUnsubscribeMsg(id, removeChannels)
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
	s.ws.OnConnect = s.onConnect
	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to Deribit TradeStream websocket: %w", err)
	}

	go func() {
		defer close(s.msgs)
		for {
			select {
			case data := <-s.ws.Messages():
				msg, err := parseStreamMsg[TradeSub, []TradeMsg](s, data, s.p)
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

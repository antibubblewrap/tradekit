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
	ws                *websocket.Websocket
	msgs              chan OrderbookMsg
	errc              chan error
	p                 fastjson.Parser
	initSubscriptions []string
	subs              map[OrderbookSub]struct{}
	subRequestIds     map[int64]struct{}
	unsubRequestIds   map[int64]struct{}
}

type OrderbookSub struct {
	Instrument string
	Freq       UpdateFrequency
}

func (sub *OrderbookSub) channel() string {
	return fmt.Sprintf("book.%s.%s", sub.Instrument, sub.Freq)
}

func NewOrderbookStream(t ConnectionType, subs ...OrderbookSub) (*OrderbookStream, error) {
	url, err := wsUrl(t)
	if err != nil {
		return nil, err
	}
	ws := websocket.New(url, nil)
	ws.PingInterval = 15 * time.Second

	initSubs := make([]string, len(subs))
	for i, sub := range subs {
		initSubs[i] = sub.channel()
	}

	return &OrderbookStream{
		ws:                &ws,
		msgs:              make(chan OrderbookMsg),
		errc:              make(chan error),
		initSubscriptions: initSubs,
		subs:              make(map[OrderbookSub]struct{}),
		subRequestIds:     make(map[int64]struct{}),
		unsubRequestIds:   make(map[int64]struct{}),
	}, nil
}

func (s *OrderbookStream) subscriptions() map[OrderbookSub]struct{} {
	return s.subs
}

func (s *OrderbookStream) subscribeRequestIds() map[int64]struct{} {
	return s.subRequestIds
}

func (s *OrderbookStream) unsubscribRequestIds() map[int64]struct{} {
	return s.unsubRequestIds
}

func (s *OrderbookStream) parseChannel(channel string) (OrderbookSub, error) {
	sp := strings.Split(channel, ".")
	if len(sp) != 3 {
		return OrderbookSub{}, fmt.Errorf("invalid orderbook channel %q", channel)
	}
	if sp[0] != "book" {
		return OrderbookSub{}, fmt.Errorf("invalid orderbook channel %q", channel)
	}
	instrument := sp[1]
	freq, err := updateFreqFromString(sp[2])
	if err != nil {
		return OrderbookSub{}, fmt.Errorf("invalid orderbook channel %q: %w", channel, err)
	}
	return OrderbookSub{instrument, freq}, nil
}

func (s *OrderbookStream) parseMessage(v *fastjson.Value) (OrderbookMsg, error) {
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

// onConnect makes the channel subscriptions after the websocket connection is made,
// either on the initial connection, or after any re-connections.
func (s *OrderbookStream) onConnect() error {
	// If it's the first connection, then subscribe to the set of channels specified when
	// the stream was created, otherwise re-subscribe to the set of existing channels.
	var channels []string
	if len(s.initSubscriptions) > 0 {
		channels = s.initSubscriptions
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
	s.initSubscriptions = nil
	return nil
}

// Subscribe to a new orderbook stream.
func (s *OrderbookStream) Subscribe(subs ...OrderbookSub) error {
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

// Unsubscribe from an orderbook stream.
func (s *OrderbookStream) Unsubscribe(subs ...OrderbookSub) error {
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

// Start the connection the the Deribit orderbook stream websocket. You must start a
// stream before any messages can be received. The connection will be automatically
// retried on failure with exponential backoff.
func (s *OrderbookStream) Start(ctx context.Context) error {
	s.ws.OnConnect = s.onConnect
	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to Deribit OrderbookStream websocket: %w", err)
	}

	go func() {
		defer close(s.msgs)
		for {
			select {
			case data := <-s.ws.Messages():
				msg, err := parseStreamMsg[OrderbookSub, OrderbookMsg](s, data, s.p)
				if err != nil {
					s.errc <- fmt.Errorf("invalid Deribit orderbook message: %w", err)
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

// The error stream should be read concurrently with the messages stream. If
// this channel produces an error, the messages stream will be closed.
func (s *OrderbookStream) Err() <-chan error {
	return s.errc
}

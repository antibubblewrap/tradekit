package deribit

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

// OrderbookDepth stream streams orderbook depth updates from the Deribit book.{instrument_name}.{group}.{depth}.{interval}
// channel. For streaming incremental order book updates, use OrderbookStream.
type OrderbookDepthStream struct {
	ws                *websocket.Websocket
	msgs              chan OrderbookDepthMsg
	errc              chan error
	p                 fastjson.Parser
	initSubscriptions []string
	subs              map[OrderbookDepthSub]struct{}
	subRequestIds     map[int64]struct{}
	unsubRequestIds   map[int64]struct{}
}

// OrderbookDepthSub represents a subscription to a deribit orderbook depth stream.
// The update interval is automatically set to 100ms, and the price grouping set to none.
type OrderbookDepthSub struct {
	// Instrument name
	Instrument string
	// Depth of orderbook to stream. Valid values are 1, 10, 20
	Depth int
}

func (sub *OrderbookDepthSub) channel() string {
	return fmt.Sprintf("book.%s.none.%d.100ms", sub.Instrument, sub.Depth)
}

// NewOrderbookDepthStream creates a new stream for reading from the Deribit book depth
// updates stream. You must call Start on the stream to receive messages.
func NewOrderbookDepthStream(t ConnectionType, subs ...OrderbookDepthSub) (*OrderbookDepthStream, error) {
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

	return &OrderbookDepthStream{
		ws:                &ws,
		msgs:              make(chan OrderbookDepthMsg),
		errc:              make(chan error),
		initSubscriptions: initSubs,
		subs:              make(map[OrderbookDepthSub]struct{}),
		subRequestIds:     make(map[int64]struct{}),
		unsubRequestIds:   make(map[int64]struct{}),
	}, nil
}

func (s *OrderbookDepthStream) subscriptions() map[OrderbookDepthSub]struct{} {
	return s.subs
}

func (s *OrderbookDepthStream) subscribeRequestIds() map[int64]struct{} {
	return s.subRequestIds
}

func (s *OrderbookDepthStream) unsubscribRequestIds() map[int64]struct{} {
	return s.unsubRequestIds
}

func (s *OrderbookDepthStream) parseChannel(channel string) (OrderbookDepthSub, error) {
	sp := strings.Split(channel, ".")
	if len(sp) != 5 {
		return OrderbookDepthSub{}, fmt.Errorf("invalid book depth channel %q", channel)
	}
	if sp[0] != "book" {
		return OrderbookDepthSub{}, fmt.Errorf("invalid book depth channel %q", channel)
	}
	instrument := sp[1]
	depth, err := strconv.ParseInt(sp[3], 10, 64)
	if err != nil {
		return OrderbookDepthSub{}, fmt.Errorf("invalid book depth channel: invalid depth %s", sp[3])
	}
	return OrderbookDepthSub{instrument, int(depth)}, nil
}

// OrderbookDepthMsg is the message type streamed from the Deribit book depth channel.
// For more info see: https://docs.deribit.com/#book-instrument_name-group-depth-interval
type OrderbookDepthMsg struct {
	Timestamp  int64            `json:"timestamp"`
	Instrument string           `json:"instrument_name"`
	ChangeID   int64            `json:"change_id"`
	Bids       []tradekit.Level `json:"bids"`
	Asks       []tradekit.Level `json:"asks"`
}

func parsePriceLevel(v *fastjson.Value) tradekit.Level {
	price := v.GetFloat64("0")
	amount := v.GetFloat64("1")
	return tradekit.Level{Price: price, Amount: amount}
}

func parsePriceLevels(items []*fastjson.Value) []tradekit.Level {
	levels := make([]tradekit.Level, len(items))
	for i, v := range items {
		levels[i] = parsePriceLevel(v)
	}
	return levels
}

func (s *OrderbookDepthStream) parseMessage(v *fastjson.Value) (OrderbookDepthMsg, error) {
	return OrderbookDepthMsg{
		Timestamp:  v.GetInt64("timestamp"),
		Instrument: string(v.GetStringBytes("instrument_name")),
		ChangeID:   v.GetInt64("change_id"),
		Bids:       parsePriceLevels(v.GetArray("bids")),
		Asks:       parsePriceLevels(v.GetArray("asks")),
	}, nil
}

// onConnect makes the channel subscriptions after the websocket connection is made,
// either on the initial connection, or after any re-connections.
func (s *OrderbookDepthStream) onConnect() error {
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

// Subscribe adds 1 or more additional subscriptions to the stream.
func (s *OrderbookDepthStream) Subscribe(subs ...OrderbookDepthSub) error {
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

// Unsubscribe removes 1 or more subscriptions from the stream.
func (s *OrderbookDepthStream) Unsubscribe(subs ...OrderbookDepthSub) error {
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

// Start the connection the the Deribit orderbook depth stream websocket. You must start a
// stream before any messages can be received. The connection will be automatically
// retried on failure with exponential backoff.
func (s *OrderbookDepthStream) Start(ctx context.Context) error {
	s.ws.OnConnect = s.onConnect
	if err := s.ws.Start(ctx); err != nil {
		return fmt.Errorf("connecting to Deribit OrderbookDepthStream websocket: %w", err)
	}

	go func() {
		defer close(s.msgs)
		for {
			select {
			case data := <-s.ws.Messages():
				msg, err := parseStreamMsg[OrderbookDepthSub, OrderbookDepthMsg](s, data, s.p)
				if err != nil {
					s.errc <- fmt.Errorf("invalid Deribit book depth message: %w", err)
					return
				}
				if msg.Instrument != "" {
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

// Messages returns a channel producing messages received from the orderbook depth stream.
// It should be read concurrently with the Err stream.
func (s *OrderbookDepthStream) Messages() <-chan OrderbookDepthMsg {
	return s.msgs
}

// The error stream should be read concurrently with the messages stream. If
// this channel produces an error, the messages stream will be closed.
func (s *OrderbookDepthStream) Err() <-chan error {
	return s.errc
}

package deribit

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

// Subscription represents a subscription to a Deribit stream created using a UserStream.
// The stream may be read from the its Messages() channel.
type Subscription[T any] struct {
	// Close stops the stream. Once called, the Messages() channel is closed.
	Close func()
	c     chan T
}

// Messages returns a channel of messages received from the subscription.
func (s Subscription[T]) Messages() <-chan T {
	return s.c
}

// UserStream provides private subscriptions for updates on user trades and orders.
type UserStream interface {
	// Start the stream. No subscriptions may be made until the stream is started. All
	// subscriptions will be closed if the provided context is cancelled.
	Start(ctx context.Context) error

	// Err returns a channel of errors encountered while processing subscriptions. If
	// this channel produces an error, all subscriptions will be closed.
	Err() <-chan error

	// TradesByCurrencyAndKind starts a subscription to a user.trades.{currency}.{kind}.raw
	// stream. It returns a channel from which trade updates may be read. It returns an
	// error if a subscription for the currency and instrument kind already exists.
	// https://docs.deribit.com/#user-trades-kind-currency-interval
	TradesByCurrencyAndKind(currency string, kind InstrumentKind) Subscription[TradeExecution]

	// TradesByInstrument starts a subscription to a user.trades.{instrument}.raw
	// stream. It returns a channel from which trade updates may be read. It returns
	// an error if a subscription for the instrument already exists.
	// https://docs.deribit.com/#user-trades-instrument_name-interval
	TradesByInstrument(instrument string) Subscription[TradeExecution]

	// OrdersByCurrencyAndKind starts a subscription to a user.orders.{currency}.{kind}.raw
	// stream. It returns a channel from which order updates may be read. It returns an
	// error if a subscription for the currency and instrument kind already exists.
	// https://docs.deribit.com/#user-orders-kind-currency-interval
	OrdersByCurrencyAndKind(currency string, kind InstrumentKind) Subscription[Order]

	// OrdersByInstrument starts a subscription to a user.orders.{instrument}.raw
	// stream. It returns a channel from which order updates may be read. It returns
	// an error if a subscription for the instrument already exists.
	// https://docs.deribit.com/#user-orders-instrument_name-interval
	OrdersByInstrument(instrument string) Subscription[Order]
}

type tradesSub struct {
	channel string
	c       chan Subscription[TradeExecution]
}

type ordersSub struct {
	channel string
	c       chan Subscription[Order]
}

type unsub struct {
	channel string
	c       chan struct{}
}

// userStream implements the UserStream interface.
type userStream struct {
	ws          *websocket.Websocket
	credentials Credentials
	errc        chan error
	p           fastjson.Parser

	tradeSubscriptions map[string]Subscription[TradeExecution]
	orderSubscriptions map[string]Subscription[Order]

	// Use channels to subscribe / unsubscribe from channels. Otherwise we'd need to wrap
	// the tradesSubscriptions and orderSubscriptions maps in mutexes to prevent race
	// conditions.
	subscribeTrades   chan tradesSub
	subscribeOrders   chan ordersSub
	unsubscribeTrades chan unsub
	unsubscribeOrders chan unsub
}

// NewUserStream creates a UserStream connecting to the given Deribit websocket URL using
// the provided credentials for authentication.
func NewUserStream(wsUrl string, c Credentials) UserStream {
	ws := websocket.New(wsUrl, nil)
	ws.PingInterval = time.Second * 15
	return &userStream{
		ws:          &ws,
		credentials: c,
		errc:        make(chan error, 1),

		tradeSubscriptions: make(map[string]Subscription[TradeExecution]),
		orderSubscriptions: make(map[string]Subscription[Order]),

		subscribeTrades:   make(chan tradesSub, 1),
		subscribeOrders:   make(chan ordersSub, 1),
		unsubscribeTrades: make(chan unsub, 1),
		unsubscribeOrders: make(chan unsub, 1),
	}
}

func (s *userStream) Start(ctx context.Context) error {
	s.ws.OnConnect = func() error {
		// Authenticate and wait for response. We don't need to do anything with it,
		// just make sure it was successful.
		id, err := s.sendAuthenticate()
		if err != nil {
			return userStreamErr(fmt.Errorf("authenticating: %w", err))
		}
		msg := <-s.ws.Messages()
		defer msg.Release()
		var p fastjson.Parser
		v, err := p.ParseBytes(msg.Data())
		if err != nil {
			return err
		}
		if id != v.GetInt64("id") {
			return userStreamErr(fmt.Errorf("expected auth response but received %s", msg.Data()))
		}
		if err := isRpcError(v); err != nil {
			return userStreamErr(fmt.Errorf("auth failure: %w", err))
		}

		// If the websocket is reconnecting, we'll need to resubscribe to all the channels
		_, err = s.sendPrivateSubscribe(s.allChannels()...)
		if err != nil {
			return userStreamErr(fmt.Errorf("subscribe failure: %w", err))
		}

		return nil
	}

	go func() {
		defer func() {
			s.ws.Close()
			s.closeAllSubscriptions()
			close(s.errc)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-s.ws.Messages():
				if err := s.handleResponse(data); err != nil {
					s.errc <- userStreamErr(err)
					return
				}
			case sub := <-s.subscribeTrades:
				sub.c <- s.makeTradesSub(sub.channel)
			case sub := <-s.subscribeOrders:
				sub.c <- s.makeOrdersSub(sub.channel)
			case unsub := <-s.unsubscribeTrades:
				s.closeTradesSubscription(unsub.channel)
				unsub.c <- struct{}{}
			case unsub := <-s.unsubscribeOrders:
				s.closeOrdersSubscription(unsub.channel)
				unsub.c <- struct{}{}
			case err := <-s.ws.Err():
				s.errc <- userStreamErr(err)
				return
			}
		}
	}()
	return nil
}

func (s *userStream) Err() <-chan error {
	return s.errc
}

// TradesByCurrencyAndKind implements the UserStream interface.
func (s *userStream) TradesByCurrencyAndKind(currency string, kind InstrumentKind) Subscription[TradeExecution] {
	channel := fmt.Sprintf("user.trades.%s.%s.raw", currency, kind)
	req := tradesSub{channel, make(chan Subscription[TradeExecution], 1)}
	s.subscribeTrades <- req
	return <-req.c
}

// TradesByInstrument implements the UserStream interface.
func (s *userStream) TradesByInstrument(instrument string) Subscription[TradeExecution] {
	channel := fmt.Sprintf("user.trades.%s.raw", instrument)
	req := tradesSub{channel, make(chan Subscription[TradeExecution], 1)}
	s.subscribeTrades <- req
	return <-req.c
}

// OrdersByCurrencyAndKind implements the UserStream interface.
func (s *userStream) OrdersByCurrencyAndKind(currency string, kind InstrumentKind) Subscription[Order] {
	channel := fmt.Sprintf("user.orders.%s.%s.raw", currency, kind)
	req := ordersSub{channel, make(chan Subscription[Order], 1)}
	s.subscribeOrders <- req
	return <-req.c
}

// OrdersByInstrument implements the UserStream interface.
func (s *userStream) OrdersByInstrument(instrument string) Subscription[Order] {
	channel := fmt.Sprintf("user.orders.%s.raw", instrument)
	req := ordersSub{channel, make(chan Subscription[Order], 1)}
	s.subscribeOrders <- req
	return <-req.c
}

// handleResponse takes a message received from the stream's websocket and sends it to
// the appropriate subscription.
func (s *userStream) handleResponse(msg websocket.Message) error {
	defer msg.Release()

	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return fmt.Errorf("parsing message: %w (%s)", err, string(msg.Data()))
	}

	if !bytes.Equal(v.GetStringBytes("method"), []byte("subscription")) {
		// This message is just a subscribe response. We can ignore it.
		return nil
	}

	params := v.Get("params")
	if params == nil {
		return fmt.Errorf("message missing field %q: %s", "params", string(msg.Data()))
	}
	channel := string(params.GetStringBytes("channel"))

	if strings.HasPrefix(channel, "user.trades") {
		sub, ok := s.tradeSubscriptions[channel]
		if !ok {
			return nil
		}
		for _, item := range params.GetArray("data") {
			sub.c <- parseTradeExecution(item)
		}
	} else if strings.HasPrefix(channel, "user.orders") {
		sub, ok := s.orderSubscriptions[channel]
		if !ok {
			return nil
		}
		for _, item := range params.GetArray("data") {
			sub.c <- parseOrder(item)
		}
	}

	return nil
}

func (s *userStream) makeTradesSub(channel string) Subscription[TradeExecution] {
	if sub, ok := s.tradeSubscriptions[channel]; ok {
		return sub
	}
	c := make(chan TradeExecution, 10)
	sub := Subscription[TradeExecution]{
		Close: func() {
			req := unsub{channel, make(chan struct{}, 1)}
			s.unsubscribeTrades <- req
			<-req.c
		},
		c: c,
	}
	s.tradeSubscriptions[channel] = sub
	return sub
}

func (s *userStream) makeOrdersSub(channel string) Subscription[Order] {
	if sub, ok := s.orderSubscriptions[channel]; ok {
		return sub
	}
	c := make(chan Order, 10)
	sub := Subscription[Order]{
		Close: func() {
			req := unsub{channel, make(chan struct{}, 1)}
			s.unsubscribeOrders <- req
			<-req.c
		},
		c: c,
	}
	s.orderSubscriptions[channel] = sub
	return sub
}

func (s *userStream) closeTradesSubscription(channel string) {
	if sub, ok := s.tradeSubscriptions[channel]; ok {
		delete(s.tradeSubscriptions, channel)
		s.sendPrivateUnsubscribe(channel)
		close(sub.c)
	}
}

func (s *userStream) closeOrdersSubscription(channel string) {
	if sub, ok := s.orderSubscriptions[channel]; ok {
		delete(s.orderSubscriptions, channel)
		s.sendPrivateUnsubscribe(channel)
		close(sub.c)
	}
}

func (s *userStream) closeAllSubscriptions() {
	for channel, sub := range s.tradeSubscriptions {
		delete(s.tradeSubscriptions, channel)
		close(sub.c)
	}
	for channel, sub := range s.orderSubscriptions {
		delete(s.orderSubscriptions, channel)
		close(sub.c)
	}
}

func (s *userStream) allChannels() []string {
	channels := make([]string, 0, len(s.orderSubscriptions)+len(s.tradeSubscriptions))
	for c := range s.orderSubscriptions {
		channels = append(channels, c)
	}
	for c := range s.tradeSubscriptions {
		channels = append(channels, c)
	}
	return channels
}

// Send an authentication request along the stream's websocket
func (s *userStream) sendAuthenticate() (id int64, err error) {
	id = genId()
	method := methodPublicAuth
	params := authCredentialsRequest(s.credentials)
	msg, err := rpcRequestMsg(method, id, params)
	if err != nil {
		return
	}
	s.ws.Send(msg)
	return
}

// Send a private subscribe request along the stream's websocket
func (s *userStream) sendPrivateSubscribe(channels ...string) (id int64, err error) {
	id = genId()
	method := methodPrivateSubscribe
	params := map[string]interface{}{"channels": channels}
	msg, err := rpcRequestMsg(method, id, params)
	if err != nil {
		return
	}
	s.ws.Send(msg)
	return
}

// Send a private unsubscribe request along the stream's websocket
func (s *userStream) sendPrivateUnsubscribe(channels ...string) {
	id := genId()
	method := methodPrivateUnsubscribe
	params := map[string]interface{}{"channels": channels}
	msg, err := rpcRequestMsg(method, id, params)
	if err != nil {
		return
	}
	s.ws.Send(msg)
	return
}

func userStreamErr(err error) error {
	return fmt.Errorf("Deribit UserStream: %w", err)
}

func subAlreadyExistsErr(channel string) error {
	return fmt.Errorf("a subscription to channel %q already exists", channel)
}

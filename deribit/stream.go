package deribit

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/internal/set"
	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

// A Stream represents a connection to a collection of Deribit streaming subscriptions
// over a websocket connection. For more details see:
//   - https://docs.deribit.com/#subscriptions
//
// The following functions create Streams to public channels:
//   - [NewTradesStream]
//   - [NewOrderbookStream]
//   - [NewOrderbookDepthStream]
//   - [NewInstrumentStateStream]
//
// The following functions create Streams to private user channels:
//   - [NewUserTradesStream]
//   - [NewUserOrdersStream]
type Stream[T any, U subscription] interface {
	// SetStreamOptions sets optional parameters for the stream. If used, it should be
	// called before Start.
	SetStreamOptions(*tradekit.StreamOptions)

	// SetCredentials sets credentials for authentication with Deribit. Check the Deribit
	// documentation to see if authentication is required for a particular stream.
	SetCredentials(*Credentials)

	// Start the stream. The stream must be started before any messages will be received
	// or any new subscriptions may be made.
	Start(context.Context) error

	// Messages returns a channel of messages received from the stream's subscriptions.
	Messages() <-chan T

	// Err returns a channel which produces an error when there is an irrevocable failure
	// with the stream's connection. It should be read concurrently with the Messages
	// channel. If the channel produces and error, the stream stops and the Messages
	// channel is closed, and no further subscriptions may be made.
	Err() <-chan error

	// Subscribe adds a new subscription to the stream. This is a no-op if the
	// subscription already exists.
	Subscribe(subs ...U)

	// Unsubscribe removes a subscription from the stream. This is a no-op if the
	// subscription does not already exist.
	Unsubscribe(subs ...U)
}

// stream makes subscriptions to Deribit channels and provides a channel to receive
// the messages they produce.
type stream[T any] struct {
	name          string
	url           string
	msgs          chan T
	errc          chan error
	parseMessage  func(*fastjson.Value) T
	isPrivate     bool
	subscriptions set.Set[string]
	opts          *tradekit.StreamOptions
	credentials   *Credentials

	subRequests          chan []subscription
	unsubRequests        chan []subscription
	subscribeAllRequests chan struct{}

	closed atomic.Bool
	p      fastjson.Parser
}

type subscription interface {
	channel() string
}

type streamParams[T any] struct {
	name         string
	wsUrl        string
	isPrivate    bool
	parseMessage func(*fastjson.Value) T
	subs         []subscription
}

func newStream[T any](p streamParams[T]) *stream[T] {

	channels := make([]string, len(p.subs))
	for i, sub := range p.subs {
		channels[i] = sub.channel()
	}

	return &stream[T]{
		name:                 p.name,
		url:                  p.wsUrl,
		msgs:                 make(chan T, 10),
		errc:                 make(chan error, 1),
		parseMessage:         p.parseMessage,
		subscriptions:        set.New[string](channels...),
		isPrivate:            p.isPrivate,
		subRequests:          make(chan []subscription, 10),
		unsubRequests:        make(chan []subscription, 10),
		subscribeAllRequests: make(chan struct{}, 10),
	}
}

func (s *stream[T]) SetStreamOptions(opts *tradekit.StreamOptions) {
	s.opts = opts
}

func (s *stream[T]) SetCredentials(c *Credentials) {
	s.credentials = c
}

func (s *stream[T]) Start(ctx context.Context) error {
	var wsOpts websocket.Options
	if s.opts != nil {
		if s.opts.BufferCapacity != 0 {
			wsOpts.BufCapacity = int(s.opts.BufferCapacity)
		}
		if s.opts.BufferPoolSize != 0 {
			wsOpts.PoolSize = int(s.opts.BufferPoolSize)
		}
		if s.opts.ResetInterval != 0 {
			wsOpts.ResetInterval = s.opts.ResetInterval
		}
		if s.opts.PingInterval != 0 {
			wsOpts.PingInterval = s.opts.PingInterval
		}
	}
	ws := websocket.New(s.url, &wsOpts)

	ws.OnConnect = func() error {
		// Authenticate and wait for the response if the credentials are supplied.
		if s.credentials != nil {
			if s.closed.Load() {
				return nil
			}

			id, err := s.authenticate(&ws)
			if err != nil {
				return fmt.Errorf("authenticating: %w", err)
			}
			msg := <-ws.Messages()
			defer msg.Release()
			var p fastjson.Parser
			v, err := p.ParseBytes(msg.Data())
			if err != nil {
				return err
			}
			if id != v.GetInt64("id") {
				return fmt.Errorf("expected auth response but received %s", msg.Data())
			}
			if err := isRpcError(v); err != nil {
				return fmt.Errorf("auth failure: %w", err)
			}
		}

		s.subscribeAllRequests <- struct{}{}
		return nil
	}

	go func() {
		defer func() {
			s.closed.Store(true)
			close(s.msgs)
			close(s.errc)
			close(s.subRequests)
			close(s.unsubRequests)
			close(s.subscribeAllRequests)
			ws.Close()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ws.Messages():
				if err := s.handleMessage(msg); err != nil {
					s.errc <- s.nameErr(err)
					return
				}
			case subs := <-s.subRequests:
				if err := s.subscribe(&ws, subs...); err != nil {
					s.errc <- s.nameErr(err)
					return
				}
			case subs := <-s.unsubRequests:
				if err := s.unsubscribe(&ws, subs...); err != nil {
					s.errc <- s.nameErr(err)
					return
				}
			case <-s.subscribeAllRequests:
				if err := s.subscribeAll(&ws); err != nil {
					s.errc <- s.nameErr(err)
					return
				}
			case err := <-ws.Err():
				s.errc <- s.nameErr(err)
				return
			}
		}
	}()

	if err := ws.Start(ctx); err != nil {
		return s.nameErr(fmt.Errorf("connecting to websocket: %w", err))
	}

	return nil
}

func (s *stream[T]) handleMessage(msg websocket.Message) error {
	defer msg.Release()

	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return fmt.Errorf("invalid message: %s", string(msg.Data()))
	}

	method := v.GetStringBytes("method")
	if method == nil {
		// This is a subscribe / unsubscribe response. We can ignore it.
		return nil
	}

	params := v.Get("params")
	if params == nil {
		return fmt.Errorf(`field "params" is missing: %s`, string(msg.Data()))
	}

	channel := string(params.GetStringBytes("channel"))
	if !s.subscriptions.Exists(channel) {
		// We've received a message from a channel which we're no longer subscribed to. It's
		// okay to ignore it.
		return nil
	}

	data := params.Get("data")
	if data == nil {
		return fmt.Errorf(`field "params.data" is missing: %s`, string(msg.Data()))
	}
	s.msgs <- s.parseMessage(data)
	return nil
}

// Subscribe to all of the stream's subscriptions
func (s *stream[T]) subscribeAll(ws *websocket.Websocket) error {
	if s.closed.Load() {
		return errors.New("stream is closed")
	}
	if s.subscriptions.Len() == 0 {
		return nil
	}
	channels := s.subscriptions.Slice()
	id := genId()
	params := map[string]interface{}{"channels": channels}
	var method rpcMethod
	if s.isPrivate {
		method = methodPrivateSubscribe
	} else {
		method = methodPublicSubscribe
	}
	msg, err := rpcRequestMsg(method, id, params)
	if err != nil {
		return err
	}
	ws.Send(msg)
	return nil
}

func (s *stream[T]) Subscribe(subs ...subscription) {
	if s.closed.Load() {
		return
	}
	s.subRequests <- subs
}

func (s *stream[T]) Unsubscribe(subs ...subscription) {
	if s.closed.Load() {
		return
	}
	s.unsubRequests <- subs
}

// subscribe to the provided slice of channels. If a channel already exists in the
// stream's subscriptions then it will be ignored. Returns an error if the stream is
// closed.
func (s *stream[T]) subscribe(ws *websocket.Websocket, subs ...subscription) error {
	if s.closed.Load() {
		return errors.New("stream is closed")
	}
	newChannels := make([]string, 0)
	for _, sub := range subs {
		c := sub.channel()
		if !s.subscriptions.Exists(c) {
			newChannels = append(newChannels, c)
			s.subscriptions.Add(c)
		}
	}

	if len(newChannels) == 0 {
		return nil
	}

	id := genId()
	params := map[string]interface{}{"channels": newChannels}
	var method rpcMethod
	if s.isPrivate {
		method = methodPrivateSubscribe
	} else {
		method = methodPublicSubscribe
	}
	msg, err := rpcRequestMsg(method, id, params)
	if err != nil {
		return err
	}
	ws.Send(msg)
	return nil
}

// unsubscribe from the provided slice of channels. Returns an error if the stream is
// closed.
func (s *stream[T]) unsubscribe(ws *websocket.Websocket, subs ...subscription) error {
	if s.closed.Load() {
		return errors.New("stream is closed")
	}
	removeChannels := make([]string, 0)
	for _, sub := range subs {
		c := sub.channel()
		if s.subscriptions.Pop(c) {
			removeChannels = append(removeChannels, c)
		}
	}

	if len(removeChannels) == 0 {
		return nil
	}

	id := genId()
	params := map[string]interface{}{"channels": removeChannels}
	var method rpcMethod
	if s.isPrivate {
		method = methodPrivateUnsubscribe
	} else {
		method = methodPublicUnsubscribe
	}
	msg, err := rpcRequestMsg(method, id, params)
	if err != nil {
		return err
	}
	ws.Send(msg)
	return nil
}

func (s *stream[T]) Err() <-chan error {
	return s.errc
}

func (s *stream[T]) Messages() <-chan T {
	return s.msgs
}

func (s *stream[T]) nameErr(err error) error {
	return fmt.Errorf("Deribit %s: %w", s.name, err)
}

// Send an authentication request along the stream's websocket
func (s *stream[T]) authenticate(ws *websocket.Websocket) (id int64, err error) {
	id = genId()
	method := methodPublicAuth
	params := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     s.credentials.ClientId,
		"client_secret": s.credentials.ClientSecret,
	}
	msg, err := rpcRequestMsg(method, id, params)
	if err != nil {
		return
	}
	ws.Send(msg)
	return
}

func genId() int64 {
	return time.Now().UnixNano()
}

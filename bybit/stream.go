package bybit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/internal/set"
	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

type Stream[T any] interface {
	SetOptions(*tradekit.StreamOptions)

	Start(context.Context) error

	Messages() <-chan T

	Err() <-chan error
}

type subscription interface {
	channel() string
}

func heartbeatMsg() []byte {
	return []byte(`{"op": "ping"}`)
}

// We don't need to pass on ping responses or subscribe responses to the consumer of a
// stream.
func isPingOrSubscribeMsg(v *fastjson.Value) bool {
	op := v.GetStringBytes("op")
	if bytes.Equal(op, []byte("ping")) || bytes.Equal(op, []byte("subscribe")) {
		return true
	}
	return false
}

type stream[T any] struct {
	name          string
	url           string
	msgs          chan T
	errc          chan error
	parseMessage  func(*fastjson.Value) (T, error)
	subscriptions set.Set[string]

	subscribeAllRequests chan struct{}

	opts *tradekit.StreamOptions
	p    fastjson.Parser
}

func newStream[T any](url string, name string, parseMessage func(*fastjson.Value) (T, error), subs []subscription) *stream[T] {
	subscriptions := set.New[string]()
	for _, s := range subs {
		subscriptions.Add(s.channel())
	}

	return &stream[T]{
		name:                 name,
		url:                  url,
		msgs:                 make(chan T, 10),
		errc:                 make(chan error, 1),
		parseMessage:         parseMessage,
		subscriptions:        subscriptions,
		subscribeAllRequests: make(chan struct{}, 1),
	}
}

func (s *stream[T]) SetOptions(opts *tradekit.StreamOptions) {
	s.opts = opts
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
		s.subscribeAllRequests <- struct{}{}
		return nil
	}

	if err := ws.Start(ctx); err != nil {
		return s.nameErr(err)
	}

	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer func() {
			ticker.Stop()
			close(s.msgs)
			close(s.errc)
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
			case <-ticker.C:
				ws.Send(heartbeatMsg())
			case <-s.subscribeAllRequests:
				if err := s.subscribeAll(&ws); err != nil {
					s.errc <- s.nameErr(err)
					return
				}
			case err := <-ws.Err():
				s.errc <- err
				return
			}
		}
	}()

	return nil
}

func (s *stream[T]) Err() <-chan error {
	return s.errc
}

func (s *stream[T]) Messages() <-chan T {
	return s.msgs
}

func (s *stream[T]) handleMessage(msg websocket.Message) error {
	defer msg.Release()
	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return err
	}

	op := v.GetStringBytes("op")
	if equalAny(op, "pong", "ping", "subscribe", "unsubscribe") {
		return nil
	}

	m, err := s.parseMessage(v)
	if err != nil {
		return err
	}
	s.msgs <- m

	return nil
}

func (s *stream[T]) subscribeAll(ws *websocket.Websocket) error {
	msg, err := subscribeMsg(s.subscriptions.Slice())
	if err != nil {
		return err
	}
	ws.Send(msg)
	return nil
}

func subscribeMsg(channels []string) ([]byte, error) {
	msg := map[string]interface{}{
		"op":   "subscribe",
		"args": channels,
	}
	return json.Marshal(msg)
}

func (s *stream[T]) nameErr(err error) error {
	return fmt.Errorf("Bybit %s: %w", s.name, err)
}

func equalAny(b []byte, strs ...string) bool {
	for _, s := range strs {
		if bytes.Equal(b, []byte(s)) {
			return true
		}
	}
	return false
}

package deribit

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

type ConnectionType int

const (
	prodEventNodeUrl string = "wss://streams.deribit.com/ws/api/v2"
	testEventNodeUrl        = "wss://test.deribit.com/den/ws"
	prodWsUrl               = "wss://www.deribit.com/ws/api/v2"
	testWsUrl               = "wss://test.deribit.com/ws/api/v2"

	prodApiUrl = "https://www.deribit.com/api/v2"
	testApiUrl = "https://test.deribit.com/api/v2"
)

const (
	ProdEventNode ConnectionType = iota + 1
	TestEventNode
	Test
	Prod
)

type InstrumentKind string

const (
	FutureInstrument      InstrumentKind = "future"
	OptionInstrument                     = "option"
	SpotInstrument                       = "spot"
	FutureComboInstrument                = "future_combo"
	OptionComboInstrument                = "option_combo"
	AnyInstrument                        = "any"
)

func wsUrl(conn ConnectionType) (string, error) {
	switch conn {
	case ProdEventNode:
		return prodEventNodeUrl, nil
	case TestEventNode:
		return testEventNodeUrl, nil
	case Prod:
		return prodWsUrl, nil
	case Test:
		return testWsUrl, nil
	default:
		return "", fmt.Errorf("invalid Deribit connection type %d", conn)
	}
}

func newSubscribeMsg(id int64, channels []string) ([]byte, error) {
	m := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "public/subscribe",
		"id":      id,
		"params": map[string]interface{}{
			"channels": channels,
		},
	}
	buf, err := json.Marshal(m)
	return buf, err
}

func newUnsubscribeMsg(id int64, channels []string) ([]byte, error) {
	m := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "public/unsubscribe",
		"id":      id,
		"params": map[string]interface{}{
			"channels": channels,
		},
	}
	buf, err := json.Marshal(m)
	return buf, err
}

type UpdateFrequency string

const (
	UpdateRaw   UpdateFrequency = "raw"
	Update100ms                 = "100ms"
)

func updateFreqFromString(s string) (UpdateFrequency, error) {
	switch s {
	case string(UpdateRaw):
		return UpdateRaw, nil
	case Update100ms:
		return Update100ms, nil
	default:
		return "", fmt.Errorf("invalid update frequency %q", s)
	}
}

type subscriptionParams struct {
	Channel string
	Data    json.RawMessage
}

type subscriptionMsg struct {
	Method string
	Params subscriptionParams
}

func parseSubscriptionMsg(b []byte) (subscriptionMsg, error) {
	var m subscriptionMsg
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("parsing Deribit subscriptionMsg: %w (%s)", err, string(b))
	}
	return m, nil
}

func genId() int64 {
	return time.Now().UnixNano()
}

type stream[T comparable, U any] interface {
	subscriptions() map[T]struct{}
	subscribeRequestIds() map[int64]struct{}
	unsubscribRequestIds() map[int64]struct{}
	parseChannel(string) (T, error)
	parseMessage(*fastjson.Value) (U, error)
}

func parseStreamMsg[T comparable, U any](s stream[T, U], msg websocket.Message, p fastjson.Parser) (U, error) {
	defer msg.Release()

	v, err := p.ParseBytes(msg.Data())
	if err != nil {
		var empty U
		return empty, fmt.Errorf("invalid message: %s", string(msg.Data()))
	}

	// Check if the message is a subscribe or unsubscribe response, and update the set
	// of active subscriptions accordingly.
	subscriptions := s.subscriptions()
	method := v.GetStringBytes("method")
	if method == nil {
		var empty U
		id := v.GetInt64("id")
		channels := v.GetArray("result")
		subUpdates := make([]T, len(channels))
		for i, c := range channels {
			channel, err := c.StringBytes()
			if err != nil {
				return empty, err
			}
			sub, err := s.parseChannel(string(channel))
			if err != nil {
				return empty, err
			}
			subUpdates[i] = sub
		}
		if _, ok := s.subscribeRequestIds()[id]; ok {
			delete(s.subscribeRequestIds(), id)
			for _, sub := range subUpdates {
				subscriptions[sub] = struct{}{}
			}
			return empty, nil
		} else if _, ok := s.unsubscribRequestIds()[id]; ok {
			delete(s.unsubscribRequestIds(), id)
			for _, sub := range subUpdates {
				delete(subscriptions, sub)
			}
			return empty, nil
		} else {
			var empty U
			return empty, fmt.Errorf("expected subscribe or unsubscribe response. Received: %s", string(msg.Data()))
		}
	}

	// If the message is not a subscribe or unsubscribe response, it should be the stream's
	// canonical message type.

	params := v.Get("params")
	if params == nil {
		var empty U
		return empty, fmt.Errorf(`field "params" is missing: %s`, string(msg.Data()))
	}
	data := params.Get("data")
	if data == nil {
		var empty U
		return empty, fmt.Errorf(`field "params.data" is missing: %s`, string(msg.Data()))
	}
	return s.parseMessage(data)
}

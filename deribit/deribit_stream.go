package deribit

import (
	"fmt"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

const (
	prodEventNodeUrl string = "wss://streams.deribit.com/ws/api/v2"
	testEventNodeUrl        = "wss://test.deribit.com/den/ws"
	prodWsUrl               = "wss://www.deribit.com/ws/api/v2"
	testWsUrl               = "wss://test.deribit.com/ws/api/v2"
)

type ConnectionType int

const (
	ProdEventNode ConnectionType = iota + 1
	TestEventNode
	Test
	Prod
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

func genId() int64 {
	return time.Now().UnixNano()
}

type subscription interface {
	channel() string
}

type stream[U any] interface {
	subscriptions() map[subscription]struct{}
	subscribeRequestIds() map[int64]struct{}
	unsubscribRequestIds() map[int64]struct{}
	parseChannel(string) (subscription, error)
	parseMessage(*fastjson.Value) (U, error)
	websocket() *websocket.Websocket
}

func parseStreamMsg[U any](s stream[U], msg websocket.Message, p fastjson.Parser) (U, error) {
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
		subUpdates := make([]subscription, len(channels))
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

// subscribeAll sends a subscription request to the stream's websocket for all of its
// subscriptions.
func subscribeAll[U any](s stream[U]) error {
	// If there are many channels, a subscribe message containing all of them may be so
	// large that the server will close the connection with a "message too big" error. To
	// prevent this, we'll split the channels into chunks and send multiple subscribe
	// messages
	channels := make([]string, 0, len(s.subscriptions()))
	for sub := range s.subscriptions() {
		channels = append(channels, sub.channel())
	}
	for _, c := range chunk[string](channels, 100) {
		id := genId()
		subMsg, err := rpcSubscribeMsg(id, c)
		if err != nil {
			return err
		}
		s.websocket().Send(subMsg)
		s.subscribeRequestIds()[id] = struct{}{}
	}

	return nil
}

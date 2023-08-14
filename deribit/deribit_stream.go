package deribit

import (
	"encoding/json"
	"fmt"
)

type ConnectionType int

const (
	prodEventNodeUrl string = "wss://streams.deribit.com/ws/api/v2"
	testEventNodeUrl        = "wss://test.deribit.com/den/ws"
	prodWsUrl               = "wss://www.deribit.com/ws/api/v2"
	testWsUrl               = "wss://test.deribit.com/ws/api/v2"
)

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

func newSubscribeMsg(channels []string) ([]byte, error) {
	m := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "public/subscribe",
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

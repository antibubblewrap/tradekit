package bybit

import (
	"bytes"
	"fmt"

	"github.com/valyala/fastjson"
)

type Market int

const (
	Spot Market = iota + 1
	Linear
	Inverse
	Private
)

const (
	spotWsUrl        string = "wss://stream.bybit.com/v5/public/spot"
	linearPerpWsUrl         = "wss://stream.bybit.com/v5/public/linear"
	inversePerpWsUrl        = "wss://stream.bybit.com/v5/public/inverse"
	optionWsUrl             = "wss://stream.bybit.com/v5/public/option"
	privateWsUrl            = "wss://stream.bybit.com/v5/private"
)

func channelUrl(market Market) (string, error) {
	switch market {
	case Spot:
		return spotWsUrl, nil
	case Linear:
		return linearPerpWsUrl, nil
	case Inverse:
		return inversePerpWsUrl, nil
	case Private:
		return privateWsUrl, nil
	default:
		return "", fmt.Errorf("invalid Bybit market type")
	}
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

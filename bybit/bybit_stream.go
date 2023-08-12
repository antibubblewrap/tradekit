package bybit

import (
	"encoding/json"
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

type subscriptionResponse struct {
	Success bool   `json:"success"`
	RetMsg  string `json:"ret_msg"`
	Op      string `json:"op"`
}

func isPingResponseMsg(data []byte) bool {
	return fastjson.GetString(data, "op") == "ping"
}

func checkSubscribeResponse(data []byte) error {
	var msg subscriptionResponse
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	if msg.Op != "subscribe" {
		return fmt.Errorf("expected subscribe response but received: %s", string(data))
	}
	if !msg.Success {
		return fmt.Errorf("subscription failed: %s", msg.RetMsg)
	}
	return nil
}

func heartbeatMsg() []byte {
	return []byte(`{"op": "ping"}`)
}

package deribit

import (
	"encoding/json"
)

// rpcMethod represents the method to be called on Deribit's JSON-RPC mechanism.
type rpcMethod string

const (
	// public methods
	methodPublicAuth        rpcMethod = "public/auth"
	methodPublicSubscribe   rpcMethod = "public/subscribe"
	methodPublicUnsubscribe rpcMethod = "public/unsubscribe"

	// private methods
	methodPrivateSubscribe           rpcMethod = "private/subscribe"
	methodPrivateUnsubscribe         rpcMethod = "private/unsubscribe"
	methodPrivateBuy                 rpcMethod = "private/buy"
	methodPrivateSell                rpcMethod = "private/sell"
	methodPrivateEdit                rpcMethod = "private/edit"
	methodPrivateCancel              rpcMethod = "private/cancel"
	methodPrivateCancelAll           rpcMethod = "private/cancel_all"
	methodPrivateCancelAllCurrency   rpcMethod = "private/cancel_all_by_currency"
	methodPrivateCancelAllInstrument rpcMethod = "private/cancel_all_by_instrument"
	methodPrivateCancelLabel         rpcMethod = "private/cancel_by_label"
	methodPrivateClosePosition       rpcMethod = "private/close_position"
)

func (m rpcMethod) String() string {
	return string(m)
}

// rpcRequestMsg creates a new request JSON-RPC request
func rpcRequestMsg(method rpcMethod, id int64, params interface{}) ([]byte, error) {
	m := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Create a new subscribe RPC request
func rpcSubscribeMsg(id int64, channels []string) ([]byte, error) {
	return rpcRequestMsg(
		methodPublicSubscribe,
		id,
		map[string]interface{}{
			"channels": channels,
		},
	)
}

// Create a new unsubscribe RPC request
func rpcUnsubscribeMsg(id int64, channels []string) ([]byte, error) {
	return rpcRequestMsg(
		methodPublicUnsubscribe,
		id,
		map[string]interface{}{
			"channels": channels,
		},
	)
}

// rpcResponse is the response from Deribit's JSON-RPC mechanism. Either through the
// websocket or plain HTTP.
type rpcResponse[T any] struct {
	Id      int    `json:"id"`
	Testnet bool   `json:"testnet"`
	Error   *Error `json:"error"`
	Result  T      `json:"result"`
}

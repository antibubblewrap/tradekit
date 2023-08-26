package deribit

import (
	"encoding/json"
)

// rpcMethod represents the method to be called on Deribit's JSON-RPC mechanism.
type rpcMethod string

const (
	// public methods
	methodPublicAuth                             rpcMethod = "public/auth"
	methodPublicSubscribe                        rpcMethod = "public/subscribe"
	methodPublicUnsubscribe                      rpcMethod = "public/unsubscribe"
	methodPublicGetInstruments                   rpcMethod = "public/get_instruments"
	methodPublicGetCurrencies                    rpcMethod = "public/get_currencies"
	methodPublicGetLastTradesByCurrency          rpcMethod = "public/get_last_trades_by_currency"
	methodPublicGetLastTradesByCurrencyAndTime   rpcMethod = "public/get_last_trades_by_currency_and_time"
	methodPublicGetLastTradesByInstrument        rpcMethod = "public/get_last_trades_by_instrument"
	methodPublicGetLastTradesByInstrumentAndTime rpcMethod = "public/get_last_trades_by_instrument_and_time"
	methodPublicGetIndexPrice                    rpcMethod = "public/get_index_price"
	methodPublicGetDeliveryPrices                rpcMethod = "public/get_delivery_prices"

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
	methodPrivateCancelByLabel       rpcMethod = "private/cancel_by_label"
	methodPrivateClosePosition       rpcMethod = "private/close_position"
	methodPrivateGetPositions        rpcMethod = "private/get_positions"
	methodPrivateGetPosition         rpcMethod = "private/get_position"
)

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

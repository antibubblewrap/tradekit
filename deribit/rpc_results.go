package deribit

import "github.com/valyala/fastjson"

// RpcResponse is the type returned by a JSON-RPC request to Deribit. If the response is
// an error, the Error field will be non-nil and the Result should be ignored.
type RpcResponse[T any] struct {
	// Id is the identifier of the request corresponding to this result.
	Id     int64  `json:"id"`
	Error  *Error `json:"error"`
	Result T      `json:"result"`
}

// BuyResult is the response of a successful submission of a buy order.
type BuyResult struct {
	// Trades are the trades which executed subsequent to submitting the order.
	Trades []TradeExecution `json:"trades"`
	// Order is the state of the order subsequent to submission.
	Order Order `json:"order"`
}

// SellResult is the response of a successful submission of a sell order.
type SellResult struct {
	// Trades are the trades which executed subsequent to submitting the order.
	Trades []TradeExecution `json:"trades"`
	// Order is the state of the order subsequent to submission.
	Order Order `json:"order"`
}

// CloseResult is the response of a successful submission of a close position request.
type CloseResult struct {
	// Trades are the trades which executed subsequent to submitting the close.
	Trades []TradeExecution `json:"trades"`
	// Order is the order that was created subsequent to submitting the close.
	Order Order `json:"order"`
}

// CloseResult is the response of a successful submission of an edit order request.
type EditResult struct {
	// Trades are the trades which executed subsequent to submitting the order edit.
	Trades []TradeExecution `json:"trades"`
	// Order is the state of the order subsequent to submission.
	Order Order `json:"order"`
}

// TradeExecution represents a trade which was executed.
type TradeExecution struct {
	TradeSeq       int64   `json:"trade_seq"`
	TradeId        string  `json:"trade_id"`
	Timestamp      int64   `json:"timestamp"`
	InstrumentName string  `json:"instrument_name"`
	Price          float64 `json:"price"`
	Amount         float64 `json:"amount"`
	Direction      string  `json:"direction"`
	Fee            float64 `json:"fee"`
	OrderId        string  `json:"order_id"`
	Liquidity      string  `json:"liquidity"`
}

// Order represents the state of an order.
type Order struct {
	InstrumentName      string      `json:"instrument_name"`
	OrderType           OrderType   `json:"order_type"`
	TimeInForce         TimeInForce `json:"time_in_force"`
	OrderState          string      `json:"order_state"`
	OrderId             string      `json:"order_id"`
	Amount              float64     `json:"amount"`
	FilledAmount        float64     `json:"filled_amount"`
	AveragePrice        float64     `json:"average_price"`
	Direction           string      `json:"direction"`
	LastUpdateTimestamp int64       `json:"last_update_timestamp"`
	ReduceOnly          bool        `json:"reduce_only"`
	PostOnly            bool        `json:"post_only"`
	MaxShow             float64     `json:"max_show"`
	Label               string      `json:"label"`
	IsLiqidation        bool        `json:"is_liquidation"`
	CreationTimestamp   int64       `json:"creation_timestamp"`
	Commission          float64     `json:"commission"`
	ProfitLoss          float64     `json:"profit_loss"`
	Price               float64     `json:"price"`
	Triggered           bool        `json:"triggered"`
	Trigger             string      `json:"trigger"`
	TriggerPrice        float64     `json:"trigger_price"`
}

func parseBuyResult(v *fastjson.Value) BuyResult {
	return BuyResult{
		Trades: parseTradeExecutions(v.GetArray("trades")),
		Order:  parseOrder(v.Get("order")),
	}
}

func parseSellResult(v *fastjson.Value) SellResult {
	return SellResult{
		Trades: parseTradeExecutions(v.GetArray("trades")),
		Order:  parseOrder(v.Get("order")),
	}
}

func parseCloseResult(v *fastjson.Value) CloseResult {
	return CloseResult{
		Trades: parseTradeExecutions(v.GetArray("trades")),
		Order:  parseOrder(v.Get("order")),
	}
}

func parseEditResult(v *fastjson.Value) EditResult {
	return EditResult{
		Trades: parseTradeExecutions(v.GetArray("trades")),
		Order:  parseOrder(v.Get("order")),
	}
}

func parseTradeExecutions(items []*fastjson.Value) []TradeExecution {
	trades := make([]TradeExecution, len(items))
	for i, item := range items {
		trades[i] = parseTradeExecution(item)
	}
	return trades
}

func parseTradeExecution(v *fastjson.Value) TradeExecution {
	return TradeExecution{
		TradeSeq:       v.GetInt64("trade_seq"),
		TradeId:        string(v.GetStringBytes("trade_id")),
		Timestamp:      v.GetInt64("timestamp"),
		InstrumentName: string(v.GetStringBytes("instrument_name")),
		Price:          v.GetFloat64("price"),
		Amount:         v.GetFloat64("amount"),
		Direction:      string(v.GetStringBytes("direction")),
		Fee:            v.GetFloat64("fee"),
		OrderId:        string(v.GetStringBytes("order_id")),
		Liquidity:      string(v.GetStringBytes("liquidity")),
	}
}

func parseOrder(v *fastjson.Value) Order {
	return Order{
		InstrumentName:      string(v.GetStringBytes("instrument_name")),
		OrderType:           OrderType(v.GetStringBytes("order_type")),
		TimeInForce:         TimeInForce(v.GetStringBytes("time_in_force")),
		OrderState:          string(v.GetStringBytes("order_state")),
		OrderId:             string(v.GetStringBytes("order_id")),
		Amount:              v.GetFloat64("amount"),
		FilledAmount:        v.GetFloat64("filled_amount"),
		AveragePrice:        v.GetFloat64("average_price"),
		Direction:           string(v.GetStringBytes("direction")),
		LastUpdateTimestamp: v.GetInt64("last_update_timestamp"),
		ReduceOnly:          v.GetBool("reduce_only"),
		PostOnly:            v.GetBool("post_only"),
		MaxShow:             v.GetFloat64("max_show"),
		Label:               string(v.GetStringBytes("label")),
		IsLiqidation:        v.GetBool("is_liquidation"),
		CreationTimestamp:   v.GetInt64("creation_timestamp"),
		Commission:          v.GetFloat64("commission"),
		ProfitLoss:          v.GetFloat64("profit_loss"),
		Price:               v.GetFloat64("price"),
		Triggered:           v.GetBool("triggered"),
		Trigger:             string(v.GetStringBytes("trigger")),
		TriggerPrice:        v.GetFloat64("trigger_price"),
	}
}

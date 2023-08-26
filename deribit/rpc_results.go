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

// OrderUpdate is the response of a successful submission of creating/editing an order or
// closing a position through one of [TradingExecutor.Buy], [TradingExecutor.Sell],
// [TradingExecutor.ClosePositionLimit], [TradingExecutor.ClosePositionMarket] or
// [TradingExecutor.EditOrder].
type OrderUpdate struct {
	// Trades are the trades which executed subsequent to submitting/editing the order.
	Trades []TradeExecution `json:"trades"`
	// Order is the state of the order subsequent to submission.
	Order Order `json:"order"`
}

// TradeExecution represents a trade which was executed.
type TradeExecution struct {
	TradeSeq       int64          `json:"trade_seq"`
	TradeId        string         `json:"trade_id"`
	Timestamp      int64          `json:"timestamp"`
	InstrumentName string         `json:"instrument_name"`
	Price          float64        `json:"price"`
	Amount         float64        `json:"amount"`
	Direction      TradeDirection `json:"direction"`
	Fee            float64        `json:"fee"`
	OrderId        string         `json:"order_id"`
	Liquidity      string         `json:"liquidity"`
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

// Position represents the state of a user's position in an instrument as returned by
// [TradeExecutor.GetPosition] or [TradeExecutor.GetPositions].
type Position struct {
	AveragePrice              float64
	AveragePriceUSD           float64
	Delta                     float64
	Direction                 string
	EstimatedLiquidationPrice float64
	FloatingProfitLoss        float64
	FloadingProfitLossUSD     float64
	Gamma                     float64
	IndexPrice                float64
	InitialMargin             float64
	InstrumentName            string
	InterestValue             float64
	Kind                      InstrumentKind
	Leverage                  int
	MaintenanceMargin         float64
	MarkPrice                 float64
	OpenOrdersMargin          float64
	RealizedFunding           float64
	RealizedProfitLoss        float64
	SettlementPrice           float64
	Size                      float64
	SizeCurrency              float64
	Theta                     float64
	TotalProfitLoss           float64
	Vega                      float64
}

func parsePosition(v *fastjson.Value) Position {
	return Position{
		AveragePrice:              v.GetFloat64("average_price"),
		AveragePriceUSD:           v.GetFloat64("average_price_usd"),
		Delta:                     v.GetFloat64("delta"),
		Direction:                 string(v.GetStringBytes("direction")),
		EstimatedLiquidationPrice: v.GetFloat64("estimated_liquidation_price"),
		FloatingProfitLoss:        v.GetFloat64("floating_profit_loss"),
		FloadingProfitLossUSD:     v.GetFloat64("floating_profit_loss_usd"),
		Gamma:                     v.GetFloat64("gamma"),
		IndexPrice:                v.GetFloat64("index_price"),
		InitialMargin:             v.GetFloat64("initial_margin"),
		InstrumentName:            string(v.GetStringBytes("instrument_name")),
		InterestValue:             v.GetFloat64("interest_value"),
		Kind:                      InstrumentKind(v.GetStringBytes("kind")),
		Leverage:                  v.GetInt("leverage"),
		MaintenanceMargin:         v.GetFloat64("maintenance_margin"),
		MarkPrice:                 v.GetFloat64("mark_price"),
		OpenOrdersMargin:          v.GetFloat64("open_orders_margin"),
		RealizedFunding:           v.GetFloat64("realized_funding"),
		RealizedProfitLoss:        v.GetFloat64("realized_profit_loss"),
		SettlementPrice:           v.GetFloat64("settlement_price"),
		Size:                      v.GetFloat64("size"),
		SizeCurrency:              v.GetFloat64("size_currency"),
		Theta:                     v.GetFloat64("theta"),
		TotalProfitLoss:           v.GetFloat64("total_profit_loss"),
		Vega:                      v.GetFloat64("vega"),
	}
}

func parsePositions(v *fastjson.Value) []Position {
	items := v.GetArray()
	positions := make([]Position, len(items))
	for i, item := range items {
		positions[i] = parsePosition(item)
	}
	return positions
}

func parseOrderUpdate(v *fastjson.Value) OrderUpdate {
	return OrderUpdate{
		Trades: parseTradeExecutions(v.Get("trades")),
		Order:  parseOrder(v.Get("order")),
	}
}

func parseTradeExecutions(v *fastjson.Value) []TradeExecution {
	items := v.GetArray()
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
		Direction:      parseDirection(v.GetStringBytes("direction")),
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

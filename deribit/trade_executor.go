package deribit

import (
	"context"
	"fmt"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

// Credentials are used to authenticate to the Deribit JSON-RPC API for accessing private
// endpoints.
type Credentials struct {
	ClientId     string
	ClientSecret string
}

// OrderOptions may be used to specify optional parameters when submitting a buy or sell
// order.
type OrderOptions struct {
	Type           OrderType
	Label          string
	Price          float64
	TimeInForce    TimeInForce
	MaxShow        float64
	PostOnly       bool
	RejectPostOnly bool
	ReduceOnly     bool
	TriggerPrice   float64
	TriggerOffset  float64
	Trigger        string
	ValidUntil     int64
}

// CancelCurrencyOptions may be used to specify optional parameters when submitting a
// cancel all orders by currency request.
type CancelCurrencyOptions struct {
	Kind InstrumentKind
	Type OrderType
}

// CancelInstrumentOptions may be used to specify optional parameters when submitting a
// cancel all orders by instrument request.
type CancelInstrumentOptions struct {
	Type OrderType
}

// CancelLabelOptions may be used to specify optional parameters when submitting a
// cancel all orders by label request.
type CancelLabelOptions struct {
	Currency string
}

// EditOrderOptions may be used to specify optional parameters when submitting an edit
// order request.
type EditOrderOptions struct {
	Price          float64
	PostOnly       bool
	RejectPostOnly bool
	ReduceOnly     bool
	TriggerPrice   float64
	TriggerOffset  float64
	ValidUntil     int64
}

// TradingExecutor allows for placing and cancelling orders through a Deribit JSON-RPC
// websocket connection. Each request takes a callback function which will be called with the
// response when it is received. You can set the callback function to nil if you wish
// to ignore the response, however, it is reccommended that you supply it so that you
// can properly handle RPC errors.
type TradingExecutor interface {
	// Start the executor. The executor must be started before any requests can be made.
	Start(ctx context.Context) error

	// Buy places a buy order for an instrument. By default, it submits a limit order,
	// unless the order type is specified in the options.
	// https://docs.deribit.com/#private-buy
	Buy(instrument string, amount float64, opts *OrderOptions, cb func(res RpcResponse[BuyResult]))

	// Sell places a sell order for an instrument. By default, it submits a limit order,
	// unless the order type is specfied in the options.
	// https://docs.deribit.com/#private-sell
	Sell(instrument string, amount float64, opts *OrderOptions, cb func(res RpcResponse[SellResult]))

	// Cancel an order with a given order ID.
	// https://docs.deribit.com/#private-cancel
	Cancel(orderId string, cb func(res RpcResponse[struct{}]))

	// CancelAll cancels all open orders. A successful response contains the number of
	// orders cancelled.
	// https://docs.deribit.com/#private-cancel_all
	CancelAll(cb func(res RpcResponse[int]))

	// CancelCurrency cancels all open orders in the provided currency. You may optionally
	// narrow the request to an instrument kind and/or order type. A successful response
	// contains the number of orders cancelled.
	// https://docs.deribit.com/#private-cancel_all_by_currency
	CancelCurrency(currency string, p *CancelCurrencyOptions, cb func(res RpcResponse[int]))

	// CancelCurrency cancels all open orders in the provided instrument. You may optionally
	// narrow the request to an order type. A successful response contains the number of
	// orders cancelled.
	// https://docs.deribit.com/#private-cancel_all_by_instrument
	CancelInstrument(instrument string, p *CancelInstrumentOptions, cb func(res RpcResponse[int]))

	// CancelLabel cancels all open orders with the provided label. You may optionally
	// narrow the request to a specific currency. A successful response contains the
	// number of orders cancelled.
	// https://docs.deribit.com/#private-cancel_by_label
	CancelLabel(label string, p *CancelLabelOptions, cb func(res RpcResponse[int]))

	// ClosePositionLimit closes an open position in a given instrument with a limit order
	// at the given price.
	ClosePositionLimit(instrument string, price float64, cb func(res RpcResponse[CloseResult]))

	// ClosePositionMarket closes an open position in a given instrument with a market
	// order.
	// https://docs.deribit.com/#private-close_position
	ClosePositionMarket(instrument string, cb func(res RpcResponse[CloseResult]))

	// EditOrder edits an open order to a new amount. Other optional modifications may be
	// specified with the optional params.
	// https://docs.deribit.com/#private-close_position
	EditOrder(orderId string, amount float64, p *EditOrderOptions, cb func(res RpcResponse[EditResult]))

	// Err returns a channel of errors. This does not include errors arising from
	// malformed RPC requests, which are included in the RpcResponse messages from the
	// Responses channel, but rather internal errors which could not be handled by the
	// executor.
	Err() <-chan error
}

func (o *OrderOptions) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if o == nil {
		return m
	}
	if o.Type != "" {
		m["type"] = string(o.Type)
	}
	if o.Label != "" {
		m["label"] = o.Label
	}
	if o.Price != 0 {
		m["price"] = o.Price
	}
	if o.TimeInForce != "" {
		m["time_in_force"] = string(o.TimeInForce)
	}
	if o.MaxShow != 0 {
		m["max_show"] = o.MaxShow
	}
	if o.PostOnly {
		m["post_only"] = o.PostOnly
	}
	if o.RejectPostOnly {
		m["reject_post_only"] = o.RejectPostOnly
	}
	if o.ReduceOnly {
		m["reduce_only"] = o.ReduceOnly
	}
	if o.TriggerPrice != 0 {
		m["trigger_price"] = o.TriggerPrice
	}
	if o.TriggerOffset != 0 {
		m["trigger_offset"] = o.TriggerOffset
	}
	if o.Trigger != "" {
		m["trigger"] = o.Trigger
	}
	if o.ValidUntil != 0 {
		m["valid_until"] = o.ValidUntil
	}
	return m
}

func (o *CancelCurrencyOptions) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if o == nil {
		return m
	}
	if o.Kind != "" {
		m["kind"] = string(o.Kind)
	}
	if o.Type != "" {
		m["type"] = string(o.Type)
	}
	return m
}

func (o *CancelInstrumentOptions) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if o == nil {
		return m
	}
	if o.Type != "" {
		m["type"] = string(o.Type)
	}
	return m
}

func (o *CancelLabelOptions) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if o == nil {
		return m
	}
	if o.Currency != "" {
		m["currency"] = o.Currency
	}
	return m
}

func (o *EditOrderOptions) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	if o == nil {
		return m
	}
	if o.Price != 0 {
		m["price"] = o.Price
	}
	if o.PostOnly {
		m["post_only"] = o.PostOnly
	}
	if o.RejectPostOnly {
		m["reject_post_only"] = o.RejectPostOnly
	}
	if o.ReduceOnly {
		m["reduce_only"] = o.ReduceOnly
	}
	if o.TriggerPrice != 0 {
		m["trigger_price"] = o.TriggerPrice
	}
	if o.TriggerOffset != 0 {
		m["trigger_offset"] = o.TriggerOffset
	}
	if o.ValidUntil != 0 {
		m["valid_until"] = o.ValidUntil
	}
	return m
}

type liveTradeExecutor struct {
	ws    *websocket.Websocket
	creds Credentials
	errc  chan error
	p     fastjson.Parser

	// Used to identify the request method when receiving responses
	requestIdMethod map[int64]rpcMethod

	// Callback functions supplied for request results
	buyCallbacks        map[int64]func(RpcResponse[BuyResult])
	sellCallbacks       map[int64]func(RpcResponse[SellResult])
	editCallbacks       map[int64]func(RpcResponse[EditResult])
	cancelCallbacks     map[int64]func(RpcResponse[struct{}])
	cancelManyCallbacks map[int64]func(RpcResponse[int])
	closeCallbacks      map[int64]func(RpcResponse[CloseResult])
}

// NewTradeExecutor creates a new Deribit TradingExecutor with the given websocket URL
// and client credentials.
func NewTradingExecutor(wsUrl string, credentials Credentials) TradingExecutor {
	ws := websocket.New(wsUrl, nil)
	ws.PingInterval = time.Second * 15

	return &liveTradeExecutor{
		ws:              &ws,
		creds:           credentials,
		errc:            make(chan error),
		requestIdMethod: make(map[int64]rpcMethod),

		buyCallbacks:        make(map[int64]func(RpcResponse[BuyResult])),
		sellCallbacks:       make(map[int64]func(RpcResponse[SellResult])),
		editCallbacks:       make(map[int64]func(RpcResponse[EditResult])),
		cancelCallbacks:     make(map[int64]func(RpcResponse[struct{}])),
		cancelManyCallbacks: make(map[int64]func(RpcResponse[int])),
		closeCallbacks:      make(map[int64]func(RpcResponse[CloseResult])),
	}
}

func isRpcError(v *fastjson.Value) *Error {
	errField := v.Get("error")
	if errField != nil {
		err := Error{
			Code:    v.GetInt("code"),
			Message: string(v.GetStringBytes("message")),
		}
		return &err
	}
	return nil
}

func (ex *liveTradeExecutor) handleResponse(msg websocket.Message) error {
	defer msg.Release()

	v, err := ex.p.ParseBytes(msg.Data())
	if err != nil {
		return fmt.Errorf("invalid JSON: %s", string(msg.Data()))
	}

	id := v.GetInt64("id")
	if id == 0 {
		return fmt.Errorf("missing request id: %s", string(msg.Data()))
	}

	method, ok := ex.requestIdMethod[id]
	if !ok {
		return fmt.Errorf("response from unknown request: %s", msg.Data())
	}
	delete(ex.requestIdMethod, id)

	// Check if it's an error response
	if err := isRpcError(v); err != nil {
		switch method {
		case methodPrivateBuy:
			ex.buyCallbacks[id](RpcResponse[BuyResult]{Error: err})
			delete(ex.buyCallbacks, id)
		case methodPrivateSell:
			ex.sellCallbacks[id](RpcResponse[SellResult]{Error: err})
			delete(ex.sellCallbacks, id)
		case methodPrivateEdit:
			ex.editCallbacks[id](RpcResponse[EditResult]{Error: err})
			delete(ex.editCallbacks, id)
		case methodPrivateCancel:
			ex.cancelCallbacks[id](RpcResponse[struct{}]{Error: err})
			delete(ex.cancelCallbacks, id)
		case methodPrivateCancelAll:
			ex.cancelManyCallbacks[id](RpcResponse[int]{Error: err})
			delete(ex.cancelManyCallbacks, id)
		case methodPrivateCancelAllCurrency:
			ex.cancelManyCallbacks[id](RpcResponse[int]{Error: err})
			delete(ex.cancelManyCallbacks, id)
		case methodPrivateCancelAllInstrument:
			ex.cancelManyCallbacks[id](RpcResponse[int]{Error: err})
			delete(ex.cancelManyCallbacks, id)
		case methodPrivateCancelLabel:
			ex.cancelManyCallbacks[id](RpcResponse[int]{Error: err})
			delete(ex.cancelManyCallbacks, id)
		case methodPrivateClosePosition:
			ex.closeCallbacks[id](RpcResponse[CloseResult]{Error: err})
			delete(ex.closeCallbacks, id)
		default:
			return fmt.Errorf("unknown method %q", method)
		}
		return nil
	}

	result := v.Get("result")
	if result == nil {
		return fmt.Errorf("missing field %q: %s", "result", msg.Data())
	}
	switch method {
	case methodPrivateBuy:
		ex.buyCallbacks[id](RpcResponse[BuyResult]{Result: parseBuyResult(result)})
		delete(ex.buyCallbacks, id)
	case methodPrivateSell:
		ex.sellCallbacks[id](RpcResponse[SellResult]{Result: parseSellResult(result)})
		delete(ex.sellCallbacks, id)
	case methodPrivateEdit:
		ex.editCallbacks[id](RpcResponse[EditResult]{Result: parseEditResult(result)})
		delete(ex.editCallbacks, id)
	case methodPrivateCancel:
		ex.cancelCallbacks[id](RpcResponse[struct{}]{Result: struct{}{}})
		delete(ex.cancelCallbacks, id)
	case methodPrivateCancelAll:
		ex.cancelManyCallbacks[id](RpcResponse[int]{Result: result.GetInt()})
		delete(ex.cancelManyCallbacks, id)
	case methodPrivateCancelAllCurrency:
		ex.cancelManyCallbacks[id](RpcResponse[int]{Result: result.GetInt()})
		delete(ex.cancelManyCallbacks, id)
	case methodPrivateCancelAllInstrument:
		ex.cancelManyCallbacks[id](RpcResponse[int]{Result: result.GetInt()})
		delete(ex.cancelManyCallbacks, id)
	case methodPrivateCancelLabel:
		ex.cancelManyCallbacks[id](RpcResponse[int]{Result: result.GetInt()})
		delete(ex.cancelManyCallbacks, id)
	case methodPrivateClosePosition:
		ex.closeCallbacks[id](RpcResponse[CloseResult]{Result: parseCloseResult(result)})
		delete(ex.closeCallbacks, id)
	default:
		return fmt.Errorf("unknown method %q", method)
	}

	return nil
}

func (ex *liveTradeExecutor) Start(ctx context.Context) error {
	ex.ws.OnConnect = func() error {
		// Authenticate and wait for response. We don't need to do anything with it,
		// just make sure it was successful.
		id := ex.authenticate()
		msg := <-ex.ws.Messages()
		defer msg.Release()
		v, err := ex.p.ParseBytes(msg.Data())
		if err != nil {
			return err
		}
		if id != v.GetInt64("id") {
			return fmt.Errorf("Deribit TradingExecutor: expected auth response but received %s", msg.Data())
		}
		delete(ex.requestIdMethod, id)
		if err := isRpcError(v); err != nil {
			return fmt.Errorf("Deribit TradingExecutor auth failure: %w", err)
		}
		return nil
	}

	if err := ex.ws.Start(ctx); err != nil {
		return fmt.Errorf("Deribit TradingExecutor: websocket connect: %w", err)
	}

	go func() {
		defer func() {
			ex.ws.Close()
			close(ex.errc)
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-ex.ws.Messages():
				if err := ex.handleResponse(data); err != nil {
					ex.errc <- fmt.Errorf("Deribit TradingExecutor: %w", err)
					return
				}
			case err := <-ex.ws.Err():
				ex.errc <- fmt.Errorf("Deribit TradingExecutor: websocket: %w", err)
				return
			}
		}
	}()
	return nil
}

func (ex *liveTradeExecutor) Buy(instrument string, amount float64, opts *OrderOptions, cb func(RpcResponse[BuyResult])) {
	id := genId()
	method := methodPrivateBuy
	params := opts.toMap()
	params["instrument"] = instrument
	params["amount"] = amount
	ex.buyCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) Sell(instrument string, amount float64, opts *OrderOptions, cb func(RpcResponse[SellResult])) {
	id := genId()
	method := methodPrivateSell
	params := opts.toMap()
	params["instrument"] = instrument
	params["amount"] = amount
	ex.sellCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) Cancel(orderId string, cb func(RpcResponse[struct{}])) {
	id := genId()
	method := methodPrivateCancel
	params := map[string]string{"order_id": orderId}
	ex.cancelCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) CancelAll(cb func(RpcResponse[int])) {
	id := genId()
	params := make(map[string]string)
	method := methodPrivateCancelAll
	ex.cancelManyCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) CancelCurrency(currency string, opts *CancelCurrencyOptions, cb func(RpcResponse[int])) {
	id := genId()
	method := methodPrivateCancelAllCurrency
	params := opts.toMap()
	params["currency"] = currency
	ex.cancelManyCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) CancelInstrument(instrument string, opts *CancelInstrumentOptions, cb func(RpcResponse[int])) {
	id := genId()
	method := methodPrivateCancelAllInstrument
	params := opts.toMap()
	params["instrument"] = instrument
	ex.cancelManyCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) CancelLabel(label string, opts *CancelLabelOptions, cb func(RpcResponse[int])) {
	id := genId()
	method := methodPrivateCancelLabel
	params := opts.toMap()
	params["label"] = label
	ex.cancelManyCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) ClosePositionLimit(instrument string, price float64, cb func(RpcResponse[CloseResult])) {
	id := genId()
	params := map[string]interface{}{
		"instrument_name": instrument,
		"type":            string(LimitOrder),
		"price":           price,
	}
	method := methodPrivateClosePosition
	ex.closeCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) ClosePositionMarket(instrument string, cb func(RpcResponse[CloseResult])) {
	id := genId()
	params := map[string]interface{}{
		"instrument_name": instrument,
		"type":            string(MarketOrder),
	}
	method := methodPrivateClosePosition
	ex.closeCallbacks[id] = cb
	ex.sendRPC(id, method, params)
	return
}

func (ex *liveTradeExecutor) EditOrder(orderId string, amount float64, opts *EditOrderOptions, cb func(res RpcResponse[EditResult])) {
	id := genId()
	method := methodPrivateEdit
	ex.editCallbacks[id] = cb
	params := opts.toMap()
	params["order_id"] = orderId
	params["amount"] = amount
	ex.sendRPC(id, method, params)
	return

}

func (ex *liveTradeExecutor) Err() <-chan error {
	return ex.errc
}

func (ex *liveTradeExecutor) sendRPC(id int64, method rpcMethod, params interface{}) error {
	msg, err := rpcRequestMsg(method, id, params)
	if err != nil {
		return err
	}
	ex.requestIdMethod[id] = method
	ex.ws.Send(msg)
	return nil
}

type authRequest struct {
	GrantType    string `json:"grant_type"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func authCredentialsRequest(creds Credentials) authRequest {
	return authRequest{
		GrantType:    "client_credentials",
		ClientId:     creds.ClientId,
		ClientSecret: creds.ClientSecret,
	}
}

func (ex *liveTradeExecutor) authenticate() (id int64) {
	id = genId()
	method := methodPublicAuth
	params := authCredentialsRequest(ex.creds)
	ex.sendRPC(id, method, params)
	return
}

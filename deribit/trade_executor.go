package deribit

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

// Credentials are used to authenticate to the Deribit JSON-RPC API to access private
// methods.
type Credentials struct {
	ClientId     string
	ClientSecret string
}

// OrderOptions may be used to specify optional parameters when submitting a buy or sell
// order with [TradingExecutor.Buy] or [TradingExecutor.Sell].
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

// CancelOrderOptions specify the options for a [TradingExecutor.CancelMany] request.
// Refer to the following for valid settings:
//   - https://docs.deribit.com/#private-cancel_all_by_currency
//   - https://docs.deribit.com/#private-cancel_all_by_instrument
//   - https://docs.deribit.com/#private-cancel_by_label
type CancelOrderOptions struct {
	Currency   string
	Kind       InstrumentKind
	Type       OrderType
	Instrument string
	Label      string
}

// EditOrderOptions specify the options for a [TradingExecutor.EditOrder] request.
type EditOrderOptions struct {
	Price          float64
	PostOnly       bool
	RejectPostOnly bool
	ReduceOnly     bool
	TriggerPrice   float64
	TriggerOffset  float64
	ValidUntil     int64
}

// TradingExecutor allows making trades through a Deribit JSON-RPC websocket connection.
// Each request takes a callback function which will be called with the response when it
// is received. You can set the callback function to nil if you wish to ignore the
// response, however, it is recommended that you supply it so that you can properly
// handle RPC errors. Requests return an error if the executor is closed.
type TradingExecutor interface {
	// Start the executor. The executor must be started before any requests can be made.
	Start(ctx context.Context) error

	// Buy places a buy order for an instrument. By default, it submits a limit order,
	// unless the order type is specified in the options. For details see:
	// https://docs.deribit.com/#private-buy
	Buy(instrument string, amount float64, opts *OrderOptions, cb func(res RpcResponse[OrderUpdate])) error

	// Sell places a sell order for an instrument. By default, it submits a limit order,
	// unless the order type is specfied in the options. For details see:
	// https://docs.deribit.com/#private-sell
	Sell(instrument string, amount float64, opts *OrderOptions, cb func(res RpcResponse[OrderUpdate])) error

	// Cancel an order with a given order ID. For details see:
	// https://docs.deribit.com/#private-cancel
	Cancel(orderId string, cb func(res RpcResponse[struct{}])) error

	// CancelMany cancels multiple orders according to the provided parameters. If the
	// options are nil, then *all* orders will be cancelled. If successful, the result
	// contains the number of orders cancelled. For details see:
	//
	//  - https://docs.deribit.com/#private-cancel_all
	//  - https://docs.deribit.com/#private-cancel_all_by_currency
	//  - https://docs.deribit.com/#private-cancel_all_by_instrument
	//  - https://docs.deribit.com/#private-cancel_by_label
	CancelMany(p *CancelOrderOptions, cb func(RpcResponse[int])) error

	// ClosePositionLimit closes an open position in the given instrument with a limit
	// reduce only order at the given price. For details see:
	// https://docs.deribit.com/#private-close_position
	ClosePositionLimit(instrument string, price float64, cb func(res RpcResponse[OrderUpdate])) error

	// ClosePositionMarket closes an open position in the given instrument with a reduce
	// only market order. For details see:
	// https://docs.deribit.com/#private-close_position
	ClosePositionMarket(instrument string, cb func(res RpcResponse[OrderUpdate])) error

	// EditOrder edits an open order to a new amount. Other modifications may be specified
	// with the optional params. For details see:
	// https://docs.deribit.com/#private-close_position
	EditOrder(orderId string, amount float64, p *EditOrderOptions, cb func(res RpcResponse[OrderUpdate])) error

	// Err returns a channel of errors. This does not include errors arising from
	// malformed RPC requests, which are included in the RpcResponse of reqeusts, but
	// rather internal errors which could not be handled by the executor. If this channel
	// produces an error you should not send any further requests.
	Err() <-chan error
}

type liveTradeExecutor struct {
	ws       *websocket.Websocket
	creds    Credentials
	errc     chan error
	p        fastjson.Parser
	isClosed bool

	// Use a mutex to prevent race conditions when storing/removing callbacks & checking
	// or setting isClosed
	m sync.Mutex

	// Used to identify the request method when receiving responses
	requestIdMethod map[int64]rpcMethod

	// Callback functions supplied for request results
	orderStateCallbacks map[int64]func(RpcResponse[OrderUpdate])
	cancelCallbacks     map[int64]func(RpcResponse[struct{}])
	cancelManyCallbacks map[int64]func(RpcResponse[int])
	positionCallbacks   map[int64]func(RpcResponse[Position])
	positionsCallbacks  map[int64]func(RpcResponse[[]Position])
}

// NewTradeExecutor creates a new Deribit TradingExecutor with the given websocket URL
// and client credentials.
func NewTradingExecutor(wsUrl string, credentials Credentials) TradingExecutor {
	ws := websocket.New(wsUrl, nil)

	return &liveTradeExecutor{
		ws:              &ws,
		creds:           credentials,
		errc:            make(chan error, 1),
		requestIdMethod: make(map[int64]rpcMethod),

		orderStateCallbacks: make(map[int64]func(RpcResponse[OrderUpdate])),
		cancelCallbacks:     make(map[int64]func(RpcResponse[struct{}])),
		cancelManyCallbacks: make(map[int64]func(RpcResponse[int])),
		positionCallbacks:   make(map[int64]func(RpcResponse[Position])),
		positionsCallbacks:  make(map[int64]func(RpcResponse[[]Position])),
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

	result := v.Get("result")
	rpcErr := isRpcError(v)

	if rpcErr == nil && result == nil {
		return fmt.Errorf("missing field %q: %s", "result", msg.Data())
	}

	if method == methodPrivateBuy ||
		method == methodPrivateSell ||
		method == methodPrivateEdit ||
		method == methodPrivateClosePosition {
		cb, ok := ex.orderStateCallbacks[id]
		if ok {
			delete(ex.orderStateCallbacks, id)
			if rpcErr != nil {
				cb(RpcResponse[OrderUpdate]{Error: rpcErr})
			} else {
				cb(RpcResponse[OrderUpdate]{Result: parseOrderUpdate(result)})
			}
		}
	} else if method == methodPrivateCancel {
		cb, ok := ex.cancelCallbacks[id]
		if ok {
			delete(ex.cancelCallbacks, id)
			if rpcErr != nil {
				cb(RpcResponse[struct{}]{Error: rpcErr})
			} else {
				cb(RpcResponse[struct{}]{Result: struct{}{}})
			}
		}
	} else if method == methodPrivateCancelAll ||
		method == methodPrivateCancelAllCurrency ||
		method == methodPrivateCancelAllInstrument ||
		method == methodPrivateCancelByLabel {
		cb, ok := ex.cancelManyCallbacks[id]
		if ok {
			delete(ex.cancelManyCallbacks, id)
			if rpcErr != nil {
				cb(RpcResponse[int]{Error: rpcErr})
			} else {
				cb(RpcResponse[int]{Result: result.GetInt()})
			}
		}
	} else if method == methodPrivateGetPositions {
		cb, ok := ex.positionsCallbacks[id]
		if ok {
			delete(ex.positionsCallbacks, id)
			if rpcErr != nil {
				cb(RpcResponse[[]Position]{Error: rpcErr})
			} else {
				cb(RpcResponse[[]Position]{Result: parsePositions(result)})
			}
		}
	} else if method == methodPrivateGetPosition {
		cb, ok := ex.positionCallbacks[id]
		if ok {
			delete(ex.positionCallbacks, id)
			if rpcErr != nil {
				cb(RpcResponse[Position]{Error: rpcErr})
			} else {
				cb(RpcResponse[Position]{Result: parsePosition(result)})
			}
		}
	} else {
		return fmt.Errorf("unknown method %q", method)
	}

	return nil
}

func (ex *liveTradeExecutor) Start(ctx context.Context) error {
	ex.ws.OnConnect = func() error {
		ex.m.Lock()
		defer ex.m.Unlock()

		// Authenticate and wait for response. We don't need to do anything with it,
		// just make sure it was successful.
		id, err := ex.authenticate()
		if err != nil {
			return tradingExErr(fmt.Errorf("auth failure: %w", err))
		}
		msg := <-ex.ws.Messages()
		defer msg.Release()
		v, err := ex.p.ParseBytes(msg.Data())
		if err != nil {
			return err
		}
		if id != v.GetInt64("id") {
			return tradingExErr(fmt.Errorf("expected auth response but received: %s", msg.Data()))
		}
		delete(ex.requestIdMethod, id)
		if err := isRpcError(v); err != nil {
			return tradingExErr(fmt.Errorf("auth failure: %w", err))
		}
		return nil
	}

	if err := ex.ws.Start(ctx); err != nil {
		return tradingExErr(fmt.Errorf("websocket connect: %w", err))
	}

	go func() {
		defer func() {
			ex.isClosed = true
			ex.ws.Close()
			close(ex.errc)
			ex.m.Unlock()
		}()
		for {
			ex.m.Lock()
			select {
			case <-ctx.Done():
				return
			case data := <-ex.ws.Messages():
				if err := ex.handleResponse(data); err != nil {
					ex.errc <- tradingExErr(err)
					return
				}
			case err := <-ex.ws.Err():
				ex.errc <- tradingExErr(fmt.Errorf("websocket: %w", err))
				return
			}
			ex.m.Unlock()
		}
	}()
	return nil
}

func (ex *liveTradeExecutor) Buy(instrument string, amount float64, opts *OrderOptions, cb func(RpcResponse[OrderUpdate])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted Buy but executor is closed"))
	}
	id := genId()
	method := methodPrivateBuy
	params := opts.params()
	params["instrument_name"] = instrument
	params["amount"] = amount
	if cb != nil {
		ex.orderStateCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
}

func (ex *liveTradeExecutor) Sell(instrument string, amount float64, opts *OrderOptions, cb func(RpcResponse[OrderUpdate])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted Sell but executor is closed"))
	}
	id := genId()
	method := methodPrivateSell
	params := opts.params()
	params["instrument_name"] = instrument
	params["amount"] = amount
	if cb != nil {
		ex.orderStateCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
}

func (ex *liveTradeExecutor) Cancel(orderId string, cb func(RpcResponse[struct{}])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted Cancel but executor is closed"))
	}
	id := genId()
	method := methodPrivateCancel
	params := map[string]string{"order_id": orderId}
	if cb != nil {
		ex.cancelCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
}

func (ex *liveTradeExecutor) CancelMany(opts *CancelOrderOptions, cb func(RpcResponse[int])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted CancelMany but executor is closed"))
	}
	id := genId()
	method, params := opts.methodAndParams()
	if cb != nil {
		ex.cancelManyCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
}

func (ex *liveTradeExecutor) ClosePositionLimit(instrument string, price float64, cb func(RpcResponse[OrderUpdate])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted ClosePositionLimit but executor is closed"))
	}
	id := genId()
	params := map[string]interface{}{
		"instrument_name": instrument,
		"type":            string(LimitOrder),
		"price":           price,
	}
	method := methodPrivateClosePosition
	if cb != nil {
		ex.orderStateCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
}

func (ex *liveTradeExecutor) ClosePositionMarket(instrument string, cb func(RpcResponse[OrderUpdate])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted ClosePositionMarket but executor is closed"))
	}
	id := genId()
	params := map[string]interface{}{
		"instrument_name": instrument,
		"type":            string(MarketOrder),
	}
	method := methodPrivateClosePosition
	if cb != nil {
		ex.orderStateCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
}

func (ex *liveTradeExecutor) EditOrder(orderId string, amount float64, opts *EditOrderOptions, cb func(res RpcResponse[OrderUpdate])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted EditOrder but executor is closed"))
	}
	id := genId()
	method := methodPrivateEdit
	params := opts.params()
	params["order_id"] = orderId
	params["amount"] = amount
	if cb != nil {
		ex.orderStateCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
}

type GetPositionsOptions struct {
	Kind InstrumentKind
}

func (ex *liveTradeExecutor) GetPositions(currency string, opts *GetPositionsOptions, cb func(res RpcResponse[[]Position])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted GetPositions but executor is closed"))
	}
	id := genId()
	method := methodPrivateGetPositions
	params := opts.params()
	params["currency"] = currency
	if cb != nil {
		ex.positionsCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
}

func (ex *liveTradeExecutor) GetPosition(instrument string, cb func(RpcResponse[Position])) error {
	ex.m.Lock()
	defer ex.m.Unlock()
	if ex.isClosed {
		return tradingExErr(errors.New("attempted GetPosition but executor is closed"))
	}
	id := genId()
	method := methodPrivateGetPosition
	params := map[string]string{"instrument_name": instrument}
	if cb != nil {
		ex.positionCallbacks[id] = cb
	}
	ex.sendRPC(id, method, params)
	return nil
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

func (ex *liveTradeExecutor) authenticate() (id int64, err error) {
	id = genId()
	method := methodPublicAuth
	params := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     ex.creds.ClientId,
		"client_secret": ex.creds.ClientSecret,
	}
	err = ex.sendRPC(id, method, params)
	return
}

func tradingExErr(err error) error {
	return fmt.Errorf("Deribit TradingExecutor: %w", err)
}

func (o *OrderOptions) params() map[string]interface{} {
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

func (o *EditOrderOptions) params() map[string]interface{} {
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

func (o *CancelOrderOptions) methodAndParams() (rpcMethod, map[string]interface{}) {
	var method rpcMethod
	params := make(map[string]interface{})
	if o == nil {
		method = methodPrivateCancelAll
	} else {
		if o.Label != "" {
			method = methodPrivateCancelByLabel
			params["label"] = o.Label
			if o.Currency != "" {
				params["currency"] = o.Currency
			}
		} else if o.Instrument != "" {
			method = methodPrivateCancelAllInstrument
			params["instrument_name"] = o.Instrument
			if o.Type != "" {
				params["type"] = o.Type
			}
		} else if o.Currency != "" {
			method = methodPrivateCancelAllCurrency
			params["currency"] = o.Currency
			if o.Kind != "" {
				params["kind"] = o.Kind
			}
			if o.Type != "" {
				params["type"] = o.Type
			}
		} else {
			method = methodPrivateCancelAll
		}
	}
	return method, params
}

func (o *GetPositionsOptions) params() map[string]interface{} {
	params := make(map[string]interface{})
	if o == nil {
		return params
	}
	if o.Kind != "" {
		params["kind"] = o.Kind
	}
	return params
}

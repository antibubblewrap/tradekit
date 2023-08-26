package deribit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Error is returned by requests to the Deribit API in the event that the request
// was not successful. For details see https://docs.deribit.com/#json-rpc
type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("Deribit RPC error [%d]: %s", e.Code, e.Message)
}

// Iterator allows for iteration over paginated methods on the Deribit API.
type Iterator[T any] interface {
	// Done returns true when there are no more results in the iterator.
	Done() bool
	// Next returns the next batch of items from the iterator. You should stop calling
	// Next after Done returns true.
	Next() (T, error)
}

// Api allows for sending requests to the Deribit JSON-RPC API over HTTP.
type Api struct {
	baseUrl *url.URL
}

// NewApi creates a new Api to either the prod or testing Deribit server.
func NewApi(apiUrl string) (*Api, error) {
	baseUrl, err := url.Parse(apiUrl)
	if err != nil {
		return nil, err
	}
	return &Api{baseUrl: baseUrl}, nil
}

// Option is the type of a Deribit option instrument.
// For more details see https://docs.deribit.com/#public-get_instrument
type Option struct {
	Name                     string         `json:"instrument_name"`
	Strike                   float64        `json:"strike"`
	OptionType               string         `json:"option_type"`
	IsActive                 bool           `json:"is_active"`
	ExpirationTimestamp      int64          `json:"expiration_timestamp"`
	CreationTimestamp        int64          `json:"creation_timestamp"`
	BaseCurrency             string         `json:"base_currency"`
	QuoteCurrency            string         `json:"quote_currency"`
	CounterCurrency          string         `json:"counter_currency"`
	SettlementCurrency       string         `json:"settlement_currency"`
	TickSize                 float64        `json:"tick_size"`
	TickSizeSteps            []TickSizeStep `json:"tick_size_steps"`
	TakerCommission          float64        `json:"taker_commission"`
	MakerCommission          float64        `json:"maker_commission"`
	SettlementPeriod         string         `json:"settlement_period"`
	RFQ                      bool           `json:"rfq"`
	PriceIndex               string         `json:"price_index"`
	MinTradeAmount           float64        `json:"min_trade_amount"`
	InstrumentId             int64          `json:"instrument_id"`
	ContractSize             float64        `json:"contract_size"`
	BlockTradeCommission     float64        `json:"block_trade_commission,omitempty"`
	BlockTradeMinTradeAmount float64        `json:"block_trade_min_trade_amount,omitempty"`
	BlockTradeTickSize       float64        `json:"block_trade_tick_size,omitempty"`
}

type TickSizeStep struct {
	AbovePrice float64 `json:"above_price"`
	TickSize   float64 `json:"tick_size"`
}

func apiGet[T any](api *Api, method rpcMethod, params map[string]string) (T, error) {
	u := urlWithParams(api.baseUrl, method, params)

	r, err := http.Get(u)
	if err != nil {
		var zero T
		return zero, apiErr(method, err)
	}
	defer r.Body.Close()

	var resp RpcResponse[T]
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		var zero T
		return zero, apiErr(method, err)
	}
	if resp.Error != nil {
		var zero T
		return zero, apiErr(method, *resp.Error)
	}

	return resp.Result, nil
}

// GetOptionInstruments retrieves all Deribit option instruments on the given currency.
// Set expired to true to show recently expired options instead of active ones.
func (api *Api) GetOptionInstruments(currency string, expired bool) ([]Option, error) {
	exp := "false"
	if expired {
		exp = "true"
	}
	params := map[string]string{"currency": currency, "kind": "option", "expired": exp}
	return apiGet[[]Option](api, methodPublicGetInstruments, params)
}

// CurrencyInfo is the type returned from the Deribit /public/get_currencies endpoint.
// For details see https://docs.deribit.com/#public-get_currencies
type CurrencyInfo struct {
	CoinType         string  `json:"coin_type"`
	Currency         string  `json:"currency"`
	CurrencyLong     string  `json:"currency_long"`
	FeePrecision     int     `json:"fee_precision"`
	MinConfirmations int     `json:"min_confirmations"`
	WithdrawalFee    float64 `json:"withdrawal_fee"`
}

// GetCurrencies returns a slice of the supported currencies on Deribit from the
// /public/get_currencies endpoint.
// For details see https://docs.deribit.com/#public-get_currencies
func (api *Api) GetCurrencies() ([]CurrencyInfo, error) {
	return apiGet[[]CurrencyInfo](api, methodPublicGetCurrencies, nil)
}

type DeliveryPrice struct {
	Date  string  `json:"date"`
	Price float64 `json:"delivery_price"`
}

type deliveryPrices struct {
	Prices       []DeliveryPrice `json:"data"`
	RecordsTotal int             `json:"records_total"`
}

type OptionsGetDeliveryPrices struct {
	Count int
}

type deliveryPricesIterator struct {
	done      bool
	api       *Api
	count     int
	offset    int
	indexName string
}

func (it *deliveryPricesIterator) Done() bool {
	return it.done
}

func (it *deliveryPricesIterator) Next() ([]DeliveryPrice, error) {
	method := methodPublicGetDeliveryPrices
	params := map[string]string{"index_name": it.indexName, "offset": strconv.Itoa(it.offset), "count": strconv.Itoa(it.count)}
	resp, err := apiGet[deliveryPrices](it.api, method, params)
	if err != nil {
		return nil, apiErr(method, err)
	}

	if len(resp.Prices) == 0 {
		it.done = true
	} else {
		it.offset += len(resp.Prices)
	}

	return resp.Prices, nil
}

// GetDeliveryPrices returns an iterator over delivery prices for a given index.
// For more details see: https://docs.deribit.com/#public-get_delivery_prices. Results
// are returned in descending order from the most recent delivery price.
func (api *Api) GetDeliveryPrices(indexName string, p *OptionsGetDeliveryPrices) Iterator[[]DeliveryPrice] {
	count := 10
	if p != nil && p.Count != 0 {
		count = p.Count
	}
	return &deliveryPricesIterator{api: api, count: count, indexName: indexName}
}

type IndexPrice struct {
	EstimatedDeliveryPrice float64 `json:"estimated_delivery_price"`
	IndexPrice             float64 `json:"index_price"`
}

// GetIndexPrice returns the current price of a given index from the /public/get_index_price
// endpoint.
func (api *Api) GetIndexPrice(indexName string) (IndexPrice, error) {
	params := map[string]string{"index_name": indexName}
	return apiGet[IndexPrice](api, methodPublicGetIndexPrice, params)
}

// GetTradesOptions define optional parameters for retrieving trades.
type GetTradesOptions struct {
	// StartTimestamp and EndTimestamp define the timeframe over which trades will be
	// returned. If StartTimestamp is specfied then trades will be returned in ascending
	// order, otherwise they will be returned in descending order.
	StartTimestamp time.Time
	EndTimestamp   time.Time

	// Count defines the number of trades to return per pagination request. If unspecified,
	// it defaults to 10
	Count int

	// We set these internally to know what method to use.
	currency   string
	kind       InstrumentKind
	instrument string

	// We use these for pagination.
	startTradeId  string
	endTradeId    string
	startSequence int
	endSequence   int
}

func (p GetTradesOptions) methodAndParams() (rpcMethod, map[string]string, error) {
	params := make(map[string]string)
	var method rpcMethod
	if p.currency != "" {
		params["currency"] = p.currency
		if p.startTradeId != "" || p.endTradeId != "" {
			method = methodPublicGetLastTradesByCurrency
			if p.startTradeId != "" {
				params["start_id"] = p.startTradeId
			} else {
				params["end_id"] = p.endTradeId
			}
		} else {
			if !p.StartTimestamp.IsZero() && !p.EndTimestamp.IsZero() {
				method = methodPublicGetLastTradesByCurrencyAndTime
			} else {
				method = methodPublicGetLastTradesByCurrency
			}
		}
		if p.kind != "" {
			params["kind"] = string(p.kind)
		}
	} else if p.instrument != "" {
		params["instrument_name"] = p.instrument
		if p.startSequence != 0 || p.endSequence != 0 {
			method = methodPublicGetLastTradesByInstrument
			if p.startSequence != 0 {
				params["start_seq"] = strconv.Itoa(p.startSequence)
			} else {
				params["end_seq"] = strconv.Itoa(p.endSequence)
			}
		} else {
			if !p.StartTimestamp.IsZero() && !p.EndTimestamp.IsZero() {
				method = methodPublicGetLastTradesByInstrumentAndTime
			} else {
				method = methodPublicGetLastTradesByInstrument
			}
		}
	} else {
		panic("GetTradesOptions methodAndParams unreachable!")
	}

	if p.isIterDescending() {
		params["sorting"] = "desc"
	} else {
		params["sorting"] = "asc"
	}

	if !p.StartTimestamp.IsZero() && p.startSequence == 0 && p.startTradeId == "" {
		params["start_timestamp"] = strconv.Itoa(int(p.StartTimestamp.UnixMilli()))
	}
	if !p.EndTimestamp.IsZero() {
		params["end_timestamp"] = strconv.Itoa(int(p.EndTimestamp.UnixMilli()))
	}

	if p.Count != 0 {
		params["count"] = strconv.Itoa(p.Count)
	} else {
		params["count"] = "10"
	}

	return method, params, nil
}

func (p GetTradesOptions) isIterDescending() bool {
	return p.StartTimestamp.IsZero()
}

type getTradesResponse struct {
	Trades  []PublicTrade `json:"trades"`
	HasMore bool          `json:"has_more"`
}

type tradesIterator struct {
	done      bool
	params    GetTradesOptions
	api       *Api
	doneFirst bool
}

func (it *tradesIterator) Next() ([]PublicTrade, error) {
	method, params, err := it.params.methodAndParams()
	if err != nil {
		it.done = true
		return nil, err
	}
	resp, err := apiGet[getTradesResponse](it.api, method, params)
	if err != nil {
		it.done = true
		return nil, apiErr(method, err)
	}

	trades := resp.Trades
	if it.doneFirst {
		// After the first request, we skip the first trade. This is because subsequent
		// requests set either the start trade Id, or start sequence (depending it we're
		// querying the currency or instrument), and we need to exclude it, otherwise
		// the first trade will be a duplicate
		trades = trades[1:]
	}

	if len(trades) == 0 {
		it.done = true
		return trades, nil
	}

	if resp.HasMore {
		t := trades[len(trades)-1]
		if it.params.isIterDescending() {
			if it.params.currency != "" {
				it.params.endTradeId = t.TradeId
			} else if it.params.instrument != "" {
				it.params.endSequence = int(t.TradeSeq)
			}
		} else {
			if it.params.currency != "" {
				it.params.startTradeId = t.TradeId
			} else if it.params.instrument != "" {
				it.params.startSequence = int(t.TradeSeq)
			}
		}
	} else {
		it.done = true
	}

	it.doneFirst = true

	return trades, nil
}

func (it *tradesIterator) Done() bool {
	return it.done
}

func (api *Api) GetLastTradesByCurrencyAndKind(currency string, kind InstrumentKind, opts *GetTradesOptions) Iterator[[]PublicTrade] {
	return api.getLastTrades("", currency, kind, opts)
}

func (api *Api) GetLastTradesByInstrument(instrument string, opts *GetTradesOptions) Iterator[[]PublicTrade] {
	return api.getLastTrades(instrument, "", "", opts)
}

func (api *Api) getLastTrades(instrument string, currency string, kind InstrumentKind, opts *GetTradesOptions) Iterator[[]PublicTrade] {
	var options GetTradesOptions
	if opts == nil {
		options.Count = 10
	} else {
		if opts.Count != 0 {
			options.Count = 10
		}
		if !opts.StartTimestamp.IsZero() {
			options.StartTimestamp = opts.StartTimestamp
		}
		if !opts.EndTimestamp.IsZero() {
			options.EndTimestamp = opts.EndTimestamp
		}
	}
	if instrument != "" {
		options.instrument = instrument
	} else {
		options.currency = currency
		options.kind = kind
	}
	return &tradesIterator{api: api, params: options}
}

func urlWithParams(baseUrl *url.URL, method rpcMethod, params map[string]string) string {
	u := baseUrl.JoinPath(string(method))
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	u.RawQuery = values.Encode()
	return u.String()
}

func apiErr(method rpcMethod, err error) error {
	return fmt.Errorf("Deribit API error %s: %w", method, err)
}

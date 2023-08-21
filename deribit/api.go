package deribit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	prodApiUrl = "https://www.deribit.com/api/v2"
	testApiUrl = "https://test.deribit.com/api/v2"

	endpointGetInstruments string = "/public/get_instruments"
	endpointGetCurrencies         = "/public/get_currencies"
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

// Iterator allows for iteration over paginated endpoints on the Deribit API.
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
func NewApi(conn ConnectionType) *Api {
	var u string
	if conn == Prod || conn == ProdEventNode {
		u = prodApiUrl
	} else {
		u = testApiUrl
	}
	baseUrl, err := url.Parse(u)
	if err != nil {
		panic(err) // Panic is okay because the url strings are constants
	}

	return &Api{baseUrl: baseUrl}
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

func apiGet[T any](api *Api, endpoint string, params map[string]string) (T, error) {
	u := urlWithParams(api.baseUrl, endpoint, params)

	r, err := http.Get(u)
	if err != nil {
		var zero T
		return zero, apiErr(endpoint, err)
	}
	defer r.Body.Close()

	var resp rpcResponse[T]
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		var zero T
		return zero, apiErr(endpoint, err)
	}
	if resp.Error != nil {
		var zero T
		return zero, apiErr(endpoint, *resp.Error)
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
	return apiGet[[]Option](api, endpointGetInstruments, params)
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
	return apiGet[[]CurrencyInfo](api, endpointGetCurrencies, nil)
}

type DeliveryPrice struct {
	Date  string  `json:"date"`
	Price float64 `json:"delivery_price"`
}

type deliveryPrices struct {
	Prices       []DeliveryPrice `json:"data"`
	RecordsTotal int             `json:"records_total"`
}

type GetDeliveryPricesParams struct {
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
	endpoint := "/public/get_delivery_prices"
	params := map[string]string{"index_name": it.indexName, "offset": strconv.Itoa(it.offset), "count": strconv.Itoa(it.count)}
	resp, err := apiGet[deliveryPrices](it.api, endpoint, params)
	if err != nil {
		return nil, apiErr(endpoint, err)
	}

	if len(resp.Prices) == 0 {
		it.done = true
	}

	it.offset += len(resp.Prices)
	return resp.Prices, nil
}

// GetDeliveryPrices returns an iterator over delivery prices for a given index.
// For more details see: https://docs.deribit.com/#public-get_delivery_prices. Results
// are returned in descending order from the most recent delivery price.
func (api *Api) GetDeliveryPrices(indexName string, p *GetDeliveryPricesParams) Iterator[[]DeliveryPrice] {
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
	endpoint := "/public/get_index_price"
	params := map[string]string{"index_name": indexName}
	return apiGet[IndexPrice](api, endpoint, params)
}

// GetTradesParams defines the query parameters for the GetLastTrades method on an Api.
// Either Currency or Instrument must be specified. If both are specified, then Instrument
// is ignored.
//
// Kind defines the type of instrument to return trades for. It is ignored if Instrument
// is specified. Valid values are "future", "option", "spot", "future_combo",
// "option_combo" or "any". If unspecified, "any" will be set.
//
// StartTimestamp and EndTimestamp are optional, and define the timeframe, over which
// trades will be returned. The timestamps must be specified in Unix milliseconds. If
// StartTimestamp is specified, then trades will be returned in ascending order, otherwise
// trades are returned in descending order.
//
// Count defines the number of trades to return per pagination request. If unspecified,
// it defaults to 10.
type GetTradesParams struct {
	Currency       string
	Kind           InstrumentKind
	Instrument     string
	StartTimestamp int64
	EndTimestamp   int64
	Count          int

	// We use these for pagination. Even if the user specifies the timestamps,
	startTradeId  string
	endTradeId    string
	startSequence int
	endSequence   int
}

func (p GetTradesParams) endpointAndParams() (string, map[string]string, error) {
	params := make(map[string]string)
	var endpoint string
	if p.Currency != "" {
		params["currency"] = p.Currency
		if p.startTradeId != "" || p.endTradeId != "" {
			endpoint = "/public/get_last_trades_by_currency"
			if p.startTradeId != "" {
				params["start_id"] = p.startTradeId
			} else {
				params["end_id"] = p.endTradeId
			}
		} else {
			if p.StartTimestamp != 0 && p.EndTimestamp != 0 {
				endpoint = "/public/get_last_trades_by_currency_and_time"
			} else {
				endpoint = "/public/get_last_trades_by_currency"
			}
		}
		if p.Kind != "" {
			params["kind"] = string(p.Kind)
		} else {
			params["kind"] = string(AnyInstrument)
		}
	} else if p.Instrument != "" {
		params["instrument_name"] = p.Instrument
		if p.startSequence != 0 || p.endSequence != 0 {
			endpoint = "/public/get_last_trades_by_instrument"
			if p.startSequence != 0 {
				params["start_seq"] = strconv.Itoa(p.startSequence)
			} else {
				params["end_seq"] = strconv.Itoa(p.endSequence)
			}
		} else {
			if p.StartTimestamp != 0 && p.EndTimestamp != 0 {
				endpoint = "/public/get_last_trades_by_instrument_and_time"
			} else {
				endpoint = "/public/get_last_trades_by_instrument"
			}
		}
	} else {
		return "", nil, fmt.Errorf("invalid GetTradesParams: either Currency or Instrument must be specified")
	}

	if p.StartTimestamp != 0 {
		params["sorting"] = "asc"
	} else {
		params["sorting"] = "desc"
	}

	if p.StartTimestamp != 0 && p.startSequence == 0 && p.startTradeId == "" {
		params["start_timestamp"] = strconv.Itoa(int(p.StartTimestamp))
	}
	if p.EndTimestamp != 0 {
		params["end_timestamp"] = strconv.Itoa(int(p.EndTimestamp))
	}

	if p.Count != 0 {
		params["count"] = strconv.Itoa(p.Count)
	} else {
		params["count"] = "10"
	}

	return endpoint, params, nil
}

func (p GetTradesParams) isIterDescending() bool {
	if p.StartTimestamp != 0 {
		return false
	}
	return true
}

// Trade is the type returned by the GetLastTrades iterator.
type Trade struct {
	TradeSeq      int64   `json:"trade_seq"`
	TradeId       string  `json:"trade_id"`
	Timestamp     int64   `json:"timestamp"`
	TickDirection int     `json:"tick_direction"`
	Price         float64 `json:"price"`
	MarkPrice     float64 `json:"mark_price"`
	Instrument    string  `json:"instrument_name"`
	IndexPrice    float64 `json:"index_price"`
	Direction     string  `json:"direction"`
	Amount        float64 `json:"amount"`
}

type getTradesResponse struct {
	Trades  []Trade `json:"trades"`
	HasMore bool    `json:"has_more"`
}

type tradesIterator struct {
	done       bool
	params     GetTradesParams
	api        *Api
	descending bool
	doneFirst  bool
}

func (it *tradesIterator) Next() ([]Trade, error) {
	endpoint, params, err := it.params.endpointAndParams()
	fmt.Println("----")
	fmt.Printf("it.params = %+v\n", it.params)
	fmt.Printf("endpoint = %s\n", endpoint)
	fmt.Printf("params = %+v\n", params)
	if err != nil {
		it.done = true
		return nil, err
	}
	resp, err := apiGet[getTradesResponse](it.api, endpoint, params)
	if err != nil {
		it.done = true
		return nil, apiErr(endpoint, err)
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
		if it.descending {
			if it.params.Currency != "" {
				it.params.endTradeId = t.TradeId
			} else {
				it.params.endSequence = int(t.TradeSeq)
			}
		} else {
			if it.params.Currency != "" {
				it.params.startTradeId = t.TradeId
			} else {
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

// GetLastTrades returns an iterator over the last trades for a given currency or
// instrument, as specified by the provided params. The iterator returns trades in
// ascending order.
func (api *Api) GetLastTrades(p GetTradesParams) Iterator[[]Trade] {
	descending := p.isIterDescending()
	return &tradesIterator{api: api, params: p, descending: descending}
}

func urlWithParams(baseUrl *url.URL, path string, params map[string]string) string {
	u := baseUrl.JoinPath(path)
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	u.RawQuery = values.Encode()
	return u.String()
}

func apiErr(endpoint string, err error) error {
	return fmt.Errorf("Deribit API error %s: %w", endpoint, err)
}

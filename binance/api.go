package binance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/antibubblewrap/tradekit"
	"github.com/valyala/fastjson"
)

type Market int

const (
	Spot = iota + 1
	Perpetual
	InversePerpetual
)

// Api provides access to the Binance HTTP API for spot, perpetual and inverse perpetual
// markets.
type Api struct {
	baseUrl *url.URL
	market  Market
	pools   map[string]*fastjson.ParserPool
}

// Error is the type returned when the Binance API responds with an error.
type Error struct {
	// HttpCode is the HTTP status code of the request.
	HttpCode int `json:"-"`
	// Code is the Binance error code.
	Code int `json:"code"`
	// Msg is a description of the error.
	Msg string `json:"msg"`
}

func (e Error) Error() string {
	return fmt.Sprintf("Binance API error (%d): %s [HTTP %d]", e.Code, e.Msg, e.HttpCode)
}

// NewApi creates a new Binance Api. The provided Market, should match the baseUrl.
func NewApi(baseUrl string, market Market) (*Api, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid apiUrl: %s", baseUrl)
	}
	pools := make(map[string]*fastjson.ParserPool)
	return &Api{baseUrl: u, market: market, pools: pools}, nil
}

// getParser returns a JSON parser for the provided endpoint.
func (api *Api) getParser(endpoint string) *fastjson.Parser {
	if pool, ok := api.pools[endpoint]; ok {
		return pool.Get()
	} else {
		var pool fastjson.ParserPool
		parser := pool.Get()
		api.pools[endpoint] = &pool
		return parser
	}
}

// returnParser returns a parser that was retrieved using getParser so that it can be
// reused for subsequent requests to the endpoint.
func (api *Api) returnParser(endpoint string, parser *fastjson.Parser) {
	api.pools[endpoint].Put(parser)
}

// OrderbookResponse is the type returned by GetOrderbook.
type OrderbookResponse struct {
	LastUpdateId int64            `json:"lastUpdateId"`
	EventTime    int64            `json:"E"`
	Bids         []tradekit.Level `json:"bids"`
	Asks         []tradekit.Level `json:"asks"`
}

// GetOrderbook gets a snapshot of an orderbook from the Binance API for a given
// symbol and depth limit.
func (api *Api) GetOrderbook(symbol string, limit int) (OrderbookResponse, error) {
	var endpoint string
	switch api.market {
	case Spot:
		endpoint = "/api/v3/depth"
	case Perpetual:
		endpoint = "/fapi/v1/depth"
	case InversePerpetual:
		endpoint = "/dapi/v1/depth"
	}

	return apiGet[OrderbookResponse](api, endpoint, map[string]string{"symbol": symbol, "limit": strconv.Itoa(limit)})
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
	return fmt.Errorf("Binance API error %s: %w", endpoint, err)
}

func apiGet[T any](api *Api, endpoint string, params map[string]string) (T, error) {
	u := urlWithParams(api.baseUrl, endpoint, params)

	r, err := http.Get(u)
	if err != nil {
		var zero T
		return zero, apiErr(endpoint, err)
	}
	defer r.Body.Close()

	if r.StatusCode < 300 {
		var resp T
		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			var zero T
			return zero, apiErr(endpoint, err)
		}
		return resp, nil
	} else {
		var zero T
		var e Error
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			return zero, apiErr(endpoint, err)
		}
		e.HttpCode = r.StatusCode
		return zero, apiErr(endpoint, e)
	}
}

# Crypto TradeKit

A collection of utiltities for cryptocurrency trading with Go. This package is a work
in progress.

```
go get -u github.com/antibubblewrap/tradekit
```

## Features 

  - A high performance, cache-efficient, level 2 order book implementation. An update
    to an orderbook takes __~300ns__, on average. The backing datastructure is based on
    a custom cache-efficient ordered array map, storing price levels in a collection of
    contiguous fixed size slices.
  - Orderbook metrics — spread, liquidity, market impact. More metrics will be added in
    the future. Custom metrics may be efficient implemented with the `book.IterBids()` and
    `book.IterAsks()` methods.
  - Websocket connections with low to zero memory allocation overhead.
  - Bybit market data stream connections. Compatible with the `Orderbook` data type.
  - Deribit market data stream connections. Compatible with the `Orderbook` data type.
  - Stats: exponential moving average, rolling sums, etc.


## Examples

See [cmd/main.go](./cmd/main.go)



# Crypto TradeKit

[![Go Reference](https://pkg.go.dev/badge/github.com/antibubblewrap/tradekit.svg)](https://pkg.go.dev/github.com/antibubblewrap/tradekit)

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
  - Streaming and API connections to Binance, Bybit and Deribit.
  - Stats: exponential moving average, rolling sums, etc.

## Bybit Features

See [`github.com/antibubblewrap/tradekit/bybit`](https://pkg.go.dev/github.com/antibubblewrap/tradekit/bybit) for all features.

  - Market data streams (spot, perpetual futures & inverse perpetual futures)
      1. `TradeStream`: a realtime stream of trades
      2. `OrderbookStream`: stream of incremental orderbook updates. Compatible with the 
         `tradekit.Orderbook`. Updates at 10ms-100ms depending on the level.

## Deribit Features

See [`github.com/antibubblewrap/tradekit/deribit`](https://pkg.go.dev/github.com/antibubblewrap/tradekit/deribit) for all features.

  - Market data streams (spot, futures & options)
    1. `NewTradesStream`: a realtime stream of trades.
    2. `NewOrderbookStream`: a realtime stream of incremental orderbook updates. Compatible
       with the `tradekit.Orderbook`.
    3. `NewOrderbookDepthStream`: a stream of orderbook snapshots at a given depth. Updates
       at 100ms intervals.
    4. `NewInstrumentStateStream`: a stream of updates about instrument states (created, closed etc.)
  - HTTP API (spot, futures & options)
    1. `GetOptionInstruments`: returns all Option instruments in a given base currency.
    2. `GetCurrencies`: returns all information on all supported currencies.
    3. `GetDeliveryPrices`: returns delivery prices on an index for options / futures. 
    4. `GetIndexPrice`: returns the current price of an index.
    5. `GetLastTrades`: returns past trades for a given currency / instrument.
  - Private APIs:
    1. `TradingExecutor`: a connector to the Deribit private trading API over a websocket.
       It may be used to place, edit & cancel orders, and close positions.
    2. `NewUserTradesStream`: a realtime stream of private trade executions.
    3. `NewUserOrdersStream`: a realtime stream of private order updates


## Binance Features

See [`github.com/antibubblewrap/tradekit/binance`](https://pkg.go.dev/github.com/antibubblewrap/tradekit/binance) for all features.

  - Market data streams (spot, USD-M perpetual futures & COIN-M inverse perpetual futures)
    1. `TradeStream`: a realtime stream of trades.
    2. `AggTradeStream`: an aggregated trade stream. Updates on a 100ms interval.
    3. `OrderbookStream`: a stream of incremental orderbook updates. Updates on a 100ms
       interval. Compatible with the `tradekit.Orderbook`.
  - HTTP API (spot, USD-M perpetual futures & COIN-M inverse perpetual futures)
    1. `GetOrderbook`: returns a snapshot of an orderbook.


## Examples

See [cmd/main.go](./cmd/main.go)



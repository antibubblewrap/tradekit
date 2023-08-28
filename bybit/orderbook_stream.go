package bybit

import (
	"fmt"
)

// OrderbookSub represents a subscription to the orderbook stream of trading symbol.
// See [NewOrderbookStream].
type OrderbookSub struct {
	Symbol string
	Depth  int
}

func (s OrderbookSub) channel() string {
	return fmt.Sprintf("orderbook.%d.%s", s.Depth, s.Symbol)
}

// NewLiquidationStream returns a stream of orderbook updates. For details see:
//   - https://bybit-exchange.github.io/docs/v5/websocket/public/orderbook
func NewOrderbookStream(wsUrl string, subs ...OrderbookSub) Stream[OrderbookUpdate] {
	subscriptions := make([]subscription, len(subs))
	for i, sub := range subs {
		subscriptions[i] = sub
	}
	return newStream[OrderbookUpdate](wsUrl, "OrderbookStream", parseOrderbookUpdate, subscriptions)
}

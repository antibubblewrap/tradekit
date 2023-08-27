package bybit

import (
	"fmt"
)

type OrderbookSub struct {
	Symbol string
	Depth  int
}

func (s OrderbookSub) channel() string {
	return fmt.Sprintf("orderbook.%d.%s", s.Depth, s.Symbol)
}

// Create a new stream to the Bybit orderbook websocket stream for the given symbol and
// market depth.
func NewOrderbookStream(wsUrl string, subs ...OrderbookSub) Stream[OrderbookUpdate] {
	subscriptions := make([]subscription, len(subs))
	for i, sub := range subs {
		subscriptions[i] = sub
	}
	return newStream[OrderbookUpdate](wsUrl, "OrderbookStream", parseOrderbookUpdate, subscriptions)
}

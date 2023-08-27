package bybit

import (
	"fmt"
)

type TradesSub struct {
	Symbol string
}

func (s TradesSub) channel() string {
	return fmt.Sprintf("publicTrade.%s", s.Symbol)
}

func NewTradesStream(wsUrl string, subs ...TradesSub) Stream[Trades] {
	subscriptions := make([]subscription, len(subs))
	for i, sub := range subs {
		subscriptions[i] = sub
	}
	return newStream[Trades](wsUrl, "TradesStream", parseTrades, subscriptions)
}

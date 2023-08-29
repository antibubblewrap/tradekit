package bybit

import (
	"fmt"
)

// LiquidationSub represents a subscription to a liquidation stream. See [NewLiquidationStream]
type LiquidationSub struct {
	Symbol string
}

func (s LiquidationSub) channel() string {
	return fmt.Sprintf("liquidation.%s", s.Symbol)
}

// NewLiquidationStream returns a stream of liquidations. For details see:
//   - https://bybit-exchange.github.io/docs/v5/websocket/public/liquidation
func NewLiquidationStream(wsUrl string, subs ...LiquidationSub) Stream[Liquidation] {
	subscriptions := make([]subscription, len(subs))
	for i, sub := range subs {
		subscriptions[i] = sub
	}
	return newStream[Liquidation](wsUrl, "LiquidationStream", parseLiquidation, subscriptions)
}

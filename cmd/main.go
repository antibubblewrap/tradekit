package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/bybit"
	"github.com/antibubblewrap/tradekit/deribit"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Create a streams to the Bybit BTC trades and orderbook
	bybitUrl := "wss://stream.bybit.com/v5/public/linear"
	bybitBookStream := bybit.NewOrderbookStream(bybitUrl, bybit.OrderbookSub{Symbol: "BTCUSDT", Depth: 200})
	bybitTradeStream := bybit.NewTradesStream(bybitUrl, bybit.TradesSub{Symbol: "BTCUSDT"})

	// Create streams to the Deribit BTC trades and orderbook
	deribitUrl := "wss://streams.deribit.com/ws/api/v2"
	sub1 := deribit.TradesSub{Instrument: "BTC-PERPETUAL"}
	sub2 := deribit.OrderbookSub{Instrument: "BTC-PERPETUAL"}
	deribitTradeStream := deribit.NewTradesStream(deribitUrl, sub1)
	deribitBookStream := deribit.NewOrderbookStream(deribitUrl, sub2)

	// Start the streams
	if err := deribitTradeStream.Start(ctx); err != nil {
		panic(err)
	}
	if err := deribitBookStream.Start(ctx); err != nil {
		panic(err)
	}
	if err := bybitBookStream.Start(ctx); err != nil {
		panic(err)
	}
	if err := bybitTradeStream.Start(ctx); err != nil {
		panic(err)
	}

	// Gather some metrics
	bybit1mVolume := tradekit.NewRollingSum(time.Minute)
	deribitMidPriceEWMA := tradekit.NewEWMA(time.Minute)

	// Maintain a local order book for Bybit and Deribit
	bybitBook := tradekit.NewOrderbook(nil, nil)
	deribitBook := tradekit.NewOrderbook(nil, nil)

	// Read the streams
	n := 0
	elapsed := 0
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-bybitBookStream.Messages():
			if msg.Type == "snapshot" {
				bybitBook.UpdateSnapshot(msg.Data.Bids, msg.Data.Asks)
			} else {
				start := time.Now()
				bybitBook.UpdateBids(msg.Data.Bids)
				bybitBook.UpdateAsks(msg.Data.Asks)
				n += len(msg.Data.Asks) + len(msg.Data.Bids)
				elapsed += int(time.Since(start).Nanoseconds())
			}
			if n >= 1000 {
				avg := elapsed / n
				fmt.Printf("Bybit avg book update speed = %dns (%d)\n", avg, n)
				elapsed = 0
				n = 0
			}
		case msg := <-bybitTradeStream.Messages():
			for _, trade := range msg.Data {
				bybit1mVolume.Update(trade.Amount)
			}
		case msg := <-deribitBookStream.Messages():
			if msg.Type == "snapshot" {
				deribitBook.UpdateSnapshot(msg.Bids, msg.Asks)
			} else {
				deribitBook.UpdateBids(msg.Bids)
				deribitBook.UpdateAsks(msg.Asks)
			}
			deribitMidPriceEWMA.Update(deribitBook.MidPrice(), msg.Timestamp)
		case _ = <-deribitTradeStream.Messages():
			// do something with the trade ...
		case <-ticker.C:
			fmt.Printf("Deribit midprice 1 min EWMA = %.2f\n", deribitMidPriceEWMA.Value)
			fmt.Printf("Bybit 1 min rolling volume = %.2f\n", bybit1mVolume.Value())
		case err := <-bybitBookStream.Err():
			panic(err)
		case err := <-bybitTradeStream.Err():
			panic(err)
		case err := <-deribitBookStream.Err():
			panic(err)
		case err := <-deribitTradeStream.Err():
			panic(err)
		}
	}
}

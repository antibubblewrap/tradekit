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

func startBybitOrderbookStream(ctx context.Context, wsUrl string, symbol string, depth int) *bybit.OrderbookStream {
	stream := bybit.NewOrderbookStream(wsUrl, depth, symbol)
	if err := stream.Start(ctx); err != nil {
		panic(err)
	}
	fmt.Println("ByBit book stream connected")
	return stream
}

func startBybitTradeStream(ctx context.Context, wsUrl, symbol string) *bybit.TradeStream {
	stream := bybit.NewTradeStream(wsUrl, symbol)
	if err := stream.Start(ctx); err != nil {
		panic(err)
	}
	fmt.Println("Bybit trade stream connected")
	return stream
}

func startDeribitOrderbookStream(ctx context.Context, instrument string) *deribit.OrderbookStream {
	sub := deribit.OrderbookSub{Instrument: instrument, Freq: deribit.UpdateRaw}
	wsUrl := "wss://streams.deribit.com/ws/api/v2"
	stream, err := deribit.NewOrderbookStream(wsUrl, nil, sub)
	if err != nil {
		panic(err)
	}
	if err := stream.Start(ctx); err != nil {
		panic(err)
	}
	fmt.Println("Deribit order book stream connected")
	return stream
}

func startDeribitTradeStream(ctx context.Context, instrument string) *deribit.TradeStream {
	sub := deribit.TradeSub{Instrument: instrument, Freq: deribit.UpdateRaw}
	wsUrl := "wss://streams.deribit.com/ws/api/v2"
	stream, err := deribit.NewTradeStream(wsUrl, nil, sub)
	if err != nil {
		panic(err)
	}
	if err := stream.Start(ctx); err != nil {
		panic(err)
	}
	fmt.Println("Deribit trade stream connected")
	return stream
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	bybitUrl := "wss://stream.bybit.com/v5/public/linear"
	bybitBookStream := startBybitOrderbookStream(ctx, bybitUrl, "BTCUSDT", 200)
	bybitTradeStream := startBybitTradeStream(ctx, bybitUrl, "BTCUSDT")
	bybit1mVolume := tradekit.NewRollingSum(time.Minute)

	deribitBookStream := startDeribitOrderbookStream(ctx, "BTC-PERPETUAL")
	deribitTradeStream := startDeribitTradeStream(ctx, "BTC-PERPETUAL")
	deribitMidPriceEWMA := tradekit.NewEWMA(time.Minute)

	bybitBook := tradekit.NewOrderbook(nil, nil)
	deribitBook := tradekit.NewOrderbook(nil, nil)

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
			// Do something with the trade message
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

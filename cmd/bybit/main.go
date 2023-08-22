package main

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/signal"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/bybit"
)

type bookUpdate struct {
	Type string           `json:"type"`
	Bids []tradekit.Level `json:"bids"`
	Asks []tradekit.Level `json:"asks"`
}

func main() {
	// Connects to the ByBit order book stream for a symbol and outputs the messages to
	// a file "bybit_orderbook_stream.jsonl".

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	bybitUrl := "wss://stream.bybit.com/v5/public/linear"
	bybitBookStream := bybit.NewOrderbookStream(bybitUrl, 200, "BTCUSDT")
	if err := bybitBookStream.Start(ctx); err != nil {
		panic(err)
	}

	f, err := os.Create("bybit_orderbook_stream.jsonl")
	if err != nil {
		panic(err)
	}
	fb := bufio.NewWriter(f)
	defer func() {
		fb.Flush()
		f.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-bybitBookStream.Messages():
			m := bookUpdate{msg.Type, msg.Data.Bids, msg.Data.Asks}
			data, err := json.Marshal(m)
			if err != nil {
				panic(err)
			}
			if _, err := fb.Write(data); err != nil {
				panic(err)
			}
			fb.WriteString("\n")
		case err := <-bybitBookStream.Err():
			panic(err)
		}
	}

}

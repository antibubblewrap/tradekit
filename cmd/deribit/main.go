package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/antibubblewrap/tradekit/deribit"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	api, err := deribit.NewApi("https://www.deribit.com/api/v2/")
	if err != nil {
		panic(err)
	}

	options, err := api.GetOptionInstruments("BTC", false)
	if err != nil {
		panic(err)
	}

	// Create orderbook depth stream for all active options
	bookSubs := make([]deribit.OrderbookDepthSub, len(options))
	for i, inst := range options {
		bookSubs[i] = deribit.OrderbookDepthSub{Instrument: inst.Name, Depth: 20}
	}
	bookStream := deribit.NewOrderbookDepthStream("wss://streams.deribit.com/ws/api/v2", bookSubs...)

	if err := bookStream.Start(ctx); err != nil {
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-bookStream.Messages():
			fmt.Printf("%+v\n", msg)
		case err := <-bookStream.Err():
			panic(err)
		}
	}

}

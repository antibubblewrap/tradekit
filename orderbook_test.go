package tradekit

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

var (
	testBook *Orderbook = NewOrderbook(
		[]Level{{1.6, 0.4}, {1.4, 1.0}, {1.1, 0.2}},
		[]Level{{1.7, 2.3}, {1.8, 0.1}, {2.1, 2.1}, {2.3, 11.9}},
	)
)

func TestOrderBook(t *testing.T) {
	book := NewOrderbook([]Level{}, []Level{})
	assert.Equal(t, Level{}, book.BestBid())
	assert.Equal(t, Level{}, book.BestAsk())
	assert.Equal(t, []Level{}, book.Bids())
	assert.Equal(t, []Level{}, book.Asks())
	assert.True(t, math.IsInf(book.Spread(), 1))

	initBids := []Level{{1.1, 0.2}, {1.3, 0.5}, {1.4, 1.2}}
	initAsks := []Level{{1.8, 0.1}, {2.0, 3.0}, {2.1, 2.1}}
	book.UpdateSnapshot(initBids, initAsks)

	slices.Reverse(initBids)
	assert.Equal(t, initBids, book.Bids())
	assert.Equal(t, initAsks, book.Asks())

	book.UpdateBid(1.6, 0.4)
	book.UpdateBid(1.3, 0)
	book.UpdateBid(1.4, 1.0)
	assert.Equal(t, []Level{{1.6, 0.4}, {1.4, 1.0}, {1.1, 0.2}}, book.Bids())
	assert.Equal(t, Level{1.6, 0.4}, book.BestBid())

	book.UpdateAsk(1.8, 0.0)
	book.UpdateAsk(1.7, 2.3)
	book.UpdateAsk(2.0, 0.0)
	book.UpdateAsk(2.3, 11.9)
	book.UpdateAsk(1.8, 0.1)
	assert.Equal(t, []Level{{1.7, 2.3}, {1.8, 0.1}, {2.1, 2.1}, {2.3, 11.9}}, book.Asks())
	assert.Equal(t, Level{1.7, 2.3}, book.BestAsk())

	assert.InEpsilon(t, 2.3+0.1+2.1+11.9, book.AskLiquidity(), 1e-6)
	assert.InEpsilon(t, 0.4+1.0+0.2, book.BidLiquidity(), 1e-6)
}

func TestMarketImpact(t *testing.T) {
	price, rem := testBook.BuyMarketImpact(2.0)
	assert.InEpsilon(t, 1.7, price, 1e-6)
	assert.Zero(t, rem)

	price, rem = testBook.BuyMarketImpact(2.5)
	assert.InEpsilon(t, 1.72, price, 1e-6)
	assert.Zero(t, rem)

	price, rem = testBook.SellMarketImpact(1.2)
	assert.InEpsilon(t, 1.46666666666, price, 1e-6)
	assert.Zero(t, rem)

	price, rem = testBook.SellMarketImpact(1000)
	assert.NotZero(t, rem)
}

func TestSpread(t *testing.T) {
	assert.InEpsilon(t, 0.1, testBook.Spread(), 1e-6)
}

type bookUpdate struct {
	Type string  `json:"type"`
	Bids []Level `json:"bids"`
	Asks []Level `json:"asks"`
}

func updateOrderbook(msgs []bookUpdate) {
	book := NewOrderbook(nil, nil)
	for _, msg := range msgs {
		if msg.Type == "snapshot" {
			book.UpdateSnapshot(msg.Bids, msg.Asks)
		} else {
			book.UpdateBids(msg.Bids)
			book.UpdateAsks(msg.Asks)
		}
	}
}

// NOTE: run ./cmd/bybit to get a sample of orderbook messages
func BenchmarkOrderbook(b *testing.B) {
	f, err := os.Open("bybit_orderbook_stream.jsonl")
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)
	bookMsgs := make([]bookUpdate, 0)
	for scanner.Scan() {
		b := scanner.Bytes()
		var msg bookUpdate
		if err := json.Unmarshal(b, &msg); err != nil {
			panic(err)
		}
		bookMsgs = append(bookMsgs, msg)
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		updateOrderbook(bookMsgs)
	}
}

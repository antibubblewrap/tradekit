package tradekit

import (
	"encoding/json"
	"math"
	"strconv"

	"github.com/antibubblewrap/tradekit/internal/arraymap"
)

// Implementation note: we store levels with a negative price in the bids ArrayMap of
// an Orderbook. This is so that we can retrieve bids in the natural reverse sorted order.
// Also, storing bids in this way is slightly more peformant as it's assumed that most
// updates and retrievals will be close to the inside of the book. Make sure that the
// bid prices are negated before returning them to the consumer of any orderbook method.

const chunkSize = 32

// An Orderbook is used to maintain a level-2 orderbook storing price levels for bids and
// asks. The orderbook can be updated in a streaming fashion using the UpdateAsk and
// UpdateBid methods. It is *not* safe to make concurrent calls an Orderbook.
//
// Internally, Orderbook uses a high-performance, cache-efficient, ordered map
// implementation for storing bids and asks.
type Orderbook struct {
	bids *arraymap.ArrayMap[float64, float64]
	asks *arraymap.ArrayMap[float64, float64]
}

// Level stores the price and amount for a level in the bid or ask side of an Orderbook.
type Level struct {
	Price  float64
	Amount float64
}

func (l *Level) UnmarshalJSON(data []byte) error {
	var s [2]string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	price, err := strconv.ParseFloat(s[0], 64)
	if err != nil {
		return err
	}
	amount, err := strconv.ParseFloat(s[1], 64)
	if err != nil {
		return err
	}
	l.Price = price
	l.Amount = amount
	return nil
}

func (l *Level) MarshalJSON() ([]byte, error) {
	price := strconv.FormatFloat(l.Price, 'f', -1, 64)
	amount := strconv.FormatFloat(l.Amount, 'f', -1, 64)
	s := [2]string{price, amount}
	return json.Marshal(s)
}

func NewOrderbook(bids []Level, asks []Level) *Orderbook {
	bidMap := arraymap.New[float64, float64](chunkSize)
	if bids != nil {
		for _, bid := range bids {
			bidMap.Insert(-bid.Price, bid.Amount)
		}
	}

	askMap := arraymap.New[float64, float64](chunkSize)
	if asks != nil {
		for _, ask := range asks {
			askMap.Insert(ask.Price, ask.Amount)
		}
	}

	return &Orderbook{
		bids: bidMap,
		asks: askMap,
	}
}

// UpdateSnapshot overwrites the orderbook with a snapshot of bid and ask levels.
func (ob *Orderbook) UpdateSnapshot(bids []Level, asks []Level) {
	bidMap := arraymap.New[float64, float64](chunkSize)
	for _, bid := range bids {
		bidMap.Insert(-bid.Price, bid.Amount)
	}

	askMap := arraymap.New[float64, float64](chunkSize)
	for _, ask := range asks {
		askMap.Insert(ask.Price, ask.Amount)
	}

	ob.bids = bidMap
	ob.asks = askMap
}

// UpdateBid inserts / updates a bid level in the order book. If the amount is zero then
// it removes the price level.
func (ob *Orderbook) UpdateBid(price, amount float64) {
	if amount == 0.0 {
		ob.bids.Delete(-price)
	} else {
		ob.bids.Insert(-price, amount)
	}
}

// UpdateAsk inserts / updates an ask level in the order book. If the amount is zero then
// it removes the price level.
func (ob *Orderbook) UpdateAsk(price, amount float64) {
	if amount == 0.0 {
		ob.asks.Delete(price)
	} else {
		ob.asks.Insert(price, amount)
	}
}

// UpdateBids updates the orderbook with a batch of bid updates. As with UpdateBid, a
// level with a zero amount means deletion.
func (ob *Orderbook) UpdateBids(levels []Level) {
	for _, lvl := range levels {
		ob.UpdateBid(lvl.Price, lvl.Amount)
	}
}

// UpdateAsks updates the orderbook with a batch of ask updates. As with UpdateAsk, a
// level with zero amount means deletion.
func (ob *Orderbook) UpdateAsks(levels []Level) {
	for _, lvl := range levels {
		ob.UpdateAsk(lvl.Price, lvl.Amount)
	}
}

// Bids returns a slice of all bid levels in the order book.
func (ob *Orderbook) Bids() []Level {
	levels := make([]Level, 0, ob.bids.Len())
	it := ob.bids.Iter()
	for {
		e, ok := it.Next()
		if !ok {
			break
		}
		levels = append(levels, Level{Price: -e.Key, Amount: e.Value})
	}
	return levels
}

// Asks returns a slice of all ask levels in the order book.
func (ob *Orderbook) Asks() []Level {
	levels := make([]Level, 0, ob.asks.Len())
	it := ob.asks.Iter()
	for {
		e, ok := it.Next()
		if !ok {
			break
		}
		levels = append(levels, Level{Price: e.Key, Amount: e.Value})
	}
	return levels
}

// BestBid returns the best bid (highest price) in the orderbook. Returns the zero level
// if the bid side of the book is empty.
func (ob *Orderbook) BestBid() Level {
	first, ok := ob.bids.First()
	if !ok {
		return Level{}
	}
	return Level{Price: -first.Key, Amount: first.Value}
}

// BestAsk returns the best ask (lowest price) in the orderbook. Returns the zero level
// if the ask side of the book is empty.
func (ob *Orderbook) BestAsk() Level {
	first, ok := ob.asks.First()
	if !ok {
		return Level{}
	}
	return Level{Price: first.Key, Amount: first.Value}
}

// Spread returns the price difference between the best bid and the best ask in the
// order book. If either side of the book is empty, it returns +infinity.
func (ob *Orderbook) Spread() float64 {
	bestBid := ob.BestBid()
	bestAsk := ob.BestAsk()
	if bestBid.Price == 0 || bestAsk.Price == 0 {
		return math.Inf(1)
	}
	return bestAsk.Price - bestBid.Price
}

// IterLevels represents an iterator of the price levels of either the ask or bid side
// of an OrderBook. To construct an iterator, call IterBids or IterAsks on an Orderbook.
type IterLevels struct {
	it *arraymap.IterMap[float64, float64]
	f  func(arraymap.Entry[float64, float64]) Level
}

// Next returns the next price level in a levels iterator. The return value is a tuple
// containing the next price level and a boolean indicating if the iterator has ended.
func (it *IterLevels) Next() (Level, bool) {
	e, ok := it.it.Next()
	if !ok {
		return Level{}, false
	}
	return it.f(e), true
}

func bidIterFn(e arraymap.Entry[float64, float64]) Level {
	return Level{Price: -e.Key, Amount: e.Value}
}

func askIterFn(e arraymap.Entry[float64, float64]) Level {
	return Level{Price: e.Key, Amount: e.Value}
}

// IterBids returns an iterator over the bid side of the book starting from the best bid.
func (ob *Orderbook) IterBids() *IterLevels {
	return &IterLevels{it: ob.bids.Iter(), f: bidIterFn}
}

// IterAsks returns an iterator over the ask side of the book starting from the best ask.
func (ob *Orderbook) IterAsks() *IterLevels {
	return &IterLevels{it: ob.asks.Iter(), f: askIterFn}
}

func liquidity(it *IterLevels) float64 {
	var liquidity float64
	for {
		level, ok := it.Next()
		if !ok {
			break
		}
		liquidity += level.Amount
	}
	return liquidity
}

// BidLiquidity returns the amount of liquidity in the bid side of the orderbook
func (ob *Orderbook) BidLiquidity() float64 {
	return liquidity(ob.IterBids())
}

// AskLiquidity returns the amount of liquidity in the ask side of the orderbook
func (ob *Orderbook) AskLiquidity() float64 {
	return liquidity(ob.IterAsks())
}

func marketImpact(it *IterLevels, amount float64) (float64, float64) {
	remAmount := amount
	var num float64
	for {
		level, ok := it.Next()
		if !ok {
			break
		}
		if remAmount <= level.Amount {
			num += level.Price * remAmount
			remAmount = 0
			break
		} else {
			num += level.Price * level.Amount
			remAmount -= level.Amount
		}
	}
	return num / amount, remAmount
}

// BuyMarketImpact returns the volume weighted average price of a buy taker order,
// followed by any remaining part of the order if there is not enough liquidity. If the
// order could be filled completely, the remainder is zero.
func (ob *Orderbook) BuyMarketImpact(amount float64) (float64, float64) {
	return marketImpact(ob.IterAsks(), amount)
}

// SellMarketImpact returns the volume weighted average price of a sell taker order,
// followed by any remaining part of the order if there is not enough liquidity. If the
// order could be filled completely, the remainder is zero.
func (ob *Orderbook) SellMarketImpact(amount float64) (float64, float64) {
	return marketImpact(ob.IterBids(), amount)
}

// MidPrice returns the mid-price between the best bid and the best ask.
func (ob *Orderbook) MidPrice() float64 {
	bestAsk := ob.BestAsk()
	bestBid := ob.BestBid()
	return (bestBid.Price + bestAsk.Price) / 2.0
}

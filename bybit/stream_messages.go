package bybit

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/antibubblewrap/tradekit"
	"github.com/valyala/fastjson"
)

type OrderbookUpdate struct {
	Topic     string              `json:"topic"`
	Type      string              `json:"type"`
	Timestamp int64               `json:"ts"`
	Data      OrderbookUpdateData `json:"data"`
}

type OrderbookUpdateData struct {
	Symbol   string           `json:"s"`
	Bids     []tradekit.Level `json:"b"`
	Asks     []tradekit.Level `json:"a"`
	UpdateID int64            `json:"u"`
	Sequence int64            `json:"seq"`
}

type Trades struct {
	Topic     string  `json:"topic"`
	Type      string  `json:"type"`
	Timestamp int64   `json:"ts"`
	Data      []Trade `json:"data"`
}

type Trade struct {
	FilledTimestamp int64          `json:"T"`
	Symbol          string         `json:"s"`
	Direction       TradeDirection `json:"S"`
	Price           float64        `json:"p"`
	Amount          float64        `json:"v"`
	TradeID         string         `json:"i"`
	BlockTrade      bool           `json:"BT"`
}

func parsePriceLevel(v *fastjson.Value) (tradekit.Level, error) {
	priceS := string(v.GetStringBytes("0"))
	price, err := strconv.ParseFloat(priceS, 64)
	if err != nil {
		return tradekit.Level{}, fmt.Errorf("invalid level price %q", priceS)

	}
	amountS := string(v.GetStringBytes("1"))
	amount, err := strconv.ParseFloat(amountS, 64)
	if err != nil {
		return tradekit.Level{}, fmt.Errorf("invalid level amount %q", amountS)
	}
	return tradekit.Level{Price: price, Amount: amount}, nil
}

func parsePriceLevels(items []*fastjson.Value) ([]tradekit.Level, error) {
	levels := make([]tradekit.Level, len(items))
	for i, v := range items {
		level, err := parsePriceLevel(v)
		if err != nil {
			return nil, err
		}
		levels[i] = level
	}
	return levels, nil
}

func parseOrderbookUpdateData(v *fastjson.Value) (OrderbookUpdateData, error) {
	bids, err := parsePriceLevels(v.GetArray("b"))
	if err != nil {
		return OrderbookUpdateData{}, fmt.Errorf("parsing field %q: %w", "b", err)
	}

	asks, err := parsePriceLevels(v.GetArray("a"))
	if err != nil {
		return OrderbookUpdateData{}, fmt.Errorf("parsing field %q: %w", "a", err)
	}

	return OrderbookUpdateData{
		Symbol:   string(v.GetStringBytes("s")),
		Bids:     bids,
		Asks:     asks,
		UpdateID: v.GetInt64("u"),
		Sequence: v.GetInt64("seq"),
	}, nil

}

func parseOrderbookUpdate(v *fastjson.Value) (OrderbookUpdate, error) {
	data := v.Get("data")
	if data == nil {
		return OrderbookUpdate{}, fmt.Errorf("missing field %q", "data")
	}

	odata, err := parseOrderbookUpdateData(data)
	if err != nil {
		return OrderbookUpdate{}, err
	}

	return OrderbookUpdate{
		Topic:     string(v.GetStringBytes("topic")),
		Type:      string(v.GetStringBytes("type")),
		Timestamp: v.GetInt64("ts"),
		Data:      odata,
	}, nil
}

func parseTradeDirection(b []byte) TradeDirection {
	var d TradeDirection
	if bytes.Equal(b, []byte("Buy")) {
		d = Buy
	} else {
		d = Sell
	}
	return d
}

func parseTrade(v *fastjson.Value) (Trade, error) {
	price, err := strconv.ParseFloat(string(v.GetStringBytes("p")), 64)
	if err != nil {
		return Trade{}, fmt.Errorf("invalid trade price: %w", err)
	}

	amount, err := strconv.ParseFloat(string(v.GetStringBytes("v")), 64)
	if err != nil {
		return Trade{}, fmt.Errorf("invalid trade amount: %w", err)
	}

	return Trade{
		FilledTimestamp: v.GetInt64("T"),
		Symbol:          string(v.GetStringBytes("s")),
		Direction:       parseTradeDirection(v.GetStringBytes("S")),
		Price:           price,
		Amount:          amount,
		TradeID:         string(v.GetStringBytes("i")),
		BlockTrade:      v.GetBool("BT"),
	}, nil

}

func parseTrades(v *fastjson.Value) (Trades, error) {
	data := v.GetArray("data")
	if data == nil {
		return Trades{}, fmt.Errorf("missing field %q", "data")
	}

	trades := make([]Trade, len(data))
	for i, item := range data {
		trade, err := parseTrade(item)
		if err != nil {
			return Trades{}, err
		}
		trades[i] = trade
	}

	return Trades{
		Topic:     string(v.GetStringBytes("topic")),
		Type:      string(v.GetStringBytes("type")),
		Timestamp: v.GetInt64("ts"),
		Data:      trades,
	}, nil
}

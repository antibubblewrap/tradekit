package deribit

import (
	"bytes"

	"github.com/antibubblewrap/tradekit"
	"github.com/valyala/fastjson"
)

type PublicTrade struct {
	TradeSeq      int64          `json:"trade_seq"`
	Timestamp     int64          `json:"timestamp"`
	Instrument    string         `json:"instrument_name"`
	Price         float64        `json:"price"`
	Amount        float64        `json:"amount"`
	Direction     TradeDirection `json:"direction"`
	TickDirection int            `json:"tick_direction"`
	TradeId       string         `json:"trade_id"`
	Liquidation   string         `json:"liquidation"`
}

// InstrumentState is the type of message produced by the InstrumentStateStream.
// For details, see: https://docs.deribit.com/#instrument-state-kind-currency
type InstrumentState struct {
	Timestamp  int64  `json:"timestamp"`
	State      string `json:"state"`
	Instrument string `json:"instrument_name"`
}

type OrderbookDepth struct {
	Timestamp  int64            `json:"timestamp"`
	Instrument string           `json:"instrument_name"`
	ChangeID   int64            `json:"change_id"`
	Bids       []tradekit.Level `json:"bids"`
	Asks       []tradekit.Level `json:"asks"`
}

// OrderbookMsg is the message type streamed from the Deribit book channel.
// For more info see: https://docs.deribit.com/#book-instrument_name-interval
type OrderbookUpdate struct {
	// The type of orderbook update. Either "snapshot" or "change".
	Type         string
	Timestamp    int64
	Instrument   string
	ChangeID     int64
	PrevChangeID int64
	Bids         []tradekit.Level
	Asks         []tradekit.Level
}

func parseOrderbookUpdate(v *fastjson.Value) OrderbookUpdate {
	return OrderbookUpdate{
		Type:         string(v.GetStringBytes("type")),
		Timestamp:    v.GetInt64("timestamp"),
		Instrument:   string(v.GetStringBytes("instrument_name")),
		ChangeID:     v.GetInt64("change_id"),
		PrevChangeID: v.GetInt64("prev_change_id"),
		Bids:         parseOrderbookLevels(v.GetArray("bids")),
		Asks:         parseOrderbookLevels(v.GetArray("asks")),
	}
}

func parseOrderbookLevel(v *fastjson.Value) tradekit.Level {
	action := v.GetStringBytes("0")
	price := v.GetFloat64("1")
	amount := v.GetFloat64("2")
	if bytes.Equal(action, []byte("delete")) {
		amount = 0
	}
	return tradekit.Level{Price: price, Amount: amount}
}

func parseOrderbookLevels(items []*fastjson.Value) []tradekit.Level {
	levels := make([]tradekit.Level, len(items))
	for i, item := range items {
		levels[i] = parseOrderbookLevel(item)
	}
	return levels
}

func parseDirection(v []byte) TradeDirection {
	if string(v) == "buy" {
		return Buy
	} else {
		return Sell
	}
}

func parsePublicTrade(v *fastjson.Value) PublicTrade {
	return PublicTrade{
		TradeSeq:      v.GetInt64("trade_seq"),
		Timestamp:     v.GetInt64("timestamp"),
		Instrument:    string(v.GetStringBytes("instrument_name")),
		Price:         v.GetFloat64("price"),
		Amount:        v.GetFloat64("amount"),
		Direction:     parseDirection(v.GetStringBytes("direction")),
		TickDirection: v.GetInt("tick_direction"),
		TradeId:       string(v.GetStringBytes("trade_id")),
		Liquidation:   string(v.GetStringBytes("liquidation")),
	}
}

func parsePublicTrades(v *fastjson.Value) []PublicTrade {
	items := v.GetArray()
	trades := make([]PublicTrade, len(items))
	for i, item := range items {
		trades[i] = parsePublicTrade(item)
	}
	return trades
}

func parseInstrumentState(v *fastjson.Value) InstrumentState {
	return InstrumentState{
		Timestamp:  v.GetInt64("timestamp"),
		State:      string(v.GetStringBytes("state")),
		Instrument: string(v.GetStringBytes("instrument_name")),
	}
}

func parseOrderbookDepth(v *fastjson.Value) OrderbookDepth {
	return OrderbookDepth{
		Timestamp:  v.GetInt64("timestamp"),
		Instrument: string(v.GetStringBytes("instrument_name")),
		ChangeID:   v.GetInt64("change_id"),
		Bids:       parsePriceLevels(v.GetArray("bids")),
		Asks:       parsePriceLevels(v.GetArray("asks")),
	}
}

func parsePriceLevel(v *fastjson.Value) tradekit.Level {
	price := v.GetFloat64("0")
	amount := v.GetFloat64("1")
	return tradekit.Level{Price: price, Amount: amount}
}

func parsePriceLevels(items []*fastjson.Value) []tradekit.Level {
	levels := make([]tradekit.Level, len(items))
	for i, v := range items {
		levels[i] = parsePriceLevel(v)
	}
	return levels
}

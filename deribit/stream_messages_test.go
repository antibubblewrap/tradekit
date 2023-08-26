package deribit

import (
	"testing"

	"github.com/antibubblewrap/tradekit"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fastjson"
)

func TestParsePublicTrade(t *testing.T) {
	input := `
	{
      "trade_seq" : 30289442,
      "trade_id" : "48079269",
      "timestamp" : 1590484512188,
      "tick_direction" : 2,
      "price" : 8950,
      "mark_price" : 8948.9,
      "instrument_name" : "BTC-PERPETUAL",
      "index_price" : 8955.88,
      "direction" : "sell",
      "amount" : 10
    }	
	`
	expected := PublicTrade{
		TradeSeq:      30289442,
		TradeId:       "48079269",
		Timestamp:     1590484512188,
		TickDirection: 2,
		Price:         8950,
		Instrument:    "BTC-PERPETUAL",
		Direction:     Sell,
		Amount:        10,
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parsePublicTrade(v))
}

func TestParseInstrumentState(t *testing.T) {
	input := `
	{
	  "timestamp" : 1553080940000,
	  "state" : "created",
	  "instrument_name" : "BTC-22MAR19"
	}	
	`
	expected := InstrumentState{
		Timestamp:  1553080940000,
		State:      "created",
		Instrument: "BTC-22MAR19",
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parseInstrumentState(v))
}

func TestParseOrderbookDepth(t *testing.T) {
	input := `
	{
      "timestamp" : 1554375447971,
      "instrument_name" : "ETH-PERPETUAL",
      "change_id" : 109615,
	  "bids" : [
		[160, 40]
      ],
      "asks" : [
        [161, 20]
	  ]
	}	
	`
	expected := OrderbookDepth{
		Timestamp:  1554375447971,
		Instrument: "ETH-PERPETUAL",
		ChangeID:   109615,
		Bids: []tradekit.Level{
			{Price: 160, Amount: 40},
		},
		Asks: []tradekit.Level{
			{Price: 161, Amount: 20},
		},
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parseOrderbookDepth(v))
}

func TestParseOrderbookUpdate(t *testing.T) {
	input := `
	{
		"type" : "snapshot",
		"timestamp" : 1554373962454,
		"instrument_name" : "BTC-PERPETUAL",
		"change_id" : 297217,
		"bids" : [
		  [
			"new",
			5042.34,
			30
		  ],
		  [
			"change",
			5041.94,
			20
		  ]
		],
		"asks" : [
		  [
			"new",
			5042.64,
			40
		  ],
		  [
			"delete",
			5043.3,
			0
		  ]
		]
	  }	
	`
	expected := OrderbookUpdate{
		Type:       "snapshot",
		Timestamp:  1554373962454,
		Instrument: "BTC-PERPETUAL",
		ChangeID:   297217,
		Bids: []tradekit.Level{
			{Price: 5042.34, Amount: 30},
			{Price: 5041.94, Amount: 20},
		},
		Asks: []tradekit.Level{
			{Price: 5042.64, Amount: 40},
			{Price: 5043.3, Amount: 0},
		},
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parseOrderbookUpdate(v))
}

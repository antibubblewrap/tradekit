package bybit

import (
	"testing"

	"github.com/antibubblewrap/tradekit"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fastjson"
)

func TestParseTrades(t *testing.T) {
	input := `
	{
		"topic": "publicTrade.BTCUSDT",
		"type": "snapshot",
		"ts": 1672304486868,
		"data": [
			{
				"T": 1672304486865,
				"s": "BTCUSDT",
				"S": "Buy",
				"v": "0.001",
				"p": "16578.50",
				"L": "PlusTick",
				"i": "20f43950-d8dd-5b31-9112-a178eb6023af",
				"BT": false
			}
		]
	}
	`

	expected := Trades{
		Topic:     "publicTrade.BTCUSDT",
		Type:      "snapshot",
		Timestamp: 1672304486868,
		Data: []Trade{
			{
				FilledTimestamp: 1672304486865,
				Symbol:          "BTCUSDT",
				Direction:       Buy,
				Amount:          0.001,
				Price:           16578.50,
				TradeID:         "20f43950-d8dd-5b31-9112-a178eb6023af",
				BlockTrade:      false,
			},
		},
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	trades, err := parseTrades(v)
	assert.Nil(t, err)
	assert.Equal(t, expected, trades)
}

func TestParseOrderbookUpdte(t *testing.T) {
	input := `
	{
		"topic": "orderbook.50.BTCUSDT",
		"type": "snapshot",
		"ts": 1672304484978,
		"data": {
			"s": "BTCUSDT",
			"b": [
				[
					"16493.50",
					"0.006"
				],
				[
					"16493.00",
					"0.100"
				]
			],
			"a": [
				[
					"16611.00",
					"0.029"
				],
				[
					"16612.00",
					"0.213"
				]
			],
		"u": 18521288,
		"seq": 7961638724
		}
	}	
	`

	expected := OrderbookUpdate{
		Topic:     "orderbook.50.BTCUSDT",
		Type:      "snapshot",
		Timestamp: 1672304484978,
		Data: OrderbookUpdateData{
			Symbol: "BTCUSDT",
			Bids: []tradekit.Level{
				{Price: 16493.50, Amount: 0.006},
				{Price: 16493.00, Amount: 0.100},
			},
			Asks: []tradekit.Level{
				{Price: 16611.00, Amount: 0.029},
				{Price: 16612.00, Amount: 0.213},
			},
			UpdateID: 18521288,
			Sequence: 7961638724,
		},
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	msg, err := parseOrderbookUpdate(v)
	assert.Nil(t, err)
	assert.Equal(t, expected, msg)
}

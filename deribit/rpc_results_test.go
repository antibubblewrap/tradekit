package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fastjson"
)

func TestParseOrderStateBuy(t *testing.T) {
	input := `
	  {
		"trades": [
		  {
			"trade_seq": 1966056,
			"trade_id": "ETH-2696083",
			"timestamp": 1590483938456,
			"tick_direction": 0,
			"state": "filled",
			"reduce_only": false,
			"price": 203.3,
			"post_only": false,
			"order_type": "market",
			"order_id": "ETH-584849853",
			"matching_id": null,
			"mark_price": 203.28,
			"liquidity": "T",
			"label": "market0000234",
			"instrument_name": "ETH-PERPETUAL",
			"index_price": 203.33,
			"fee_currency": "ETH",
			"fee": 0.00014757,
			"direction": "buy",
			"amount": 40
		  }
		],
		"order": {
		  "web": false,
		  "time_in_force": "good_til_cancelled",
		  "replaced": false,
		  "reduce_only": false,
		  "profit_loss": 0.00022929,
		  "price": 207.3,
		  "post_only": false,
		  "order_type": "market",
		  "order_state": "filled",
		  "order_id": "ETH-584849853",
		  "max_show": 40,
		  "last_update_timestamp": 1590483938456,
		  "label": "market0000234",
		  "is_liquidation": false,
		  "instrument_name": "ETH-PERPETUAL",
		  "filled_amount": 40,
		  "direction": "buy",
		  "creation_timestamp": 1590483938456,
		  "commission": 0.00014757,
		  "average_price": 203.3,
		  "api": true,
		  "amount": 40
		}
	  }	
	`
	expected := OrderUpdate{
		Trades: []TradeExecution{
			{
				TradeSeq:       1966056,
				TradeId:        "ETH-2696083",
				Timestamp:      1590483938456,
				Price:          203.3,
				OrderId:        "ETH-584849853",
				Liquidity:      "T",
				InstrumentName: "ETH-PERPETUAL",
				Fee:            0.00014757,
				Direction:      Buy,
				Amount:         40,
			},
		},
		Order: Order{
			TimeInForce:         GTC,
			ReduceOnly:          false,
			ProfitLoss:          0.00022929,
			Price:               207.3,
			PostOnly:            false,
			OrderType:           MarketOrder,
			OrderState:          "filled",
			OrderId:             "ETH-584849853",
			MaxShow:             40,
			LastUpdateTimestamp: 1590483938456,
			Label:               "market0000234",
			IsLiqidation:        false,
			InstrumentName:      "ETH-PERPETUAL",
			FilledAmount:        40,
			Direction:           "buy",
			CreationTimestamp:   1590483938456,
			Commission:          0.00014757,
			AveragePrice:        203.3,
			Amount:              40,
		},
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parseOrderUpdate(v))
}

func TestParseOrderStateSell(t *testing.T) {
	input := `
	  {
		"trades": [],
		"order": {
		  "triggered": false,
		  "trigger": "last_price",
		  "time_in_force": "good_til_cancelled",
		  "trigger_price": 145,
		  "reduce_only": false,
		  "profit_loss": 0,
		  "price": 145.61,
		  "post_only": false,
		  "order_type": "stop_limit",
		  "order_state": "untriggered",
		  "order_id": "ETH-SLTS-28",
		  "max_show": 123,
		  "last_update_timestamp": 1550659803407,
		  "label": "",
		  "is_liquidation": false,
		  "instrument_name": "ETH-PERPETUAL",
		  "direction": "sell",
		  "creation_timestamp": 1550659803407,
		  "api": true,
		  "amount": 123
		}
      }	
	`

	expected := OrderUpdate{
		Trades: []TradeExecution{},
		Order: Order{
			Triggered:           false,
			Trigger:             "last_price",
			TimeInForce:         GTC,
			TriggerPrice:        145,
			ReduceOnly:          false,
			ProfitLoss:          0,
			Price:               145.61,
			PostOnly:            false,
			OrderType:           StopLimit,
			OrderState:          "untriggered",
			OrderId:             "ETH-SLTS-28",
			MaxShow:             123,
			LastUpdateTimestamp: 1550659803407,
			Label:               "",
			IsLiqidation:        false,
			InstrumentName:      "ETH-PERPETUAL",
			Direction:           "sell",
			CreationTimestamp:   1550659803407,
			Amount:              123,
		},
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parseOrderUpdate(v))
}

func TestParseOrderStateEdit(t *testing.T) {
	input := `
	  {
		"trades": [],
		"order": {
		  "time_in_force": "good_til_cancelled",
		  "reduce_only": false,
		  "profit_loss": 0,
		  "price": 0.1448,
		  "post_only": false,
		  "order_type": "limit",
		  "order_state": "open",
		  "order_id": "438994",
		  "max_show": 4,
		  "last_update_timestamp": 1550585797677,
		  "label": "",
		  "is_liquidation": false,
		  "instrument_name": "BTC-22FEB19-3500-C",
		  "implv": 222,
		  "filled_amount": 0,
		  "direction": "buy",
		  "creation_timestamp": 1550585741277,
		  "commission": 0,
		  "average_price": 0,
		  "api": false,
		  "amount": 4,
		  "advanced": "implv"
		}
	  }	
	`

	expected := OrderUpdate{
		Trades: []TradeExecution{},
		Order: Order{
			TimeInForce:         GTC,
			Price:               0.1448,
			OrderType:           LimitOrder,
			OrderState:          "open",
			OrderId:             "438994",
			MaxShow:             4,
			LastUpdateTimestamp: 1550585797677,
			InstrumentName:      "BTC-22FEB19-3500-C",
			Direction:           "buy",
			CreationTimestamp:   1550585741277,
			Amount:              4,
		},
	}
	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parseOrderUpdate(v))
}

func TestParseOrderStateClose(t *testing.T) {
	input := `
	{
		"trades": [
		  {
			"trade_seq": 1966068,
			"trade_id": "ETH-2696097",
			"timestamp": 1590486335742,
			"tick_direction": 0,
			"state": "filled",
			"reduce_only": true,
			"price": 202.8,
			"post_only": false,
			"order_type": "limit",
			"order_id": "ETH-584864807",
			"matching_id": null,
			"mark_price": 202.79,
			"liquidity": "T",
			"instrument_name": "ETH-PERPETUAL",
			"index_price": 202.86,
			"fee_currency": "ETH",
			"fee": 0.00007766,
			"direction": "sell",
			"amount": 21
		  }
		],
		"order": {
		  "web": false,
		  "time_in_force": "good_til_cancelled",
		  "replaced": false,
		  "reduce_only": true,
		  "profit_loss": -0.00025467,
		  "price": 198.75,
		  "post_only": false,
		  "order_type": "limit",
		  "order_state": "filled",
		  "order_id": "ETH-584864807",
		  "max_show": 21,
		  "last_update_timestamp": 1590486335742,
		  "label": "",
		  "is_liquidation": false,
		  "instrument_name": "ETH-PERPETUAL",
		  "filled_amount": 21,
		  "direction": "sell",
		  "creation_timestamp": 1590486335742,
		  "commission": 0.00007766,
		  "average_price": 202.8,
		  "api": true,
		  "amount": 21
		}
	  }	
	`

	expected := OrderUpdate{
		Trades: []TradeExecution{
			{
				TradeSeq:       1966068,
				TradeId:        "ETH-2696097",
				Timestamp:      1590486335742,
				Price:          202.8,
				OrderId:        "ETH-584864807",
				Liquidity:      "T",
				InstrumentName: "ETH-PERPETUAL",
				Fee:            0.00007766,
				Direction:      Sell,
				Amount:         21,
			},
		},
		Order: Order{
			TimeInForce:         GTC,
			ReduceOnly:          true,
			ProfitLoss:          -0.00025467,
			Price:               198.75,
			OrderType:           LimitOrder,
			OrderState:          "filled",
			OrderId:             "ETH-584864807",
			MaxShow:             21,
			LastUpdateTimestamp: 1590486335742,
			InstrumentName:      "ETH-PERPETUAL",
			FilledAmount:        21,
			Direction:           "sell",
			CreationTimestamp:   1590486335742,
			Commission:          0.00007766,
			AveragePrice:        202.8,
			Amount:              21,
		},
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parseOrderUpdate(v))
}

func TestParsePosition(t *testing.T) {
	input := `
	{
		"average_price": 0,
		"delta": 0,
		"direction": "buy",
		"estimated_liquidation_price": 0,
		"floating_profit_loss": 0,
		"index_price": 3555.86,
		"initial_margin": 0,
		"instrument_name": "BTC-PERPETUAL",
		"interest_value" : 1.7362511643080387,
		"leverage": 100,
		"kind": "future",
		"maintenance_margin": 0,
		"mark_price": 3556.62,
		"open_orders_margin": 0.000165889,
		"realized_profit_loss": 0,
		"settlement_price": 3555.44,
		"size": 0,
		"size_currency": 0,
		"total_profit_loss": 0
	}
	`
	expected := Position{
		Direction:        "buy",
		IndexPrice:       3555.86,
		InstrumentName:   "BTC-PERPETUAL",
		InterestValue:    1.7362511643080387,
		Leverage:         100,
		Kind:             "future",
		MarkPrice:        3556.62,
		OpenOrdersMargin: 0.000165889,
		SettlementPrice:  3555.44,
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parsePosition(v))
}

func TestParsePositions(t *testing.T) {
	input := `
	[
      {
          "average_price": 7440.18,
          "delta": 0.006687487,
          "direction": "buy",
          "estimated_liquidation_price": 1.74,
          "floating_profit_loss": 0,
          "index_price": 7466.79,
          "initial_margin": 0.000197283,
          "instrument_name": "BTC-PERPETUAL",
          "interest_value" : 1.7362511643080387,
          "kind": "future",
          "leverage": 34,
          "maintenance_margin": 0.000143783,
          "mark_price": 7476.65,
          "open_orders_margin": 0.000197288,
          "realized_funding": -0.001,
          "realized_profit_loss": -0.0001,
          "settlement_price": 7476.65,
          "size": 50,
          "size_currency": 0.006687487,
          "total_profit_loss": 0.000032781
      }
  	]	
	`
	expected := []Position{
		{
			AveragePrice:              7440.18,
			Delta:                     0.006687487,
			Direction:                 "buy",
			EstimatedLiquidationPrice: 1.74,
			IndexPrice:                7466.79,
			InitialMargin:             0.000197283,
			InstrumentName:            "BTC-PERPETUAL",
			InterestValue:             1.7362511643080387,
			Kind:                      FutureInstrument,
			Leverage:                  34,
			MaintenanceMargin:         0.000143783,
			MarkPrice:                 7476.65,
			OpenOrdersMargin:          0.000197288,
			RealizedFunding:           -0.001,
			RealizedProfitLoss:        -0.0001,
			SettlementPrice:           7476.65,
			Size:                      50,
			SizeCurrency:              0.006687487,
			TotalProfitLoss:           0.000032781,
		},
	}

	var p fastjson.Parser
	v, err := p.Parse(input)
	assert.Nil(t, err)
	assert.Equal(t, expected, parsePositions(v))
}

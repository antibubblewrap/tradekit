package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderOptions(t *testing.T) {
	opts := OrderOptions{
		Type:           TrailingStop,
		Label:          "xyz",
		Price:          123,
		TimeInForce:    GTC,
		MaxShow:        1,
		PostOnly:       true,
		RejectPostOnly: true,
		ReduceOnly:     true,
		TriggerPrice:   456,
		TriggerOffset:  123,
		Trigger:        "index_price",
		ValidUntil:     1,
	}
	expected := map[string]interface{}{
		"type":             "trailing_stop",
		"label":            "xyz",
		"price":            float64(123),
		"time_in_force":    "good_til_cancelled",
		"max_show":         float64(1),
		"post_only":        true,
		"reject_post_only": true,
		"reduce_only":      true,
		"trigger_price":    float64(456),
		"trigger_offset":   float64(123),
		"trigger":          "index_price",
		"valid_until":      int64(1),
	}
	assert.True(t, mapsEqual(expected, opts.toMap()))
	assert.Equal(t, expected, opts.toMap())

	var optsNil *OrderOptions
	assert.Equal(t, map[string]interface{}{}, optsNil.toMap())
}

func TestCancelCurrencyOptions(t *testing.T) {
	opts := CancelCurrencyOptions{
		Type: TakeLimitOrder,
		Kind: OptionInstrument,
	}
	expected := map[string]interface{}{"type": "take_limit", "kind": "option"}
	assert.True(t, mapsEqual(expected, opts.toMap()))

	var optsNil *CancelCurrencyOptions
	assert.Equal(t, map[string]interface{}{}, optsNil.toMap())
}

func TestCancelInstrumentOptions(t *testing.T) {
	opts := CancelInstrumentOptions{
		Type: MarketLimit,
	}
	expected := map[string]interface{}{"type": "market_limit"}
	assert.True(t, mapsEqual(expected, opts.toMap()))

	var optsNil *CancelInstrumentOptions
	assert.Equal(t, map[string]interface{}{}, optsNil.toMap())
}

func TestCancelLabelOptions(t *testing.T) {
	opts := CancelLabelOptions{
		Currency: "BTC",
	}
	expected := map[string]interface{}{"currency": "BTC"}
	assert.True(t, mapsEqual(expected, opts.toMap()))

	var optsNil *CancelLabelOptions
	assert.Equal(t, map[string]interface{}{}, optsNil.toMap())
}

func TestEditOrderOptions(t *testing.T) {
	opts := EditOrderOptions{
		Price:          123.4,
		PostOnly:       true,
		RejectPostOnly: true,
		ReduceOnly:     true,
		TriggerPrice:   11.12,
		TriggerOffset:  33.33,
		ValidUntil:     123,
	}
	expected := map[string]interface{}{
		"price":            123.4,
		"post_only":        true,
		"reject_post_only": true,
		"reduce_only":      true,
		"trigger_price":    11.12,
		"trigger_offset":   33.33,
		"valid_until":      int64(123),
	}
	assert.True(t, mapsEqual(expected, opts.toMap()))

	var optsNil *CancelLabelOptions
	assert.Equal(t, map[string]interface{}{}, optsNil.toMap())
}

func mapsEqual(m1 map[string]interface{}, m2 map[string]interface{}) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k1, v1 := range m1 {
		v2, ok := m2[k1]
		if !ok || v1 != v2 {
			return false
		}
	}
	for k2, v2 := range m2 {
		v1, ok := m1[k2]
		if !ok || v1 != v2 {
			return false
		}
	}
	return true

}

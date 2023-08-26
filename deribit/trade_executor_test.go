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
	params := opts.params()
	assert.True(t, mapsEqual(expected, params))

	var optsNil *OrderOptions
	params = optsNil.params()
	assert.Equal(t, map[string]interface{}{}, params)
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
	assert.True(t, mapsEqual(expected, opts.params()))

	var optsNil *EditOrderOptions
	assert.Equal(t, map[string]interface{}{}, optsNil.params())
}

func TestCancelOrderOptions(t *testing.T) {

	table := []struct {
		opts           *CancelOrderOptions
		expectedParams map[string]interface{}
		expectedMethod rpcMethod
	}{
		{
			opts: &CancelOrderOptions{
				Currency: "BTC",
				Kind:     OptionInstrument,
				Type:     LimitOrder,
			},
			expectedParams: map[string]interface{}{
				"currency": "BTC",
				"kind":     OptionInstrument,
				"type":     LimitOrder,
			},
			expectedMethod: methodPrivateCancelAllCurrency,
		},
		{
			opts:           &CancelOrderOptions{Instrument: "BTC-PERPETUAL"},
			expectedParams: map[string]interface{}{"instrument_name": "BTC-PERPETUAL"},
			expectedMethod: methodPrivateCancelAllInstrument,
		},
		{
			opts: &CancelOrderOptions{
				Label:    "abc",
				Currency: "BTC",
				Type:     LimitOrder, // should be ignored
			},
			expectedParams: map[string]interface{}{
				"label":    "abc",
				"currency": "BTC",
			},
			expectedMethod: methodPrivateCancelByLabel,
		},
		{
			opts: &CancelOrderOptions{
				Label:    "abc",
				Currency: "BTC",
				Type:     LimitOrder, // should be ignored
			},
			expectedParams: map[string]interface{}{
				"label":    "abc",
				"currency": "BTC",
			},
			expectedMethod: methodPrivateCancelByLabel,
		},
		{
			opts:           &CancelOrderOptions{},
			expectedParams: map[string]interface{}{},
			expectedMethod: methodPrivateCancelAll,
		},
		{
			opts:           nil,
			expectedParams: map[string]interface{}{},
			expectedMethod: methodPrivateCancelAll,
		},
	}

	for i, test := range table {
		method, params := test.opts.methodAndParams()
		assert.Equal(t, test.expectedMethod, method, i)
		assert.True(t, mapsEqual(test.expectedParams, params), i)
	}
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
	return true
}

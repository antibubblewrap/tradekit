package deribit

import "fmt"

// InstrumentKind specifies a type of trading instrument (future, option etc.)
type InstrumentKind string

const (
	FutureInstrument      InstrumentKind = "future"
	OptionInstrument                     = "option"
	SpotInstrument                       = "spot"
	FutureComboInstrument                = "future_combo"
	OptionComboInstrument                = "option_combo"
	AnyInstrument                        = "any"
)

// UpdateFrequency specifies the update frequency for a data stream.
type UpdateFrequency string

const (
	UpdateRaw   UpdateFrequency = "raw"
	Update100ms                 = "100ms"
)

// OrderType represents the type of a buy or sell order.
type OrderType string

const (
	MarketOrder     OrderType = "market"
	LimitOrder      OrderType = "limit"
	TakeLimitOrder  OrderType = "take_limit"
	StopMarketOrder OrderType = "stop_market"
	TakeMarketOrder OrderType = "take_market"
	MarketLimit     OrderType = "market_limit"
	TrailingStop    OrderType = "trailing_stop"
	StopLimit       OrderType = "stop_limit"
)

// TimeInForce represents the conditions under which an order should remain active.
type TimeInForce string

const (
	GTC TimeInForce = "good_til_cancelled"
	GTD             = "good_til_day"
	FOK             = "fill_or_kill"
	IOC             = "immediate_or_cancel"
)

func updateFreqFromString(s string) (UpdateFrequency, error) {
	switch s {
	case string(UpdateRaw):
		return UpdateRaw, nil
	case Update100ms:
		return Update100ms, nil
	default:
		return "", fmt.Errorf("invalid update frequency %q", s)
	}
}

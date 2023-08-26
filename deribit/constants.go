package deribit

// InstrumentKind specifies a type of trading instrument (future, option etc.)
type InstrumentKind string

const (
	FutureInstrument      InstrumentKind = "future"
	OptionInstrument      InstrumentKind = "option"
	SpotInstrument        InstrumentKind = "spot"
	FutureComboInstrument InstrumentKind = "future_combo"
	OptionComboInstrument InstrumentKind = "option_combo"
)

// UpdateFrequency specifies the update frequency for a data stream.
type UpdateFrequency string

const (
	UpdateRaw   UpdateFrequency = "raw"
	Update100ms UpdateFrequency = "100ms"
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
	GTD TimeInForce = "good_til_day"
	FOK TimeInForce = "fill_or_kill"
	IOC TimeInForce = "immediate_or_cancel"
)

// TradeDirection represents the direction of a trade - buy / sell
type TradeDirection int

const (
	Buy TradeDirection = iota
	Sell
)

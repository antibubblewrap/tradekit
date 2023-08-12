package tradekit

import (
	"math"
	"time"
)

// EWMA implements an exponentially weighted moving average over irregularly spaced
// time intervals. See https://en.wikipedia.org/wiki/Exponential_smoothing
type EWMA struct {
	tau    float64
	prevTs int64
	// The value of the EWMA
	Value float64
}

// NewEWMA creates a new EWMA with a given halflife.
func NewEWMA(halflife time.Duration) *EWMA {
	halflifeMillis := halflife.Milliseconds()
	prevTs := int64(0)
	tau := float64(halflifeMillis) / math.Log(2)
	return &EWMA{tau: tau, prevTs: prevTs, Value: 0}
}

// Update the EWMA with a new value produced at timestamp in millisecond units. Returns
// the updated EWMA value.
func (ewma *EWMA) Update(v float64, tsMillis int64) float64 {
	if ewma.prevTs == 0 {
		ewma.Value = v
		ewma.prevTs = tsMillis
		return ewma.Value
	}
	dt := float64(tsMillis - ewma.prevTs)
	alpha := 1 - math.Exp(-dt/ewma.tau)
	ewma.Value += alpha * (v - ewma.Value)
	ewma.prevTs = tsMillis
	return ewma.Value
}

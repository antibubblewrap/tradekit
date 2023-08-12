package tradekit

import (
	"time"

	"github.com/edwingeng/deque/v2"
)

// RollingSum calculates a rolling sum over a specified time window.
type RollingSum struct {
	events *deque.Deque[event]
	window time.Duration
	value  float64
}

type event struct {
	ts    int64
	value float64
}

// NewRollingSum creates a RollingSum with a specified time window.
func NewRollingSum(window time.Duration) *RollingSum {
	events := deque.NewDeque[event]()
	return &RollingSum{events: events, window: window, value: 0}
}

func (rs *RollingSum) refreshWindow(ts int64) {
	for {
		e, ok := rs.events.Front()
		if !ok {
			break
		}
		if e.ts < ts-rs.window.Microseconds() {
			rs.events.PopFront()
			rs.value -= e.value
		} else {
			break
		}
	}
}

// Update the rolling sum with a new value.
func (rs *RollingSum) Update(v float64) {
	ts := time.Now().UTC().UnixMicro()
	rs.events.PushBack(event{ts, v})
	rs.value += v
	rs.refreshWindow(ts)
}

// Get the current value of the rolling sum.
func (rs *RollingSum) Value() float64 {
	rs.refreshWindow(time.Now().UTC().UnixMicro())
	return rs.value
}

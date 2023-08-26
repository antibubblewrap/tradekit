package bybit

import (
	"bytes"

	"github.com/valyala/fastjson"
)

func heartbeatMsg() []byte {
	return []byte(`{"op": "ping"}`)
}

// We don't need to pass on ping responses or subscribe responses to the consumer of a
// stream.
func isPingOrSubscribeMsg(v *fastjson.Value) bool {
	op := v.GetStringBytes("op")
	if bytes.Equal(op, []byte("ping")) || bytes.Equal(op, []byte("subscribe")) {
		return true
	}
	return false
}

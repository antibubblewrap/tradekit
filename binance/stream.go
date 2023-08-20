package binance

import (
	"encoding/json"
	"fmt"
	"time"
)

func genId() int64 {
	return time.Now().UnixNano()
}

func newSubscribeMsg(id int64, channels []string) ([]byte, error) {
	m := map[string]interface{}{
		"id":     id,
		"method": "SUBSCRIBE",
		"params": channels,
	}
	return json.Marshal(m)
}

func streamParseErr(stream string, msg []byte, err error) error {
	return fmt.Errorf("parsing Binance %s stream message: %w (%s)", stream, err, string(msg))
}

func streamError(stream string, err error) error {
	return fmt.Errorf("Binance %s: %w", stream, err)
}

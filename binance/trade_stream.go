package binance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

// Trade is the type produced by a Binance TradeStream.
type Trade struct {
	// EventTime is the timestamp at which Binance produced the message.
	EventTime     int64   `json:"E"`
	Symbol        string  `json:"s"`
	TradeId       int64   `json:"t"`
	Price         float64 `json:"p,string"`
	Quantity      float64 `json:"q,string"`
	BuyerOrderId  int64   `json:"b"`
	SellerOrderId int64   `json:"a"`
	// TradeTime is the timestamp at which the trade was executed.
	TradeTime    int64 `json:"T"`
	IsBuyerMaker bool  `json:"m"`
	M            bool  `json:"M"`
}

// TradeStream provides a realtime stream of trades from Binance. This stream is only
// available on spot markets.
type TradeStream struct {
	url     string
	symbols []string
	msgs    chan Trade
	errc    chan error
	p       fastjson.Parser
	subIds  map[int64]struct{}
}

func NewTradeSteam(url string, symbols ...string) *TradeStream {
	errc := make(chan error, 1)
	msgs := make(chan Trade, 10)
	subIds := make(map[int64]struct{})
	return &TradeStream{url: url, symbols: symbols, msgs: msgs, errc: errc, subIds: subIds}
}

func (s *TradeStream) parseTradeMsg(msg websocket.Message) (Trade, error) {
	defer msg.Release()
	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return Trade{}, streamParseErr("trade", msg.Data(), err)
	}

	if id := v.GetInt64("id"); id != 0 {
		// This should be a subscription response message
		if _, ok := s.subIds[id]; ok {
			delete(s.subIds, id)
			return Trade{}, nil
		}
		return Trade{}, streamParseErr("trade", msg.Data(), errors.New("expected a trade message"))
	}

	eventType := v.GetStringBytes("e")
	if !bytes.Equal(eventType, []byte("trade")) {
		return Trade{}, streamParseErr("trade", msg.Data(), errors.New("expected trade type"))
	}

	price, err := strconv.ParseFloat(string(v.GetStringBytes("p")), 64)
	if err != nil {
		return Trade{}, streamParseErr("trade", msg.Data(), errors.New("invalid price"))
	}
	quantity, err := strconv.ParseFloat(string(v.GetStringBytes("q")), 64)
	if err != nil {
		return Trade{}, streamParseErr("trade", msg.Data(), errors.New("invalid quanity"))
	}

	return Trade{
		EventTime:     v.GetInt64("E"),
		Symbol:        string(v.GetStringBytes("s")),
		TradeId:       v.GetInt64("t"),
		Price:         price,
		Quantity:      quantity,
		BuyerOrderId:  v.GetInt64("b"),
		SellerOrderId: v.GetInt64("a"),
		TradeTime:     v.GetInt64("T"),
		IsBuyerMaker:  v.GetBool("m"),
		M:             v.GetBool("M"),
	}, nil
}

func (s *TradeStream) Start(ctx context.Context) error {
	// Websocket setup
	ws := websocket.New(s.url, nil)
	ws.OnConnect = func() error {
		channels := make([]string, len(s.symbols))
		for i, symbol := range s.symbols {
			channels[i] = fmt.Sprintf("%s@trade", symbol)
		}
		id := genId()
		subMsg, err := newSubscribeMsg(id, channels)
		if err != nil {
			return streamError("TradeStream", err)
		}
		s.subIds[id] = struct{}{}
		ws.Send(subMsg)
		return nil
	}
	if err := ws.Start(ctx); err != nil {
		return streamError("TradeStream", fmt.Errorf("websocket connect: %w", err))
	}

	go func() {
		defer close(s.msgs)
		for {
			select {
			case data := <-ws.Messages():
				msg, err := s.parseTradeMsg(data)
				if err != nil {
					s.errc <- streamError("TradeStream", err)
					return
				}
				if msg.Symbol != "" {
					s.msgs <- msg
				}
			case err := <-ws.Err():
				s.errc <- streamError("TradeStream", err)
				return
			}
		}
	}()

	return nil
}

func (s *TradeStream) Messages() <-chan Trade {
	return s.msgs
}

func (s *TradeStream) Err() <-chan error {
	return s.errc
}

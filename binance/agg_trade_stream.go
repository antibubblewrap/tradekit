package binance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

type AggTrade struct {
	EventTime    int64   `json:"E"`
	Symbol       string  `json:"s"`
	AggTradeId   int64   `json:"a"`
	Price        float64 `json:"p,string"`
	Quantity     float64 `json:"q,string"`
	FirstTradeId int64   `json:"f"`
	LastTradeId  int64   `json:"l"`
	Timestamp    int64   `json:"T"`
	IsBuyerMaker bool    `json:"m"`
	M            bool    `json:"M"`
}

// AggTradesStream connects to the Binance Aggregate Trade Streams websocket.
type AggTradeStream struct {
	url     string
	symbols []string
	msgs    chan AggTrade
	errc    chan error
	p       fastjson.Parser
	subIds  map[int64]struct{}
}

func NewAggTradeStream(url string, symbols ...string) *AggTradeStream {
	errc := make(chan error, 1)
	msgs := make(chan AggTrade, 10)
	subIds := make(map[int64]struct{})
	return &AggTradeStream{url: url, symbols: symbols, msgs: msgs, errc: errc, subIds: subIds}
}

func (s *AggTradeStream) parseAggTradeMsg(msg websocket.Message) (AggTrade, error) {
	defer msg.Release()
	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return AggTrade{}, streamParseErr("aggTrade", msg.Data(), err)
	}

	if id := v.GetInt64("id"); id != 0 {
		// This should be a subscription response message
		if _, ok := s.subIds[id]; ok {
			delete(s.subIds, id)
			return AggTrade{}, nil
		}
		return AggTrade{}, streamParseErr("aggTrade", msg.Data(), errors.New("expected a agg trade message"))
	}

	eventType := v.GetStringBytes("e")
	if !bytes.Equal(eventType, []byte("aggTrade")) {
		return AggTrade{}, streamParseErr("trade", msg.Data(), errors.New("expected aggTrade type"))
	}

	price, err := strconv.ParseFloat(string(v.GetStringBytes("p")), 64)
	if err != nil {
		return AggTrade{}, streamParseErr("aggTrade", msg.Data(), errors.New("invalid price"))
	}
	quantity, err := strconv.ParseFloat(string(v.GetStringBytes("q")), 64)
	if err != nil {
		return AggTrade{}, streamParseErr("aggTrade", msg.Data(), errors.New("invalid quanity"))
	}

	return AggTrade{
		EventTime:    v.GetInt64("E"),
		Symbol:       string(v.GetStringBytes("s")),
		AggTradeId:   v.GetInt64("a"),
		Price:        price,
		Quantity:     quantity,
		FirstTradeId: v.GetInt64("f"),
		LastTradeId:  v.GetInt64("l"),
		Timestamp:    v.GetInt64("T"),
		IsBuyerMaker: v.GetBool("m"),
		M:            v.GetBool("M"),
	}, nil
}

func (s *AggTradeStream) Start(ctx context.Context) error {
	// Websocket setup
	ws := websocket.New(s.url, nil)
	ws.PingInterval = 15 * time.Second
	ws.OnConnect = func() error {
		channels := make([]string, len(s.symbols))
		for i, symbol := range s.symbols {
			channels[i] = fmt.Sprintf("%s@aggTrade", symbol)
		}
		id := genId()
		subMsg, err := newSubscribeMsg(id, channels)
		if err != nil {
			return streamError("AggTrade", err)
		}
		s.subIds[id] = struct{}{}
		ws.Send(subMsg)
		return nil
	}
	if err := ws.Start(ctx); err != nil {
		return streamError("AggTradeStream", fmt.Errorf("websocket connect: %w", err))
	}

	go func() {
		defer close(s.msgs)
		for {
			select {
			case data := <-ws.Messages():
				msg, err := s.parseAggTradeMsg(data)
				if err != nil {
					s.errc <- streamError("AggTradeStream", err)
					return
				}
				if msg.Symbol != "" {
					s.msgs <- msg
				}
			case err := <-ws.Err():
				s.errc <- streamError("AggTradeStream", err)
				return
			}
		}
	}()

	return nil
}

func (s *AggTradeStream) Messages() <-chan AggTrade {
	return s.msgs
}

func (s *AggTradeStream) Err() <-chan error {
	return s.errc
}

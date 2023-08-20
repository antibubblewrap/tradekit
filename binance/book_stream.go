package binance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/antibubblewrap/tradekit"
	"github.com/antibubblewrap/tradekit/internal/websocket"
	"github.com/valyala/fastjson"
)

type bookUpdate struct {
	EventTime     int64
	Symbol        string
	FirstUpdateId int64
	FinalUpdateId int64
	Bids          []tradekit.Level
	Asks          []tradekit.Level
}

// BookUpdate is the message type produced by OrderbookStream.
type BookUpdate struct {
	// Type is either "snapshot" or "change". If you receive a snapshot, you should
	// overwrite your orderbook with its contents.
	Type string

	// EventTime is the time at which the message was produced by Binance. This field is
	// zero for "snapshot" updates on spot markets.
	EventTime int64

	// Symbol is the trading symbol corresponding to the stream.
	Symbol string

	// Bids and Asks levels. For "change" messages, a level with a zero Amount indicates
	// that price level should be deleted from your local orderbook.
	Bids []tradekit.Level
	Asks []tradekit.Level
}

// OrderbookStream provides a streaming view of updates to the orderbook of of a Binance
// trading symbol. It may be used to maintain a local tradekit Orderbook. The first
// message produced is always a snapshot, and subsequent messages are orderbook updates.
// A new snapshot is sent in the event that the stream's internal websocket reconnects.
type OrderbookStream struct {
	url    string
	symbol string
	msgs   chan BookUpdate
	errc   chan error
	p      fastjson.Parser
	subIds map[int64]struct{}
	api    *Api
}

// NewOrderbookStream creates a new Binance OrderbookStream for a given symbol. An Api
// is required so that the stream can retrieve an initial orderbook snapshot. The provided
// wsUrl should match the market of the Api (for example, if the wsUrl is for Spot, then
// the Api should also be for Spot.). Updates are sent on a 100ms interval.
func NewOrderbookStream(wsUrl string, symbol string, api *Api) *OrderbookStream {
	errc := make(chan error)
	msgs := make(chan BookUpdate)
	subIds := make(map[int64]struct{})
	return &OrderbookStream{url: wsUrl, symbol: symbol, msgs: msgs, errc: errc, subIds: subIds, api: api}
}

func (s *OrderbookStream) parseBookUpdateMsg(msg websocket.Message) (bookUpdate, error) {
	defer msg.Release()
	v, err := s.p.ParseBytes(msg.Data())
	if err != nil {
		return bookUpdate{}, streamParseErr("depthUpdate", msg.Data(), err)
	}

	if id := v.GetInt64("id"); id != 0 {
		// This should be a subscription response message
		if _, ok := s.subIds[id]; ok {
			delete(s.subIds, id)
			return bookUpdate{}, nil
		}
		return bookUpdate{}, streamParseErr("depthUpdate", msg.Data(), errors.New("expected a book update message"))
	}

	eventType := v.GetStringBytes("e")
	if !bytes.Equal(eventType, []byte("depthUpdate")) {
		return bookUpdate{}, streamParseErr("depthUpdate", msg.Data(), errors.New("expected depthUpdate type"))
	}

	bidLevels, err := parsePriceLevels(v.GetArray("b"))
	if err != nil {
		return bookUpdate{}, streamParseErr("depthUpdate", msg.Data(), err)
	}

	askLevels, err := parsePriceLevels(v.GetArray("a"))
	if err != nil {
		return bookUpdate{}, streamParseErr("depthUpdate", msg.Data(), err)
	}

	return bookUpdate{
		EventTime:     v.GetInt64("E"),
		Symbol:        string(v.GetStringBytes("s")),
		FirstUpdateId: v.GetInt64("U"),
		FinalUpdateId: v.GetInt64("u"),
		Bids:          bidLevels,
		Asks:          askLevels,
	}, nil
}

func (s *OrderbookStream) getSnapshot() (OrderbookResponse, error) {
	symbol := strings.ToUpper(s.symbol)
	return s.api.GetOrderbook(symbol, 100)
}

// Start initiates the websocket connection to the orderbook stream.
func (s *OrderbookStream) Start(ctx context.Context) error {
	// Websocket setup
	connecting := make(chan struct{}, 1)
	ws := websocket.New(s.url, nil)
	ws.PingInterval = 15 * time.Second
	ws.OnConnect = func() error {
		connecting <- struct{}{}
		channels := []string{fmt.Sprintf("%s@depth@100ms", s.symbol)}
		id := genId()
		subMsg, err := newSubscribeMsg(id, channels)
		if err != nil {
			return err
		}
		s.subIds[id] = struct{}{}
		ws.Send(subMsg)
		return nil
	}
	if err := ws.Start(ctx); err != nil {
		return streamError("OrderbookStream", fmt.Errorf("websocket connect: %w", err))
	}

	go func() {
		// TODO: we should keep track of the FinalUpdateId of the previous message, and
		// make sure that it's equal to the FirstUpdateId + 1 of the current message, and
		// if not, discard the message and reconnect.
		defer close(s.msgs)
		defer ws.Close()
		getSnapshot := false
		waitingForNextUpdate := false
		var snapshot OrderbookResponse
		timer := time.NewTimer(15 * time.Second)
		for {
			select {
			case <-connecting:
				getSnapshot = true
				timer.Reset(15 * time.Second)
			case data := <-ws.Messages():
				msg, err := s.parseBookUpdateMsg(data)
				if err != nil {
					s.errc <- streamError("OrderbookStream", err)
					return
				}
				if msg.Symbol == "" {
					// It's just the subscribe response. We can discard it
					continue
				}
				if getSnapshot {
					// The websocket has just connected and we've received the first
					// message. Now we need to get a snapshot of the orderbook and then
					// discard any update messages containing only updates prior to the
					// final update of the snapshot.
					// For details see: https://binance-docs.github.io/apidocs/spot/en/#how-to-manage-a-local-order-book-correctly
					var err error
					snapshot, err = s.getSnapshot()
					if err != nil {
						s.errc <- streamError("OrderbookStream", err)
						break
					}
					getSnapshot = false
					waitingForNextUpdate = true

					s.msgs <- snapshotUpdate(snapshot)

					if msg.FinalUpdateId > snapshot.LastUpdateId {
						waitingForNextUpdate = false
						if !timer.Stop() {
							<-timer.C
						}
						s.msgs <- changeUpdate(msg)
					}
				} else if waitingForNextUpdate {
					if msg.FinalUpdateId > snapshot.LastUpdateId {
						waitingForNextUpdate = false
						if !timer.Stop() {
							<-timer.C
						}
						s.msgs <- changeUpdate(msg)
					}
				} else {
					s.msgs <- changeUpdate(msg)
				}
			case err := <-ws.Err():
				s.errc <- streamError("OrderbookStream", err)
				return
			case <-timer.C:
				// We have taken a snapshot, but haven't received the first valid update
				// message within 15 seconds.
				s.errc <- streamError("OrderbookStream", errors.New("snapshot update not received within timeout"))
			}
		}
	}()

	return nil
}

// Messages returns a channel for reading BookUpdate message produced by the stream.
func (s *OrderbookStream) Messages() <-chan BookUpdate {
	return s.msgs
}

// Err returns a channel which produces an error if there is a problem with the stream.
// If an error is produced, then the Messages channel will be closed.
func (s *OrderbookStream) Err() <-chan error {
	return s.errc
}

func snapshotUpdate(m OrderbookResponse) BookUpdate {
	return BookUpdate{
		Type:      "snapshot",
		EventTime: m.EventTime,
		Bids:      m.Bids,
		Asks:      m.Asks,
	}
}

func changeUpdate(m bookUpdate) BookUpdate {
	return BookUpdate{
		Type:      "change",
		EventTime: m.EventTime,
		Bids:      m.Bids,
		Asks:      m.Asks,
	}
}

func parseLevel(v *fastjson.Value) (tradekit.Level, error) {
	priceS := v.GetStringBytes("0")
	amountS := v.GetStringBytes("1")
	if priceS == nil || amountS == nil {
		return tradekit.Level{}, fmt.Errorf("invalid price level: %s", v.String())
	}

	price, err := strconv.ParseFloat(string(priceS), 64)
	if err != nil {
		return tradekit.Level{}, fmt.Errorf("invalid price level: %s", v.String())
	}
	amount, err := strconv.ParseFloat(string(amountS), 64)
	if err != nil {
		return tradekit.Level{}, fmt.Errorf("invalid price level: %s", v.String())
	}

	return tradekit.Level{Price: price, Amount: amount}, nil
}

func parsePriceLevels(items []*fastjson.Value) ([]tradekit.Level, error) {
	levels := make([]tradekit.Level, len(items))
	for i, item := range items {
		level, err := parseLevel(item)
		if err != nil {
			return nil, err
		}
		levels[i] = level
	}
	return levels, nil
}

package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func sleepCtx(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}
}

// Make a webscoket connection. Retries the connection with exponential backoff if the
// connection attempt fails.
func connect(ctx context.Context, url string) (*websocket.Conn, error) {
	sleep := time.Second
	var err error
	var conn *websocket.Conn
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			break
		default:
			conn, _, err = websocket.DefaultDialer.DialContext(ctx, url, nil)
			if err != nil {
				sleepCtx(ctx, sleep)
				sleep *= 2
			} else {
				break
			}
		}
	}

	if conn == nil {
		return nil, err
	}

	return conn, err
}

type Websocket struct {
	Url           string
	conn          *websocket.Conn
	responses     chan []byte
	requests      chan []byte
	close         chan struct{}
	errc          chan error
	closed        bool
	wg            sync.WaitGroup
	OnConnect     func() error
	ResetInterval time.Duration
	PingInterval  time.Duration
}

func New(url string) Websocket {
	return Websocket{
		Url:           url,
		responses:     make(chan []byte, 10),
		requests:      make(chan []byte, 10),
		close:         make(chan struct{}, 1),
		OnConnect:     func() error { return nil },
		ResetInterval: time.Hour * 100000,
		PingInterval:  time.Hour * 100000,
	}
}

func (ws *Websocket) connect(ctx context.Context) error {
	conn, err := connect(ctx, ws.Url)
	if err != nil {
		return err
	}
	ws.conn = conn
	if err := ws.OnConnect(); err != nil {
		return err
	}
	return nil
}

func (ws *Websocket) Start(ctx context.Context) error {
	defer ws.wg.Done()
	if err := ws.connect(ctx); err != nil {
		return err
	}

	isConnecting := false
	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		defer close(ws.responses)
		for {
			_, msg, err := ws.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					ws.closed = true
					return
				} else if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					isConnecting = true
					if err := ws.connect(ctx); err != nil {
						ws.closed = true
						ws.errc <- err
						return
					}
					isConnecting = false
				} else {
					ws.closed = true
					ws.errc <- err
					return
				}
			}
			ws.responses <- msg
		}
	}()

	closeNormalMsg, err := websocket.NewPreparedMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}
	closeGoingAwayMsg, err := websocket.NewPreparedMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
	if err != nil {
		return err
	}

	resetTicker := time.NewTicker(ws.ResetInterval)
	pingTicker := time.NewTicker(ws.PingInterval)
	pingC := make(chan struct{})

	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		for {
			select {
			case <-ctx.Done():
				ws.conn.WritePreparedMessage(closeNormalMsg)
				return
			case <-resetTicker.C:
				if !isConnecting {
					ws.conn.WritePreparedMessage(closeGoingAwayMsg)
				}
			case <-pingTicker.C:
				if !isConnecting {
					pingC <- struct{}{}
				}
			case <-ws.close:
				ws.conn.WritePreparedMessage(closeNormalMsg)
				return
			}
		}
	}()

	ws.wg.Add(1)
	go func() {
		defer ws.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ws.requests:
				ws.conn.WriteMessage(websocket.TextMessage, msg)
			case <-pingC:
				ws.conn.WriteMessage(websocket.PingMessage, nil)
			}
		}
	}()

	return nil
}

func (ws *Websocket) Send(data []byte) {
	ws.requests <- data
}

func (ws *Websocket) Messages() <-chan []byte {
	return ws.responses
}

func (ws *Websocket) Close() {
	if ws.closed {
		return
	}
	ws.closed = true
	ws.close <- struct{}{}
}

func (ws *Websocket) Err() <-chan error {
	return ws.errc
}

// func (ws *Websocket) Err() error {
// 	ws.wg.Wait()
// 	return ws.err
// }

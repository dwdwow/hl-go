package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dwdwow/hl-go/utils"
	"github.com/gorilla/websocket"
)

type PostRequestType string

const (
	PostRequestTypeInfo   PostRequestType = "info"
	PostRequestTypeAction PostRequestType = "action"
)

type PostResponseType string

const (
	PostResponseInfo   PostResponseType = "info"
	PostResponseAction PostResponseType = "action"
	PostResponseError  PostResponseType = "error"
)

type PostResponse struct {
	Channel string           `json:"channel"`
	Data    PostResponseData `json:"data"`
	Err     error            `json:"-"`
}

type PostResponseData struct {
	ID       int64               `json:"id"`
	Response PostResponseContent `json:"response"`
}

type PostResponseContent struct {
	Type    PostResponseType `json:"type"`
	Payload json.RawMessage  `json:"payload"`
}

type PostOnlyRespWaiter struct {
	ID int64
	ch chan *PostResponse
}

func (w *PostOnlyRespWaiter) Chan() <-chan *PostResponse {
	return w.ch
}

type PostOnlyClient struct {
	url     string
	conn    *websocket.Conn
	writeMu sync.Mutex

	id            int64
	respWaiters   map[int64]PostOnlyRespWaiter
	respWaitersMu sync.Mutex

	ctx          context.Context
	cancel       context.CancelFunc
	pingInterval time.Duration
}

func NewPostOnlyClient() *PostOnlyClient {
	return &PostOnlyClient{
		url:          MainnetWsURL,
		pingInterval: 40 * time.Second,                   // Default ping interval
		respWaiters:  make(map[int64]PostOnlyRespWaiter), // Initialize respWaiters to avoid nil map panic
	}
}

func (c *PostOnlyClient) Request(magType PostRequestType, payload any) (waiter PostOnlyRespWaiter, err error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.conn == nil {
		err = fmt.Errorf("client not connected")
		return
	}
	c.id++
	msg := utils.NewOrderedMap(
		"method", "post",
		"id", c.id,
		"request", utils.NewOrderedMap(
			"type", magType,
			"payload", payload,
		),
	)
	err = c.conn.WriteJSON(msg)
	if err != nil {
		return
	}
	waiter = PostOnlyRespWaiter{
		ID: c.id,
		ch: make(chan *PostResponse, 1),
	}
	c.respWaitersMu.Lock()
	c.respWaiters[c.id] = waiter
	c.respWaitersMu.Unlock()
	return
}

func (c *PostOnlyClient) Start() error {
	// Create context for controlling the ping goroutine
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Connect to WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	c.conn = conn

	// Start ping goroutine
	go c.pingRoutine()
	go c.Read()

	return nil
}

func (c *PostOnlyClient) Close() error {
	// Cancel context to stop ping goroutine
	if c.cancel != nil {
		c.cancel()
	}

	if c.conn != nil {
		err := c.conn.Close()
		return err
	}

	return nil
}

// pingRoutine runs in a goroutine and sends periodic ping messages
// It stops when the context is canceled
func (c *PostOnlyClient) pingRoutine() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	defer func() {
		c.Close()
		c.respWaitersMu.Lock()
		for _, waiter := range c.respWaiters {
			waiter.ch <- &PostResponse{Err: fmt.Errorf("websocket closed")}
			close(waiter.ch)
		}
		c.respWaiters = make(map[int64]PostOnlyRespWaiter)
		c.respWaitersMu.Unlock()
	}()

	for {
		select {
		case <-c.ctx.Done():
			// Context canceled - stop ping routine
			return
		case <-ticker.C:
			// Check connection status (no lock needed - single threaded use)
			conn := c.conn

			if conn != nil {
				// Send ping with write lock (only lock needed for concurrent writes)
				pingMsg := map[string]string{"method": "ping"}
				err := conn.WriteJSON(pingMsg)

				if err != nil {
					// Failed to send ping - connection likely broken
					return
				}
			}
		}
	}
}

func (c *PostOnlyClient) Read() {
	for {
		if c.ctx.Err() != nil {
			return
		}
		// Read raw message (blocking)
		_, rawMsg, readErr := c.conn.ReadMessage()
		if readErr != nil {
			return
		}

		// Handle text messages like "Websocket connection established."
		if len(rawMsg) > 0 && rawMsg[0] != '{' {
			// should not happen
			// TODO: log error
			continue
		}

		resp := &PostResponse{}

		// Parse message structure
		if unmarshalErr := json.Unmarshal(rawMsg, resp); unmarshalErr != nil {
			// should not happen
			// TODO: log error
			continue
		}

		id := resp.Data.ID

		c.respWaitersMu.Lock()
		waiter, ok := c.respWaiters[id]
		if !ok {
			// should not happen
			// TODO: log error
			c.respWaitersMu.Unlock()
			continue
		}
		delete(c.respWaiters, id)
		c.respWaitersMu.Unlock()

		if resp.Data.Response.Type == PostResponseError {
			resp.Err = fmt.Errorf("%v", string(resp.Data.Response.Payload))
		}

		waiter.ch <- resp

		close(waiter.ch)
	}
}

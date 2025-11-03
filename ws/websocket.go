// Package ws provides WebSocket client functionality for Hyperliquid real-time data.
package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// DefaultURL is the default Hyperliquid WebSocket URL
	DefaultURL = "wss://api.hyperliquid.xyz/ws"
)

// wsMessage represents the raw WebSocket message structure
type wsMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

// Client is a generic WebSocket client that subscribes to a single feed
// Designed for single-threaded use: one goroutine calls Read() in a loop
type Client[T any] struct {
	url          string
	conn         *websocket.Conn
	subscription map[string]any
	// writeMu      sync.Mutex // Serializes WebSocket writes (for ping goroutine)
	isConnected  bool
	ctx          context.Context
	cancel       context.CancelFunc
	pingInterval time.Duration
}

// newClient creates a new WebSocket client for a specific data type
// The subscription parameter should match the Hyperliquid subscription format:
//
//	Example for trades:
//	  subscription := Subscription{
//	    Type: SubscriptionTrades,
//	    Coin: &"BTC",
//	  }
//
//	Example for user fills:
//	  subscription := Subscription{
//	    Type: SubscriptionUserFills,
//	    User: &"0x1234...",
//	  }
func newClient[T any](url string, subscription map[string]any) *Client[T] {
	return &Client[T]{
		url:          url,
		subscription: subscription,
		isConnected:  false,
		pingInterval: 50 * time.Second, // Default ping interval
	}
}

// subscriptionHandler converts the subscription into a list of subscription messages
// If any field contains a slice, it will expand into multiple subscriptions
func (c *Client[T]) subscriptionHandler() []map[string]any {
	// Check if any field is a slice
	var sliceField string
	var sliceValues []string

	for key, value := range c.subscription {
		// Check if value is a string slice
		if strSlice, ok := value.([]string); ok {
			sliceField = key
			sliceValues = strSlice
			break
		}
	}

	// If no slice field found, return single subscription
	if sliceField == "" {
		return []map[string]any{
			{
				"method":       "subscribe",
				"subscription": c.subscription,
			},
		}
	}

	// Expand slice field into multiple subscriptions
	var result []map[string]any
	for _, value := range sliceValues {
		// Create a copy of the subscription
		sub := make(map[string]any)
		for k, v := range c.subscription {
			if k == sliceField {
				sub[k] = value // Replace slice with single value
			} else {
				sub[k] = v
			}
		}

		result = append(result, map[string]any{
			"method":       "subscribe",
			"subscription": sub,
		})
	}

	return result
}

// start connects to the WebSocket and subscribes to the specified feed
// It also starts a background goroutine to send ping messages periodically
// Not thread-safe: should only be called from Read() once
func (c *Client[T]) start() error {
	fmt.Println("[DEBUG] start() called")

	if c.isConnected {
		return fmt.Errorf("client already started")
	}

	// Create context for controlling the ping goroutine
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Connect to WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	fmt.Printf("[DEBUG] Dialing %s...\n", c.url)
	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		c.cancel()
		fmt.Printf("[DEBUG] Dial failed: %v\n", err)
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}
	fmt.Println("[DEBUG] Dial succeeded")

	c.conn = conn
	c.isConnected = true

	// Read and ignore the "Websocket connection established." message
	fmt.Println("[DEBUG] Reading initial message...")
	_, msg, err := conn.ReadMessage()
	if err != nil {
		c.conn.Close()
		c.isConnected = false
		c.cancel()
		fmt.Printf("[DEBUG] Failed to read initial message: %v\n", err)
		return fmt.Errorf("failed to read initial message: %w", err)
	}
	fmt.Printf("[DEBUG] Initial message: %s\n", string(msg))

	// Send subscription messages
	subs := c.subscriptionHandler()
	fmt.Printf("[DEBUG] Sending %d subscription(s)\n", len(subs))
	for i, sub := range subs {
		s, _ := json.Marshal(sub)
		fmt.Printf("[DEBUG] Subscription %d: %s\n", i, string(s))
		if err = conn.WriteJSON(sub); err != nil {
			c.conn.Close()
			c.isConnected = false
			c.cancel()
			fmt.Printf("[DEBUG] WriteJSON failed: %v\n", err)
			return fmt.Errorf("failed to send subscription: %w", err)
		}
	}
	fmt.Println("[DEBUG] All subscriptions sent")

	// Start ping goroutine
	fmt.Println("[DEBUG] Starting ping goroutine")
	go c.pingRoutine()

	return nil
}

// Read blocks until data is received and returns the unmarshaled data
// It automatically connects if not connected, and closes on error
// It skips subscription response messages and only returns actual data
// Not thread-safe: should only be called from a single goroutine
func (c *Client[T]) Read() (data T, err error) {
	fmt.Println("[DEBUG] Read() called")

	// Use defer to automatically close on error
	defer func() {
		if err != nil {
			fmt.Printf("[DEBUG] Read() error, closing: %v\n", err)
			c.Close()
		}
	}()

	// Auto-start if not connected
	if !c.isConnected || c.conn == nil {
		fmt.Println("[DEBUG] Not connected, calling start()")
		if err = c.start(); err != nil {
			fmt.Printf("[DEBUG] start() failed: %v\n", err)
			return data, fmt.Errorf("failed to start client: %w", err)
		}
		fmt.Println("[DEBUG] start() succeeded")
	}

	conn := c.conn
	if conn == nil {
		err = fmt.Errorf("client not connected")
		fmt.Println("[DEBUG] conn is nil after start")
		return
	}

	fmt.Println("[DEBUG] Entering read loop")
	for {
		// Read raw message (blocking)
		fmt.Println("[DEBUG] Calling ReadMessage()...")
		msgType, rawMsg, readErr := conn.ReadMessage()
		if readErr != nil {
			err = readErr
			fmt.Printf("[DEBUG] ReadMessage() error: %v\n", readErr)
			return
		}

		// DEBUG: Print message type and first 100 chars
		fmt.Printf("[DEBUG] msgType=%d, len=%d, msg=%s\n", msgType, len(rawMsg), string(rawMsg[:min(len(rawMsg), 100)]))

		// Handle text messages like "Websocket connection established."
		if len(rawMsg) > 0 && rawMsg[0] != '{' {
			// Skip non-JSON messages
			fmt.Println("[DEBUG] Skipping non-JSON message")
			continue
		}

		// Parse message structure
		var msg wsMessage
		if unmarshalErr := json.Unmarshal(rawMsg, &msg); unmarshalErr != nil {
			// If it's not a valid wsMessage, skip it
			fmt.Printf("[DEBUG] Failed to unmarshal wsMessage: %v\n", unmarshalErr)
			continue
		}

		fmt.Printf("[DEBUG] Parsed wsMessage: channel=%s\n", msg.Channel)

		// Skip subscription response and pong messages
		if msg.Channel == "subscriptionResponse" || msg.Channel == "pong" {
			fmt.Printf("[DEBUG] Skipping %s message\n", msg.Channel)
			continue
		}

		// Unmarshal data to the specified type
		if unmarshalErr := json.Unmarshal(msg.Data, &data); unmarshalErr != nil {
			err = fmt.Errorf("failed to unmarshal data: %w", unmarshalErr)
			return
		}

		fmt.Println("[DEBUG] Successfully unmarshaled data, returning")
		return data, nil
	}
}

// min helper for debug output
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Close closes the WebSocket connection and stops the ping goroutine
// Safe to call multiple times
func (c *Client[T]) Close() error {
	// Cancel context to stop ping goroutine
	if c.cancel != nil {
		c.cancel()
	}

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return err
	}

	return nil
}

// pingRoutine runs in a goroutine and sends periodic ping messages
// It stops when the context is canceled
func (c *Client[T]) pingRoutine() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Context canceled - stop ping routine
			return
		case <-ticker.C:
			// Check connection status (no lock needed - single threaded use)
			conn := c.conn
			connected := c.isConnected

			if conn != nil && connected {
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

// Helper functions for creating common subscriptions

// NewTradesClient creates a client for subscribing to trades
// Can subscribe to single or multiple coins:
//
//	NewTradesClient("BTC")           // single coin
//	NewTradesClient("BTC", "ETH")    // multiple coins
func NewTradesClient(coins ...string) *Client[[]WsTrade] {
	sub := map[string]any{
		"type": "trades",
	}
	if len(coins) == 1 {
		sub["coin"] = coins[0]
	} else {
		sub["coin"] = coins
	}
	return newClient[[]WsTrade](DefaultURL, sub)
}

// NewL2BookClient creates a client for subscribing to order book updates
// Can subscribe to single or multiple coins:
//
//	NewL2BookClient("BTC")           // single coin
//	NewL2BookClient("BTC", "ETH")    // multiple coins
func NewL2BookClient(coins ...string) *Client[WsBook] {
	sub := map[string]any{
		"type": "l2Book",
	}
	if len(coins) == 1 {
		sub["coin"] = coins[0]
	} else {
		sub["coin"] = coins
	}
	return newClient[WsBook](DefaultURL, sub)
}

// NewUserFillsClient creates a client for subscribing to user fills
func NewUserFillsClient(user string) *Client[WsUserFills] {
	return newClient[WsUserFills](DefaultURL, map[string]any{
		"type": "userFills",
		"user": user,
	})
}

// NewOrderUpdatesClient creates a client for subscribing to order updates
func NewOrderUpdatesClient(user string) *Client[[]WsOrder] {
	return newClient[[]WsOrder](DefaultURL, map[string]any{
		"type": "orderUpdates",
		"user": user,
	})
}

// NewUserEventsClient creates a client for subscribing to user events
func NewUserEventsClient(user string) *Client[WsUserEvent] {
	return newClient[WsUserEvent](DefaultURL, map[string]any{
		"type": "userEvents",
		"user": user,
	})
}

// NewCandleClient creates a client for subscribing to candle updates
// Can subscribe to single or multiple coins:
//
//	NewCandleClient("1m", "BTC")           // single coin
//	NewCandleClient("1m", "BTC", "ETH")    // multiple coins
func NewCandleClient(interval string, coins ...string) *Client[[]Candle] {
	sub := map[string]any{
		"type":     "candle",
		"interval": interval,
	}
	if len(coins) == 1 {
		sub["coin"] = coins[0]
	} else {
		sub["coin"] = coins
	}
	return newClient[[]Candle](DefaultURL, sub)
}

// NewAllMidsClient creates a client for subscribing to all mid prices
func NewAllMidsClient() *Client[AllMids] {
	return newClient[AllMids](DefaultURL, map[string]any{
		"type": "allMids",
	})
}

// NewBboClient creates a client for subscribing to best bid/offer updates
// Can subscribe to single or multiple coins:
//
//	NewBboClient("BTC")           // single coin
//	NewBboClient("BTC", "ETH")    // multiple coins
func NewBboClient(coins ...string) *Client[WsBbo] {
	sub := map[string]any{
		"type": "bbo",
	}
	if len(coins) == 1 {
		sub["coin"] = coins[0]
	} else {
		sub["coin"] = coins
	}
	return newClient[WsBbo](DefaultURL, sub)
}

// NewUserFundingsClient creates a client for subscribing to user funding payments
func NewUserFundingsClient(user string) *Client[WsUserFundings] {
	return newClient[WsUserFundings](DefaultURL, map[string]any{
		"type": "userFundings",
		"user": user,
	})
}

// NewActiveAssetCtxClient creates a client for subscribing to active asset context
// Can subscribe to single or multiple coins:
//
//	NewActiveAssetCtxClient("BTC")           // single coin
//	NewActiveAssetCtxClient("BTC", "ETH")    // multiple coins
func NewActiveAssetCtxClient(coins ...string) *Client[any] {
	sub := map[string]any{
		"type": "activeAssetCtx",
	}
	if len(coins) == 1 {
		sub["coin"] = coins[0]
	} else {
		sub["coin"] = coins
	}
	return newClient[any](DefaultURL, sub)
}

// NewActiveAssetDataClient creates a client for subscribing to active asset data
func NewActiveAssetDataClient(user string, coin string) *Client[WsActiveAssetData] {
	return newClient[WsActiveAssetData](DefaultURL, map[string]any{
		"type": "activeAssetData",
		"user": user,
		"coin": coin,
	})
}

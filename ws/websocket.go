// Package ws provides WebSocket client functionality for Hyperliquid real-time data.
//
// This package offers a simple, type-safe way to subscribe to Hyperliquid WebSocket feeds.
// Each client uses Go generics to provide compile-time type safety for the data you receive.
//
// Basic usage:
//
//	client := ws.NewTradesClient("BTC")
//	defer client.Close()
//
//	for {
//	    trades, err := client.Read()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    // Process trades...
//	}
//
// Features:
//   - Automatic connection on first Read()
//   - Automatic heartbeat (ping every 50s)
//   - Automatic cleanup on error
//   - Support for multiple subscriptions (e.g., multiple coins)
//   - Type-safe data structures
//
// Concurrency:
//
// Each Client instance is designed for single-threaded use. Create multiple
// clients if you need to subscribe to multiple feeds concurrently:
//
//	go func() {
//	    client := ws.NewTradesClient("BTC")
//	    defer client.Close()
//	    for {
//	        trades, _ := client.Read()
//	        // Process BTC trades...
//	    }
//	}()
//
//	go func() {
//	    client := ws.NewL2BookClient("ETH")
//	    defer client.Close()
//	    for {
//	        book, _ := client.Read()
//	        // Process ETH order book...
//	    }
//	}()
package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// MainnetWsURL is the default Hyperliquid WebSocket URL
	MainnetWsURL = "wss://api.hyperliquid.xyz/ws"
)

// wsMessage represents the raw WebSocket message structure
type wsMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

// Client is a generic WebSocket client that subscribes to a single feed.
//
// The client handles connection management, heartbeat, and data unmarshaling automatically.
// Use the New*Client() helper functions to create clients for specific feed types.
//
// Designed for single-threaded use: one goroutine calls Read() in a loop.
// The background ping routine is the only concurrent operation.
//
// Type parameter T specifies the data type returned by Read().
type Client[T any] struct {
	url          string
	conn         *websocket.Conn
	subscription map[string]any
	isConnected  bool
	writeMu      sync.Mutex
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
		pingInterval: 40 * time.Second, // Default ping interval
	}
}

// subscriptionHandler converts the subscription into a list of subscription messages
// If any field contains a slice, it will expand into multiple subscriptions
func (c *Client[T]) subscriptionHandler() []map[string]any {
	if len(c.subscription) == 0 {
		return []map[string]any{}
	}

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

func (c *Client[T]) Write(msg any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("client not connected")
	}
	return c.conn.WriteJSON(msg)
}

// start connects to the WebSocket and subscribes to the specified feed
// It also starts a background goroutine to send ping messages periodically
// Not thread-safe: should only be called from Read() once
func (c *Client[T]) start() error {
	if c.isConnected {
		return fmt.Errorf("client already started")
	}

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
	c.isConnected = true

	// Send subscription messages
	subs := c.subscriptionHandler()
	for _, sub := range subs {
		if err = c.Write(sub); err != nil {
			c.conn.Close()
			c.isConnected = false
			c.cancel()
			return fmt.Errorf("failed to send subscription: %w", err)
		}
	}

	// Start ping goroutine
	go c.pingRoutine()

	return nil
}

// Read blocks until data is received and returns the unmarshaled data.
//
// On first call, Read automatically establishes the WebSocket connection and
// subscribes to the configured feed. Subsequent calls read from the existing connection.
//
// Read filters out subscription responses and pong messages, returning only actual data.
// Non-JSON messages (like "Websocket connection established.") are also skipped.
//
// If an error occurs, the connection is automatically closed before returning.
// After an error, you must create a new client to reconnect.
//
// Not thread-safe: should only be called from a single goroutine.
func (c *Client[T]) Read() (data T, err error) {
	// Use defer to automatically close on error
	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	// Auto-start if not connected
	if !c.isConnected || c.conn == nil {
		if err = c.start(); err != nil {
			return data, fmt.Errorf("failed to start client: %w", err)
		}
	}

	conn := c.conn
	if conn == nil {
		err = fmt.Errorf("client not connected")
		return
	}

	for {
		// Read raw message (blocking)
		_, rawMsg, readErr := conn.ReadMessage()
		if readErr != nil {
			err = readErr
			return
		}

		// Handle text messages like "Websocket connection established."
		if len(rawMsg) > 0 && rawMsg[0] != '{' {
			// Skip non-JSON messages
			continue
		}

		// Parse message structure
		var msg wsMessage
		if unmarshalErr := json.Unmarshal(rawMsg, &msg); unmarshalErr != nil {
			// If it's not a valid wsMessage, skip it
			continue
		}
		
		if len(msg.Data) >= 20 &&
			string(msg.Data[:20]) == `{"method":"subscribe` {
			continue
		}
		
		// Unmarshal data to the specified type
		if unmarshalErr := json.Unmarshal(msg.Data, &data); unmarshalErr != nil {
			err = fmt.Errorf("failed to unmarshal data: %w", unmarshalErr)
			return
		}

		return data, nil
	}
}

// Close closes the WebSocket connection and stops the ping goroutine.
//
// This method:
//   - Cancels the background ping routine
//   - Closes the WebSocket connection
//   - Resets the connection state
//
// Safe to call multiple times. Subsequent calls after the first are no-ops.
// Close is automatically called by Read() when an error occurs.
func (c *Client[T]) Close() error {
	// Cancel context to stop ping goroutine
	if c.cancel != nil {
		c.cancel()
	}

	if c.conn != nil {
		err := c.conn.Close()
		// c.conn = nil
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
	return newClient[[]WsTrade](MainnetWsURL, sub)
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
	return newClient[WsBook](MainnetWsURL, sub)
}

// NewUserFillsClient creates a client for subscribing to user fills
func NewUserFillsClient(user string) *Client[WsUserFills] {
	return newClient[WsUserFills](MainnetWsURL, map[string]any{
		"type": "userFills",
		"user": user,
	})
}

// NewOrderUpdatesClient creates a client for subscribing to order updates
func NewOrderUpdatesClient(user string) *Client[[]WsOrder] {
	return newClient[[]WsOrder](MainnetWsURL, map[string]any{
		"type": "orderUpdates",
		"user": user,
	})
}

// NewUserEventsClient creates a client for subscribing to user events
func NewUserEventsClient(user string) *Client[WsUserEvent] {
	return newClient[WsUserEvent](MainnetWsURL, map[string]any{
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
	return newClient[[]Candle](MainnetWsURL, sub)
}

// NewAllMidsClient creates a client for subscribing to all mid prices
func NewAllMidsClient() *Client[AllMids] {
	return newClient[AllMids](MainnetWsURL, map[string]any{
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
	return newClient[WsBbo](MainnetWsURL, sub)
}

// NewUserFundingsClient creates a client for subscribing to user funding payments
func NewUserFundingsClient(user string) *Client[WsUserFundings] {
	return newClient[WsUserFundings](MainnetWsURL, map[string]any{
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
	return newClient[any](MainnetWsURL, sub)
}

// NewActiveAssetDataClient creates a client for subscribing to active asset data
func NewActiveAssetDataClient(user string, coin string) *Client[WsActiveAssetData] {
	return newClient[WsActiveAssetData](MainnetWsURL, map[string]any{
		"type": "activeAssetData",
		"user": user,
		"coin": coin,
	})
}

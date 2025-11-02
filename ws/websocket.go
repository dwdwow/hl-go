// Package ws provides WebSocket client functionality for Hyperliquid real-time data.
package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/dwdwow/hl-go/types"
)

// MessageCallback is the type for subscription callbacks
type MessageCallback func(message map[string]any)

// ActiveSubscription represents an active WebSocket subscription
type ActiveSubscription struct {
	callback       MessageCallback
	subscriptionID int
}

// Manager manages WebSocket connections and subscriptions
type Manager struct {
	baseURL               string
	conn                  *websocket.Conn
	subscriptionIDCounter int
	wsReady               bool
	queuedSubscriptions   []struct {
		subscription types.Subscription
		active       ActiveSubscription
	}
	activeSubscriptions map[string][]ActiveSubscription
	mu                  sync.RWMutex
	ctx                 context.Context
	cancel              context.CancelFunc
	pingInterval        time.Duration
}

// NewManager creates a new WebSocket manager
func NewManager(baseURL string) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		baseURL:               baseURL,
		subscriptionIDCounter: 0,
		wsReady:               false,
		queuedSubscriptions: make([]struct {
			subscription types.Subscription
			active       ActiveSubscription
		}, 0),
		activeSubscriptions: make(map[string][]ActiveSubscription),
		ctx:                 ctx,
		cancel:              cancel,
		pingInterval:        50 * time.Second,
	}
}

// Start starts the WebSocket connection
func (m *Manager) Start() error {
	// Construct WebSocket URL
	wsURL := strings.Replace(m.baseURL, "http", "ws", 1) + "/ws"

	// Connect
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	m.conn = conn
	m.wsReady = true

	// Process queued subscriptions
	m.mu.Lock()
	for _, queued := range m.queuedSubscriptions {
		if err := m.subscribe(queued.subscription, queued.active.callback, &queued.active.subscriptionID); err != nil {
			log.Printf("Failed to subscribe to queued subscription: %v", err)
		}
	}
	m.queuedSubscriptions = nil
	m.mu.Unlock()

	// Start ping routine
	go m.pingRoutine()

	// Start message handler
	go m.messageHandler()

	return nil
}

// Stop stops the WebSocket connection
func (m *Manager) Stop() error {
	m.cancel()

	if m.conn != nil {
		return m.conn.Close()
	}

	return nil
}

// pingRoutine sends periodic ping messages
func (m *Manager) pingRoutine() {
	ticker := time.NewTicker(m.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			if m.conn != nil {
				ping := map[string]string{"method": "ping"}
				if err := m.conn.WriteJSON(ping); err != nil {
					log.Printf("Failed to send ping: %v", err)
				}
			}
			m.mu.Unlock()
		}
	}
}

// messageHandler handles incoming WebSocket messages
func (m *Manager) messageHandler() {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			_, message, err := m.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// Handle connection establishment message
			if string(message) == "Websocket connection established." {
				continue
			}

			// Parse message
			var wsMsg map[string]any
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				log.Printf("Failed to parse message: %v", err)
				continue
			}

			// Handle pong
			channel, ok := wsMsg["channel"].(string)
			if ok && channel == "pong" {
				continue
			}

			// Route message to subscribers
			identifier := m.wsMsgToIdentifier(wsMsg)
			if identifier == "" {
				continue
			}

			m.mu.RLock()
			subs, ok := m.activeSubscriptions[identifier]
			m.mu.RUnlock()

			if !ok || len(subs) == 0 {
				log.Printf("Received message for unexpected subscription: %s", identifier)
				continue
			}

			// Call all callbacks for this subscription
			for _, sub := range subs {
				go sub.callback(wsMsg)
			}
		}
	}
}

// Subscribe subscribes to a WebSocket channel
func (m *Manager) Subscribe(subscription types.Subscription, callback MessageCallback) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscriptionIDCounter++
	subscriptionID := m.subscriptionIDCounter

	if !m.wsReady {
		// Queue subscription
		m.queuedSubscriptions = append(m.queuedSubscriptions, struct {
			subscription types.Subscription
			active       ActiveSubscription
		}{
			subscription: subscription,
			active: ActiveSubscription{
				callback:       callback,
				subscriptionID: subscriptionID,
			},
		})
		return subscriptionID, nil
	}

	return subscriptionID, m.subscribe(subscription, callback, &subscriptionID)
}

// subscribe performs the actual subscription (must be called with lock held or after wsReady)
func (m *Manager) subscribe(subscription types.Subscription, callback MessageCallback, subscriptionID *int) error {
	identifier := m.subscriptionToIdentifier(subscription)

	// Check for userEvents and orderUpdates - only one subscription allowed
	if identifier == "userEvents" || identifier == "orderUpdates" {
		if len(m.activeSubscriptions[identifier]) > 0 {
			return fmt.Errorf("cannot subscribe to %s multiple times", identifier)
		}
	}

	// Add to active subscriptions
	m.activeSubscriptions[identifier] = append(
		m.activeSubscriptions[identifier],
		ActiveSubscription{
			callback:       callback,
			subscriptionID: *subscriptionID,
		},
	)

	// Send subscription message
	msg := map[string]any{
		"method":       "subscribe",
		"subscription": subscription,
	}

	return m.conn.WriteJSON(msg)
}

// Unsubscribe unsubscribes from a WebSocket channel
func (m *Manager) Unsubscribe(subscription types.Subscription, subscriptionID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.wsReady {
		return fmt.Errorf("websocket not ready")
	}

	identifier := m.subscriptionToIdentifier(subscription)

	// Remove from active subscriptions
	subs := m.activeSubscriptions[identifier]
	newSubs := make([]ActiveSubscription, 0)
	for _, sub := range subs {
		if sub.subscriptionID != subscriptionID {
			newSubs = append(newSubs, sub)
		}
	}

	// If no more subscriptions for this identifier, unsubscribe
	if len(newSubs) == 0 {
		msg := map[string]any{
			"method":       "unsubscribe",
			"subscription": subscription,
		}
		if err := m.conn.WriteJSON(msg); err != nil {
			return err
		}
		delete(m.activeSubscriptions, identifier)
	} else {
		m.activeSubscriptions[identifier] = newSubs
	}

	return nil
}

// subscriptionToIdentifier converts a subscription to an identifier string
func (m *Manager) subscriptionToIdentifier(sub types.Subscription) string {
	switch sub.Type {
	case types.SubscriptionAllMids:
		return "allMids"
	case types.SubscriptionL2Book:
		if sub.Coin != nil {
			return fmt.Sprintf("l2Book:%s", strings.ToLower(*sub.Coin))
		}
	case types.SubscriptionTrades:
		if sub.Coin != nil {
			return fmt.Sprintf("trades:%s", strings.ToLower(*sub.Coin))
		}
	case types.SubscriptionUserEvents:
		return "userEvents"
	case types.SubscriptionUserFills:
		if sub.User != nil {
			return fmt.Sprintf("userFills:%s", strings.ToLower(*sub.User))
		}
	case types.SubscriptionCandle:
		if sub.Coin != nil && sub.Interval != nil {
			return fmt.Sprintf("candle:%s,%s", strings.ToLower(*sub.Coin), *sub.Interval)
		}
	case types.SubscriptionOrderUpdates:
		return "orderUpdates"
	case types.SubscriptionUserFundings:
		if sub.User != nil {
			return fmt.Sprintf("userFundings:%s", strings.ToLower(*sub.User))
		}
	case types.SubscriptionUserNonFundingLedgerUpdates:
		if sub.User != nil {
			return fmt.Sprintf("userNonFundingLedgerUpdates:%s", strings.ToLower(*sub.User))
		}
	case types.SubscriptionWebData2:
		if sub.User != nil {
			return fmt.Sprintf("webData2:%s", strings.ToLower(*sub.User))
		}
	case types.SubscriptionBBO:
		if sub.Coin != nil {
			return fmt.Sprintf("bbo:%s", strings.ToLower(*sub.Coin))
		}
	case types.SubscriptionActiveAssetCtx:
		if sub.Coin != nil {
			return fmt.Sprintf("activeAssetCtx:%s", strings.ToLower(*sub.Coin))
		}
	case types.SubscriptionActiveAssetData:
		if sub.Coin != nil && sub.User != nil {
			return fmt.Sprintf("activeAssetData:%s,%s", strings.ToLower(*sub.Coin), strings.ToLower(*sub.User))
		}
	}
	return ""
}

// wsMsgToIdentifier converts a WebSocket message to an identifier string
func (m *Manager) wsMsgToIdentifier(wsMsg map[string]any) string {
	channel, ok := wsMsg["channel"].(string)
	if !ok {
		return ""
	}

	switch channel {
	case "pong":
		return "pong"
	case "allMids":
		return "allMids"
	case "l2Book":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			if coin, ok := data["coin"].(string); ok {
				return fmt.Sprintf("l2Book:%s", strings.ToLower(coin))
			}
		}
	case "trades":
		if data, ok := wsMsg["data"].([]any); ok && len(data) > 0 {
			if trade, ok := data[0].(map[string]any); ok {
				if coin, ok := trade["coin"].(string); ok {
					return fmt.Sprintf("trades:%s", strings.ToLower(coin))
				}
			}
		}
	case "user":
		return "userEvents"
	case "userFills":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			if user, ok := data["user"].(string); ok {
				return fmt.Sprintf("userFills:%s", strings.ToLower(user))
			}
		}
	case "candle":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			coin, coinOk := data["s"].(string)
			interval, intervalOk := data["i"].(string)
			if coinOk && intervalOk {
				return fmt.Sprintf("candle:%s,%s", strings.ToLower(coin), interval)
			}
		}
	case "orderUpdates":
		return "orderUpdates"
	case "userFundings":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			if user, ok := data["user"].(string); ok {
				return fmt.Sprintf("userFundings:%s", strings.ToLower(user))
			}
		}
	case "userNonFundingLedgerUpdates":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			if user, ok := data["user"].(string); ok {
				return fmt.Sprintf("userNonFundingLedgerUpdates:%s", strings.ToLower(user))
			}
		}
	case "webData2":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			if user, ok := data["user"].(string); ok {
				return fmt.Sprintf("webData2:%s", strings.ToLower(user))
			}
		}
	case "bbo":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			if coin, ok := data["coin"].(string); ok {
				return fmt.Sprintf("bbo:%s", strings.ToLower(coin))
			}
		}
	case "activeAssetCtx", "activeSpotAssetCtx":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			if coin, ok := data["coin"].(string); ok {
				return fmt.Sprintf("activeAssetCtx:%s", strings.ToLower(coin))
			}
		}
	case "activeAssetData":
		if data, ok := wsMsg["data"].(map[string]any); ok {
			coin, coinOk := data["coin"].(string)
			user, userOk := data["user"].(string)
			if coinOk && userOk {
				return fmt.Sprintf("activeAssetData:%s,%s", strings.ToLower(coin), strings.ToLower(user))
			}
		}
	}

	return ""
}

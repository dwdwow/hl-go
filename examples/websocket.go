package examples

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dwdwow/hl-go/constants"
	"github.com/dwdwow/hl-go/types"
	"github.com/dwdwow/hl-go/ws"
)

func WebsocketExample() {
	// Create WebSocket manager
	manager := ws.NewManager(constants.TestnetAPIURL)

	// Start WebSocket connection
	if err := manager.Start(); err != nil {
		log.Fatalf("Failed to start WebSocket: %v", err)
	}
	defer manager.Stop()

	fmt.Println("WebSocket connected successfully")

	// Subscribe to all mids
	_, err := manager.Subscribe(types.Subscription{
		Type: types.SubscriptionAllMids,
	}, func(msg map[string]any) {
		data, _ := json.Marshal(msg)
		fmt.Printf("All Mids: %s\n", string(data))
	})
	if err != nil {
		log.Printf("Failed to subscribe to allMids: %v", err)
	}

	// Subscribe to L2 order book for ETH
	ethCoin := "ETH"
	_, err = manager.Subscribe(types.Subscription{
		Type: types.SubscriptionL2Book,
		Coin: &ethCoin,
	}, func(msg map[string]any) {
		// Only print the first 3 levels to avoid spam
		if data, ok := msg["data"].(map[string]any); ok {
			if levels, ok := data["levels"].([]any); ok && len(levels) >= 2 {
				bids := levels[0].([]any)
				asks := levels[1].([]any)

				fmt.Printf("\nETH L2 Book:\n")
				fmt.Printf("  Bids: %d levels\n", len(bids))
				if len(bids) > 0 {
					if bid, ok := bids[0].(map[string]any); ok {
						fmt.Printf("    Best bid: %s @ %s\n", bid["sz"], bid["px"])
					}
				}
				fmt.Printf("  Asks: %d levels\n", len(asks))
				if len(asks) > 0 {
					if ask, ok := asks[0].(map[string]any); ok {
						fmt.Printf("    Best ask: %s @ %s\n", ask["sz"], ask["px"])
					}
				}
			}
		}
	})
	if err != nil {
		log.Printf("Failed to subscribe to l2Book: %v", err)
	}

	// Subscribe to trades for ETH
	_, err = manager.Subscribe(types.Subscription{
		Type: types.SubscriptionTrades,
		Coin: &ethCoin,
	}, func(msg map[string]any) {
		if data, ok := msg["data"].([]any); ok && len(data) > 0 {
			for _, trade := range data {
				if t, ok := trade.(map[string]any); ok {
					fmt.Printf("ETH Trade: %s %s @ %s\n", t["side"], t["sz"], t["px"])
				}
			}
		}
	})
	if err != nil {
		log.Printf("Failed to subscribe to trades: %v", err)
	}

	// Subscribe to BBO (Best Bid/Offer) for BTC
	btcCoin := "BTC"
	_, err = manager.Subscribe(types.Subscription{
		Type: types.SubscriptionBBO,
		Coin: &btcCoin,
	}, func(msg map[string]any) {
		if data, ok := msg["data"].(map[string]any); ok {
			fmt.Printf("BTC BBO update at time %v\n", data["time"])
		}
	})
	if err != nil {
		log.Printf("Failed to subscribe to bbo: %v", err)
	}

	// If you want to subscribe to user events, uncomment below
	// (requires your wallet address)
	/*
		userAddress := "0xYourAddressHere"
		_, err = manager.Subscribe(types.Subscription{
			Type: types.SubscriptionUserEvents,
			User: &userAddress,
		}, func(msg map[string]any) {
			data, _ := json.Marshal(msg)
			fmt.Printf("User Event: %s\n", string(data))
		})
		if err != nil {
			log.Printf("Failed to subscribe to userEvents: %v", err)
		}
	*/

	fmt.Println("\nListening for WebSocket messages... Press Ctrl+C to exit")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Also show a timer so user knows it's running
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nShutting down...")
			return
		case <-ticker.C:
			fmt.Println("Still listening...")
		}
	}
}

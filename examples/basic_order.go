package examples

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dwdwow/hl-go/client"
	"github.com/dwdwow/hl-go/constants"
	"github.com/dwdwow/hl-go/types"
)

func BasicOrderExample() {
	// Load private key from environment variable
	privateKeyHex := os.Getenv("HYPERLIQUID_PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("HYPERLIQUID_PRIVATE_KEY environment variable not set")
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// Create exchange client (using testnet)
	exchange, err := client.NewExchange(
		privateKey,
		constants.TestnetAPIURL,
		30*time.Second,
		nil, // no vault address
		nil, // account address same as wallet
	)
	if err != nil {
		log.Fatalf("Failed to create exchange client: %v", err)
	}

	address := exchange.GetAccountAddress()
	fmt.Printf("Trading with account: %s\n", address)

	// Create info client to check positions
	info, err := client.NewInfo(constants.TestnetAPIURL, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to create info client: %v", err)
	}

	// Get user state
	userState, err := info.UserState(address, "")
	if err != nil {
		log.Fatalf("Failed to get user state: %v", err)
	}

	// Print positions
	if len(userState.AssetPositions) > 0 {
		fmt.Println("\nCurrent positions:")
		for _, assetPos := range userState.AssetPositions {
			posJSON, _ := json.MarshalIndent(assetPos.Position, "  ", "  ")
			fmt.Printf("  %s\n", string(posJSON))
		}
	} else {
		fmt.Println("\nNo open positions")
	}

	// Place a limit order (this will rest because price is low)
	fmt.Println("\nPlacing limit order for 0.2 ETH at $1100...")

	orderType := types.OrderType{
		Limit: &types.LimitOrderType{Tif: types.TifGtc},
	}

	result, err := exchange.Order(
		"ETH",     // coin
		true,      // is buy
		0.2,       // size
		1100.0,    // limit price
		orderType, // order type
		false,     // reduce only
		nil,       // cloid
		nil,       // builder
	)
	if err != nil {
		log.Fatalf("Failed to place order: %v", err)
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Order result: %s\n", string(resultJSON))

	// Extract order ID if order was successful
	if status, ok := result["status"].(string); ok && status == "ok" {
		if response, ok := result["response"].(map[string]any); ok {
			if data, ok := response["data"].(map[string]any); ok {
				if statuses, ok := data["statuses"].([]any); ok && len(statuses) > 0 {
					if statusMap, ok := statuses[0].(map[string]any); ok {
						if resting, ok := statusMap["resting"].(map[string]any); ok {
							if oid, ok := resting["oid"].(float64); ok {
								fmt.Printf("\nOrder placed successfully with OID: %d\n", int(oid))

								// Query order status
								orderStatus, err := info.QueryOrderByOid(address, int(oid))
								if err == nil {
									statusJSON, _ := json.MarshalIndent(orderStatus, "", "  ")
									fmt.Printf("Order status: %s\n", string(statusJSON))
								}

								// Cancel the order
								fmt.Println("\nCancelling order...")
								cancelResult, err := exchange.Cancel("ETH", int(oid))
								if err != nil {
									log.Printf("Failed to cancel order: %v", err)
								} else {
									cancelJSON, _ := json.MarshalIndent(cancelResult, "", "  ")
									fmt.Printf("Cancel result: %s\n", string(cancelJSON))
								}
							}
						}
					}
				}
			}
		}
	}
}

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
)

func MarketOrderExample() {
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

	// Place a market order to buy 0.01 ETH
	// Market orders use IOC (Immediate or Cancel) with slippage protection
	fmt.Println("\nPlacing market order to buy 0.01 ETH...")

	result, err := exchange.MarketOpen(
		"ETH",                     // coin
		true,                      // is buy
		0.01,                      // size
		nil,                       // px (will fetch current mid price)
		constants.DefaultSlippage, // 5% slippage
		nil,                       // cloid
		nil,                       // builder
	)
	if err != nil {
		log.Fatalf("Failed to place market order: %v", err)
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Market order result: %s\n", string(resultJSON))

	// Get updated positions
	info, err := client.NewInfo(constants.TestnetAPIURL, 30*time.Second)
	if err != nil {
		log.Fatalf("Failed to create info client: %v", err)
	}

	time.Sleep(1 * time.Second) // Give time for order to fill

	userState, err := info.UserState(address, "")
	if err != nil {
		log.Fatalf("Failed to get user state: %v", err)
	}

	fmt.Println("\nUpdated positions:")
	for _, assetPos := range userState.AssetPositions {
		if assetPos.Position.Coin == "ETH" {
			posJSON, _ := json.MarshalIndent(assetPos.Position, "  ", "  ")
			fmt.Printf("  %s\n", string(posJSON))
		}
	}
}

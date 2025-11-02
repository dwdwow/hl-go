# Hyperliquid Go SDK

A robust, efficient, and easy-to-use Go SDK for the [Hyperliquid](https://hyperliquid.xyz) decentralized exchange.

## Features

-  **Complete API Coverage**: Full support for trading, market data, and account management
- = **Secure Signing**: EIP-712 signing for all transactions
- =� **High Performance**: Efficient HTTP client with connection pooling
- =� **WebSocket Support**: Real-time market data and user event subscriptions
- =� **Type Safe**: Strongly typed with comprehensive error handling
- =� **Well Documented**: Extensive documentation and examples
- >� **Production Ready**: Robust error handling and retry logic

## Installation

```bash
go get github.com/dwdwow/hl-go
```

## Quick Start

### Basic Order Placement

```go
package main

import (
    "log"
    "time"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/dwdwow/hl-go/client"
    "github.com/dwdwow/hl-go/constants"
    "github.com/dwdwow/hl-go/types"
)

func main() {
    // Parse private key
    privateKey, err := crypto.HexToECDSA("your_private_key_hex")
    if err != nil {
        log.Fatal(err)
    }

    // Create exchange client
    exchange, err := client.NewExchange(
        privateKey,
        constants.MainnetAPIURL,
        30*time.Second,
        nil, // vault address (optional)
        nil, // account address (optional, for API wallets)
    )
    if err != nil {
        log.Fatal(err)
    }

    // Place a limit order
    orderType := types.OrderType{
        Limit: &types.LimitOrderType{Tif: types.TifGtc},
    }

    result, err := exchange.Order(
        "ETH",     // coin
        true,      // is buy
        0.1,       // size
        2000.0,    // limit price
        orderType,
        false,     // reduce only
        nil,       // cloid
        nil,       // builder
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Order result: %+v", result)
}
```

### Market Data Query

```go
package main

import (
    "log"
    "time"
    "github.com/dwdwow/hl-go/client"
    "github.com/dwdwow/hl-go/constants"
)

func main() {
    // Create info client
    info, err := client.NewInfo(constants.MainnetAPIURL, 30*time.Second)
    if err != nil {
        log.Fatal(err)
    }

    // Get all mid prices
    mids, err := info.AllMids("")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("BTC mid price: %s", mids["BTC"])

    // Get user state
    userState, err := info.UserState("0x...", "")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Account value: %s", userState.MarginSummary.AccountValue)
}
```

### WebSocket Subscriptions

```go
package main

import (
    "log"
    "github.com/dwdwow/hl-go/constants"
    "github.com/dwdwow/hl-go/types"
    "github.com/dwdwow/hl-go/ws"
)

func main() {
    // Create WebSocket manager
    manager := ws.NewManager(constants.MainnetAPIURL)

    // Start connection
    if err := manager.Start(); err != nil {
        log.Fatal(err)
    }
    defer manager.Stop()

    // Subscribe to trades
    coin := "ETH"
    _, err := manager.Subscribe(types.Subscription{
        Type: types.SubscriptionTrades,
        Coin: &coin,
    }, func(msg map[string]any) {
        log.Printf("Trade: %+v", msg)
    })
    if err != nil {
        log.Fatal(err)
    }

    // Keep running
    select {}
}
```

## API Reference

### Client Packages

#### `client.Info`

Read-only client for querying market data and account information.

**Market Data Methods:**
- `AllMids(dex string)` - Get all mid prices
- `L2Snapshot(name string)` - Get L2 order book
- `CandlesSnapshot(name, interval string, startTime, endTime int64)` - Get candlestick data
- `Meta(dex string)` - Get perpetual exchange metadata
- `SpotMeta()` - Get spot exchange metadata
- `FundingHistory(name string, startTime int64, endTime *int64)` - Get funding rate history

**Account Methods:**
- `UserState(address, dex string)` - Get user positions and margin
- `SpotUserState(address string)` - Get spot account state
- `OpenOrders(address, dex string)` - Get open orders
- `UserFills(address string)` - Get trade history
- `UserFees(address string)` - Get fee information
- `HistoricalOrders(user string)` - Get order history (max 2000)
- `Portfolio(user string)` - Get portfolio performance

#### `client.Exchange`

Trading client for executing orders and managing positions.

**Order Methods:**
- `Order(name string, isBuy bool, sz, limitPx float64, orderType types.OrderType, reduceOnly bool, cloid *types.Cloid, builder *types.BuilderInfo)` - Place a single order
- `BulkOrders(orders []types.OrderRequest, builder *types.BuilderInfo)` - Place multiple orders
- `MarketOpen(name string, isBuy bool, sz float64, px *float64, slippage float64, cloid *types.Cloid, builder *types.BuilderInfo)` - Market order to open position
- `MarketClose(coin string, sz, px *float64, slippage float64, cloid *types.Cloid, builder *types.BuilderInfo)` - Market order to close position
- `Cancel(name string, oid int)` - Cancel order by ID
- `CancelByCloid(name string, cloid types.Cloid)` - Cancel order by client ID
- `BulkCancel(cancels []types.CancelRequest)` - Cancel multiple orders

**Account Management:**
- `UpdateLeverage(leverage int, name string, isCross bool)` - Update position leverage
- `USDTransfer(amount float64, destination string)` - Transfer USDC
- `USDClassTransfer(amount float64, toPerp bool)` - Transfer between perp and spot
- `CreateSubAccount(name string)` - Create sub-account
- `SetReferrer(code string)` - Set referral code

**Configuration:**
- `SetExpiresAfter(expiresAfter *int64)` - Set expiration time for actions

#### `ws.Manager`

WebSocket client for real-time data subscriptions.

**Methods:**
- `Start()` - Start WebSocket connection
- `Stop()` - Stop WebSocket connection
- `Subscribe(subscription types.Subscription, callback MessageCallback)` - Subscribe to a channel
- `Unsubscribe(subscription types.Subscription, subscriptionID int)` - Unsubscribe from a channel

**Subscription Types:**
- `types.SubscriptionAllMids` - All mid prices
- `types.SubscriptionL2Book` - L2 order book for a coin
- `types.SubscriptionTrades` - Trades for a coin
- `types.SubscriptionBBO` - Best bid/offer for a coin
- `types.SubscriptionCandle` - Candlestick data for a coin
- `types.SubscriptionActiveAssetCtx` - Asset context (funding, open interest, etc.)
- `types.SubscriptionActiveAssetData` - Active asset data for user and coin
- `types.SubscriptionUserEvents` - User trading events
- `types.SubscriptionUserFills` - User trade fills
- `types.SubscriptionOrderUpdates` - Order status updates
- `types.SubscriptionUserFundings` - Funding payments
- `types.SubscriptionUserNonFundingLedgerUpdates` - Non-funding ledger updates
- `types.SubscriptionWebData2` - Web data for a user

### Types

#### Order Types

```go
// Limit order
orderType := types.OrderType{
    Limit: &types.LimitOrderType{
        Tif: types.TifGtc, // Good Till Cancel
        // types.TifIoc - Immediate or Cancel
        // types.TifAlo - Add Liquidity Only
    },
}

// Trigger order (Stop Loss / Take Profit)
orderType := types.OrderType{
    Trigger: &types.TriggerOrderType{
        TriggerPx: 2100.0,
        IsMarket:  true,
        Tpsl:      types.TpslTp, // Take Profit
        // types.TpslSl - Stop Loss
    },
}
```

#### Client Order ID (Cloid)

```go
// Create from integer
cloid := types.NewCloidFromInt(12345)

// Create from hex string
cloid, err := types.NewCloidFromString("0x00000000000000000000000000003039")
```

### Constants

```go
import "github.com/dwdwow/hl-go/constants"

// API URLs
constants.MainnetAPIURL  // "https://api.hyperliquid.xyz"
constants.TestnetAPIURL  // "https://api.hyperliquid-testnet.xyz"
constants.LocalAPIURL    // "http://localhost:3001"

// Configuration
constants.DefaultTimeout   // 30 seconds
constants.DefaultSlippage  // 0.05 (5%)
```

## Advanced Usage

### Using API Wallets

API wallets allow you to trade without exposing your main wallet's private key.

```go
// Main wallet address
mainAddress := "0xYourMainWalletAddress"

// Create exchange with API wallet
exchange, err := client.NewExchange(
    apiWalletPrivateKey,      // API wallet's private key
    constants.MainnetAPIURL,
    30*time.Second,
    nil,           // vault address
    &mainAddress,  // account address (main wallet)
)
```

### Trading with Vaults

```go
vaultAddress := "0xYourVaultAddress"

exchange, err := client.NewExchange(
    privateKey,
    constants.MainnetAPIURL,
    30*time.Second,
    &vaultAddress, // vault address
    nil,
)
```

### Bulk Operations

```go
// Place multiple orders at once
orders := []types.OrderRequest{
    {
        Coin:       "BTC",
        IsBuy:      true,
        Sz:         0.1,
        LimitPx:    30000.0,
        OrderType:  types.OrderType{Limit: &types.LimitOrderType{Tif: types.TifGtc}},
        ReduceOnly: false,
    },
    {
        Coin:       "ETH",
        IsBuy:      false,
        Sz:         1.0,
        LimitPx:    2000.0,
        OrderType:  types.OrderType{Limit: &types.LimitOrderType{Tif: types.TifGtc}},
        ReduceOnly: false,
    },
}

result, err := exchange.BulkOrders(orders, nil)
```

### Custom Timeout

```go
// Create client with custom timeout
info, err := client.NewInfo(constants.MainnetAPIURL, 60*time.Second)
```

### Error Handling

```go
result, err := exchange.Order(...)
if err != nil {
    // Check if it's an API error
    if apiErr, ok := err.(*client.APIError); ok {
        log.Printf("API Error: Status=%d, Code=%s, Message=%s",
            apiErr.StatusCode, *apiErr.Code, apiErr.Message)
    } else {
        log.Printf("Error: %v", err)
    }
    return
}
```

## Examples

See the [examples](./examples) directory for complete working examples:

- `basic_order.go` - Basic order placement and cancellation
- `market_order.go` - Market order execution
- `websocket.go` - WebSocket subscriptions

To run examples:

```bash
export HYPERLIQUID_PRIVATE_KEY="your_private_key_hex"
go run examples/basic_order.go
```

## Project Structure

```
hl-go/
   client/          # API clients (Info, Exchange)
   types/           # Type definitions
   signing/         # EIP-712 signing
   utils/           # Utility functions
   ws/              # WebSocket manager
   constants/       # Configuration constants
   examples/        # Example code
   README.md        # This file
```

## Security

- **Private Keys**: Never commit private keys to version control
- **Environment Variables**: Use environment variables for sensitive data
- **Testnet First**: Always test on testnet before using mainnet
- **API Wallets**: Consider using API wallets for additional security
- **Rate Limits**: Respect API rate limits to avoid being throttled

## Requirements

- Go 1.18 or higher
- Dependencies (automatically installed with `go get`):
  - `github.com/ethereum/go-ethereum`
  - `github.com/gorilla/websocket`
  - `github.com/vmihailenco/msgpack/v5`

## Development

### Building

```bash
go build ./...
```

### Testing

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit pull requests.

## License

MIT License - see LICENSE file for details

## Disclaimer

This SDK is provided as-is. Use at your own risk. Always test thoroughly on testnet before using on mainnet with real funds.

## Support

- [Hyperliquid Documentation](https://hyperliquid.gitbook.io/)
- [Hyperliquid Discord](https://discord.gg/hyperliquid)
- [GitHub Issues](https://github.com/yourusername/hl-go/issues)

## Changelog

### v1.0.0 (2025-01-XX)

- Initial release
- Complete API coverage for Info and Exchange
- WebSocket support
- EIP-712 signing
- Comprehensive examples and documentation

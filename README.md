# HL Go Library

This is a Go library that provides type definitions and utilities for working with Hyperliquid (HL) API.

## Types

The library includes comprehensive type definitions converted from Python TypedDict and Union types to Go structs and interfaces.

### Core Types

- `AssetInfo` - Asset information with name and size decimals
- `Meta` - Metadata containing universe of assets
- `Side` - Trading side (A or B)
- `SpotAssetInfo` - Spot asset information
- `SpotTokenInfo` - Spot token information
- `SpotMeta` - Spot metadata
- `SpotAssetCtx` - Spot asset context

### Subscription Types

The library provides various subscription types for different data streams:

```go
// Create subscriptions
bboSub := NewBboSubscription("BTC")
tradesSub := NewTradesSubscription("ETH")
userEventsSub := NewUserEventsSubscription("user123")
```

### Message Types

WebSocket message types for handling real-time data:

- `AllMidsMsg` - All mids data
- `BboMsg` - Best bid/offer data
- `L2BookMsg` - Level 2 order book data
- `TradesMsg` - Trade data
- `UserEventsMsg` - User events
- `UserFillsMsg` - User fills
- `ActiveAssetCtxMsg` - Active asset context
- `ActiveAssetDataMsg` - Active asset data

### Leverage Types

Support for different leverage types:

```go
// Cross leverage
crossLeverage := NewCrossLeverage(10)

// Isolated leverage
isolatedLeverage := NewIsolatedLeverage(5, "1000.50")
```

### Cloid (Client Order ID)

The `Cloid` type provides validation and utilities for client order IDs:

```go
// Create from string
cloid, err := CloidFromStr("0x1234567890abcdef1234567890abcdef")

// Create from integer
cloid := CloidFromInt(12345)

// Validate and get raw value
raw := cloid.ToRaw()
```

## Usage Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/dwdwow/hl-go"
)

func main() {
    // Create a subscription
    subscription := hl.NewBboSubscription("BTC")
    
    // Create leverage
    leverage := hl.NewCrossLeverage(10)
    
    // Create a trade
    trade := hl.Trade{
        Coin: "BTC",
        Side: hl.SideA,
        Px:   "50000.00",
        Sz:   100,
        Hash: "0x123...",
        Time: 1640995200,
    }
    
    // Marshal to JSON
    data, _ := json.Marshal(trade)
    fmt.Println(string(data))
}
```

## Constants

The library provides constants for subscription types, channels, and leverage types to ensure consistency across your application.

## JSON Support

All types include JSON tags for seamless serialization and deserialization. The `Cloid` type implements custom JSON marshaling/unmarshaling for proper hex string handling.

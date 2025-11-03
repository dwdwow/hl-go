# Hyperliquid Go SDK

A robust, efficient, and comprehensive Go SDK for the [Hyperliquid](https://hyperliquid.xyz) decentralized exchange.

## Features

- ✅ **Complete API Coverage**: All 60+ exchange methods and 40+ info endpoints
- 🔐 **Secure Signing**: EIP-712 signing for all authenticated operations
- ⚡ **High Performance**: Efficient HTTP client with connection pooling
- 📡 **WebSocket Support**: Real-time market data and user event subscriptions
- 🛡️ **Type Safe**: Strongly typed with comprehensive error handling
- 📚 **Well Documented**: Extensive GoDoc comments and examples
- 🚀 **Production Ready**: Battle-tested error handling and retry logic

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
    exchange, err := client.NewExchange(&client.ExchangeOptions{
        Wallet:  privateKey,
        BaseURL: constants.MainnetAPIURL,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Place a limit order
    result, err := exchange.Order(
        "ETH",                                      // coin
        true,                                       // is buy
        0.1,                                        // size
        2000.0,                                     // limit price
        types.OrderType{                            // order type
            Limit: &types.LimitOrderType{
                Tif: types.TifGtc,                  // Good Till Cancel
            },
        },
        false,                                      // reduce only
        nil,                                        // cloid (optional)
        nil,                                        // builder (optional)
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
    "github.com/dwdwow/hl-go/client"
    "github.com/dwdwow/hl-go/constants"
)

func main() {
    // Create info client (no authentication needed)
    info, err := client.NewInfo(&client.InfoOptions{
        BaseURL: constants.MainnetAPIURL,
    })
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

## Complete Feature List

### Exchange Client (`client.Exchange`)

#### Core Trading (8 methods)
- `Order` - Place a single order with full configurability
- `BulkOrders` - Place multiple orders atomically
- `ModifyOrder` - Modify existing order price/size
- `BulkModifyOrders` - Modify multiple orders atomically
- `Cancel` - Cancel order by order ID
- `CancelByCloid` - Cancel order by client order ID
- `BulkCancel` - Cancel multiple orders atomically
- `BulkCancelByCloid` - Cancel multiple orders by client order ID

#### Market Orders (2 methods)
- `MarketOpen` - Open position with market order (uses IOC limit orders)
- `MarketClose` - Close position with market order

#### Position Management (3 methods)
- `UpdateLeverage` - Change leverage for cross or isolated margin
- `UpdateIsolatedMargin` - Add/remove isolated margin
- `ScheduleCancel` - Schedule automatic cancellation of all orders

#### Transfers (7 methods)
- `USDTransfer` - Transfer USDC to another address
- `SpotTransfer` - Transfer spot tokens
- `USDClassTransfer` - Transfer between perp and spot balances
- `SendAsset` - Send assets between DEXs (perp/spot)
- `SubAccountTransfer` - Transfer between main and sub-accounts
- `SubAccountSpotTransfer` - Transfer spot tokens to/from sub-accounts
- `VaultTransfer` - Deposit/withdraw from vaults
- `WithdrawFromBridge` - Withdraw from Hyperliquid bridge

#### Account Management (5 methods)
- `CreateSubAccount` - Create new sub-account
- `SetReferrer` - Set referral code for fee discounts
- `ApproveAgent` - Approve an agent (API wallet) for trading
- `ApproveBuilderFee` - Approve builder for fee sharing
- `ConvertToMultiSigUser` - Convert account to multi-sig

#### Advanced Features (10 methods)
- `TWAPOrder` - Place Time-Weighted Average Price order
- `TWAPCancel` - Cancel TWAP order
- `Noop` - No-operation (for testing signatures)
- `UserDexAbstraction` - Enable DEX abstraction for a user
- `AgentEnableDexAbstraction` - Enable DEX abstraction as agent
- `UseBigBlocks` - Enable big blocks for EVM operations
- `TokenDelegate` - Delegate tokens to validators (staking)
- `MultiSig` - Execute multi-signature transaction
- `SetExpiresAfter` - Set expiration timestamp for actions

#### Spot Deployment (10 methods - HIP-2)
- `SpotDeployRegisterToken` - Register new token
- `SpotDeployUserGenesis` - Distribute tokens to users
- `SpotDeployEnableFreezePrivilege` - Enable freeze capability
- `SpotDeployFreezeUser` - Freeze/unfreeze user tokens
- `SpotDeployRevokeFreezePrivilege` - Revoke freeze capability
- `SpotDeployEnableQuoteToken` - Enable token as quote asset
- `SpotDeployGenesis` - Complete token genesis
- `SpotDeployRegisterSpot` - Register spot trading pair
- `SpotDeployRegisterHyperliquidity` - Setup AMM liquidity
- `SpotDeploySetDeployerTradingFeeShare` - Configure fee sharing

#### Perp Deployment (2 methods - HIP-3)
- `PerpDeployRegisterAsset` - Register perpetual on builder DEX
- `PerpDeploySetOracle` - Set oracle prices for perp DEX

#### Validator Operations (5 methods)
- `CSignerUnjailSelf` - Unjail validator signer
- `CSignerJailSelf` - Jail validator signer
- `CValidatorRegister` - Register as consensus validator
- `CValidatorChangeProfile` - Update validator profile
- `CValidatorUnregister` - Unregister validator

### Info Client (`client.Info`)

#### Market Data (10 methods)
- `AllMids` - Get all mid prices
- `L2Snapshot` - Get L2 orderbook snapshot
- `CandlesSnapshot` - Get historical candles
- `FundingHistory` - Get funding rate history
- `Meta` - Get perpetual exchange metadata
- `MetaAndAssetCtxs` - Get metadata with asset contexts
- `SpotMeta` - Get spot exchange metadata
- `SpotMetaAndAssetCtxs` - Get spot metadata with contexts
- `PerpDexs` - Get all perpetual DEXs
- `FrontendOpenOrders` - Get detailed open orders

#### User Account Data (14 methods)
- `UserState` - Get positions and margin summary
- `SpotUserState` - Get spot account balances
- `OpenOrders` - Get user's open orders
- `UserFills` - Get user's trade history
- `UserFillsByTime` - Get fills in time range
- `UserFundingHistory` - Get funding payment history
- `UserFees` - Get fee tier and trading volume
- `UserNonFundingLedgerUpdates` - Get ledger updates
- `HistoricalOrders` - Get order history (max 2000)
- `Portfolio` - Get portfolio performance data
- `QueryOrderByOid` - Query order status by ID
- `QueryOrderByCloid` - Query order status by client ID
- `QuerySubAccounts` - Get user's sub-accounts
- `QueryReferralState` - Get referral information

#### Staking (4 methods)
- `UserStakingSummary` - Get staking summary
- `UserStakingDelegations` - Get active delegations
- `UserStakingRewards` - Get staking rewards history
- `DelegatorHistory` - Get comprehensive staking history

#### Advanced Queries (8 methods)
- `ExtraAgents` - Get approved agents
- `UserTwapSliceFills` - Get TWAP execution fills
- `UserVaultEquities` - Get vault equity positions
- `UserRole` - Get account role/type
- `UserRateLimit` - Get rate limit status
- `QueryUserToMultiSigSigners` - Get multi-sig signers
- `QueryPerpDeployAuctionStatus` - Get perp deployment auction
- `QueryUserDexAbstractionState` - Get DEX abstraction state
- `QuerySpotDeployAuctionStatus` - Get spot deployment state

### WebSocket Manager (`ws.Manager`)

#### Subscription Types (13 types)
- `SubscriptionAllMids` - All mid prices updates
- `SubscriptionL2Book` - L2 orderbook updates
- `SubscriptionTrades` - Trade executions
- `SubscriptionBBO` - Best bid/offer updates
- `SubscriptionCandle` - Candlestick data
- `SubscriptionActiveAssetCtx` - Asset context (funding, OI)
- `SubscriptionActiveAssetData` - User asset data
- `SubscriptionUserEvents` - User trading events
- `SubscriptionUserFills` - User fill updates
- `SubscriptionOrderUpdates` - Order status changes
- `SubscriptionUserFundings` - Funding payments
- `SubscriptionUserNonFundingLedgerUpdates` - Ledger updates
- `SubscriptionWebData2` - Frontend web data

## Advanced Usage

### Using API Wallets (Agents)

API wallets allow you to trade without exposing your main wallet's private key.

```go
// Main wallet address
mainAddress := "0xYourMainWalletAddress"

// Create exchange with API wallet
exchange, err := client.NewExchange(&client.ExchangeOptions{
    Wallet:         apiWalletPrivateKey,  // API wallet's private key
    BaseURL:        constants.MainnetAPIURL,
    AccountAddress: &mainAddress,          // Main account address
})
```

### Trading with Vaults

```go
vaultAddress := "0xYourVaultAddress"

exchange, err := client.NewExchange(&client.ExchangeOptions{
    Wallet:       privateKey,
    BaseURL:      constants.MainnetAPIURL,
    VaultAddress: &vaultAddress,
})
```

### Bulk Operations

```go
// Place multiple orders atomically
orders := []types.OrderRequest{
    {
        Coin:       "BTC",
        IsBuy:      true,
        Sz:         0.1,
        LimitPx:    50000.0,
        OrderType:  types.OrderType{Limit: &types.LimitOrderType{Tif: types.TifGtc}},
        ReduceOnly: false,
    },
    {
        Coin:       "ETH",
        IsBuy:      false,
        Sz:         1.0,
        LimitPx:    3000.0,
        OrderType:  types.OrderType{Limit: &types.LimitOrderType{Tif: types.TifGtc}},
        ReduceOnly: false,
    },
}

result, err := exchange.BulkOrders(orders, nil)
```

### TWAP Orders

```go
// Place a TWAP order that executes over 1 hour
result, err := exchange.TWAPOrder(
    "BTC",           // coin
    true,            // is buy
    10.0,            // total size
    50000.0,         // limit price
    3600000,         // duration in ms (1 hour)
    false,           // randomize
)
```

### Multi-Signature Transactions

```go
// 1. Create inner action
innerAction := map[string]any{
    "type": "usdSend",
    "destination": "0x...",
    "amount": "1000",
    "time": time.Now().UnixMilli(),
}

// 2. Sign with each authorized user
signatures := []map[string]any{
    // ... signatures from authorized users
}

// 3. Execute multi-sig transaction
result, err := exchange.MultiSig(
    multiSigUserAddress,
    innerAction,
    signatures,
    nonce,
    nil,  // vault address (optional)
)
```

### Spot Token Deployment

```go
// 1. Register token
result, err := exchange.SpotDeployRegisterToken(
    "MYTOKEN",  // token name
    6,          // sz decimals
    18,         // wei decimals
    1000000,    // max gas
    "My Token", // full name
)

// 2. Genesis distribution
result, err := exchange.SpotDeployGenesis(
    tokenIndex,
    "1000000000", // max supply
    false,        // no hyperliquidity
)

// 3. Register spot pair
result, err := exchange.SpotDeployRegisterSpot(
    baseTokenIndex,
    quoteTokenIndex,
)
```

### Client Order IDs (Cloid)

```go
// Create from integer (useful for sequential IDs)
cloid := types.NewCloidFromInt(12345)

// Create from hex string
cloid, err := types.NewCloidFromString("0x00000000000000000000000000003039")

// Use in order
result, err := exchange.Order("ETH", true, 0.1, 2000.0, orderType, false, &cloid, nil)
```

### Error Handling

```go
result, err := exchange.Order(...)
if err != nil {
    log.Printf("Error: %v", err)
    return
}

// Check response status
if result.Status != "ok" {
    log.Printf("Order failed: %s", result.Error)
    return
}

// Check order statuses
for i, status := range result.Response.Data.Statuses {
    if status.Error != "" {
        log.Printf("Order %d failed: %s", i, status.Error)
    } else if status.Filled != nil {
        log.Printf("Order %d filled", i)
    } else if status.Resting != nil {
        log.Printf("Order %d resting with oid %d", i, status.Resting.Oid)
    }
}
```

## Type Reference

### Order Types

```go
// Limit Order - Good Till Cancel
orderType := types.OrderType{
    Limit: &types.LimitOrderType{
        Tif: types.TifGtc,
    },
}

// Limit Order - Immediate or Cancel (like a market order)
orderType := types.OrderType{
    Limit: &types.LimitOrderType{
        Tif: types.TifIoc,
    },
}

// Limit Order - Add Liquidity Only (maker-only)
orderType := types.OrderType{
    Limit: &types.LimitOrderType{
        Tif: types.TifAlo,
    },
}

// Trigger Order - Stop Loss
orderType := types.OrderType{
    Trigger: &types.TriggerOrderType{
        TriggerPx: 1900.0,
        IsMarket:  true,
        Tpsl:      types.TpslSl,
    },
}

// Trigger Order - Take Profit
orderType := types.OrderType{
    Trigger: &types.TriggerOrderType{
        TriggerPx: 2100.0,
        IsMarket:  false,  // false = limit order at trigger price
        Tpsl:      types.TpslTp,
    },
}
```

### Subscription Types

```go
// All mids
subscription := types.Subscription{
    Type: types.SubscriptionAllMids,
}

// L2 book for specific coin
coin := "ETH"
subscription := types.Subscription{
    Type: types.SubscriptionL2Book,
    Coin: &coin,
}

// User fills
user := "0x..."
subscription := types.Subscription{
    Type: types.SubscriptionUserFills,
    User: &user,
}

// Candles
coin := "BTC"
interval := "1m"
subscription := types.Subscription{
    Type:     types.SubscriptionCandle,
    Coin:     &coin,
    Interval: &interval,
}
```

## Constants

```go
import "github.com/dwdwow/hl-go/constants"

// API URLs
constants.MainnetAPIURL   // "https://api.hyperliquid.xyz"
constants.TestnetAPIURL   // "https://api.hyperliquid-testnet.xyz"
constants.LocalAPIURL     // "http://localhost:3001"

// Configuration
constants.DefaultTimeout   // 30 seconds
constants.DefaultSlippage  // 0.05 (5%)
```

## Project Structure

```
hl-go/
├── client/           # API clients (Info, Exchange, API)
├── types/            # Type definitions and structures
├── signing/          # EIP-712 signing implementation
├── utils/            # Utility functions (address, float conversion)
├── ws/               # WebSocket manager
├── constants/        # Configuration constants
└── README.md         # This file
```

## Security Best Practices

- **Private Keys**: Never commit private keys to version control
- **Environment Variables**: Use environment variables for sensitive data
  ```bash
  export HYPERLIQUID_PRIVATE_KEY="your_key_without_0x_prefix"
  ```
- **Testnet First**: Always test on testnet before using mainnet
- **API Wallets**: Consider using API wallets (agents) for additional security
- **Rate Limits**: Respect API rate limits to avoid throttling
- **Vault Permissions**: Carefully manage vault agent permissions
- **Multi-Sig**: Use multi-sig for high-value accounts

## Requirements

- Go 1.18 or higher
- Dependencies (automatically installed):
  - `github.com/ethereum/go-ethereum` - Ethereum crypto and signing
  - `github.com/gorilla/websocket` - WebSocket client
  - `github.com/vmihailenco/msgpack/v5` - MessagePack encoding
  - `github.com/dwdwow/evmutil-go` - EVM utilities

## Development

### Building

```bash
go build ./...
```

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run tests and linting
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Disclaimer

This SDK is provided as-is with no warranties. Trading cryptocurrencies carries significant risk. Always:

- Test thoroughly on testnet first
- Start with small amounts
- Understand the risks involved
- Never invest more than you can afford to lose
- Do your own research (DYOR)

## Resources

- [Hyperliquid Documentation](https://hyperliquid.gitbook.io/)
- [Hyperliquid Discord](https://discord.gg/hyperliquid)
- [API Documentation](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api)
- [EIP-712 Specification](https://eips.ethereum.org/EIPS/eip-712)

## Acknowledgments

This SDK is a complete Go implementation inspired by the official Python SDK, with full feature parity and additional Go-specific improvements for type safety and performance.

// Package client provides Exchange and Info clients for interacting with Hyperliquid.
//
// # Exchange Client
//
// The Exchange client provides comprehensive trading and account management functionality:
//
//   - Order Management: Place, modify, cancel orders (limit, market, trigger, TWAP)
//   - Position Management: Update leverage, isolated margin
//   - Transfers: USD, spot tokens, cross-chain assets
//   - Account Operations: Sub-accounts, vaults, referrals
//   - Advanced Features: Multi-sig, DEX abstraction, agents
//   - Deployment: Spot and perp asset deployment (HIP-2, HIP-3)
//   - Staking: Token delegation to validators
//
// # Info Client
//
// The Info client provides read-only market data and account queries:
//
//   - Market Data: L2 orderbook, trades, candles, funding rates
//   - User Data: Positions, fills, orders, PnL, portfolio
//   - Metadata: Asset info, spot tokens, perp DEXs
//   - Staking: Delegations, rewards, validator info
//
// # Authentication
//
// All Exchange operations require an Ethereum private key for EIP-712 signing.
// The wallet address is derived from this private key and used as the user address.
//
// # Vault Trading
//
// To trade on behalf of a vault, provide the vault address in ExchangeOptions.
// The wallet must be authorized as an agent for the vault.
//
// # API Wallet (Account Address)
//
// If using an API wallet (agent), provide both:
//   - wallet: the agent's private key (for signing)
//   - accountAddress: the main account address (for queries)
//
// # Mainnet vs Testnet
//
// The base URL determines the network:
//   - Mainnet: https://api.hyperliquid.xyz
//   - Testnet: https://api.hyperliquid-testnet.xyz
//
// # Error Handling
//
// All methods return typed responses with Status field:
//   - "ok": successful operation
//   - "err": operation failed, check Error field for details
//
// # Example Usage
//
//	// Create Exchange client
//	privateKey, _ := crypto.HexToECDSA("your-private-key")
//	exchange, _ := client.NewExchange(&client.ExchangeOptions{
//	    Wallet:  privateKey,
//	    BaseURL: constants.MainnetAPIURL,
//	})
//
//	// Place a limit order
//	response, _ := exchange.Order(
//	    "BTC",                              // coin
//	    true,                               // is buy
//	    0.1,                                // size
//	    50000.0,                            // limit price
//	    types.OrderType{                    // order type
//	        Limit: &types.LimitOrderType{
//	            Tif: types.TifGtc,
//	        },
//	    },
//	    false,                              // reduce only
//	    nil,                                // cloid
//	    nil,                                // builder
//	)
package client

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dwdwow/evmutil-go"
	"github.com/dwdwow/hl-go/constants"
	"github.com/dwdwow/hl-go/signing"
	"github.com/dwdwow/hl-go/types"
	"github.com/dwdwow/hl-go/utils"
)

// Exchange provides trading functionality for the Hyperliquid exchange
type Exchange struct {
	*API
	wallet         *ecdsa.PrivateKey
	walletAddress  string
	vaultAddress   *string
	accountAddress *string
	info           *Info
	expiresAfter   *int64
}

type ExchangeOptions struct {
	Wallet         *ecdsa.PrivateKey
	BaseURL        string
	Timeout        time.Duration
	VaultAddress   *string
	AccountAddress *string
}

// NewExchange creates a new Exchange client
// wallet: private key for signing transactions
// vaultAddress: optional vault address for vault trading
// accountAddress: optional account address (if different from wallet address, e.g., when using API wallet)
func NewExchange(
	options *ExchangeOptions,
) (*Exchange, error) {
	if options == nil {
		options = &ExchangeOptions{}
	}
	if options.BaseURL == "" {
		options.BaseURL = constants.MainnetAPIURL
	}

	// Create info client
	info, err := NewInfo(options.BaseURL, options.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create info client: %w", err)
	}

	// Get wallet address
	pubKey := options.Wallet.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to get public key")
	}
	walletAddress := crypto.PubkeyToAddress(*pubKeyECDSA).Hex()

	return &Exchange{
		API:            NewAPI(options.BaseURL, options.Timeout),
		wallet:         options.Wallet,
		walletAddress:  walletAddress,
		vaultAddress:   options.VaultAddress,
		accountAddress: options.AccountAddress,
		info:           info,
	}, nil
}

func NewExchangePrivateKeyFromTerminal(vaultAddress, accountAddress *string) (*Exchange, error) {
	wallet, _, err := evmutil.ReadEncryptedPrivateKeyFromTerminal()
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from terminal: %w", err)
	}
	opts := &ExchangeOptions{
		Wallet:         wallet,
		VaultAddress:   vaultAddress,
		AccountAddress: accountAddress,
	}
	return NewExchange(opts)
}

// SetExpiresAfter sets the expiration time for actions (in milliseconds)
// Set to nil to disable expiration
func (e *Exchange) SetExpiresAfter(expiresAfter *int64) {
	e.expiresAfter = expiresAfter
}

// GetWallet returns the private key
func (e *Exchange) GetWallet() *ecdsa.PrivateKey {
	return e.wallet
}

// GetWalletAddress returns the wallet address
func (e *Exchange) GetWalletAddress() string {
	return e.walletAddress
}

// NameToAsset converts a coin name to asset ID
func (e *Exchange) NameToAsset(name string) (int, error) {
	return e.info.NameToAsset(name)
}

// postAction posts a signed action to the exchange and parses into typed response
func (e *Exchange) postAction(action map[string]any, signature *types.Signature, nonce int64, result any) error {
	// Special handling for usdClassTransfer and sendAsset - they don't use vaultAddress
	actionType, _ := action["type"].(string)
	var vaultAddr *string
	if actionType != "usdClassTransfer" && actionType != "sendAsset" {
		vaultAddr = e.vaultAddress
	}

	payload := map[string]any{
		"action":       action,
		"nonce":        nonce,
		"signature":    signature,
		"vaultAddress": vaultAddr,
	}

	if e.expiresAfter != nil {
		payload["expiresAfter"] = *e.expiresAfter
	}

	return e.exchangePost("/exchange", payload, result)
}

// slippagePrice calculates the price with slippage applied
func (e *Exchange) slippagePrice(name string, isBuy bool, slippage float64, px *float64) (float64, error) {
	coin, ok := e.info.nameToCoin[name]
	if !ok {
		return 0, fmt.Errorf("unknown coin: %s", name)
	}

	// Get mid price if not provided
	price := float64(0)
	if px == nil {
		mids, err := e.info.AllMids("")
		if err != nil {
			return 0, fmt.Errorf("failed to get mid price: %w", err)
		}
		midStr, ok := mids[coin]
		if !ok {
			return 0, fmt.Errorf("no mid price for %s", coin)
		}
		price, _ = strconv.ParseFloat(midStr, 64)
	} else {
		price = *px
	}

	asset, ok := e.info.coinToAsset[coin]
	if !ok {
		return 0, fmt.Errorf("unknown coin: %s", coin)
	}

	// Check if spot asset
	isSpot := asset >= constants.SpotAssetOffset

	// Apply slippage
	if isBuy {
		price *= (1 + slippage)
	} else {
		price *= (1 - slippage)
	}

	// Round to appropriate decimals
	decimals := 6
	if isSpot {
		decimals = 8
	}

	szDecimals, ok := e.info.assetToSzDecimals[asset]
	if !ok {
		szDecimals = 0
	}

	decimals = decimals - szDecimals

	// Round to 5 significant figures and appropriate decimals
	rounded := utils.RoundPrice(price, 5, decimals)

	return rounded, nil
}

// Order places a single order
func (e *Exchange) Order(
	name string,
	isBuy bool,
	sz float64,
	limitPx float64,
	orderType types.OrderType,
	reduceOnly bool,
	cloid *types.Cloid,
	builder *types.BuilderInfo,
) (*types.OrderResponse, error) {
	order := types.OrderRequest{
		Coin:       name,
		IsBuy:      isBuy,
		Sz:         sz,
		LimitPx:    limitPx,
		OrderType:  orderType,
		ReduceOnly: reduceOnly,
		Cloid:      cloid,
	}

	return e.BulkOrders([]types.OrderRequest{order}, builder)
}

// BulkOrders places multiple orders in a single transaction
func (e *Exchange) BulkOrders(orders []types.OrderRequest, builder *types.BuilderInfo) (*types.OrderResponse, error) {
	// Convert orders to wire format
	orderWires := make([]types.OrderWire, len(orders))
	for i, order := range orders {
		asset, err := e.info.NameToAsset(order.Coin)
		if err != nil {
			return nil, fmt.Errorf("invalid coin for order %d: %w", i, err)
		}

		wire, err := signing.OrderRequestToOrderWire(order, asset)
		if err != nil {
			return nil, fmt.Errorf("failed to convert order %d to wire format: %w", i, err)
		}
		orderWires[i] = wire
	}

	timestamp := utils.GetTimestampMs()

	// Prepare builder info
	if builder != nil {
		builder.B = strings.ToLower(builder.B)
	}

	// Create order action
	action := signing.OrderWiresToOrderAction(orderWires, builder)

	// Sign action
	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign order: %w", err)
	}

	// Post action with typed response
	var result types.OrderResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// MarketOpen opens a position with a market order (aggressive limit order with IOC)
func (e *Exchange) MarketOpen(
	name string,
	isBuy bool,
	sz float64,
	px *float64,
	slippage float64,
	cloid *types.Cloid,
	builder *types.BuilderInfo,
) (*types.OrderResponse, error) {
	if slippage == 0 {
		slippage = constants.DefaultSlippage
	}

	// Calculate price with slippage
	price, err := e.slippagePrice(name, isBuy, slippage, px)
	if err != nil {
		return nil, err
	}

	// Market order is an aggressive limit order with IOC
	orderType := types.OrderType{
		Limit: &types.LimitOrderType{Tif: types.TifIoc},
	}

	return e.Order(name, isBuy, sz, price, orderType, false, cloid, builder)
}

// MarketClose closes a position with a market order
func (e *Exchange) MarketClose(
	name string,
	sz *float64,
	px *float64,
	slippage float64,
	cloid *types.Cloid,
	builder *types.BuilderInfo,
) (*types.OrderResponse, error) {
	if slippage == 0 {
		slippage = constants.DefaultSlippage
	}

	// Get user address
	address := e.walletAddress
	if e.accountAddress != nil {
		address = *e.accountAddress
	} else if e.vaultAddress != nil {
		address = *e.vaultAddress
	}

	// Get positions
	userState, err := e.info.UserState(address, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	// Find position for this coin
	var positionSzi float64
	found := false
	for _, assetPos := range userState.AssetPositions {
		if assetPos.Position.Coin == name {
			szi, _ := strconv.ParseFloat(assetPos.Position.Szi, 64)
			positionSzi = szi
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("no position found for %s", name)
	}

	// Calculate size and direction
	size := sz
	if size == nil {
		absSize := math.Abs(positionSzi)
		size = &absSize
	}

	isBuy := positionSzi < 0

	// Calculate price with slippage
	price, err := e.slippagePrice(name, isBuy, slippage, px)
	if err != nil {
		return nil, err
	}

	// Market order is an aggressive limit order with IOC
	orderType := types.OrderType{
		Limit: &types.LimitOrderType{Tif: types.TifIoc},
	}

	return e.Order(name, isBuy, *size, price, orderType, true, cloid, builder)
}

// Cancel cancels a single order by order ID
func (e *Exchange) Cancel(name string, oid int) (*types.CancelResponse, error) {
	return e.BulkCancel([]types.CancelRequest{{Coin: name, Oid: oid}})
}

// CancelByCloid cancels a single order by client order ID
func (e *Exchange) CancelByCloid(name string, cloid types.Cloid) (*types.CancelResponse, error) {
	return e.BulkCancelByCloid([]types.CancelByCloidRequest{{Coin: name, Cloid: cloid}})
}

// BulkCancel cancels multiple orders by order ID
func (e *Exchange) BulkCancel(cancels []types.CancelRequest) (*types.CancelResponse, error) {
	timestamp := utils.GetTimestampMs()

	// Create cancel action
	cancelWires := make([]map[string]any, len(cancels))
	for i, cancel := range cancels {
		asset, err := e.info.NameToAsset(cancel.Coin)
		if err != nil {
			return nil, fmt.Errorf("invalid coin for cancel %d: %w", i, err)
		}

		cancelWires[i] = map[string]any{
			"a": asset,
			"o": cancel.Oid,
		}
	}

	action := map[string]any{
		"type":    "cancel",
		"cancels": cancelWires,
	}

	// Sign action
	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign cancel: %w", err)
	}

	// Post action with typed response
	var result types.CancelResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// BulkCancelByCloid cancels multiple orders by client order ID
func (e *Exchange) BulkCancelByCloid(cancels []types.CancelByCloidRequest) (*types.CancelResponse, error) {
	timestamp := utils.GetTimestampMs()

	// Create cancel action
	cancelWires := make([]map[string]any, len(cancels))
	for i, cancel := range cancels {
		asset, err := e.info.NameToAsset(cancel.Coin)
		if err != nil {
			return nil, fmt.Errorf("invalid coin for cancel %d: %w", i, err)
		}

		cancelWires[i] = map[string]any{
			"asset": asset,
			"cloid": cancel.Cloid.ToRaw(),
		}
	}

	action := map[string]any{
		"type":    "cancelByCloid",
		"cancels": cancelWires,
	}

	// Sign action
	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign cancel: %w", err)
	}

	// Post action with typed response
	var result types.CancelResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateLeverage updates the leverage for a coin
func (e *Exchange) UpdateLeverage(leverage int, name string, isCross bool) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	asset, err := e.info.NameToAsset(name)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type":     "updateLeverage",
		"asset":    asset,
		"isCross":  isCross,
		"leverage": leverage,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign leverage update: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// USDTransfer transfers USD to another address
func (e *Exchange) USDTransfer(amount float64, destination string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":        "usdSend",
		"destination": destination,
		"amount":      fmt.Sprintf("%f", amount),
		"time":        timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.USDSendSignTypes,
		"HyperliquidTransaction:UsdSend",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign USD transfer: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// USDClassTransfer transfers funds between perpetual and spot wallets
func (e *Exchange) USDClassTransfer(amount float64, toPerp bool) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	amountStr := fmt.Sprintf("%f", amount)
	if e.vaultAddress != nil {
		amountStr += fmt.Sprintf(" subaccount:%s", *e.vaultAddress)
	}

	action := map[string]any{
		"type":   "usdClassTransfer",
		"amount": amountStr,
		"toPerp": toPerp,
		"nonce":  timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.USDClassTransferSignTypes,
		"HyperliquidTransaction:UsdClassTransfer",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign USD class transfer: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateSubAccount creates a new sub-account
func (e *Exchange) CreateSubAccount(name string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "createSubAccount",
		"name": name,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign sub-account creation: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetReferrer sets the referral code for the account
func (e *Exchange) SetReferrer(code string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "setReferrer",
		"code": code,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign referrer update: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ModifyOrder modifies a single order
func (e *Exchange) ModifyOrder(
	oid any, // can be int or *types.Cloid
	name string,
	isBuy bool,
	sz float64,
	limitPx float64,
	orderType types.OrderType,
	reduceOnly bool,
	cloid *types.Cloid,
) (*types.ModifyResponse, error) {
	modify := types.ModifyRequest{
		Oid: oid,
		Order: types.OrderRequest{
			Coin:       name,
			IsBuy:      isBuy,
			Sz:         sz,
			LimitPx:    limitPx,
			OrderType:  orderType,
			ReduceOnly: reduceOnly,
			Cloid:      cloid,
		},
	}
	return e.BulkModifyOrders([]types.ModifyRequest{modify})
}

// BulkModifyOrders modifies multiple orders
func (e *Exchange) BulkModifyOrders(modifies []types.ModifyRequest) (*types.ModifyResponse, error) {
	timestamp := utils.GetTimestampMs()

	modifyWires := make([]types.ModifyWire, len(modifies))
	for i, modify := range modifies {
		asset, err := e.info.NameToAsset(modify.Order.Coin)
		if err != nil {
			return nil, fmt.Errorf("invalid coin for modify %d: %w", i, err)
		}

		orderWire, err := signing.OrderRequestToOrderWire(modify.Order, asset)
		if err != nil {
			return nil, fmt.Errorf("failed to convert order %d to wire format: %w", i, err)
		}

		// Handle oid - can be int or Cloid
		var oidValue any
		if cloid, ok := modify.Oid.(*types.Cloid); ok {
			oidValue = cloid.ToRaw()
		} else {
			oidValue = modify.Oid
		}

		modifyWires[i] = types.ModifyWire{
			Oid:   oidValue,
			Order: orderWire,
		}
	}

	action := map[string]any{
		"type":     "batchModify",
		"modifies": modifyWires,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign modify: %w", err)
	}

	var result types.ModifyResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ScheduleCancel schedules a time to cancel all open orders (dead man's switch)
func (e *Exchange) ScheduleCancel(time *int64) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "scheduleCancel",
	}
	if time != nil {
		action["time"] = *time
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign schedule cancel: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateIsolatedMargin adds or removes margin from isolated position
func (e *Exchange) UpdateIsolatedMargin(amount float64, name string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	asset, err := e.info.NameToAsset(name)
	if err != nil {
		return nil, err
	}

	// Convert amount to ntli (with 6 decimals)
	ntli := int64(amount * 1e6)

	action := map[string]any{
		"type":  "updateIsolatedMargin",
		"asset": asset,
		"isBuy": true,
		"ntli":  ntli,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign isolated margin update: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotTransfer sends spot assets to another address
func (e *Exchange) SpotTransfer(amount float64, destination string, token string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":        "spotSend",
		"destination": destination,
		"token":       token,
		"amount":      fmt.Sprintf("%f", amount),
		"time":        timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.SpotSendSignTypes,
		"HyperliquidTransaction:SpotSend",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot transfer: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// WithdrawFromBridge initiates a withdrawal request
func (e *Exchange) WithdrawFromBridge(amount float64, destination string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":        "withdraw3",
		"destination": destination,
		"amount":      fmt.Sprintf("%f", amount),
		"time":        timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.Withdraw3SignTypes,
		"HyperliquidTransaction:Withdraw",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign withdrawal: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SendAsset transfers tokens between different perp DEXs, spot, users, and/or sub-accounts
func (e *Exchange) SendAsset(
	destination string,
	sourceDex string,
	destinationDex string,
	token string,
	amount float64,
) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	fromSubAccount := ""
	if e.vaultAddress != nil {
		fromSubAccount = *e.vaultAddress
	}

	action := map[string]any{
		"type":           "sendAsset",
		"destination":    destination,
		"sourceDex":      sourceDex,
		"destinationDex": destinationDex,
		"token":          token,
		"amount":         fmt.Sprintf("%f", amount),
		"fromSubAccount": fromSubAccount,
		"nonce":          timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.SendAssetSignTypes,
		"HyperliquidTransaction:SendAsset",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign send asset: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SubAccountTransfer transfers USDC between main account and sub-account
func (e *Exchange) SubAccountTransfer(subAccountUser string, isDeposit bool, usd int) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":           "subAccountTransfer",
		"subAccountUser": subAccountUser,
		"isDeposit":      isDeposit,
		"usd":            usd,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign sub-account transfer: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SubAccountSpotTransfer transfers spot assets between main account and sub-account
func (e *Exchange) SubAccountSpotTransfer(subAccountUser string, isDeposit bool, token string, amount float64) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":           "subAccountSpotTransfer",
		"subAccountUser": subAccountUser,
		"isDeposit":      isDeposit,
		"token":          token,
		"amount":         fmt.Sprintf("%f", amount),
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign sub-account spot transfer: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// VaultTransfer deposits or withdraws from a vault
func (e *Exchange) VaultTransfer(vaultAddress string, isDeposit bool, usd int) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":         "vaultTransfer",
		"vaultAddress": vaultAddress,
		"isDeposit":    isDeposit,
		"usd":          usd,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign vault transfer: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TokenDelegate delegates or undelegates stake from validator
func (e *Exchange) TokenDelegate(validator string, wei int64, isUndelegate bool) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":         "tokenDelegate",
		"validator":    validator,
		"wei":          wei,
		"isUndelegate": isUndelegate,
		"nonce":        timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.TokenDelegateSignTypes,
		"HyperliquidTransaction:TokenDelegate",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token delegate: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ApproveAgent approves an API wallet
func (e *Exchange) ApproveAgent(agentAddress string, agentName *string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":         "approveAgent",
		"agentAddress": agentAddress,
		"nonce":        timestamp,
	}
	if agentName != nil {
		action["agentName"] = *agentName
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.ApproveAgentSignTypes,
		"HyperliquidTransaction:ApproveAgent",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign approve agent: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ApproveBuilderFee approves a maximum fee rate for a builder
func (e *Exchange) ApproveBuilderFee(builder string, maxFeeRate string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":       "approveBuilderFee",
		"maxFeeRate": maxFeeRate,
		"builder":    builder,
		"nonce":      timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.ApproveBuilderFeeSignTypes,
		"HyperliquidTransaction:ApproveBuilderFee",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign approve builder fee: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Noop does nothing but marks the nonce as used (useful for canceling in-flight orders)
func (e *Exchange) Noop(nonce int64) (*types.DefaultResponse, error) {
	action := map[string]any{
		"type": "noop",
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		nonce,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign noop: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, nonce, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UserDexAbstraction enables HIP-3 DEX abstraction
func (e *Exchange) UserDexAbstraction(user string, enabled bool) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":    "userDexAbstraction",
		"user":    strings.ToLower(user),
		"enabled": enabled,
		"nonce":   timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.UserDexAbstractionSignTypes,
		"HyperliquidTransaction:UserDexAbstraction",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign user dex abstraction: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// AgentEnableDexAbstraction enables HIP-3 DEX abstraction (agent version)
func (e *Exchange) AgentEnableDexAbstraction() (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "agentEnableDexAbstraction",
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign agent enable dex abstraction: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TWAPOrder places a TWAP order
func (e *Exchange) TWAPOrder(
	name string,
	isBuy bool,
	sz float64,
	reduceOnly bool,
	minutes int,
	randomize bool,
) (*types.TWAPOrderResponse, error) {
	timestamp := utils.GetTimestampMs()

	asset, err := e.info.NameToAsset(name)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type": "twapOrder",
		"twap": map[string]any{
			"a": asset,
			"b": isBuy,
			"s": fmt.Sprintf("%f", sz),
			"r": reduceOnly,
			"m": minutes,
			"t": randomize,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign TWAP order: %w", err)
	}

	var result types.TWAPOrderResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TWAPCancel cancels a TWAP order
func (e *Exchange) TWAPCancel(name string, twapID int) (*types.TWAPCancelResponse, error) {
	timestamp := utils.GetTimestampMs()

	asset, err := e.info.NameToAsset(name)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type": "twapCancel",
		"a":    asset,
		"t":    twapID,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		e.vaultAddress,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign TWAP cancel: %w", err)
	}

	var result types.TWAPCancelResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UseBigBlocks enables or disables big blocks for EVM
func (e *Exchange) UseBigBlocks(enable bool) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":           "evmUserModify",
		"usingBigBlocks": enable,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign use big blocks: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ConvertToMultiSigUser converts an account to multi-sig
func (e *Exchange) ConvertToMultiSigUser(authorizedUsers []string, threshold int) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	// Sort authorized users
	sortedUsers := make([]string, len(authorizedUsers))
	copy(sortedUsers, authorizedUsers)
	// Simple sort
	for i := 0; i < len(sortedUsers)-1; i++ {
		for j := i + 1; j < len(sortedUsers); j++ {
			if sortedUsers[i] > sortedUsers[j] {
				sortedUsers[i], sortedUsers[j] = sortedUsers[j], sortedUsers[i]
			}
		}
	}

	signersJSON := fmt.Sprintf(`{"authorizedUsers":["%s"],"threshold":%d}`, strings.Join(sortedUsers, `","`), threshold)

	action := map[string]any{
		"type":    "convertToMultiSigUser",
		"signers": signersJSON,
		"nonce":   timestamp,
	}

	signature, err := signing.SignUserSignedAction(
		e.wallet,
		action,
		signing.ConvertToMultiSigUserSignTypes,
		"HyperliquidTransaction:ConvertToMultiSigUser",
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign convert to multi-sig: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotDeployRegisterToken registers a new spot token
func (e *Exchange) SpotDeployRegisterToken(
	tokenName string,
	szDecimals int,
	weiDecimals int,
	maxGas int,
	fullName string,
) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "spotDeploy",
		"registerToken2": map[string]any{
			"spec": map[string]any{
				"name":        tokenName,
				"szDecimals":  szDecimals,
				"weiDecimals": weiDecimals,
			},
			"maxGas":   maxGas,
			"fullName": fullName,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot deploy register token: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotDeployUserGenesis sets initial token distribution
func (e *Exchange) SpotDeployUserGenesis(
	token int,
	userAndWei []struct {
		User string
		Wei  string
	},
	existingTokenAndWei []struct {
		Token int
		Wei   string
	},
) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	userWeiList := make([][]string, len(userAndWei))
	for i, uw := range userAndWei {
		userWeiList[i] = []string{strings.ToLower(uw.User), uw.Wei}
	}

	existingList := make([][]any, len(existingTokenAndWei))
	for i, etw := range existingTokenAndWei {
		existingList[i] = []any{etw.Token, etw.Wei}
	}

	action := map[string]any{
		"type": "spotDeploy",
		"userGenesis": map[string]any{
			"token":               token,
			"userAndWei":          userWeiList,
			"existingTokenAndWei": existingList,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot deploy user genesis: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotDeployEnableFreezePrivilege enables freeze privilege for a token
func (e *Exchange) SpotDeployEnableFreezePrivilege(token int) (*types.DefaultResponse, error) {
	return e.spotDeployTokenActionInner("enableFreezePrivilege", token)
}

// SpotDeployFreezeUser freezes or unfreezes a user for a token
func (e *Exchange) SpotDeployFreezeUser(token int, user string, freeze bool) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "spotDeploy",
		"freezeUser": map[string]any{
			"token":  token,
			"user":   strings.ToLower(user),
			"freeze": freeze,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot deploy freeze user: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotDeployRevokeFreezePrivilege revokes freeze privilege for a token
func (e *Exchange) SpotDeployRevokeFreezePrivilege(token int) (*types.DefaultResponse, error) {
	return e.spotDeployTokenActionInner("revokeFreezePrivilege", token)
}

// SpotDeployEnableQuoteToken enables a token as a quote token
func (e *Exchange) SpotDeployEnableQuoteToken(token int) (*types.DefaultResponse, error) {
	return e.spotDeployTokenActionInner("enableQuoteToken", token)
}

// spotDeployTokenActionInner is a helper for spot deploy token actions
func (e *Exchange) spotDeployTokenActionInner(variant string, token int) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "spotDeploy",
		variant: map[string]any{
			"token": token,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot deploy token action: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotDeployGenesis performs genesis for a token
func (e *Exchange) SpotDeployGenesis(token int, maxSupply string, noHyperliquidity bool) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	genesis := map[string]any{
		"token":     token,
		"maxSupply": maxSupply,
	}
	if noHyperliquidity {
		genesis["noHyperliquidity"] = true
	}

	action := map[string]any{
		"type":    "spotDeploy",
		"genesis": genesis,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot deploy genesis: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotDeployRegisterSpot registers a spot market
func (e *Exchange) SpotDeployRegisterSpot(baseToken int, quoteToken int) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "spotDeploy",
		"registerSpot": map[string]any{
			"tokens": []int{baseToken, quoteToken},
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot deploy register spot: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotDeployRegisterHyperliquidity registers hyperliquidity for a spot market
func (e *Exchange) SpotDeployRegisterHyperliquidity(
	spot int,
	startPx float64,
	orderSz float64,
	nOrders int,
	nSeededLevels *int,
) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	registerHL := map[string]any{
		"spot":    spot,
		"startPx": fmt.Sprintf("%f", startPx),
		"orderSz": fmt.Sprintf("%f", orderSz),
		"nOrders": nOrders,
	}
	if nSeededLevels != nil {
		registerHL["nSeededLevels"] = *nSeededLevels
	}

	action := map[string]any{
		"type":                   "spotDeploy",
		"registerHyperliquidity": registerHL,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot deploy register hyperliquidity: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotDeploySetDeployerTradingFeeShare sets the deployer trading fee share
func (e *Exchange) SpotDeploySetDeployerTradingFeeShare(token int, share string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "spotDeploy",
		"setDeployerTradingFeeShare": map[string]any{
			"token": token,
			"share": share,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign spot deploy set deployer trading fee share: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PerpDeployRegisterAsset registers a new asset on a perp DEX
func (e *Exchange) PerpDeployRegisterAsset(
	dex string,
	maxGas *int,
	coin string,
	szDecimals int,
	oraclePx string,
	marginTableID int,
	onlyIsolated bool,
	schema *struct {
		FullName        string
		CollateralToken string
		OracleUpdater   *string
	},
) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	var schemaWire map[string]any
	if schema != nil {
		var oracleUpdater any
		if schema.OracleUpdater != nil {
			oracleUpdater = strings.ToLower(*schema.OracleUpdater)
		}
		schemaWire = map[string]any{
			"fullName":        schema.FullName,
			"collateralToken": schema.CollateralToken,
			"oracleUpdater":   oracleUpdater,
		}
	}

	action := map[string]any{
		"type": "perpDeploy",
		"registerAsset": map[string]any{
			"maxGas": maxGas,
			"assetRequest": map[string]any{
				"coin":          coin,
				"szDecimals":    szDecimals,
				"oraclePx":      oraclePx,
				"marginTableId": marginTableID,
				"onlyIsolated":  onlyIsolated,
			},
			"dex":    dex,
			"schema": schemaWire,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign perp deploy register asset: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PerpDeploySetOracle sets oracle prices for a perp DEX
func (e *Exchange) PerpDeploySetOracle(
	dex string,
	oraclePxs map[string]string,
	allMarkPxs []map[string]string,
	externalPerpPxs map[string]string,
) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	// Sort oracle prices
	oraclePxsWire := make([][]string, 0, len(oraclePxs))
	for k, v := range oraclePxs {
		oraclePxsWire = append(oraclePxsWire, []string{k, v})
	}

	// Sort mark prices
	markPxsWire := make([][][]string, len(allMarkPxs))
	for i, markPxs := range allMarkPxs {
		sorted := make([][]string, 0, len(markPxs))
		for k, v := range markPxs {
			sorted = append(sorted, []string{k, v})
		}
		markPxsWire[i] = sorted
	}

	// Sort external perp prices
	externalPerpPxsWire := make([][]string, 0, len(externalPerpPxs))
	for k, v := range externalPerpPxs {
		externalPerpPxsWire = append(externalPerpPxsWire, []string{k, v})
	}

	action := map[string]any{
		"type": "perpDeploy",
		"setOracle": map[string]any{
			"dex":             dex,
			"oraclePxs":       oraclePxsWire,
			"markPxs":         markPxsWire,
			"externalPerpPxs": externalPerpPxsWire,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign perp deploy set oracle: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CSignerUnjailSelf unjails the C-signer
func (e *Exchange) CSignerUnjailSelf() (*types.DefaultResponse, error) {
	return e.cSignerInner("unjailSelf")
}

// CSignerJailSelf jails the C-signer
func (e *Exchange) CSignerJailSelf() (*types.DefaultResponse, error) {
	return e.cSignerInner("jailSelf")
}

// cSignerInner is a helper for C-signer actions
func (e *Exchange) cSignerInner(variant string) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":  "CSignerAction",
		variant: nil,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign C-signer action: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CValidatorRegister registers a new validator
func (e *Exchange) CValidatorRegister(
	nodeIP string,
	name string,
	description string,
	delegationsDisabled bool,
	commissionBps int,
	signer string,
	unjailed bool,
	initialWei int64,
) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type": "CValidatorAction",
		"register": map[string]any{
			"profile": map[string]any{
				"node_ip":              map[string]string{"Ip": nodeIP},
				"name":                 name,
				"description":          description,
				"delegations_disabled": delegationsDisabled,
				"commission_bps":       commissionBps,
				"signer":               signer,
			},
			"unjailed":    unjailed,
			"initial_wei": initialWei,
		},
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign C-validator register: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CValidatorChangeProfile changes validator profile
func (e *Exchange) CValidatorChangeProfile(
	nodeIP *string,
	name *string,
	description *string,
	unjailed bool,
	disableDelegations *bool,
	commissionBps *int,
	signer *string,
) (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	profile := map[string]any{
		"unjailed": unjailed,
	}

	if nodeIP != nil {
		profile["node_ip"] = map[string]string{"Ip": *nodeIP}
	} else {
		profile["node_ip"] = nil
	}

	if name != nil {
		profile["name"] = *name
	} else {
		profile["name"] = nil
	}

	if description != nil {
		profile["description"] = *description
	} else {
		profile["description"] = nil
	}

	if disableDelegations != nil {
		profile["disable_delegations"] = *disableDelegations
	} else {
		profile["disable_delegations"] = nil
	}

	if commissionBps != nil {
		profile["commission_bps"] = *commissionBps
	} else {
		profile["commission_bps"] = nil
	}

	if signer != nil {
		profile["signer"] = *signer
	} else {
		profile["signer"] = nil
	}

	action := map[string]any{
		"type":          "CValidatorAction",
		"changeProfile": profile,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign C-validator change profile: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CValidatorUnregister unregisters a validator
func (e *Exchange) CValidatorUnregister() (*types.DefaultResponse, error) {
	timestamp := utils.GetTimestampMs()

	action := map[string]any{
		"type":       "CValidatorAction",
		"unregister": nil,
	}

	signature, err := signing.SignL1Action(
		e.wallet,
		action,
		nil,
		timestamp,
		e.expiresAfter,
		e.IsMainnet(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign C-validator unregister: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(action, signature, timestamp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// MultiSig executes a multi-sig action
func (e *Exchange) MultiSig(
	multiSigUser string,
	innerAction map[string]any,
	signatures []map[string]any,
	nonce int64,
	vaultAddress *string,
) (*types.DefaultResponse, error) {
	multiSigAction := map[string]any{
		"type":             "multiSig",
		"signatureChainId": "0x66eee",
		"signatures":       signatures,
		"payload": map[string]any{
			"multiSigUser": strings.ToLower(multiSigUser),
			"outerSigner":  strings.ToLower(e.walletAddress),
			"action":       innerAction,
		},
	}

	signature, err := signing.SignMultiSigAction(
		e.wallet,
		multiSigAction,
		e.IsMainnet(),
		vaultAddress,
		nonce,
		e.expiresAfter,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign multi-sig action: %w", err)
	}

	var result types.DefaultResponse
	if err := e.postAction(multiSigAction, signature, nonce, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetAddress returns the wallet address
func (e *Exchange) GetAddress() string {
	return e.walletAddress
}

// GetAccountAddress returns the account address being used (may differ from wallet if using API wallet)
func (e *Exchange) GetAccountAddress() string {
	if e.accountAddress != nil {
		return *e.accountAddress
	}
	return e.walletAddress
}

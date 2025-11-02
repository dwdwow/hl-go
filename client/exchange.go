// Package client provides the Exchange client for executing trades on Hyperliquid.
package client

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

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

// NewExchange creates a new Exchange client
// wallet: private key for signing transactions
// vaultAddress: optional vault address for vault trading
// accountAddress: optional account address (if different from wallet address, e.g., when using API wallet)
func NewExchange(
	wallet *ecdsa.PrivateKey,
	baseURL string,
	timeout time.Duration,
	vaultAddress *string,
	accountAddress *string,
) (*Exchange, error) {
	if baseURL == "" {
		baseURL = constants.MainnetAPIURL
	}

	// Create info client
	info, err := NewInfo(baseURL, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create info client: %w", err)
	}

	// Get wallet address
	pubKey := wallet.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to get public key")
	}
	walletAddress := crypto.PubkeyToAddress(*pubKeyECDSA).Hex()

	return &Exchange{
		API:            NewAPI(baseURL, timeout),
		wallet:         wallet,
		walletAddress:  walletAddress,
		vaultAddress:   vaultAddress,
		accountAddress: accountAddress,
		info:           info,
	}, nil
}

// SetExpiresAfter sets the expiration time for actions (in milliseconds)
// Set to nil to disable expiration
func (e *Exchange) SetExpiresAfter(expiresAfter *int64) {
	e.expiresAfter = expiresAfter
}

// postAction posts a signed action to the exchange
func (e *Exchange) postAction(action map[string]any, signature *types.Signature, nonce int64) (map[string]any, error) {
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

	var result map[string]any
	if err := e.Post("/exchange", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
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
) (map[string]any, error) {
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
func (e *Exchange) BulkOrders(orders []types.OrderRequest, builder *types.BuilderInfo) (map[string]any, error) {
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

	// Post action
	return e.postAction(action, signature, timestamp)
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
) (map[string]any, error) {
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
	coin string,
	sz *float64,
	px *float64,
	slippage float64,
	cloid *types.Cloid,
	builder *types.BuilderInfo,
) (map[string]any, error) {
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
		if assetPos.Position.Coin == coin {
			szi, _ := strconv.ParseFloat(assetPos.Position.Szi, 64)
			positionSzi = szi
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("no position found for %s", coin)
	}

	// Calculate size and direction
	size := sz
	if size == nil {
		absSize := math.Abs(positionSzi)
		size = &absSize
	}

	isBuy := positionSzi < 0

	// Calculate price with slippage
	price, err := e.slippagePrice(coin, isBuy, slippage, px)
	if err != nil {
		return nil, err
	}

	// Market order is an aggressive limit order with IOC
	orderType := types.OrderType{
		Limit: &types.LimitOrderType{Tif: types.TifIoc},
	}

	return e.Order(coin, isBuy, *size, price, orderType, true, cloid, builder)
}

// Cancel cancels a single order by order ID
func (e *Exchange) Cancel(name string, oid int) (map[string]any, error) {
	return e.BulkCancel([]types.CancelRequest{{Coin: name, Oid: oid}})
}

// CancelByCloid cancels a single order by client order ID
func (e *Exchange) CancelByCloid(name string, cloid types.Cloid) (map[string]any, error) {
	return e.BulkCancelByCloid([]types.CancelByCloidRequest{{Coin: name, Cloid: cloid}})
}

// BulkCancel cancels multiple orders by order ID
func (e *Exchange) BulkCancel(cancels []types.CancelRequest) (map[string]any, error) {
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

	return e.postAction(action, signature, timestamp)
}

// BulkCancelByCloid cancels multiple orders by client order ID
func (e *Exchange) BulkCancelByCloid(cancels []types.CancelByCloidRequest) (map[string]any, error) {
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

	return e.postAction(action, signature, timestamp)
}

// UpdateLeverage updates the leverage for a coin
func (e *Exchange) UpdateLeverage(leverage int, name string, isCross bool) (map[string]any, error) {
	timestamp := utils.GetTimestampMs()

	asset, err := e.info.NameToAsset(name)
	if err != nil {
		return nil, err
	}

	action := map[string]any{
		"type":    "updateLeverage",
		"asset":   asset,
		"isCross": isCross,
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

	return e.postAction(action, signature, timestamp)
}

// USDTransfer transfers USD to another address
func (e *Exchange) USDTransfer(amount float64, destination string) (map[string]any, error) {
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

	return e.postAction(action, signature, timestamp)
}

// USDClassTransfer transfers funds between perpetual and spot wallets
func (e *Exchange) USDClassTransfer(amount float64, toPerp bool) (map[string]any, error) {
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

	return e.postAction(action, signature, timestamp)
}

// CreateSubAccount creates a new sub-account
func (e *Exchange) CreateSubAccount(name string) (map[string]any, error) {
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

	return e.postAction(action, signature, timestamp)
}

// SetReferrer sets the referral code for the account
func (e *Exchange) SetReferrer(code string) (map[string]any, error) {
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

	return e.postAction(action, signature, timestamp)
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

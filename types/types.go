// Package types provides type definitions for the Hyperliquid SDK.
package types

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Side represents the order side (Ask or Bid)
type Side string

const (
	// SideAsk represents a sell order
	SideAsk Side = "A"
	// SideBid represents a buy order
	SideBid Side = "B"
)

// Tif represents Time In Force for limit orders
type Tif string

const (
	// TifAlo is Add Liquidity Only (maker-only)
	TifAlo Tif = "Alo"
	// TifIoc is Immediate Or Cancel
	TifIoc Tif = "Ioc"
	// TifGtc is Good Till Cancel
	TifGtc Tif = "Gtc"
)

// Tpsl represents Take Profit / Stop Loss type
type Tpsl string

const (
	// TpslTp is Take Profit
	TpslTp Tpsl = "tp"
	// TpslSl is Stop Loss
	TpslSl Tpsl = "sl"
)

// Grouping represents order grouping type
type Grouping string

const (
	// GroupingNa is no grouping
	GroupingNa Grouping = "na"
	// GroupingNormalTpsl is normal TP/SL grouping
	GroupingNormalTpsl Grouping = "normalTpsl"
	// GroupingPositionTpsl is position TP/SL grouping
	GroupingPositionTpsl Grouping = "positionTpsl"
)

// LimitOrderType represents a limit order configuration
type LimitOrderType struct {
	Tif Tif `json:"tif"`
}

// TriggerOrderType represents a trigger order configuration
type TriggerOrderType struct {
	TriggerPx float64 `json:"triggerPx"`
	IsMarket  bool    `json:"isMarket"`
	Tpsl      Tpsl    `json:"tpsl"`
}

// TriggerOrderTypeWire is the wire format for trigger orders
type TriggerOrderTypeWire struct {
	TriggerPx string `json:"triggerPx"`
	IsMarket  bool   `json:"isMarket"`
	Tpsl      Tpsl   `json:"tpsl"`
}

// OrderType represents the order type (limit or trigger)
type OrderType struct {
	Limit   *LimitOrderType   `json:"limit,omitempty"`
	Trigger *TriggerOrderType `json:"trigger,omitempty"`
}

// OrderTypeWire is the wire format for order types
type OrderTypeWire struct {
	Limit   *LimitOrderType       `json:"limit,omitempty"`
	Trigger *TriggerOrderTypeWire `json:"trigger,omitempty"`
}

// OrderRequest represents a request to place an order
type OrderRequest struct {
	Coin       string     `json:"coin"`
	IsBuy      bool       `json:"is_buy"`
	Sz         float64    `json:"sz"`
	LimitPx    float64    `json:"limit_px"`
	OrderType  OrderType  `json:"order_type"`
	ReduceOnly bool       `json:"reduce_only"`
	Cloid      *Cloid     `json:"cloid,omitempty"`
}

// OrderWire is the wire format for orders sent to the API
type OrderWire struct {
	Asset      int           `json:"a"`
	IsBuy      bool          `json:"b"`
	LimitPx    string        `json:"p"`
	Sz         string        `json:"s"`
	ReduceOnly bool          `json:"r"`
	OrderType  OrderTypeWire `json:"t"`
	Cloid      *string       `json:"c,omitempty"`
}

// ModifyRequest represents a request to modify an order
type ModifyRequest struct {
	Oid   any  `json:"oid"` // can be int or Cloid
	Order OrderRequest `json:"order"`
}

// ModifyWire is the wire format for modify requests
type ModifyWire struct {
	Oid   any `json:"oid"`
	Order OrderWire   `json:"order"`
}

// CancelRequest represents a request to cancel an order
type CancelRequest struct {
	Coin string `json:"coin"`
	Oid  int    `json:"oid"`
}

// CancelByCloidRequest represents a request to cancel by client order ID
type CancelByCloidRequest struct {
	Coin  string `json:"coin"`
	Cloid Cloid  `json:"cloid"`
}

// AssetInfo represents information about a trading asset
type AssetInfo struct {
	Name       string `json:"name"`
	SzDecimals int    `json:"szDecimals"`
}

// Meta represents exchange metadata
type Meta struct {
	Universe []AssetInfo `json:"universe"`
}

// SpotAssetInfo represents information about a spot trading pair
type SpotAssetInfo struct {
	Name        string `json:"name"`
	Tokens      [2]int `json:"tokens"`
	Index       int    `json:"index"`
	IsCanonical bool   `json:"isCanonical"`
}

// SpotTokenInfo represents information about a spot token
type SpotTokenInfo struct {
	Name        string  `json:"name"`
	SzDecimals  int     `json:"szDecimals"`
	WeiDecimals int     `json:"weiDecimals"`
	Index       int     `json:"index"`
	TokenID     string  `json:"tokenId"`
	IsCanonical bool    `json:"isCanonical"`
	EvmContract *string `json:"evmContract,omitempty"`
	FullName    *string `json:"fullName,omitempty"`
}

// SpotMeta represents spot exchange metadata
type SpotMeta struct {
	Universe []SpotAssetInfo `json:"universe"`
	Tokens   []SpotTokenInfo `json:"tokens"`
}

// SpotAssetCtx represents spot asset context
type SpotAssetCtx struct {
	DayNtlVlm         string  `json:"dayNtlVlm"`
	MarkPx            string  `json:"markPx"`
	MidPx             *string `json:"midPx"`
	PrevDayPx         string  `json:"prevDayPx"`
	CirculatingSupply string  `json:"circulatingSupply"`
	Coin              string  `json:"coin"`
}

// SpotMetaAndAssetCtxs represents spot metadata with asset contexts
type SpotMetaAndAssetCtxs struct {
	Meta       SpotMeta       `json:"meta"`
	AssetCtxs  []SpotAssetCtx `json:"assetCtxs"`
}

// BuilderInfo represents builder fee information
type BuilderInfo struct {
	B string `json:"b"` // builder address
	F int    `json:"f"` // fee in tenths of basis points
}

// Signature represents an ECDSA signature
type Signature struct {
	R string `json:"r"`
	S string `json:"s"`
	V int    `json:"v"`
}

// Leverage represents position leverage
type Leverage struct {
	Type   string  `json:"type"` // "cross" or "isolated"
	Value  int     `json:"value"`
	RawUsd *string `json:"rawUsd,omitempty"` // only for isolated
}

// Position represents a trading position
type Position struct {
	Coin           string  `json:"coin"`
	EntryPx        *string `json:"entryPx"`
	Leverage       Leverage `json:"leverage"`
	LiquidationPx  *string `json:"liquidationPx"`
	MarginUsed     string  `json:"marginUsed"`
	PositionValue  string  `json:"positionValue"`
	ReturnOnEquity string  `json:"returnOnEquity"`
	Szi            string  `json:"szi"`
	UnrealizedPnl  string  `json:"unrealizedPnl"`
}

// AssetPosition represents an asset position wrapper
type AssetPosition struct {
	Position Position `json:"position"`
	Type     string   `json:"type"`
}

// MarginSummary represents margin summary information
type MarginSummary struct {
	AccountValue    string `json:"accountValue"`
	TotalMarginUsed string `json:"totalMarginUsed"`
	TotalNtlPos     string `json:"totalNtlPos"`
	TotalRawUsd     string `json:"totalRawUsd"`
}

// UserState represents user trading state
type UserState struct {
	AssetPositions     []AssetPosition `json:"assetPositions"`
	CrossMarginSummary MarginSummary   `json:"crossMarginSummary"`
	MarginSummary      MarginSummary   `json:"marginSummary"`
	Withdrawable       string          `json:"withdrawable"`
}

// OpenOrder represents an open order
type OpenOrder struct {
	Coin      string `json:"coin"`
	LimitPx   string `json:"limitPx"`
	Oid       int    `json:"oid"`
	Side      Side   `json:"side"`
	Sz        string `json:"sz"`
	Timestamp int64  `json:"timestamp"`
}

// Fill represents a trade fill
type Fill struct {
	Coin          string `json:"coin"`
	Px            string `json:"px"`
	Sz            string `json:"sz"`
	Side          Side   `json:"side"`
	Time          int64  `json:"time"`
	StartPosition string `json:"startPosition"`
	Dir           string `json:"dir"`
	ClosedPnl     string `json:"closedPnl"`
	Hash          string `json:"hash"`
	Oid           int    `json:"oid"`
	Crossed       bool   `json:"crossed"`
	Fee           string `json:"fee"`
	Tid           int    `json:"tid"`
	FeeToken      string `json:"feeToken"`
}

// L2Level represents a level in the L2 order book
type L2Level struct {
	Px string `json:"px"`
	Sz string `json:"sz"`
	N  int    `json:"n"`
}

// L2BookData represents L2 order book data
type L2BookData struct {
	Coin   string      `json:"coin"`
	Levels [2][]L2Level `json:"levels"` // [bids, asks]
	Time   int64       `json:"time"`
}

// SubscriptionType represents the type of WebSocket subscription
type SubscriptionType string

const (
	// SubscriptionAllMids subscribes to all mid prices
	SubscriptionAllMids SubscriptionType = "allMids"

	// SubscriptionL2Book subscribes to L2 order book for a specific coin
	SubscriptionL2Book SubscriptionType = "l2Book"

	// SubscriptionTrades subscribes to trades for a specific coin
	SubscriptionTrades SubscriptionType = "trades"

	// SubscriptionBBO subscribes to best bid/offer for a specific coin
	SubscriptionBBO SubscriptionType = "bbo"

	// SubscriptionCandle subscribes to candlestick data for a specific coin
	SubscriptionCandle SubscriptionType = "candle"

	// SubscriptionActiveAssetCtx subscribes to asset context (funding, open interest, etc.)
	SubscriptionActiveAssetCtx SubscriptionType = "activeAssetCtx"

	// SubscriptionActiveAssetData subscribes to active asset data for a user and coin
	SubscriptionActiveAssetData SubscriptionType = "activeAssetData"

	// SubscriptionUserEvents subscribes to user trading events
	SubscriptionUserEvents SubscriptionType = "userEvents"

	// SubscriptionUserFills subscribes to user trade fills
	SubscriptionUserFills SubscriptionType = "userFills"

	// SubscriptionOrderUpdates subscribes to order status updates
	SubscriptionOrderUpdates SubscriptionType = "orderUpdates"

	// SubscriptionUserFundings subscribes to user funding payments
	SubscriptionUserFundings SubscriptionType = "userFundings"

	// SubscriptionUserNonFundingLedgerUpdates subscribes to non-funding ledger updates
	SubscriptionUserNonFundingLedgerUpdates SubscriptionType = "userNonFundingLedgerUpdates"

	// SubscriptionWebData2 subscribes to web data for a user
	SubscriptionWebData2 SubscriptionType = "webData2"
)

// Subscription represents a WebSocket subscription
type Subscription struct {
	Type     SubscriptionType `json:"type"`
	Coin     *string          `json:"coin,omitempty"`
	User     *string          `json:"user,omitempty"`
	Interval *string          `json:"interval,omitempty"`
}

// Cloid represents a client order ID (16 bytes hex string)
type Cloid struct {
	raw string
}

// NewCloidFromInt creates a Cloid from an integer
func NewCloidFromInt(value int64) *Cloid {
	return &Cloid{raw: fmt.Sprintf("0x%032x", value)}
}

// NewCloidFromString creates a Cloid from a hex string
func NewCloidFromString(value string) (*Cloid, error) {
	if !strings.HasPrefix(value, "0x") {
		return nil, fmt.Errorf("cloid must start with 0x")
	}
	if len(value[2:]) != 32 {
		return nil, fmt.Errorf("cloid must be 16 bytes (32 hex chars), got %d", len(value[2:]))
	}
	// Validate hex
	if _, err := hex.DecodeString(value[2:]); err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}
	return &Cloid{raw: value}, nil
}

// ToRaw returns the raw hex string representation
func (c *Cloid) ToRaw() string {
	return c.raw
}

// String returns the string representation
func (c *Cloid) String() string {
	return c.raw
}

// MarshalJSON implements json.Marshaler
func (c *Cloid) MarshalJSON() ([]byte, error) {
	return []byte(`"` + c.raw + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (c *Cloid) UnmarshalJSON(data []byte) error {
	s := string(data)
	if len(s) < 2 {
		return fmt.Errorf("invalid cloid JSON")
	}
	// Remove quotes
	s = s[1 : len(s)-1]
	cloid, err := NewCloidFromString(s)
	if err != nil {
		return err
	}
	c.raw = cloid.raw
	return nil
}

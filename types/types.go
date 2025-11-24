// Package types provides comprehensive type definitions for the Hyperliquid SDK.
//
// This package contains all the data structures used for interacting with the
// Hyperliquid API, including:
//   - Order types and configurations (limit, trigger, TP/SL)
//   - Request and response structures for exchange operations
//   - Market data and user state structures
//   - WebSocket subscription types
//   - Signature and authentication types
//
// The types are designed to be type-safe and user-friendly, with custom
// string types for enums to provide better IDE support and compile-time validation.
package types

import (
	"encoding/hex"
	"encoding/json"
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
	Tif Tif `json:"tif" msgpack:"tif"`
}

// TriggerOrderType represents a trigger order configuration
type TriggerOrderType struct {
	TriggerPx float64 `json:"triggerPx"`
	IsMarket  bool    `json:"isMarket"`
	Tpsl      Tpsl    `json:"tpsl"`
}

// TriggerOrderTypeWire is the wire format for trigger orders
type TriggerOrderTypeWire struct {
	TriggerPx string `json:"triggerPx" msgpack:"triggerPx"`
	IsMarket  bool   `json:"isMarket" msgpack:"isMarket"`
	Tpsl      Tpsl   `json:"tpsl" msgpack:"tpsl"`
}

// OrderType represents the order type (limit or trigger)
type OrderType struct {
	Limit   *LimitOrderType   `json:"limit,omitempty"`
	Trigger *TriggerOrderType `json:"trigger,omitempty"`
}

// OrderTypeWire is the wire format for order types
type OrderTypeWire struct {
	Limit   *LimitOrderType       `json:"limit,omitempty" msgpack:"limit,omitempty"`
	Trigger *TriggerOrderTypeWire `json:"trigger,omitempty" msgpack:"trigger,omitempty"`
}

// OrderRequest represents a request to place an order
type OrderRequest struct {
	Coin       string    `json:"coin"`
	IsBuy      bool      `json:"is_buy"`
	Sz         float64   `json:"sz"`
	LimitPx    float64   `json:"limit_px"`
	OrderType  OrderType `json:"order_type"`
	ReduceOnly bool      `json:"reduce_only"`
	Cloid      *Cloid    `json:"cloid,omitempty"`
}

// OrderWire is the wire format for orders sent to the API
type OrderWire struct {
	Asset      int           `json:"a" msgpack:"a"`
	IsBuy      bool          `json:"b" msgpack:"b"`
	LimitPx    string        `json:"p" msgpack:"p"`
	Sz         string        `json:"s" msgpack:"s"`
	ReduceOnly bool          `json:"r" msgpack:"r"`
	OrderType  OrderTypeWire `json:"t" msgpack:"t"`
	Cloid      *string       `json:"c,omitempty" msgpack:"c,omitempty"`
}

// ModifyRequest represents a request to modify an order
type ModifyRequest struct {
	Oid   any          `json:"oid"` // can be int or Cloid
	Order OrderRequest `json:"order"`
}

// ModifyWire is the wire format for modify requests
type ModifyWire struct {
	Oid   any       `json:"oid" msgpack:"oid"`
	Order OrderWire `json:"order" msgpack:"order"`
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
	Name        string `json:"name"`
	SzDecimals  int    `json:"szDecimals"`
	MaxLeverage int    `json:"maxLeverage"`
}

// Meta represents exchange metadata
type Meta struct {
	Universe     []AssetInfo       `json:"universe"`
	MarginTables []MarginTablePair `json:"marginTables,omitempty"`
}

// MarginTier represents a single tier in a margin table
type MarginTier struct {
	LowerBound  string `json:"lowerBound"`
	MaxLeverage int    `json:"maxLeverage"`
}

// MarginTableData is the object carried alongside the margin table index
type MarginTableData struct {
	Description string       `json:"description"`
	MarginTiers []MarginTier `json:"marginTiers"`
}

// MarginTablePair represents an entry in the marginTables array which is
// encoded as [index, { ...data... }] in the wire format.
type MarginTablePair struct {
	Index int             `json:"index"`
	Data  MarginTableData `json:"data"`
}

// UnmarshalJSON supports both the array form [index,obj] and an object form
func (m *MarginTablePair) UnmarshalJSON(b []byte) error {
	// try array form first
	var arr []json.RawMessage
	if err := json.Unmarshal(b, &arr); err == nil && len(arr) == 2 {
		if err := json.Unmarshal(arr[0], &m.Index); err != nil {
			return err
		}
		if err := json.Unmarshal(arr[1], &m.Data); err != nil {
			return err
		}
		return nil
	}

	// fallback to object form
	var obj struct {
		Index int             `json:"index"`
		Data  MarginTableData `json:"data"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}
	m.Index = obj.Index
	m.Data = obj.Data
	return nil
}

// SpotAssetInfo represents information about a spot trading pair
type SpotAssetInfo struct {
	Name        string `json:"name"`
	Tokens      [2]int `json:"tokens"`
	Index       int    `json:"index"`
	IsCanonical bool   `json:"isCanonical"`
}

// EvmContract represents information about an EVM (Ethereum Virtual Machine) contract.
type EvmContract struct {
	Address             string `json:"address"`
	EvmExtraWeiDecimals int    `json:"evm_extra_wei_decimals"`
}

// SpotTokenInfo represents information about a spot token
type SpotTokenInfo struct {
	Name        string       `json:"name"`
	SzDecimals  int          `json:"szDecimals"`
	WeiDecimals int          `json:"weiDecimals"`
	Index       int          `json:"index"`
	TokenID     string       `json:"tokenId"`
	IsCanonical bool         `json:"isCanonical"`
	EvmContract *EvmContract `json:"evmContract,omitempty"`
	FullName    *string      `json:"fullName,omitempty"`
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
	Meta      SpotMeta       `json:"meta"`
	AssetCtxs []SpotAssetCtx `json:"assetCtxs"`
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
	Coin           string   `json:"coin"`
	EntryPx        *string  `json:"entryPx"`
	Leverage       Leverage `json:"leverage"`
	LiquidationPx  *string  `json:"liquidationPx"`
	MarginUsed     string   `json:"marginUsed"`
	PositionValue  string   `json:"positionValue"`
	ReturnOnEquity string   `json:"returnOnEquity"`
	Szi            string   `json:"szi"`
	UnrealizedPnl  string   `json:"unrealizedPnl"`
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
	Coin   string       `json:"coin"`
	Levels [2][]L2Level `json:"levels"` // [bids, asks]
	Time   int64        `json:"time"`
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

// Exchange Response Types

// ApiResponse is the common response structure for all exchange API calls
type ApiResponse struct {
	Status   string          `json:"status"` // "ok" or "err"
	Response json.RawMessage `json:"response,omitempty"`
}

// DecodeResponse decodes the response field based on status
// If status is not "ok", it decodes response as error string
// If status is "ok", it decodes response into the provided struct
func (r *ApiResponse) DecodeResponse(v interface{}) error {
	if r.Status != "ok" {
		// Response is an error string
		var errMsg string
		if err := json.Unmarshal(r.Response, &errMsg); err != nil {
			return fmt.Errorf("failed to decode error message: %w", err)
		}
		return fmt.Errorf("%s", errMsg)
	}

	// Response is the actual data, decode into v
	if err := json.Unmarshal(r.Response, v); err != nil {
		return fmt.Errorf("failed to decode response data: %w", err)
	}
	return nil
}

// GetError returns the error message if status is not "ok"
func (r *ApiResponse) GetError() (string, error) {
	if r.Status == "ok" {
		return "", nil
	}
	var errMsg string
	if err := json.Unmarshal(r.Response, &errMsg); err != nil {
		return "", fmt.Errorf("failed to decode error message: %w", err)
	}
	return errMsg, nil
}

// OrderResponse represents the response from order placement
type OrderResponse struct {
	Type string        `json:"type"` // "order"
	Data OrderDataBody `json:"data"`
}

// OrderDataBody represents the actual order data
type OrderDataBody struct {
	Statuses []OrderStatus `json:"statuses"`
}

// OrderStatusType represents the canonical status string for an order.
type OrderStatusType string

const (
	OrderStatusOpen                                      OrderStatusType = "open"                                      // Placed successfully
	OrderStatusFilled                                    OrderStatusType = "filled"                                    // Filled
	OrderStatusCanceled                                  OrderStatusType = "canceled"                                  // Canceled by user
	OrderStatusTriggered                                 OrderStatusType = "triggered"                                 // Trigger order triggered
	OrderStatusRejected                                  OrderStatusType = "rejected"                                  // Rejected at time of placement
	OrderStatusMarginCanceled                            OrderStatusType = "marginCanceled"                            // Canceled because insufficient margin to fill
	OrderStatusVaultWithdrawalCanceled                   OrderStatusType = "vaultWithdrawalCanceled"                   // Vaults only. Canceled due to a user's withdrawal from vault
	OrderStatusOpenInterestCapCanceled                   OrderStatusType = "openInterestCapCanceled"                   // Canceled due to order being too aggressive when open interest was at cap
	OrderStatusSelfTradeCanceled                         OrderStatusType = "selfTradeCanceled"                         // Canceled due to self-trade prevention
	OrderStatusReduceOnlyCanceled                        OrderStatusType = "reduceOnlyCanceled"                        // Canceled reduced-only order that does not reduce position
	OrderStatusSiblingFilledCanceled                     OrderStatusType = "siblingFilledCanceled"                     // TP/SL only. Canceled due to sibling ordering being filled
	OrderStatusDelistedCanceled                          OrderStatusType = "delistedCanceled"                          // Canceled due to asset delisting
	OrderStatusLiquidatedCanceled                        OrderStatusType = "liquidatedCanceled"                        // Canceled due to liquidation
	OrderStatusScheduledCancel                           OrderStatusType = "scheduledCancel"                           // API only. Canceled due to exceeding scheduled cancel deadline (dead man's switch)
	OrderStatusTickRejected                              OrderStatusType = "tickRejected"                              // Rejected due to invalid tick price
	OrderStatusMinTradeNtlRejected                       OrderStatusType = "minTradeNtlRejected"                       // Rejected due to order notional below minimum
	OrderStatusPerpMarginRejected                        OrderStatusType = "perpMarginRejected"                        // Rejected due to insufficient margin
	OrderStatusReduceOnlyRejected                        OrderStatusType = "reduceOnlyRejected"                        // Rejected due to reduce only
	OrderStatusBadAloPxRejected                          OrderStatusType = "badAloPxRejected"                          // Rejected due to post-only immediate match
	OrderStatusIocCancelRejected                         OrderStatusType = "iocCancelRejected"                         // Rejected due to IOC not able to match
	OrderStatusBadTriggerPxRejected                      OrderStatusType = "badTriggerPxRejected"                      // Rejected due to invalid TP/SL price
	OrderStatusMarketOrderNoLiquidityRejected            OrderStatusType = "marketOrderNoLiquidityRejected"            // Rejected due to lack of liquidity for market order
	OrderStatusPositionIncreaseAtOpenInterestCapRejected OrderStatusType = "positionIncreaseAtOpenInterestCapRejected" // Rejected due to open interest cap
	OrderStatusPositionFlipAtOpenInterestCapRejected     OrderStatusType = "positionFlipAtOpenInterestCapRejected"     // Rejected due to open interest cap
	OrderStatusTooAggressiveAtOpenInterestCapRejected    OrderStatusType = "tooAggressiveAtOpenInterestCapRejected"    // Rejected due to price too aggressive at open interest cap
	OrderStatusOpenInterestIncreaseRejected              OrderStatusType = "openInterestIncreaseRejected"              // Rejected due to open interest cap
	OrderStatusInsufficientSpotBalanceRejected           OrderStatusType = "insufficientSpotBalanceRejected"           // Rejected due to insufficient spot balance
	OrderStatusOracleRejected                            OrderStatusType = "oracleRejected"                            // Rejected due to price too far from oracle
	OrderStatusPerpMaxPositionRejected                   OrderStatusType = "perpMaxPositionRejected"                   // Rejected due to exceeding margin tier limit at current leverage
)

// orderStatusDescriptions maps statuses to human-friendly explanations.
var orderStatusDescriptions = map[OrderStatusType]string{
	OrderStatusOpen:                                      "Placed successfully",
	OrderStatusFilled:                                    "Filled",
	OrderStatusCanceled:                                  "Canceled by user",
	OrderStatusTriggered:                                 "Trigger order triggered",
	OrderStatusRejected:                                  "Rejected at time of placement",
	OrderStatusMarginCanceled:                            "Canceled because insufficient margin to fill",
	OrderStatusVaultWithdrawalCanceled:                   "Vaults only. Canceled due to a user's withdrawal from vault",
	OrderStatusOpenInterestCapCanceled:                   "Canceled due to order being too aggressive when open interest was at cap",
	OrderStatusSelfTradeCanceled:                         "Canceled due to self-trade prevention",
	OrderStatusReduceOnlyCanceled:                        "Canceled reduced-only order that does not reduce position",
	OrderStatusSiblingFilledCanceled:                     "TP/SL only. Canceled due to sibling ordering being filled",
	OrderStatusDelistedCanceled:                          "Canceled due to asset delisting",
	OrderStatusLiquidatedCanceled:                        "Canceled due to liquidation",
	OrderStatusScheduledCancel:                           "API only. Canceled due to exceeding scheduled cancel deadline (dead man's switch)",
	OrderStatusTickRejected:                              "Rejected due to invalid tick price",
	OrderStatusMinTradeNtlRejected:                       "Rejected due to order notional below minimum",
	OrderStatusPerpMarginRejected:                        "Rejected due to insufficient margin",
	OrderStatusReduceOnlyRejected:                        "Rejected due to reduce only",
	OrderStatusBadAloPxRejected:                          "Rejected due to post-only immediate match",
	OrderStatusIocCancelRejected:                         "Rejected due to IOC not able to match",
	OrderStatusBadTriggerPxRejected:                      "Rejected due to invalid TP/SL price",
	OrderStatusMarketOrderNoLiquidityRejected:            "Rejected due to lack of liquidity for market order",
	OrderStatusPositionIncreaseAtOpenInterestCapRejected: "Rejected due to open interest cap",
	OrderStatusPositionFlipAtOpenInterestCapRejected:     "Rejected due to open interest cap",
	OrderStatusTooAggressiveAtOpenInterestCapRejected:    "Rejected due to price too aggressive at open interest cap",
	OrderStatusOpenInterestIncreaseRejected:              "Rejected due to open interest cap",
	OrderStatusInsufficientSpotBalanceRejected:           "Rejected due to insufficient spot balance",
	OrderStatusOracleRejected:                            "Rejected due to price too far from oracle",
	OrderStatusPerpMaxPositionRejected:                   "Rejected due to exceeding margin tier limit at current leverage",
}

// String returns the raw status string.
func (s OrderStatusType) String() string { return string(s) }

// Description returns a human-friendly explanation for the status if available.
func (s OrderStatusType) Description() string {
	if d, ok := orderStatusDescriptions[s]; ok {
		return d
	}
	return ""
}

// OrderStatus represents the status of a single order
type OrderStatus struct {
	Resting *RestingOrder `json:"resting,omitempty"`
	Filled  *FilledOrder  `json:"filled,omitempty"`
	Error   string        `json:"error,omitempty"`
}

// RestingOrder represents an order that is resting on the book
type RestingOrder struct {
	Oid int `json:"oid"` // Order ID
}

// FilledOrder represents a filled order
type FilledOrder struct {
	TotalSz string `json:"totalSz,omitempty"`
	AvgPx   string `json:"avgPx,omitempty"`
	Oid     int    `json:"oid,omitempty"`
}

// CancelResponse represents the response from order cancellation
type CancelResponse struct {
	Type string         `json:"type"` // "cancel" or "cancelByCloid"
	Data CancelDataBody `json:"data"`
}

// CancelDataBody represents the actual cancel data
type CancelDataBody struct {
	Statuses []string `json:"statuses"` // e.g. ["success"]
}

// ModifyResponse represents the response from order modification
type ModifyResponse struct {
	Type string         `json:"type"` // "modify" or "batchModify"
	Data ModifyDataBody `json:"data"`
}

// ModifyDataBody represents the actual modify data
type ModifyDataBody struct {
	Statuses []OrderStatus `json:"statuses"`
}

// TWAPOrderResponse represents the response from TWAP order placement
type TWAPOrderResponse struct {
	Type string            `json:"type"` // "twapOrder"
	Data TWAPOrderDataBody `json:"data"`
}

// TWAPOrderDataBody represents the actual TWAP order data
type TWAPOrderDataBody struct {
	Status TWAPOrderStatus `json:"status"`
}

// TWAPOrderStatus represents the status of a TWAP order
type TWAPOrderStatus struct {
	Running *TWAPRunning `json:"running,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// TWAPRunning represents a running TWAP order
type TWAPRunning struct {
	TwapID int `json:"twapId"`
}

// TWAPCancelResponse represents the response from TWAP order cancellation
type TWAPCancelResponse struct {
	Type string             `json:"type"` // "twapCancel"
	Data TWAPCancelDataBody `json:"data"`
}

// TWAPCancelDataBody represents the actual TWAP cancel data
type TWAPCancelDataBody struct {
	Status string `json:"status"` // "success"
}

// DefaultResponse represents the default response for most operations
type DefaultResponse struct {
	Type string `json:"type"` // "default"
}

// FrontendOpenOrder represents open order with frontend-specific fields
type FrontendOpenOrder struct {
	Coin             string `json:"coin"`
	IsPositionTpsl   bool   `json:"isPositionTpsl"`
	IsTrigger        bool   `json:"isTrigger"`
	LimitPx          string `json:"limitPx"`
	Oid              int    `json:"oid"`
	OrderType        string `json:"orderType"`
	OrigSz           string `json:"origSz"`
	ReduceOnly       bool   `json:"reduceOnly"`
	Side             Side   `json:"side"`
	Sz               string `json:"sz"`
	Timestamp        int64  `json:"timestamp"`
	TriggerCondition string `json:"triggerCondition"`
	TriggerPx        string `json:"triggerPx"`
}

// SpotBalance represents a balance entry in spot state
type SpotBalance struct {
	Coin     string `json:"coin"`
	Token    int    `json:"token"`
	Total    string `json:"total"`
	Hold     string `json:"hold"`
	EntryNtl string `json:"entryNtl"`
}

// SpotUserState represents spot clearinghouse state for a user
type SpotUserState struct {
	Balances []SpotBalance `json:"balances"`
}

// Candle represents a single candle entry in candle snapshot
type Candle struct {
	T  int64  `json:"T"` // end time in ms
	C  string `json:"c"`
	H  string `json:"h"`
	I  string `json:"i"` // interval
	L  string `json:"l"`
	N  int    `json:"n"`
	O  string `json:"o"`
	S  string `json:"s"` // coin
	T0 int64  `json:"t"` // start time
	V  string `json:"v"`
}

// FundingRecord is a minimal funding history record
type FundingRecord struct {
	Time int64  `json:"time"`
	Px   string `json:"px"`
	Rate string `json:"rate"`
}

// UserFees represents the user fees summary and schedule
type UserFees struct {
	DailyUserVlm                []RawJSON   `json:"dailyUserVlm"`
	FeeSchedule                 RawJSON     `json:"feeSchedule"`
	UserCrossRate               string      `json:"userCrossRate"`
	UserAddRate                 string      `json:"userAddRate"`
	UserSpotCrossRate           string      `json:"userSpotCrossRate"`
	UserSpotAddRate             string      `json:"userSpotAddRate"`
	ActiveReferralDiscount      string      `json:"activeReferralDiscount"`
	Trial                       interface{} `json:"trial"`
	FeeTrialReward              string      `json:"feeTrialReward"`
	NextTrialAvailableTimestamp *int64      `json:"nextTrialAvailableTimestamp"`
	StakingLink                 RawJSON     `json:"stakingLink"`
	ActiveStakingDiscount       RawJSON     `json:"activeStakingDiscount"`
}

// Delegation represents a staking delegation entry
type Delegation struct {
	Validator            string `json:"validator"`
	Amount               string `json:"amount"`
	LockedUntilTimestamp int64  `json:"lockedUntilTimestamp"`
}

// DelegatorSummary represents staking summary for a user
type DelegatorSummary struct {
	Delegated              string `json:"delegated"`
	Undelegated            string `json:"undelegated"`
	TotalPendingWithdrawal string `json:"totalPendingWithdrawal"`
	NPendingWithdrawals    int    `json:"nPendingWithdrawals"`
}

// DelegatorHistoryEntry represents a single history entry for delegations
type DelegatorHistoryEntry struct {
	Time  int64   `json:"time"`
	Hash  string  `json:"hash"`
	Delta RawJSON `json:"delta"`
}

// VaultFollower represents a follower in a vault
type VaultFollower struct {
	User           string `json:"user"`
	VaultEquity    string `json:"vaultEquity"`
	Pnl            string `json:"pnl"`
	AllTimePnl     string `json:"allTimePnl"`
	DaysFollowing  int    `json:"daysFollowing"`
	VaultEntryTime int64  `json:"vaultEntryTime"`
	LockupUntil    int64  `json:"lockupUntil"`
}

// VaultDetails represents detailed information about a vault
type VaultDetails struct {
	Name                  string          `json:"name"`
	VaultAddress          string          `json:"vaultAddress"`
	Leader                string          `json:"leader"`
	Description           string          `json:"description"`
	Portfolio             []RawJSON       `json:"portfolio"`
	Apr                   float64         `json:"apr"`
	FollowerState         interface{}     `json:"followerState"`
	LeaderFraction        float64         `json:"leaderFraction"`
	LeaderCommission      int             `json:"leaderCommission"`
	Followers             []VaultFollower `json:"followers"`
	MaxDistributable      float64         `json:"maxDistributable"`
	MaxWithdrawable       float64         `json:"maxWithdrawable"`
	IsClosed              bool            `json:"isClosed"`
	Relationship          RawJSON         `json:"relationship"`
	AllowDeposits         bool            `json:"allowDeposits"`
	AlwaysCloseOnWithdraw bool            `json:"alwaysCloseOnWithdraw"`
}

// VaultEquity represents a user's equity in a vault
type VaultEquity struct {
	VaultAddress string `json:"vaultAddress"`
	Equity       string `json:"equity"`
}

// ReferralResponse is a minimal representation for referral query
type ReferralResponse struct {
	ReferredBy       RawJSON   `json:"referredBy"`
	CumVlm           string    `json:"cumVlm"`
	UnclaimedRewards string    `json:"unclaimedRewards"`
	ClaimedRewards   string    `json:"claimedRewards"`
	BuilderRewards   string    `json:"builderRewards"`
	TokenToState     []RawJSON `json:"tokenToState"`
	ReferrerState    RawJSON   `json:"referrerState"`
	RewardHistory    []RawJSON `json:"rewardHistory"`
}

// UserRateLimitResponse represents the user rate limit response
type UserRateLimitResponse struct {
	CumVlm           string `json:"cumVlm"`
	NRequestsUsed    int    `json:"nRequestsUsed"`
	NRequestsCap     int    `json:"nRequestsCap"`
	NRequestsSurplus int    `json:"nRequestsSurplus"`
}

// RawJSON is an alias for json.RawMessage to represent arbitrary JSON blobs
type RawJSON = json.RawMessage

// MetaAndAssetCtxs represents perp meta with arbitrary asset contexts
// PerpAssetCtx represents asset-specific runtime context for perp markets
type PerpAssetCtx struct {
	DayNtlVlm    string   `json:"dayNtlVlm"`
	Funding      string   `json:"funding"`
	ImpactPxs    []string `json:"impactPxs"`
	MarkPx       string   `json:"markPx"`
	MidPx        string   `json:"midPx"`
	OpenInterest string   `json:"openInterest"`
	OraclePx     string   `json:"oraclePx"`
	Premium      string   `json:"premium"`
	PrevDayPx    string   `json:"prevDayPx"`
}

type MetaAndAssetCtxs struct {
	Meta      Meta           `json:"meta"`
	AssetCtxs []PerpAssetCtx `json:"assetCtxs"`
}

// PerpDex represents a perpetual DEX entry (shape can vary)
// PerpDex represents a perpetual DEX entry with known fields where available.
type PerpDex struct {
	Name                  string     `json:"name"`
	FullName              *string    `json:"fullName,omitempty"`
	Deployer              *string    `json:"deployer,omitempty"`
	OracleUpdater         *string    `json:"oracleUpdater,omitempty"`
	FeeRecipient          *string    `json:"feeRecipient,omitempty"`
	AssetToStreamingOiCap [][]string `json:"assetToStreamingOiCap,omitempty"`
}

// DelegatorReward represents a staking reward entry
type DelegatorReward struct {
	Time        int64  `json:"time"`
	Source      string `json:"source"`
	TotalAmount string `json:"totalAmount"`
}

// PerpDeployAuctionStatus describes the auction status for perp deploys.
type PerpDeployAuctionStatus struct {
	StartTimeSeconds int64   `json:"startTimeSeconds"`
	DurationSeconds  int64   `json:"durationSeconds"`
	StartGas         string  `json:"startGas"`
	CurrentGas       string  `json:"currentGas"`
	EndGas           *string `json:"endGas"`
}

// AuctionStatus is a generic auction status structure reused for perp/spot auctions
type AuctionStatus = PerpDeployAuctionStatus

// SpotTokenSpec describes a spot token specification used in spot deploy state
type SpotTokenSpec struct {
	Name        string `json:"name"`
	SzDecimals  int    `json:"szDecimals"`
	WeiDecimals int    `json:"weiDecimals"`
}

// SpotDeployStateEntry represents a single entry in the spot deploy "states" array
type SpotDeployStateEntry struct {
	Token                        int           `json:"token"`
	Spec                         SpotTokenSpec `json:"spec"`
	FullName                     *string       `json:"fullName,omitempty"`
	Spots                        []int         `json:"spots"`
	MaxSupply                    string        `json:"maxSupply"`
	HyperliquidityGenesisBalance string        `json:"hyperliquidityGenesisBalance"`
	TotalGenesisBalanceWei       string        `json:"totalGenesisBalanceWei"`
	UserGenesisBalances          [][]string    `json:"userGenesisBalances"`
	ExistingTokenGenesisBalances []interface{} `json:"existingTokenGenesisBalances"`
}

// SpotDeployState is the response for `spotDeployState` which includes states and a gas auction
type SpotDeployState struct {
	States     []SpotDeployStateEntry `json:"states"`
	GasAuction AuctionStatus          `json:"gasAuction"`
}

// TokenDetails is the response for `tokenDetails`
type TokenDetails struct {
	Name                       string    `json:"name"`
	MaxSupply                  string    `json:"maxSupply"`
	TotalSupply                string    `json:"totalSupply"`
	CirculatingSupply          string    `json:"circulatingSupply"`
	SzDecimals                 int       `json:"szDecimals"`
	WeiDecimals                int       `json:"weiDecimals"`
	MidPx                      *string   `json:"midPx,omitempty"`
	MarkPx                     *string   `json:"markPx,omitempty"`
	PrevDayPx                  *string   `json:"prevDayPx,omitempty"`
	Genesis                    RawJSON   `json:"genesis"`
	Deployer                   *string   `json:"deployer,omitempty"`
	DeployGas                  *string   `json:"deployGas,omitempty"`
	DeployTime                 *string   `json:"deployTime,omitempty"`
	SeededUsdc                 *string   `json:"seededUsdc,omitempty"`
	NonCirculatingUserBalances []RawJSON `json:"nonCirculatingUserBalances"`
	FutureEmissions            *string   `json:"futureEmissions,omitempty"`
}

// Predicted funding structures and custom unmarshalling because the
// wire format is an array-of-arrays: [ ["COIN", [["Venue", {..}], ... ] ], ... ]
type PredictedFundingInfo struct {
	FundingRate     string `json:"fundingRate"`
	NextFundingTime int64  `json:"nextFundingTime"`
}

type PredictedFundingVenue struct {
	Venue string               `json:"venue"`
	Info  PredictedFundingInfo `json:"info"`
}

type PredictedFundingEntry struct {
	Coin   string                  `json:"coin"`
	Venues []PredictedFundingVenue `json:"venues"`
}

type PredictedFundings []PredictedFundingEntry

// UnmarshalJSON supports the wire-format described above.
func (p *PredictedFundings) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		// Maybe it's empty or another form
		return err
	}

	var entries []PredictedFundingEntry
	for _, item := range raw {
		// each item is [ coin, venuesArray ]
		var pair []json.RawMessage
		if err := json.Unmarshal(item, &pair); err != nil {
			return err
		}
		if len(pair) != 2 {
			continue
		}
		var coin string
		if err := json.Unmarshal(pair[0], &coin); err != nil {
			return err
		}
		var venuesRaw []json.RawMessage
		if err := json.Unmarshal(pair[1], &venuesRaw); err != nil {
			return err
		}
		var venues []PredictedFundingVenue
		for _, vr := range venuesRaw {
			var vpair []json.RawMessage
			if err := json.Unmarshal(vr, &vpair); err != nil {
				return err
			}
			if len(vpair) != 2 {
				continue
			}
			var venue string
			if err := json.Unmarshal(vpair[0], &venue); err != nil {
				return err
			}
			var info PredictedFundingInfo
			if err := json.Unmarshal(vpair[1], &info); err != nil {
				return err
			}
			venues = append(venues, PredictedFundingVenue{Venue: venue, Info: info})
		}
		entries = append(entries, PredictedFundingEntry{Coin: coin, Venues: venues})
	}
	*p = entries
	return nil
}

// SubAccount represents a user's subaccount info returned by info API
type SubAccount struct {
	Name               string    `json:"name"`
	SubAccountUser     string    `json:"subAccountUser"`
	Master             string    `json:"master"`
	ClearinghouseState UserState `json:"clearinghouseState"`
	SpotState          struct {
		Balances []SpotBalance `json:"balances"`
	} `json:"spotState"`
}

// OrderQueryInner models the inner order/status structure returned by orderStatus
type OrderQueryInner struct {
	Order           OpenOrder `json:"order"`
	Status          string    `json:"status"`
	StatusTimestamp int64     `json:"statusTimestamp"`
}

// OrderQueryResponse is the wrapper returned by orderStatus
type OrderQueryResponse struct {
	Status string          `json:"status"`
	Order  OrderQueryInner `json:"order"`
}

// TwapSliceFill represents a TWAP slice fill with metadata
type TwapSliceFill struct {
	Fill   Fill `json:"fill"`
	TwapId int  `json:"twapId"`
}

// UserRole represents a user's role
type UserRole struct {
	Role string `json:"role"`
}

// PerpDexLimits represents builder-deployed perp market limits
type PerpDexLimits struct {
	TotalOiCap     string     `json:"totalOiCap"`
	OiSzCapPerPerp string     `json:"oiSzCapPerPerp"`
	MaxTransferNtl string     `json:"maxTransferNtl"`
	CoinToOiCap    [][]string `json:"coinToOiCap"`
}

// PerpDexStatus represents simple status for perp dex
type PerpDexStatus struct {
	TotalNetDeposit string `json:"totalNetDeposit"`
}

// ActiveAssetData represents user's active asset data
type ActiveAssetData struct {
	User             string   `json:"user"`
	Coin             string   `json:"coin"`
	Leverage         Leverage `json:"leverage"`
	MaxTradeSzs      []string `json:"maxTradeSzs"`
	AvailableToTrade []string `json:"availableToTrade"`
	MarkPx           string   `json:"markPx"`
}

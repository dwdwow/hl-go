// Package ws types defines all WebSocket message structures for Hyperliquid.
package ws

import "github.com/dwdwow/hl-go/types"

// WebSocket data type definitions based on Hyperliquid API documentation.
//
// These types represent the various messages received from Hyperliquid WebSocket feeds.
// All types are designed to match the exact structure of messages from the API.

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
// type Subscription struct {
// 	Type     SubscriptionType `json:"type"`
// 	Coin     *string          `json:"coin,omitempty"`
// 	User     *string          `json:"user,omitempty"`
// 	Interval *string          `json:"interval,omitempty"`
// }

// WsTrade represents a trade update
type WsTrade struct {
	Coin  string    `json:"coin"`
	Side  string    `json:"side"`
	Px    float64   `json:"px,string"`
	Sz    float64   `json:"sz,string"`
	Hash  string    `json:"hash"`
	Time  int64     `json:"time"`
	Tid   int64     `json:"tid"`   // 50-bit hash of (buyer_oid, seller_oid)
	Users [2]string `json:"users"` // [buyer, seller]
}

// WsBook represents order book snapshot updates
type WsBook struct {
	Coin   string       `json:"coin"`
	Levels [2][]WsLevel `json:"levels"` // [bids, asks]
	Time   int64        `json:"time"`
}

// WsLevel represents a price level in the order book
type WsLevel struct {
	Px float64 `json:"px,string"` // price
	Sz float64 `json:"sz,string"` // size
	N  int     `json:"n"`         // number of orders
}

// WsBbo represents best bid/offer updates
type WsBbo struct {
	Coin string      `json:"coin"`
	Time int64       `json:"time"`
	Bbo  [2]*WsLevel `json:"bbo"` // [bid, ask], can be null
}

// AllMids represents all mid prices
type AllMids struct {
	Mids map[string]string `json:"mids"`
}

// Notification represents a notification message
type Notification struct {
	Notification string `json:"notification"`
}

// Candle represents candlestick data
type Candle struct {
	T  int64   `json:"t"` // open millis
	T2 int64   `json:"T"` // close millis
	S  string  `json:"s"` // coin
	I  string  `json:"i"` // interval
	O  float64 `json:"o"` // open price
	C  float64 `json:"c"` // close price
	H  float64 `json:"h"` // high price
	L  float64 `json:"l"` // low price
	V  float64 `json:"v"` // volume (base unit)
	N  int     `json:"n"` // number of trades
}

// WsOrder represents user order updates
type WsOrder struct {
	Order           WsBasicOrder          `json:"order"`
	Status          types.OrderStatusType `json:"status"`
	StatusTimestamp int64                 `json:"statusTimestamp"`
}

// WsBasicOrder represents basic order information
type WsBasicOrder struct {
	Coin      string  `json:"coin"`
	Side      string  `json:"side"`
	LimitPx   string  `json:"limitPx"`
	Sz        string  `json:"sz"`
	Oid       int64   `json:"oid"`
	Timestamp int64   `json:"timestamp"`
	OrigSz    string  `json:"origSz"`
	Cloid     *string `json:"cloid,omitempty"`
}

// WsUserEvent represents user events (fills, funding, liquidation, or non-user cancel)
type WsUserEvent struct {
	Fills         []WsFill          `json:"fills,omitempty"`
	Funding       *WsUserFunding    `json:"funding,omitempty"`
	Liquidation   *WsLiquidation    `json:"liquidation,omitempty"`
	NonUserCancel []WsNonUserCancel `json:"nonUserCancel,omitempty"`
}

// WsUserFills represents fills snapshot followed by streaming fills
type WsUserFills struct {
	IsSnapshot *bool    `json:"isSnapshot,omitempty"`
	User       string   `json:"user"`
	Fills      []WsFill `json:"fills"`
}

// WsFill represents a fill
type WsFill struct {
	Coin          string           `json:"coin"`
	Px            string           `json:"px"` // price
	Sz            string           `json:"sz"` // size
	Side          string           `json:"side"`
	Time          int64            `json:"time"`
	StartPosition string           `json:"startPosition"`
	Dir           string           `json:"dir"` // used for frontend display
	ClosedPnl     string           `json:"closedPnl"`
	Hash          string           `json:"hash"`    // L1 transaction hash
	Oid           int64            `json:"oid"`     // order id
	Crossed       bool             `json:"crossed"` // whether order crossed the spread (was taker)
	Fee           string           `json:"fee"`     // negative means rebate
	Tid           int64            `json:"tid"`     // unique trade id
	Liquidation   *FillLiquidation `json:"liquidation,omitempty"`
	FeeToken      string           `json:"feeToken"`             // the token the fee was paid in
	BuilderFee    *string          `json:"builderFee,omitempty"` // amount paid to builder
}

// FillLiquidation represents liquidation details in a fill
type FillLiquidation struct {
	LiquidatedUser *string `json:"liquidatedUser,omitempty"`
	MarkPx         float64 `json:"markPx"`
	Method         string  `json:"method"` // "market" or "backstop"
}

// WsUserFunding represents a funding payment
type WsUserFunding struct {
	Time        int64  `json:"time"`
	Coin        string `json:"coin"`
	Usdc        string `json:"usdc"`
	Szi         string `json:"szi"`
	FundingRate string `json:"fundingRate"`
}

// WsUserFundings represents funding payments snapshot followed by funding payments
type WsUserFundings struct {
	IsSnapshot *bool           `json:"isSnapshot,omitempty"`
	User       string          `json:"user"`
	Fundings   []WsUserFunding `json:"fundings"`
}

// WsLiquidation represents a liquidation event
type WsLiquidation struct {
	Lid                    int64  `json:"lid"`
	Liquidator             string `json:"liquidator"`
	LiquidatedUser         string `json:"liquidated_user"`
	LiquidatedNtlPos       string `json:"liquidated_ntl_pos"`
	LiquidatedAccountValue string `json:"liquidated_account_value"`
}

// WsNonUserCancel represents a non-user cancel event
type WsNonUserCancel struct {
	Coin string `json:"coin"`
	Oid  int64  `json:"oid"`
}

// WsUserNonFundingLedgerUpdates represents ledger updates not including funding payments
type WsUserNonFundingLedgerUpdates struct {
	IsSnapshot *bool                    `json:"isSnapshot,omitempty"`
	User       string                   `json:"user"`
	Updates    []NonFundingLedgerUpdate `json:"updates"`
}

// NonFundingLedgerUpdate represents a ledger update (withdrawal, deposit, transfer, or liquidation)
type NonFundingLedgerUpdate struct {
	// Define based on actual API response structure
	// This is a placeholder - adjust according to actual data
	Time int64                  `json:"time"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// WsActiveAssetCtx represents active asset context (perps)
type WsActiveAssetCtx struct {
	Coin string        `json:"coin"`
	Ctx  PerpsAssetCtx `json:"ctx"`
}

// WsActiveSpotAssetCtx represents active asset context (spot)
type WsActiveSpotAssetCtx struct {
	Coin string       `json:"coin"`
	Ctx  SpotAssetCtx `json:"ctx"`
}

// SharedAssetCtx contains shared asset context fields
type SharedAssetCtx struct {
	DayNtlVlm float64  `json:"dayNtlVlm"`
	PrevDayPx float64  `json:"prevDayPx"`
	MarkPx    float64  `json:"markPx"`
	MidPx     *float64 `json:"midPx,omitempty"`
}

// PerpsAssetCtx represents perpetual asset context
type PerpsAssetCtx struct {
	SharedAssetCtx
	Funding      float64 `json:"funding"`
	OpenInterest float64 `json:"openInterest"`
	OraclePx     float64 `json:"oraclePx"`
}

// SpotAssetCtx represents spot asset context
type SpotAssetCtx struct {
	SharedAssetCtx
	CirculatingSupply float64 `json:"circulatingSupply"`
}

// WsActiveAssetData represents active asset data for a user
type WsActiveAssetData struct {
	User             string      `json:"user"`
	Coin             string      `json:"coin"`
	Leverage         interface{} `json:"leverage"` // Can be various types
	MaxTradeSzs      [2]float64  `json:"maxTradeSzs"`
	AvailableToTrade [2]float64  `json:"availableToTrade"`
}

// WsUserTwapSliceFills represents TWAP slice fills
type WsUserTwapSliceFills struct {
	IsSnapshot     *bool             `json:"isSnapshot,omitempty"`
	User           string            `json:"user"`
	TwapSliceFills []WsTwapSliceFill `json:"twapSliceFills"`
}

// WsTwapSliceFill represents a TWAP slice fill
type WsTwapSliceFill struct {
	Fill   WsFill `json:"fill"`
	TwapId int64  `json:"twapId"`
}

// WsUserTwapHistory represents TWAP history
type WsUserTwapHistory struct {
	IsSnapshot *bool           `json:"isSnapshot,omitempty"`
	User       string          `json:"user"`
	History    []WsTwapHistory `json:"history"`
}

// WsTwapHistory represents a TWAP history entry
type WsTwapHistory struct {
	State  TwapState  `json:"state"`
	Status TwapStatus `json:"status"`
	Time   int64      `json:"time"`
}

// TwapState represents TWAP state
type TwapState struct {
	Coin        string  `json:"coin"`
	User        string  `json:"user"`
	Side        string  `json:"side"`
	Sz          float64 `json:"sz"`
	ExecutedSz  float64 `json:"executedSz"`
	ExecutedNtl float64 `json:"executedNtl"`
	Minutes     int     `json:"minutes"`
	ReduceOnly  bool    `json:"reduceOnly"`
	Randomize   bool    `json:"randomize"`
	Timestamp   int64   `json:"timestamp"`
}

// TwapStatus represents TWAP status
type TwapStatus struct {
	Status      string `json:"status"` // "activated" | "terminated" | "finished" | "error"
	Description string `json:"description"`
}

// WebData2 represents aggregate information about a user
type WebData2 struct {
	// Define based on actual API response structure
	// This is a placeholder - adjust according to actual data
	User string                 `json:"user"`
	Data map[string]interface{} `json:"data"`
}

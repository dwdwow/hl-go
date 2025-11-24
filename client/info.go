// Package client provides the Info client for querying Hyperliquid market data and user information.
package client

import (
	"fmt"
	"time"

	"github.com/dwdwow/hl-go/constants"
	"github.com/dwdwow/hl-go/types"
)

// Info provides read-only access to Hyperliquid market data and user information
type Info struct {
	*API
	coinToAsset       map[string]int
	nameToCoin        map[string]string
	assetToSzDecimals map[int]int
}

// NewInfo creates a new Info client
// If skipWS is false, WebSocket connections will be initialized (not yet implemented)
func NewInfoUsingHTTP(baseURL string, timeout time.Duration) (*Info, error) {
	info := &Info{
		API:               NewAPIUsingHTTP(baseURL, timeout),
		coinToAsset:       make(map[string]int),
		nameToCoin:        make(map[string]string),
		assetToSzDecimals: make(map[int]int),
	}

	// Initialize metadata
	if err := info.initializeMetadata(); err != nil {
		return nil, fmt.Errorf("failed to initialize metadata: %w", err)
	}

	return info, nil
}

func NewInfoUsingWs(baseURL string, timeout time.Duration) (*Info, error) {
	w, err := newAPIUsingWs(baseURL, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create API: %w", err)
	}
	info := &Info{
		API:               w,
		coinToAsset:       make(map[string]int),
		nameToCoin:        make(map[string]string),
		assetToSzDecimals: make(map[int]int),
	}

	// Initialize metadata
	if err := info.initializeMetadata(); err != nil {
		return nil, fmt.Errorf("failed to initialize metadata: %w", err)
	}

	return info, nil
}

// initializeMetadata fetches and caches asset metadata
func (i *Info) initializeMetadata() error {
	// Get spot metadata
	spotMeta, err := i.SpotMeta()
	if err != nil {
		return fmt.Errorf("failed to get spot meta: %w", err)
	}

	// Process spot assets (start at 10000)
	for _, spotInfo := range spotMeta.Universe {
		asset := spotInfo.Index + constants.SpotAssetOffset
		i.coinToAsset[spotInfo.Name] = asset
		i.nameToCoin[spotInfo.Name] = spotInfo.Name

		baseToken := spotMeta.Tokens[spotInfo.Tokens[0]]
		quoteToken := spotMeta.Tokens[spotInfo.Tokens[1]]
		i.assetToSzDecimals[asset] = baseToken.SzDecimals

		// Also map base/quote format
		name := fmt.Sprintf("%s/%s", baseToken.Name, quoteToken.Name)
		if _, exists := i.nameToCoin[name]; !exists {
			i.nameToCoin[name] = spotInfo.Name
		}
	}

	// Get perp metadata (default dex "")
	perpMeta, err := i.Meta("")
	if err != nil {
		return fmt.Errorf("failed to get perp meta: %w", err)
	}

	// Process perp assets
	for asset, assetInfo := range perpMeta.Universe {
		i.coinToAsset[assetInfo.Name] = asset
		i.nameToCoin[assetInfo.Name] = assetInfo.Name
		i.assetToSzDecimals[asset] = assetInfo.SzDecimals
	}

	return nil
}

func (i *Info) NameToCoin(name string) (string, error) {
	coin, ok := i.nameToCoin[name]
	if !ok {
		return "", fmt.Errorf("unknown coin name: %s", name)
	}
	return coin, nil
}

// NameToAsset converts a coin name to its asset ID
func (i *Info) NameToAsset(name string) (int, error) {
	coin, ok := i.nameToCoin[name]
	if !ok {
		return 0, fmt.Errorf("unknown coin name: %s", name)
	}

	asset, ok := i.coinToAsset[coin]
	if !ok {
		return 0, fmt.Errorf("unknown coin: %s", coin)
	}

	return asset, nil
}

// UserState retrieves trading details about a user
// Returns position information, margin summary, and withdrawable balance
func (i *Info) UserState(address string, dex string) (*types.UserState, error) {
	payload := map[string]any{
		"type": "clearinghouseState",
		"user": address,
		"dex":  dex,
	}

	var result types.UserState
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotUserState retrieves spot trading state for a user
func (i *Info) SpotUserState(address string) (*types.SpotUserState, error) {
	payload := map[string]any{
		"type": "spotClearinghouseState",
		"user": address,
	}

	var result types.SpotUserState
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// OpenOrders retrieves a user's open orders
func (i *Info) OpenOrders(address string, dex string) ([]types.OpenOrder, error) {
	payload := map[string]any{
		"type": "openOrders",
		"user": address,
		"dex":  dex,
	}

	var result []types.OpenOrder
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// FrontendOpenOrders retrieves a user's open orders with additional frontend info
func (i *Info) FrontendOpenOrders(address string, dex string) ([]types.FrontendOpenOrder, error) {
	payload := map[string]any{
		"type": "frontendOpenOrders",
		"user": address,
		"dex":  dex,
	}

	var result []types.FrontendOpenOrder
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// AllMids retrieves all mid prices for actively traded coins
func (i *Info) AllMids(dex string) (map[string]string, error) {
	payload := map[string]any{
		"type": "allMids",
		"dex":  dex,
	}

	var result map[string]string
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UserFills retrieves a given user's fills
func (i *Info) UserFills(address string) ([]types.Fill, error) {
	payload := map[string]any{
		"type": "userFills",
		"user": address,
	}

	var result []types.Fill
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UserFillsByTime retrieves a given user's fills by time range
func (i *Info) UserFillsByTime(address string, startTime int64, endTime *int64, aggregateByTime bool) ([]types.Fill, error) {
	payload := map[string]any{
		"type":            "userFillsByTime",
		"user":            address,
		"startTime":       startTime,
		"aggregateByTime": aggregateByTime,
	}

	if endTime != nil {
		payload["endTime"] = *endTime
	}

	var result []types.Fill
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Meta retrieves exchange perpetual metadata
func (i *Info) Meta(dex string) (*types.Meta, error) {
	payload := map[string]any{
		"type": "meta",
		"dex":  dex,
	}

	var result types.Meta
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// MetaAndAssetCtxs retrieves exchange metadata with asset contexts
func (i *Info) MetaAndAssetCtxs() (*types.MetaAndAssetCtxs, error) {
	payload := map[string]any{
		"type": "metaAndAssetCtxs",
	}

	var result types.MetaAndAssetCtxs
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PerpDexs retrieves all perpetual DEXs
func (i *Info) PerpDexs() ([]types.PerpDex, error) {
	payload := map[string]any{
		"type": "perpDexs",
	}

	var result []types.PerpDex
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SpotMeta retrieves exchange spot metadata
func (i *Info) SpotMeta() (*types.SpotMeta, error) {
	payload := map[string]any{
		"type": "spotMeta",
	}

	var result types.SpotMeta
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SpotMetaAndAssetCtxs retrieves exchange spot asset contexts
func (i *Info) SpotMetaAndAssetCtxs() (*types.SpotMetaAndAssetCtxs, error) {
	payload := map[string]any{
		"type": "spotMetaAndAssetCtxs",
	}

	var result types.SpotMetaAndAssetCtxs
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FundingHistory retrieves funding history for a given coin
func (i *Info) FundingHistory(name string, startTime int64, endTime *int64) ([]types.FundingRecord, error) {
	coin, ok := i.nameToCoin[name]
	if !ok {
		return nil, fmt.Errorf("unknown coin: %s", name)
	}

	payload := map[string]any{
		"type":      "fundingHistory",
		"coin":      coin,
		"startTime": startTime,
	}

	if endTime != nil {
		payload["endTime"] = *endTime
	}

	var result []types.FundingRecord
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UserFundingHistory retrieves a user's funding history
func (i *Info) UserFundingHistory(user string, startTime int64, endTime *int64) ([]types.FundingRecord, error) {
	payload := map[string]any{
		"type":      "userFunding",
		"user":      user,
		"startTime": startTime,
	}

	if endTime != nil {
		payload["endTime"] = *endTime
	}

	var result []types.FundingRecord
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// L2Snapshot retrieves L2 order book snapshot for a given coin
func (i *Info) L2Snapshot(name string) (*types.L2BookData, error) {
	coin, ok := i.nameToCoin[name]
	if !ok {
		return nil, fmt.Errorf("unknown coin: %s", name)
	}

	payload := map[string]any{
		"type": "l2Book",
		"coin": coin,
	}

	var result types.L2BookData
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CandlesSnapshot retrieves candles snapshot for a given coin
func (i *Info) CandlesSnapshot(name string, interval string, startTime int64, endTime int64) ([]types.Candle, error) {
	coin, ok := i.nameToCoin[name]
	if !ok {
		return nil, fmt.Errorf("unknown coin: %s", name)
	}

	req := map[string]any{
		"coin":      coin,
		"interval":  interval,
		"startTime": startTime,
		"endTime":   endTime,
	}

	payload := map[string]any{
		"type": "candleSnapshot",
		"req":  req,
	}

	var result []types.Candle
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UserFees retrieves the volume of trading activity associated with a user
func (i *Info) UserFees(address string) (*types.UserFees, error) {
	payload := map[string]any{
		"type": "userFees",
		"user": address,
	}

	var result types.UserFees
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UserStakingSummary retrieves the staking summary associated with a user
func (i *Info) UserStakingSummary(address string) (*types.DelegatorSummary, error) {
	payload := map[string]any{
		"type": "delegatorSummary",
		"user": address,
	}

	var result types.DelegatorSummary
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UserStakingDelegations retrieves the user's staking delegations
func (i *Info) UserStakingDelegations(address string) ([]types.Delegation, error) {
	payload := map[string]any{
		"type": "delegations",
		"user": address,
	}

	var result []types.Delegation
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UserStakingRewards retrieves the historic staking rewards associated with a user
func (i *Info) UserStakingRewards(address string) ([]types.DelegatorReward, error) {
	payload := map[string]any{
		"type": "delegatorRewards",
		"user": address,
	}

	var result []types.DelegatorReward
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// DelegatorHistory retrieves comprehensive staking history for a user
func (i *Info) DelegatorHistory(user string) ([]types.DelegatorHistoryEntry, error) {
	payload := map[string]any{
		"type": "delegatorHistory",
		"user": user,
	}

	var result []types.DelegatorHistoryEntry
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// QueryOrderByOid queries order status by order ID
func (i *Info) QueryOrderByOid(user string, oid int) (*types.OrderQueryResponse, error) {
	payload := map[string]any{
		"type": "orderStatus",
		"user": user,
		"oid":  oid,
	}

	var result types.OrderQueryResponse
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QueryOrderByCloid queries order status by client order ID
func (i *Info) QueryOrderByCloid(user string, cloid *types.Cloid) (*types.OrderQueryResponse, error) {
	payload := map[string]any{
		"type": "orderStatus",
		"user": user,
		"oid":  cloid.ToRaw(),
	}

	var result types.OrderQueryResponse
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QueryReferralState queries referral state for a user
func (i *Info) QueryReferralState(user string) (*types.ReferralResponse, error) {
	payload := map[string]any{
		"type": "referral",
		"user": user,
	}

	var result types.ReferralResponse
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QuerySubAccounts queries sub-accounts for a user
func (i *Info) QuerySubAccounts(user string) ([]types.SubAccount, error) {
	payload := map[string]any{
		"type": "subAccounts",
		"user": user,
	}

	var result []types.SubAccount
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// HistoricalOrders retrieves a user's historical orders (max 2000 most recent)
func (i *Info) HistoricalOrders(user string) ([]types.OrderQueryInner, error) {
	payload := map[string]any{
		"type": "historicalOrders",
		"user": user,
	}

	var result []types.OrderQueryInner
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UserNonFundingLedgerUpdates retrieves non-funding ledger updates for a user
func (i *Info) UserNonFundingLedgerUpdates(user string, startTime int64, endTime *int64) (types.RawJSON, error) {
	payload := map[string]any{
		"type":      "userNonFundingLedgerUpdates",
		"user":      user,
		"startTime": startTime,
	}

	if endTime != nil {
		payload["endTime"] = *endTime
	}

	var result types.RawJSON
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Portfolio retrieves comprehensive portfolio performance data
func (i *Info) Portfolio(user string) (types.RawJSON, error) {
	payload := map[string]any{
		"type": "portfolio",
		"user": user,
	}
	var result types.RawJSON
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ExtraAgents retrieves extra agents associated with a user
func (i *Info) ExtraAgents(user string) (types.RawJSON, error) {
	payload := map[string]any{
		"type": "extraAgents",
		"user": user,
	}

	var result types.RawJSON
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// QueryUserToMultiSigSigners retrieves the multi-sig signers for a multi-sig user
func (i *Info) QueryUserToMultiSigSigners(multiSigUser string) (types.RawJSON, error) {
	payload := map[string]any{
		"type": "userToMultiSigSigners",
		"user": multiSigUser,
	}

	var result types.RawJSON
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// QueryPerpDeployAuctionStatus retrieves the perp deploy auction status
func (i *Info) QueryPerpDeployAuctionStatus() (*types.PerpDeployAuctionStatus, error) {
	payload := map[string]any{
		"type": "perpDeployAuctionStatus",
	}

	var result types.PerpDeployAuctionStatus
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QueryUserDexAbstractionState retrieves the dex abstraction state for a user
func (i *Info) QueryUserDexAbstractionState(user string) (bool, error) {
	payload := map[string]any{
		"type": "userDexAbstraction",
		"user": user,
	}

	var result bool
	if err := i.infoPost("/info", payload, &result); err != nil {
		return false, err
	}

	return result, nil
}

// UserTwapSliceFills retrieves a user's TWAP slice fills (at most 2000 most recent)
func (i *Info) UserTwapSliceFills(user string) ([]types.TwapSliceFill, error) {
	payload := map[string]any{
		"type": "userTwapSliceFills",
		"user": user,
	}

	var result []types.TwapSliceFill
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UserVaultEquities retrieves user's equity positions across all vaults
func (i *Info) UserVaultEquities(user string) ([]types.VaultEquity, error) {
	payload := map[string]any{
		"type": "userVaultEquities",
		"user": user,
	}

	var result []types.VaultEquity
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// UserRole retrieves the role and account type information for a user
func (i *Info) UserRole(user string) (*types.UserRole, error) {
	payload := map[string]any{
		"type": "userRole",
		"user": user,
	}

	var result types.UserRole
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UserRateLimit retrieves user's API rate limit configuration and usage
func (i *Info) UserRateLimit(user string) (*types.UserRateLimitResponse, error) {
	payload := map[string]any{
		"type": "userRateLimit",
		"user": user,
	}

	var result types.UserRateLimitResponse
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QuerySpotDeployAuctionStatus retrieves the spot deploy auction status for a user
func (i *Info) QuerySpotDeployAuctionStatus(user string) (*types.SpotDeployState, error) {
	payload := map[string]any{
		"type": "spotDeployState",
		"user": user,
	}

	var result types.SpotDeployState
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// QuerySpotPairDeployAuctionStatus retrieves the spot pair deploy auction status
func (i *Info) QuerySpotPairDeployAuctionStatus() (*types.AuctionStatus, error) {
	payload := map[string]any{
		"type": "spotPairDeployAuctionStatus",
	}

	var result types.AuctionStatus
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TokenDetails retrieves information about a token by tokenId
func (i *Info) TokenDetails(tokenId string) (*types.TokenDetails, error) {
	payload := map[string]any{
		"type":    "tokenDetails",
		"tokenId": tokenId,
	}

	var result types.TokenDetails
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PredictedFundings retrieves predicted funding rates for different venues
func (i *Info) PredictedFundings() (types.PredictedFundings, error) {
	payload := map[string]any{
		"type": "predictedFundings",
	}

	var result types.PredictedFundings
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// PerpsAtOpenInterestCap returns coin names that are at open interest cap
func (i *Info) PerpsAtOpenInterestCap() ([]string, error) {
	payload := map[string]any{
		"type": "perpsAtOpenInterestCap",
	}

	var result []string
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// PerpDexLimits retrieves builder-deployed perp market limits
func (i *Info) PerpDexLimits(dex string) (*types.PerpDexLimits, error) {
	payload := map[string]any{
		"type": "perpDexLimits",
		"dex":  dex,
	}

	var result types.PerpDexLimits
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PerpDexStatus retrieves simple status data for a perp dex
func (i *Info) PerpDexStatus(dex string) (*types.PerpDexStatus, error) {
	payload := map[string]any{
		"type": "perpDexStatus",
		"dex":  dex,
	}

	var result types.PerpDexStatus
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ActiveAssetData retrieves a user's active asset data for a coin
func (i *Info) ActiveAssetData(user string, coin string) (*types.ActiveAssetData, error) {
	payload := map[string]any{
		"type": "activeAssetData",
		"user": user,
		"coin": coin,
	}

	var result types.ActiveAssetData
	if err := i.infoPost("/info", payload, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

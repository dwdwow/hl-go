package client

import (
	"testing"
	"time"

	"github.com/dwdwow/hl-go/constants"
)

const (
	testAddress = "0x0000000000000000000000000000000000000000" // 测试地址
	testCoin    = "BTC"
)

func getTestInfo(t *testing.T) *Info {
	info, err := NewInfo(constants.MainnetAPIURL, 30*time.Second)
	if err != nil {
		t.Fatalf("NewInfo() error = %v", err)
	}
	return info
}

func TestInfo_AllMids(t *testing.T) {
	info := getTestInfo(t)

	mids, err := info.AllMids("")
	if err != nil {
		t.Fatalf("AllMids() error = %v", err)
	}

	t.Logf("All mids count: %d", len(mids))
	for coin, price := range mids {
		t.Logf("  %s: %s", coin, price)
		if len(mids) > 5 {
			break // 只打印前几个
		}
	}
}

func TestInfo_Meta(t *testing.T) {
	info := getTestInfo(t)

	meta, err := info.Meta("")
	if err != nil {
		t.Fatalf("Meta() error = %v", err)
	}

	t.Logf("Universe count: %d", len(meta.Universe))
	for i, asset := range meta.Universe {
		t.Logf("  [%d] %s (decimals: %d)", i, asset.Name, asset.SzDecimals)
		if i >= 4 {
			break
		}
	}
}

func TestInfo_SpotMeta(t *testing.T) {
	info := getTestInfo(t)

	meta, err := info.SpotMeta()
	if err != nil {
		t.Fatalf("SpotMeta() error = %v", err)
	}

	t.Logf("Spot universe count: %d", len(meta.Universe))
	t.Logf("Spot tokens count: %d", len(meta.Tokens))

	for i, asset := range meta.Universe {
		t.Logf("  [%d] %s (canonical: %v)", i, asset.Name, asset.IsCanonical)
		if i >= 4 {
			break
		}
	}
}

func TestInfo_MetaAndAssetCtxs(t *testing.T) {
	info := getTestInfo(t)

	result, err := info.MetaAndAssetCtxs()
	if err != nil {
		t.Fatalf("MetaAndAssetCtxs() error = %v", err)
	}

	t.Logf("MetaAndAssetCtxs result keys: %v", getMapKeys(result))
}

func TestInfo_SpotMetaAndAssetCtxs(t *testing.T) {
	info := getTestInfo(t)

	result, err := info.SpotMetaAndAssetCtxs()
	if err != nil {
		t.Fatalf("SpotMetaAndAssetCtxs() error = %v", err)
	}

	t.Logf("SpotMetaAndAssetCtxs result keys: %v", getMapKeys(result))
}

func TestInfo_UserState(t *testing.T) {
	info := getTestInfo(t)

	state, err := info.UserState(testAddress, "")
	if err != nil {
		t.Fatalf("UserState() error = %v", err)
	}

	t.Logf("Account value: %s", state.MarginSummary.AccountValue)
	t.Logf("Withdrawable: %s", state.Withdrawable)
	t.Logf("Asset positions count: %d", len(state.AssetPositions))
}

func TestInfo_OpenOrders(t *testing.T) {
	info := getTestInfo(t)

	orders, err := info.OpenOrders(testAddress, "")
	if err != nil {
		t.Fatalf("OpenOrders() error = %v", err)
	}

	t.Logf("Open orders count: %d", len(orders))
	for i, order := range orders {
		t.Logf("  [%d] %s: %s @ %s (oid: %d)", i, order.Coin, order.Sz, order.LimitPx, order.Oid)
		if i >= 4 {
			break
		}
	}
}

func TestInfo_FrontendOpenOrders(t *testing.T) {
	info := getTestInfo(t)

	orders, err := info.FrontendOpenOrders(testAddress, "")
	if err != nil {
		t.Fatalf("FrontendOpenOrders() error = %v", err)
	}

	t.Logf("Frontend open orders count: %d", len(orders))
}

func TestInfo_UserFills(t *testing.T) {
	info := getTestInfo(t)

	fills, err := info.UserFills(testAddress)
	if err != nil {
		t.Fatalf("UserFills() error = %v", err)
	}

	t.Logf("User fills count: %d", len(fills))
	for i, fill := range fills {
		t.Logf("  [%d] %s: %s @ %s (fee: %s)", i, fill.Coin, fill.Sz, fill.Px, fill.Fee)
		if i >= 4 {
			break
		}
	}
}

func TestInfo_UserFillsByTime(t *testing.T) {
	info := getTestInfo(t)

	startTime := time.Now().Add(-24 * time.Hour).UnixMilli()
	fills, err := info.UserFillsByTime(testAddress, startTime, nil, false)
	if err != nil {
		t.Fatalf("UserFillsByTime() error = %v", err)
	}

	t.Logf("User fills (24h) count: %d", len(fills))
}

func TestInfo_L2Snapshot(t *testing.T) {
	info := getTestInfo(t)

	l2, err := info.L2Snapshot(testCoin)
	if err != nil {
		t.Fatalf("L2Snapshot() error = %v", err)
	}

	t.Logf("L2 Book for %s:", l2.Coin)
	t.Logf("  Bids: %d levels", len(l2.Levels[0]))
	t.Logf("  Asks: %d levels", len(l2.Levels[1]))

	if len(l2.Levels[0]) > 0 {
		t.Logf("  Best bid: %s @ %s", l2.Levels[0][0].Sz, l2.Levels[0][0].Px)
	}
	if len(l2.Levels[1]) > 0 {
		t.Logf("  Best ask: %s @ %s", l2.Levels[1][0].Sz, l2.Levels[1][0].Px)
	}
}

func TestInfo_CandlesSnapshot(t *testing.T) {
	info := getTestInfo(t)

	endTime := time.Now().UnixMilli()
	startTime := endTime - 3600000 // 1小时前

	candles, err := info.CandlesSnapshot(testCoin, "1m", startTime, endTime)
	if err != nil {
		t.Fatalf("CandlesSnapshot() error = %v", err)
	}

	t.Logf("Candles count: %d", len(candles))
	for i, candle := range candles {
		t.Logf("  [%d] %v", i, candle)
		if i >= 2 {
			break
		}
	}
}

func TestInfo_FundingHistory(t *testing.T) {
	info := getTestInfo(t)

	endTime := time.Now().UnixMilli()
	startTime := endTime - 24*3600000 // 24小时前

	history, err := info.FundingHistory(testCoin, startTime, &endTime)
	if err != nil {
		t.Fatalf("FundingHistory() error = %v", err)
	}

	t.Logf("Funding history count: %d", len(history))
	for i, record := range history {
		t.Logf("  [%d] %v", i, record)
		if i >= 2 {
			break
		}
	}
}

func TestInfo_UserFundingHistory(t *testing.T) {
	info := getTestInfo(t)

	endTime := time.Now().UnixMilli()
	startTime := endTime - 24*3600000

	history, err := info.UserFundingHistory(testAddress, startTime, &endTime)
	if err != nil {
		t.Fatalf("UserFundingHistory() error = %v", err)
	}

	t.Logf("User funding history count: %d", len(history))
}

func TestInfo_UserFees(t *testing.T) {
	info := getTestInfo(t)

	fees, err := info.UserFees(testAddress)
	if err != nil {
		t.Fatalf("UserFees() error = %v", err)
	}

	t.Logf("User fees: %v", fees)
}

func TestInfo_UserStakingSummary(t *testing.T) {
	info := getTestInfo(t)

	summary, err := info.UserStakingSummary(testAddress)
	if err != nil {
		t.Fatalf("UserStakingSummary() error = %v", err)
	}

	t.Logf("Staking summary: %v", summary)
}

func TestInfo_UserStakingDelegations(t *testing.T) {
	info := getTestInfo(t)

	delegations, err := info.UserStakingDelegations(testAddress)
	if err != nil {
		t.Fatalf("UserStakingDelegations() error = %v", err)
	}

	t.Logf("Delegations count: %d", len(delegations))
}

func TestInfo_UserStakingRewards(t *testing.T) {
	info := getTestInfo(t)

	rewards, err := info.UserStakingRewards(testAddress)
	if err != nil {
		t.Fatalf("UserStakingRewards() error = %v", err)
	}

	t.Logf("Rewards count: %d", len(rewards))
}

func TestInfo_DelegatorHistory(t *testing.T) {
	info := getTestInfo(t)

	history, err := info.DelegatorHistory(testAddress)
	if err != nil {
		t.Fatalf("DelegatorHistory() error = %v", err)
	}

	t.Logf("Delegator history: %v", getMapKeys(history))
}

func TestInfo_QueryOrderByOid(t *testing.T) {
	info := getTestInfo(t)

	result, err := info.QueryOrderByOid(testAddress, 12345)
	if err != nil {
		t.Fatalf("QueryOrderByOid() error = %v", err)
	}

	t.Logf("Order query result: %v", result)
}

func TestInfo_QueryReferralState(t *testing.T) {
	info := getTestInfo(t)

	state, err := info.QueryReferralState(testAddress)
	if err != nil {
		t.Fatalf("QueryReferralState() error = %v", err)
	}

	t.Logf("Referral state: %v", state)
}

func TestInfo_QuerySubAccounts(t *testing.T) {
	info := getTestInfo(t)

	subAccounts, err := info.QuerySubAccounts(testAddress)
	if err != nil {
		t.Fatalf("QuerySubAccounts() error = %v", err)
	}

	t.Logf("Sub accounts: %v", subAccounts)
}

func TestInfo_HistoricalOrders(t *testing.T) {
	info := getTestInfo(t)

	orders, err := info.HistoricalOrders(testAddress)
	if err != nil {
		t.Fatalf("HistoricalOrders() error = %v", err)
	}

	t.Logf("Historical orders count: %d", len(orders))
}

func TestInfo_UserNonFundingLedgerUpdates(t *testing.T) {
	info := getTestInfo(t)

	endTime := time.Now().UnixMilli()
	startTime := endTime - 24*3600000

	updates, err := info.UserNonFundingLedgerUpdates(testAddress, startTime, &endTime)
	if err != nil {
		t.Fatalf("UserNonFundingLedgerUpdates() error = %v", err)
	}

	t.Logf("Ledger updates count: %d", len(updates))
}

func TestInfo_Portfolio(t *testing.T) {
	info := getTestInfo(t)

	portfolio, err := info.Portfolio(testAddress)
	if err != nil {
		t.Fatalf("Portfolio() error = %v", err)
	}

	t.Logf("Portfolio: %v", getMapKeys(portfolio))
}

func TestInfo_ExtraAgents(t *testing.T) {
	info := getTestInfo(t)

	agents, err := info.ExtraAgents(testAddress)
	if err != nil {
		t.Fatalf("ExtraAgents() error = %v", err)
	}

	t.Logf("Extra agents count: %d", len(agents))
}

func TestInfo_QueryUserToMultiSigSigners(t *testing.T) {
	info := getTestInfo(t)

	signers, err := info.QueryUserToMultiSigSigners(testAddress)
	if err != nil {
		t.Fatalf("QueryUserToMultiSigSigners() error = %v", err)
	}

	t.Logf("Multi-sig signers: %v", signers)
}

func TestInfo_QueryPerpDeployAuctionStatus(t *testing.T) {
	info := getTestInfo(t)

	status, err := info.QueryPerpDeployAuctionStatus()
	if err != nil {
		t.Fatalf("QueryPerpDeployAuctionStatus() error = %v", err)
	}

	t.Logf("Perp deploy auction status: %v", status)
}

func TestInfo_QueryUserDexAbstractionState(t *testing.T) {
	info := getTestInfo(t)

	state, err := info.QueryUserDexAbstractionState(testAddress)
	if err != nil {
		t.Fatalf("QueryUserDexAbstractionState() error = %v", err)
	}

	t.Logf("DEX abstraction state: %v", state)
}

func TestInfo_UserTwapSliceFills(t *testing.T) {
	info := getTestInfo(t)

	fills, err := info.UserTwapSliceFills(testAddress)
	if err != nil {
		t.Fatalf("UserTwapSliceFills() error = %v", err)
	}

	t.Logf("TWAP slice fills count: %d", len(fills))
}

func TestInfo_UserVaultEquities(t *testing.T) {
	info := getTestInfo(t)

	equities, err := info.UserVaultEquities(testAddress)
	if err != nil {
		t.Fatalf("UserVaultEquities() error = %v", err)
	}

	t.Logf("Vault equities: %v", equities)
}

func TestInfo_UserRole(t *testing.T) {
	info := getTestInfo(t)

	role, err := info.UserRole(testAddress)
	if err != nil {
		t.Fatalf("UserRole() error = %v", err)
	}

	t.Logf("User role: %v", role)
}

func TestInfo_UserRateLimit(t *testing.T) {
	info := getTestInfo(t)

	rateLimit, err := info.UserRateLimit(testAddress)
	if err != nil {
		t.Fatalf("UserRateLimit() error = %v", err)
	}

	t.Logf("Rate limit: %v", rateLimit)
}

func TestInfo_QuerySpotDeployAuctionStatus(t *testing.T) {
	info := getTestInfo(t)

	status, err := info.QuerySpotDeployAuctionStatus(testAddress)
	if err != nil {
		t.Fatalf("QuerySpotDeployAuctionStatus() error = %v", err)
	}

	t.Logf("Spot deploy auction status: %v", status)
}

func TestInfo_PerpDexs(t *testing.T) {
	info := getTestInfo(t)

	dexs, err := info.PerpDexs()
	if err != nil {
		t.Fatalf("PerpDexs() error = %v", err)
	}

	t.Logf("Perp DEXs count: %d", len(dexs))
	for i, dex := range dexs {
		t.Logf("  [%d] %v", i, dex)
		if i >= 2 {
			break
		}
	}
}

func TestInfo_SpotUserState(t *testing.T) {
	info := getTestInfo(t)

	state, err := info.SpotUserState(testAddress)
	if err != nil {
		t.Fatalf("SpotUserState() error = %v", err)
	}

	t.Logf("Spot user state: %v", getMapKeys(state))
}

// 辅助函数：获取map的keys
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

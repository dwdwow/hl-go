// Package constants provides configuration constants for the Hyperliquid API.
package constants

const (
	// MainnetAPIURL is the URL for Hyperliquid mainnet API
	MainnetAPIURL = "https://api.hyperliquid.xyz"

	// TestnetAPIURL is the URL for Hyperliquid testnet API
	TestnetAPIURL = "https://api.hyperliquid-testnet.xyz"

	// LocalAPIURL is the URL for local development
	LocalAPIURL = "http://localhost:3001"

	// DefaultTimeout is the default HTTP request timeout in seconds
	DefaultTimeout = 30

	// DefaultSlippage is the default slippage for market orders (5%)
	DefaultSlippage = 0.05

	// SpotAssetOffset is the starting index for spot assets
	SpotAssetOffset = 10000

	// BuilderPerpDexOffset is the starting index for builder-deployed perp dexs
	BuilderPerpDexOffset = 110000
)

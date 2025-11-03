// Package signing provides EIP-712 signing functionality for Hyperliquid API requests.
//
// This package implements the cryptographic signing required for all authenticated
// operations on the Hyperliquid exchange, following the EIP-712 standard for typed
// structured data hashing and signing.
//
// # Signing Methods
//
// There are two main categories of actions:
//
//  1. L1 Actions (signed with SignL1Action):
//     - Order placement, modification, and cancellation
//     - Leverage and margin updates
//     - Account operations (sub-accounts, referrals, etc.)
//     - Spot and Perp deployment
//     - Validator operations
//     Uses a "phantom agent" pattern where the action is hashed with msgpack,
//     combined with nonce and vault address, then signed via EIP-712.
//
//  2. User-Signed Actions (signed with SignUserSignedAction):
//     - USD and spot transfers
//     - Withdrawals from bridge
//     - Asset transfers between DEXs
//     - Token delegation (staking)
//     - Agent approval
//     - Multi-sig operations
//     Uses direct EIP-712 signing with specific type schemas for each action type.
//
// # Action Hash
//
// The ActionHash function computes a keccak256 hash of:
//   - msgpack-encoded action
//   - 8-byte nonce (big endian)
//   - vault address (if present)
//   - expires_after timestamp (if present)
//
// This hash is used as the connectionId in the phantom agent for L1 actions,
// or directly as the multiSigActionHash for multi-sig envelopes.
//
// # EIP-712 Domains
//
// L1 actions use domain:
//   - name: "Exchange"
//   - version: "1"
//   - chainId: 1337
//
// User-signed actions use domain:
//   - name: "HyperliquidSignTransaction"
//   - version: "1"
//   - chainId: 0x66eee (421614)
//
// # Wire Format Conversion
//
// Float values must be converted to strings with proper precision:
//   - Prices and sizes: 8 decimal places maximum
//   - Values are normalized to remove trailing zeros
//   - Prevents rounding errors during transmission
package signing

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/dwdwow/hl-go/types"
	"github.com/dwdwow/hl-go/utils"
)

// ActionHash computes the hash of an action for signing
func ActionHash(action any, vaultAddress *string, nonce int64, expiresAfter *int64) ([]byte, error) {
	// Encode action with msgpack
	data, err := msgpack.Marshal(action)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal action: %w", err)
	}

	// Append nonce (8 bytes, big endian)
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, uint64(nonce))
	data = append(data, nonceBytes...)

	// Append vault address
	if vaultAddress == nil {
		data = append(data, 0x00)
	} else {
		data = append(data, 0x01)
		addrBytes, err := utils.AddressToBytes(*vaultAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid vault address: %w", err)
		}
		data = append(data, addrBytes...)
	}

	// Append expires after if present
	if expiresAfter != nil {
		data = append(data, 0x00)
		expiresBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(expiresBytes, uint64(*expiresAfter))
		data = append(data, expiresBytes...)
	}

	// Return keccak256 hash
	hash := crypto.Keccak256(data)
	return hash, nil
}

// ConstructPhantomAgent constructs a phantom agent object for L1 signing
func ConstructPhantomAgent(hash []byte, isMainnet bool) map[string]any {
	source := "b"
	if isMainnet {
		source = "a"
	}

	// Convert []byte to common.Hash for EIP-712 bytes32 encoding
	// crypto.Keccak256 always returns 32 bytes, which matches common.Hash size
	hash32 := common.BytesToHash(hash)

	return map[string]any{
		"source":       source,
		"connectionId": hash32,
	}
}

// L1Payload constructs the EIP-712 payload for L1 actions
func L1Payload(phantomAgent map[string]any) apitypes.TypedData {
	// Match Python SDK structure: Agent first, then EIP712Domain
	return apitypes.TypedData{
		Types: apitypes.Types{
			"Agent": []apitypes.Type{
				{Name: "source", Type: "string"},
				{Name: "connectionId", Type: "bytes32"},
			},
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		PrimaryType: "Agent",
		Domain: apitypes.TypedDataDomain{
			Name:              "Exchange",
			Version:           "1",
			ChainId:           (*math.HexOrDecimal256)(big.NewInt(1337)),
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Message: apitypes.TypedDataMessage(phantomAgent),
	}
}

// UserSignedPayload constructs the EIP-712 payload for user-signed actions
func UserSignedPayload(action map[string]any, signatureTypes []apitypes.Type, primaryType string) apitypes.TypedData {
	// Get chainId from action (signatureChainId is used for domain but not in message)
	chainIDHex, ok := action["signatureChainId"].(string)
	if !ok {
		chainIDHex = "0x66eee"
	}

	chainID := new(big.Int)
	chainID.SetString(chainIDHex[2:], 16)

	// Build message with only fields defined in signatureTypes
	// This excludes signatureChainId which is only used for domain
	message := make(apitypes.TypedDataMessage)
	for _, fieldType := range signatureTypes {
		if value, ok := action[fieldType.Name]; ok {
			// Convert to *big.Int for integer types (uint64, uint256, etc.)
			if strings.HasPrefix(fieldType.Type, "uint") || strings.HasPrefix(fieldType.Type, "int") {
				var bigIntVal *big.Int
				switch v := value.(type) {
				case int64:
					bigIntVal = big.NewInt(v)
				case uint64:
					bigIntVal = new(big.Int).SetUint64(v)
				case int:
					bigIntVal = big.NewInt(int64(v))
				case *big.Int:
					bigIntVal = v
				default:
					bigIntVal = nil
				}
				if bigIntVal != nil {
					message[fieldType.Name] = bigIntVal
				} else {
					message[fieldType.Name] = value
				}
			} else {
				message[fieldType.Name] = value
			}
		}
	}

	// Match Python SDK structure: primary type first, then EIP712Domain
	return apitypes.TypedData{
		Types: apitypes.Types{
			primaryType: signatureTypes,
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		PrimaryType: primaryType,
		Domain: apitypes.TypedDataDomain{
			Name:              "HyperliquidSignTransaction",
			Version:           "1",
			ChainId:           (*math.HexOrDecimal256)(chainID),
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Message: message,
	}
}

// SignL1Action signs an L1 action (orders, cancels, etc.)
func SignL1Action(
	privateKey *ecdsa.PrivateKey,
	action any,
	vaultAddress *string,
	nonce int64,
	expiresAfter *int64,
	isMainnet bool,
) (*types.Signature, error) {
	hash, err := ActionHash(action, vaultAddress, nonce, expiresAfter)
	if err != nil {
		return nil, err
	}

	phantomAgent := ConstructPhantomAgent(hash, isMainnet)
	typedData := L1Payload(phantomAgent)

	return signTypedData(privateKey, typedData)
}

// SignUserSignedAction signs a user-signed action (transfers, etc.)
func SignUserSignedAction(
	privateKey *ecdsa.PrivateKey,
	action map[string]any,
	signatureTypes []apitypes.Type,
	primaryType string,
	isMainnet bool,
) (*types.Signature, error) {
	// Set chainId and hyperliquidChain
	action["signatureChainId"] = "0x66eee"
	if isMainnet {
		action["hyperliquidChain"] = "Mainnet"
	} else {
		action["hyperliquidChain"] = "Testnet"
	}

	typedData := UserSignedPayload(action, signatureTypes, primaryType)
	return signTypedData(privateKey, typedData)
}

// SignMultiSigAction signs a multi-sig action
func SignMultiSigAction(
	privateKey *ecdsa.PrivateKey,
	action map[string]any,
	isMainnet bool,
	vaultAddress *string,
	nonce int64,
	expiresAfter *int64,
) (*types.Signature, error) {
	// Create a copy without the type field
	actionWithoutTag := make(map[string]any)
	for k, v := range action {
		if k != "type" {
			actionWithoutTag[k] = v
		}
	}

	// Compute action hash
	multiSigActionHashBytes, err := ActionHash(actionWithoutTag, vaultAddress, nonce, expiresAfter)
	if err != nil {
		return nil, fmt.Errorf("failed to compute multi-sig action hash: %w", err)
	}

	// Convert []byte to common.Hash for EIP-712 bytes32 encoding
	// crypto.Keccak256 always returns 32 bytes, which matches common.Hash size
	multiSigActionHash := common.BytesToHash(multiSigActionHashBytes)

	// Create envelope
	envelope := map[string]any{
		"multiSigActionHash": multiSigActionHash,
		"nonce":              nonce,
	}

	return SignUserSignedAction(
		privateKey,
		envelope,
		MultiSigEnvelopeSignTypes,
		"HyperliquidTransaction:SendMultiSig",
		isMainnet,
	)
}

// signTypedData signs EIP-712 typed data
func signTypedData(privateKey *ecdsa.PrivateKey, typedData apitypes.TypedData) (*types.Signature, error) {
	// Compute the typed data hash
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash message: %w", err)
	}

	// Construct the final hash: keccak256("\x19\x01" + domainSeparator + typedDataHash)
	rawData := []byte{0x19, 0x01}
	rawData = append(rawData, domainSeparator...)
	rawData = append(rawData, typedDataHash...)
	hash := crypto.Keccak256(rawData)

	// Sign the hash
	sig, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Extract r, s, v
	r := sig[:32]
	s := sig[32:64]
	v := int(sig[64])

	// Adjust v value (Ethereum uses 27/28 instead of 0/1)
	if v < 27 {
		v += 27
	}

	// Format R and S as hex strings, removing leading zeros to match Python SDK
	// Python SDK uses hex() which doesn't include leading zeros
	formatHex := func(b []byte) string {
		hexStr := common.Bytes2Hex(b)
		// Remove leading zeros but keep at least one digit
		trimmed := hexStr
		for len(trimmed) > 1 && trimmed[0] == '0' {
			trimmed = trimmed[1:]
		}
		return "0x" + trimmed
	}

	return &types.Signature{
		R: formatHex(r),
		S: formatHex(s),
		V: v,
	}, nil
}

// OrderTypeToWire converts OrderType to wire format
func OrderTypeToWire(orderType types.OrderType) (types.OrderTypeWire, error) {
	wire := types.OrderTypeWire{}

	if orderType.Limit != nil {
		wire.Limit = orderType.Limit
	} else if orderType.Trigger != nil {
		triggerPx, err := utils.FloatToWire(orderType.Trigger.TriggerPx)
		if err != nil {
			return wire, fmt.Errorf("invalid trigger price: %w", err)
		}
		wire.Trigger = &types.TriggerOrderTypeWire{
			TriggerPx: triggerPx,
			IsMarket:  orderType.Trigger.IsMarket,
			Tpsl:      orderType.Trigger.Tpsl,
		}
	} else {
		return wire, fmt.Errorf("invalid order type: must have either limit or trigger")
	}

	return wire, nil
}

// OrderRequestToOrderWire converts an OrderRequest to wire format
func OrderRequestToOrderWire(order types.OrderRequest, asset int) (types.OrderWire, error) {
	limitPx, err := utils.FloatToWire(order.LimitPx)
	if err != nil {
		return types.OrderWire{}, fmt.Errorf("invalid limit price: %w", err)
	}

	sz, err := utils.FloatToWire(order.Sz)
	if err != nil {
		return types.OrderWire{}, fmt.Errorf("invalid size: %w", err)
	}

	orderTypeWire, err := OrderTypeToWire(order.OrderType)
	if err != nil {
		return types.OrderWire{}, err
	}

	wire := types.OrderWire{
		Asset:      asset,
		IsBuy:      order.IsBuy,
		LimitPx:    limitPx,
		Sz:         sz,
		ReduceOnly: order.ReduceOnly,
		OrderType:  orderTypeWire,
	}

	if order.Cloid != nil {
		raw := order.Cloid.ToRaw()
		wire.Cloid = &raw
	}

	return wire, nil
}

// OrderWiresToOrderAction creates an order action from order wires
func OrderWiresToOrderAction(orderWires []types.OrderWire, builder *types.BuilderInfo) map[string]any {
	action := map[string]any{
		"type":     "order",
		"orders":   orderWires,
		"grouping": "na",
	}

	if builder != nil {
		action["builder"] = builder
	}

	return action
}

// Signature type definitions for user-signed actions
var (
	USDSendSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	SpotSendSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	SpotTransferSignTypes = SpotSendSignTypes // Alias for compatibility

	Withdraw3SignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	WithdrawSignTypes = Withdraw3SignTypes // Alias for compatibility

	USDClassTransferSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "toPerp", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	SendAssetSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "sourceDex", Type: "string"},
		{Name: "destinationDex", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "fromSubAccount", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	TokenDelegateSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "validator", Type: "address"},
		{Name: "wei", Type: "uint64"},
		{Name: "isUndelegate", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	ApproveAgentSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "agentAddress", Type: "address"},
		{Name: "agentName", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	AgentSignTypes = ApproveAgentSignTypes // Alias for compatibility

	ApproveBuilderFeeSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "maxFeeRate", Type: "string"},
		{Name: "builder", Type: "address"},
		{Name: "nonce", Type: "uint64"},
	}

	UserDexAbstractionSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "user", Type: "address"},
		{Name: "enabled", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	ConvertToMultiSigUserSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "signers", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	MultiSigEnvelopeSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "multiSigActionHash", Type: "bytes32"},
		{Name: "nonce", Type: "uint64"},
	}
)

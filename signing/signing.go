// Package signing provides EIP-712 signing functionality for Hyperliquid API requests.
package signing

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math/big"

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

	return map[string]any{
		"source":       source,
		"connectionId": hash,
	}
}

// L1Payload constructs the EIP-712 payload for L1 actions
func L1Payload(phantomAgent map[string]any) apitypes.TypedData {
	return apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Agent": []apitypes.Type{
				{Name: "source", Type: "string"},
				{Name: "connectionId", Type: "bytes32"},
			},
		},
		PrimaryType: "Agent",
		Domain: apitypes.TypedDataDomain{
			Name:              "Exchange",
			Version:           "1",
			ChainId:           (*math.HexOrDecimal256)(big.NewInt(1337)),
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Message: apitypes.TypedDataMessage{
			"source":       phantomAgent["source"],
			"connectionId": phantomAgent["connectionId"],
		},
	}
}

// UserSignedPayload constructs the EIP-712 payload for user-signed actions
func UserSignedPayload(action map[string]any, signatureTypes []apitypes.Type, primaryType string) apitypes.TypedData {
	// Get chainId from action
	chainIDHex, ok := action["signatureChainId"].(string)
	if !ok {
		chainIDHex = "0x66eee"
	}

	chainID := new(big.Int)
	chainID.SetString(chainIDHex[2:], 16)

	return apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			primaryType: signatureTypes,
		},
		PrimaryType: primaryType,
		Domain: apitypes.TypedDataDomain{
			Name:              "HyperliquidSignTransaction",
			Version:           "1",
			ChainId:           (*math.HexOrDecimal256)(chainID),
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Message: apitypes.TypedDataMessage(action),
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

	return &types.Signature{
		R: "0x" + common.Bytes2Hex(r),
		S: "0x" + common.Bytes2Hex(s),
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

	SpotTransferSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	WithdrawSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	USDClassTransferSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "toPerp", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	AgentSignTypes = []apitypes.Type{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "agentAddress", Type: "address"},
		{Name: "agentName", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}
)

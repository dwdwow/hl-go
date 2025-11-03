package signing

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dwdwow/hl-go/types"
	"github.com/dwdwow/hl-go/utils"
)

// Test private key: 0x0123456789012345678901234567890123456789012345678901234567890123
var testPrivateKeyHex = "0123456789012345678901234567890123456789012345678901234567890123"

func getTestPrivateKey(t *testing.T) *ecdsa.PrivateKey {
	privateKey, err := crypto.HexToECDSA(testPrivateKeyHex)
	if err != nil {
		t.Fatalf("Failed to load test private key: %v", err)
	}
	return privateKey
}

func TestPhantomAgentCreationMatchesProduction(t *testing.T) {
	timestamp := int64(1677777606040)
	orderRequest := types.OrderRequest{
		Coin:       "ETH",
		IsBuy:      true,
		Sz:         0.0147,
		LimitPx:    1670.1,
		ReduceOnly: false,
		OrderType: types.OrderType{
			Limit: &types.LimitOrderType{
				Tif: types.TifIoc,
			},
		},
		Cloid: nil,
	}

	orderWire, err := OrderRequestToOrderWire(orderRequest, 4)
	if err != nil {
		t.Fatalf("OrderRequestToOrderWire() error = %v", err)
	}

	orderAction := OrderWiresToOrderAction([]types.OrderWire{orderWire}, nil)
	hash, err := ActionHash(orderAction, nil, timestamp, nil)
	if err != nil {
		t.Fatalf("ActionHash() error = %v", err)
	}

	phantomAgent := ConstructPhantomAgent(hash, true)
	connectionID, ok := phantomAgent["connectionId"].(common.Hash)
	if !ok {
		t.Fatal("connectionId should be common.Hash")
	}

	expected := common.HexToHash("0x0fcbeda5ae3c4950a548021552a4fea2226858c4453571bf3f24ba017eac2908")
	if connectionID != expected {
		t.Errorf("Phantom agent connectionId = %s, want %s", connectionID.Hex(), expected.Hex())
	}
}

func TestL1ActionSigningMatches(t *testing.T) {
	privateKey := getTestPrivateKey(t)

	// Convert float to int for hashing (1000 * 10^8)
	num, err := utils.FloatToIntForHashing(1000)
	if err != nil {
		t.Fatalf("FloatToIntForHashing() error = %v", err)
	}

	// Python msgpack encodes positive integers as uint64, so convert to match
	// Python creates: {"type": "dummy", "num": ...} - ensure key order matches
	action := NewOrderedMap(
		"type", "dummy",
		"num", uint64(num),
	)

	// Test mainnet
	signatureMainnet, err := SignL1Action(privateKey, action, nil, 0, nil, true)
	if err != nil {
		t.Fatalf("SignL1Action(mainnet) error = %v", err)
	}
	if signatureMainnet.R != "0x53749d5b30552aeb2fca34b530185976545bb22d0b3ce6f62e31be961a59298" {
		t.Errorf("SignatureMainnet.R = %s, want 0x53749d5b30552aeb2fca34b530185976545bb22d0b3ce6f62e31be961a59298", signatureMainnet.R)
	}
	if signatureMainnet.S != "0x755c40ba9bf05223521753995abb2f73ab3229be8ec921f350cb447e384d8ed8" {
		t.Errorf("SignatureMainnet.S = %s, want 0x755c40ba9bf05223521753995abb2f73ab3229be8ec921f350cb447e384d8ed8", signatureMainnet.S)
	}
	if signatureMainnet.V != 27 {
		t.Errorf("SignatureMainnet.V = %d, want 27", signatureMainnet.V)
	}

	// Test testnet
	signatureTestnet, err := SignL1Action(privateKey, action, nil, 0, nil, false)
	if err != nil {
		t.Fatalf("SignL1Action(testnet) error = %v", err)
	}
	if signatureTestnet.R != "0x542af61ef1f429707e3c76c5293c80d01f74ef853e34b76efffcb57e574f9510" {
		t.Errorf("SignatureTestnet.R = %s, want 0x542af61ef1f429707e3c76c5293c80d01f74ef853e34b76efffcb57e574f9510", signatureTestnet.R)
	}
	if signatureTestnet.S != "0x17b8b32f086e8cdede991f1e2c529f5dd5297cbe8128500e00cbaf766204a613" {
		t.Errorf("SignatureTestnet.S = %s, want 0x17b8b32f086e8cdede991f1e2c529f5dd5297cbe8128500e00cbaf766204a613", signatureTestnet.S)
	}
	if signatureTestnet.V != 28 {
		t.Errorf("SignatureTestnet.V = %d, want 28", signatureTestnet.V)
	}
}

func TestL1ActionSigningOrderMatches(t *testing.T) {
	privateKey := getTestPrivateKey(t)

	orderRequest := types.OrderRequest{
		Coin:       "ETH",
		IsBuy:      true,
		Sz:         100,
		LimitPx:    100,
		ReduceOnly: false,
		OrderType: types.OrderType{
			Limit: &types.LimitOrderType{
				Tif: types.TifGtc,
			},
		},
		Cloid: nil,
	}

	orderWire, err := OrderRequestToOrderWire(orderRequest, 1)
	if err != nil {
		t.Fatalf("OrderRequestToOrderWire() error = %v", err)
	}

	orderAction := OrderWiresToOrderAction([]types.OrderWire{orderWire}, nil)
	timestamp := int64(0)

	// Test mainnet
	signatureMainnet, err := SignL1Action(privateKey, orderAction, nil, timestamp, nil, true)
	if err != nil {
		t.Fatalf("SignL1Action(mainnet) error = %v", err)
	}
	if signatureMainnet.R != "0xd65369825a9df5d80099e513cce430311d7d26ddf477f5b3a33d2806b100d78e" {
		t.Errorf("SignatureMainnet.R = %s, want 0xd65369825a9df5d80099e513cce430311d7d26ddf477f5b3a33d2806b100d78e", signatureMainnet.R)
	}
	if signatureMainnet.S != "0x2b54116ff64054968aa237c20ca9ff68000f977c93289157748a3162b6ea940e" {
		t.Errorf("SignatureMainnet.S = %s, want 0x2b54116ff64054968aa237c20ca9ff68000f977c93289157748a3162b6ea940e", signatureMainnet.S)
	}
	if signatureMainnet.V != 28 {
		t.Errorf("SignatureMainnet.V = %d, want 28", signatureMainnet.V)
	}

	// Test testnet
	signatureTestnet, err := SignL1Action(privateKey, orderAction, nil, timestamp, nil, false)
	if err != nil {
		t.Fatalf("SignL1Action(testnet) error = %v", err)
	}
	if signatureTestnet.R != "0x82b2ba28e76b3d761093aaded1b1cdad4960b3af30212b343fb2e6cdfa4e3d54" {
		t.Errorf("SignatureTestnet.R = %s, want 0x82b2ba28e76b3d761093aaded1b1cdad4960b3af30212b343fb2e6cdfa4e3d54", signatureTestnet.R)
	}
	if signatureTestnet.S != "0x6b53878fc99d26047f4d7e8c90eb98955a109f44209163f52d8dc4278cbbd9f5" {
		t.Errorf("SignatureTestnet.S = %s, want 0x6b53878fc99d26047f4d7e8c90eb98955a109f44209163f52d8dc4278cbbd9f5", signatureTestnet.S)
	}
	if signatureTestnet.V != 27 {
		t.Errorf("SignatureTestnet.V = %d, want 27", signatureTestnet.V)
	}
}

func TestL1ActionSigningOrderWithCloidMatches(t *testing.T) {
	privateKey := getTestPrivateKey(t)

	cloid, err := types.NewCloidFromString("0x00000000000000000000000000000001")
	if err != nil {
		t.Fatalf("NewCloidFromString() error = %v", err)
	}

	orderRequest := types.OrderRequest{
		Coin:       "ETH",
		IsBuy:      true,
		Sz:         100,
		LimitPx:    100,
		ReduceOnly: false,
		OrderType: types.OrderType{
			Limit: &types.LimitOrderType{
				Tif: types.TifGtc,
			},
		},
		Cloid: cloid, // cloid is already a pointer (*Cloid)
	}

	orderWire, err := OrderRequestToOrderWire(orderRequest, 1)
	if err != nil {
		t.Fatalf("OrderRequestToOrderWire() error = %v", err)
	}

	orderAction := OrderWiresToOrderAction([]types.OrderWire{orderWire}, nil)
	timestamp := int64(0)

	// Test mainnet
	signatureMainnet, err := SignL1Action(privateKey, orderAction, nil, timestamp, nil, true)
	if err != nil {
		t.Fatalf("SignL1Action(mainnet) error = %v", err)
	}
	if signatureMainnet.R != "0x41ae18e8239a56cacbc5dad94d45d0b747e5da11ad564077fcac71277a946e3" {
		t.Errorf("SignatureMainnet.R = %s, want 0x41ae18e8239a56cacbc5dad94d45d0b747e5da11ad564077fcac71277a946e3", signatureMainnet.R)
	}
	if signatureMainnet.S != "0x3c61f667e747404fe7eea8f90ab0e76cc12ce60270438b2058324681a00116da" {
		t.Errorf("SignatureMainnet.S = %s, want 0x3c61f667e747404fe7eea8f90ab0e76cc12ce60270438b2058324681a00116da", signatureMainnet.S)
	}
	if signatureMainnet.V != 27 {
		t.Errorf("SignatureMainnet.V = %d, want 27", signatureMainnet.V)
	}

	// Test testnet
	signatureTestnet, err := SignL1Action(privateKey, orderAction, nil, timestamp, nil, false)
	if err != nil {
		t.Fatalf("SignL1Action(testnet) error = %v", err)
	}
	if signatureTestnet.R != "0xeba0664bed2676fc4e5a743bf89e5c7501aa6d870bdb9446e122c9466c5cd16d" {
		t.Errorf("SignatureTestnet.R = %s, want 0xeba0664bed2676fc4e5a743bf89e5c7501aa6d870bdb9446e122c9466c5cd16d", signatureTestnet.R)
	}
	if signatureTestnet.S != "0x7f3e74825c9114bc59086f1eebea2928c190fdfbfde144827cb02b85bbe90988" {
		t.Errorf("SignatureTestnet.S = %s, want 0x7f3e74825c9114bc59086f1eebea2928c190fdfbfde144827cb02b85bbe90988", signatureTestnet.S)
	}
	if signatureTestnet.V != 28 {
		t.Errorf("SignatureTestnet.V = %d, want 28", signatureTestnet.V)
	}
}

func TestL1ActionSigningMatchesWithVault(t *testing.T) {
	privateKey := getTestPrivateKey(t)

	num, err := utils.FloatToIntForHashing(1000)
	if err != nil {
		t.Fatalf("FloatToIntForHashing() error = %v", err)
	}

	// Python msgpack encodes positive integers as uint64, so convert to match
	// Python creates: {"type": "dummy", "num": ...} - ensure key order matches
	action := NewOrderedMap(
		"type", "dummy",
		"num", uint64(num),
	)

	vaultAddress := "0x1719884eb866cb12b2287399b15f7db5e7d775ea"

	// Test mainnet
	signatureMainnet, err := SignL1Action(privateKey, action, &vaultAddress, 0, nil, true)
	if err != nil {
		t.Fatalf("SignL1Action(mainnet) error = %v", err)
	}
	if signatureMainnet.R != "0x3c548db75e479f8012acf3000ca3a6b05606bc2ec0c29c50c515066a326239" {
		t.Errorf("SignatureMainnet.R = %s, want 0x3c548db75e479f8012acf3000ca3a6b05606bc2ec0c29c50c515066a326239", signatureMainnet.R)
	}
	if signatureMainnet.S != "0x4d402be7396ce74fbba3795769cda45aec00dc3125a984f2a9f23177b190da2c" {
		t.Errorf("SignatureMainnet.S = %s, want 0x4d402be7396ce74fbba3795769cda45aec00dc3125a984f2a9f23177b190da2c", signatureMainnet.S)
	}
	if signatureMainnet.V != 28 {
		t.Errorf("SignatureMainnet.V = %d, want 28", signatureMainnet.V)
	}

	// Test testnet
	signatureTestnet, err := SignL1Action(privateKey, action, &vaultAddress, 0, nil, false)
	if err != nil {
		t.Fatalf("SignL1Action(testnet) error = %v", err)
	}
	if signatureTestnet.R != "0xe281d2fb5c6e25ca01601f878e4d69c965bb598b88fac58e475dd1f5e56c362b" {
		t.Errorf("SignatureTestnet.R = %s, want 0xe281d2fb5c6e25ca01601f878e4d69c965bb598b88fac58e475dd1f5e56c362b", signatureTestnet.R)
	}
	if signatureTestnet.S != "0x7ddad27e9a238d045c035bc606349d075d5c5cd00a6cd1da23ab5c39d4ef0f60" {
		t.Errorf("SignatureTestnet.S = %s, want 0x7ddad27e9a238d045c035bc606349d075d5c5cd00a6cd1da23ab5c39d4ef0f60", signatureTestnet.S)
	}
	if signatureTestnet.V != 27 {
		t.Errorf("SignatureTestnet.V = %d, want 27", signatureTestnet.V)
	}
}

func TestSignUsdTransferAction(t *testing.T) {
	privateKey := getTestPrivateKey(t)

	message := map[string]any{
		"destination": "0x5e9ee1089755c3435139848e47e6635505d5a13a",
		"amount":      "1",
		"time":        int64(1687816341423),
	}

	signature, err := SignUserSignedAction(
		privateKey,
		message,
		USDSendSignTypes,
		"HyperliquidTransaction:UsdSend",
		false,
	)
	if err != nil {
		t.Fatalf("SignUserSignedAction() error = %v", err)
	}
	if signature.R != "0x637b37dd731507cdd24f46532ca8ba6eec616952c56218baeff04144e4a77073" {
		t.Errorf("Signature.R = %s, want 0x637b37dd731507cdd24f46532ca8ba6eec616952c56218baeff04144e4a77073", signature.R)
	}
	if signature.S != "0x11a6a24900e6e314136d2592e2f8d502cd89b7c15b198e1bee043c9589f9fad7" {
		t.Errorf("Signature.S = %s, want 0x11a6a24900e6e314136d2592e2f8d502cd89b7c15b198e1bee043c9589f9fad7", signature.S)
	}
	if signature.V != 27 {
		t.Errorf("Signature.V = %d, want 27", signature.V)
	}
}

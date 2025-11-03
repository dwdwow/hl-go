package signing

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/dwdwow/hl-go/utils"
)

// trimLeadingZeros removes leading zeros from hex string (but keeps at least one digit)
func trimLeadingZeros(hexStr string) string {
	if !strings.HasPrefix(hexStr, "0x") {
		return hexStr
	}
	hex := hexStr[2:]
	// Remove leading zeros but keep at least one
	trimmed := strings.TrimLeft(hex, "0")
	if trimmed == "" {
		return "0x0"
	}
	return "0x" + trimmed
}

// TestEIP712StepsForSimpleAction 逐步验证 EIP-712 签名的每个步骤
func TestEIP712StepsForSimpleAction(t *testing.T) {
	privateKey, _ := crypto.HexToECDSA(testPrivateKeyHex)

	// Test case: {"type": "dummy", "num": 100000000000}
	num, _ := utils.FloatToIntForHashing(1000)
	action := map[string]any{
		"type": "dummy",
		"num":  uint64(num),
	}

	// Step 1: Compute ActionHash
	hash, err := ActionHash(action, nil, 0, nil)
	if err != nil {
		t.Fatalf("ActionHash() error = %v", err)
	}
	t.Logf("Step 1 - ActionHash: 0x%x", hash)

	// Step 2: Construct Phantom Agent
	phantomAgent := ConstructPhantomAgent(hash, true)
	t.Logf("Step 2 - PhantomAgent: %+v", phantomAgent)
	if connID, ok := phantomAgent["connectionId"].(common.Hash); ok {
		t.Logf("  connectionId: %s", connID.Hex())
	}
	if source, ok := phantomAgent["source"].(string); ok {
		t.Logf("  source: %s", source)
	}

	// Step 3: Build L1Payload (TypedData)
	typedData := L1Payload(phantomAgent)
	t.Logf("Step 3 - TypedData:")
	t.Logf("  PrimaryType: %s", typedData.PrimaryType)
	t.Logf("  Domain.Name: %s", typedData.Domain.Name)
	t.Logf("  Domain.Version: %s", typedData.Domain.Version)
	t.Logf("  Domain.ChainId: %v", typedData.Domain.ChainId)
	t.Logf("  Domain.VerifyingContract: %s", typedData.Domain.VerifyingContract)
	t.Logf("  Message: %+v", typedData.Message)

	// Step 4: Compute Domain Separator
	domainMap := typedData.Domain.Map()
	domainSeparator, err := typedData.HashStruct("EIP712Domain", domainMap)
	if err != nil {
		t.Fatalf("HashStruct(EIP712Domain) error = %v", err)
	}
	t.Logf("Step 4 - Domain Separator: 0x%x", domainSeparator)

	// Step 5: Compute Message Hash
	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		t.Fatalf("HashStruct(%s) error = %v", typedData.PrimaryType, err)
	}
	t.Logf("Step 5 - Message Hash: 0x%x", messageHash)

	// Step 6: Compute Final Hash
	// EIP-712: keccak256("\x19\x01" || domainSeparator || messageHash)
	rawData := []byte{0x19, 0x01}
	rawData = append(rawData, domainSeparator...)
	rawData = append(rawData, messageHash...)
	finalHash := crypto.Keccak256(rawData)
	t.Logf("Step 6 - Final Hash (to be signed): 0x%x", finalHash)

	// Step 7: Sign
	sig, err := crypto.Sign(finalHash, privateKey)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	// Step 8: Extract r, s, v
	r := sig[:32]
	s := sig[32:64]
	v := int(sig[64])
	if v < 27 {
		v += 27
	}

	t.Logf("Step 7&8 - Signature:")
	t.Logf("  R: 0x%x", r)
	t.Logf("  S: 0x%x", s)
	t.Logf("  V: %d", v)

	// Expected from Python test (hex strings without leading zeros)
	expectedRHex := "0x53749d5b30552aeb2fca34b530185976545bb22d0b3ce6f62e31be961a59298"
	expectedSHex := "0x755c40ba9bf05223521753995abb2f73ab3229be8ec921f350cb447e384d8ed8"
	expectedV := 27

	// Convert actual bytes to hex (removing leading zeros for comparison)
	actualRHex := trimLeadingZeros(fmt.Sprintf("0x%x", r))
	actualSHex := trimLeadingZeros(fmt.Sprintf("0x%x", s))

	t.Logf("Expected R: %s", expectedRHex)
	t.Logf("Actual R:   %s", actualRHex)
	t.Logf("Expected S: %s", expectedSHex)
	t.Logf("Actual S:   %s", actualSHex)
	t.Logf("Expected V: %d", expectedV)
	t.Logf("Actual V:   %d", v)

	if actualRHex != expectedRHex {
		t.Errorf("R mismatch: got %s, want %s", actualRHex, expectedRHex)
		// Show byte-by-byte comparison
		expectedRBytes := common.FromHex(expectedRHex)
		if len(expectedRBytes) < 32 {
			expectedRBytes = append(make([]byte, 32-len(expectedRBytes)), expectedRBytes...)
		}
		for i := 0; i < 32; i++ {
			if r[i] != expectedRBytes[i] {
				t.Errorf("  Byte %d: got 0x%02x, want 0x%02x", i, r[i], expectedRBytes[i])
			}
		}
	}
	if actualSHex != expectedSHex {
		t.Errorf("S mismatch: got %s, want %s", actualSHex, expectedSHex)
	}
	if v != expectedV {
		t.Errorf("V mismatch: got %d, want %d", v, expectedV)
	}

	if actualRHex == expectedRHex && actualSHex == expectedSHex && v == expectedV {
		t.Logf("✓ All signature components match Python SDK!")
	}
}

// TestEIP712TypesAndEncoding 验证 EIP-712 类型定义和编码
func TestEIP712TypesAndEncoding(t *testing.T) {
	num, _ := utils.FloatToIntForHashing(1000)
	action := map[string]any{
		"type": "dummy",
		"num":  uint64(num),
	}

	hash, _ := ActionHash(action, nil, 0, nil)
	phantomAgent := ConstructPhantomAgent(hash, true)
	typedData := L1Payload(phantomAgent)

	// Log Types structure
	t.Logf("Types structure:")
	for typeName, types := range typedData.Types {
		t.Logf("  %s:", typeName)
		for _, field := range types {
			t.Logf("    %s: %s", field.Name, field.Type)
		}
	}

	// Verify Types order - should be Agent first, then EIP712Domain
	typeNames := make([]string, 0, len(typedData.Types))
	for name := range typedData.Types {
		typeNames = append(typeNames, name)
	}
	t.Logf("Types order: %v", typeNames)

	// Encode Type for Agent
	encodedType := typedData.EncodeType("Agent")
	t.Logf("Encoded Type for Agent: %s", encodedType)
	t.Logf("Encoded Type (keccak256): 0x%x", crypto.Keccak256([]byte(encodedType)))

	// Encode Type for EIP712Domain
	encodedDomainType := typedData.EncodeType("EIP712Domain")
	t.Logf("Encoded Type for EIP712Domain: %s", encodedDomainType)
	t.Logf("Encoded Type (keccak256): 0x%x", crypto.Keccak256([]byte(encodedDomainType)))

	// Note: EncodeData is internal, we use HashStruct instead which calls EncodeData internally
	t.Logf("Using HashStruct to encode Agent message")

	// Verify message structure
	if source, ok := typedData.Message["source"].(string); ok {
		t.Logf("Message source: %s", source)
	}
	if connID, ok := typedData.Message["connectionId"]; ok {
		t.Logf("Message connectionId type: %T", connID)
		if hash, ok := connID.(common.Hash); ok {
			t.Logf("Message connectionId: %s", hash.Hex())
		}
	}
}



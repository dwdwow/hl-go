package signing

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/dwdwow/hl-go/types"
	"github.com/dwdwow/hl-go/utils"
)

// TestMsgpackEncodingExactMatch 测试 msgpack 编码是否与 Python 完全一致
func TestMsgpackEncodingExactMatch(t *testing.T) {
	// Test case 1: Simple action {"type": "dummy", "num": 100000000000}
	num, _ := utils.FloatToIntForHashing(1000)

	t.Run("SimpleAction", func(t *testing.T) {
		// Python expected: 82a474797065a564756d6d79a36e756dcf000000174876e800
		// Key order in Python: "type" first, then "num"
		
		action := map[string]any{
			"type": "dummy",
			"num":  uint64(num), // Use uint64 to match Python
		}

		data, err := msgpack.Marshal(action)
		if err != nil {
			t.Fatalf("msgpack.Marshal() error = %v", err)
		}

		expectedHex := "82a474797065a564756d6d79a36e756dcf000000174876e800"
		actualHex := fmt.Sprintf("%x", data)

		t.Logf("Expected (Python): %s", expectedHex)
		t.Logf("Actual (Go):      %s", actualHex)
		t.Logf("Match: %v", actualHex == expectedHex)

		if actualHex != expectedHex {
			// Show byte-by-byte comparison
			expectedBytes, _ := hexToBytes(expectedHex)
			t.Logf("Expected bytes: %x", expectedBytes)
			t.Logf("Actual bytes:   %x", data)
			t.Logf("Length match: expected=%d, actual=%d", len(expectedBytes), len(data))
			
			// Find first difference
			for i := 0; i < len(data) && i < len(expectedBytes); i++ {
				if data[i] != expectedBytes[i] {
					t.Errorf("First difference at byte %d: expected=0x%02x, actual=0x%02x", i, expectedBytes[i], data[i])
					break
				}
			}
		} else {
			t.Logf("✓ msgpack encoding matches Python!")
		}
	})
}

// TestActionHashExactMatch 测试 ActionHash 是否与 Python 完全一致
func TestActionHashExactMatch(t *testing.T) {
	t.Run("SimpleAction", func(t *testing.T) {
		num, _ := utils.FloatToIntForHashing(1000)
		action := map[string]any{
			"type": "dummy",
			"num":  uint64(num),
		}

		// Step 1: Verify msgpack encoding
		msgpackData, err := msgpack.Marshal(action)
		if err != nil {
			t.Fatalf("msgpack.Marshal() error = %v", err)
		}
		
		expectedMsgpack := "82a474797065a564756d6d79a36e756dcf000000174876e800"
		if fmt.Sprintf("%x", msgpackData) != expectedMsgpack {
			t.Fatalf("msgpack encoding doesn't match. This must be fixed first!")
		}
		t.Logf("✓ Step 1: msgpack encoding matches")

		// Step 2: Append nonce (8 bytes, big endian, 0)
		nonceBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(nonceBytes, 0)
		fullData := append(msgpackData, nonceBytes...)
		t.Logf("After nonce (hex): %x", fullData)
		t.Logf("After nonce (length): %d", len(fullData))

		// Step 3: Append vault address marker (0x00 for nil)
		fullData = append(fullData, 0x00)
		t.Logf("After vault marker (hex): %x", fullData)
		t.Logf("After vault marker (length): %d", len(fullData))

		// Step 4: Compute hash
		hash := crypto.Keccak256(fullData)
		hashHex := fmt.Sprintf("%x", hash)
		t.Logf("ActionHash: %s", hashHex)
		t.Logf("ActionHash (0x): 0x%s", hashHex)

		// This is what we compute - need to verify if Python computes the same
		// The hash should be used in phantom agent's connectionId
	})

	t.Run("UsingActionHashFunction", func(t *testing.T) {
		num, _ := utils.FloatToIntForHashing(1000)
		action := map[string]any{
			"type": "dummy",
			"num":  uint64(num),
		}

		hash, err := ActionHash(action, nil, 0, nil)
		if err != nil {
			t.Fatalf("ActionHash() error = %v", err)
		}

		hashHex := fmt.Sprintf("%x", hash)
		t.Logf("ActionHash result: 0x%s", hashHex)
		t.Logf("ActionHash length: %d bytes", len(hash))
		
		// Verify it's 32 bytes
		if len(hash) != 32 {
			t.Errorf("ActionHash length = %d, want 32", len(hash))
		}
	})
}

// TestPhantomAgentHash 测试 phantom agent 的 connectionId hash 是否匹配
func TestPhantomAgentHash(t *testing.T) {
	// This test case from Python SDK
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

	// Step 1: Verify msgpack encoding of orderAction
	msgpackData, err := msgpack.Marshal(orderAction)
	if err != nil {
		t.Fatalf("msgpack.Marshal() error = %v", err)
	}
	t.Logf("OrderAction msgpack (hex): %x", msgpackData)
	t.Logf("OrderAction msgpack (length): %d", len(msgpackData))

	// Step 2: Compute ActionHash
	hash, err := ActionHash(orderAction, nil, timestamp, nil)
	if err != nil {
		t.Fatalf("ActionHash() error = %v", err)
	}
	t.Logf("ActionHash: 0x%x", hash)

	// Step 3: Expected hash from Python test
	expectedHash := common.HexToHash("0x0fcbeda5ae3c4950a548021552a4fea2226858c4453571bf3f24ba017eac2908")
	actualHash := common.BytesToHash(hash)

	t.Logf("Expected hash: %s", expectedHash.Hex())
	t.Logf("Actual hash:   %s", actualHash.Hex())
	t.Logf("Match: %v", actualHash == expectedHash)

	if actualHash != expectedHash {
		t.Errorf("ActionHash mismatch!")
		// Show byte-by-byte difference
		for i := 0; i < 32; i++ {
			if hash[i] != expectedHash[i] {
				t.Errorf("Byte %d: expected=0x%02x, actual=0x%02x", i, expectedHash[i], hash[i])
			}
		}
	} else {
		t.Logf("✓ ActionHash matches Python SDK!")
	}
}

// TestSimpleActionHashForPythonTest 验证 Python 测试用例 test_l1_action_signing_matches 使用的 ActionHash
// Python test: action = {"type": "dummy", "num": float_to_int_for_hashing(1000)}
// nonce=0, vault=None
func TestSimpleActionHashForPythonTest(t *testing.T) {
	num, _ := utils.FloatToIntForHashing(1000)
	action := map[string]any{
		"type": "dummy",
		"num":  uint64(num),
	}

	hash, err := ActionHash(action, nil, 0, nil)
	if err != nil {
		t.Fatalf("ActionHash() error = %v", err)
	}

	t.Logf("ActionHash for Python test case: 0x%x", hash)
	t.Logf("This hash will be used as connectionId in phantom agent")
	
	// Now verify phantom agent construction
	phantomAgent := ConstructPhantomAgent(hash, true)
	t.Logf("PhantomAgent: %+v", phantomAgent)
	if connID, ok := phantomAgent["connectionId"].(common.Hash); ok {
		t.Logf("connectionId: %s", connID.Hex())
	}
}

// Helper function to convert hex string to bytes
func hexToBytes(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}

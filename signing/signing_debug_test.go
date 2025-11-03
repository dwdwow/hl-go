package signing

import (
	"testing"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/dwdwow/hl-go/utils"
)

func TestDebugMsgpackIntEncoding(t *testing.T) {
	val := int64(100000000000)

	// Test 1: int64
	action1 := map[string]any{
		"type": "dummy",
		"num":  val,
	}
	data1, _ := msgpack.Marshal(action1)
	t.Logf("int64 msgpack: %x", data1)

	// Test 2: uint64
	action2 := map[string]any{
		"type": "dummy",
		"num":  uint64(val),
	}
	data2, _ := msgpack.Marshal(action2)
	t.Logf("uint64 msgpack: %x", data2)

	// Test 3: Check which matches Python
	pythonExpected := "82a474797065a564756d6d79a36e756dcf000000174876e800"
	t.Logf("Python expected: %s", pythonExpected)
	t.Logf("Go int64 hex: %x", data1)
	t.Logf("Go uint64 hex: %x", data2)
}

func TestDebugActionHashWithUint64(t *testing.T) {
	// Test using uint64 instead of int64
	num, _ := utils.FloatToIntForHashing(1000)
	action := map[string]any{
		"type": "dummy",
		"num":  uint64(num), // Convert to uint64
	}

	hash, err := ActionHash(action, nil, 0, nil)
	if err != nil {
		t.Fatalf("ActionHash() error = %v", err)
	}
	t.Logf("ActionHash with uint64: 0x%x", hash)
}

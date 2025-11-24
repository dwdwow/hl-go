package ws

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestWebSocket_Trades(t *testing.T) {
	client := NewTradesClient("HYPE")

	for {
		trades, err := client.Read()
		if err != nil {
			t.Fatalf("failed to read trades: %v", err)
		}

		s, _ := json.MarshalIndent(trades, "", "  ")
		fmt.Println(string(s))
	}
}

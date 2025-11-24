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

func TestWebSocket_L2Book(t *testing.T) {
	client := NewL2BookClient("HYPE")
	
	book, err := client.Read()
	if err != nil {
		t.Fatal(err)
	}
	
	lastTime := book.Time

	for {
		book, err := client.Read()
		if err != nil {
			t.Fatalf("failed to read book: %v", err)
		}
		
		fmt.Println(book.Time - lastTime, book.Time)
		
		lastTime = book.Time

		// s, _ := json.MarshalIndent(book, "", "  ")
		// fmt.Println(string(s))
	}
}

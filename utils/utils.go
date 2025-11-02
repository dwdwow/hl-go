// Package utils provides utility functions for the Hyperliquid SDK.
package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// FloatToWire converts a float to a string representation suitable for the API.
// It rounds to 8 decimal places and normalizes the output (removes trailing zeros).
func FloatToWire(x float64) (string, error) {
	// Round to 8 decimal places
	rounded := fmt.Sprintf("%.8f", x)

	// Check if rounding caused significant change
	parsedBack, err := strconv.ParseFloat(rounded, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse rounded value: %w", err)
	}

	if math.Abs(parsedBack-x) >= 1e-12 {
		return "", fmt.Errorf("float_to_wire causes rounding: %f", x)
	}

	// Handle -0 case
	if rounded == "-0.00000000" {
		rounded = "0.00000000"
	}

	// Normalize: remove trailing zeros and decimal point if not needed
	normalized := strings.TrimRight(rounded, "0")
	normalized = strings.TrimRight(normalized, ".")

	return normalized, nil
}

// FloatToIntForHashing converts a float to an integer for hashing (8 decimals)
func FloatToIntForHashing(x float64) (int64, error) {
	return FloatToInt(x, 8)
}

// FloatToUsdInt converts a float to a USD integer (6 decimals)
func FloatToUsdInt(x float64) (int64, error) {
	return FloatToInt(x, 6)
}

// FloatToInt converts a float to an integer with specified decimal places
func FloatToInt(x float64, power int) (int64, error) {
	multiplier := math.Pow(10, float64(power))
	withDecimals := x * multiplier

	// Check if rounding would occur
	if math.Abs(math.Round(withDecimals)-withDecimals) >= 1e-3 {
		return 0, fmt.Errorf("float_to_int causes rounding: %f", x)
	}

	return int64(math.Round(withDecimals)), nil
}

// GetTimestampMs returns the current timestamp in milliseconds
func GetTimestampMs() int64 {
	return time.Now().UnixMilli()
}

// RoundPrice rounds a price to the specified number of significant figures and decimals
func RoundPrice(px float64, sigFigs int, decimals int) float64 {
	// Round to significant figures
	if px == 0 {
		return 0
	}

	// Calculate the power of 10 for significant figures
	magnitude := math.Floor(math.Log10(math.Abs(px)))
	power := float64(sigFigs-1) - magnitude
	multiplier := math.Pow(10, power)

	rounded := math.Round(px*multiplier) / multiplier

	// Then round to decimals
	decimalMultiplier := math.Pow(10, float64(decimals))
	rounded = math.Round(rounded*decimalMultiplier) / decimalMultiplier

	return rounded
}

// FormatFloat formats a float with up to 8 decimal places, removing trailing zeros
func FormatFloat(f float64) string {
	s := fmt.Sprintf("%.8f", f)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// AddressToBytes converts a hex address string to bytes
func AddressToBytes(address string) ([]byte, error) {
	// Remove 0x prefix if present
	// if strings.HasPrefix(address, "0x") {
	// 	address = address[2:]
	// }
	address = strings.TrimPrefix(address, "0x")

	// Decode hex string
	bytes := make([]byte, len(address)/2)
	for i := 0; i < len(bytes); i++ {
		b, err := strconv.ParseUint(address[i*2:i*2+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex address: %w", err)
		}
		bytes[i] = byte(b)
	}

	return bytes, nil
}

// BytesToHex converts bytes to a hex string with 0x prefix
func BytesToHex(b []byte) string {
	hex := make([]byte, len(b)*2+2)
	hex[0] = '0'
	hex[1] = 'x'

	const hexChars = "0123456789abcdef"
	for i, v := range b {
		hex[i*2+2] = hexChars[v>>4]
		hex[i*2+3] = hexChars[v&0x0f]
	}

	return string(hex)
}

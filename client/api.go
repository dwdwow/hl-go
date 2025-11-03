// Package client provides HTTP client functionality for the Hyperliquid API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dwdwow/hl-go/constants"
)

// APIError represents an API error response
type APIError struct {
	StatusCode int
	Code       *string
	Message    string
	Data       any
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Code != nil {
		return fmt.Sprintf("API error %d: %s - %s", e.StatusCode, *e.Code, e.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// API is the base client for making HTTP requests to the Hyperliquid API
type API struct {
	BaseURL    string
	HTTPClient *http.Client
	timeout    time.Duration
}

// NewAPI creates a new API client
// If baseURL is empty, it defaults to MainnetAPIURL
// If timeout is 0, it defaults to DefaultTimeout
func NewAPI(baseURL string, timeout time.Duration) *API {
	if baseURL == "" {
		baseURL = constants.MainnetAPIURL
	}

	if timeout == 0 {
		timeout = constants.DefaultTimeout * time.Second
	}

	return &API{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

type ExchangeResponse struct {
	Status   string          `json:"status"`
	Response json.RawMessage `json:"response,omitempty"`
}

func (a *API) exchangePost(urlPath string, payload any, result any) error {
	// Marshal payload
	var body []byte
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	} else {
		body = []byte("{}")
	}

	// Create request
	url := a.BaseURL + urlPath
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	respData := &ExchangeResponse{}

	if err := json.Unmarshal(respBody, respData); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return a.handleError(resp.StatusCode, respData.Response)
	}

	// Check API status
	if respData.Status != "ok" {
		// Response is an error message string
		var errMsg string
		if err := json.Unmarshal(respData.Response, &errMsg); err != nil {
			return fmt.Errorf("API error (status: %s): failed to parse error message", respData.Status)
		}
		return fmt.Errorf("API error: %s", errMsg)
	}

	// Parse response (status is "ok", response is data object)
	if result != nil {
		if err := json.Unmarshal(respData.Response, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

func (a *API) infoPost(urlPath string, payload any, result any) error {
	// Marshal payload
	var body []byte
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	} else {
		body = []byte("{}")
	}

	// Create request
	url := a.BaseURL + urlPath
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		return a.handleError(resp.StatusCode, respBody)
	}

	// Parse response
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// handleError processes error responses
func (a *API) handleError(statusCode int, body []byte) error {
	apiErr := &APIError{
		StatusCode: statusCode,
		Message:    string(body),
	}

	// Try to parse error as JSON
	var errResp struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data any    `json:"data"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Code != "" {
			apiErr.Code = &errResp.Code
		}
		if errResp.Msg != "" {
			apiErr.Message = errResp.Msg
		}
		apiErr.Data = errResp.Data
	}

	return apiErr
}

// IsMainnet returns true if the client is configured for mainnet
func (a *API) IsMainnet() bool {
	return a.BaseURL == constants.MainnetAPIURL
}

// SetTimeout updates the HTTP client timeout
func (a *API) SetTimeout(timeout time.Duration) {
	a.timeout = timeout
	a.HTTPClient.Timeout = timeout
}

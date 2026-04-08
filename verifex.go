// Package verifex provides a Go client for the Verifex sanctions screening API.
//
// Usage:
//
//	client := verifex.New("vfx_your_api_key")
//
//	// Screen a name
//	result, err := client.Screen(context.Background(), verifex.ScreenRequest{
//	    Name: "Vladimir Putin",
//	    Type: "person",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.RiskLevel)       // "critical"
//	fmt.Println(result.Matches[0].Name) // "PUTIN, Vladimir Vladimirovich"
//
//	// Batch screen
//	batch, err := client.BatchScreen(context.Background(), []verifex.ScreenRequest{
//	    {Name: "Vladimir Putin"},
//	    {Name: "John Doe"},
//	})
package verifex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultBaseURL = "https://api.verifex.dev"
	DefaultTimeout = 30 * time.Second
	Version        = "0.2.0"
	userAgent      = "verifex-go/" + Version
)

// Client is the Verifex API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures the Client.
type Option func(*Client)

// WithBaseURL sets a custom API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// WithHTTPClient sets a custom http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a new Verifex API client.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ── Screening ───────────────────────────────────────────────────────────────

// ScreenRequest is the input for screening a single entity.
type ScreenRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`         // "person" or "entity"
	Country     string `json:"country,omitempty"`
	DateOfBirth string `json:"date_of_birth,omitempty"`
	Mode        string `json:"mode,omitempty"`         // "exact" or "broad"
}

// Screen screens a single entity against all sanctions lists.
func (c *Client) Screen(ctx context.Context, req ScreenRequest) (*ScreenResult, error) {
	var result ScreenResult
	if err := c.do(ctx, http.MethodPost, "/v1/screen", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// BatchScreen screens multiple entities in a single request (Pro plan required).
func (c *Client) BatchScreen(ctx context.Context, entities []ScreenRequest) (*BatchScreenResult, error) {
	body := struct {
		Entities []ScreenRequest `json:"entities"`
	}{Entities: entities}

	var result BatchScreenResult
	if err := c.do(ctx, http.MethodPost, "/v1/screen/batch", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ── Usage ───────────────────────────────────────────────────────────────────

// Usage returns the current month's API usage statistics.
func (c *Client) Usage(ctx context.Context) (*UsageStats, error) {
	var result UsageStats
	if err := c.do(ctx, http.MethodGet, "/v1/usage", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ── API Keys ────────────────────────────────────────────────────────────────

// ListKeys returns all API keys for the authenticated user.
func (c *Client) ListKeys(ctx context.Context) ([]APIKeyInfo, error) {
	var result []APIKeyInfo
	if err := c.do(ctx, http.MethodGet, "/v1/keys", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateKey creates a new API key.
func (c *Client) CreateKey(ctx context.Context, name string) (*APIKeyCreated, error) {
	body := struct {
		Name string `json:"name"`
	}{Name: name}

	var result APIKeyCreated
	if err := c.do(ctx, http.MethodPost, "/v1/keys", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeKey permanently revokes an API key.
func (c *Client) RevokeKey(ctx context.Context, keyID string) error {
	return c.do(ctx, http.MethodDelete, "/v1/keys/"+keyID, nil, nil)
}

// ── Health ───────────────────────────────────────────────────────────────────

// Health checks the API health status (no authentication required).
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var result HealthResponse
	if err := c.doNoAuth(ctx, http.MethodGet, "/v1/health", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ── HTTP client ─────────────────────────────────────────────────────────────

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	return c.request(ctx, method, path, body, out, true)
}

func (c *Client) doNoAuth(ctx context.Context, method, path string, body, out any) error {
	return c.request(ctx, method, path, body, out, false)
}

func (c *Client) request(ctx context.Context, method, path string, body, out any, auth bool) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("verifex: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("verifex: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if auth {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("verifex: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("verifex: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, respBody)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("verifex: decode response: %w", err)
		}
	}

	return nil
}

func parseAPIError(status int, body []byte) error {
	var apiErr struct {
		Error     string `json:"error"`
		Code      string `json:"code"`
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(body, &apiErr)

	msg := apiErr.Error
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	code := apiErr.Code
	if code == "" {
		code = "UNKNOWN"
	}

	switch status {
	case 401:
		return &AuthenticationError{APIError{Message: msg, Code: code, StatusCode: status, RequestID: apiErr.RequestID}}
	case 402:
		return &QuotaExceededError{APIError{Message: msg, Code: code, StatusCode: status, RequestID: apiErr.RequestID}}
	case 429:
		return &RateLimitError{APIError: APIError{Message: msg, Code: code, StatusCode: status, RequestID: apiErr.RequestID}}
	default:
		return &APIError{Message: msg, Code: code, StatusCode: status, RequestID: apiErr.RequestID}
	}
}

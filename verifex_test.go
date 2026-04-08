package verifex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	c := New("vfx_test_key")
	if c.apiKey != "vfx_test_key" {
		t.Fatal("apiKey not set")
	}
	if c.baseURL != DefaultBaseURL {
		t.Fatalf("expected %s, got %s", DefaultBaseURL, c.baseURL)
	}
}

func TestWithOptions(t *testing.T) {
	c := New("key", WithBaseURL("https://custom.api.com"), WithTimeout(5000000000))
	if c.baseURL != "https://custom.api.com" {
		t.Fatalf("expected custom URL, got %s", c.baseURL)
	}
}

func TestHealthLive(t *testing.T) {
	c := New("dummy", WithBaseURL("https://api.verifex.dev"))
	h, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if h.Status != "ok" {
		t.Fatalf("expected ok, got %s", h.Status)
	}
	if h.Database != "connected" {
		t.Fatalf("expected connected DB, got %s", h.Database)
	}
	if !h.IsHealthy() {
		t.Fatal("expected healthy")
	}
	if h.TotalEntities() < 900000 {
		t.Fatalf("expected >900K entities, got %d", h.TotalEntities())
	}
}

func TestAuthErrorLive(t *testing.T) {
	c := New("invalid_key", WithBaseURL("https://api.verifex.dev"))
	_, err := c.Screen(context.Background(), ScreenRequest{Name: "test"})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !IsAuthError(err) {
		t.Fatalf("expected AuthenticationError, got %T: %v", err, err)
	}
}

func TestScreenMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/screen" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test_key" {
			t.Fatal("missing auth header")
		}
		if r.Header.Get("User-Agent") != userAgent {
			t.Fatalf("unexpected user-agent: %s", r.Header.Get("User-Agent"))
		}

		var req ScreenRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Name != "Vladimir Putin" {
			t.Fatalf("expected Putin, got %s", req.Name)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"query":         map[string]string{"name": "Vladimir Putin"},
			"matches":       []map[string]any{{"id": "1", "name": "PUTIN, Vladimir", "source": "OFAC", "entity_type": "person", "confidence": 98, "risk_level": "critical", "match_type": "EXACT", "aliases": []string{}}},
			"total_matches": 1,
			"risk_level":    "critical",
			"screened_at":   "2026-01-01T00:00:00Z",
			"request_id":    "test-123",
			"lists_checked": []string{"OFAC"},
			"api_version":   "v1",
		})
	}))
	defer srv.Close()

	c := New("test_key", WithBaseURL(srv.URL))
	result, err := c.Screen(context.Background(), ScreenRequest{Name: "Vladimir Putin"})
	if err != nil {
		t.Fatalf("Screen failed: %v", err)
	}
	if result.RiskLevel != "critical" {
		t.Fatalf("expected critical, got %s", result.RiskLevel)
	}
	if result.TotalMatches != 1 {
		t.Fatalf("expected 1 match, got %d", result.TotalMatches)
	}
	if !result.IsMatch() {
		t.Fatal("expected IsMatch true")
	}
	if result.IsClear() {
		t.Fatal("expected IsClear false")
	}
	if result.HighestConfidence() != 98 {
		t.Fatalf("expected 98 confidence, got %d", result.HighestConfidence())
	}
	if result.Matches[0].Name != "PUTIN, Vladimir" {
		t.Fatalf("expected PUTIN, Vladimir, got %s", result.Matches[0].Name)
	}
}

func TestClearResultMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"query":         map[string]string{"name": "Nobody Real"},
			"matches":       []any{},
			"total_matches": 0,
			"risk_level":    "clear",
			"screened_at":   "2026-01-01T00:00:00Z",
			"request_id":    "test-456",
			"lists_checked": []string{},
			"api_version":   "v1",
		})
	}))
	defer srv.Close()

	c := New("key", WithBaseURL(srv.URL))
	result, err := c.Screen(context.Background(), ScreenRequest{Name: "Nobody Real"})
	if err != nil {
		t.Fatalf("Screen failed: %v", err)
	}
	if !result.IsClear() {
		t.Fatal("expected clear")
	}
	if result.IsMatch() {
		t.Fatal("expected no match")
	}
	if result.HighestConfidence() != 0 {
		t.Fatalf("expected 0 confidence, got %d", result.HighestConfidence())
	}
}

func TestBatchScreenMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"query": map[string]string{"name": "A"}, "matches": []any{}, "total_matches": 0, "risk_level": "clear", "screened_at": "2026-01-01", "request_id": "r1", "api_version": "v1"},
				{"query": map[string]string{"name": "B"}, "matches": []any{}, "total_matches": 0, "risk_level": "clear", "screened_at": "2026-01-01", "request_id": "r2", "api_version": "v1"},
			},
			"total_duration_ms": 42,
		})
	}))
	defer srv.Close()

	c := New("key", WithBaseURL(srv.URL))
	result, err := c.BatchScreen(context.Background(), []ScreenRequest{{Name: "A"}, {Name: "B"}})
	if err != nil {
		t.Fatalf("BatchScreen failed: %v", err)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.TotalDurationMs != 42 {
		t.Fatalf("expected 42ms, got %d", result.TotalDurationMs)
	}
}

func TestAPIErrorParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]string{
			"error":      "Rate limit exceeded",
			"code":       "RATE_LIMIT_EXCEEDED",
			"request_id": "rl-123",
		})
	}))
	defer srv.Close()

	c := New("key", WithBaseURL(srv.URL))
	_, err := c.Screen(context.Background(), ScreenRequest{Name: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsRateLimitError(err) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	apiErr := err.(*RateLimitError)
	if apiErr.Code != "RATE_LIMIT_EXCEEDED" {
		t.Fatalf("expected RATE_LIMIT_EXCEEDED, got %s", apiErr.Code)
	}
	if apiErr.RequestID != "rl-123" {
		t.Fatalf("expected rl-123, got %s", apiErr.RequestID)
	}
}

func TestQuotaExceededError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(402)
		json.NewEncoder(w).Encode(map[string]string{"error": "Quota exceeded", "code": "QUOTA_EXCEEDED"})
	}))
	defer srv.Close()

	c := New("key", WithBaseURL(srv.URL))
	_, err := c.Screen(context.Background(), ScreenRequest{Name: "test"})
	if !IsQuotaExceededError(err) {
		t.Fatalf("expected QuotaExceededError, got %T", err)
	}
}

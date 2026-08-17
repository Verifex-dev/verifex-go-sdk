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
			// Complete coverage is required for IsClear. Before VER-6 this mock
			// omitted the field and still asserted IsClear() — the test encoded
			// the false-clear defect it should have caught.
			"coverage_status": "complete",
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

// ── VER-6 / REQ-SDK-001: coverage must gate "clear" ──────────────────────────
//
// A "clear" risk level only means nothing was found in the sources that were
// actually searched. If a sanctions list was unreachable, that is not the same
// as "this party is not sanctioned". Reporting it as clear is a false clear.

func screenWith(t *testing.T, body map[string]any) *ScreenResult {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()
	c := New("key", WithBaseURL(srv.URL))
	result, err := c.Screen(context.Background(), ScreenRequest{Name: "Nobody Real"})
	if err != nil {
		t.Fatalf("Screen failed: %v", err)
	}
	return result
}

func clearBody(coverage any) map[string]any {
	b := map[string]any{
		"query":         map[string]string{"name": "Nobody Real"},
		"matches":       []any{},
		"total_matches": 0,
		"risk_level":    "clear",
		"screened_at":   "2026-01-01T00:00:00Z",
		"request_id":    "cov-test",
		"lists_checked": []string{},
		"api_version":   "v1",
	}
	if coverage != nil {
		b["coverage_status"] = coverage
	}
	return b
}

func TestIsClearRequiresCompleteCoverage(t *testing.T) {
	r := screenWith(t, clearBody("complete"))
	if !r.IsClear() {
		t.Fatal("complete coverage + clear risk must be clear")
	}
	if !r.HasCompleteCoverage() {
		t.Fatal("expected HasCompleteCoverage")
	}
}

func TestPartialCoverageIsNotClear(t *testing.T) {
	body := clearBody("partial")
	body["unavailable_sources"] = []string{"OFAC"}
	r := screenWith(t, body)
	if r.IsClear() {
		t.Fatal("FALSE CLEAR: partial coverage reported as clear")
	}
	// The engine genuinely found nothing; that fact is still available, under a
	// name that does not imply a compliance conclusion.
	if !r.IsRiskClear() {
		t.Fatal("IsRiskClear should still be true")
	}
	if len(r.UnavailableSources) != 1 || r.UnavailableSources[0] != "OFAC" {
		t.Fatalf("unavailable sources not parsed: %v", r.UnavailableSources)
	}
}

func TestAbsentCoverageIsNotClear(t *testing.T) {
	// Absent is not a fourth flavour of complete. It means the question was
	// never answered, and there is no safe way to assume it was.
	r := screenWith(t, clearBody(nil))
	if r.IsClear() {
		t.Fatal("FALSE CLEAR: absent coverage treated as complete")
	}
	if r.HasCompleteCoverage() {
		t.Fatal("absent coverage must not report complete")
	}
	if !r.IsRiskClear() {
		t.Fatal("IsRiskClear should still be true")
	}
}

func TestMatchIsNeverClearRegardlessOfCoverage(t *testing.T) {
	body := clearBody("complete")
	body["risk_level"] = "high"
	body["total_matches"] = 1
	r := screenWith(t, body)
	if r.IsClear() || r.IsRiskClear() {
		t.Fatal("a match must never be clear")
	}
}

// ── VER-6 full safety contract: coverage is not the only way to lose a clear ──

func TestRestrictedMatchesBlockClear(t *testing.T) {
	// The most misleading case in the product: matches EXIST, the plan hid
	// them, and TotalMatches reads 0. A clear here would be actively false.
	body := clearBody("complete")
	body["restricted_matches"] = 2
	body["restricted_sources"] = []string{"PEP"}
	r := screenWith(t, body)
	if r.IsClear() {
		t.Fatal("FALSE CLEAR: restricted matches reported as clear")
	}
	if !contains(r.ClearBlockers(), "restricted_matches") {
		t.Fatalf("expected restricted_matches blocker, got %v", r.ClearBlockers())
	}
}

func TestScreeningUnavailableBlocksClear(t *testing.T) {
	body := clearBody("complete")
	body["screening_unavailable"] = true
	r := screenWith(t, body)
	if r.IsClear() {
		t.Fatal("FALSE CLEAR: unavailable screening reported as clear")
	}
}

func TestPlanScopedClearIsNotUniversalClear(t *testing.T) {
	// "Nothing found in the sources your plan covers" is not "nothing found".
	body := clearBody("complete")
	body["clear_scope"] = "checked_sources_only"
	body["sources_excluded_by_plan"] = []string{"PEP"}
	r := screenWith(t, body)
	if r.IsClear() {
		t.Fatal("FALSE CLEAR: plan-scoped no-hit reported as universal clear")
	}
}

func TestBlockersAccumulate(t *testing.T) {
	// Reporting only the first reason would send someone to fix coverage while
	// a restricted sanctions hit sat untouched.
	body := clearBody("partial")
	body["restricted_matches"] = 1
	body["screening_unavailable"] = true
	r := screenWith(t, body)
	for _, want := range []string{"coverage_incomplete", "restricted_matches", "screening_unavailable"} {
		if !contains(r.ClearBlockers(), want) {
			t.Fatalf("missing blocker %q in %v", want, r.ClearBlockers())
		}
	}
}

func TestGenuineClearHasNoBlockers(t *testing.T) {
	r := screenWith(t, clearBody("complete"))
	if len(r.ClearBlockers()) != 0 {
		t.Fatalf("expected no blockers, got %v", r.ClearBlockers())
	}
	if !r.IsClear() {
		t.Fatal("expected clear")
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

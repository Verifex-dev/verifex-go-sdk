package verifex

import (
	"context"
	"os"
	"testing"
)

func requireLiveSDK(t *testing.T) {
	t.Helper()
	if os.Getenv("VERIFEX_SDK_LIVE") != "1" {
		t.Skip("NOT_CONFIGURED: set VERIFEX_SDK_LIVE=1 to run production-network SDK smoke tests")
	}
}

func TestHealthLive(t *testing.T) {
	requireLiveSDK(t)
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
}

func TestAuthErrorLive(t *testing.T) {
	requireLiveSDK(t)
	c := New("invalid_key", WithBaseURL("https://api.verifex.dev"))
	_, err := c.Screen(context.Background(), ScreenRequest{Name: "test"})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !IsAuthError(err) {
		t.Fatalf("expected AuthenticationError, got %T: %v", err, err)
	}
}

package onedrive

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestVerifyDeviceCodeSetsExpiry(t *testing.T) {
	// Mock token endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"access_token":  "abc123",
			"refresh_token": "ref456",
			"token_type":    "Bearer",
			"expires_in":    3600,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// Override endpoints
	oldTokenURL := customTokenURL
	SetCustomEndpoints(customAuthURL, ts.URL, customDeviceURL)
	defer func() { SetCustomEndpoints(customAuthURL, oldTokenURL, customDeviceURL) }()

	tok, err := VerifyDeviceCode("client", "device", false)
	if err != nil {
		t.Fatalf("VerifyDeviceCode returned error: %v", err)
	}
	if tok.AccessToken != "abc123" {
		t.Fatalf("unexpected access token: %s", tok.AccessToken)
	}
	if tok.RefreshToken != "ref456" {
		t.Fatalf("unexpected refresh token: %s", tok.RefreshToken)
	}
	if tok.Expiry.Before(time.Now().Add(3590 * time.Second)) {
		t.Fatalf("expiry not set correctly: %v", tok.Expiry)
	}
}

func TestCustomTokenSourceMergeRefreshToken(t *testing.T) {
	// cached token with refresh token
	cached := &oauth2.Token{
		AccessToken:  "old",
		RefreshToken: "refresh-keep",
		Expiry:       time.Now().Add(-time.Hour),
	}
	// base source returns new token without refresh token
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "new", Expiry: time.Now().Add(time.Hour)})

	cts := &customTokenSource{base: src, cachedToken: cached}
	tok, err := cts.Token()
	if err != nil {
		t.Fatalf("Token() error: %v", err)
	}
	if tok.RefreshToken != "refresh-keep" {
		t.Fatalf("refresh token not preserved, got: %s", tok.RefreshToken)
	}
}

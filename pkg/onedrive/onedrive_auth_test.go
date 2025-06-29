package onedrive

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVerifyDeviceCodeSetsExpiry(t *testing.T) {
	// A token with expires_in should have Expiry set correctly
	responseToken := map[string]interface{}{
		"access_token":  "test_access_token",
		"token_type":    "Bearer",
		"refresh_token": "test_refresh_token",
		"expires_in":    3600, // 1 hour
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseToken)
	}))
	defer ts.Close()

	// Temporarily override the token URL to point to our mock server
	originalTokenURL := customTokenURL
	SetCustomEndpoints(oAuthAuthURL, ts.URL, oAuthDeviceURL)
	defer SetCustomEndpoints(oAuthAuthURL, originalTokenURL, oAuthDeviceURL)

	token, err := VerifyDeviceCode("test-client-id", "test-device-code", false)
	assert.NoError(t, err)
	assert.NotNil(t, token)

	// Check that Expiry is set and is in the future
	assert.False(t, token.Expiry.IsZero(), "Expiry should be set")
	assert.True(t, token.Expiry.After(time.Now()), "Expiry should be in the future")
	// Check that it's within a reasonable range (e.g., expires in > 59 minutes)
	assert.True(t, token.Expiry.After(time.Now().Add(59*time.Minute)), "Expiry should be about an hour from now")
}

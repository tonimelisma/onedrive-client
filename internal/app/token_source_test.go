package app

import (
	"errors"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

// mockTokenSource is a helper for testing the persistingTokenSource.
type mockTokenSource struct {
	mu         sync.Mutex
	token      *oauth2.Token
	err        error
	tokenCalls int
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokenCalls++
	return m.token, m.err
}

func (m *mockTokenSource) setToken(token *oauth2.Token) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = token
}

func TestPersistingTokenSource(t *testing.T) {
	initialToken := &oauth2.Token{AccessToken: "initial_access"}
	refreshedToken := &oauth2.Token{AccessToken: "refreshed_access"}
	var onNewTokenCalls int
	var receivedToken *oauth2.Token

	onNewTokenCallback := func(token *oauth2.Token) error {
		onNewTokenCalls++
		receivedToken = token
		return nil
	}

	mockSource := &mockTokenSource{token: initialToken}
	persistingSource := newPersistingTokenSource(mockSource, initialToken, onNewTokenCallback)

	// First call should return the initial token, no refresh.
	token, err := persistingSource.Token()
	if err != nil {
		t.Fatalf("Expected no error on first call, got %v", err)
	}
	if token.AccessToken != "initial_access" {
		t.Errorf("Expected initial access token, got %s", token.AccessToken)
	}
	if onNewTokenCalls != 0 {
		t.Errorf("Expected onNewToken to not be called, but it was called %d times", onNewTokenCalls)
	}
	if mockSource.tokenCalls != 1 {
		t.Errorf("Expected underlying source to be called once, got %d", mockSource.tokenCalls)
	}

	// Second call, token is the same, should not trigger callback.
	_, err = persistingSource.Token()
	if err != nil {
		t.Fatalf("Expected no error on second call, got %v", err)
	}
	if onNewTokenCalls != 0 {
		t.Errorf("Expected onNewToken to still not be called, but it was called %d times", onNewTokenCalls)
	}
	if mockSource.tokenCalls != 2 {
		t.Errorf("Expected underlying source to be called twice, got %d", mockSource.tokenCalls)
	}

	// Simulate a token refresh.
	mockSource.setToken(refreshedToken)
	token, err = persistingSource.Token()
	if err != nil {
		t.Fatalf("Expected no error on third call, got %v", err)
	}
	if token.AccessToken != "refreshed_access" {
		t.Errorf("Expected refreshed access token, got %s", token.AccessToken)
	}
	if onNewTokenCalls != 1 {
		t.Errorf("Expected onNewToken to be called once, got %d", onNewTokenCalls)
	}
	if receivedToken.AccessToken != "refreshed_access" {
		t.Errorf("Callback received wrong token: got %s", receivedToken.AccessToken)
	}
	if mockSource.tokenCalls != 3 {
		t.Errorf("Expected underlying source to be called three times, got %d", mockSource.tokenCalls)
	}
}

func TestPersistingTokenSource_CallbackError(t *testing.T) {
	initialToken := &oauth2.Token{AccessToken: "initial_access"}
	refreshedToken := &oauth2.Token{AccessToken: "refreshed_access"}
	callbackErr := errors.New("failed to save token")

	onNewTokenCallback := func(token *oauth2.Token) error {
		return callbackErr
	}

	mockSource := &mockTokenSource{token: initialToken}
	persistingSource := newPersistingTokenSource(mockSource, initialToken, onNewTokenCallback)

	// Simulate refresh
	mockSource.setToken(refreshedToken)
	token, err := persistingSource.Token()
	if err != nil {
		t.Fatalf("Expected no error from Token() even if callback fails, got %v", err)
	}
	if token.AccessToken != "refreshed_access" {
		t.Errorf("Expected to receive the new token even if callback fails, got %s", token.AccessToken)
	}
}

package onedrive

import (
	"errors"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

// mockTokenSource is a helper for testing the persistingTokenSource.
// It implements oauth2.TokenSource and allows us to swap out the token
// it returns to simulate refreshes.
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

func TestPersistingTokenSource_SuccessPath(t *testing.T) {
	initial := &oauth2.Token{AccessToken: "initial"}
	refreshed := &oauth2.Token{AccessToken: "refreshed"}

	var onNewCalls int
	var received *Token

	cb := func(tok *Token) error {
		onNewCalls++
		received = tok
		return nil
	}

	src := &mockTokenSource{token: initial}
	pts := &persistingTokenSource{
		source:     src,
		onNewToken: cb,
		lastToken:  initial,
	}

	// First call should return initial token.
	tok, err := pts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "initial" {
		t.Fatalf("expected initial token, got %s", tok.AccessToken)
	}
	if onNewCalls != 0 {
		t.Fatalf("onNewToken should not have been called yet")
	}

	// Simulate a refresh.
	src.setToken(refreshed)
	tok, err = pts.Token()
	if err != nil {
		t.Fatalf("unexpected error after refresh: %v", err)
	}
	if tok.AccessToken != "refreshed" {
		t.Fatalf("expected refreshed token, got %s", tok.AccessToken)
	}
	if onNewCalls != 1 {
		t.Fatalf("onNewToken should have been called exactly once, got %d", onNewCalls)
	}
	if received.AccessToken != "refreshed" {
		t.Fatalf("callback received wrong token: %s", received.AccessToken)
	}
}

func TestPersistingTokenSource_CallbackErrorPropagates(t *testing.T) {
	initial := &oauth2.Token{AccessToken: "initial"}
	refreshed := &oauth2.Token{AccessToken: "refreshed"}
	expectedErr := errors.New("persist failed")

	cb := func(tok *Token) error { return expectedErr }

	src := &mockTokenSource{token: initial}
	pts := &persistingTokenSource{
		source:     src,
		onNewToken: cb,
		lastToken:  initial,
	}

	// Trigger refresh
	src.setToken(refreshed)
	_, err := pts.Token()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped error to contain %v, got %v", expectedErr, err)
	}
}

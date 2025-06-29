package app

import (
	"sync"

	"golang.org/x/oauth2"
)

// persistingTokenSource is a wrapper around an oauth2.TokenSource that
// detects when a token has been refreshed and calls a callback to persist it.
type persistingTokenSource struct {
	base       oauth2.TokenSource
	mu         sync.Mutex
	lastToken  *oauth2.Token
	onNewToken func(token *oauth2.Token) error
}

// newPersistingTokenSource creates a new TokenSource that persists tokens.
func newPersistingTokenSource(base oauth2.TokenSource, initialToken *oauth2.Token, onNew func(token *oauth2.Token) error) *persistingTokenSource {
	return &persistingTokenSource{
		base:       base,
		lastToken:  initialToken,
		onNewToken: onNew,
	}
}

// Token returns a token from the underlying source. If the token was refreshed,
// it calls the onNewToken callback to persist it.
func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newToken, err := s.base.Token()
	if err != nil {
		return nil, err
	}

	// If the access token has changed, it means it was refreshed.
	if s.lastToken == nil || s.lastToken.AccessToken != newToken.AccessToken {
		s.lastToken = newToken
		if s.onNewToken != nil {
			if err := s.onNewToken(newToken); err != nil {
				// We don't want to fail the entire operation if saving fails,
				// but we should log it. The token is still valid in memory.
				// The calling application will need to log this.
				// For now, we return the token and let the app decide.
				return newToken, nil // a logging wrapper could be useful here
			}
		}
	}

	return newToken, nil
}

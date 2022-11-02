package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"
)

func getOauth2Config() (context.Context, *oauth2.Config) {
	return context.Background(), &oauth2.Config{
		ClientID: "71ae7ad2-0207-4618-90d3-d21db38f9f7a",
		Scopes:   []string{"offline_access", "files.readwrite.all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
	}
}

func authenticate(config *Configuration) (*http.Client, error) {
	ctx, oauthConfig := getOauth2Config()

	codeVerifier, err := cv.CreateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("creating code verifier: %v", err)
	}
	codeChallenge := codeVerifier.CodeChallengeS256()

	codeChallengeAuthParam := oauth2.SetAuthURLParam("code_challenge", codeChallenge)
	codeChallengeMethodAuthParam := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	fmt.Println("Visit the following URL in your browser and authorize the app.")
	fmt.Println(oauthConfig.AuthCodeURL("local", codeChallengeAuthParam, codeChallengeMethodAuthParam))
	fmt.Println("")
	fmt.Println("You'll be redirected to an empty page. Copy-paste its URL here:")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("scanning [%v]: %v", code, err)
	}
	fmt.Println("")

	// Parse oauth callback URL to ensure there's no error
	parsedUrl, err := url.Parse(code)
	if err != nil {
		return nil, fmt.Errorf("parsing url [%v]: %v", code, err)
	}

	// Parse oauth callback URL's query parameters
	parsedQuery, err := url.ParseQuery(parsedUrl.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("parsing query [%v]: %v", code, err)
	}

	// If callback returned an error, bring it up
	if parsedQuery.Has("error") {
		if parsedQuery.Has("error_description") {
			return nil, fmt.Errorf("oauth authorization failed: %v: %v", parsedQuery.Get("error"), parsedQuery.Get("error_description"))
		} else {
			return nil, fmt.Errorf("oauth authorization failed: %v", parsedQuery.Get("error"))
		}
	}

	if !parsedQuery.Has("code") {
		return nil, fmt.Errorf("couldn't parse code in callback: %v", parsedQuery.Encode())
	}

	codeVerifierAuthParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
	token, err := oauthConfig.Exchange(ctx, parsedQuery.Get("code"), codeVerifierAuthParam)
	if err != nil {
		return nil, err
	}

	err = saveToken(*token, config)
	if err != nil {
		return nil, err
	}

	client := oauthConfig.Client(ctx, token)
	return client, nil
}

// saveToken copies the token into the configuration and saves it to cache
func saveToken(token oauth2.Token, config *Configuration) error {
	debug(*config, "Old token: "+config.Token.AccessToken)
	debug(*config, "New token: "+token.AccessToken)
	config.Token = token
	err := saveConfiguration(*config)
	if err != nil {
		return err
	}
	return nil
}

// checkAndSaveToken() takes an oauth2 token and refreshes and saves it if its expired
func checkAndSaveToken(token *oauth2.Token, config *Configuration) error {
	ctx, oauthConfig := getOauth2Config()
	if token.Expiry.Before(time.Now()) {
		debug(*config, "Token expired, refreshing...")
		tokenSource := oauthConfig.TokenSource(ctx, token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return fmt.Errorf("couldn't refresh token: %v", err)
		}
		if newToken.AccessToken != token.AccessToken {
			debug(*config, "Token has changed, saving new token...")
			saveToken(*newToken, config)
			token = newToken
		}
	} else {
		debug(*config, "Token not yet expired: "+token.Expiry.String())
	}
	return nil
}

// validateToken restores a token from the configuration and refreshes it
func validateToken(config *Configuration) (*http.Client, error) {
	ctx, oauthConfig := getOauth2Config()

	debug(*config, "Validating token...")

	err := checkAndSaveToken(&config.Token, config)
	if err != nil {
		return nil, err
	}

	client := oauthConfig.Client(ctx, &config.Token)
	return client, nil
}

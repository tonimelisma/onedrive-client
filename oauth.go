package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"
)

func authenticate(config *Configuration) (*http.Client, error) {
	ctx := context.Background()
	oauthConf := &oauth2.Config{
		ClientID: "71ae7ad2-0207-4618-90d3-d21db38f9f7a",
		Scopes:   []string{"offline_access", "files.readwrite.all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
	}

	codeVerifier, err := cv.CreateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("creating code verifier: %v", err)
	}
	codeChallenge := codeVerifier.CodeChallengeS256()

	codeChallengeAuthParam := oauth2.SetAuthURLParam("code_challenge", codeChallenge)
	codeChallengeMethodAuthParam := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	fmt.Println("Visit the following URL in your browser and authorize the app.")
	fmt.Println(oauthConf.AuthCodeURL("local", codeChallengeAuthParam, codeChallengeMethodAuthParam))
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

	parsedQuery, err := url.ParseQuery(parsedUrl.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("parsing query [%v]: %v", code, err)
	}

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
	token, err := oauthConf.Exchange(ctx, parsedQuery.Get("code"), codeVerifierAuthParam)
	if err != nil {
		return nil, err
	}

	config.AccessToken = token.AccessToken
	config.Expiry = token.Expiry
	config.RefreshToken = token.RefreshToken
	config.TokenType = token.TokenType
	saveConfiguration(*config)

	client := oauthConf.Client(ctx, token)
	return client, nil
}

func validateToken(config *Configuration) (*http.Client, error) {
	return nil, nil
}

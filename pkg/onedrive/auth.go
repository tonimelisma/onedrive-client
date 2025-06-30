package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"
)

// SetCustomEndpoints allows overriding the default OAuth endpoints for testing.
func SetCustomEndpoints(authURL, tokenURL, deviceURL string) {
	customAuthURL = authURL
	customTokenURL = tokenURL
	customDeviceURL = deviceURL
}

// SetCustomGraphEndpoint allows overriding the default Graph API endpoint for testing.
func SetCustomGraphEndpoint(graphURL string) {
	customRootURL = graphURL
}

// OAuthConfig represents an OAuth2 Config.
type OAuthConfig oauth2.Config

// apiCallWithDebug handles HTTP requests with optional debug logging.
// It's used for unauthenticated calls like the device code flow.
func apiCallWithDebug(method, url, contentType string, body io.Reader, debug bool) (*http.Response, error) {
	var reqBodyBytes []byte
	if body != nil {
		reqBodyBytes, _ = io.ReadAll(body)
		body = bytes.NewBuffer(reqBodyBytes) // Restore body for request
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request failed: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if debug {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Println("Error dumping request:", err)
		} else {
			log.Printf("DEBUG Request:\n%s\n", string(dump))
		}
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %v", err)
	}

	if debug {
		dump, err := httputil.DumpResponse(res, true)
		if err != nil {
			log.Println("Error dumping response:", err)
		} else {
			log.Printf("DEBUG Response:\n%s\n", string(dump))
		}
	}

	if res.StatusCode >= 400 {
		defer res.Body.Close()
		resBody, _ := io.ReadAll(res.Body)

		var oneDriveError struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}

		jsonErr := json.Unmarshal(resBody, &oneDriveError)
		if jsonErr == nil && oneDriveError.Error != "" {
			switch oneDriveError.Error {
			case "authorization_pending":
				return nil, ErrAuthorizationPending
			case "authorization_declined":
				return nil, ErrAuthorizationDeclined
			case "expired_token":
				return nil, ErrTokenExpired
			case "invalid_request":
				return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, oneDriveError.ErrorDescription)
			default:
				return nil, fmt.Errorf("authentication error: %s - %s", oneDriveError.Error, oneDriveError.ErrorDescription)
			}
		}
		return nil, fmt.Errorf("HTTP error: %s", res.Status)
	}

	return res, nil
}

// StartAuthentication begins the OAuth2 PKCE flow.
// It returns the authentication URL for the user and a code verifier.
func StartAuthentication(
	ctx context.Context,
	oauthConfig *OAuthConfig,
) (authURL string, codeVerifier string, err error) {
	if ctx == nil {
		return "", "", fmt.Errorf("ctx is nil")
	}

	var codeVerifierObj *cv.CodeVerifier
	codeVerifierObj, err = cv.CreateCodeVerifier()
	if err != nil {
		return "", "", fmt.Errorf("could not create PKCE code verifier: %w", err)
	}
	codeVerifier = codeVerifierObj.String()
	codeChallenge := codeVerifierObj.CodeChallengeS256()

	pkceParams := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}

	authURL = (*oauth2.Config)(oauthConfig).AuthCodeURL("state-does-not-matter", pkceParams...)
	return authURL, codeVerifier, nil
}

// CompleteAuthentication exchanges the authorization code for an OAuth token.
func CompleteAuthentication(
	ctx context.Context,
	oauthConfig *OAuthConfig,
	code string,
	verifier string,
) (*Token, error) {
	pkceCodeVerifier := oauth2.SetAuthURLParam("code_verifier", verifier)
	token, err := (*oauth2.Config)(oauthConfig).Exchange(ctx, code, pkceCodeVerifier)
	if err != nil {
		return nil, err
	}

	// The standard library does not automatically set the Expiry field from the expires_in field.
	// We must do it manually. If the Expiry is not set, the token source will believe the token
	// never expires and will never attempt to refresh it.
	if token.Expiry.IsZero() {
		if expiresIn, ok := token.Extra("expires_in").(float64); ok {
			token.Expiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
		}
	}

	return (*Token)(token), nil
}

// GetOauth2Config returns the OAuth2 configuration.
func GetOauth2Config(clientID string) (context.Context, *OAuthConfig) {
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID: clientID,
		Scopes:   oAuthScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  customAuthURL,
			TokenURL: customTokenURL,
		},
	}
	return ctx, (*OAuthConfig)(conf)
}

// InitiateDeviceCodeFlow starts the device code authentication process.
func InitiateDeviceCodeFlow(clientID string, debug bool) (*DeviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", strings.Join(oAuthScopes, " "))

	res, err := apiCallWithDebug("POST", customDeviceURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()), debug)
	if err != nil {
		return nil, fmt.Errorf("requesting device code: %w", err)
	}
	defer res.Body.Close()

	var deviceCodeResponse DeviceCodeResponse
	if err := json.NewDecoder(res.Body).Decode(&deviceCodeResponse); err != nil {
		return nil, fmt.Errorf("decoding device code response failed: %v", err)
	}

	return &deviceCodeResponse, nil
}

// VerifyDeviceCode polls to verify the device code and get an access token.
func VerifyDeviceCode(clientID string, deviceCode string, debug bool) (*Token, error) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)

	res, err := apiCallWithDebug("POST", customTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()), debug)
	if err != nil {
		return nil, fmt.Errorf("verifying device code: %w", err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("retrieving token failed: %s - %s", res.Status, body)
	}

	var token oauth2.Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("parsing token failed: %v", err)
	}

	var expiresIn struct {
		ExpiresIn int `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &expiresIn); err != nil {
		return nil, fmt.Errorf("parsing expires_in failed: %v", err)
	}

	if expiresIn.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(expiresIn.ExpiresIn) * time.Second)
	}

	return (*Token)(&token), nil
}

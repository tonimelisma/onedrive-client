// Package onedrive (auth.go) provides functions for handling OAuth2 authentication
// with the Microsoft identity platform, specifically tailored for OneDrive access.
// This includes support for the OAuth2 Authorization Code Grant with PKCE (Proof Key
// for Code Exchange) and the Device Code Flow.
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

// SetCustomEndpoints allows overriding the default OAuth endpoints.
// This is primarily used for testing purposes, enabling tests to target
// mock OAuth servers instead of Microsoft's live endpoints.
// It modifies global variables `customAuthURL`, `customTokenURL`, and `customDeviceURL`.
func SetCustomEndpoints(authURL, tokenURL, deviceURL string) {
	customAuthURL = authURL
	customTokenURL = tokenURL
	customDeviceURL = deviceURL
}

// SetCustomGraphEndpoint allows overriding the default Microsoft Graph API root endpoint.
// Similar to SetCustomEndpoints, this is useful for testing against a mock Graph API server.
// It modifies the global variable `customRootURL`.
func SetCustomGraphEndpoint(graphURL string) {
	customRootURL = graphURL
}

// OAuthConfig is an alias for oauth2.Config, tailored for OneDrive.
// It represents the configuration for an OAuth2 client.
type OAuthConfig oauth2.Config

// apiCallWithDebug is an unexported helper function for making HTTP requests,
// typically used for unauthenticated calls during the OAuth device code flow.
// It includes optional debug logging of requests and responses if `debug` is true.
// This function does not use the authenticated `Client.httpClient`.
func apiCallWithDebug(method, url, contentType string, body io.Reader, debug bool) (*http.Response, error) {
	var reqBodyBytes []byte
	if body != nil {
		// Read the body for potential logging, then restore it for the actual request.
		var readErr error
		reqBodyBytes, readErr = io.ReadAll(body)
		if readErr != nil {
			log.Printf("Warning: Failed to read request body for logging: %v", readErr)
		}
		body = bytes.NewBuffer(reqBodyBytes)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s %s failed: %w", method, url, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if debug {
		// Dump the outgoing request for debugging.
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Println("Error dumping request:", err)
		} else {
			log.Printf("DEBUG Request:\n%s\n", string(dump))
		}
	}

	// Use a configured HTTP client for consistent timeout behavior during auth operations
	authClient := NewConfiguredHTTPClient(DefaultHTTPConfig())
	res, err := authClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error during API call to %s %s: %w", method, url, err)
	}

	if debug {
		// Dump the incoming response for debugging.
		dump, err := httputil.DumpResponse(res, true)
		if err != nil {
			log.Println("Error dumping response:", err)
		} else {
			log.Printf("DEBUG Response:\n%s\n", string(dump))
		}
	}

	// Handle HTTP errors from the OAuth server.
	if res.StatusCode >= StatusBadRequest {
		defer func() {
			if closeErr := res.Body.Close(); closeErr != nil {
				log.Printf("Warning: Failed to close OAuth error response body: %v", closeErr)
			}
		}()
		resBodyBytes, readErr := io.ReadAll(res.Body)
		if readErr != nil {
			log.Printf("Warning: Failed to read OAuth error response body: %v", readErr)
			return nil, fmt.Errorf("HTTP error %s from %s (could not read response body)", res.Status, url)
		}

		var oauthError struct { // Common OAuth error response structure.
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}

		jsonErr := json.Unmarshal(resBodyBytes, &oauthError)
		if jsonErr == nil && oauthError.Error != "" {
			// Map known OAuth error codes to specific sentinel errors.
			switch oauthError.Error {
			case "authorization_pending":
				return nil, ErrAuthorizationPending
			case "authorization_declined":
				return nil, ErrAuthorizationDeclined
			case "expired_token": // Typically means the device code itself expired.
				return nil, ErrTokenExpired
			case "invalid_request", "invalid_grant": // invalid_grant can occur if device code is wrong.
				return nil, fmt.Errorf("%w: %s (OAuth server)", ErrInvalidRequest, oauthError.ErrorDescription)
			default:
				return nil, fmt.Errorf("OAuth authentication error '%s': %s", oauthError.Error, oauthError.ErrorDescription)
			}
		}
		// Fallback for non-JSON errors or unknown error structures.
		return nil, fmt.Errorf("HTTP error %s from %s: %s", res.Status, url, string(resBodyBytes))
	}

	return res, nil
}

// StartAuthentication begins the OAuth2 Authorization Code Grant flow with PKCE.
// It generates a code verifier and challenge, then constructs the authorization URL
// to which the user should be redirected.
//
// Returns:
//   - authURL: The URL for the user to visit to authorize the application.
//   - codeVerifier: The PKCE code verifier string. This must be stored securely
//     by the client and sent in the token exchange request via CompleteAuthentication.
//   - err: Any error encountered during the process.
//
// Example:
//
//	ctx, oauthCfg := onedrive.GetOauth2Config("YOUR_CLIENT_ID")
//	authURL, verifier, err := onedrive.StartAuthentication(ctx, oauthCfg)
//	if err != nil { log.Fatal(err) }
//	// Store verifier securely (e.g., in a session)
//	// Redirect user to authURL
func StartAuthentication(
	ctx context.Context,
	oauthConfig *OAuthConfig,
) (authURL string, codeVerifier string, err error) {
	if ctx == nil {
		// Context is required by the underlying oauth2 library.
		return "", "", fmt.Errorf("context must not be nil for StartAuthentication")
	}

	var codeVerifierObj *cv.CodeVerifier
	codeVerifierObj, err = cv.CreateCodeVerifier() // Generate a new PKCE code verifier.
	if err != nil {
		return "", "", fmt.Errorf("could not create PKCE code verifier: %w", err)
	}
	codeVerifier = codeVerifierObj.String()
	codeChallenge := codeVerifierObj.CodeChallengeS256() // Generate S256 challenge from verifier.

	// Parameters for PKCE.
	pkceParams := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}

	// Construct the authorization URL. "state" can be used to prevent CSRF but is less critical
	// in some PKCE flows if the redirect URI is strictly controlled.
	authURL = (*oauth2.Config)(oauthConfig).AuthCodeURL("state-does-not-matter", pkceParams...)
	return authURL, codeVerifier, nil
}

// CompleteAuthentication exchanges an authorization code (obtained after user consent
// in the PKCE flow) for an OAuth token.
// It requires the original PKCE code verifier that was generated by StartAuthentication.
//
// This function also manually calculates and sets the `Expiry` field on the token,
// as the standard `oauth2.Token.Exchange` method may not populate it correctly from
// the `expires_in` field in the token response. A correct `Expiry` is vital for
// the `oauth2.TokenSource` to manage token refreshes automatically.
//
// Example:
//
//	// After user is redirected back from authURL with an authorization code:
//	code := "THE_AUTHORIZATION_CODE_FROM_REDIRECT"
//	verifier := "THE_STORED_CODE_VERIFIER" // From StartAuthentication
//	ctx, oauthCfg := onedrive.GetOauth2Config("YOUR_CLIENT_ID")
//	token, err := onedrive.CompleteAuthentication(ctx, oauthCfg, code, verifier)
//	if err != nil { log.Fatal(err) }
//	// Use token to create an authenticated onedrive.Client
func CompleteAuthentication(
	ctx context.Context,
	oauthConfig *OAuthConfig,
	code string, // Authorization code from the redirect.
	verifier string, // Original PKCE code verifier.
) (*Token, error) {
	// Include the code_verifier in the token exchange request as per PKCE spec.
	pkceCodeVerifier := oauth2.SetAuthURLParam("code_verifier", verifier)
	token, err := (*oauth2.Config)(oauthConfig).Exchange(ctx, code, pkceCodeVerifier)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to exchange authorization code for token: %w", ErrOperationFailed, err)
	}

	// Manually set the Expiry field from "expires_in".
	// The oauth2 library's default TokenSource relies on a correctly set Expiry
	// to determine when a token needs refreshing. If Expiry is zero, the
	// token source might assume the token never expires, thus never attempting a refresh.
	if token.Expiry.IsZero() {
		if expiresIn, ok := token.Extra("expires_in").(float64); ok { // expires_in is often a number.
			token.Expiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
		} else if expiresInStr, ok := token.Extra("expires_in").(string); ok { // sometimes a string.
			if expiresInInt, pErr := time.ParseDuration(expiresInStr + "s"); pErr == nil {
				token.Expiry = time.Now().Add(expiresInInt)
			}
		}
		// If Expiry is still zero, it might indicate an issue with the token response
		// or that the token truly has no server-provided expiry (rare for access tokens).
	}

	return (*Token)(token), nil
}

// GetOauth2Config returns a basic OAuth2 configuration for OneDrive.
// It uses the provided clientID and the SDK's predefined scopes and endpoints
// (which can be customized for testing via SetCustomEndpoints).
//
// Example:
//
//	ctx, oauthCfg := onedrive.GetOauth2Config("YOUR_CLIENT_ID")
//	// Use oauthCfg with StartAuthentication and CompleteAuthentication
func GetOauth2Config(clientID string) (context.Context, *OAuthConfig) {
	// A background context is generally suitable for configuration setup.
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID: clientID,
		Scopes:   oAuthScopes, // Uses the global oAuthScopes.
		Endpoint: oauth2.Endpoint{
			AuthURL:  customAuthURL,  // Uses customizable endpoint.
			TokenURL: customTokenURL, // Uses customizable endpoint.
		},
	}
	return ctx, (*OAuthConfig)(conf)
}

// InitiateDeviceCodeFlow starts the OAuth2 Device Code Flow.
// This flow is suitable for CLI applications or devices without a web browser.
// It returns a DeviceCodeResponse containing the user_code, verification_uri,
// and other details that should be displayed to the user.
// The application then polls the token endpoint using VerifyDeviceCode.
//
// Example:
//
//	resp, err := onedrive.InitiateDeviceCodeFlow("YOUR_CLIENT_ID", true) // true for debug
//	if err != nil { log.Fatal(err) }
//	fmt.Println(resp.Message) // E.g., "To sign in, use a web browser to open the page https://microsoft.com/devicelogin and enter the code XXXXXXXX to authenticate."
//	// Store resp.DeviceCode and poll using VerifyDeviceCode
func InitiateDeviceCodeFlow(clientID string, debug bool) (*DeviceCodeResponse, error) {
	// Parameters for the device code request.
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scope", strings.Join(oAuthScopes, " ")) // Scopes are space-separated.

	// Make the unauthenticated call to the device code endpoint.
	res, err := apiCallWithDebug("POST", customDeviceURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()), debug)
	if err != nil {
		return nil, fmt.Errorf("requesting device code from %s: %w", customDeviceURL, err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("Warning: failed to close device code response body: %v", err)
		}
	}()

	var deviceCodeResponse DeviceCodeResponse
	if err := json.NewDecoder(res.Body).Decode(&deviceCodeResponse); err != nil {
		return nil, fmt.Errorf("%w: decoding device code response: %w", ErrDecodingFailed, err)
	}

	return &deviceCodeResponse, nil
}

// VerifyDeviceCode polls the token endpoint to exchange a device_code (obtained from
// InitiateDeviceCodeFlow) for an OAuth token.
// This function should be called periodically according to the `interval` specified
// in the DeviceCodeResponse.
//
// If the user has not yet authenticated in their browser, this will return
// an error, typically ErrAuthorizationPending.
// Once the user authenticates, this function returns the OAuth token.
//
// Similar to CompleteAuthentication, this function manually sets the `Expiry`
// field on the returned token.
//
// Example:
//
//	// After InitiateDeviceCodeFlow, assuming 'deviceCode' is from its response:
//	// Loop with appropriate polling interval (e.g., resp.Interval from InitiateDeviceCodeFlow)
//	token, err := onedrive.VerifyDeviceCode("YOUR_CLIENT_ID", deviceCode, true)
//	if err != nil {
//	    if errors.Is(err, onedrive.ErrAuthorizationPending) {
//	        // Continue polling
//	    } else {
//	        // Handle other errors (e.g., code expired, user declined)
//	        log.Fatal(err)
//	    }
//	} else {
//	    // Token acquired successfully
//	    // Use token to create an authenticated onedrive.Client
//	}
func VerifyDeviceCode(clientID string, deviceCode string, debug bool) (*Token, error) {
	// Parameters for the token request in the device code flow.
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)

	// Poll the token endpoint.
	res, err := apiCallWithDebug("POST", customTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()), debug)
	if err != nil {
		// apiCallWithDebug already maps "authorization_pending", etc., to sentinel errors.
		return nil, fmt.Errorf("polling token endpoint %s: %w", customTokenURL, err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("Warning: failed to close token response body: %v", err)
		}
	}()

	// Read the response body once.
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response body: %w", err)
	}

	// Check for HTTP errors not caught by apiCallWithDebug's specific OAuth error parsing.
	// This is a safeguard, as apiCallWithDebug should handle most >= StatusBadRequest errors.
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: retrieving token via device code failed with status %s: %s", ErrOperationFailed, res.Status, string(bodyBytes))
	}

	// Parse the token from the response body.
	var token oauth2.Token
	if err := json.Unmarshal(bodyBytes, &token); err != nil {
		return nil, fmt.Errorf("%w: parsing token from device code response: %w", ErrDecodingFailed, err)
	}

	// Manually set the Expiry field from "expires_in", crucial for token refresh logic.
	var expiresInHolder struct { // Temporary struct to unmarshal only expires_in.
		ExpiresIn json.Number `json:"expires_in"` // Use json.Number for flexibility.
	}
	if err := json.Unmarshal(bodyBytes, &expiresInHolder); err != nil {
		// Log this, but don't fail if token was otherwise parsed, as Expiry might be set by oauth2 lib in some cases.
		if debug {
			log.Printf("DEBUG: could not parse 'expires_in' field from token response: %v", err)
		}
	}

	if expiresInHolder.ExpiresIn != "" {
		if expiresInInt, err := expiresInHolder.ExpiresIn.Int64(); err == nil && expiresInInt > 0 {
			token.Expiry = time.Now().Add(time.Duration(expiresInInt) * time.Second)
		} else if debug {
			log.Printf("DEBUG: 'expires_in' field ('%s') could not be converted to int64: %v", expiresInHolder.ExpiresIn, err)
		}
	}
	// If Expiry is still zero after this, the token source might not refresh correctly.

	return (*Token)(&token), nil
}

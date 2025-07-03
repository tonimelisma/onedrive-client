// Package onedrive provides an SDK for interacting with the Microsoft Graph API
// for OneDrive services. It includes functionality for authentication,
// file and folder management, drive operations, and other OneDrive features.
//
// The core component is the Client, which handles API calls, automatic token
// refresh, and error handling.
package onedrive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tonimelisma/onedrive-client/internal/logger"
	"golang.org/x/oauth2"
)

// Logger defines the interface for structured logging with multiple levels.
// This interface is now defined in the internal/logger package and imported here
// for backward compatibility with external SDK consumers.
type Logger = logger.Logger

// DefaultLogger is a no-op logger that satisfies the Logger interface.
// It is used if no other logger is provided to the Client.
type DefaultLogger = logger.NoopLogger

// oAuthScopes defines the necessary OAuth2 permissions (scopes) required by the SDK
// to access OneDrive resources on behalf of the user.
var oAuthScopes = []string{"offline_access", "files.readwrite.all", "user.read", "email", "openid", "profile"}

// Constants for Microsoft Graph API and OAuth2 endpoints.
// These are the standard production URLs.
const (
	oAuthAuthURL   = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"  // Standard OAuth2 authorization endpoint.
	oAuthTokenURL  = "https://login.microsoftonline.com/common/oauth2/v2.0/token"      // Standard OAuth2 token endpoint.
	oAuthDeviceURL = "https://login.microsoftonline.com/common/oauth2/v2.0/devicecode" // Endpoint for OAuth2 Device Code Flow.
	rootUrl        = "https://graph.microsoft.com/v1.0/"                               // Base URL for Microsoft Graph API v1.0.
)

// Variables that allow overriding the default API and OAuth endpoints.
// This is primarily useful for testing against mock servers or different environments.
var (
	customAuthURL   = oAuthAuthURL   // Can be overridden by SetCustomEndpoints.
	customTokenURL  = oAuthTokenURL  // Can be overridden by SetCustomEndpoints.
	customDeviceURL = oAuthDeviceURL // Can be overridden by SetCustomEndpoints.
	customRootURL   = rootUrl        // Can be overridden by SetCustomGraphEndpoint.
)

// Token represents an OAuth2 Token and is the canonical representation
// used by the SDK. It embeds oauth2.Token and can be used directly
// with the golang.org/x/oauth2 package.
type Token oauth2.Token

// HTTPConfig represents HTTP client configuration for the SDK
type HTTPConfig struct {
	Timeout       time.Duration // HTTP request timeout
	RetryAttempts int           // Maximum number of retry attempts
	RetryDelay    time.Duration // Initial retry delay
	MaxRetryDelay time.Duration // Maximum retry delay for exponential backoff
}

// DefaultHTTPConfig returns sensible default HTTP configuration values
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		MaxRetryDelay: 10 * time.Second,
	}
}

// NewConfiguredHTTPClient creates a new HTTP client with the specified configuration.
// This function creates a basic HTTP client with timeout settings.
// For OAuth2-authenticated requests, use oauth2.NewClient with the appropriate TokenSource.
func NewConfiguredHTTPClient(config HTTPConfig) *http.Client {
	return &http.Client{
		Timeout: config.Timeout,
		// Additional transport configuration can be added here if needed
	}
}

// NewConfiguredHTTPClientWithTransport creates an HTTP client with custom transport and configuration.
// This is useful when you need to preserve an existing transport (like OAuth2 transport)
// while applying timeout configuration.
func NewConfiguredHTTPClientWithTransport(config HTTPConfig, transport http.RoundTripper) *http.Client {
	return &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}
}

// Client is a stateful client for interacting with the Microsoft Graph API for OneDrive.
// It encapsulates an HTTP client that automatically handles OAuth2 token refreshes.
// The Client also provides a mechanism for persisting new tokens via a callback.
// All API calls made through this client are authenticated.
type Client struct {
	httpClient *http.Client       // The underlying HTTP client, configured with OAuth2 handling.
	onNewToken func(*Token) error // Callback invoked when a token is refreshed.
	logger     Logger             // Logger for debugging SDK operations.
	httpConfig HTTPConfig         // HTTP configuration for non-authenticated clients
}

// SetLogger allows users of the SDK to set their own logger implementation.
// This is useful for integrating SDK logs with the application's logging system.
// If no logger is set, a DefaultLogger (no-op) will be used.
//
// Example:
//
//	type MyCustomLogger struct { /* ... */ }
//	func (m *MyCustomLogger) Debug(v ...interface{}) { log.Println(v...) }
//	client.SetLogger(&MyCustomLogger{})
func (c *Client) SetLogger(l Logger) {
	c.logger = l
}

// NewClient creates a new OneDrive client.
// It requires an initial OAuth2 token, the client ID of the application,
// and a callback function `onNewToken`. This callback is triggered whenever
// the OAuth2 access token is refreshed, allowing the application to persist
// the new token. A logger can also be provided; if nil, a DefaultLogger is used.
//
// The `ctx` (context.Context) is used for the initial token source setup and
// underlying HTTP client configuration.
//
// Example:
//
//	initialToken := &onedrive.Token{AccessToken: "...", RefreshToken: "...", Expiry: ...}
//	clientID := "your-application-client-id"
//	var myLogger onedrive.Logger // Your logger implementation
//
//	onTokenRefresh := func(newToken *onedrive.Token) error {
//	    // Persist the newToken (e.g., save to a file or database)
//	    fmt.Printf("New token received: %s\n", newToken.AccessToken)
//	    return saveTokenToFile(newToken)
//	}
//
//	client := onedrive.NewClient(context.Background(), initialToken, clientID, onTokenRefresh, myLogger)
//	// Now use the client to make API calls, e.g., client.GetMe(context.Background())
func NewClient(ctx context.Context, initialToken *Token, clientID string, onNewToken func(*Token) error, logger Logger) *Client {
	return NewClientWithConfig(ctx, initialToken, clientID, onNewToken, logger, DefaultHTTPConfig())
}

// NewClientWithConfig creates a new OneDrive client with custom HTTP configuration.
// This allows fine-tuning of HTTP timeouts, retry behavior, and other client settings.
func NewClientWithConfig(ctx context.Context, initialToken *Token, clientID string, onNewToken func(*Token) error, logger Logger, httpConfig HTTPConfig) *Client {
	// The oauth2.Config is used here primarily to configure the TokenSource for refresh operations.
	// It does not initiate a new token acquisition flow itself.
	config := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  customAuthURL,  // Uses customizable endpoint.
			TokenURL: customTokenURL, // Uses customizable endpoint.
		},
		Scopes: oAuthScopes,
	}

	// persistingTokenSource wraps the standard oauth2.TokenSource.
	// It intercepts token refreshes to trigger the onNewToken callback.
	persistingSource := &persistingTokenSource{
		source:     config.TokenSource(ctx, (*oauth2.Token)(initialToken)),
		onNewToken: onNewToken,
		lastToken:  (*oauth2.Token)(initialToken), // Store initial token for comparison.
	}

	if logger == nil {
		logger = DefaultLogger{} // Use no-op logger if none provided.
	}

	// Create the base oauth2 client with authentication
	baseOAuth2Client := oauth2.NewClient(ctx, persistingSource)

	// Apply our HTTP configuration while preserving the OAuth2 transport
	configuredClient := NewConfiguredHTTPClientWithTransport(httpConfig, baseOAuth2Client.Transport)

	return &Client{
		httpClient: configuredClient,
		onNewToken: onNewToken,
		logger:     logger,
		httpConfig: httpConfig,
	}
}

// persistingTokenSource is an internal struct that wraps an oauth2.TokenSource.
// Its purpose is to detect when a token has been refreshed by the underlying
// source and then invoke the `onNewToken` callback to allow the application
// to persist the new token. This is crucial for long-lived sessions.
type persistingTokenSource struct {
	source     oauth2.TokenSource // The original token source from the oauth2 library.
	onNewToken func(*Token) error // Callback to persist the new token.
	mu         sync.Mutex         // Protects access to lastToken.
	lastToken  *oauth2.Token      // Stores the last known token to detect changes.
}

// Token retrieves a token from the underlying source. If the token is refreshed
// (i.e., the access token changes), it invokes the onNewToken callback.
// This method is called by the oauth2 HTTP client before making an API request.
func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	// This call may block if the token is expired and a refresh is needed.
	newToken, err := s.source.Token()
	if err != nil {
		return nil, err // Error during token retrieval or refresh.
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if the access token has changed, indicating a refresh.
	if s.lastToken == nil || s.lastToken.AccessToken != newToken.AccessToken {
		s.lastToken = newToken // Update the last known token.
		if s.onNewToken != nil {
			// Invoke the application's callback to persist the new token.
			if err := s.onNewToken((*Token)(newToken)); err != nil {
				// If persisting fails, log it or handle as critical.
				// Returning an error here might prevent the original API call from proceeding.
				// For now, we make the failure visible.
				return nil, fmt.Errorf("%w: failed to persist new token: %w", ErrOperationFailed, err)
			}
		}
	}

	return newToken, nil
}

// BuildPathURL constructs the full Microsoft Graph API URL for a given item path
// within the user's default OneDrive (me/drive).
// It handles encoding and correct formatting for root and nested paths.
//
// Example:
//
//	BuildPathURL("") -> "https://graph.microsoft.com/v1.0/me/drive/root"
//	BuildPathURL("/") -> "https://graph.microsoft.com/v1.0/me/drive/root"
//	BuildPathURL("/Documents/MyFile.docx") -> "https://graph.microsoft.com/v1.0/me/drive/root:/Documents/MyFile.docx"
func BuildPathURL(path string) string {
	// For the root of the drive, the path is simply "me/drive/root".
	if path == "" || path == "/" {
		return customRootURL + "me/drive/root"
	}
	// For other paths, they are relative to the root and require URI encoding.
	// The format is "me/drive/root:/<encoded_path>".
	encodedPath := strings.TrimPrefix(path, "/") // Ensure no leading slash before encoding.
	// Note: url.PathEscape is not used here as Graph API expects certain characters like ':' to be unescaped in this segment.
	// The path itself should be correctly formed by the caller if it contains special characters needing specific encoding.
	return customRootURL + "me/drive/root:/" + encodedPath
}

// GetMe retrieves the profile of the currently signed-in user.
// This is often used as a basic connectivity test or to get user identifiers.
//
// Example:
//
//	user, err := client.GetMe(context.Background())
//	if err != nil {
//	    log.Fatalf("Failed to get user: %v", err)
//	}
//	fmt.Printf("Logged in as: %s (%s)\n", user.DisplayName, user.UserPrincipalName)
func (c *Client) GetMe(ctx context.Context) (User, error) {
	c.logger.Debug("GetMe called")
	var user User

	url := customRootURL + "me" // Endpoint for the current user's profile.
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return user, err // Error from apiCall (network, auth, or API error).
	}
	defer closeBodySafely(res.Body, c.logger, "user info")

	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return user, fmt.Errorf("%w: decoding user: %w", ErrDecodingFailed, err)
	}

	return user, nil
}

// GetSharedWithMe retrieves a list of drive items that have been shared with the current user.
// This includes items shared by others that appear in the user's "Shared with me" view in OneDrive.
//
// Example:
//
//	sharedItems, err := client.GetSharedWithMe(context.Background())
//	if err != nil {
//	    log.Fatalf("Failed to get shared items: %v", err)
//	}
//	for _, item := range sharedItems.Value {
//	    fmt.Printf("Shared item: %s\n", item.Name)
//	}
func (c *Client) GetSharedWithMe(ctx context.Context) (DriveItemList, error) {
	c.logger.Debug("GetSharedWithMe called")
	var items DriveItemList

	// Endpoint for items shared with the current user.
	url := customRootURL + "me/drive/sharedWithMe"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer closeBodySafely(res.Body, c.logger, "shared items")

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("%w: decoding shared items: %w", ErrDecodingFailed, err)
	}

	return items, nil
}

// GetRecentItems retrieves a list of drive items that have been recently accessed by the current user.
// This corresponds to the "Recent" view in OneDrive.
//
// Example:
//
//	recentItems, err := client.GetRecentItems(context.Background())
//	if err != nil {
//	    log.Fatalf("Failed to get recent items: %v", err)
//	}
//	for _, item := range recentItems.Value {
//	    fmt.Printf("Recent item: %s (Last modified: %s)\n", item.Name, item.LastModifiedDateTime)
//	}
func (c *Client) GetRecentItems(ctx context.Context) (DriveItemList, error) {
	c.logger.Debug("GetRecentItems called")
	var items DriveItemList

	// Endpoint for recently accessed items.
	url := customRootURL + "me/drive/recent"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer closeBodySafely(res.Body, c.logger, "recent items")

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("%w: decoding recent items: %w", ErrDecodingFailed, err)
	}

	return items, nil
}

// GetSpecialFolder retrieves a specific "special" folder for the user, such as Documents, Photos, etc.
// Special folders are well-known folders that OneDrive provisions for users.
// Valid folder names include: "documents", "photos", "cameraroll", "approot", "music", "desktop", "downloads", "videos".
//
// Example:
//
//	documentsFolder, err := client.GetSpecialFolder(context.Background(), "documents")
//	if err != nil {
//	    log.Fatalf("Failed to get documents folder: %v", err)
//	}
//	fmt.Printf("Documents folder ID: %s, Name: %s\n", documentsFolder.ID, documentsFolder.Name)
func (c *Client) GetSpecialFolder(ctx context.Context, folderName string) (DriveItem, error) {
	c.logger.Debugf("GetSpecialFolder called for folder: %s", folderName)
	var item DriveItem

	// Endpoint for special folders, requires URL path escaping for the folder name.
	url := customRootURL + "me/drive/special/" + url.PathEscape(folderName)

	err := c.makeAPICallAndDecode(ctx, "GET", url, "", nil, &item, fmt.Sprintf("special folder '%s'", folderName))
	if err != nil {
		return item, err
	}

	return item, nil
}

// GetDelta retrieves changes to items in a drive using delta queries.
// This is useful for efficiently syncing a local state with the remote OneDrive state.
//
// An initial call without a `deltaToken` will return all items and a `DeltaLink`
// (which contains a delta token for the next call). Subsequent calls with the
// `deltaToken` from the `DeltaLink` will return only items that have changed
// since the last call.
//
// Example (initial call):
//
//	deltaResponse, err := client.GetDelta(context.Background(), "")
//	if err != nil { log.Fatal(err) }
//	for _, item := range deltaResponse.Value { fmt.Println("Item:", item.Name) }
//	nextToken := deltaResponse.DeltaLink // Save this for the next sync
//
// Example (subsequent call):
//
//	deltaResponse, err = client.GetDelta(context.Background(), extractTokenFromDeltaLink(nextToken))
//	// Process changed items...
//	nextToken = deltaResponse.DeltaLink
func (c *Client) GetDelta(ctx context.Context, deltaToken string) (DeltaResponse, error) {
	c.logger.Debugf("GetDelta called with token: %s", deltaToken)
	var deltaResponse DeltaResponse

	url := customRootURL + "me/drive/root/delta"
	if deltaToken != "" {
		// The deltaToken is the opaque token part of the @odata.deltaLink.
		// It should not be the full deltaLink URL.
		url += "?token=" + deltaToken
	}

	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return deltaResponse, err
	}
	defer closeBodySafely(res.Body, c.logger, "delta response")

	if err := json.NewDecoder(res.Body).Decode(&deltaResponse); err != nil {
		return deltaResponse, fmt.Errorf("%w: decoding delta response: %w", ErrDecodingFailed, err)
	}

	return deltaResponse, nil
}

// GetFileVersions retrieves all versions of a specific file by its path.
// This allows access to the version history of a file if versioning is enabled on the drive.
// Note: This first resolves the file path to an item ID.
//
// Example:
//
//	versions, err := client.GetFileVersions(context.Background(), "/Documents/Report.docx")
//	if err != nil { log.Fatal(err) }
//	for _, version := range versions.Value {
//	    fmt.Printf("Version ID: %s, Size: %d, Modified: %s\n", version.ID, version.Size, version.LastModifiedDateTime)
//	}
func (c *Client) GetFileVersions(ctx context.Context, filePath string) (DriveItemVersionList, error) {
	c.logger.Debugf("GetFileVersions called for path: %s", filePath)
	var versions DriveItemVersionList

	// First, get the DriveItem to resolve its ID from the path.
	// This is necessary because the versions endpoint requires an item ID.
	item, err := c.GetDriveItemByPath(ctx, filePath)
	if err != nil {
		return versions, fmt.Errorf("getting file item ID for versions: %w", err)
	}
	if item.Folder != nil {
		return versions, fmt.Errorf("cannot get versions for a folder: %s", filePath)
	}

	url := customRootURL + "me/drive/items/" + item.ID + "/versions"
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return versions, err
	}
	defer closeBodySafely(res.Body, c.logger, "file versions")

	if err := json.NewDecoder(res.Body).Decode(&versions); err != nil {
		return versions, fmt.Errorf("%w: decoding versions response for '%s': %w", ErrDecodingFailed, filePath, err)
	}

	return versions, nil
}

// collectAllPages is an unexported helper function to handle pagination for Graph API list calls.
// It fetches pages of results, starting from `initialURL` or `paging.NextLink`.
// If `paging.FetchAll` is true, it follows all `@odata.nextLink`s until all items are retrieved.
// Otherwise, it fetches only the current page specified by `initialURL` or `paging.NextLink`.
// It returns a slice of raw JSON messages (one per item) and the nextLink for further pagination, if any.
// Using json.RawMessage helps in deferring the unmarshaling of individual items, which can be
// memory-efficient if the caller processes items one by one or needs different parts of the JSON.
func (c *Client) collectAllPages(ctx context.Context, initialURL string, paging Paging) ([]json.RawMessage, string, error) {
	var allItems []json.RawMessage
	currentURL := initialURL

	// If NextLink is specified, start from there instead of initialURL.
	if paging.NextLink != "" {
		currentURL = paging.NextLink
	}

	for {
		c.logger.Debugf("collectAllPages fetching URL: %s", currentURL)

		// Add $top parameter if specified and not already in URL
		if paging.Top > 0 && !strings.Contains(currentURL, "$top=") {
			separator := "?"
			if strings.Contains(currentURL, "?") {
				separator = "&"
			}
			currentURL += fmt.Sprintf("%s$top=%d", separator, paging.Top)
		}

		res, err := c.apiCall(ctx, "GET", currentURL, "", nil)
		if err != nil {
			return allItems, "", err
		}
		defer res.Body.Close()

		// Parse the response into a generic structure to extract value and nextLink
		var response struct {
			Value    []json.RawMessage `json:"value"`
			NextLink string            `json:"@odata.nextLink"`
		}

		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			return allItems, "", fmt.Errorf("%w: decoding paginated response: %w", ErrDecodingFailed, err)
		}

		// Accumulate items from this page
		allItems = append(allItems, response.Value...)

		// If no next link or not fetching all pages, stop
		if response.NextLink == "" || !paging.FetchAll {
			return allItems, response.NextLink, nil
		}

		// Continue with the next page
		currentURL = response.NextLink
	}
}

// apiCall is an unexported helper that centralizes making HTTP requests to the Graph API.
// It handles:
// 1. Constructing the request with context.
// 2. Using the Client's authenticated httpClient.
// 3. Automatic retry on 401 Unauthorized (once): The oauth2 client transport should refresh the token.
// 4. Common error categorization based on status codes and error responses from OneDrive.
// 5. Rewinding the request body if a retry is needed (for POST/PUT requests).
//
// This function is fundamental to the SDK's operation.
func (c *Client) apiCall(ctx context.Context, method, url, contentType string, body io.ReadSeeker) (*http.Response, error) {
	maxRetries := c.httpConfig.RetryAttempts
	retryDelay := c.httpConfig.RetryDelay
	maxRetryDelay := c.httpConfig.MaxRetryDelay

	var res *http.Response

	for i := 0; i < maxRetries; i++ {
		c.logger.Debugf("apiCall attempt #%d invoked with method: %s, URL: %s", i+1, method, url)

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		// Set content type if provided
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		c.logger.Debug("Request created, sending request...")

		// Make the request using the client's configured HTTP client
		res, err = c.httpClient.Do(req)
		if err != nil {
			// Network error or other transport-level failure
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				if body != nil {
					body.Seek(0, 0) // Reset body for retry
				}
				continue
			}
			return nil, fmt.Errorf("%w: HTTP request failed after %d attempts: %w", ErrNetworkFailed, maxRetries, err)
		}

		// Handle HTTP errors
		switch res.StatusCode {
		case StatusOK, StatusCreated, StatusAccepted, StatusNoContent:
			// Success responses
			return res, nil
		case StatusBadRequest:
			// Bad Request - malformed request
			closeBodySafely(res.Body, c.logger, "bad request")
			return nil, fmt.Errorf("%w: received %d Bad Request from %s", ErrInvalidRequest, StatusBadRequest, url)
		case StatusUnauthorized:
			// Unauthorized - token might be expired
			c.logger.Debugf("Received %d Unauthorized on attempt #%d. URL: %s", StatusUnauthorized, i+1, url)
			closeBodySafely(res.Body, c.logger, "unauthorized")
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				if body != nil {
					logOnError(seekToStart(body), c.logger, "seek body for retry")
				}
				continue
			}
			c.logger.Debugf("Still received %d after retry for URL: %s, failing.", StatusUnauthorized, url)
			return nil, fmt.Errorf("%w: received %d from %s", ErrReauthRequired, StatusUnauthorized, url)
		case StatusForbidden:
			// Forbidden - access denied
			closeBodySafely(res.Body, c.logger, "forbidden")
			return nil, fmt.Errorf("%w: received %d Forbidden from %s", ErrAccessDenied, StatusForbidden, url)
		case StatusNotFound:
			// Not Found - resource doesn't exist
			closeBodySafely(res.Body, c.logger, "not found")
			return nil, fmt.Errorf("%w: received %d Not Found from %s", ErrResourceNotFound, StatusNotFound, url)
		case StatusConflict:
			// Conflict - resource conflict (e.g., name already exists)
			closeBodySafely(res.Body, c.logger, "conflict")
			return nil, fmt.Errorf("%w: received %d Conflict from %s", ErrConflict, StatusConflict, url)
		case StatusPayloadTooLarge:
			// Payload Too Large - quota exceeded
			closeBodySafely(res.Body, c.logger, "payload too large")
			return nil, fmt.Errorf("%w: received %d Payload Too Large from %s", ErrQuotaExceeded, StatusPayloadTooLarge, url)
		case StatusInsufficientStorage:
			// Insufficient Storage - quota exceeded
			closeBodySafely(res.Body, c.logger, "insufficient storage")
			return nil, fmt.Errorf("%w: received %d Insufficient Storage from %s", ErrQuotaExceeded, StatusInsufficientStorage, url)
		case StatusTooManyRequests:
			// Rate limited - should retry with backoff
			closeBodySafely(res.Body, c.logger, "rate limited")
			if i < maxRetries-1 {
				// Exponential backoff with maximum delay cap
				retryAfter := time.Duration(i+1) * retryDelay * 2
				if retryAfter > maxRetryDelay {
					retryAfter = maxRetryDelay
				}
				time.Sleep(retryAfter)
				c.logger.Debugf("Retrying request to URL: %s", url)
				if body != nil {
					logOnError(seekToStart(body), c.logger, "seek body for retry")
				}
				continue
			}
			return nil, fmt.Errorf("%w: rate limited after %d attempts from %s", ErrRetryLater, maxRetries, url)
		case StatusServiceUnavailable:
			// Service Unavailable - temporary issue
			closeBodySafely(res.Body, c.logger, "service unavailable")
			if i < maxRetries-1 {
				time.Sleep(retryDelay)
				c.logger.Debugf("Service unavailable, retrying request to URL: %s", url)
				if body != nil {
					logOnError(seekToStart(body), c.logger, "seek body for retry")
				}
				continue
			}
			return nil, fmt.Errorf("%w: service unavailable after %d attempts from %s", ErrRetryLater, maxRetries, url)
		default:
			// Other HTTP errors
			errorBody := readErrorBody(res.Body)
			closeBodySafely(res.Body, c.logger, "unexpected status")
			return nil, fmt.Errorf("HTTP %d from %s: %s", res.StatusCode, url, errorBody)
		}
	}

	return res, nil
}

// Sentinel errors are predefined error values that can be checked by callers
// using errors.Is() to handle specific types of API failures.
var (
	ErrReauthRequired        = errors.New("re-authentication required")               // Authentication failed, user needs to log in again.
	ErrAccessDenied          = errors.New("access denied")                            // User does not have permission for the operation.
	ErrRetryLater            = errors.New("service busy or unavailable, retry later") // Temporary issue, operation might succeed on retry.
	ErrInvalidRequest        = errors.New("invalid request")                          // The request was malformed or invalid.
	ErrResourceNotFound      = errors.New("resource not found")                       // The requested item or resource does not exist.
	ErrConflict              = errors.New("conflict with existing resource")          // e.g., item name already exists, or eTag mismatch.
	ErrQuotaExceeded         = errors.New("storage quota exceeded")                   // User's OneDrive storage quota has been reached.
	ErrAuthorizationPending  = errors.New("authorization pending")                    // Used in device code flow; user hasn't approved yet.
	ErrAuthorizationDeclined = errors.New("authorization declined by user")           // User explicitly denied the authorization request.
	ErrTokenExpired          = errors.New("token expired")                            // Access or refresh token has expired (during auth flow).
	ErrInternal              = errors.New("internal SDK error")                       // Indicates an unexpected issue within the SDK itself.
	ErrDecodingFailed        = errors.New("response decoding failed")                 // JSON/response decoding failed.
	ErrNetworkFailed         = errors.New("network operation failed")                 // Network-level failure.
	ErrOperationFailed       = errors.New("operation failed")                         // General operation failure.
)

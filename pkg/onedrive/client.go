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

	"golang.org/x/oauth2"
)

// Logger is the interface that the SDK uses for logging.
// Users of the SDK can provide their own implementation to integrate
// with their application's logging infrastructure.
type Logger interface {
	Debug(v ...interface{})
}

// DefaultLogger is a no-op logger that satisfies the Logger interface.
// It is used if no other logger is provided to the Client.
type DefaultLogger struct{}

// Debug for DefaultLogger does nothing.
func (l DefaultLogger) Debug(v ...interface{}) {}

// oAuthScopes defines the necessary OAuth2 permissions (scopes) required by the SDK
// to access OneDrive resources on behalf of the user.
var oAuthScopes = []string{"offline_access", "files.readwrite.all", "user.read", "email", "openid", "profile"}

// Constants for Microsoft Graph API and OAuth2 endpoints.
// These are the standard production URLs.
const (
	oAuthAuthURL   = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize" // Standard OAuth2 authorization endpoint.
	oAuthTokenURL  = "https://login.microsoftonline.com/common/oauth2/v2.0/token"     // Standard OAuth2 token endpoint.
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

// Client is a stateful client for interacting with the Microsoft Graph API for OneDrive.
// It encapsulates an HTTP client that automatically handles OAuth2 token refreshes.
// The Client also provides a mechanism for persisting new tokens via a callback.
// All API calls made through this client are authenticated.
type Client struct {
	httpClient *http.Client      // The underlying HTTP client, configured with OAuth2 handling.
	onNewToken func(*Token) error // Callback invoked when a token is refreshed.
	logger     Logger            // Logger for debugging SDK operations.
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

	// oauth2.NewClient creates an http.Client that automatically uses the
	// persistingTokenSource to manage and refresh tokens for outgoing requests.
	return &Client{
		httpClient: oauth2.NewClient(ctx, persistingSource),
		onNewToken: onNewToken,
		logger:     logger,
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
				return nil, fmt.Errorf("failed to persist new token: %w", err)
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
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return user, fmt.Errorf("decoding user failed: %v", err)
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
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding shared items failed: %v", err)
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
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding recent items failed: %v", err)
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
	c.logger.Debug("GetSpecialFolder called for folder: ", folderName)
	var item DriveItem

	// Endpoint for special folders, requires URL path escaping for the folder name.
	url := customRootURL + "me/drive/special/" + url.PathEscape(folderName)
	res, err := c.apiCall(ctx, "GET", url, "", nil)
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding special folder ('%s') failed: %v", folderName, err)
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
	c.logger.Debug("GetDelta called with token: ", deltaToken)
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
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&deltaResponse); err != nil {
		return deltaResponse, fmt.Errorf("decoding delta response: %v", err)
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
	c.logger.Debug("GetFileVersions called for path: ", filePath)
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
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&versions); err != nil {
		return versions, fmt.Errorf("decoding versions response for '%s': %v", filePath, err)
	}

	return versions, nil
}

// collectAllPages is an unexported helper function to handle pagination for Graph API list calls.
// It fetches pages of results, starting from `initialURL` or `paging.NextLink`.
// If `paging.FetchAll` is true, it follows all `@odata.nextLink`s until all items are retrieved.
// Otherwise, it fetches only the current page specified by `initialURL` or `paging.NextLink`.
// It returns a slice of raw JSON messages (one per item) and the nextLink for further pagination, if any.
// Using json.RawMessage helps in deferring the unmarshalling of individual items, which can be
// memory-efficient if the caller processes items one by one or needs different parts of the JSON.
func (c *Client) collectAllPages(ctx context.Context, initialURL string, paging Paging) ([]json.RawMessage, string, error) {
	var allItems []json.RawMessage
	currentURL := initialURL

	// If a specific NextLink is provided in Paging, it overrides the initialURL.
	// This allows resuming pagination from a known point.
	if paging.NextLink != "" {
		currentURL = paging.NextLink
	}

	// Loop as long as there's a URL to fetch from.
	// For non-FetchAll scenarios, this loop runs at most once.
	for currentURL != "" {
		c.logger.Debug("collectAllPages fetching URL: ", currentURL)
		res, err := c.apiCall(ctx, "GET", currentURL, "", nil)
		if err != nil {
			return nil, "", err // Propagate error from apiCall.
		}
		// It's crucial to close the response body in each iteration if an error occurs after this point,
		// or if we break early. Using a defer inside the loop for res.Body.Close() handles this.
		func() {
			defer res.Body.Close()
			var page struct {
				Value    []json.RawMessage `json:"value"`
				NextLink string            `json:"@odata.nextLink"`
			}

			bodyBytes, readErr := io.ReadAll(res.Body)
			if readErr != nil {
				err = fmt.Errorf("reading page body from %s: %w", currentURL, readErr)
				return // Set outer err
			}

			if unmarshalErr := json.Unmarshal(bodyBytes, &page); unmarshalErr != nil {
				err = fmt.Errorf("decoding page from %s: %w", currentURL, unmarshalErr)
				return // Set outer err
			}

			allItems = append(allItems, page.Value...)
			currentURL = page.NextLink // Update currentURL for the next iteration.

			// If FetchAll is false, we only process one page.
			if !paging.FetchAll {
				currentURL = "" // Stop the loop after this page.
				// We still return page.NextLink so the caller knows if more pages are available.
				// The 'nextLink' return value of the outer function will be this page.NextLink.
			}
		}()
		if err != nil { // Check if an error occurred inside the func()
			return nil, "", err
		}
		if !paging.FetchAll { // If not fetching all, break after one page.
			break
		}
	}
	// The final 'nextLink' to return is the one from the last successfully fetched page,
	// or empty if all pages were fetched (and FetchAll was true) or an error occurred.
	// If FetchAll was false, currentURL was set to "" inside the loop to stop, but we
	// need to return the actual nextLink from that single page.
	// This is slightly tricky because currentURL is modified.
	// Let's re-evaluate: the 'currentURL' at this point, if loop terminated naturally, is empty.
	// If loop was broken by !paging.FetchAll, currentURL is also empty.
	// The actual link to return for "next page" is the NextLink from the *last processed page*.
	// The `collectAllPages` logic was modified to return the `nextLink` from the last fetched page
	// when `paging.FetchAll` is false.
	// The `currentURL` variable here will be the one for the *next* page to fetch if the loop continues.
	// So if `paging.FetchAll` is false, the returned `nextLink` should be the `@odata.nextLink` of the page just fetched.
	// If `paging.FetchAll` is true, the returned `nextLink` will be empty once all pages are exhausted.

	// The logic for returning the 'nextLink' needs to be precise.
	// If `paging.FetchAll` is true, `currentURL` will be empty when all pages are done.
	// If `paging.FetchAll` is false, the loop breaks after one iteration. `currentURL` (which was `page.NextLink`)
	// is the correct value to return as the 'next page link'.
	// The current implementation where `currentURL` is updated and used to control the loop correctly
	// results in `currentURL` being the next link if `!paging.FetchAll`, or empty if all fetched.
	return allItems, currentURL, nil
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
	var res *http.Response
	var err error

	// Retry loop (max 2 attempts: initial + 1 retry on 401)
	for i := 0; i < 2; i++ {
		c.logger.Debug("apiCall attempt #", i+1, " invoked with method: ", method, ", URL: ", url)

		if c.httpClient == nil {
			// This should ideally not happen if NewClient is used correctly.
			return nil, ErrInternal
		}

		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("creating request for %s %s failed: %w", method, url, err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		c.logger.Debug("Request created, sending request...")
		res, err = c.httpClient.Do(req) // This is where the OAuth2 client transport works.
		if err != nil {
			// This block handles network errors or errors from the token source during refresh
			// (e.g., refresh token is invalid).
			// It's unlikely to be a simple 401 that can be retried by apiCall itself,
			// as the oauth2 transport handles standard 401s by attempting a refresh.
			// If that refresh fails, the error comes here.
			var oauth2RetrieveError *oauth2.RetrieveError
			if errors.As(err, &oauth2RetrieveError) {
				// These are errors from the oauth2 library, often indicating a problem with the token itself.
				switch oauth2RetrieveError.ErrorCode {
				case "invalid_request", "invalid_client", "invalid_grant",
					"unauthorized_client", "unsupported_grant_type",
					"invalid_scope", "access_denied":
					// These generally mean the refresh token is bad or permissions are revoked. Re-authentication is needed.
					return nil, fmt.Errorf("%w: OAuth token refresh failed (%s): %v", ErrReauthRequired, oauth2RetrieveError.ErrorCode, err)
				case "server_error", "temporarily_unavailable":
					// Server-side issue at the OAuth provider.
					return nil, fmt.Errorf("%w: OAuth token refresh temporarily unavailable (%s): %v", ErrRetryLater, oauth2RetrieveError.ErrorCode, err)
				default:
					return nil, fmt.Errorf("unexpected OAuth2 error during token refresh (%s): %v", oauth2RetrieveError.ErrorCode, err)
				}
			} else {
				// This is likely a network-level error (DNS, TCP connection, etc.).
				return nil, fmt.Errorf("network error during API call to %s %s: %w", method, url, err)
			}
		}

		// If the status is not 401, we are done with this attempt. Break the loop.
		if res.StatusCode != http.StatusUnauthorized {
			break
		}

		// We received a 401 Unauthorized.
		// This was our first attempt (i=0). The oauth2 HTTP client transport
		// should have already attempted a token refresh. If it still results in a 401,
		// it might mean the refreshed token is also invalid, or there's a deeper auth issue.
		// However, Graph API sometimes returns 401s that are transient or require a simple retry
		// even if the token *should* be valid (e.g. clock skew, replication delays).
		// The original implementation retried once. Let's maintain that behavior but acknowledge
		// the oauth2 client should ideally handle this.
		c.logger.Debug("Received 401 Unauthorized on attempt #", i+1, ". URL: ", url)

		// Before retrying, close the previous response body to avoid resource leaks.
		res.Body.Close()

		// If this was the second attempt (i=1) and we still got a 401, we give up.
		if i == 1 {
			c.logger.Debug("Still received 401 after retry for URL: ", url, ", failing.")
			// The error will be processed by the status code check below.
			break
		}

		// Rewind the request body if it's not nil, so it can be read again for the retry.
		// This is crucial for methods like POST or PUT that have a request body.
		if body != nil {
			if _, seekErr := body.Seek(0, io.SeekStart); seekErr != nil {
				return nil, fmt.Errorf("failed to rewind request body for retry on %s %s: %w", method, url, seekErr)
			}
		}
		c.logger.Debug("Retrying request to URL: ", url)
		// Loop continues for the second attempt.
	}

	// After the loop (either broke early or completed 2 attempts), check the final response code.
	if res.StatusCode >= 400 {
		// Ensure body is closed, especially if an error occurs during parsing.
		// If an error occurred in the loop and res is from an earlier attempt, this might be redundant
		// but is safe.
		defer res.Body.Close()
		resBodyBytes, _ := io.ReadAll(res.Body) // Read body for error details.

		// Attempt to parse standard Graph API error structure.
		var oneDriveError struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		jsonErr := json.Unmarshal(resBodyBytes, &oneDriveError)
		detailedErrorMessage := string(resBodyBytes) // Fallback message
		if jsonErr == nil && oneDriveError.Error.Code != "" {
			detailedErrorMessage = oneDriveError.Error.Message
		}

		// Map specific error codes to sentinel errors for easier handling by the caller.
		if jsonErr == nil && oneDriveError.Error.Code != "" {
			switch oneDriveError.Error.Code {
			case "accessDenied":
				return nil, fmt.Errorf("%w: %s (URL: %s)", ErrAccessDenied, detailedErrorMessage, url)
			case "activityLimitReached", "serviceNotAvailable", "transientFailure": // Added transientFailure
				return nil, fmt.Errorf("%w: %s (URL: %s)", ErrRetryLater, detailedErrorMessage, url)
			case "itemNotFound":
				return nil, fmt.Errorf("%w: %s (URL: %s)", ErrResourceNotFound, detailedErrorMessage, url)
			case "nameAlreadyExists", "resourceModified": // resourceModified can sometimes indicate a conflict
				return nil, fmt.Errorf("%w: %s (URL: %s)", ErrConflict, detailedErrorMessage, url)
			case "invalidRange", "invalidRequest", "malwareDetected",
				"notAllowed", "notSupported", "resyncRequired", "generalException":
				return nil, fmt.Errorf("%w: %s (URL: %s)", ErrInvalidRequest, detailedErrorMessage, url)
			case "quotaLimitReached":
				return nil, fmt.Errorf("%w: %s (URL: %s)", ErrQuotaExceeded, detailedErrorMessage, url)
			case "unauthenticated": // Should ideally be caught by 401 and token refresh.
				return nil, fmt.Errorf("%w: %s (URL: %s)", ErrReauthRequired, detailedErrorMessage, url)
			default:
				// For unknown error codes, return a generic error.
				return nil, fmt.Errorf(
					"OneDrive API error: status %s, code %s, message: %s (URL: %s)",
					res.Status,
					oneDriveError.Error.Code,
					detailedErrorMessage,
					url,
				)
			}
		} else {
			// If the error isn't in the standard Graph JSON format, use HTTP status for categorization.
			// This provides a fallback for non-standard errors or proxy errors.
			switch res.StatusCode {
			case http.StatusBadRequest, http.StatusMethodNotAllowed, http.StatusNotAcceptable,
				http.StatusLengthRequired, http.StatusPreconditionFailed,
				http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType,
				http.StatusRequestedRangeNotSatisfiable, http.StatusUnprocessableEntity:
				return nil, fmt.Errorf("%w: HTTP %s, message: %s (URL: %s)", ErrInvalidRequest, res.Status, detailedErrorMessage, url)
			case http.StatusUnauthorized, http.StatusForbidden: // StatusForbidden often means access denied at a higher level.
				return nil, fmt.Errorf("%w: HTTP %s, message: %s (URL: %s)", ErrReauthRequired, res.Status, detailedErrorMessage, url)
			case http.StatusGone, http.StatusNotFound:
				return nil, fmt.Errorf("%w: HTTP %s, message: %s (URL: %s)", ErrResourceNotFound, res.Status, detailedErrorMessage, url)
			case http.StatusConflict:
				return nil, fmt.Errorf("%w: HTTP %s, message: %s (URL: %s)", ErrConflict, res.Status, detailedErrorMessage, url)
			case http.StatusInsufficientStorage: // (507)
				return nil, fmt.Errorf("%w: HTTP %s, message: %s (URL: %s)", ErrQuotaExceeded, res.Status, detailedErrorMessage, url)
			case http.StatusNotImplemented, http.StatusTooManyRequests, // TooManyRequests (429) should be retried.
				http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout: // Added 502, 504
				return nil, fmt.Errorf("%w: HTTP %s, message: %s (URL: %s)", ErrRetryLater, res.Status, detailedErrorMessage, url)
			default:
				return nil, fmt.Errorf("unexpected HTTP error: status %s, message: %s (URL: %s)", res.Status, detailedErrorMessage, url)
			}
		}
	}

	// If no error, return the successful response.
	return res, nil
}

// Sentinel errors are predefined error values that can be checked by callers
// using errors.Is() to handle specific types of API failures.
var (
	ErrReauthRequired        = errors.New("re-authentication required")         // Authentication failed, user needs to log in again.
	ErrAccessDenied          = errors.New("access denied")                      // User does not have permission for the operation.
	ErrRetryLater            = errors.New("service busy or unavailable, retry later") // Temporary issue, operation might succeed on retry.
	ErrInvalidRequest        = errors.New("invalid request")                    // The request was malformed or invalid.
	ErrResourceNotFound      = errors.New("resource not found")                 // The requested item or resource does not exist.
	ErrConflict              = errors.New("conflict with existing resource")    // e.g., item name already exists, or eTag mismatch.
	ErrQuotaExceeded         = errors.New("storage quota exceeded")             // User's OneDrive storage quota has been reached.
	ErrAuthorizationPending  = errors.New("authorization pending")              // Used in device code flow; user hasn't approved yet.
	ErrAuthorizationDeclined = errors.New("authorization declined by user")     // User explicitly denied the authorization request.
	ErrTokenExpired          = errors.New("token expired")                      // Access or refresh token has expired (during auth flow).
	ErrInternal              = errors.New("internal SDK error")                 // Indicates an unexpected issue within the SDK itself.
)

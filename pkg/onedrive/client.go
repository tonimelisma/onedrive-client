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

	"golang.org/x/oauth2"
)

// Logger is the interface that the SDK uses for logging.
type Logger interface {
	Debug(v ...interface{})
}

type DefaultLogger struct{}

func (l DefaultLogger) Debug(v ...interface{}) {}

// OAuth2 scopes and endpoints
var oAuthScopes = []string{"offline_access", "files.readwrite.all", "user.read", "email", "openid", "profile"}

const (
	oAuthAuthURL   = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
	oAuthTokenURL  = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	oAuthDeviceURL = "https://login.microsoftonline.com/common/oauth2/v2.0/devicecode"
	rootUrl        = "https://graph.microsoft.com/v1.0/"
)

var (
	customAuthURL   = oAuthAuthURL
	customTokenURL  = oAuthTokenURL
	customDeviceURL = oAuthDeviceURL
	customRootURL   = rootUrl
)

// Token represents an OAuth2 Token and is the canonical representation
// used by the SDK.
type Token oauth2.Token

// Client is a stateful client for interacting with the Microsoft Graph API.
// It automatically handles token refreshes and persistence via a callback.
type Client struct {
	httpClient *http.Client
	onNewToken func(*Token) error
	logger     Logger
}

// SetLogger allows users of the SDK to set their own logger
func (c *Client) SetLogger(l Logger) {
	c.logger = l
}

// NewClient creates a new OneDrive client.
// It takes an initial token and a callback function that is invoked
// whenever a new token is generated after a refresh.
func NewClient(ctx context.Context, initialToken *Token, clientID string, onNewToken func(*Token) error, logger Logger) *Client {
	// The config can be minimal here because we are not using it to get a token,
	// only to configure the TokenSource for refresh operations.
	config := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  customAuthURL,
			TokenURL: customTokenURL,
		},
		Scopes: oAuthScopes,
	}

	persistingSource := &persistingTokenSource{
		source:     config.TokenSource(ctx, (*oauth2.Token)(initialToken)),
		onNewToken: onNewToken,
		lastToken:  (*oauth2.Token)(initialToken),
	}

	if logger == nil {
		logger = DefaultLogger{}
	}

	return &Client{
		httpClient: oauth2.NewClient(ctx, persistingSource),
		onNewToken: onNewToken,
		logger:     logger,
	}
}

// persistingTokenSource is a wrapper around an oauth2.TokenSource that
// invokes a callback to persist the token whenever it's refreshed.
type persistingTokenSource struct {
	source     oauth2.TokenSource
	onNewToken func(*Token) error
	mu         sync.Mutex // guards lastToken
	lastToken  *oauth2.Token
}

// Token returns a token from the underlying source and invokes the
// onNewToken callback if the access token has changed.
func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	// Get the new token from the underlying source. This may involve a refresh.
	newToken, err := s.source.Token()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// If the access token has changed, it means we have a new token.
	if s.lastToken == nil || s.lastToken.AccessToken != newToken.AccessToken {
		s.lastToken = newToken
		if s.onNewToken != nil {
			// Invoke the callback to persist the new token.
			if err := s.onNewToken((*Token)(newToken)); err != nil {
				// If persisting fails, we should probably just log it.
				// For now, we'll return an error to make the failure visible.
				return nil, fmt.Errorf("failed to persist new token: %w", err)
			}
		}
	}

	return newToken, nil
}

// BuildPathURL constructs the full API URL for a given path in the user's default drive.
func BuildPathURL(path string) string {
	// Root path special case
	if path == "" || path == "/" {
		return customRootURL + "me/drive/root"
	}
	// All other paths
	encodedPath := strings.TrimPrefix(path, "/")
	return customRootURL + "me/drive/root:/" + encodedPath
}

// GetMe retrieves the profile of the currently signed-in user.
func (c *Client) GetMe() (User, error) {
	c.logger.Debug("GetMe called")
	var user User

	url := customRootURL + "me"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return user, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return user, fmt.Errorf("decoding user failed: %v", err)
	}

	return user, nil
}

// Item-level methods moved to pkg/onedrive/item.go

// CreateFolder, UploadFile moved to pkg/onedrive/item.go

// Item management methods moved to pkg/onedrive/item.go

// GetSharedWithMe retrieves items that have been shared with the current user.
func (c *Client) GetSharedWithMe() (DriveItemList, error) {
	var items DriveItemList

	url := customRootURL + "me/drive/sharedWithMe"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding shared items failed: %v", err)
	}

	return items, nil
}

// GetRecentItems retrieves items that have been recently accessed by the current user.
func (c *Client) GetRecentItems() (DriveItemList, error) {
	var items DriveItemList

	url := customRootURL + "me/drive/recent"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding recent items failed: %v", err)
	}

	return items, nil
}

// GetSpecialFolder retrieves a special folder by its well-known name.
// Valid folder names: documents, photos, cameraroll, approot, music, recordings
func (c *Client) GetSpecialFolder(folderName string) (DriveItem, error) {
	var item DriveItem

	url := customRootURL + "me/drive/special/" + url.PathEscape(folderName)
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding special folder failed: %v", err)
	}

	return item, nil
}

// GetDelta gets changes to items in a drive using delta queries
func (c *Client) GetDelta(deltaToken string) (DeltaResponse, error) {
	var deltaResponse DeltaResponse

	url := customRootURL + "me/drive/root/delta"
	if deltaToken != "" {
		url += "?token=" + deltaToken
	}

	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return deltaResponse, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&deltaResponse); err != nil {
		return deltaResponse, fmt.Errorf("decoding delta response: %v", err)
	}

	return deltaResponse, nil
}

// GetFileVersions gets all versions of a file by its path
func (c *Client) GetFileVersions(filePath string) (DriveItemVersionList, error) {
	var versions DriveItemVersionList

	item, err := c.GetDriveItemByPath(filePath)
	if err != nil {
		return versions, fmt.Errorf("getting file item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + item.ID + "/versions"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return versions, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&versions); err != nil {
		return versions, fmt.Errorf("decoding versions response: %v", err)
	}

	return versions, nil
}

// collectAllPages is a helper to handle pagination for Graph API calls.
func (c *Client) collectAllPages(initialURL string, paging Paging) ([]json.RawMessage, string, error) {
	var allItems []json.RawMessage
	nextLink := initialURL

	if paging.NextLink != "" {
		nextLink = paging.NextLink
	}

	for nextLink != "" {
		res, err := c.apiCall("GET", nextLink, "", nil)
		if err != nil {
			return nil, "", err
		}
		defer res.Body.Close()

		var page struct {
			Value    []json.RawMessage `json:"value"`
			NextLink string            `json:"@odata.nextLink"`
		}

		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, "", fmt.Errorf("reading page body: %w", err)
		}

		if err := json.Unmarshal(bodyBytes, &page); err != nil {
			return nil, "", fmt.Errorf("decoding page: %v", err)
		}

		allItems = append(allItems, page.Value...)
		nextLink = page.NextLink

		if paging.FetchAll == false {
			break
		}
	}

	return allItems, nextLink, nil
}

// apiCall handles the HTTP request and categorizes common errors.
// It will automatically retry once on a 401 Unauthorized error.
func (c *Client) apiCall(method, url, contentType string, body io.ReadSeeker) (*http.Response, error) {
	var res *http.Response
	var err error

	for i := 0; i < 2; i++ {
		c.logger.Debug("apiCall invoked with method: ", method, ", URL: ", url)

		if c.httpClient == nil {
			return nil, ErrInternal
		}

		var req *http.Request
		req, err = http.NewRequest(method, url, body)
		if err != nil {
			return nil, fmt.Errorf("creating request failed: %v", err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		c.logger.Debug("Request created, sending request...")
		res, err = c.httpClient.Do(req)
		if err != nil {
			// This block handles network errors or errors from the token source during refresh.
			// It's unlikely to be a simple 401, so we fail fast.
			var oauth2RetrieveError *oauth2.RetrieveError
			if errors.As(err, &oauth2RetrieveError) {
				switch oauth2RetrieveError.ErrorCode {
				case "invalid_request", "invalid_client", "invalid_grant",
					"unauthorized_client", "unsupported_grant_type",
					"invalid_scope", "access_denied":
					return nil, fmt.Errorf("%w: %v", ErrReauthRequired, err)
				case "server_error", "temporarily_unavailable":
					return nil, fmt.Errorf("%w: %v", ErrRetryLater, err)
				default:
					return nil, fmt.Errorf("other oauth2 error: %v", err)
				}
			} else {
				// Likely a network error?
				return nil, fmt.Errorf("network error: %v", err)
			}
		}

		// If the status is not 401, we are done, break the loop.
		if res.StatusCode != http.StatusUnauthorized {
			break
		}

		// This was our first attempt (i=0), and we got a 401.
		// We will allow the loop to continue for a second attempt.
		// The oauth2 client is expected to refresh the token automatically.
		c.logger.Debug("Received 401 Unauthorized, attempting to refresh token and retry...")

		// Before retrying, we must close the previous response body to avoid leaks.
		res.Body.Close()

		// Also, rewind the request body so it can be read again for the retry.
		if body != nil {
			if _, err := body.Seek(0, io.SeekStart); err != nil {
				return nil, fmt.Errorf("failed to rewind request body for retry: %w", err)
			}
		}

		// If this was the second attempt (i=1) and we still got a 401, break the loop and fail.
		if i == 1 {
			c.logger.Debug("Still received 401 after retry, failing.")
			break
		}
	}

	// After the loop, check the final response code.
	if res.StatusCode >= 400 {
		defer res.Body.Close()
		resBody, _ := io.ReadAll(res.Body)

		var oneDriveError struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		jsonErr := json.Unmarshal(resBody, &oneDriveError)

		if jsonErr == nil && oneDriveError.Error.Code != "" {
			switch oneDriveError.Error.Code {
			case "accessDenied":
				return nil, fmt.Errorf("%w: %s", ErrAccessDenied, oneDriveError.Error.Message)
			case "activityLimitReached":
				return nil, fmt.Errorf("%w: %s", ErrRetryLater, oneDriveError.Error.Message)
			case "itemNotFound":
				return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, oneDriveError.Error.Message)
			case "nameAlreadyExists":
				return nil, fmt.Errorf("%w: %s", ErrConflict, oneDriveError.Error.Message)
			case "invalidRange", "invalidRequest", "malwareDetected",
				"notAllowed", "notSupported", "resourceModified",
				"resyncRequired", "generalException":
				return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, oneDriveError.Error.Message)
			case "quotaLimitReached":
				return nil, fmt.Errorf("%w: %s", ErrQuotaExceeded, oneDriveError.Error.Message)
			case "unauthenticated":
				return nil, fmt.Errorf("%w: %s", ErrReauthRequired, oneDriveError.Error.Message)
			case "serviceNotAvailable":
				return nil, fmt.Errorf("%w: %s", ErrRetryLater, oneDriveError.Error.Message)
			default:
				return nil, fmt.Errorf(
					"OneDrive error: %s - %s",
					res.Status,
					oneDriveError.Error.Message,
				)
			}
		} else {
			switch res.StatusCode {
			case http.StatusBadRequest, http.StatusMethodNotAllowed, http.StatusNotAcceptable,
				http.StatusLengthRequired, http.StatusPreconditionFailed,
				http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType,
				http.StatusRequestedRangeNotSatisfiable, http.StatusUnprocessableEntity:
				return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, oneDriveError.Error.Message)
			case http.StatusUnauthorized, http.StatusForbidden:
				return nil, fmt.Errorf("%w: %s", ErrReauthRequired, oneDriveError.Error.Message)
			case http.StatusGone, http.StatusNotFound:
				return nil, fmt.Errorf("%w: %s", ErrResourceNotFound, oneDriveError.Error.Message)
			case http.StatusConflict:
				return nil, fmt.Errorf("%w: %s", ErrConflict, oneDriveError.Error.Message)
			case http.StatusInsufficientStorage:
				return nil, fmt.Errorf("%w: %s", ErrQuotaExceeded, oneDriveError.Error.Message)
			case http.StatusNotImplemented,
				http.StatusTooManyRequests,
				http.StatusInternalServerError, http.StatusServiceUnavailable, 509:
				return nil, fmt.Errorf("%w: %s", ErrRetryLater, oneDriveError.Error.Message)
			default:
				return nil, fmt.Errorf("HTTP error: %s - %s", res.Status, oneDriveError.Error.Message)
			}
		}
	}

	return res, nil
}

// Sentinel errors
var (
	ErrReauthRequired        = errors.New("re-authentication required")
	ErrAccessDenied          = errors.New("access denied")
	ErrRetryLater            = errors.New("retry later")
	ErrInvalidRequest        = errors.New("invalid request")
	ErrResourceNotFound      = errors.New("resource not found")
	ErrConflict              = errors.New("conflict")
	ErrQuotaExceeded         = errors.New("quota exceeded")
	ErrAuthorizationPending  = errors.New("authorization pending")
	ErrAuthorizationDeclined = errors.New("authorization declined")
	ErrTokenExpired          = errors.New("token expired")
	ErrInternal              = errors.New("internal error")
)

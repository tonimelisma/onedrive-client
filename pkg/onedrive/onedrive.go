package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	cv "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"
)

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

// OAuthToken represents an OAuth2 Token.
type OAuthToken oauth2.Token

// OAuthConfig represents an OAuth2 Config.
type OAuthConfig oauth2.Config

// Custom token source to allow for caching refreshed tokens
// (golang oauth2 library issue #84, not fixed for 6 years and counting)

type customTokenSource struct {
	base           oauth2.TokenSource
	cachedToken    *oauth2.Token
	onTokenRefresh func(OAuthToken)
}

func (cts *customTokenSource) Token() (*oauth2.Token, error) {
	logger.Debug("Token called in customTokenSource")
	token, err := cts.base.Token()
	if err != nil {
		return nil, err
	}

	// Compare the new token with the cached token
	if cts.cachedToken == nil || token.AccessToken != cts.cachedToken.AccessToken {
		// Tokens are different, indicating a refresh
		if cts.onTokenRefresh != nil {
			cts.onTokenRefresh(OAuthToken(*token))
		}
		cts.cachedToken = token // Update the cached token
	}

	return token, nil
}

// Logger is the interface that the SDK uses for logging.

type Logger interface {
	Debug(v ...interface{})
	// Add more methods if needed
}

type DefaultLogger struct{}

// The Debug method by default is empty
func (l DefaultLogger) Debug(v ...interface{}) {}

// Instantiate the default logger
var logger Logger = DefaultLogger{}

// SetLogger allows users of the SDK to set their own logger
func SetLogger(l Logger) {
	logger = l
}

// apiCall handles the HTTP request and categorizes common errors.
func apiCall(client *http.Client, method, url, contentType string, body io.Reader) (*http.Response, error) {
	logger.Debug("apiCall invoked with method: ", method, ", URL: ", url)

	if client == nil {
		return nil, errors.New("HTTP client is nil, please provide a valid HTTP client")
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request failed: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	logger.Debug("Request created, sending request...")
	res, err := client.Do(req)
	if err != nil {
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

// BuildPathURL constructs the correct Microsoft Graph API URL for a given path.
func BuildPathURL(path string) string {
	if path == "" || path == "/" {
		return customRootURL + "me/drive/root"
	}
	// Trim leading/trailing slashes and encode the path
	trimmedPath := strings.Trim(path, "/")
	encodedPath := url.PathEscape(trimmedPath)
	return customRootURL + "me/drive/root:/" + encodedPath
}

// GetDriveItemByPath retrieves the metadata for a single drive item by its path.
func GetDriveItemByPath(client *http.Client, path string) (DriveItem, error) {
	logger.Debug("GetDriveItemByPath called with path: ", path)
	var item DriveItem

	url := BuildPathURL(path)
	res, err := apiCall(client, "GET", url, "", nil)
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding item failed: %v", err)
	}

	return item, nil
}

// GetDriveItemChildrenByPath retrieves the items in a specific folder by its path.
func GetDriveItemChildrenByPath(client *http.Client, path string) (DriveItemList, error) {
	logger.Debug("GetDriveItemChildrenByPath called with path: ", path)
	var items DriveItemList

	// For root, the URL is /children. For subfolders, it's :/children
	url := BuildPathURL(path)
	if url == customRootURL+"me/drive/root" {
		url += "/children"
	} else {
		url += ":/children"
	}

	res, err := apiCall(client, "GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding item list failed: %v", err)
	}

	return items, nil
}

// GetDrives retrieves the list of available drives for the user.
func GetDrives(client *http.Client) (DriveList, error) {
	logger.Debug("GetDrives called")
	var drives DriveList

	url := customRootURL + "me/drives"
	res, err := apiCall(client, "GET", url, "", nil)
	if err != nil {
		return drives, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drives); err != nil {
		return drives, fmt.Errorf("decoding drive list failed: %v", err)
	}

	return drives, nil
}

// GetDefaultDrive retrieves the default drive for the user, including quota information.
func GetDefaultDrive(client *http.Client) (Drive, error) {
	logger.Debug("GetDefaultDrive called")
	var drive Drive

	url := customRootURL + "me/drive"
	res, err := apiCall(client, "GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("decoding drive failed: %v", err)
	}

	return drive, nil
}

// GetRootDriveItems is deprecated and will be removed.
// Use GetDriveItemChildrenByPath(client, "/") instead.
func GetRootDriveItems(client *http.Client) (DriveItemList, error) {
	logger.Debug("GetRootDriveItems called")
	var items DriveItemList

	res, err := apiCall(client, "GET", customRootURL+"me/drive/root/children", "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return items, fmt.Errorf("couldn't parse body: %v", err)
	}

	err = json.Unmarshal(resBody, &items)
	if err != nil {
		return items, fmt.Errorf("unmarshalling json: %v", err)
	}

	return items, nil
}

// CreateFolder creates a new folder in the specified parent path.
func CreateFolder(client *http.Client, parentPath string, folderName string) (DriveItem, error) {
	logger.Debug("CreateFolder called with parentPath: ", parentPath, " folderName: ", folderName)
	var item DriveItem

	url := BuildPathURL(parentPath)
	if url == customRootURL+"me/drive/root" {
		url += "/children"
	} else {
		url += ":/children"
	}

	requestBody := map[string]interface{}{
		"name":                              folderName,
		"folder":                            map[string]interface{}{},
		"@microsoft.graph.conflictBehavior": "rename",
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return item, fmt.Errorf("marshalling create folder request: %w", err)
	}

	res, err := apiCall(client, "POST", url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding item failed: %v", err)
	}

	return item, nil
}

// UploadFile uploads a local file to the specified remote path.
func UploadFile(client *http.Client, localPath, remotePath string) (DriveItem, error) {
	logger.Debug("UploadFile called with localPath: ", localPath, ", remotePath: ", remotePath)
	var item DriveItem

	file, err := os.Open(localPath)
	if err != nil {
		return item, fmt.Errorf("opening local file: %w", err)
	}
	defer file.Close()

	// The URL for upload is /root:/path/to/folder/filename:/content
	url := BuildPathURL(remotePath) + ":/content"

	res, err := apiCall(client, "PUT", url, "application/octet-stream", file)
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding item failed: %v", err)
	}

	return item, nil
}

// CreateUploadSession creates a new upload session for a large file.
func CreateUploadSession(client *http.Client, remotePath string) (UploadSession, error) {
	logger.Debug("CreateUploadSession called with remotePath: ", remotePath)
	var session UploadSession

	url := BuildPathURL(remotePath) + ":/createUploadSession"

	requestBody := map[string]interface{}{
		"item": map[string]interface{}{
			"@microsoft.graph.conflictBehavior": "rename",
		},
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return session, fmt.Errorf("marshalling create session request: %w", err)
	}

	res, err := apiCall(client, "POST", url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return session, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("decoding session failed: %v", err)
	}

	return session, nil
}

// UploadChunk uploads a chunk of a file to the given upload URL.
// This function uses a standard http.Client because the upload URL is pre-authenticated
// and the Graph API expects no Authorization header on this request.
func UploadChunk(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (UploadSession, error) {
	logger.Debug("UploadChunk called for URL: ", uploadURL)
	var session UploadSession
	client := &http.Client{} // Use a standard client

	req, err := http.NewRequest("PUT", uploadURL, chunkData)
	if err != nil {
		return session, fmt.Errorf("creating upload chunk request failed: %v", err)
	}

	contentRange := fmt.Sprintf("bytes %d-%d/%d", startByte, endByte, totalSize)
	req.Header.Set("Content-Range", contentRange)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", endByte-startByte+1))

	res, err := client.Do(req)
	if err != nil {
		return session, fmt.Errorf("uploading chunk failed: %v", err)
	}
	defer res.Body.Close()

	// On success, the response can be 202 Accepted (more chunks to come),
	// 201 Created (final chunk for a new file), or 200 OK (final chunk for an existing file).
	if res.StatusCode < 200 || res.StatusCode > 202 {
		resBody, _ := io.ReadAll(res.Body)
		return session, fmt.Errorf("upload chunk failed with status %s: %s", res.Status, string(resBody))
	}

	// The response on success contains the nextExpectedRanges
	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		// For the very last chunk, the response body is the completed DriveItem,
		// not an UploadSession. We can treat this as a non-fatal error.
		return session, nil
	}

	return session, nil
}

// GetUploadSessionStatus gets the status of an existing upload session.
func GetUploadSessionStatus(uploadURL string) (UploadSession, error) {
	logger.Debug("GetUploadSessionStatus called for URL: ", uploadURL)
	var session UploadSession
	client := &http.Client{} // Use a standard client

	req, err := http.NewRequest("GET", uploadURL, nil)
	if err != nil {
		return session, fmt.Errorf("creating get status request failed: %v", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return session, fmt.Errorf("getting upload status failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return session, fmt.Errorf("get upload status failed with status %s", res.Status)
	}

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("decoding session status failed: %v", err)
	}

	return session, nil
}

// CancelUploadSession cancels an existing upload session.
func CancelUploadSession(uploadURL string) error {
	logger.Debug("CancelUploadSession called for URL: ", uploadURL)
	client := &http.Client{} // Use a standard client

	req, err := http.NewRequest("DELETE", uploadURL, nil)
	if err != nil {
		return fmt.Errorf("creating cancel request failed: %v", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cancelling upload session failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("cancel upload session failed with status %s", res.Status)
	}

	return nil
}

// DownloadFile downloads a file from OneDrive to a local path.
// It handles the 302 redirect that Microsoft Graph API returns for download requests.
func DownloadFile(client *http.Client, remotePath, localPath string) error {
	logger.Debug("DownloadFile called with remotePath: ", remotePath, ", localPath: ", localPath)

	url := BuildPathURL(remotePath) + ":/content"
	logger.Debug("Download URL constructed: ", url)

	// Create request but don't follow redirects automatically
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	// Get the response which should be a 302 redirect
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer res.Body.Close()

	logger.Debug("Download response status: ", res.StatusCode)

	// Handle 302 redirect to pre-authenticated download URL
	if res.StatusCode == http.StatusFound {
		downloadURL := res.Header.Get("Location")
		if downloadURL == "" {
			return fmt.Errorf("no download URL found in redirect response")
		}

		logger.Debug("Following redirect to: ", downloadURL)
		// Download from the pre-authenticated URL (no auth headers needed)
		return downloadFromURL(downloadURL, localPath)
	}

	// If we get 401 Unauthorized, try the alternative method using item metadata
	if res.StatusCode == http.StatusUnauthorized {
		logger.Debug("Got 401 on direct download, trying alternative method with item metadata")
		return DownloadFileByItem(client, remotePath, localPath)
	}

	// If we get 404 Not Found, try the alternative method using item metadata
	if res.StatusCode == http.StatusNotFound {
		logger.Debug("Got 404 on direct download, trying alternative method with item metadata")
		return DownloadFileByItem(client, remotePath, localPath)
	}

	// If it's not a redirect, handle as direct download (fallback)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", res.Status)
	}

	return saveResponseToFile(res, localPath)
}

// DownloadFileByItem downloads a file using the DriveItem's download URL.
// This is an alternative method that gets the download URL from item metadata first.
func DownloadFileByItem(client *http.Client, remotePath, localPath string) error {
	logger.Debug("DownloadFileByItem called with remotePath: ", remotePath, ", localPath: ", localPath)

	// First get the item metadata to get the download URL
	item, err := GetDriveItemByPath(client, remotePath)
	if err != nil {
		return fmt.Errorf("getting item metadata for download: %w", err)
	}

	if item.DownloadURL == "" {
		return fmt.Errorf("no download URL available for item: %s", remotePath)
	}

	// Download from the pre-authenticated URL
	return downloadFromURL(item.DownloadURL, localPath)
}

// downloadFromURL downloads a file from a URL to a local path (no authentication needed)
func downloadFromURL(url, localPath string) error {
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading from URL: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download from URL failed with status: %s", res.Status)
	}

	return saveResponseToFile(res, localPath)
}

// saveResponseToFile saves an HTTP response body to a local file
func saveResponseToFile(res *http.Response, localPath string) error {
	outFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating local file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, res.Body)
	if err != nil {
		return fmt.Errorf("copying content to local file: %w", err)
	}

	return nil
}

// DownloadFileChunk downloads a chunk of a file using range requests.
// This function works with pre-authenticated download URLs.
func DownloadFileChunk(client *http.Client, downloadURL string, startByte, endByte int64) (io.ReadCloser, error) {
	logger.Debug("DownloadFileChunk called for URL: ", downloadURL)

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating download chunk request failed: %v", err)
	}

	contentRange := fmt.Sprintf("bytes=%d-%d", startByte, endByte)
	req.Header.Set("Range", contentRange)

	res, err := http.DefaultClient.Do(req) // Use default client for pre-authenticated URLs
	if err != nil {
		return nil, fmt.Errorf("downloading chunk failed: %v", err)
	}

	if res.StatusCode != http.StatusPartialContent {
		defer res.Body.Close()
		return nil, fmt.Errorf("download chunk failed with status %s", res.Status)
	}

	return res.Body, nil
}

// StartAuthentication initiates the OAuth authentication process.
func StartAuthentication(
	ctx context.Context,
	oauthConfig *OAuthConfig,
) (authURL string, codeVerifier string, err error) {
	logger.Debug("StartAuthentication called")
	if ctx == nil {
		return "", "", errors.New("ctx is nil")
	}
	if oauthConfig == nil {
		return "", "", errors.New("oauth configuration is nil")
	}

	verifier, err := cv.CreateCodeVerifier()
	if err != nil {
		return "", "", fmt.Errorf("creating code verifier: %v", err)
	}

	// Creating a new oauth2.Config object that we'll cast to our type conversion
	// We maintain type conversion for oauth2 so users of the SDK don't have to import it
	nativeOAuthConfig := oauth2.Config(*oauthConfig)

	authURL = nativeOAuthConfig.AuthCodeURL(
		"state",
		oauth2.SetAuthURLParam("code_challenge", verifier.CodeChallengeS256()),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return authURL, verifier.String(), nil
}

// CompleteAuthentication completes the OAuth authentication process.
func CompleteAuthentication(
	ctx context.Context,
	oauthConfig *OAuthConfig,
	code string,
	verifier string,
) (*OAuthToken, error) {
	if oauthConfig == nil {
		return nil, errors.New("oauth configuration is nil")
	}

	logger.Debug("Exchanging code for token in CompleteAuthentication")

	// Creating a new oauth2.Config object that we'll cast to our type conversion
	// We maintain type conversion for oauth2 so users of the SDK don't have to import it
	nativeOAuthConfig := oauth2.Config(*oauthConfig)
	token, err := nativeOAuthConfig.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return nil, fmt.Errorf("exchanging code for token: %v", err)
	}

	oauthToken := OAuthToken(*token)
	return &oauthToken, nil
}

// NewClient creates a new HTTP client with the given OAuth token.
func NewClient(ctx context.Context, oauthConfig *OAuthConfig, token OAuthToken, tokenRefreshCallback func(OAuthToken)) *http.Client {
	if ctx == nil || oauthConfig == nil {
		return nil
	}

	// TODO Ensure the token is valid or initialized before using it

	// Creating a new oauth2.Config object that we'll cast to our type conversion
	// We maintain type conversion for oauth2 so users of the SDK don't have to import it
	nativeOAuthConfig := oauth2.Config(*oauthConfig)
	originalTokenSource := nativeOAuthConfig.TokenSource(ctx, (*oauth2.Token)(&token))
	customTokenSource := &customTokenSource{
		base:           originalTokenSource,
		onTokenRefresh: tokenRefreshCallback,
		cachedToken:    (*oauth2.Token)(&token),
	}

	return oauth2.NewClient(ctx, customTokenSource)
}

// GetOauth2Config returns the OAuth2 configuration.
func GetOauth2Config(clientID string) (context.Context, *OAuthConfig) {
	ctx := context.Background()
	oauthConfig := &oauth2.Config{
		ClientID: clientID,
		Scopes:   oAuthScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  customAuthURL,
			TokenURL: customTokenURL,
		},
		RedirectURL: "http://localhost:53682/",
	}

	return ctx, (*OAuthConfig)(oauthConfig)
}

// GetMe retrieves the profile of the currently signed-in user.
func GetMe(client *http.Client) (User, error) {
	logger.Debug("GetMe called")
	var user User

	url := customRootURL + "me"
	res, err := apiCall(client, "GET", url, "", nil)
	if err != nil {
		return user, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return user, fmt.Errorf("decoding user failed: %v", err)
	}

	return user, nil
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
		return nil, fmt.Errorf("decoding device code response failed: %w", err)
	}

	return &deviceCodeResponse, nil
}

// VerifyDeviceCode polls to verify the device code and get an access token.
func VerifyDeviceCode(clientID string, deviceCode string, debug bool) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", clientID)
	data.Set("device_code", deviceCode)

	res, err := apiCallWithDebug("POST", customTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()), debug)
	if err != nil {
		return nil, fmt.Errorf("verifying device code: %w", err)
	}
	defer res.Body.Close()

	var token OAuthToken
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding token failed: %v", err)
	}

	return &token, nil
}

// DeleteDriveItem deletes a file or folder by its path.
// Items are moved to the recycle bin, not permanently deleted.
func DeleteDriveItem(client *http.Client, path string) error {
	logger.Debug("DeleteDriveItem called with path: ", path)

	url := BuildPathURL(path)
	res, err := apiCall(client, "DELETE", url, "", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// DELETE returns 204 No Content on success
	if res.StatusCode != 204 {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}

// CopyDriveItem creates a copy of a DriveItem.
// Returns the monitoring URL for tracking the async copy operation.
func CopyDriveItem(client *http.Client, sourcePath, destinationParentPath, newName string) (string, error) {
	logger.Debug("CopyDriveItem called with sourcePath: ", sourcePath, " destinationParentPath: ", destinationParentPath, " newName: ", newName)

	url := BuildPathURL(sourcePath) + ":/copy"

	// Get the destination parent ID for the parentReference
	parentItem, err := GetDriveItemByPath(client, destinationParentPath)
	if err != nil {
		return "", fmt.Errorf("getting destination parent: %w", err)
	}

	requestBody := map[string]interface{}{
		"parentReference": map[string]interface{}{
			"id": parentItem.ID,
		},
	}

	// Add name if specified
	if newName != "" {
		requestBody["name"] = newName
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshalling copy request: %w", err)
	}

	res, err := apiCall(client, "POST", url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Copy returns 202 Accepted with Location header for monitoring
	if res.StatusCode != 202 {
		return "", fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	monitorURL := res.Header.Get("Location")
	if monitorURL == "" {
		return "", fmt.Errorf("no monitoring URL returned")
	}

	return monitorURL, nil
}

// MoveDriveItem moves a DriveItem to a new parent folder.
func MoveDriveItem(client *http.Client, sourcePath, destinationParentPath string) (DriveItem, error) {
	logger.Debug("MoveDriveItem called with sourcePath: ", sourcePath, " destinationParentPath: ", destinationParentPath)
	var item DriveItem

	url := BuildPathURL(sourcePath)

	// Get the destination parent ID for the parentReference
	parentItem, err := GetDriveItemByPath(client, destinationParentPath)
	if err != nil {
		return item, fmt.Errorf("getting destination parent: %w", err)
	}

	requestBody := map[string]interface{}{
		"parentReference": map[string]interface{}{
			"id": parentItem.ID,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return item, fmt.Errorf("marshalling move request: %w", err)
	}

	res, err := apiCall(client, "PATCH", url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding item failed: %v", err)
	}

	return item, nil
}

// UpdateDriveItem updates properties of a DriveItem, such as renaming it.
func UpdateDriveItem(client *http.Client, path, newName string) (DriveItem, error) {
	logger.Debug("UpdateDriveItem called with path: ", path, " newName: ", newName)
	var item DriveItem

	url := BuildPathURL(path)

	requestBody := map[string]interface{}{
		"name": newName,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return item, fmt.Errorf("marshalling update request: %w", err)
	}

	res, err := apiCall(client, "PATCH", url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding item failed: %v", err)
	}

	return item, nil
}

// MonitorCopyOperation checks the status of an async copy operation.
func MonitorCopyOperation(client *http.Client, monitorURL string) (CopyOperationStatus, error) {
	logger.Debug("MonitorCopyOperation called with monitorURL: ", monitorURL)
	var status CopyOperationStatus

	req, err := http.NewRequest("GET", monitorURL, nil)
	if err != nil {
		return status, fmt.Errorf("creating monitor request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return status, fmt.Errorf("monitoring copy operation: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 202 {
		// Operation still in progress
		status.Status = "inProgress"
		status.StatusDescription = "Copy operation is still in progress"

		// Try to get progress info from response if available
		if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
			// If we can't decode, just return the basic status
			return status, nil
		}
		return status, nil
	} else if res.StatusCode == 303 {
		// Operation completed successfully - redirect to the new resource
		status.Status = "completed"
		status.PercentageComplete = 100
		status.StatusDescription = "Copy operation completed successfully"
		status.ResourceLocation = res.Header.Get("Location")
		return status, nil
	} else if res.StatusCode >= 400 {
		// Operation failed
		status.Status = "failed"
		status.StatusDescription = "Copy operation failed"

		// Try to get error details
		var errorResponse struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := json.NewDecoder(res.Body).Decode(&errorResponse); err == nil {
			status.Error = &struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				Code:    errorResponse.Error.Code,
				Message: errorResponse.Error.Message,
			}
		}
		return status, nil
	}

	// Unexpected status code
	return status, fmt.Errorf("unexpected status code: %d", res.StatusCode)
}

// SearchDriveItems searches for items in the drive by query string.
func SearchDriveItems(client *http.Client, query string) (DriveItemList, error) {
	logger.Debug("SearchDriveItems called with query: ", query)
	var items DriveItemList

	// URL encode the query parameter
	encodedQuery := url.QueryEscape(query)
	searchURL := customRootURL + "me/drive/root/search(q='" + encodedQuery + "')"

	res, err := apiCall(client, "GET", searchURL, "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding search results failed: %v", err)
	}

	return items, nil
}

// GetSharedWithMe retrieves items that have been shared with the current user.
func GetSharedWithMe(client *http.Client) (DriveItemList, error) {
	logger.Debug("GetSharedWithMe called")
	var items DriveItemList

	sharedURL := customRootURL + "me/drive/sharedWithMe"

	res, err := apiCall(client, "GET", sharedURL, "", nil)
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
func GetRecentItems(client *http.Client) (DriveItemList, error) {
	logger.Debug("GetRecentItems called")
	var items DriveItemList

	recentURL := customRootURL + "me/drive/recent"

	res, err := apiCall(client, "GET", recentURL, "", nil)
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
func GetSpecialFolder(client *http.Client, folderName string) (DriveItem, error) {
	logger.Debug("GetSpecialFolder called with folderName: ", folderName)
	var item DriveItem

	// Validate folder name
	validFolders := map[string]bool{
		"documents":  true,
		"photos":     true,
		"cameraroll": true,
		"approot":    true,
		"music":      true,
		"recordings": true,
	}

	if !validFolders[folderName] {
		return item, fmt.Errorf("invalid special folder name: %s. Valid names are: documents, photos, cameraroll, approot, music, recordings", folderName)
	}

	specialURL := customRootURL + "me/drive/special/" + folderName

	res, err := apiCall(client, "GET", specialURL, "", nil)
	if err != nil {
		return item, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return item, fmt.Errorf("decoding special folder failed: %v", err)
	}

	return item, nil
}

// CreateSharingLink creates a sharing link for a DriveItem.
// linkType can be "view", "edit", or "embed"
// scope can be "anonymous" or "organization"
func CreateSharingLink(client *http.Client, path, linkType, scope string) (SharingLink, error) {
	logger.Debug("CreateSharingLink called with path: ", path, " linkType: ", linkType, " scope: ", scope)
	var link SharingLink

	// Validate link type
	validTypes := map[string]bool{
		"view":  true,
		"edit":  true,
		"embed": true,
	}
	if !validTypes[linkType] {
		return link, fmt.Errorf("invalid link type: %s. Valid types are: view, edit, embed", linkType)
	}

	// Validate scope
	validScopes := map[string]bool{
		"anonymous":    true,
		"organization": true,
	}
	if !validScopes[scope] {
		return link, fmt.Errorf("invalid scope: %s. Valid scopes are: anonymous, organization", scope)
	}

	url := BuildPathURL(path) + ":/createLink"

	requestBody := CreateLinkRequest{
		Type:  linkType,
		Scope: scope,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return link, fmt.Errorf("marshalling create link request: %w", err)
	}

	res, err := apiCall(client, "POST", url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return link, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&link); err != nil {
		return link, fmt.Errorf("decoding sharing link failed: %v", err)
	}

	return link, nil
}

// GetDelta gets changes to items in a drive using delta queries
func GetDelta(client *http.Client, deltaToken string) (DeltaResponse, error) {
	logger.Debug("GetDelta called with token: ", deltaToken)
	var deltaResponse DeltaResponse

	apiURL := customRootURL + "me/drive/root/delta"
	if deltaToken != "" {
		apiURL += "?token=" + url.QueryEscape(deltaToken)
	}

	res, err := apiCall(client, "GET", apiURL, "", nil)
	if err != nil {
		return deltaResponse, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&deltaResponse); err != nil {
		return deltaResponse, fmt.Errorf("decoding delta response failed: %v", err)
	}

	return deltaResponse, nil
}

// GetDriveByID gets metadata for a specific drive by its ID
func GetDriveByID(client *http.Client, driveID string) (Drive, error) {
	logger.Debug("GetDriveByID called with ID: ", driveID)
	var drive Drive

	apiURL := customRootURL + "drives/" + url.PathEscape(driveID)

	res, err := apiCall(client, "GET", apiURL, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("decoding drive failed: %v", err)
	}

	return drive, nil
}

// GetFileVersions gets all versions of a file by its path
func GetFileVersions(client *http.Client, filePath string) (DriveItemVersionList, error) {
	logger.Debug("GetFileVersions called with filePath: ", filePath)
	var versions DriveItemVersionList

	// First get the item to get its ID
	item, err := GetDriveItemByPath(client, filePath)
	if err != nil {
		return versions, fmt.Errorf("getting drive item: %w", err)
	}

	apiURL := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/versions"

	res, err := apiCall(client, "GET", apiURL, "", nil)
	if err != nil {
		return versions, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&versions); err != nil {
		return versions, fmt.Errorf("decoding file versions failed: %v", err)
	}

	return versions, nil
}

// collectAllPages is a helper function that follows @odata.nextLink to collect all pages
func collectAllPages(client *http.Client, initialURL string, paging Paging) ([]json.RawMessage, string, error) {
	var allItems []json.RawMessage
	currentURL := initialURL

	// If NextLink is provided, start from there
	if paging.NextLink != "" {
		currentURL = paging.NextLink
	}

	// Add $top parameter if specified
	if paging.Top > 0 && paging.NextLink == "" {
		separator := "?"
		if strings.Contains(currentURL, "?") {
			separator = "&"
		}
		currentURL += separator + "$top=" + fmt.Sprintf("%d", paging.Top)
	}

	for {
		res, err := apiCall(client, "GET", currentURL, "", nil)
		if err != nil {
			return nil, "", err
		}
		defer res.Body.Close()

		var response struct {
			Value    []json.RawMessage `json:"value"`
			NextLink string            `json:"@odata.nextLink"`
		}

		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			return nil, "", fmt.Errorf("decoding response failed: %v", err)
		}

		allItems = append(allItems, response.Value...)

		// If not fetching all pages, or no more pages, break
		if !paging.FetchAll || response.NextLink == "" {
			return allItems, response.NextLink, nil
		}

		currentURL = response.NextLink
	}
}

// DownloadFileAsFormat downloads a file from OneDrive in a specific format.
func DownloadFileAsFormat(client *http.Client, remotePath, localPath, format string) error {
	logger.Debug("DownloadFileAsFormat called with remotePath: ", remotePath, ", localPath: ", localPath, ", format: ", format)

	url := BuildPathURL(remotePath) + ":/content?format=" + url.QueryEscape(format)
	logger.Debug("Download URL with format constructed: ", url)

	// Create request but don't follow redirects automatically
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	// Get the response which should be a 302 redirect
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer res.Body.Close()

	logger.Debug("Download response status: ", res.StatusCode)

	// Handle 302 redirect to pre-authenticated download URL
	if res.StatusCode == http.StatusFound {
		downloadURL := res.Header.Get("Location")
		if downloadURL == "" {
			return fmt.Errorf("no download URL found in redirect response")
		}

		logger.Debug("Following redirect to: ", downloadURL)
		// Download from the pre-authenticated URL (no auth headers needed)
		return downloadFromURL(downloadURL, localPath)
	}

	// If it's not a redirect, handle as direct download (fallback)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", res.Status)
	}

	return saveResponseToFile(res, localPath)
}

// SearchDriveItemsInFolder searches for items within a specific folder.
func SearchDriveItemsInFolder(client *http.Client, folderPath, query string, paging Paging) (DriveItemList, string, error) {
	logger.Debug("SearchDriveItemsInFolder called with folderPath: ", folderPath, ", query: ", query)
	var items DriveItemList

	// First get the folder item to get its ID
	folderItem, err := GetDriveItemByPath(client, folderPath)
	if err != nil {
		return items, "", fmt.Errorf("getting folder item: %w", err)
	}

	// URL encode the query parameter
	encodedQuery := url.QueryEscape(query)
	searchURL := customRootURL + "me/drive/items/" + url.PathEscape(folderItem.ID) + "/search(q='" + encodedQuery + "')"

	rawItems, nextLink, err := collectAllPages(client, searchURL, paging)
	if err != nil {
		return items, "", err
	}

	// Convert raw messages to DriveItems
	for _, rawItem := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(rawItem, &item); err != nil {
			return items, "", fmt.Errorf("unmarshaling drive item: %v", err)
		}
		items.Value = append(items.Value, item)
	}

	return items, nextLink, nil
}

// GetDriveActivities retrieves activities for the entire drive.
func GetDriveActivities(client *http.Client, paging Paging) (ActivityList, string, error) {
	logger.Debug("GetDriveActivities called")
	var activities ActivityList

	activitiesURL := customRootURL + "me/drive/activities"

	rawItems, nextLink, err := collectAllPages(client, activitiesURL, paging)
	if err != nil {
		return activities, "", err
	}

	// Convert raw messages to Activities
	for _, rawItem := range rawItems {
		var activity Activity
		if err := json.Unmarshal(rawItem, &activity); err != nil {
			return activities, "", fmt.Errorf("unmarshaling activity: %v", err)
		}
		activities.Value = append(activities.Value, activity)
	}

	return activities, nextLink, nil
}

// GetItemActivities retrieves activities for a specific item.
func GetItemActivities(client *http.Client, remotePath string, paging Paging) (ActivityList, string, error) {
	logger.Debug("GetItemActivities called with remotePath: ", remotePath)
	var activities ActivityList

	// First get the item to get its ID
	item, err := GetDriveItemByPath(client, remotePath)
	if err != nil {
		return activities, "", fmt.Errorf("getting drive item: %w", err)
	}

	activitiesURL := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/activities"

	rawItems, nextLink, err := collectAllPages(client, activitiesURL, paging)
	if err != nil {
		return activities, "", err
	}

	// Convert raw messages to Activities
	for _, rawItem := range rawItems {
		var activity Activity
		if err := json.Unmarshal(rawItem, &activity); err != nil {
			return activities, "", fmt.Errorf("unmarshaling activity: %v", err)
		}
		activities.Value = append(activities.Value, activity)
	}

	return activities, nextLink, nil
}

// SearchDriveItemsWithPaging searches for items in the drive with paging support.
func SearchDriveItemsWithPaging(client *http.Client, query string, paging Paging) (DriveItemList, string, error) {
	logger.Debug("SearchDriveItemsWithPaging called with query: ", query)
	var items DriveItemList

	// URL encode the query parameter
	encodedQuery := url.QueryEscape(query)
	searchURL := customRootURL + "me/drive/root/search(q='" + encodedQuery + "')"

	rawItems, nextLink, err := collectAllPages(client, searchURL, paging)
	if err != nil {
		return items, "", err
	}

	// Convert raw messages to DriveItems
	for _, rawItem := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(rawItem, &item); err != nil {
			return items, "", fmt.Errorf("unmarshaling drive item: %v", err)
		}
		items.Value = append(items.Value, item)
	}

	return items, nextLink, nil
}

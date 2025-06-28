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

// OAuthToken represents an OAuth2 Token.
type OAuthToken oauth2.Token

// OAuthConfig represents an OAuth2 Config.
type OAuthConfig oauth2.Config

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

// buildPathURL constructs the correct Microsoft Graph API URL for a given path.
func buildPathURL(path string) string {
	if path == "" || path == "/" {
		return rootUrl + "me/drive/root"
	}
	// Trim leading/trailing slashes and encode the path
	trimmedPath := strings.Trim(path, "/")
	encodedPath := url.PathEscape(trimmedPath)
	return rootUrl + "me/drive/root:/" + encodedPath + ":"
}

// GetDriveItemByPath retrieves the metadata for a single drive item by its path.
func GetDriveItemByPath(client *http.Client, path string) (DriveItem, error) {
	logger.Debug("GetDriveItemByPath called with path: ", path)
	var item DriveItem

	url := buildPathURL(path)
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

	// For root, the URL is /children. For subfolders, it's /:/children
	url := buildPathURL(path)
	if url == rootUrl+"me/drive/root" {
		url += "/children"
	} else {
		url += "/children"
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

	url := rootUrl + "me/drives"
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

	url := rootUrl + "me/drive"
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

	res, err := apiCall(client, "GET", rootUrl+"me/drive/root/children", "", nil)
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

	url := buildPathURL(parentPath)
	if url == rootUrl+"me/drive/root" {
		url += "/children"
	} else {
		url += "/children"
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
	url := buildPathURL(remotePath) + ":/content"

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

	url := buildPathURL(remotePath) + ":/createUploadSession"

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

// DownloadFile downloads a remote file to the specified local path.
func DownloadFile(client *http.Client, remotePath, localPath string) error {
	logger.Debug("DownloadFile called with remotePath: ", remotePath, ", localPath: ", localPath)

	url := buildPathURL(remotePath) + "/content"
	res, err := apiCall(client, "GET", url, "", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

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
	logger.Debug("Creating OAuth2 configuration in getOauth2Config")
	if clientID == "" {
		return nil, nil
	}

	// Creating a new oauth2.Config object that we'll cast to our type conversion
	// We maintain type conversion for oauth2 so users of the SDK don't have to import it
	oauth2Config := oauth2.Config{
		ClientID: clientID,
		Scopes:   oAuthScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  oAuthAuthURL,
			TokenURL: oAuthTokenURL,
		},
	}
	return context.Background(), (*OAuthConfig)(&oauth2Config)
}

// GetMe retrieves the profile of the currently signed-in user.
func GetMe(client *http.Client) (User, error) {
	logger.Debug("GetMe called")
	var user User

	url := rootUrl + "me"
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

	res, err := apiCallWithDebug("POST", oAuthDeviceURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()), debug)
	if err != nil {
		return nil, err
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

	res, err := apiCallWithDebug("POST", oAuthTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()), debug)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var token OAuthToken
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("decoding token failed: %v", err)
	}

	return &token, nil
}

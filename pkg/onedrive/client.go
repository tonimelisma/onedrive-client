package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
func NewClient(ctx context.Context, initialToken *Token, onNewToken func(*Token) error, logger Logger) *Client {
	// The config can be minimal here because we are not using it to get a token,
	// only to configure the TokenSource for refresh operations.
	config := &oauth2.Config{
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

// GetDrives retrieves the list of available drives for the user.
func (c *Client) GetDrives() (DriveList, error) {
	c.logger.Debug("GetDrives called")
	var drives DriveList

	url := customRootURL + "me/drives"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return drives, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drives); err != nil {
		return drives, fmt.Errorf("decoding drives failed: %v", err)
	}

	return drives, nil
}

// GetDefaultDrive retrieves the default drive for the user, including quota information.
func (c *Client) GetDefaultDrive() (Drive, error) {
	c.logger.Debug("GetDefaultDrive called")
	var drive Drive

	url := customRootURL + "me/drive"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("decoding drive failed: %v", err)
	}

	return drive, nil
}

// GetDriveByID gets metadata for a specific drive by its ID
func (c *Client) GetDriveByID(driveID string) (Drive, error) {
	c.logger.Debug("GetDriveByID called with ID: ", driveID)
	var drive Drive

	url := customRootURL + "drives/" + url.PathEscape(driveID)
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return drive, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&drive); err != nil {
		return drive, fmt.Errorf("decoding drive failed: %v", err)
	}

	return drive, nil
}

// GetDriveItemByPath retrieves the metadata for a single drive item by its path.
func (c *Client) GetDriveItemByPath(path string) (DriveItem, error) {
	var item DriveItem

	url := BuildPathURL(path)
	res, err := c.apiCall("GET", url, "", nil)
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
func (c *Client) GetDriveItemChildrenByPath(path string) (DriveItemList, error) {
	var items DriveItemList

	// For root, the URL is /children. For subfolders, it's :/children
	url := BuildPathURL(path)
	if url == customRootURL+"me/drive/root" {
		url += "/children"
	} else {
		url += ":/children"
	}

	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return items, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		return items, fmt.Errorf("decoding item list failed: %v", err)
	}

	return items, nil
}

// GetRootDriveItems is deprecated and will be removed.
// Use GetDriveItemChildrenByPath(client, "/") instead.
func (c *Client) GetRootDriveItems() (DriveItemList, error) {
	var items DriveItemList

	res, err := c.apiCall("GET", customRootURL+"me/drive/root/children", "", nil)
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
func (c *Client) CreateFolder(parentPath string, folderName string) (DriveItem, error) {
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

	res, err := c.apiCall("POST", url, "application/json", strings.NewReader(string(jsonBody)))
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
func (c *Client) UploadFile(localPath, remotePath string) (DriveItem, error) {
	var item DriveItem

	file, err := os.Open(localPath)
	if err != nil {
		return item, fmt.Errorf("opening local file: %w", err)
	}
	defer file.Close()

	url := BuildPathURL(remotePath) + ":/content"
	res, err := c.apiCall("PUT", url, "application/octet-stream", file)
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
func (c *Client) CreateUploadSession(remotePath string) (UploadSession, error) {
	var session UploadSession

	url := BuildPathURL(remotePath) + ":/createUploadSession"
	res, err := c.apiCall("POST", url, "application/json", nil)
	if err != nil {
		return session, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("decoding upload session failed: %v", err)
	}

	return session, nil
}

// UploadChunk uploads a chunk of a large file using an upload session.
// Note: this uses a standard http client because it's a pre-authenticated URL
// and the Graph API expects no Authorization header on this request.
func (c *Client) UploadChunk(uploadURL string, startByte, endByte, totalSize int64, chunkData io.Reader) (UploadSession, error) {
	var session UploadSession
	client := &http.Client{} // Use a standard client

	req, err := http.NewRequest("PUT", uploadURL, chunkData)
	if err != nil {
		return session, fmt.Errorf("creating chunk upload request: %w", err)
	}
	req.Header.Set("Content-Length", fmt.Sprintf("%d", endByte-startByte+1))
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startByte, endByte, totalSize))

	res, err := client.Do(req)
	if err != nil {
		return session, fmt.Errorf("uploading chunk: %w", err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("decoding upload session response: %v", err)
	}

	return session, nil
}

// GetUploadSessionStatus gets the status of an existing upload session.
func (c *Client) GetUploadSessionStatus(uploadURL string) (UploadSession, error) {
	var session UploadSession
	client := &http.Client{} // Use a standard client

	res, err := client.Get(uploadURL)
	if err != nil {
		return session, fmt.Errorf("getting upload session status: %w", err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return session, fmt.Errorf("decoding upload session status: %v", err)
	}

	return session, nil
}

// CancelUploadSession cancels an existing upload session.
func (c *Client) CancelUploadSession(uploadURL string) error {
	client := &http.Client{} // Use a standard client

	req, err := http.NewRequest("DELETE", uploadURL, nil)
	if err != nil {
		return fmt.Errorf("creating cancel request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("canceling upload session: %w", err)
	}
	defer res.Body.Close()

	return nil
}

// DownloadFile downloads a file from OneDrive.
// It handles the 302 redirect that Microsoft Graph API returns for download requests.
func (c *Client) DownloadFile(remotePath, localPath string) error {
	url := BuildPathURL(remotePath) + ":/content"

	// Create request but don't follow redirects automatically
	// The http client from oauth2 follows redirects by default, so we need a new one for this.
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		// Copy transport from authenticated client
		Transport: c.httpClient.Transport,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	res, err := noRedirectClient.Do(req)
	if err != nil {
		return fmt.Errorf("initiating download: %w", err)
	}
	defer res.Body.Close()

	// Handle 302 redirect to pre-authenticated download URL
	if res.StatusCode == http.StatusFound {
		downloadURL := res.Header.Get("Location")
		if downloadURL == "" {
			return fmt.Errorf("no download location in redirect header")
		}

		// Download from the pre-authenticated URL (no auth headers needed)
		return c.downloadFromURL(downloadURL, localPath)
	}

	// If we get 401 Unauthorized, try the alternative method using item metadata
	if res.StatusCode == http.StatusUnauthorized {
		return c.DownloadFileByItem(remotePath, localPath)
	}

	// If we get 404 Not Found, try the alternative method using item metadata
	if res.StatusCode == http.StatusNotFound {
		return c.DownloadFileByItem(remotePath, localPath)
	}

	// If not a redirect, assume it's the file content and save it
	return saveResponseToFile(res, localPath)
}

// DownloadFileByItem downloads a file by first getting its metadata.
// This is an alternative method that gets the download URL from item metadata first.
func (c *Client) DownloadFileByItem(remotePath, localPath string) error {
	// First get the item metadata to get the download URL
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return fmt.Errorf("getting item metadata for download: %w", err)
	}

	if item.DownloadURL == "" {
		return fmt.Errorf("item has no download URL")
	}

	return c.downloadFromURL(item.DownloadURL, localPath)
}

// downloadFromURL downloads a file from a URL (typically pre-authenticated).
func (c *Client) downloadFromURL(url, localPath string) error {
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading from URL: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", res.Status)
	}

	return saveResponseToFile(res, localPath)
}

// saveResponseToFile saves an HTTP response body to a local file.
func saveResponseToFile(res *http.Response, localPath string) error {
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating local file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return fmt.Errorf("saving to local file: %w", err)
	}

	return nil
}

// DownloadFileChunk downloads a specific chunk of a file.
func (c *Client) DownloadFileChunk(downloadURL string, startByte, endByte int64) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating chunk download request: %w", err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startByte, endByte))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading chunk: %w", err)
	}

	if res.StatusCode != http.StatusPartialContent {
		defer res.Body.Close()
		return nil, fmt.Errorf("unexpected status code for chunk download: %d", res.StatusCode)
	}

	return res.Body, nil
}

// DeleteDriveItem deletes a file or folder from OneDrive.
func (c *Client) DeleteDriveItem(path string) error {
	url := BuildPathURL(path)
	res, err := c.apiCall("DELETE", url, "", nil)
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
func (c *Client) CopyDriveItem(sourcePath, destinationParentPath, newName string) (string, error) {
	url := BuildPathURL(sourcePath) + ":/copy"

	// Get the destination parent ID for the parentReference
	parentItem, err := c.GetDriveItemByPath(destinationParentPath)
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

	res, err := c.apiCall("POST", url, "application/json", strings.NewReader(string(jsonBody)))
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
func (c *Client) MoveDriveItem(sourcePath, destinationParentPath string) (DriveItem, error) {
	var item DriveItem

	url := BuildPathURL(sourcePath)

	// Get the destination parent ID for the parentReference
	parentItem, err := c.GetDriveItemByPath(destinationParentPath)
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

	res, err := c.apiCall("PATCH", url, "application/json", strings.NewReader(string(jsonBody)))
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
func (c *Client) UpdateDriveItem(path, newName string) (DriveItem, error) {
	var item DriveItem

	url := BuildPathURL(path)

	requestBody := map[string]interface{}{
		"name": newName,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return item, fmt.Errorf("marshalling update request: %w", err)
	}

	res, err := c.apiCall("PATCH", url, "application/json", strings.NewReader(string(jsonBody)))
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
func (c *Client) MonitorCopyOperation(monitorURL string) (CopyOperationStatus, error) {
	var status CopyOperationStatus

	req, err := http.NewRequest("GET", monitorURL, nil)
	if err != nil {
		return status, fmt.Errorf("creating monitor request: %w", err)
	}

	res, err := c.httpClient.Do(req)
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
		body, _ := io.ReadAll(res.Body)
		status.StatusDescription = fmt.Sprintf("Copy operation failed with status: %s, %s", res.Status, string(body))
		json.Unmarshal(body, &status)
		return status, fmt.Errorf("copy operation failed: %s", res.Status)
	}

	// For other statuses, just decode the response
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		return status, fmt.Errorf("decoding monitor response: %v", err)
	}

	return status, nil
}

// SearchDriveItems searches for items in the drive by query string.
func (c *Client) SearchDriveItems(query string) (DriveItemList, error) {
	var items DriveItemList

	url := customRootURL + "me/drive/root/search(q='" + url.QueryEscape(query) + "')"
	res, err := c.apiCall("GET", url, "", nil)
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

// CreateSharingLink creates a sharing link for a file or folder.
// linkType can be "view", "edit", or "embed"
// scope can be "anonymous" or "organization"
func (c *Client) CreateSharingLink(path, linkType, scope string) (SharingLink, error) {
	var link SharingLink

	url := BuildPathURL(path) + ":/createLink"
	requestBody := map[string]interface{}{
		"type":  linkType,
		"scope": scope,
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return link, fmt.Errorf("marshalling sharing link request: %w", err)
	}

	res, err := c.apiCall("POST", url, "application/json", strings.NewReader(string(jsonBody)))
	if err != nil {
		return link, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&link); err != nil {
		return link, fmt.Errorf("decoding sharing link response: %v", err)
	}

	return link, nil
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

// DownloadFileAsFormat downloads a file from OneDrive in a specific format.
func (c *Client) DownloadFileAsFormat(remotePath, localPath, format string) error {
	url := BuildPathURL(remotePath) + ":/content?format=" + url.QueryEscape(format)

	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: c.httpClient.Transport,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	res, err := noRedirectClient.Do(req)
	if err != nil {
		return fmt.Errorf("initiating download: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusFound {
		downloadURL := res.Header.Get("Location")
		if downloadURL == "" {
			return fmt.Errorf("no download location in redirect header")
		}
		return c.downloadFromURL(downloadURL, localPath)
	}

	return saveResponseToFile(res, localPath)
}

// SearchDriveItemsInFolder searches for items within a specific folder.
func (c *Client) SearchDriveItemsInFolder(folderPath, query string, paging Paging) (DriveItemList, string, error) {
	var items DriveItemList

	folder, err := c.GetDriveItemByPath(folderPath)
	if err != nil {
		return items, "", fmt.Errorf("getting folder item: %w", err)
	}

	initialURL := fmt.Sprintf("%sme/drives/%s/items/%s/search(q='%s')", customRootURL, folder.ParentReference.DriveID, folder.ID, url.QueryEscape(query))
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("&$top=%d", paging.Top)
	}

	rawItems, nextLink, err := c.collectAllPages(initialURL, paging)
	if err != nil {
		return items, "", err
	}

	for _, raw := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return items, "", fmt.Errorf("decoding item: %w", err)
		}
		items.Value = append(items.Value, item)
	}

	return items, nextLink, nil
}

// GetDriveActivities retrieves activities for the entire drive.
func (c *Client) GetDriveActivities(paging Paging) (ActivityList, string, error) {
	var activities ActivityList
	initialURL := customRootURL + "me/drive/activities"
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("?$top=%d", paging.Top)
	}

	rawActivities, nextLink, err := c.collectAllPages(initialURL, paging)
	if err != nil {
		return activities, "", err
	}

	for _, raw := range rawActivities {
		var activity Activity
		if err := json.Unmarshal(raw, &activity); err != nil {
			return activities, "", fmt.Errorf("decoding activity: %w", err)
		}
		activities.Value = append(activities.Value, activity)
	}

	return activities, nextLink, nil
}

// GetItemActivities retrieves activities for a specific item.
func (c *Client) GetItemActivities(remotePath string, paging Paging) (ActivityList, string, error) {
	var activities ActivityList
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return activities, "", fmt.Errorf("getting item: %w", err)
	}
	initialURL := fmt.Sprintf("%sme/drive/items/%s/activities", customRootURL, item.ID)
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("?$top=%d", paging.Top)
	}

	rawActivities, nextLink, err := c.collectAllPages(initialURL, paging)
	if err != nil {
		return activities, "", err
	}

	for _, raw := range rawActivities {
		var activity Activity
		if err := json.Unmarshal(raw, &activity); err != nil {
			return activities, "", fmt.Errorf("decoding activity: %w", err)
		}
		activities.Value = append(activities.Value, activity)
	}

	return activities, nextLink, nil
}

// SearchDriveItemsWithPaging searches for items in the drive with paging support.
func (c *Client) SearchDriveItemsWithPaging(query string, paging Paging) (DriveItemList, string, error) {
	var items DriveItemList
	initialURL := fmt.Sprintf("%sme/drive/root/search(q='%s')", customRootURL, url.QueryEscape(query))
	if paging.Top > 0 {
		initialURL += fmt.Sprintf("&$top=%d", paging.Top)
	}

	rawItems, nextLink, err := c.collectAllPages(initialURL, paging)
	if err != nil {
		return items, "", err
	}

	for _, raw := range rawItems {
		var item DriveItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return items, "", fmt.Errorf("decoding item: %w", err)
		}
		items.Value = append(items.Value, item)
	}

	return items, nextLink, nil
}

// GetThumbnails retrieves thumbnail images for a drive item.
func (c *Client) GetThumbnails(remotePath string) (ThumbnailSetList, error) {
	var thumbnails ThumbnailSetList
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return thumbnails, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/thumbnails"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return thumbnails, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&thumbnails); err != nil {
		return thumbnails, fmt.Errorf("decoding thumbnails response: %v", err)
	}

	return thumbnails, nil
}

// GetThumbnailBySize retrieves a specific size thumbnail for a drive item.
func (c *Client) GetThumbnailBySize(remotePath, thumbID, size string) (Thumbnail, error) {
	var thumbnail Thumbnail
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return thumbnail, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/thumbnails/" + url.PathEscape(thumbID) + "/" + url.PathEscape(size)
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return thumbnail, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&thumbnail); err != nil {
		return thumbnail, fmt.Errorf("decoding thumbnail response: %v", err)
	}

	return thumbnail, nil
}

// PreviewItem creates a preview for a drive item.
func (c *Client) PreviewItem(remotePath string, request PreviewRequest) (PreviewResponse, error) {
	var preview PreviewResponse
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return preview, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/preview"
	var bodyReader io.ReadSeeker
	if request.Page != "" || request.Zoom != 0 {
		requestBody, err := json.Marshal(request)
		if err != nil {
			return preview, fmt.Errorf("marshaling preview request: %v", err)
		}
		bodyReader = bytes.NewReader(requestBody)
	}

	res, err := c.apiCall("POST", url, "application/json", bodyReader)
	if err != nil {
		return preview, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&preview); err != nil {
		return preview, fmt.Errorf("decoding preview response: %v", err)
	}

	return preview, nil
}

// InviteUsers invites users to access a drive item.
func (c *Client) InviteUsers(remotePath string, request InviteRequest) (InviteResponse, error) {
	var invite InviteResponse
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return invite, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/invite"
	requestBody, err := json.Marshal(request)
	if err != nil {
		return invite, fmt.Errorf("marshaling invite request: %v", err)
	}

	res, err := c.apiCall("POST", url, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return invite, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&invite); err != nil {
		return invite, fmt.Errorf("decoding invite response: %v", err)
	}

	return invite, nil
}

// ListPermissions lists all permissions on a drive item.
func (c *Client) ListPermissions(remotePath string) (PermissionList, error) {
	var permissions PermissionList
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return permissions, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions"
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return permissions, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&permissions); err != nil {
		return permissions, fmt.Errorf("decoding permissions response: %v", err)
	}

	return permissions, nil
}

// GetPermission retrieves a specific permission on a drive item.
func (c *Client) GetPermission(remotePath, permissionID string) (Permission, error) {
	var permission Permission
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return permission, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	res, err := c.apiCall("GET", url, "", nil)
	if err != nil {
		return permission, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&permission); err != nil {
		return permission, fmt.Errorf("decoding permission response: %v", err)
	}

	return permission, nil
}

// UpdatePermission updates a specific permission on a drive item.
func (c *Client) UpdatePermission(remotePath, permissionID string, request UpdatePermissionRequest) (Permission, error) {
	var permission Permission
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return permission, fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	requestBody, err := json.Marshal(request)
	if err != nil {
		return permission, fmt.Errorf("marshaling update permission request: %v", err)
	}

	res, err := c.apiCall("PATCH", url, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return permission, err
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&permission); err != nil {
		return permission, fmt.Errorf("decoding permission response: %v", err)
	}

	return permission, nil
}

// DeletePermission deletes a specific permission on a drive item.
func (c *Client) DeletePermission(remotePath, permissionID string) error {
	item, err := c.GetDriveItemByPath(remotePath)
	if err != nil {
		return fmt.Errorf("getting drive item: %w", err)
	}

	url := customRootURL + "me/drive/items/" + url.PathEscape(item.ID) + "/permissions/" + url.PathEscape(permissionID)
	res, err := c.apiCall("DELETE", url, "", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		return fmt.Errorf("delete permission failed with status: %s", res.Status)
	}

	return nil
}

// apiCall handles the HTTP request and categorizes common errors.
// It will automatically retry once on a 401 Unauthorized error.
func (c *Client) apiCall(method, url, contentType string, body io.ReadSeeker) (*http.Response, error) {
	var res *http.Response
	var err error

	for i := 0; i < 2; i++ {
		c.logger.Debug("apiCall invoked with method: ", method, ", URL: ", url)

		if c.httpClient == nil {
			return nil, errors.New("HTTP client is nil, please provide a valid HTTP client")
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
)

// Package onedrive (download.go) provides methods for downloading files from OneDrive.
// This includes handling standard downloads, chunked downloads for resumable operations,
// and downloading files converted to different formats.
package onedrive

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

// DownloadFile downloads a file from the specified `remotePath` in OneDrive to the given `localPath`.
// It handles the common Microsoft Graph API pattern where a request to the `/content` endpoint
// returns an HTTP 302 redirect to a pre-authenticated download URL.
//
// This function attempts two methods:
//  1. Direct content request: Issues a GET to `/{driveItemPath}:/content`. If this results
//     in a 302, it follows the redirect.
//  2. Fallback via metadata: If the direct content request fails with 401/404, it attempts
//     to fetch the item's metadata (using `GetDriveItemByPath`) to obtain the
//     `@microsoft.graph.downloadUrl` and then downloads from that URL. This can sometimes
//     succeed due to different permission checks or URL structures for the pre-authenticated URL.
//
// Example:
//
//	err := client.DownloadFile(context.Background(), "/Documents/MyReport.docx", "./MyReport_local.docx")
//	if err != nil { log.Fatal(err) }
//	fmt.Println("File downloaded successfully.")
func (c *Client) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	c.logger.Debugf("DownloadFile called for remotePath: '%s', localPath: '%s'", remotePath, localPath)
	contentURL := BuildPathURL(remotePath) + ":/content"

	// Create a new HTTP client that does *not* automatically follow redirects.
	// The standard oauth2 client transport follows redirects by default. We need to
	// capture the 302 redirect from Graph API to get the pre-authenticated download URL.
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Prevents following redirects.
		},
		Transport: c.httpClient.Transport, // Use the authenticated transport from our client.
	}

	req, err := http.NewRequestWithContext(ctx, "GET", contentURL, nil)
	if err != nil {
		return fmt.Errorf("creating download request for '%s': %w", remotePath, err)
	}

	c.logger.Debug("Attempting direct content download from: ", contentURL)
	res, err := noRedirectClient.Do(req)
	if err != nil {
		return fmt.Errorf("initiating download for '%s' from content URL: %w", remotePath, err)
	}
	defer closeBodySafely(res.Body, c.logger, "download file")

	// If a 302 Found is received, get the pre-authenticated download URL from the Location header.
	if res.StatusCode == http.StatusFound {
		actualDownloadURL := res.Header.Get("Location")
		if actualDownloadURL == "" {
			return fmt.Errorf("download for '%s' redirected (302) but no Location header found", remotePath)
		}
		c.logger.Debugf("Download for '%s' redirected to: %s. Proceeding with download.", remotePath, actualDownloadURL)
		// Download from the pre-authenticated URL. This typically does not require auth headers.
		return c.downloadFromURL(ctx, actualDownloadURL, localPath, "pre-authenticated URL for "+remotePath)
	}

	// If the direct content access fails with 401 (Unauthorized) or 404 (Not Found),
	// attempt the alternative download method via item metadata.
	// This can sometimes succeed if the `@microsoft.graph.downloadUrl` property has different
	// access characteristics or if the item is shared in a way that direct content access is restricted
	// but metadata (including the download URL) is available.
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusNotFound {
		c.logger.Debugf("Direct content download for '%s' failed with status %s. Attempting fallback via item metadata.", remotePath, res.Status)
		return c.DownloadFileByItem(ctx, remotePath, localPath)
	}

	// If not a redirect and not a handled error, and status is OK, assume it's the file content directly.
	// This is less common for Graph API but handled as a possibility.
	if res.StatusCode == http.StatusOK {
		c.logger.Debugf("Direct content download for '%s' returned status 200 OK. Saving response.", remotePath)
		return saveResponseToFile(res, localPath, "direct content for "+remotePath)
	}

	// For any other status codes, return an error.
	errorBody, _ := io.ReadAll(res.Body)
	return fmt.Errorf("downloading file '%s' from content URL failed with status %s: %s", remotePath, res.Status, string(errorBody))
}

// DownloadFileByItem downloads a file by first retrieving its metadata to get the
// `@microsoft.graph.downloadUrl` property, and then downloading from that URL.
// This serves as an alternative download method, often used as a fallback if direct
// content access fails or if explicitly preferred.
//
// Example:
//
//	err := client.DownloadFileByItem(context.Background(), "/Shared/LargeFile.zip", "./LargeFile_local.zip")
//	if err != nil { log.Fatal(err) }
//	fmt.Println("File downloaded via item metadata URL.")
func (c *Client) DownloadFileByItem(ctx context.Context, remotePath, localPath string) error {
	c.logger.Debugf("DownloadFileByItem called for remotePath: '%s', localPath: '%s'", remotePath, localPath)
	// First, get the item metadata.
	item, err := c.GetDriveItemByPath(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("getting item metadata for '%s' to download: %w", remotePath, err)
	}

	// The @microsoft.graph.downloadUrl property contains a pre-authenticated URL for downloading the file.
	if item.DownloadURL == "" {
		return fmt.Errorf("item '%s' has no @microsoft.graph.downloadUrl in its metadata", remotePath)
	}

	c.logger.Debugf("Found @microsoft.graph.downloadUrl for '%s': %s. Proceeding with download.", remotePath, item.DownloadURL)
	return c.downloadFromURL(ctx, item.DownloadURL, localPath, "@microsoft.graph.downloadUrl for "+remotePath)
}

// downloadFromURL is an unexported helper that downloads content from a given URL
// (typically a pre-authenticated download URL from OneDrive) and saves it to `localPath`.
// It uses a configured HTTP client for consistent timeout and retry behavior.
// `sourceDescription` is used for logging/error messages.
func (c *Client) downloadFromURL(ctx context.Context, downloadURL, localPath, sourceDescription string) error {
	c.logger.Debugf("downloadFromURL called for URL: '%s', localPath: '%s' (source: %s)", downloadURL, localPath, sourceDescription)
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("creating download request for '%s' from %s: %w", localPath, sourceDescription, err)
	}

	// Use configured HTTP client for consistent timeout behavior.
	// Pre-authenticated URLs don't require OAuth headers, so we create a basic configured client.
	downloadClient := NewConfiguredHTTPClient(c.httpConfig)
	res, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading '%s' from %s (%s): %w", localPath, sourceDescription, downloadURL, err)
	}
	defer closeBodySafely(res.Body, c.logger, "download from URL")

	if res.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("downloading '%s' from %s (%s) failed with status %s: %s", localPath, sourceDescription, downloadURL, res.Status, string(errorBody))
	}

	return saveResponseToFile(res, localPath, sourceDescription)
}

// saveResponseToFile is an unexported helper that saves an HTTP response body to a local file.
// `sourceDescription` is used for logging/error messages.
func saveResponseToFile(res *http.Response, localPath, sourceDescription string) error {
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating local file '%s' for content from %s: %w", localPath, sourceDescription, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file '%s': %v", localPath, err)
		}
	}()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return fmt.Errorf("saving content from %s to local file '%s': %w", sourceDescription, localPath, err)
	}
	return nil
}

// DownloadFileChunk downloads a specific chunk (byte range) of a file.
// This is used for resumable downloads. `downloadURL` should be the pre-authenticated
// download URL for the file (often obtained from item metadata or a download session).
// `startByte` and `endByte` are inclusive.
// Returns an io.ReadCloser for the chunk data, which the caller must close.
//
// Example:
//
//	// Assume downloadURL is known for a large file
//	chunkStream, err := client.DownloadFileChunk(context.Background(), downloadURL, 0, 1023) // Download first 1KB
//	if err != nil { log.Fatal(err) }
//	defer chunkStream.Close()
//	// Read data from chunkStream
func (c *Client) DownloadFileChunk(ctx context.Context, downloadURL string, startByte, endByte int64) (io.ReadCloser, error) {
	c.logger.Debugf("DownloadFileChunk called for URL: '%s', range: %d-%d", downloadURL, startByte, endByte)
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating chunk download request for URL '%s': %w", downloadURL, err)
	}
	// Set the Range header to request a specific part of the file.
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", startByte, endByte))

	// Use the authenticated client here, as some chunked download scenarios might still
	// operate on the primary item URL rather than a short-lived pre-authenticated one,
	// or if the pre-authenticated URL itself requires original auth context for range requests.
	// If downloadURL is always a publicly accessible pre-signed URL, http.DefaultClient could be used.
	// However, sticking to c.httpClient is safer if the nature of downloadURL can vary.
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading chunk from '%s' (range %d-%d): %w", downloadURL, startByte, endByte, err)
	}

	// For a successful range request, the server should respond with HTTP 206 Partial Content.
	if res.StatusCode != http.StatusPartialContent {
		closeBodySafely(res.Body, c.logger, "download file chunk error")
		errorBody, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("unexpected status code %d for chunk download from '%s' (range %d-%d): %s", res.StatusCode, downloadURL, startByte, endByte, string(errorBody))
	}

	return res.Body, nil // Caller is responsible for closing the body.
}

// DownloadFileAsFormat downloads a file from OneDrive and converts it to a specified format
// (e.g., from ".docx" to ".pdf"). The converted file is saved to `localPath`.
// Not all file types and format conversions are supported by the Graph API.
// This function also handles the 302 redirect pattern for downloads.
//
// Example:
//
//	err := client.DownloadFileAsFormat(context.Background(), "/Presentations/MySlides.pptx", "./MySlides.pdf", "pdf")
//	if err != nil { log.Fatal(err) }
//	fmt.Println("Presentation downloaded as PDF.")
func (c *Client) DownloadFileAsFormat(ctx context.Context, remotePath, localPath, format string) error {
	c.logger.Debugf("DownloadFileAsFormat called for remotePath: '%s', localPath: '%s', format: '%s'", remotePath, localPath, format)
	// Construct the URL for format conversion: /content?format={format}
	contentURL := BuildPathURL(remotePath) + ":/content?format=" + url.QueryEscape(format)

	// Use a client that doesn't follow redirects to capture the pre-authenticated download URL.
	noRedirectClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: c.httpClient.Transport, // Use authenticated transport.
	}

	req, err := http.NewRequestWithContext(ctx, "GET", contentURL, nil)
	if err != nil {
		return fmt.Errorf("creating download-as-format request for '%s' (format %s): %w", remotePath, format, err)
	}

	c.logger.Debugf("Attempting download-as-format from: %s", contentURL)
	res, err := noRedirectClient.Do(req)
	if err != nil {
		return fmt.Errorf("initiating download-as-format for '%s' (format %s): %w", remotePath, format, err)
	}
	defer closeBodySafely(res.Body, c.logger, "download file as format")

	if res.StatusCode == http.StatusFound {
		actualDownloadURL := res.Header.Get("Location")
		if actualDownloadURL == "" {
			return fmt.Errorf("download-as-format for '%s' (format %s) redirected (302) but no Location header", remotePath, format)
		}
		c.logger.Debugf("Download-as-format for '%s' (format %s) redirected to: %s. Proceeding.", remotePath, format, actualDownloadURL)
		return c.downloadFromURL(ctx, actualDownloadURL, localPath, fmt.Sprintf("pre-authenticated URL for %s (format %s)", remotePath, format))
	}

	// If not a redirect and status is OK, save the content.
	if res.StatusCode == http.StatusOK {
		c.logger.Debugf("Download-as-format for '%s' (format %s) returned 200 OK. Saving response.", remotePath, format)
		return saveResponseToFile(res, localPath, fmt.Sprintf("direct content for %s (format %s)", remotePath, format))
	}

	// Handle other errors.
	errorBody, _ := io.ReadAll(res.Body)
	return fmt.Errorf("downloading file '%s' as format '%s' failed with status %s: %s", remotePath, format, res.Status, string(errorBody))
}

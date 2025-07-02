// Package items (items_upload.go) defines Cobra commands for creating folders (mkdir)
// and uploading files to OneDrive. It supports both resumable uploads for large files
// (via 'items upload') and simple, non-resumable uploads for smaller files
// ('items upload-simple'). It also includes commands for managing upload sessions
// ('items cancel-upload', 'items get-upload-status').
package items

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath" // Used for path manipulation.
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// filesMkdirCmd handles 'items mkdir <path>'.
// It creates a new, empty folder at the specified remote path in OneDrive.
var filesMkdirCmd = &cobra.Command{
	Use:   "mkdir <remote-folder-path>",
	Short: "Create a new folder in OneDrive",
	Long: `Creates a new, empty folder at the specified remote path within your OneDrive.
The path should be the full path where the new folder will be created.
Example: onedrive-client items mkdir /Documents/NewProject`,
	Args: cobra.ExactArgs(1), // Requires exactly one argument: the remote folder path.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items mkdir': %w", err)
		}
		return filesMkdirLogic(a, cmd, args)
	},
}

// filesUploadCmd handles 'items upload <local-file> [remote-path]'.
// It uploads a local file to OneDrive, using resumable upload sessions for large files.
var filesUploadCmd = &cobra.Command{
	Use:   "upload <local-file-path> [remote-destination-path]",
	Short: "Upload a file to OneDrive (resumable for large files)",
	Long: `Uploads a local file to a specific folder in your OneDrive.
If the remote destination path is a folder, the file is uploaded into that folder with its original name.
If the remote destination path is omitted or is "/", the file is uploaded to the root of your OneDrive.
This command automatically uses resumable upload sessions for files, making it suitable for large files
and resilient to network interruptions. Progress is saved, and interrupted uploads can be resumed.`,
	Example: `onedrive-client items upload ./report.docx /Documents
onedrive-client items upload ./archive.zip /Backup/Archives
onedrive-client items upload video.mp4`, // Uploads to root
	Args: cobra.RangeArgs(1, 2), // Requires local file path, optionally a remote destination path.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items upload': %w", err)
		}
		return filesUploadLogic(a, cmd, args)
	},
}

// filesCancelUploadCmd handles 'items cancel-upload <upload-url>'.
// It cancels an active resumable upload session.
var filesCancelUploadCmd = &cobra.Command{
	Use:     "cancel-upload <upload-session-url>",
	Short:   "Cancel an active resumable upload session",
	Long:    `Cancels an active resumable upload session using its unique upload URL. This is useful if an upload is no longer needed or needs to be restarted.`,
	Example: `onedrive-client items cancel-upload "https://graph.microsoft.com/v1.0/drive/items/.../uploadSession?..."`,
	Args:    cobra.ExactArgs(1), // Requires the upload session URL.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items cancel-upload': %w", err)
		}
		return filesCancelUploadLogic(a, cmd, args)
	},
}

// filesGetUploadStatusCmd handles 'items get-upload-status <upload-url>'.
// It retrieves the status of an ongoing or paused resumable upload session.
var filesGetUploadStatusCmd = &cobra.Command{
	Use:     "get-upload-status <upload-session-url>",
	Short:   "Get the status of a resumable upload session",
	Long:    `Retrieves the current status of an active or paused resumable upload session using its upload URL. This shows which byte ranges have been successfully uploaded.`,
	Example: `onedrive-client items get-upload-status "https://graph.microsoft.com/v1.0/drive/items/.../uploadSession?..."`,
	Args:    cobra.ExactArgs(1), // Requires the upload session URL.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items get-upload-status': %w", err)
		}
		return filesGetUploadStatusLogic(a, cmd, args)
	},
}

// filesUploadSimpleCmd handles 'items upload-simple <local-file> <remote-path>'.
// It uploads a file using a non-resumable ("simple") PUT request, suitable for small files only.
var filesUploadSimpleCmd = &cobra.Command{
	Use:   "upload-simple <local-file-path> <remote-file-path>",
	Short: "Upload a small file using non-resumable upload",
	Long: `Uploads a local file to a specific, full remote path in your OneDrive using a non-resumable ("simple") PUT request.
This method is suitable for small files only (typically under 4MB). For larger files, use 'items upload'.
The remote path must be the full path including the desired filename on OneDrive.`,
	Example: `onedrive-client items upload-simple ./config.txt /Settings/config_backup.txt`,
	Args:    cobra.ExactArgs(2), // Requires local file path and full remote file path.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items upload-simple': %w", err)
		}
		return filesUploadSimpleLogic(a, cmd, args)
	},
}

// filesMkdirLogic contains the core logic for the 'items mkdir' command.
func filesMkdirLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) == 0 { // Should be caught by Args validation.
		return fmt.Errorf("remote folder path for 'mkdir' is required")
	}
	remotePath := args[0]

	// Determine parent path and the name of the folder to be created.
	// Example: if remotePath is "/Documents/NewFolder", parentPath is "/Documents", folderName is "NewFolder".
	parentPath := filepath.Dir(remotePath)
	folderName := filepath.Base(remotePath)

	// filepath.Dir("/") returns "/". filepath.Dir("NewFolderAtRoot") returns ".".
	// If the parent path is ".", it means the folder should be created in the root.
	// The SDK's CreateFolder expects "/" for the root parent path.
	if parentPath == "." {
		parentPath = "/"
	}

	item, err := a.SDK.CreateFolder(cmd.Context(), parentPath, folderName)
	if err != nil {
		return fmt.Errorf("creating folder '%s' in '%s': %w", folderName, parentPath, err)
	}
	log.Printf("Folder '%s' created successfully in '%s'. ID: %s", item.Name, parentPath, item.ID)
	return nil
}

// filesUploadLogic contains the core logic for the 'items upload' command (resumable).
func filesUploadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) == 0 { // Should be caught by Args validation.
		return fmt.Errorf("local file path for 'upload' is required")
	}
	localPath := args[0]

	var remoteDestPath string // This is the destination folder or root.
	if len(args) > 1 {
		remoteDestPath = args[1]
	} else {
		// If no remote path is specified, default to uploading to the root directory.
		remoteDestPath = "/"
	}

	// Verify the local file exists before proceeding.
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local file '%s' does not exist", localPath)
	}

	// Session manager for handling resumable upload state.
	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("creating session manager for upload: %w", err)
	}

	// Determine the final remote path for the file, including its name.
	// e.g., if remoteDestPath is "/Documents" and local file is "report.docx",
	// finalRemotePath becomes "/Documents/report.docx".
	finalRemotePath := joinRemotePath(remoteDestPath, filepath.Base(localPath))

	// Check if a resumable session already exists for this file combination.
	state, err := mgr.Load(localPath, finalRemotePath)
	if err != nil {
		return fmt.Errorf("loading existing upload session state for '%s': %w", localPath, err)
	}

	if state != nil && state.UploadURL != "" {
		// An existing session was found, attempt to resume.
		log.Printf("Resuming upload for '%s' to '%s' from %d bytes.", localPath, finalRemotePath, state.CompletedBytes)
		// Reconstruct the UploadSession object from the saved state.
		uploadSession := onedrive.UploadSession{
			UploadURL:          state.UploadURL,
			ExpirationDateTime: state.ExpirationDateTime.Format(time.RFC3339), // Ensure correct format.
			// NextExpectedRanges will be queried by GetUploadSessionStatus or implicitly handled by UploadChunk.
		}
		return uploadFileInChunks(a, cmd, mgr, localPath, finalRemotePath, uploadSession, state.CompletedBytes)
	}
	// No existing session, start a new upload.
	log.Printf("Starting new upload for '%s' to '%s'.", localPath, finalRemotePath)
	return startNewUpload(a, cmd, mgr, localPath, finalRemotePath)
}

// startNewUpload initiates a new resumable upload session.
func startNewUpload(a *app.App, cmd *cobra.Command, mgr *session.Manager, localPath, remotePath string) error {
	// Create a new upload session with the OneDrive API.
	uploadSession, err := a.SDK.CreateUploadSession(cmd.Context(), remotePath)
	if err != nil {
		return fmt.Errorf("creating new upload session for '%s': %w", remotePath, err)
	}
	log.Printf("New upload session created for '%s'. Upload URL: %s", remotePath, uploadSession.UploadURL)

	// Proceed to upload file in chunks using the new session, starting from byte 0.
	return uploadFileInChunks(a, cmd, mgr, localPath, remotePath, uploadSession, 0)
}

// uploadFileInChunks handles the chunked file upload process for a resumable session.
// `startFromByte` indicates where to resume if this is a continued upload.
func uploadFileInChunks(a *app.App, cmd *cobra.Command, mgr *session.Manager, localPath, remotePath string, uploadSession onedrive.UploadSession, startFromByte int64) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening local file '%s' for chunked upload: %w", localPath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("getting file info for '%s': %w", localPath, err)
	}
	totalSize := fileInfo.Size()

	// Initialize UI progress bar.
	progressBar := ui.NewProgressBar(int(totalSize), "Uploading "+filepath.Base(localPath))
	progressBar.Set(int(startFromByte)) // Set initial progress if resuming.
	defer progressBar.Close()

	// Define chunk size (e.g., 5MB). Must be a multiple of 320 KiB (327,680 bytes).
	// Graph API recommends chunks between 5-10 MiB.
	const chunkSize = 5 * 320 * 1024 // 1.6 MiB, for faster testing. Use larger for production.
	currentByte := startFromByte

	// Set up signal handling for graceful interruption (Ctrl+C).
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan) // Clean up signal notification.

	// Seek to the starting byte if resuming.
	if currentByte > 0 {
		if _, seekErr := file.Seek(currentByte, io.SeekStart); seekErr != nil {
			return fmt.Errorf("seeking to resume position %d in '%s': %w", currentByte, localPath, seekErr)
		}
	}

	for currentByte < totalSize {
		select {
		case <-sigChan: // Handle interruption signal.
			log.Println("\nUpload interrupted by user. Saving session state for resumption...")
			// Save current progress before exiting.
			expirationTime, _ := time.Parse(time.RFC3339, uploadSession.ExpirationDateTime)
			state := &session.State{
				UploadURL:          uploadSession.UploadURL,
				ExpirationDateTime: expirationTime,
				LocalPath:          localPath,
				RemotePath:         remotePath,
				CompletedBytes:     currentByte,
			}
			if saveErr := mgr.Save(state); saveErr != nil {
				log.Printf("Error saving session state on interruption: %v", saveErr)
			} else {
				log.Println("Session state saved.")
			}
			return fmt.Errorf("upload interrupted") // Return an error to stop processing.
		default:
			// Continue with upload if no signal.
		}

		// Determine the end byte for the current chunk.
		endByte := currentByte + chunkSize - 1
		if endByte >= totalSize {
			endByte = totalSize - 1 // This is the last chunk.
		}

		// Read the chunk data from the file.
		// Use io.LimitReader to ensure we only read up to the chunk size.
		chunkReader := io.LimitReader(file, endByte-currentByte+1)

		// Upload the current chunk.
		// The SDK's UploadChunk will handle reading from chunkReader.
		result, errSdk := a.SDK.UploadChunk(cmd.Context(), uploadSession.UploadURL, currentByte, endByte, totalSize, chunkReader)
		if errSdk != nil {
			// On error, save session state for potential resumption.
			log.Printf("\nError uploading chunk for '%s' (range %d-%d). Saving session state...", localPath, currentByte, endByte)
			expirationTime, _ := time.Parse(time.RFC3339, uploadSession.ExpirationDateTime)
			state := &session.State{
				UploadURL:          uploadSession.UploadURL,
				ExpirationDateTime: expirationTime,
				LocalPath:          localPath,
				RemotePath:         remotePath,
				CompletedBytes:     currentByte, // Save progress up to the start of the failed chunk.
			}
			if saveErr := mgr.Save(state); saveErr != nil {
				log.Printf("Error saving session state after chunk upload error: %v", saveErr)
			} else {
				log.Println("Session state saved for resumption.")
			}
			return fmt.Errorf("uploading chunk (bytes %d-%d) for '%s': %w", currentByte, endByte, localPath, errSdk)
		}

		// Update progress.
		currentByte = endByte + 1
		progressBar.Set64(currentByte)

		// The OneDrive API might return a new UploadSession object with updated details (e.g., new expiry).
		// Or, if this was the final chunk, it might return the DriveItem metadata of the completed file.
		if result.UploadURL != "" { // If UploadURL is still present, it's likely an intermediate response.
			uploadSession = result
		} else if currentByte >= totalSize { // If all bytes are uploaded.
			// This was the last chunk, and the response might be the DriveItem itself.
			// The SDK's UploadChunk should ideally parse this correctly if it's a DriveItem.
			// For now, we assume completion if no error and currentByte >= totalSize.
			log.Printf("\nFinal chunk for '%s' uploaded.", localPath)
			break // Exit loop as upload is complete.
		}
	}

	// Upload completed successfully. Clean up the session file.
	if err := mgr.Delete(localPath, remotePath); err != nil {
		log.Printf("Warning: failed to delete session file for completed upload '%s': %v", localPath, err)
	}
	log.Printf("\nFile '%s' uploaded successfully to '%s'.", localPath, remotePath)
	return nil
}

// filesCancelUploadLogic contains the core logic for 'items cancel-upload'.
func filesCancelUploadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) == 0 { // Should be caught by Args validation.
		return fmt.Errorf("upload session URL for 'cancel-upload' is required")
	}
	uploadURL := args[0]
	if uploadURL == "" {
		return fmt.Errorf("upload session URL cannot be empty")
	}

	err := a.SDK.CancelUploadSession(cmd.Context(), uploadURL)
	if err != nil {
		return fmt.Errorf("canceling upload session '%s': %w", uploadURL, err)
	}
	log.Printf("Upload session '%s' cancelled successfully.", uploadURL)
	return nil
}

// filesGetUploadStatusLogic contains the core logic for 'items get-upload-status'.
func filesGetUploadStatusLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) == 0 { // Should be caught by Args validation.
		return fmt.Errorf("upload session URL for 'get-upload-status' is required")
	}
	uploadURL := args[0]
	if uploadURL == "" {
		return fmt.Errorf("upload session URL cannot be empty")
	}

	status, err := a.SDK.GetUploadSessionStatus(cmd.Context(), uploadURL)
	if err != nil {
		return fmt.Errorf("getting status for upload session '%s': %w", uploadURL, err)
	}

	// Display the upload session status information.
	fmt.Println("Upload Session Status:")
	fmt.Printf("  Upload URL:          %s\n", status.UploadURL)
	fmt.Printf("  Expiration DateTime: %s\n", status.ExpirationDateTime)
	if len(status.NextExpectedRanges) > 0 {
		fmt.Println("  Next Expected Ranges (start-end, inclusive):")
		for _, r := range status.NextExpectedRanges {
			fmt.Printf("    %s\n", r)
		}
	} else {
		// If NextExpectedRanges is empty, it usually means the upload is complete or the session is invalid/expired.
		fmt.Println("  Status: Upload appears to be complete, or session is no longer active for new ranges.")
	}
	return nil
}

// filesUploadSimpleLogic contains the core logic for 'items upload-simple'.
func filesUploadSimpleLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) < 2 { // Should be caught by Args validation.
		return fmt.Errorf("both local file path and remote file path are required for 'upload-simple'")
	}
	localPath := args[0]
	remotePath := args[1] // This is the full remote path including filename.

	// Verify local file exists.
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local file '%s' does not exist", localPath)
	}

	item, err := a.SDK.UploadFile(cmd.Context(), localPath, remotePath)
	if err != nil {
		return fmt.Errorf("simple upload of '%s' to '%s' failed: %w", localPath, remotePath, err)
	}
	log.Printf("File '%s' uploaded successfully to '%s' using simple upload. Item ID: %s, Size: %d bytes", localPath, remotePath, item.ID, item.Size)
	return nil
}

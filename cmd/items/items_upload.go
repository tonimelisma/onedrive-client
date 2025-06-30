package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

var filesMkdirCmd = &cobra.Command{
	Use:   "mkdir <path>",
	Short: "Create a new folder",
	Long:  "Creates a new, empty folder at the specified remote path.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesMkdirLogic(a, cmd, args)
	},
}

var filesUploadCmd = &cobra.Command{
	Use:   "upload <local-file> [remote-path]",
	Short: "Upload a file",
	Long:  "Uploads a local file to a specific folder in your OneDrive. If remote path is omitted, uploads to the root.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesUploadLogic(a, cmd, args)
	},
}

var filesCancelUploadCmd = &cobra.Command{
	Use:   "cancel-upload <upload-url>",
	Short: "Cancel a resumable upload session",
	Long:  "Cancels an active resumable upload session using its upload URL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesCancelUploadLogic(a, cmd, args)
	},
}

var filesGetUploadStatusCmd = &cobra.Command{
	Use:   "get-upload-status <upload-url>",
	Short: "Get the status of a resumable upload session",
	Long:  "Retrieves the current status of a resumable upload session using its upload URL.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesGetUploadStatusLogic(a, cmd, args)
	},
}

var filesUploadSimpleCmd = &cobra.Command{
	Use:   "upload-simple <local-file> <remote-path>",
	Short: "Upload a file using simple upload",
	Long:  "Uploads a local file to a specific path in your OneDrive using non-resumable upload. Suitable for small files only.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesUploadSimpleLogic(a, cmd, args)
	},
}

func filesMkdirLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("folder path is required")
	}

	remotePath := args[0]
	parentPath := filepath.Dir(remotePath)
	folderName := filepath.Base(remotePath)

	// Use the root path as parent if the parent is "."
	if parentPath == "." {
		parentPath = "/"
	}

	item, err := a.SDK.CreateFolder(parentPath, folderName)
	if err != nil {
		return err
	}
	log.Printf("Folder '%s' created successfully with ID: %s", item.Name, item.ID)
	return nil
}

func filesUploadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("local path is required")
	}

	localPath := args[0]
	var remotePath string
	if len(args) > 1 {
		remotePath = args[1]
	} else {
		remotePath = "/"
	}

	// Check if local file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local file '%s' does not exist", localPath)
	}

	mgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("creating session manager: %w", err)
	}

	// Determine the final remote path (including filename)
	finalRemotePath := joinRemotePath(remotePath, filepath.Base(localPath))

	// Check for existing session
	state, err := mgr.Load(localPath, finalRemotePath)
	if err != nil {
		return fmt.Errorf("loading session state: %w", err)
	}

	if state != nil {
		// Resume existing upload
		log.Printf("Resuming upload from %d bytes", state.CompletedBytes)
		uploadSession := onedrive.UploadSession{
			UploadURL:          state.UploadURL,
			ExpirationDateTime: state.ExpirationDateTime.Format("2006-01-02T15:04:05Z"),
		}
		return uploadFileInChunks(a, mgr, localPath, finalRemotePath, uploadSession)
	} else {
		// Start new upload
		return startNewUpload(a, mgr, localPath, finalRemotePath)
	}
}

func startNewUpload(a *app.App, mgr *session.Manager, localPath, remotePath string) error {
	uploadSession, err := a.SDK.CreateUploadSession(remotePath)
	if err != nil {
		return fmt.Errorf("creating upload session: %w", err)
	}

	return uploadFileInChunks(a, mgr, localPath, remotePath, uploadSession)
}

func uploadFileInChunks(a *app.App, mgr *session.Manager, localPath, remotePath string, uploadSession onedrive.UploadSession) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}
	totalSize := fileInfo.Size()

	// Initialize progress bar
	progressBar := ui.NewProgressBar(int(totalSize))
	defer progressBar.Close()

	chunkSize := int64(5 * 1024 * 1024) // 5MB chunks
	var currentByte int64 = 0

	// Set up signal handling for graceful interruption
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for currentByte < totalSize {
		select {
		case <-sigChan:
			log.Println("\nUpload interrupted. Session saved for resumption.")
			return nil
		default:
		}

		endByte := currentByte + chunkSize - 1
		if endByte >= totalSize {
			endByte = totalSize - 1
		}

		// Read the chunk
		chunkData := make([]byte, endByte-currentByte+1)
		_, err := file.ReadAt(chunkData, currentByte)
		if err != nil && err != io.EOF {
			return fmt.Errorf("reading chunk: %w", err)
		}

		// Upload the chunk
		result, err := a.SDK.UploadChunk(uploadSession.UploadURL, currentByte, endByte, totalSize, bytes.NewReader(chunkData))
		if err != nil {
			// Save session state for resumption
			expirationTime, _ := time.Parse(time.RFC3339, uploadSession.ExpirationDateTime)
			state := &session.State{
				UploadURL:          uploadSession.UploadURL,
				ExpirationDateTime: expirationTime,
				LocalPath:          localPath,
				RemotePath:         remotePath,
				CompletedBytes:     currentByte,
			}
			mgr.Save(state)
			return fmt.Errorf("uploading chunk: %w", err)
		}

		currentByte = endByte + 1
		progressBar.Set64(currentByte)

		// Update upload session info if provided
		if result.UploadURL != "" {
			uploadSession = result
		}
	}

	// Upload completed successfully, clean up session
	mgr.Delete(localPath, remotePath)
	log.Printf("File '%s' uploaded successfully.", localPath)
	return nil
}

func filesCancelUploadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("upload URL is required")
	}

	uploadURL := args[0]
	if uploadURL == "" {
		return fmt.Errorf("upload URL cannot be empty")
	}

	err := a.SDK.CancelUploadSession(uploadURL)
	if err != nil {
		return err
	}
	log.Println("Upload session cancelled successfully.")
	return nil
}

func filesGetUploadStatusLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("upload URL is required")
	}

	uploadURL := args[0]
	if uploadURL == "" {
		return fmt.Errorf("upload URL cannot be empty")
	}

	status, err := a.SDK.GetUploadSessionStatus(uploadURL)
	if err != nil {
		return err
	}

	fmt.Println("Upload Session Status:")
	fmt.Printf("  Upload URL: %s\n", status.UploadURL)
	fmt.Printf("  Expiration: %s\n", status.ExpirationDateTime)
	if len(status.NextExpectedRanges) == 0 {
		fmt.Println("  Status: Upload completed")
	} else {
		fmt.Println("  Next Expected Ranges:")
		for _, r := range status.NextExpectedRanges {
			fmt.Printf("    %s\n", r)
		}
	}
	return nil
}

func filesUploadSimpleLogic(a *app.App, cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("both local path and remote path are required")
	}

	localPath := args[0]
	remotePath := args[1]

	// Check if local file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local file '%s' does not exist", localPath)
	}

	item, err := a.SDK.UploadFile(localPath, remotePath)
	if err != nil {
		return err
	}
	log.Printf("File uploaded successfully. Item ID: %s, Size: %d bytes", item.ID, item.Size)
	return nil
}

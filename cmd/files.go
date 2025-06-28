package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Manage files and folders",
	Long:  "Provides commands to list, stat, upload, download, and manage files and folders.",
}

var filesListCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "List files and folders in a given path",
	Long:  "Lists the contents of a directory in your OneDrive. If no path is provided, it defaults to the root directory.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesListLogic(a, cmd, args)
	},
}

var filesStatCmd = &cobra.Command{
	Use:   "stat <path>",
	Short: "Get metadata for a file or folder",
	Long:  "Retrieves detailed metadata for a specific file or folder by its path.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesStatLogic(a, cmd, args)
	},
}

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

var filesDownloadCmd = &cobra.Command{
	Use:   "download <remote-path> [local-path]",
	Short: "Download a file",
	Long:  "Downloads a file from your OneDrive. If local path is omitted, it saves to the current directory with the same name.",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesDownloadLogic(a, cmd, args)
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

var filesListRootDeprecatedCmd = &cobra.Command{
	Use:   "list-root-deprecated",
	Short: "List items in the root drive (deprecated)",
	Long:  "Lists items in the root drive using the deprecated GetRootDriveItems method. Use 'files list /' instead.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesListRootDeprecatedLogic(a, cmd, args)
	},
}

var filesRmCmd = &cobra.Command{
	Use:   "rm <remote-path>",
	Short: "Delete a file or folder",
	Long:  "Deletes a file or folder from your OneDrive. Items are moved to the recycle bin, not permanently deleted.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesRmLogic(a, cmd, args)
	},
}

var filesCopyCmd = &cobra.Command{
	Use:   "copy <source-path> <destination-path> [new-name]",
	Short: "Copy a file or folder",
	Long:  "Creates a copy of a file or folder in OneDrive. The copy operation is asynchronous by default. Use --wait to block until completion.",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesCopyLogic(a, cmd, args)
	},
}

var filesCopyStatusCmd = &cobra.Command{
	Use:   "copy-status <monitor-url>",
	Short: "Check the status of a copy operation",
	Long:  "Monitors the progress of an asynchronous copy operation using the monitor URL returned by the copy command.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesCopyStatusLogic(a, cmd, args)
	},
}

var filesMvCmd = &cobra.Command{
	Use:   "mv <source-path> <destination-path>",
	Short: "Move a file or folder",
	Long:  "Moves a file or folder to a new location in your OneDrive.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesMvLogic(a, cmd, args)
	},
}

var filesRenameCmd = &cobra.Command{
	Use:   "rename <remote-path> <new-name>",
	Short: "Rename a file or folder",
	Long:  "Renames a file or folder to a new name.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesRenameLogic(a, cmd, args)
	},
}

var filesSearchCmd = &cobra.Command{
	Use:   "search \"<query>\"",
	Short: "Search for files and folders",
	Long:  "Searches for files and folders across your entire OneDrive by query string. The search includes file names, content, and metadata.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesSearchLogic(a, cmd, args)
	},
}

var filesRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "List recently accessed files",
	Long:  "Lists files and folders that have been recently accessed by you. This includes items from your own drive as well as items you have access to from other drives.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesRecentLogic(a, cmd, args)
	},
}

var filesSpecialCmd = &cobra.Command{
	Use:   "special <folder-name>",
	Short: "Access a special folder",
	Long:  "Accesses well-known special folders in your OneDrive. Valid folder names: documents, photos, cameraroll, approot, music, recordings.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesSpecialLogic(a, cmd, args)
	},
}

var filesShareCmd = &cobra.Command{
	Use:   "share <remote-path> <link-type> <scope>",
	Short: "Create a sharing link for a file or folder",
	Long: `Creates a sharing link for a file or folder in your OneDrive.
	
Link types:
  view  - Read-only access
  edit  - Read and write access
  embed - Embeddable link for web pages (OneDrive personal only)

Scopes:
  anonymous    - Anyone with the link can access
  organization - Only people in your organization can access`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesShareLogic(a, cmd, args)
	},
}

func filesListLogic(a *app.App, cmd *cobra.Command, args []string) error {
	path := "/"
	if len(args) > 0 {
		path = args[0]
	}

	items, err := a.SDK.GetDriveItemChildrenByPath(path)
	if err != nil {
		return fmt.Errorf("getting drive items: %w", err)
	}
	ui.DisplayDriveItems(items)
	return nil
}

func filesStatLogic(a *app.App, cmd *cobra.Command, args []string) error {
	path := args[0]
	item, err := a.SDK.GetDriveItemByPath(path)
	if err != nil {
		return fmt.Errorf("getting drive item metadata: %w", err)
	}
	ui.DisplayDriveItem(item)
	return nil
}

func filesMkdirLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	parentDir := path.Dir(remotePath)
	if parentDir == "." {
		parentDir = "/"
	}
	newDirName := path.Base(remotePath)

	_, err := a.SDK.CreateFolder(parentDir, newDirName)
	if err != nil {
		return fmt.Errorf("creating folder: %w", err)
	}
	ui.PrintSuccess("Folder '%s' created successfully in '%s'.\n", newDirName, parentDir)
	return nil
}

const (
	chunkSize = 320 * 1024 * 10 // 3.2MB, a multiple of 320KB
)

// joinRemotePath joins remote paths using forward slashes, not platform-specific separators
func joinRemotePath(dir, file string) string {
	if dir == "" || dir == "/" {
		return "/" + file
	}
	// Ensure dir starts with / and doesn't end with /
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}
	dir = strings.TrimSuffix(dir, "/")
	return dir + "/" + file
}

func filesUploadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	localPath := args[0]
	remoteDir := "/"
	if len(args) == 2 {
		remoteDir = args[1]
	}
	// Use proper remote path joining instead of filepath.Join
	remotePath := joinRemotePath(remoteDir, filepath.Base(localPath))

	// Handle graceful interruption
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("\nUpload interrupted. Run the command again to resume.")
		os.Exit(0)
	}()

	// Check for existing session
	state, err := session.Load(localPath, remotePath)
	if err != nil {
		return fmt.Errorf("loading session state: %w", err)
	}

	var uploadSession onedrive.UploadSession
	if state != nil {
		fmt.Println("Resuming previous upload session.")
		// Get the latest status in case more bytes were uploaded
		sessionStatus, err := a.SDK.GetUploadSessionStatus(state.UploadURL)
		if err != nil {
			// If session expired or is invalid, start a new one
			fmt.Println("Could not get session status, starting a new upload.")
			return startNewUpload(a, localPath, remotePath)
		}
		uploadSession = sessionStatus
	} else {
		fmt.Println("Starting new upload.")
		return startNewUpload(a, localPath, remotePath)
	}

	return uploadFileInChunks(a, localPath, remotePath, uploadSession)
}

func startNewUpload(a *app.App, localPath, remotePath string) error {
	uploadSession, err := a.SDK.CreateUploadSession(remotePath)
	if err != nil {
		return fmt.Errorf("creating upload session: %w", err)
	}

	expTime, err := time.Parse(time.RFC3339, uploadSession.ExpirationDateTime)
	if err != nil {
		return fmt.Errorf("parsing expiration time: %w", err)
	}

	state := &session.State{
		UploadURL:          uploadSession.UploadURL,
		ExpirationDateTime: expTime,
		LocalPath:          localPath,
		RemotePath:         remotePath,
	}
	if err := session.Save(state); err != nil {
		return fmt.Errorf("saving session state: %w", err)
	}

	return uploadFileInChunks(a, localPath, remotePath, uploadSession)
}

func uploadFileInChunks(a *app.App, localPath, remotePath string, uploadSession onedrive.UploadSession) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening local file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}
	totalSize := fileInfo.Size()

	startByte := int64(0)
	if len(uploadSession.NextExpectedRanges) > 0 {
		rangeStart, _ := strconv.ParseInt(strings.Split(uploadSession.NextExpectedRanges[0], "-")[0], 10, 64)
		startByte = rangeStart
	}

	if _, err := file.Seek(startByte, io.SeekStart); err != nil {
		return fmt.Errorf("seeking file: %w", err)
	}

	bar := ui.NewProgressBar(int(totalSize))
	bar.Set(int(startByte))

	reader := io.TeeReader(file, bar)

	for startByte < totalSize {
		endByte := startByte + chunkSize - 1
		if endByte >= totalSize {
			endByte = totalSize - 1
		}

		// Use a LimitedReader to ensure we only read the chunk size
		chunkReader := io.LimitReader(reader, chunkSize)

		_, err := a.SDK.UploadChunk(uploadSession.UploadURL, startByte, endByte, totalSize, chunkReader)
		if err != nil {
			// Save session and exit, allowing resume
			return fmt.Errorf("uploading chunk: %w. Run command again to resume", err)
		}

		startByte = endByte + 1
	}

	// Clean up session file on success
	if err := session.Delete(localPath, remotePath); err != nil {
		// Log this as a warning, as the upload itself was successful
		log.Printf("Warning: failed to delete session file %s: %v", remotePath, err)
	}

	ui.PrintSuccess("File '%s' uploaded successfully to '%s'.\n", localPath, remotePath)
	return nil
}

func filesDownloadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	localPath := filepath.Base(remotePath)
	if len(args) == 2 {
		localPath = args[1]
	}

	// Handle graceful interruption
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("\nDownload interrupted. Run the command again to resume.")
		os.Exit(0)
	}()

	// Get file metadata to know the total size
	item, err := a.SDK.GetDriveItemByPath(remotePath)
	if err != nil {
		return fmt.Errorf("getting remote file info: %w", err)
	}
	totalSize := item.Size

	// Check for existing session
	state, err := session.Load(localPath, remotePath)
	if err != nil {
		return fmt.Errorf("loading session state: %w", err)
	}

	var startByte int64
	var file *os.File

	if state != nil {
		fmt.Println("Resuming previous download session.")
		startByte = state.CompletedBytes
		file, err = os.OpenFile(localPath, os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("opening local file for resuming: %w", err)
		}
	} else {
		fmt.Println("Starting new download.")
		file, err = os.Create(localPath)
		if err != nil {
			return fmt.Errorf("creating local file: %w", err)
		}
		state = &session.State{
			LocalPath:  localPath,
			RemotePath: remotePath,
		}
	}
	defer file.Close()

	if _, err := file.Seek(startByte, io.SeekStart); err != nil {
		return fmt.Errorf("seeking file: %w", err)
	}

	bar := ui.NewProgressBar(int(totalSize))
	bar.Set(int(startByte))

	for startByte < totalSize {
		endByte := startByte + chunkSize - 1
		if endByte >= totalSize {
			endByte = totalSize - 1
		}

		// The download URL is part of the DriveItem model, but not yet exposed in the SDK call.
		// For now, we will construct it manually.
		downloadURL := onedrive.BuildPathURL(remotePath) + ":/content"

		body, err := a.SDK.DownloadFileChunk(downloadURL, startByte, endByte)
		if err != nil {
			return fmt.Errorf("downloading chunk: %w. Run command again to resume", err)
		}

		written, err := io.Copy(file, io.TeeReader(body, bar))
		if err != nil {
			body.Close()
			return fmt.Errorf("writing chunk to file: %w", err)
		}
		body.Close()

		startByte += written
		state.CompletedBytes = startByte
		if err := session.Save(state); err != nil {
			return fmt.Errorf("saving session state: %w", err)
		}
	}

	// Clean up session file on success
	if err := session.Delete(localPath, remotePath); err != nil {
		log.Printf("Warning: failed to delete session file %s: %v", remotePath, err)
	}

	ui.PrintSuccess("File '%s' downloaded successfully to '%s'.\n", remotePath, localPath)
	return nil
}

func filesCancelUploadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	uploadURL := args[0]
	if uploadURL == "" {
		return fmt.Errorf("upload URL cannot be empty")
	}

	err := a.SDK.CancelUploadSession(uploadURL)
	if err != nil {
		return fmt.Errorf("cancelling upload session: %w", err)
	}

	ui.PrintSuccess("Upload session cancelled successfully.\n")
	return nil
}

func filesGetUploadStatusLogic(a *app.App, cmd *cobra.Command, args []string) error {
	uploadURL := args[0]
	if uploadURL == "" {
		return fmt.Errorf("upload URL cannot be empty")
	}

	status, err := a.SDK.GetUploadSessionStatus(uploadURL)
	if err != nil {
		return fmt.Errorf("getting upload session status: %w", err)
	}

	fmt.Printf("Upload Session Status:\n")
	fmt.Printf("  Upload URL: %s\n", uploadURL)
	fmt.Printf("  Expiration: %s\n", status.ExpirationDateTime)
	if len(status.NextExpectedRanges) > 0 {
		fmt.Printf("  Next Expected Ranges: %v\n", status.NextExpectedRanges)
	} else {
		fmt.Printf("  Status: Upload completed\n")
	}
	return nil
}

func filesUploadSimpleLogic(a *app.App, cmd *cobra.Command, args []string) error {
	localPath := args[0]
	remotePath := args[1]

	// Validate that local file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("local file '%s' does not exist", localPath)
	}

	item, err := a.SDK.UploadFile(localPath, remotePath)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}

	ui.PrintSuccess("File '%s' uploaded successfully to '%s' (ID: %s).\n", localPath, remotePath, item.ID)
	return nil
}

func filesListRootDeprecatedLogic(a *app.App, cmd *cobra.Command, args []string) error {
	items, err := a.SDK.GetRootDriveItems()
	if err != nil {
		return fmt.Errorf("getting root drive items: %w", err)
	}

	ui.DisplayDriveItems(items)
	return nil
}

func filesRmLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	err := a.SDK.DeleteDriveItem(remotePath)
	if err != nil {
		return fmt.Errorf("deleting drive item: %w", err)
	}

	ui.PrintSuccess("Drive item '%s' deleted successfully.\n", remotePath)
	return nil
}

func filesCopyLogic(a *app.App, cmd *cobra.Command, args []string) error {
	sourcePath := args[0]
	destinationPath := args[1]
	newName := ""
	if len(args) == 3 {
		newName = args[2]
	}

	if sourcePath == "" || destinationPath == "" {
		return fmt.Errorf("source path and destination path cannot be empty")
	}

	wait, _ := cmd.Flags().GetBool("wait")

	monitorURL, err := a.SDK.CopyDriveItem(sourcePath, destinationPath, newName)
	if err != nil {
		return fmt.Errorf("copying drive item: %w", err)
	}

	if !wait {
		// Fire-and-forget mode
		ui.PrintSuccess("Drive item '%s' copy initiated successfully.\n", sourcePath)
		fmt.Printf("Monitor URL: %s\n", monitorURL)
		fmt.Printf("Use 'files copy-status %s' to check progress.\n", monitorURL)
		return nil
	}

	// Wait mode - poll until completion
	fmt.Printf("Copying '%s'...\n", sourcePath)
	for {
		status, err := a.SDK.MonitorCopyOperation(monitorURL)
		if err != nil {
			return fmt.Errorf("monitoring copy operation: %w", err)
		}

		switch status.Status {
		case "completed":
			ui.PrintSuccess("Copy completed successfully!\n")
			if status.ResourceLocation != "" {
				fmt.Printf("New resource location: %s\n", status.ResourceLocation)
			}
			return nil
		case "failed":
			if status.Error != nil {
				return fmt.Errorf("copy operation failed: %s - %s", status.Error.Code, status.Error.Message)
			}
			return fmt.Errorf("copy operation failed: %s", status.StatusDescription)
		case "inProgress":
			if status.PercentageComplete > 0 {
				fmt.Printf("\rProgress: %d%% - %s", status.PercentageComplete, status.StatusDescription)
			} else {
				fmt.Printf("\rProgress: %s", status.StatusDescription)
			}
		default:
			fmt.Printf("\rStatus: %s", status.StatusDescription)
		}

		// Wait 2 seconds before next poll
		time.Sleep(2 * time.Second)
	}
}

func filesCopyStatusLogic(a *app.App, cmd *cobra.Command, args []string) error {
	monitorURL := args[0]
	if monitorURL == "" {
		return fmt.Errorf("monitor URL cannot be empty")
	}

	status, err := a.SDK.MonitorCopyOperation(monitorURL)
	if err != nil {
		return fmt.Errorf("getting copy status: %w", err)
	}

	fmt.Printf("Copy Operation Status:\n")
	fmt.Printf("  Monitor URL: %s\n", monitorURL)
	fmt.Printf("  Status: %s\n", status.Status)
	if status.PercentageComplete > 0 {
		fmt.Printf("  Progress: %d%%\n", status.PercentageComplete)
	}
	fmt.Printf("  Description: %s\n", status.StatusDescription)

	if status.Status == "completed" && status.ResourceLocation != "" {
		fmt.Printf("  Resource Location: %s\n", status.ResourceLocation)
	}

	if status.Status == "failed" && status.Error != nil {
		fmt.Printf("  Error Code: %s\n", status.Error.Code)
		fmt.Printf("  Error Message: %s\n", status.Error.Message)
	}

	return nil
}

func filesMvLogic(a *app.App, cmd *cobra.Command, args []string) error {
	sourcePath := args[0]
	destinationPath := args[1]

	if sourcePath == "" || destinationPath == "" {
		return fmt.Errorf("source path and destination path cannot be empty")
	}

	item, err := a.SDK.MoveDriveItem(sourcePath, destinationPath)
	if err != nil {
		return fmt.Errorf("moving drive item: %w", err)
	}

	ui.PrintSuccess("Drive item '%s' moved successfully to '%s'.\n", sourcePath, item.ParentReference.Path)
	return nil
}

func filesRenameLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	newName := args[1]

	if remotePath == "" || newName == "" {
		return fmt.Errorf("remote path and new name cannot be empty")
	}

	item, err := a.SDK.UpdateDriveItem(remotePath, newName)
	if err != nil {
		return fmt.Errorf("renaming drive item: %w", err)
	}

	ui.PrintSuccess("Drive item renamed successfully to '%s'.\n", item.Name)
	return nil
}

func filesSearchLogic(a *app.App, cmd *cobra.Command, args []string) error {
	query := args[0]
	if query == "" {
		return fmt.Errorf("search query cannot be empty")
	}

	items, err := a.SDK.SearchDriveItems(query)
	if err != nil {
		return fmt.Errorf("searching drive items: %w", err)
	}

	ui.DisplaySearchResults(items, query)
	return nil
}

func filesRecentLogic(a *app.App, cmd *cobra.Command, args []string) error {
	items, err := a.SDK.GetRecentItems()
	if err != nil {
		return fmt.Errorf("getting recent items: %w", err)
	}

	ui.DisplayRecentItems(items)
	return nil
}

func filesSpecialLogic(a *app.App, cmd *cobra.Command, args []string) error {
	folderName := args[0]
	if folderName == "" {
		return fmt.Errorf("folder name cannot be empty")
	}

	item, err := a.SDK.GetSpecialFolder(folderName)
	if err != nil {
		return fmt.Errorf("getting special folder: %w", err)
	}

	ui.DisplaySpecialFolder(item, folderName)
	return nil
}

func filesShareLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	linkType := args[1]
	scope := args[2]

	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	if linkType == "" {
		return fmt.Errorf("link type cannot be empty")
	}

	if scope == "" {
		return fmt.Errorf("scope cannot be empty")
	}

	link, err := a.SDK.CreateSharingLink(remotePath, linkType, scope)
	if err != nil {
		return fmt.Errorf("creating sharing link: %w", err)
	}

	ui.DisplaySharingLink(link)
	return nil
}

func init() {
	rootCmd.AddCommand(filesCmd)
	filesCmd.AddCommand(filesListCmd)
	filesCmd.AddCommand(filesStatCmd)
	filesCmd.AddCommand(filesMkdirCmd)
	filesCmd.AddCommand(filesUploadCmd)
	filesCmd.AddCommand(filesDownloadCmd)
	filesCmd.AddCommand(filesCancelUploadCmd)
	filesCmd.AddCommand(filesGetUploadStatusCmd)
	filesCmd.AddCommand(filesUploadSimpleCmd)
	filesCmd.AddCommand(filesListRootDeprecatedCmd)
	filesCmd.AddCommand(filesRmCmd)
	filesCmd.AddCommand(filesCopyCmd)
	filesCmd.AddCommand(filesCopyStatusCmd)
	filesCmd.AddCommand(filesMvCmd)
	filesCmd.AddCommand(filesRenameCmd)
	filesCmd.AddCommand(filesSearchCmd)
	filesCmd.AddCommand(filesRecentCmd)
	filesCmd.AddCommand(filesSpecialCmd)
	filesCmd.AddCommand(filesShareCmd)

	// Add flags
	filesCopyCmd.Flags().Bool("wait", false, "Wait for copy operation to complete instead of returning immediately")
}

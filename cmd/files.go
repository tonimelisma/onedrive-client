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

	err := a.SDK.DownloadFile(remotePath, localPath)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	ui.PrintSuccess("File '%s' downloaded successfully to '%s'.\n", remotePath, localPath)
	return nil
}

func init() {
	rootCmd.AddCommand(filesCmd)
	filesCmd.AddCommand(filesListCmd)
	filesCmd.AddCommand(filesStatCmd)
	filesCmd.AddCommand(filesMkdirCmd)
	filesCmd.AddCommand(filesUploadCmd)
	filesCmd.AddCommand(filesDownloadCmd)
}

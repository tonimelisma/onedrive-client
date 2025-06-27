package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
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
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		if err := filesListLogic(a, cmd, args); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

var filesStatCmd = &cobra.Command{
	Use:   "stat <path>",
	Short: "Get metadata for a file or folder",
	Long:  "Retrieves detailed metadata for a specific file or folder by its path.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		if err := filesStatLogic(a, cmd, args); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

var filesMkdirCmd = &cobra.Command{
	Use:   "mkdir <path>",
	Short: "Create a new folder",
	Long:  "Creates a new, empty folder at the specified remote path.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		if err := filesMkdirLogic(a, cmd, args); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

var filesUploadCmd = &cobra.Command{
	Use:   "upload <local-file> [remote-path]",
	Short: "Upload a file",
	Long:  "Uploads a local file to a specific folder in your OneDrive. If remote path is omitted, uploads to the root.",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		if err := filesUploadLogic(a, cmd, args); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

var filesDownloadCmd = &cobra.Command{
	Use:   "download <remote-path> [local-path]",
	Short: "Download a file",
	Long:  "Downloads a file from your OneDrive. If local path is omitted, it saves to the current directory with the same name.",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		if err := filesDownloadLogic(a, cmd, args); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

func filesListLogic(a *app.App, cmd *cobra.Command, args []string) error {
	path := "/"
	if len(args) > 0 {
		path = args[0]
	}

	items, err := a.SDK.GetDriveItemChildrenByPath(a.Client, path)
	if err != nil {
		return fmt.Errorf("getting drive items: %w", err)
	}
	ui.DisplayDriveItems(items)
	return nil
}

func filesStatLogic(a *app.App, cmd *cobra.Command, args []string) error {
	path := args[0]
	item, err := a.SDK.GetDriveItemByPath(a.Client, path)
	if err != nil {
		return fmt.Errorf("getting drive item metadata: %w", err)
	}
	ui.DisplayDriveItem(item)
	return nil
}

func filesMkdirLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	parentDir := filepath.Dir(remotePath)
	newDirName := filepath.Base(remotePath)

	_, err := a.SDK.CreateFolder(a.Client, parentDir, newDirName)
	if err != nil {
		return fmt.Errorf("creating folder: %w", err)
	}
	ui.PrintSuccess("Folder '%s' created successfully in '%s'.\n", newDirName, parentDir)
	return nil
}

func filesUploadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	localPath := args[0]
	remotePath := "/"
	if len(args) == 2 {
		remotePath = args[1]
	}
	remoteFile := filepath.Join(remotePath, filepath.Base(localPath))

	_, err := a.SDK.UploadFile(a.Client, localPath, remoteFile)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	ui.PrintSuccess("File '%s' uploaded successfully to '%s'.\n", localPath, remoteFile)
	return nil
}

func filesDownloadLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	localPath := filepath.Base(remotePath)
	if len(args) == 2 {
		localPath = args[1]
	}

	err := a.SDK.DownloadFile(a.Client, remotePath, localPath)
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

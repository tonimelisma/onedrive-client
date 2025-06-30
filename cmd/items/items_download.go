package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

var filesDownloadCmd = &cobra.Command{
	Use:   "download <remote-path> [local-path]",
	Short: "Download a file from OneDrive",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app: %w", err)
		}

		remotePath := args[0]

		// Determine local path
		localPath := ""
		if len(args) > 1 {
			localPath = args[1]
		} else {
			// Extract filename from remote path
			parts := strings.Split(remotePath, "/")
			localPath = parts[len(parts)-1]
		}

		// Check if format flag is specified
		format, _ := cmd.Flags().GetString("format")
		if format != "" {
			err = app.SDK.DownloadFileAsFormat(remotePath, localPath, format)
		} else {
			err = app.SDK.DownloadFile(remotePath, localPath)
		}

		if err != nil {
			return fmt.Errorf("downloading file: %w", err)
		}

		log.Printf("Downloaded '%s' to '%s'", remotePath, localPath)
		return nil
	},
}

var filesListRootDeprecatedCmd = &cobra.Command{
	Use:   "list-root-deprecated",
	Short: "List items in the root drive (deprecated)",
	Long:  "Lists items in the root drive using the deprecated GetRootDriveItems method. Use 'items list /' instead.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesListRootDeprecatedLogic(a, cmd, args)
	},
}

func filesListRootDeprecatedLogic(a *app.App, cmd *cobra.Command, args []string) error {
	items, err := a.SDK.GetRootDriveItems()
	if err != nil {
		return err
	}
	ui.DisplayDriveItems(items)
	return nil
}

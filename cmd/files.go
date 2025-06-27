package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
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
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}

		path := "/"
		if len(args) > 0 {
			path = args[0]
		}

		items, err := onedrive.GetDriveItemChildrenByPath(a.Client, path)
		if err != nil {
			log.Fatalf("Error getting drive items: %v", err)
		}
		// TODO: The DisplayDriveItems function has a hardcoded title.
		// This should be improved later to be more generic.
		ui.DisplayDriveItems(items)
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

		path := args[0]
		item, err := onedrive.GetDriveItemByPath(a.Client, path)
		if err != nil {
			log.Fatalf("Error getting drive item metadata: %v", err)
		}

		ui.DisplayDriveItem(item)
	},
}

func init() {
	rootCmd.AddCommand(filesCmd)
	filesCmd.AddCommand(filesListCmd)
	filesCmd.AddCommand(filesStatCmd)
}

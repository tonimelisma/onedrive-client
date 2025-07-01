package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

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

var filesSearchCmd = &cobra.Command{
	Use:   "search <query> --in <folder-path>",
	Short: "Search within a specific folder",
	Long:  "Searches for files and folders within a specific folder path. For drive-wide search, use 'drives search' instead.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesSearchLogic(a, cmd, args)
	},
}

var filesVersionsCmd = &cobra.Command{
	Use:   "versions <path>",
	Short: "List file versions",
	Long:  "Lists all versions of a specific file with version history and details.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesVersionsLogic(a, args[0])
	},
}

var activitiesCmd = &cobra.Command{
	Use:   "activities <path>",
	Short: "Show activity history for a file or folder",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return activitiesLogic(a, cmd, args)
	},
}

var filesThumbnailsCmd = &cobra.Command{
	Use:   "thumbnails <path>",
	Short: "Get thumbnails for a file",
	Long:  "Retrieves thumbnail images in multiple sizes for files in OneDrive.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesThumbnailsLogic(a, cmd, args)
	},
}

var filesPreviewCmd = &cobra.Command{
	Use:   "preview <path>",
	Short: "Generate preview URL for a file",
	Long:  "Generates preview URLs for Office documents, PDFs, and images with optional page and zoom parameters.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return filesPreviewLogic(a, cmd, args)
	},
}

func filesListLogic(a *app.App, cmd *cobra.Command, args []string) error {
	var path string
	if len(args) > 0 {
		path = args[0]
	} else {
		path = "/"
	}

	items, err := a.SDK.GetDriveItemChildrenByPath(path)
	if err != nil {
		return err
	}
	ui.DisplayDriveItems(items)
	return nil
}

func filesStatLogic(a *app.App, cmd *cobra.Command, args []string) error {
	item, err := a.SDK.GetDriveItemByPath(args[0])
	if err != nil {
		return err
	}
	ui.DisplayDriveItem(item)
	return nil
}

func filesSearchLogic(a *app.App, cmd *cobra.Command, args []string) error {
	query := args[0]

	// Parse paging options
	paging, err := ui.ParsePagingFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing pagination flags: %w", err)
	}

	// Items search requires a folder path
	folderPath, _ := cmd.Flags().GetString("in")
	if folderPath == "" {
		return fmt.Errorf("items search requires --in flag with folder path. For drive-wide search, use 'drives search'")
	}

	items, nextLink, err := a.SDK.SearchDriveItemsInFolder(folderPath, query, paging)
	if err != nil {
		return fmt.Errorf("searching items in folder: %w", err)
	}

	ui.DisplaySearchResults(items, query)
	ui.HandleNextPageInfo(nextLink, paging.FetchAll)
	return nil
}

func filesVersionsLogic(a *app.App, filePath string) error {
	versions, err := a.SDK.GetFileVersions(filePath)
	if err != nil {
		return err
	}
	ui.DisplayFileVersions(versions, filePath)
	return nil
}

func activitiesLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]

	// Parse paging options
	paging, err := ui.ParsePagingFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing pagination flags: %w", err)
	}

	activities, nextLink, err := a.SDK.GetItemActivities(remotePath, paging)
	if err != nil {
		return fmt.Errorf("getting item activities: %w", err)
	}

	ui.DisplayActivities(activities, remotePath)
	ui.HandleNextPageInfo(nextLink, paging.FetchAll)
	return nil
}

func filesThumbnailsLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	thumbnails, err := a.SDK.GetThumbnails(remotePath)
	if err != nil {
		return fmt.Errorf("getting thumbnails: %w", err)
	}

	ui.DisplayThumbnails(thumbnails, remotePath)
	return nil
}

func filesPreviewLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" {
		return fmt.Errorf("remote path cannot be empty")
	}

	// Parse optional flags
	page, _ := cmd.Flags().GetString("page")
	zoom, _ := cmd.Flags().GetFloat64("zoom")

	request := onedrive.PreviewRequest{
		Page: page,
		Zoom: zoom,
	}

	preview, err := a.SDK.PreviewItem(remotePath, request)
	if err != nil {
		return fmt.Errorf("generating preview: %w", err)
	}

	ui.DisplayPreview(preview, remotePath)
	return nil
}

// Package cmd (items_meta.go) defines Cobra commands for retrieving metadata
// and information about OneDrive items (files and folders). This includes
// listing folder contents, getting detailed stats for an item, searching within
// folders, and viewing item versions, activities, thumbnails, and previews.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// filesListCmd handles 'items list [path]'.
// It lists files and folders within a specified path, defaulting to the root if no path is given.
var filesListCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "List files and folders in a OneDrive path",
	Long: `Lists the contents (files and folders) of a specified directory in your OneDrive.
If no path is provided, it defaults to listing the contents of the root directory.
Example: onedrive-client items list /Documents/Reports`,
	Args: cobra.MaximumNArgs(1), // Accepts zero or one argument (the path).
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items list': %w", err)
		}
		return filesListLogic(a, cmd, args)
	},
}

// filesStatCmd handles 'items stat <path>'.
// It retrieves and displays detailed metadata for a specific file or folder.
var filesStatCmd = &cobra.Command{
	Use:   "stat <path>",
	Short: "Get metadata for a specific file or folder",
	Long:  `Retrieves and displays detailed metadata for a specific file or folder in your OneDrive, identified by its path. Metadata includes size, modification dates, ID, etc.`,
	Args:  cobra.ExactArgs(1), // Requires exactly one argument: the path to the item.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items stat': %w", err)
		}
		return filesStatLogic(a, cmd, args)
	},
}

// filesSearchCmd handles 'items search <query> --in <folder-path>'.
// It searches for items matching the query within a specified folder, supporting pagination.
var filesSearchCmd = &cobra.Command{
	Use:   "search <query>", // The --in flag is specified in items_root.go and marked required.
	Short: "Search for items within a specific folder",
	Long: `Searches for files and folders matching the query string, but only within the specified folder path.
This command requires the '--in <folder-path>' flag to define the search scope.
For searching across the entire drive, use the 'drives search' command instead.
Supports pagination flags (--top, --all, --next).`,
	Args: cobra.ExactArgs(1), // Requires one argument: the search query.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items search': %w", err)
		}
		return filesSearchLogic(a, cmd, args)
	},
}

// filesVersionsCmd handles 'items versions <path>'.
// It lists all available versions of a specific file.
var filesVersionsCmd = &cobra.Command{
	Use:   "versions <path>",
	Short: "List all versions of a specific file",
	Long:  `Lists all available versions of a specific file in your OneDrive, including version ID, size, and modification date for each version. This is useful for tracking changes or restoring previous versions (restore functionality not yet implemented in CLI).`,
	Args:  cobra.ExactArgs(1), // Requires one argument: the path to the file.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items versions': %w", err)
		}
		return filesVersionsLogic(a, cmd, args[0])
	},
}

// activitiesCmd handles 'items activities <path>'. Note: This is the item-specific activities command.
// It shows the activity history for a specific file or folder, with pagination.
var activitiesCmd = &cobra.Command{
	Use:   "activities <path>",
	Short: "Show activity history for a specific file or folder",
	Long:  `Displays a list of recent activities (edits, shares, comments, etc.) that have occurred on a specific file or folder. Supports pagination flags.`,
	Args:  cobra.ExactArgs(1), // Requires one argument: the path to the item.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items activities': %w", err)
		}
		return activitiesLogic(a, cmd, args)
	},
}

// filesThumbnailsCmd handles 'items thumbnails <path>'.
// It retrieves and displays available thumbnail URLs for a file.
var filesThumbnailsCmd = &cobra.Command{
	Use:   "thumbnails <path>",
	Short: "Get available thumbnails for a file",
	Long:  `Retrieves and displays URLs for available thumbnail images (e.g., small, medium, large) for a specified file in OneDrive. Useful for image and video files.`,
	Args:  cobra.ExactArgs(1), // Requires one argument: the path to the file.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items thumbnails': %w", err)
		}
		return filesThumbnailsLogic(a, cmd, args)
	},
}

// filesPreviewCmd handles 'items preview <path>'.
// It generates and displays preview URLs for a file, with optional page and zoom parameters.
var filesPreviewCmd = &cobra.Command{
	Use:   "preview <path>",
	Short: "Generate embeddable preview URLs for a file",
	Long: `Generates and displays short-lived, embeddable preview URLs for supported file types (e.g., Office documents, PDFs, images).
Optional flags allow specifying a page number ('--page') for multi-page documents and a zoom level ('--zoom').`,
	Args: cobra.ExactArgs(1), // Requires one argument: the path to the file.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'items preview': %w", err)
		}
		return filesPreviewLogic(a, cmd, args)
	},
}

// filesListLogic contains the core logic for the 'items list' command.
func filesListLogic(a *app.App, cmd *cobra.Command, args []string) error {
	var path string
	if len(args) > 0 {
		path = args[0]
	} else {
		// Default to listing the root directory if no path is provided.
		path = "/"
	}

	items, err := a.SDK.GetDriveItemChildrenByPath(cmd.Context(), path)
	if err != nil {
		return fmt.Errorf("listing items in '%s': %w", path, err)
	}
	ui.DisplayDriveItems(items) // Assumes DisplayDriveItems can handle general DriveItemList
	return nil
}

// filesStatLogic contains the core logic for the 'items stat' command.
func filesStatLogic(a *app.App, cmd *cobra.Command, args []string) error {
	path := args[0]
	item, err := a.SDK.GetDriveItemByPath(cmd.Context(), path)
	if err != nil {
		return fmt.Errorf("getting metadata for '%s': %w", path, err)
	}
	ui.DisplayDriveItem(item)
	return nil
}

// filesSearchLogic contains the core logic for the 'items search' command.
func filesSearchLogic(a *app.App, cmd *cobra.Command, args []string) error {
	query := args[0]

	paging, err := ui.ParsePagingFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing pagination flags for 'items search': %w", err)
	}

	// The '--in' flag (folderPath) is marked as required in items_root.go.
	folderPath, _ := cmd.Flags().GetString("in")
	// Defensive check even though Cobra should handle required flag validation
	if folderPath == "" {
		return fmt.Errorf("folder path is required for search. Use --in flag to specify the folder to search within")
	}

	items, nextLink, err := a.SDK.SearchDriveItemsInFolder(cmd.Context(), folderPath, query, paging)
	if err != nil {
		return fmt.Errorf("searching for '%s' in folder '%s': %w", query, folderPath, err)
	}

	ui.DisplaySearchResults(items, query)
	ui.HandleNextPageInfo(nextLink, paging.FetchAll)
	return nil
}

// filesVersionsLogic contains the core logic for the 'items versions' command.
func filesVersionsLogic(a *app.App, cmd *cobra.Command, filePath string) error {
	versions, err := a.SDK.GetFileVersions(cmd.Context(), filePath)
	if err != nil {
		return fmt.Errorf("listing versions for '%s': %w", filePath, err)
	}
	ui.DisplayFileVersions(versions, filePath)
	return nil
}

// activitiesLogic contains the core logic for the 'items activities' command.
func activitiesLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]

	paging, err := ui.ParsePagingFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing pagination flags for 'items activities': %w", err)
	}

	activities, nextLink, err := a.SDK.GetItemActivities(cmd.Context(), remotePath, paging)
	if err != nil {
		return fmt.Errorf("getting activities for '%s': %w", remotePath, err)
	}

	// Pass the remotePath as a title for the display function.
	ui.DisplayActivities(activities, remotePath)
	ui.HandleNextPageInfo(nextLink, paging.FetchAll)
	return nil
}

// filesThumbnailsLogic contains the core logic for the 'items thumbnails' command.
func filesThumbnailsLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" { // Should be caught by Args validation, but defensive.
		return fmt.Errorf("remote path for thumbnails cannot be empty")
	}

	thumbnails, err := a.SDK.GetThumbnails(cmd.Context(), remotePath)
	if err != nil {
		return fmt.Errorf("getting thumbnails for '%s': %w", remotePath, err)
	}

	ui.DisplayThumbnails(thumbnails, remotePath)
	return nil
}

// filesPreviewLogic contains the core logic for the 'items preview' command.
func filesPreviewLogic(a *app.App, cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	if remotePath == "" { // Should be caught by Args validation.
		return fmt.Errorf("remote path for preview cannot be empty")
	}

	// Parse optional flags for page and zoom, with defaults handled by Cobra/SDK.
	page, _ := cmd.Flags().GetString("page")
	zoom, _ := cmd.Flags().GetFloat64("zoom")

	request := onedrive.PreviewRequest{
		Page: page,
		Zoom: zoom,
	}

	preview, err := a.SDK.PreviewItem(cmd.Context(), remotePath, request)
	if err != nil {
		return fmt.Errorf("generating preview for '%s': %w", remotePath, err)
	}

	ui.DisplayPreview(preview, remotePath)
	return nil
}

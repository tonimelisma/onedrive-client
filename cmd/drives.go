// Package cmd (drives.go) defines the Cobra commands for interacting with
// OneDrive drives. This includes listing available drives, checking storage quota,
// retrieving specific drive metadata, and accessing drive-level features like
// activities, search, delta changes, special folders, recent items, and shared items.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

// drivesCmd represents the base 'drives' command.
// It groups all drive-related subcommands.
var drivesCmd = &cobra.Command{
	Use:   "drives",
	Short: "Manage and inspect OneDrive drives",
	Long:  `Provides commands to list available drives, check quota, view drive activities, search, and access special drive views like recent or shared items.`,
}

// drivesListCmd handles 'drives list'.
// It lists all OneDrive drives (personal, business, SharePoint document libraries) accessible to the user.
var drivesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available OneDrive drives",
	Long:  `Lists all OneDrive drives that the authenticated user has access to, including personal drives, OneDrive for Business, and SharePoint document libraries.`,
	Args:  cobra.NoArgs, // This command takes no arguments.
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize the application context, which handles configuration and SDK setup.
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives list': %w", err)
		}
		return drivesListLogic(a, cmd)
	},
}

// drivesQuotaCmd handles 'drives quota'.
// It displays storage quota information (total, used, remaining) for the user's default OneDrive drive.
var drivesQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Get storage quota for the default drive",
	Long:  `Displays the total, used, and remaining storage quota for the user's default OneDrive drive.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives quota': %w", err)
		}
		return drivesQuotaLogic(a, cmd)
	},
}

// drivesGetCmd handles 'drives get <drive-id>'.
// It retrieves and displays detailed metadata for a specific OneDrive drive using its ID.
var drivesGetCmd = &cobra.Command{
	Use:   "get <drive-id>",
	Short: "Get metadata for a specific drive by its ID",
	Long:  `Retrieves and displays detailed metadata for a specific OneDrive drive, identified by its unique drive ID.`,
	Args:  cobra.ExactArgs(1), // Requires exactly one argument: the drive-id.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives get': %w", err)
		}
		return drivesGetLogic(a, cmd, args[0])
	},
}

// drivesActivitiesCmd handles 'drives activities'.
// It shows the activity history for the entire default drive, with pagination support.
var drivesActivitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "Show activity history for the entire default drive",
	Long:  `Displays a list of recent activities (creations, deletions, edits, shares, etc.) that have occurred across the entire default OneDrive drive. Supports pagination flags.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd) // Renamed 'app' to 'a' to avoid conflict
		if err != nil {
			return fmt.Errorf("initializing app for 'drives activities': %w", err)
		}

		// Parse pagination flags (--top, --all, --next) provided by the user.
		paging, err := ui.ParsePagingFlags(cmd)
		if err != nil {
			return fmt.Errorf("parsing pagination flags for 'drives activities': %w", err)
		}

		// Call the SDK to get drive activities.
		activities, nextLink, err := a.SDK.GetDriveActivities(cmd.Context(), paging)
		if err != nil {
			return fmt.Errorf("getting drive activities: %w", err)
		}

		// Display the activities and any next page information.
		ui.DisplayActivities(activities, "default drive") // Pass a descriptive title for the display.
		ui.HandleNextPageInfo(nextLink, paging.FetchAll)

		return nil
	},
}

// drivesRootCmd handles 'drives root'.
// It lists items (files and folders) in the root directory of the user's default OneDrive.
var drivesRootCmd = &cobra.Command{
	Use:   "root",
	Short: "List items in the root of the default drive",
	Long:  `Lists all files and folders located directly in the root directory of your default OneDrive drive.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives root': %w", err)
		}
		return drivesRootLogic(a, cmd)
	},
}

// drivesSearchCmd handles 'drives search <query>'.
// It searches for files and folders across the entire default drive using the provided query string, with pagination.
var drivesSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for items across the entire default drive",
	Long:  `Searches for files and folders across your entire default OneDrive drive based on the provided query string. Supports pagination flags.`,
	Args:  cobra.ExactArgs(1), // Requires exactly one argument: the search query.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives search': %w", err)
		}
		return drivesSearchLogic(a, cmd, args)
	},
}

// drivesDeltaCmd handles 'drives delta [delta-token]'.
// It tracks changes to items in the default drive using delta queries, allowing for efficient synchronization.
var drivesDeltaCmd = &cobra.Command{
	Use:   "delta [delta-token]",
	Short: "Track changes in the default drive using delta queries",
	Long: `Tracks changes to items in your default OneDrive using delta queries.
An initial call (without a delta-token) returns all items and a new delta-token.
Subsequent calls with the previously obtained delta-token return only items that have changed.
This is useful for building synchronization logic.`,
	Args: cobra.MaximumNArgs(1), // Accepts zero or one argument (the delta-token).
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives delta': %w", err)
		}
		return drivesDeltaLogic(a, cmd, args)
	},
}

// drivesSpecialCmd handles 'drives special <folder-name>'.
// It accesses and displays information about a well-known special folder (e.g., "documents", "photos").
var drivesSpecialCmd = &cobra.Command{
	Use:   "special <folder-name>",
	Short: "Access a special folder by its well-known name",
	Long: `Accesses and displays metadata for OneDrive special folders like 'documents', 'photos', 'cameraroll', 'music', etc.
Provide the well-known name of the special folder as an argument.`,
	Args: cobra.ExactArgs(1), // Requires exactly one argument: the folder-name.
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives special': %w", err)
		}
		return drivesSpecialLogic(a, cmd, args)
	},
}

// drivesRecentCmd handles 'drives recent'.
// It lists files and folders that have been recently accessed by the user.
var drivesRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "List recently accessed files and folders",
	Long:  `Lists files and folders that have been recently accessed or modified in your OneDrive.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives recent': %w", err)
		}
		return drivesRecentLogic(a, cmd)
	},
}

// drivesSharedCmd handles 'drives shared'.
// It lists all files and folders that have been shared with the authenticated user by others.
var drivesSharedCmd = &cobra.Command{
	Use:   "shared",
	Short: "List items shared with you by others",
	Long:  `Lists all files and folders that have been shared with you by other OneDrive users.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app for 'drives shared': %w", err)
		}
		return drivesSharedLogic(a, cmd)
	},
}

// drivesListLogic contains the core logic for the 'drives list' command.
func drivesListLogic(a *app.App, cmd *cobra.Command) error {
	drives, err := a.SDK.GetDrives(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetching drives list: %w", err)
	}
	ui.DisplayDrives(drives)
	return nil
}

// drivesQuotaLogic contains the core logic for the 'drives quota' command.
func drivesQuotaLogic(a *app.App, cmd *cobra.Command) error {
	drive, err := a.SDK.GetDefaultDrive(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetching default drive quota: %w", err)
	}
	ui.DisplayQuota(drive)
	return nil
}

// drivesGetLogic contains the core logic for the 'drives get' command.
func drivesGetLogic(a *app.App, cmd *cobra.Command, driveID string) error {
	drive, err := a.SDK.GetDriveByID(cmd.Context(), driveID)
	if err != nil {
		return fmt.Errorf("fetching drive by ID '%s': %w", driveID, err)
	}
	ui.DisplayDrive(drive)
	return nil
}

// drivesRootLogic contains the core logic for the 'drives root' command.
func drivesRootLogic(a *app.App, cmd *cobra.Command) error {
	items, err := a.SDK.GetRootDriveItems(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetching root drive items: %w", err)
	}
	ui.DisplayDriveItems(items) // Assuming DisplayDriveItems can handle general DriveItemList
	return nil
}

// drivesSearchLogic contains the core logic for the 'drives search' command.
func drivesSearchLogic(a *app.App, cmd *cobra.Command, args []string) error {
	query := args[0]

	paging, err := ui.ParsePagingFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing pagination flags for 'drives search': %w", err)
	}

	// For a drive-level search, SearchDriveItemsWithPaging is used, which searches the entire default drive.
	items, nextLink, err := a.SDK.SearchDriveItemsWithPaging(cmd.Context(), query, paging)
	if err != nil {
		return fmt.Errorf("searching drive with query '%s': %w", query, err)
	}

	ui.DisplaySearchResults(items, query)
	ui.HandleNextPageInfo(nextLink, paging.FetchAll)
	return nil
}

// drivesDeltaLogic contains the core logic for the 'drives delta' command.
func drivesDeltaLogic(a *app.App, cmd *cobra.Command, args []string) error {
	var deltaToken string
	if len(args) > 0 {
		deltaToken = args[0]
	}

	delta, err := a.SDK.GetDelta(cmd.Context(), deltaToken)
	if err != nil {
		return fmt.Errorf("fetching delta changes (token: '%s'): %w", deltaToken, err)
	}

	ui.DisplayDelta(delta)
	return nil
}

// drivesSpecialLogic contains the core logic for the 'drives special' command.
func drivesSpecialLogic(a *app.App, cmd *cobra.Command, args []string) error {
	folderName := args[0]
	item, err := a.SDK.GetSpecialFolder(cmd.Context(), folderName)
	if err != nil {
		return fmt.Errorf("fetching special folder '%s': %w", folderName, err)
	}
	ui.DisplaySpecialFolder(item, folderName)
	return nil
}

// drivesRecentLogic contains the core logic for the 'drives recent' command.
func drivesRecentLogic(a *app.App, cmd *cobra.Command) error {
	items, err := a.SDK.GetRecentItems(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetching recent items: %w", err)
	}
	ui.DisplayRecentItems(items)
	return nil
}

// drivesSharedLogic contains the core logic for the 'drives shared' command.
func drivesSharedLogic(a *app.App, cmd *cobra.Command) error {
	items, err := a.SDK.GetSharedWithMe(cmd.Context())
	if err != nil {
		return fmt.Errorf("getting items shared with you: %w", err)
	}
	ui.DisplaySharedItems(items)
	return nil
}

// init registers the 'drives' command and its subcommands with the root command.
// It also adds pagination flags to commands that support them.
func init() {
	rootCmd.AddCommand(drivesCmd)
	drivesCmd.AddCommand(drivesListCmd)
	drivesCmd.AddCommand(drivesQuotaCmd)
	drivesCmd.AddCommand(drivesGetCmd)
	drivesCmd.AddCommand(drivesActivitiesCmd)
	drivesCmd.AddCommand(drivesRootCmd)
	drivesCmd.AddCommand(drivesSearchCmd)
	drivesCmd.AddCommand(drivesDeltaCmd)
	drivesCmd.AddCommand(drivesSpecialCmd)
	drivesCmd.AddCommand(drivesRecentCmd)
	drivesCmd.AddCommand(drivesSharedCmd)

	// Add pagination flags to commands that support paginated responses.
	ui.AddPagingFlags(drivesActivitiesCmd)
	ui.AddPagingFlags(drivesSearchCmd)
	// Note: Delta, Recent, Shared might also support paging in Graph API,
	// but current SDK methods might not expose it or handle it internally.
	// If SDK methods are updated, add paging flags here accordingly.
}

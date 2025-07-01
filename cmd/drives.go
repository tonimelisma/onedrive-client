package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

var drivesCmd = &cobra.Command{
	Use:   "drives",
	Short: "Manage available drives",
	Long:  "Provides commands to list available OneDrive drives.",
}

var drivesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available drives",
	Long:  "Lists all available OneDrive drives for the current user.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesListLogic(a)
	},
}

var drivesQuotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Get storage quota for the default drive",
	Long:  "Displays the total, used, and remaining storage quota for the default drive.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesQuotaLogic(a)
	},
}

var drivesGetCmd = &cobra.Command{
	Use:   "get <drive-id>",
	Short: "Get metadata for a specific drive by ID",
	Long:  "Retrieves detailed metadata for a specific OneDrive drive using its ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesGetLogic(a, args[0])
	},
}

var drivesActivitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "Show activity history for the entire drive",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("initializing app: %w", err)
		}

		// Parse paging options
		paging, err := ui.ParsePagingFlags(cmd)
		if err != nil {
			return fmt.Errorf("parsing pagination flags: %w", err)
		}

		activities, nextLink, err := app.SDK.GetDriveActivities(paging)
		if err != nil {
			return fmt.Errorf("getting drive activities: %w", err)
		}

		ui.DisplayActivities(activities, "drive")
		ui.HandleNextPageInfo(nextLink, paging.FetchAll)

		return nil
	},
}

var drivesRootCmd = &cobra.Command{
	Use:   "root",
	Short: "List items in the root of the default drive",
	Long:  "Lists all files and folders in the root directory of your default OneDrive.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesRootLogic(a)
	},
}

var drivesSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across the entire drive",
	Long:  "Searches for files and folders across your entire OneDrive by query string.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesSearchLogic(a, cmd, args)
	},
}

var drivesDeltaCmd = &cobra.Command{
	Use:   "delta [delta-token]",
	Short: "Track changes in the drive using delta queries",
	Long:  "Track changes to items in your OneDrive using delta queries. Provides efficient synchronization by only returning changed items.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesDeltaLogic(a, args)
	},
}

var drivesSpecialCmd = &cobra.Command{
	Use:   "special <folder-name>",
	Short: "Access special folders",
	Long:  "Access OneDrive special folders like Documents, Photos, Music, etc.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesSpecialLogic(a, args)
	},
}

var drivesRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "List recently accessed files",
	Long:  "Lists files and folders that have been recently accessed in your OneDrive.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesRecentLogic(a)
	},
}

var drivesSharedCmd = &cobra.Command{
	Use:   "shared",
	Short: "List items shared with you",
	Long:  "Lists all files and folders that have been shared with you from other OneDrive users.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return drivesSharedLogic(a)
	},
}

func drivesListLogic(a *app.App) error {
	drives, err := a.SDK.GetDrives()
	if err != nil {
		return err
	}
	ui.DisplayDrives(drives)
	return nil
}

func drivesQuotaLogic(a *app.App) error {
	drive, err := a.SDK.GetDefaultDrive()
	if err != nil {
		return err
	}
	ui.DisplayQuota(drive)
	return nil
}

func drivesGetLogic(a *app.App, driveID string) error {
	drive, err := a.SDK.GetDriveByID(driveID)
	if err != nil {
		return err
	}
	ui.DisplayDrive(drive)
	return nil
}

func drivesRootLogic(a *app.App) error {
	items, err := a.SDK.GetRootDriveItems()
	if err != nil {
		return err
	}
	ui.DisplayDriveItems(items)
	return nil
}

func drivesSearchLogic(a *app.App, cmd *cobra.Command, args []string) error {
	query := args[0]

	// Parse paging options
	paging, err := ui.ParsePagingFlags(cmd)
	if err != nil {
		return fmt.Errorf("parsing pagination flags: %w", err)
	}

	// Drive-level search always uses SearchDriveItemsWithPaging (no folder scope)
	items, nextLink, err := a.SDK.SearchDriveItemsWithPaging(query, paging)
	if err != nil {
		return fmt.Errorf("searching drive: %w", err)
	}

	ui.DisplaySearchResults(items, query)
	ui.HandleNextPageInfo(nextLink, paging.FetchAll)
	return nil
}

func drivesDeltaLogic(a *app.App, args []string) error {
	var deltaToken string
	if len(args) > 0 {
		deltaToken = args[0]
	}

	delta, err := a.SDK.GetDelta(deltaToken)
	if err != nil {
		return err
	}

	ui.DisplayDelta(delta)
	return nil
}

func drivesSpecialLogic(a *app.App, args []string) error {
	folderName := args[0]
	item, err := a.SDK.GetSpecialFolder(folderName)
	if err != nil {
		return err
	}
	ui.DisplaySpecialFolder(item, folderName)
	return nil
}

func drivesRecentLogic(a *app.App) error {
	items, err := a.SDK.GetRecentItems()
	if err != nil {
		return err
	}
	ui.DisplayRecentItems(items)
	return nil
}

func drivesSharedLogic(a *app.App) error {
	items, err := a.SDK.GetSharedWithMe()
	if err != nil {
		return fmt.Errorf("getting shared items: %w", err)
	}
	ui.DisplaySharedItems(items)
	return nil
}

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

	// Add pagination flags for commands that need them
	ui.AddPagingFlags(drivesActivitiesCmd)
	ui.AddPagingFlags(drivesSearchCmd)
}

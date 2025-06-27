package cmd

import (
	"log"

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
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		if err := drivesListLogic(a); err != nil {
			log.Fatalf("Error: %v", err)
		}
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

func init() {
	rootCmd.AddCommand(drivesCmd)
	drivesCmd.AddCommand(drivesListCmd)
}

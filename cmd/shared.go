package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	cmdItems "github.com/tonimelisma/onedrive-client/cmd/items"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

var sharedCmd = &cobra.Command{
	Use:   "shared",
	Short: "Manage content shared with you",
	Long:  "Provides commands to view and manage content that has been shared with you from other users.",
}

var sharedListCmd = &cobra.Command{
	Use:   "list",
	Short: "List items shared with you",
	Long:  "Lists all files and folders that have been shared with you from other OneDrive users.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			return fmt.Errorf("error creating app: %w", err)
		}
		return sharedListLogic(a, cmd, args)
	},
}

func sharedListLogic(a *app.App, cmd *cobra.Command, args []string) error {
	items, err := a.SDK.GetSharedWithMe()
	if err != nil {
		return fmt.Errorf("getting shared items: %w", err)
	}

	ui.DisplaySharedItems(items)
	return nil
}

func init() {
	cmdItems.ItemsCmd.AddCommand(sharedCmd)
	sharedCmd.AddCommand(sharedListCmd)
}

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

var rootCmd = &cobra.Command{
	Use:   "onedrive-client",
	Short: "A CLI client for OneDrive",
	Long: `A simple CLI client to interact with Microsoft OneDrive.
			It supports basic file operations and will evolve to support background sync.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// The 'auth' command and its subcommands should be allowed to run
		// even if the user is not logged in or a login is pending.
		if cmd.Parent() != nil && cmd.Parent().Name() == "auth" {
			return nil
		}

		// We need a lightweight check that doesn't create a full app instance
		// if we can avoid it.
		_, err := app.NewApp(cmd)
		if err != nil {
			if errors.Is(err, app.ErrLoginPending) {
				// The specific error from app.go contains the user-friendly message.
				fmt.Println(err.Error())
				// We return a special error to stop execution but not show a generic error message.
				return errors.New("login pending")
			}
			if errors.Is(err, onedrive.ErrReauthRequired) {
				// This is not an error condition for the root command,
				// but subcommands will handle it.
				return nil
			}
			// For other errors, we let them bubble up.
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Default action when no command is provided
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Don't print the generic "login pending" error message.
		if err.Error() != "login pending" {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.onedrive-client.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
}

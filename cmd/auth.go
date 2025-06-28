package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/internal/ui"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Provides commands to manage authentication with OneDrive.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Microsoft OneDrive",
	Long: `This command starts the authentication process with Microsoft OneDrive.
You will be prompted to visit a URL and enter a code to authorize the application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadOrCreate()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			cfg.Debug = true
		}
		err = session.Login(cfg)
		if err != nil {
			log.Fatalf("Login failed: %v", err)
		}
		ui.Success("Successfully authenticated.")
	},
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display the current logged-in user",
	Long:  `This command displays the display name and email of the user who is currently logged in.`,
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp(cmd)
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		err = a.WhoAmI()
		if err != nil {
			log.Fatalf("could not get current user: %v", err)
		}
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the current user session",
	Long:  "This command clears the saved user session, effectively logging the user out.",
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp(cmd)
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		err = a.Logout()
		if err != nil {
			log.Fatalf("could not logout: %v", err)
		}
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the current authentication status",
	Long:  `This command checks the current authentication status and displays it.`,
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp(cmd)
		if err != nil {
			log.Fatalf("Error creating app: %v", err)
		}
		err = a.Status()
		if err != nil {
			log.Fatalf("could not get status: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authWhoamiCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

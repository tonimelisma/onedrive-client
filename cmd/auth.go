package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func getAuthSessionFilePath() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}
	sessionDir := filepath.Join(configDir, "sessions")
	return filepath.Join(sessionDir, "auth_session.json"), nil
}

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

		if cfg.Token.AccessToken != "" {
			fmt.Println("You are already logged in. Please run 'auth logout' first.")
			return
		}
		authSessionPath, err := getAuthSessionFilePath()
		if err != nil {
			log.Fatalf("Could not get auth session path: %v", err)
		}
		if _, err := os.Stat(authSessionPath); !os.IsNotExist(err) {
			fmt.Println("A login is already pending. Please complete it or run 'auth logout' to cancel.")
			return
		}

		debug, _ := cmd.Flags().GetBool("debug")
		deviceCodeResp, err := onedrive.InitiateDeviceCodeFlow(config.ClientID, debug)
		if err != nil {
			log.Fatalf("Login failed: %v", err)
		}

		authState := &session.AuthState{
			DeviceCode:      deviceCodeResp.DeviceCode,
			VerificationURI: deviceCodeResp.VerificationURI,
			UserCode:        deviceCodeResp.UserCode,
			Interval:        deviceCodeResp.Interval,
		}

		if err := session.SaveAuthState(authState); err != nil {
			log.Fatalf("Could not save auth session: %v", err)
		}

		fmt.Printf("You have %d minutes to complete the login.\n", deviceCodeResp.ExpiresIn/60)
		fmt.Println(deviceCodeResp.Message)
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the current user session",
	Long:  "This command clears the saved user session, effectively logging the user out.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadOrCreate()
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}
		if err := app.Logout(cfg); err != nil {
			log.Fatalf("could not logout: %v", err)
		}
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the current authentication status",
	Long:  `This command checks and displays the current authentication status, including the logged in user.`,
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp(cmd)
		if err != nil {
			if errors.Is(err, app.ErrLoginPending) {
				fmt.Println(err.Error())
				return
			}
			if errors.Is(err, onedrive.ErrReauthRequired) {
				fmt.Println("You are not logged in. Please run 'onedrive-client auth login'.")
				return
			}
			if err.Error() == "login pending" {
				return
			}
			log.Fatalf("Error creating app: %v", err)
		}

		user, err := a.GetMe()
		if err != nil {
			log.Fatalf("Could not get user information: %v", err)
		}
		fmt.Printf("You are logged in as: %s (%s)\n", user.DisplayName, user.UserPrincipalName)
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

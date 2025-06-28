package cmd

import (
	"errors"
	"fmt"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadOrCreate()
		if err != nil {
			return fmt.Errorf("error loading config: %v", err)
		}

		if cfg.Token.AccessToken != "" {
			fmt.Println("You are already logged in. Please run 'auth logout' first.")
			return nil
		}
		authSessionPath, err := getAuthSessionFilePath()
		if err != nil {
			return fmt.Errorf("could not get auth session path: %v", err)
		}
		if _, err := os.Stat(authSessionPath); !os.IsNotExist(err) {
			fmt.Println("A login is already pending. Please complete it or run 'auth logout' to cancel.")
			return nil
		}

		debug, _ := cmd.Flags().GetBool("debug")
		deviceCodeResp, err := onedrive.InitiateDeviceCodeFlow(config.ClientID, debug)
		if err != nil {
			return fmt.Errorf("login failed: %v", err)
		}

		authState := &session.AuthState{
			DeviceCode:      deviceCodeResp.DeviceCode,
			VerificationURI: deviceCodeResp.VerificationURI,
			UserCode:        deviceCodeResp.UserCode,
			Interval:        deviceCodeResp.Interval,
		}

		if err := session.SaveAuthState(authState); err != nil {
			return fmt.Errorf("could not save auth session: %v", err)
		}

		fmt.Printf("You have %d minutes to complete the login.\n", deviceCodeResp.ExpiresIn/60)
		fmt.Println(deviceCodeResp.Message)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the current user session",
	Long:  "This command clears the saved user session, effectively logging the user out.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadOrCreate()
		if err != nil {
			return fmt.Errorf("error loading config: %v", err)
		}
		if err := app.Logout(cfg); err != nil {
			return fmt.Errorf("could not logout: %v", err)
		}
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the current authentication status",
	Long:  `This command checks and displays the current authentication status, including the logged in user.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.NewApp(cmd)
		if err != nil {
			if errors.Is(err, app.ErrLoginPending) {
				fmt.Println(err.Error())
				return nil
			}
			if errors.Is(err, onedrive.ErrReauthRequired) {
				fmt.Println("You are not logged in. Please run 'onedrive-client auth login'.")
				return nil
			}
			if err.Error() == "login pending" {
				return nil
			}
			return fmt.Errorf("error creating app: %v", err)
		}

		user, err := a.GetMe()
		if err != nil {
			return fmt.Errorf("could not get user information: %v", err)
		}
		fmt.Printf("You are logged in as: %s (%s)\n", user.DisplayName, user.UserPrincipalName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

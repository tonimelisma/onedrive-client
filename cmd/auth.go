// Package cmd (auth.go) defines the Cobra commands related to authentication
// with Microsoft OneDrive. This includes 'auth login', 'auth logout', and 'auth status'.
// These commands interact with the internal app logic to manage OAuth tokens
// and session state.
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

// getAuthSessionFilePath is a helper function to construct the path to the
// authentication session file. This file stores temporary state during the
// device code flow.
func getAuthSessionFilePath() (string, error) {
	// Uses the application's standard configuration directory.
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config directory for auth session: %w", err)
	}
	// Auth session is stored in a 'sessions' subdirectory.
	sessionDir := filepath.Join(configDir, "sessions")
	return filepath.Join(sessionDir, "auth_session.json"), nil
}

// authCmd represents the base 'auth' command.
// It serves as a parent for subcommands like 'login', 'logout', 'status'.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with Microsoft OneDrive",
	Long:  `Provides subcommands to initiate login, clear current session (logout), and check authentication status.`,
	// Run: func(cmd *cobra.Command, args []string) { cmd.Help() }, // Optional: show help if 'auth' is called alone
}

// authLoginCmd handles the 'auth login' command.
// It initiates the OAuth 2.0 Device Code Flow, prompting the user to visit a URL
// and enter a code to authorize the application.
var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Microsoft OneDrive using Device Code Flow",
	Long: `Starts the authentication process with Microsoft OneDrive.
You will be prompted to visit a URL in a web browser and enter a unique code
to authorize this application to access your OneDrive data.

This command does not require you to be previously logged in. If an existing
login session or a pending login attempt is found, it will advise accordingly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load current configuration to check existing token.
		cfg, err := config.LoadOrCreate()
		if err != nil {
			return fmt.Errorf("loading configuration for login: %w", err)
		}

		// Prevent starting a new login if already authenticated.
		if cfg.Token.AccessToken != "" {
			fmt.Println("You are already logged in. To switch accounts or re-authenticate, please run 'onedrive-client auth logout' first.")
			return nil
		}

		// Check if an auth session file already exists (indicating a pending login).
		authSessionPath, err := getAuthSessionFilePath()
		if err != nil {
			return fmt.Errorf("getting auth session path for login: %w", err)
		}
		if _, err := os.Stat(authSessionPath); !os.IsNotExist(err) {
			// A session file exists, meaning a login was started but not completed.
			fmt.Println("A login attempt is already pending. Please complete it by visiting the previously provided URL and entering the code.")
			fmt.Println("Alternatively, run 'onedrive-client auth logout' to cancel the pending attempt and start over.")
			return nil
		}

		// Get debug flag status from command line.
		debug, _ := cmd.Flags().GetBool("debug")
		// Initiate the device code flow via the SDK.
		deviceCodeResp, err := onedrive.InitiateDeviceCodeFlow(config.ClientID, debug)
		if err != nil {
			return fmt.Errorf("login initiation failed: %w", err)
		}

		// Create session manager for auth state management
		sessionMgr, err := session.NewManager()
		if err != nil {
			return fmt.Errorf("creating session manager: %w", err)
		}

		// Persist the device code flow state (device_code, user_code, etc.) to the session file.
		// This allows other commands to complete the authentication if the user authorizes later.
		authState := &session.AuthState{
			DeviceCode:      deviceCodeResp.DeviceCode,
			VerificationURI: deviceCodeResp.VerificationURI,
			UserCode:        deviceCodeResp.UserCode,
			Interval:        deviceCodeResp.Interval, // Polling interval.
		}
		if err := sessionMgr.SaveAuthState(authState); err != nil {
			return fmt.Errorf("saving auth session state failed: %w", err)
		}

		// Display the user_code and verification_uri to the user.
		// ExpiresIn is in seconds, convert to minutes for user-friendliness.
		fmt.Printf("To complete authentication, please open a web browser and go to: \n%s\n", deviceCodeResp.VerificationURI)
		fmt.Printf("Then, enter the following code: %s\n\n", deviceCodeResp.UserCode)
		fmt.Printf("This code will expire in approximately %d minutes.\n", deviceCodeResp.ExpiresIn/60)
		fmt.Println(deviceCodeResp.Message) // This often repeats the core instruction.
		return nil
	},
}

// authLogoutCmd handles the 'auth logout' command.
// It clears the saved user session (OAuth token) and any pending auth session state,
// effectively logging the user out of the application.
var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear the current user session and log out",
	Long:  `Removes the stored authentication token and any pending login state. After logging out, you will need to run 'auth login' again to use commands that require authentication.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration to pass to app.Logout.
		cfg, err := config.LoadOrCreate()
		if err != nil {
			return fmt.Errorf("loading configuration for logout: %w", err)
		}
		// app.Logout handles clearing the token from config and deleting session files.
		if err := app.Logout(cfg); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}
		// Success message is printed by app.Logout.
		return nil
	},
}

// authStatusCmd handles the 'auth status' command.
// It checks and displays the current authentication status, including the
// logged-in user's display name and principal name if authenticated.
var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display the current authentication status",
	Long:  `Checks if you are currently logged in. If authenticated, it displays your user information. If a login is pending, it provides instructions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Attempt to initialize the app. This will try to complete any pending login.
		a, err := app.NewApp(cmd)
		if err != nil {
			// If a login is pending, app.NewApp returns ErrLoginPending with a user message.
			if errors.Is(err, app.ErrLoginPending) {
				fmt.Println(err.Error()) // Display the "Please go to URL..." message.
				return nil
			}
			// If no token is found and no login is pending, ErrReauthRequired is returned.
			if errors.Is(err, onedrive.ErrReauthRequired) {
				fmt.Println("You are not logged in. Please run 'onedrive-client auth login'.")
				return nil
			}
			// Handle other potential errors from app initialization.
			return fmt.Errorf("checking authentication status: %w", err)
		}

		// If app initialization was successful, a valid token exists. Get user info.
		user, err := a.GetMe(cmd.Context())
		if err != nil {
			// This might happen if the token became invalid between app.NewApp and GetMe,
			// or if there's an API issue.
			return fmt.Errorf("could not retrieve user information: %w", err)
		}
		fmt.Printf("You are logged in as: %s (%s)\n", user.DisplayName, user.UserPrincipalName)
		return nil
	},
}

// init registers the 'auth' command and its subcommands with the root command.
func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	// Flags specific to auth subcommands could be added here if needed.
}

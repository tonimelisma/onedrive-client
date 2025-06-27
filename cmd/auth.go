package cmd

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/app"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Provides commands to manage authentication with OneDrive.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Start the authentication process",
	Long:  "Initiates the device code flow for authentication.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := authLoginLogic(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

func authLoginLogic() error {
	resp, err := onedrive.InitiateDeviceCodeFlow(config.ClientID)
	if err != nil {
		return fmt.Errorf("initiating device code flow: %w", err)
	}

	state := &session.AuthState{
		DeviceCode: resp.DeviceCode,
		Expires:    time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
		Interval:   resp.Interval,
	}

	if err := session.SaveAuthState(state); err != nil {
		return fmt.Errorf("saving auth state: %w", err)
	}

	fmt.Println(resp.Message)
	return nil
}

var authConfirmCmd = &cobra.Command{
	Use:   "confirm",
	Short: "Confirm and complete the authentication process",
	Long:  "Verifies the pending authentication and saves the token.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := authConfirmLogic(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

func authConfirmLogic() error {
	authState, err := session.LoadAuthState()
	if err != nil {
		return fmt.Errorf("loading auth state: %w", err)
	}
	if authState == nil {
		return errors.New("no pending authentication found. Please run 'auth login' first")
	}

	token, err := onedrive.VerifyDeviceCode(config.ClientID, authState.DeviceCode)
	if err != nil {
		if errors.Is(err, onedrive.ErrAuthorizationPending) {
			fmt.Println("Authorization is pending. Please complete the sign-in in your browser and run this command again.")
			return nil
		}
		if errors.Is(err, onedrive.ErrAuthorizationDeclined) || errors.Is(err, onedrive.ErrTokenExpired) {
			_ = session.DeleteAuthState()
			return fmt.Errorf("authentication failed: %w. Please try 'auth login' again", err)
		}
		return fmt.Errorf("verifying device code: %w", err)
	}

	cfg, err := config.LoadOrCreate()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}
	cfg.Token = *token
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving token to configuration: %w", err)
	}

	if err := session.DeleteAuthState(); err != nil {
		log.Printf("Warning: could not delete auth session file: %v", err)
	}
	fmt.Println("Authentication successful. You are now logged in.")
	return nil
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the application",
	Long:  "Removes the locally stored credentials.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := authLogoutLogic(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

func authLogoutLogic() error {
	cfg, err := config.LoadOrCreate()
	if err != nil {
		return fmt.Errorf("could not load configuration: %w", err)
	}
	cfg.Token = onedrive.OAuthToken{} // Clear the token
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("error clearing token from configuration: %w", err)
	}
	fmt.Println("You have been logged out.")
	return nil
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  "Checks if the user is currently logged in and displays user info.",
	Run: func(cmd *cobra.Command, args []string) {
		a, err := app.NewApp()
		if err != nil {
			if errors.Is(err, onedrive.ErrReauthRequired) {
				fmt.Println("You are not logged in. Please run 'onedrive-client auth login'.")
				return
			}
			log.Fatalf("Error creating app: %v", err)
		}
		if err := authStatusLogic(a); err != nil {
			log.Fatalf("Error: %v", err)
		}
	},
}

func authStatusLogic(a *app.App) error {
	user, err := a.SDK.GetMe()
	if err != nil {
		return fmt.Errorf("getting user info: %w", err)
	}

	fmt.Printf("You are logged in as: %s (%s)\n", user.DisplayName, user.UserPrincipalName)
	return nil
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authConfirmCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

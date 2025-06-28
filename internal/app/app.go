package app

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

var ErrLoginPending = errors.New("login pending")

type App struct {
	Config *config.Configuration
	Client *http.Client
	SDK    SDK
}

func NewApp(cmd *cobra.Command) (*App, error) {
	cfg, err := config.LoadOrCreate()
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	// Set debug mode from the flag if it was passed.
	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		cfg.Debug = true
	}

	app := &App{
		Config: cfg,
	}

	if app.Config.Debug {
		onedrive.SetLogger(ui.StdLogger{})
	}

	client, err := app.initializeOnedriveClient()
	if err != nil {
		// Forward ErrLoginPending without wrapping
		if errors.Is(err, ErrLoginPending) {
			return nil, err
		}
		return nil, fmt.Errorf("initializing onedrive client: %w", err)
	}
	app.Client = client
	app.SDK = NewOneDriveSDK(client)

	return app, nil
}

func (a *App) initializeOnedriveClient() (*http.Client, error) {
	if a.Config == nil {
		return nil, errors.New("configuration is nil")
	}

	// Step 1: Check for a pending authentication session
	pendingAuth, err := session.LoadAuthState()
	if err != nil {
		return nil, fmt.Errorf("could not load auth state: %w", err)
	}

	if pendingAuth != nil {
		// A pending session exists, try to complete it
		token, err := onedrive.VerifyDeviceCode(config.ClientID, pendingAuth.DeviceCode, a.Config.Debug)
		if err != nil {
			if errors.Is(err, onedrive.ErrAuthorizationPending) {
				// User has not yet completed the browser flow.
				// Return a specific error to be handled by the command layer.
				return nil, fmt.Errorf("%w: Please go to %s and enter code %s", ErrLoginPending, pendingAuth.VerificationURI, pendingAuth.UserCode)
			}
			// A different error occurred (e.g., code expired, user declined).
			// The pending session is now invalid, so we clean it up.
			_ = session.DeleteAuthState()
			return nil, fmt.Errorf("authentication failed. Your login code may have expired. Please try again: %w", err)
		}

		// Success! User has authenticated.
		a.Config.Token = *token
		if err := a.Config.Save(); err != nil {
			return nil, fmt.Errorf("saving token: %w", err)
		}
		// Clean up the now-used auth session file
		if err := session.DeleteAuthState(); err != nil {
			// Log this error, but don't fail the whole operation
			log.Printf("Warning: could not delete auth session file: %v", err)
		}
		fmt.Println("Login successful!")
	}

	ctx, oauthConfig := onedrive.GetOauth2Config(config.ClientID)

	if a.Config.Token.AccessToken == "" {
		return nil, onedrive.ErrReauthRequired
	}

	tokenRefreshCallbackFunc := func(token onedrive.OAuthToken) {
		a.tokenRefreshCallback(token)
	}

	return onedrive.NewClient(ctx, oauthConfig, a.Config.Token, tokenRefreshCallbackFunc), nil
}

func (a *App) tokenRefreshCallback(token onedrive.OAuthToken) {
	if err := a.Config.UpdateToken(token); err != nil {
		log.Fatalf("Error saving updated token to configuration on disk: %v\n", err)
	}
}

// GetMe fetches the current user's information.
func (a *App) GetMe() (onedrive.User, error) {
	return a.SDK.GetMe()
}

// Logout clears the stored credentials and any pending auth session.
func Logout(cfg *config.Configuration) error {
	cfg.Token = onedrive.OAuthToken{}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("could not clear token: %w", err)
	}
	// Also delete any pending auth session
	if err := session.DeleteAuthState(); err != nil {
		// Don't fail the whole logout if this fails, but log it.
		log.Printf("Warning: could not delete auth session file during logout: %v", err)
	}
	ui.Success("You have been logged out.")
	return nil
}

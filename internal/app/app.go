// Package app (app.go) provides the core application logic for the onedrive-client.
// It handles application initialization, configuration loading, SDK setup (including
// authentication management like completing pending device code flows), and provides
// high-level application operations such as GetMe (current user info) and Logout.
// This package acts as an intermediary between the command-line interface (cmd)
// and the underlying OneDrive SDK (pkg/onedrive).
package app

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/internal/logger"
	"github.com/tonimelisma/onedrive-client/internal/session"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// ErrLoginPending is a sentinel error indicating that a device code login
// process has been initiated but not yet completed by the user.
var ErrLoginPending = errors.New("login pending")

// App encapsulates the core application state, including configuration
// and the OneDrive SDK client.
type App struct {
	Config *config.Configuration // Loaded application configuration (tokens, debug settings).
	SDK    SDK                   // Interface to the OneDrive SDK for making API calls.
}

// NewApp creates and initializes a new App instance.
// It loads the application configuration, sets debug mode based on command flags,
// and initializes the OneDrive SDK client. A critical part of SDK initialization
// is handling any pending OAuth device code flow authentications.
//
// The `cmd` parameter is the currently executing Cobra command, used here to
// access global flags like --debug.
func NewApp(cmd *cobra.Command) (*App, error) {
	// Load existing configuration or create a new default one.
	cfg, err := config.LoadOrCreate()
	if err != nil {
		return nil, fmt.Errorf("loading application configuration: %w", err)
	}

	// Set debug mode in the configuration if the --debug flag was passed.
	// This allows the SDK and other parts of the app to enable verbose logging.
	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		cfg.Debug = true
	}

	app := &App{
		Config: cfg,
	}

	// Initialize the OneDrive SDK. This step also handles authentication.
	sdk, err := app.initializeOnedriveSDK()
	if err != nil {
		// If ErrLoginPending is returned, it means an auth flow was started but not completed.
		// We forward this specific error without wrapping so the CLI can provide user instructions.
		if errors.Is(err, ErrLoginPending) {
			return nil, err
		}
		// For other initialization errors, wrap them for context.
		return nil, fmt.Errorf("initializing OneDrive client/SDK: %w", err)
	}
	app.SDK = sdk

	return app, nil
}

// initializeOnedriveSDK sets up the OneDrive SDK client.
// It checks for a pending device code authentication session and attempts to complete it.
// If no pending session and no existing token, it returns ErrReauthRequired.
// If a token exists (either loaded or newly acquired), it configures the SDK client
// with automatic token refresh and persistence.
func (a *App) initializeOnedriveSDK() (SDK, error) {
	if a.Config == nil {
		// This should not happen if NewApp is used, but as a safeguard.
		return nil, errors.New("app configuration is nil during SDK initialization")
	}

	// Create session manager for auth state management
	sessionMgr, err := session.NewManager()
	if err != nil {
		return nil, fmt.Errorf("creating session manager: %w", err)
	}

	// Step 1: Check for a pending authentication session (from device code flow).
	// `sessionMgr.LoadAuthState()` reads the temporary file storing device_code, user_code etc.
	pendingAuth, err := sessionMgr.LoadAuthState()
	if err != nil {
		// Failure to load auth state is an issue, but might not be fatal if a token already exists.
		// However, if a pending flow was expected, this is problematic.
		// For now, we treat it as an error that prevents checking for pending login.
		log.Printf("Warning: could not load pending auth state: %v. Proceeding without checking for pending login.", err)
	}

	if pendingAuth != nil {
		// A pending session exists. Attempt to exchange the device_code for an OAuth token.
		a.Config.DebugPrintln("Pending authentication session found. Attempting to verify device code...")
		token, err := onedrive.VerifyDeviceCode(config.ClientID, pendingAuth.DeviceCode, a.Config.Debug)
		if err != nil {
			// If authorization is still pending (user hasn't entered code in browser).
			if errors.Is(err, onedrive.ErrAuthorizationPending) {
				a.Config.DebugPrintln("Authorization still pending for device code.")
				// Return a specific error with user instructions, to be handled by the command layer.
				return nil, fmt.Errorf("%w: Please go to %s and enter code %s", ErrLoginPending, pendingAuth.VerificationURI, pendingAuth.UserCode)
			}
			// Another error occurred (e.g., code expired, user declined, network issue).
			// The pending session is now considered invalid, so clean it up.
			a.Config.DebugPrintf("Device code verification failed: %v. Deleting pending auth session.", err)
			if delErr := sessionMgr.DeleteAuthState(); delErr != nil {
				log.Printf("Warning: failed to delete invalid pending auth session file: %v", delErr)
			}
			return nil, fmt.Errorf("authentication failed (e.g., code expired or declined). Please try 'auth login' again: %w", err)
		}

		// Success! User has authenticated via device code flow.
		a.Config.DebugPrintln("Device code successfully exchanged for token.")
		a.Config.Token = *token // Store the newly acquired token.
		if err := a.Config.Save(); err != nil {
			// This is a critical failure, as the token might be lost.
			return nil, fmt.Errorf("saving newly acquired token failed: %w", err)
		}

		// Clean up the now-used auth session file.
		if err := sessionMgr.DeleteAuthState(); err != nil {
			// Log this error, but don't fail the whole operation as login was successful.
			log.Printf("Warning: could not delete completed auth session file: %v", err)
		}
		fmt.Println("Login successful!") // Provide user feedback.
	}

	// After attempting to complete any pending login, check if we have a valid access token.
	// If not, the user needs to log in.
	if a.Config.Token.AccessToken == "" {
		a.Config.DebugPrintln("No valid access token found after checking pending auth. Re-authentication required.")
		return nil, onedrive.ErrReauthRequired
	}

	// Define the callback function that will be invoked by the SDK when a token is refreshed.
	// This callback is responsible for persisting the new token.
	onNewToken := func(token *onedrive.Token) error {
		a.Config.DebugPrintln("Access token refreshed by SDK. Persisting new token...")
		a.Config.Token = *token
		if err := a.Config.Save(); err != nil {
			log.Printf("Critical: Failed to save refreshed token: %v", err)
			return fmt.Errorf("saving refreshed token: %w", err)
		}
		a.Config.DebugPrintln("Refreshed token persisted successfully.")
		return nil
	}

	// Configure a logger for the SDK. If debug mode is enabled in the app config,
	// use a structured logger; otherwise, the SDK will use its DefaultLogger (no-op).
	var sdkLogger onedrive.Logger
	if a.Config.Debug {
		sdkLogger = logger.NewDefaultLogger(true) // Debug level logging
		a.Config.DebugPrintln("Debug mode enabled, SDK logging will be active.")
	} else {
		sdkLogger = logger.NewDefaultLogger(false) // Info level logging
	}

	// Create the OneDrive SDK client instance using the current token (which might be newly acquired or loaded),
	// the application's client ID, the token refresh callback, and the logger.
	// The context.Background() is used for the token source operations within the client.
	client := onedrive.NewClient(context.Background(), &a.Config.Token, config.ClientID, onNewToken, sdkLogger)
	a.Config.DebugPrintln("OneDrive SDK client initialized.")
	return client, nil
}

// GetMe fetches the profile information of the currently authenticated user.
// It's a simple wrapper around the SDK's GetMe method.
func (a *App) GetMe(ctx context.Context) (onedrive.User, error) {
	if a.SDK == nil {
		return onedrive.User{}, errors.New("SDK not initialized in App.GetMe")
	}
	return a.SDK.GetMe(ctx)
}

// Logout clears the stored OAuth token from the configuration and deletes any
// pending authentication session files. This effectively logs the user out.
func Logout(cfg *config.Configuration) error {
	if cfg == nil {
		return errors.New("configuration cannot be nil for logout")
	}

	// Create session manager for auth state management
	sessionMgr, err := session.NewManager()
	if err != nil {
		return fmt.Errorf("creating session manager for logout: %w", err)
	}

	cfg.Token = onedrive.Token{} // Clear the token fields.
	if err := cfg.Save(); err != nil {
		// Even if saving fails, proceed to delete auth session to ensure a clean state.
		log.Printf("Warning: could not clear token from config during logout: %v. Attempting to delete auth session anyway.", err)
	}

	// Also delete any pending auth session file to prevent issues on next login.
	if err := sessionMgr.DeleteAuthState(); err != nil {
		// This is less critical than clearing the token but should be logged.
		log.Printf("Warning: could not delete auth session file during logout: %v", err)
	}
	ui.Success("You have been logged out successfully.")
	return nil // Return nil even if warnings occurred, as primary goal (token clear attempt) was made.
}

package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/tonimelisma/onedrive-client/internal/config"
	"github.com/tonimelisma/onedrive-client/internal/ui"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

type App struct {
	Config *config.Configuration
	Client *http.Client
}

func NewApp() (*App, error) {
	cfg, err := config.LoadOrCreate()
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	app := &App{
		Config: cfg,
	}

	if app.Config.Debug {
		onedrive.SetLogger(ui.StdLogger{})
	}

	client, err := app.initializeOnedriveClient()
	if err != nil {
		return nil, fmt.Errorf("initializing onedrive client: %w", err)
	}
	app.Client = client

	return app, nil
}

func (a *App) initializeOnedriveClient() (*http.Client, error) {
	if a.Config == nil {
		return nil, errors.New("configuration is nil")
	}

	ctx, oauthConfig := onedrive.GetOauth2Config(config.ClientID)

	if a.Config.Token.AccessToken == "" {
		err := a.authenticateOnedriveClient(ctx, oauthConfig)
		if err != nil {
			return nil, err
		}
	}

	tokenRefreshCallbackFunc := func(token onedrive.OAuthToken) {
		a.tokenRefreshCallback(token)
	}

	client := onedrive.NewClient(ctx, oauthConfig, a.Config.Token, tokenRefreshCallbackFunc)
	if client == nil {
		return nil, errors.New("client is nil")
	} else {
		return client, nil
	}
}

func (a *App) authenticateOnedriveClient(
	ctx context.Context,
	oauthConfig *onedrive.OAuthConfig,
) (err error) {
	authURL, codeVerifier, err := onedrive.StartAuthentication(ctx, oauthConfig)
	if err != nil {
		return err
	}

	fmt.Println("Visit the following URL in your browser and authorize the app:", authURL)
	fmt.Print("Enter the authorization code: ")

	var redirectURL string
	fmt.Scan(&redirectURL)

	parsedUrl, err := url.Parse(redirectURL)
	if err != nil {
		return fmt.Errorf("parsing redirect URL: %v", err)
	}

	code := parsedUrl.Query().Get("code")
	if code == "" {
		return fmt.Errorf("authorization code not found in the URL")
	}

	token, err := onedrive.CompleteAuthentication(
		ctx,
		oauthConfig,
		code,
		codeVerifier,
	)
	if err != nil {
		return err
	}

	a.Config.Token = *token
	err = a.Config.Save()
	if err != nil {
		return err
	}

	return nil
}

func (a *App) tokenRefreshCallback(token onedrive.OAuthToken) {
	a.Config.Token = token
	if err := a.Config.Save(); err != nil {
		log.Fatalf("Error saving updated token to configuration on disk: %v\n", err)
	}
}

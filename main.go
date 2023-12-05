package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/tonimelisma/onedrive-sdk-go"
	"golang.org/x/oauth2"
)

const configDir = ".onedrive-client"
const configFile = "config.json"
const clientID = "71ae7ad2-0207-4618-90d3-d21db38f9f7a"

// Logging

type StdLogger struct{}

func (l StdLogger) Debug(v ...interface{}) {
	log.Println(v...)
}

// Config

type configuration struct {
	Token onedrive.OAuthToken `json:"token"`
	Debug bool                `json:"debug"`
	mu    sync.RWMutex        // Add mutex for thread-safe writes
}

func (c *configuration) save() error {
	c.mu.Lock() // Lock for writing
	defer c.mu.Unlock()

	jsonData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config to JSON: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %v", err)
	}

	configDirPath := filepath.Join(homeDir, configDir)
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		if err := os.Mkdir(configDirPath, 0700); err != nil {
			return fmt.Errorf("creating config directory: %v", err)
		}
	}

	configFilePath := filepath.Join(configDirPath, configFile)
	if err := os.WriteFile(configFilePath, jsonData, 0600); err != nil {
		return fmt.Errorf("writing configuration file: %v", err)
	}

	return nil
}

// Main
func main() {
	config, err := loadConfiguration()
	if err != nil {
		handleConfigurationError(err, config)
	}

	config.Debug = true
	client, err := initializeOnedriveClient(config)
	if err != nil {
		log.Fatalf("Error during OneDrive client initialization: %v\n", err)
	}

	if config.Debug {
		onedrive.SetLogger(StdLogger{})
	}

	fmt.Println("Getting drives...")
	err = onedrive.GetMyDrives(client)
	if err != nil {
		if errors.Is(err, onedrive.ErrReauthRequired) {
			log.Fatalf("Re-authenticate: %v\n", err)
		} else if errors.Is(err, onedrive.ErrRetryLater) {
			// Do retry logic
			log.Fatalf("Retry later: %v\n", err)
		} else {
			log.Fatalf("Unknown error: %v\n", err)
		}
	}
}

func authenticateOnedriveClient(config *configuration,
	ctx context.Context, oauthConfig *oauth2.Config) (err error) {

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
		ctx, oauthConfig,
		code,
		codeVerifier,
	)
	if err != nil {
		return err
	}

	config.Token = *token
	err = config.save()
	if err != nil {
		return err
	}

	return nil
}

func tokenRefreshCallback(config *configuration, token *oauth2.Token) {
	config.Token = onedrive.OAuthToken(*token)
	if err := config.save(); err != nil {
		log.Fatalf("Error saving updated token to configuration on disk: %v\n", err)
	}
}

func initializeOnedriveClient(config *configuration) (*http.Client, error) {
	if config == nil {
		return nil, errors.New("configuration is nil")
	}

	ctx, oauthConfig := onedrive.GetOauth2Config(clientID)

	if config.Token.AccessToken == "" {
		err := authenticateOnedriveClient(config, ctx, oauthConfig)
		if err != nil {
			return nil, err
		}
	}

	tokenRefreshCallbackFunc := func(token *oauth2.Token) {
		tokenRefreshCallback(config, token)
	}

	client := onedrive.NewClient(ctx, oauthConfig, config.Token, tokenRefreshCallbackFunc)
	if client == nil {
		return nil, errors.New("client is nil")
	} else {
		return client, nil
	}
}

func loadConfiguration() (*configuration, error) {
	config := &configuration{} // Create a pointer to a new configuration
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %v", err)
	}

	configPath := filepath.Join(homeDir, configDir, configFile)
	fileHandle, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(fileHandle, config)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling json: %v", err)
	}

	return config, nil
}

func handleConfigurationError(err error, config *configuration) {
	if os.IsNotExist(err) {
		fmt.Println("No configuration file found. Proceeding with authentication.")
		config.Token = onedrive.OAuthToken{} // Reset the token
	} else {
		log.Fatalf("Couldn't load configuration: %v\n", err)
	}
}

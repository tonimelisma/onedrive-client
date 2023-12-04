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

	"github.com/tonimelisma/onedrive-sdk-go"
	"golang.org/x/oauth2"
)

const configDir = ".onedrive-client"
const configFile = "config.json"

// Logging

type StdLogger struct{}

func (l StdLogger) Debug(v ...interface{}) {
	log.Println(v...)
}

// Config

type Configuration struct {
	Token onedrive.OAuthToken `json:"token"`
	Debug bool                `json:"debug"`
}

// Main

func main() {
	config, err := loadConfiguration()
	if err != nil {
		handleConfigurationError(err, &config)
	}

	config.Debug = true
	client, err := initializeOnedriveClient(&config)
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

func authenticateOnedriveClient(
	config *Configuration,
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
	err = saveConfiguration(*config)
	if err != nil {
		return err
	}

	return nil
}

func initializeOnedriveClient(config *Configuration) (*http.Client, error) {
	if config == nil {
		return nil, errors.New("configuration is nil")
	}

	ctx, oauthConfig := onedrive.GetOauth2Config(
		"71ae7ad2-0207-4618-90d3-d21db38f9f7a",
	)

	if config.Token.AccessToken == "" {
		err := authenticateOnedriveClient(config, ctx, oauthConfig)
		if err != nil {
			return nil, err
		}
	}

	client := onedrive.NewClient(ctx, oauthConfig, config.Token)
	if client == nil {
		return nil, errors.New("client is nil")
	} else {
		return client, nil
	}
}

func loadConfiguration() (Configuration, error) {
	var config Configuration
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, fmt.Errorf("getting home directory: %v", err)
	}

	configPath := filepath.Join(homeDir, configDir, configFile)
	fileHandle, err := os.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(fileHandle, &config)
	if err != nil {
		return config, fmt.Errorf("unmarshalling json: %v", err)
	}

	return config, nil
}

func saveConfiguration(config Configuration) error {
	jsonData, err := json.MarshalIndent(config, "", "  ")
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

func handleConfigurationError(err error, config *Configuration) {
	if os.IsNotExist(err) {
		fmt.Println("No configuration file found. Proceeding with authentication.")
		config.Token = onedrive.OAuthToken{} // Reset the token
	} else {
		log.Fatalf("Couldn't load configuration: %v\n", err)
	}
}

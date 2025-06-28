package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

const configDir = ".onedrive-client"
const configFile = "config.json"
const ClientID = "57caa7f2-c679-440c-8de2-f8ec86510722"

var ErrConfigNotFound = errors.New("configuration file not found")

// Configuration struct holds all the application's persisted settings.
// It includes the OAuth token and a debug flag.
type Configuration struct {
	Token onedrive.OAuthToken `json:"token"`
	Debug bool                `json:"debug"`
	mu    sync.RWMutex        // Add mutex for thread-safe writes
}

// getConfigPath determines the path for the configuration file.
// It prioritizes the ONEDRIVE_CONFIG_PATH environment variable if set.
func getConfigPath() (string, error) {
	if customPath := os.Getenv("ONEDRIVE_CONFIG_PATH"); customPath != "" {
		return customPath, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %v", err)
	}

	configDirPath := filepath.Join(homeDir, configDir)
	return filepath.Join(configDirPath, configFile), nil
}

// Save persists the configuration struct to a JSON file on disk.
func (c *Configuration) Save() error {
	c.mu.Lock() // Lock for writing
	defer c.mu.Unlock()

	jsonData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling config to JSON: %v", err)
	}

	configFilePath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Ensure the directory exists if we are not using a custom file path
	if os.Getenv("ONEDRIVE_CONFIG_PATH") == "" {
		configDirPath := filepath.Dir(configFilePath)
		if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
			if err := os.Mkdir(configDirPath, 0700); err != nil {
				return fmt.Errorf("creating config directory: %v", err)
			}
		}
	}

	if err := os.WriteFile(configFilePath, jsonData, 0600); err != nil {
		return fmt.Errorf("writing configuration file: %v", err)
	}

	return nil
}

// Load reads the configuration file from disk and returns a Configuration struct.
func Load() (*Configuration, error) {
	config := &Configuration{} // Create a pointer to a new configuration
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, ErrConfigNotFound
	}

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

// LoadOrCreate attempts to load a configuration file. If it doesn't exist,
// it returns a new, empty configuration struct.
func LoadOrCreate() (*Configuration, error) {
	cfg, err := Load()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return &Configuration{}, nil
		}
		return nil, err
	}
	return cfg, nil
}

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

// GetConfigDir returns the root directory for application configuration files.
func GetConfigDir() (string, error) {
	if customPath := os.Getenv("ONEDRIVE_CONFIG_PATH"); customPath != "" {
		return filepath.Dir(customPath), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %v", err)
	}

	return filepath.Join(homeDir, configDir), nil
}

// getConfigPath determines the path for the configuration file.
// It prioritizes the ONEDRIVE_CONFIG_PATH environment variable if set.
func getConfigPath() (string, error) {
	if customPath := os.Getenv("ONEDRIVE_CONFIG_PATH"); customPath != "" {
		return customPath, nil
	}
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, configFile), nil
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
	configDirPath, err := GetConfigDir()
	if err != nil {
		return err
	}
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		if err := os.Mkdir(configDirPath, 0700); err != nil {
			return fmt.Errorf("creating config directory: %v", err)
		}
	}

	tmpPath := configFilePath + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0600); err != nil {
		return fmt.Errorf("writing temp configuration file: %v", err)
	}
	if err := os.Rename(tmpPath, configFilePath); err != nil {
		return fmt.Errorf("renaming temp configuration file: %v", err)
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

// UpdateToken safely updates the OAuth token using the mutex
func (c *Configuration) UpdateToken(token onedrive.OAuthToken) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Token = token
	return c.Save()
}

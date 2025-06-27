package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

const configDir = ".onedrive-client"
const configFile = "config.json"
const ClientID = "71ae7ad2-0207-4618-90d3-d21db38f9f7a"

// Configuration struct holds all the application's persisted settings.
// It includes the OAuth token and a debug flag.
type Configuration struct {
	Token onedrive.OAuthToken `json:"token"`
	Debug bool                `json:"debug"`
	mu    sync.RWMutex        // Add mutex for thread-safe writes
}

// Save persists the configuration struct to a JSON file on disk.
func (c *Configuration) Save() error {
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

// Load reads the configuration file from disk and returns a Configuration struct.
func Load() (*Configuration, error) {
	config := &Configuration{} // Create a pointer to a new configuration
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

// LoadOrCreate attempts to load a configuration file. If it doesn't exist,
// it returns a new, empty configuration struct.
func LoadOrCreate() (*Configuration, error) {
	config, err := Load()
	if err != nil {
		if os.IsNotExist(err) {
			return &Configuration{}, nil
		}
		return nil, err
	}
	return config, nil
}

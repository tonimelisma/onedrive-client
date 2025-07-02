// Package config (config.go) is responsible for managing the application's
// persistent configuration. This includes loading the configuration from a file,
// saving it, and defining the structure of the configuration data (primarily
// OAuth tokens and debug settings). It also handles locating the configuration
// directory, respecting environment variable overrides for custom paths.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

// Constants defining the default configuration directory and file names.
const (
	configDirDefault = ".onedrive-client" // Default directory name within user's config/home.
	configFile       = "config.json"      // Name of the configuration file.
)

// ClientID is the public client ID for this application, registered with Microsoft identity platform.
// This ID is used for OAuth 2.0 authentication flows.
const ClientID = "57caa7f2-c679-440c-8de2-f8ec86510722"

// ErrConfigNotFound is returned by Load when the configuration file does not exist.
var ErrConfigNotFound = errors.New("configuration file not found")

// HTTPConfig holds HTTP client configuration settings
type HTTPConfig struct {
	Timeout       time.Duration `json:"timeout"`         // HTTP request timeout
	RetryAttempts int           `json:"retry_attempts"`  // Maximum number of retry attempts
	RetryDelay    time.Duration `json:"retry_delay"`     // Initial retry delay
	MaxRetryDelay time.Duration `json:"max_retry_delay"` // Maximum retry delay for exponential backoff
}

// PollingConfig holds configuration for polling operations (copy status, upload status, etc.)
type PollingConfig struct {
	InitialInterval time.Duration `json:"initial_interval"` // Initial polling interval
	MaxInterval     time.Duration `json:"max_interval"`     // Maximum polling interval
	Multiplier      float64       `json:"multiplier"`       // Exponential backoff multiplier
	MaxAttempts     int           `json:"max_attempts"`     // Maximum polling attempts (0 = unlimited)
}

// DefaultHTTPConfig returns sensible default HTTP configuration values
func DefaultHTTPConfig() HTTPConfig {
	return HTTPConfig{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		MaxRetryDelay: 10 * time.Second,
	}
}

// DefaultPollingConfig returns sensible default polling configuration values
func DefaultPollingConfig() PollingConfig {
	return PollingConfig{
		InitialInterval: 2 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      1.5,
		MaxAttempts:     0, // Unlimited by default
	}
}

// Configuration holds all the application's persisted settings.
// It includes the OAuth token for accessing OneDrive and a flag for enabling debug mode.
// A RWMutex is used to ensure thread-safe access and modification, especially during Save.
type Configuration struct {
	Token   onedrive.Token `json:"token"`   // OAuth2 token (access, refresh, expiry).
	Debug   bool           `json:"debug"`   // Flag to enable debug logging throughout the application.
	HTTP    HTTPConfig     `json:"http"`    // HTTP client configuration
	Polling PollingConfig  `json:"polling"` // Polling configuration for async operations
	mu      sync.RWMutex   // Protects concurrent access to the Configuration struct, particularly for Save.
}

// DebugPrintln prints a debug message if Debug mode is enabled in the configuration.
// It's a convenience method for conditional logging based on the Debug flag.
func (c *Configuration) DebugPrintln(v ...interface{}) {
	if c.Debug {
		log.Println(append([]interface{}{"DEBUG:"}, v...)...)
	}
}

// DebugPrintf prints a formatted debug message if Debug mode is enabled.
func (c *Configuration) DebugPrintf(format string, v ...interface{}) {
	if c.Debug {
		log.Printf("DEBUG: "+format, v...)
	}
}

// GetConfigDir returns the root directory path for application configuration files.
// It prioritizes the `ONEDRIVE_CONFIG_PATH` environment variable if set (using its directory part).
// Otherwise, it defaults to a subdirectory within the user's standard configuration directory
// (e.g., `~/.config/onedrive-client` on Linux, `~/Library/Application Support/onedrive-client` on macOS).
func GetConfigDir() (string, error) {
	// Allow overriding the entire config file path via an environment variable.
	// If ONEDRIVE_CONFIG_PATH points to a file, its directory is used as the config dir.
	if customPath := os.Getenv("ONEDRIVE_CONFIG_PATH"); customPath != "" {
		// This ensures that if a custom *file* path is given, related files (like sessions)
		// are stored in the same directory as that custom config file.
		return filepath.Dir(customPath), nil
	}

	// Default to user's OS-specific configuration directory.
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config directory: %w", err)
	}
	return filepath.Join(userConfigDir, configDirDefault), nil
}

// getConfigPath determines the full path for the main `config.json` file.
// It prioritizes the `ONEDRIVE_CONFIG_PATH` environment variable if set directly to a file path.
// Otherwise, it constructs the path using the directory from GetConfigDir and the default configFile name.
func getConfigPath() (string, error) {
	// If ONEDRIVE_CONFIG_PATH is set and is intended to be the direct file path.
	if customPath := os.Getenv("ONEDRIVE_CONFIG_PATH"); customPath != "" {
		// Check if it looks like a directory or a file. If it ends with a separator,
		// or if it's an existing directory, treat it as a directory.
		// For simplicity here, we assume if ONEDRIVE_CONFIG_PATH is set, it's the full file path.
		// GetConfigDir() handles the directory part correctly if customPath is a file.
		// If customPath was intended as a directory, the user should ensure it doesn't look like a file.
		// A more robust check might involve os.Stat, but this can be complex if the path doesn't exist yet.
		// The current logic in GetConfigDir (filepath.Dir) handles this by taking the dir part.
		// If customPath is just "myconfig.json", GetConfigDir returns ".", so it's relative.
		// If customPath is "/custom/dir/myconfig.json", GetConfigDir returns "/custom/dir".
		// This logic seems to imply ONEDRIVE_CONFIG_PATH should be the full file path.
		return customPath, nil
	}

	// If no custom path, use the default directory and file name.
	configDirPath, err := GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("determining config directory for config path: %w", err)
	}
	return filepath.Join(configDirPath, configFile), nil
}

// Save persists the current Configuration struct to a JSON file on disk.
// It uses a write lock to ensure thread safety.
// The save operation is atomic: it writes to a temporary file first, then renames it
// to the actual configuration file path. This prevents data corruption if the
// application is interrupted during the save process.
func (c *Configuration) Save() error {
	c.mu.Lock() // Acquire a write lock.
	defer c.mu.Unlock()
	return c.saveUnlocked() // Call the internal save method.
}

// saveUnlocked performs the actual file write operation without acquiring a lock.
// It assumes the caller (Save method) has already acquired the necessary lock.
// This separation allows potential internal use without re-locking if already locked.
func (c *Configuration) saveUnlocked() error {
	c.DebugPrintln("Attempting to save configuration...")
	jsonData, err := json.MarshalIndent(c, "", "  ") // Pretty-print JSON.
	if err != nil {
		return fmt.Errorf("marshalling configuration to JSON: %w", err)
	}

	configFilePath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("getting config file path for save: %w", err)
	}
	c.DebugPrintf("Configuration file path for save: %s", configFilePath)

	// Ensure the configuration directory exists.
	// This is important especially on first run or if the directory was deleted.
	configDirPath := filepath.Dir(configFilePath) // Get directory from the full file path.
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		c.DebugPrintf("Configuration directory '%s' does not exist, creating...", configDirPath)
		if err := os.MkdirAll(configDirPath, 0700); err != nil { // 0700: User rwx, no group/other.
			return fmt.Errorf("creating config directory '%s': %w", configDirPath, err)
		}
	}

	// Atomic save: Write to a temporary file first, then rename.
	// This prevents a corrupted config file if the process is interrupted during write.
	tmpPath := configFilePath + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0600); err != nil { // 0600: User rw, no group/other.
		return fmt.Errorf("writing temporary configuration file '%s': %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, configFilePath); err != nil {
		return fmt.Errorf("renaming temporary configuration file '%s' to '%s': %w", tmpPath, configFilePath, err)
	}
	c.DebugPrintf("Configuration saved successfully to %s", configFilePath)
	return nil
}

// Load reads the configuration file from disk and unmarshals it into a Configuration struct.
// If the configuration file is not found, it returns ErrConfigNotFound.
func Load() (*Configuration, error) {
	cfg := &Configuration{} // Initialize with a new Configuration struct.
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("getting config file path for load: %w", err)
	}

	// Check if the config file exists.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("Debug: Configuration file '%s' not found.", configPath)
		return nil, ErrConfigNotFound
	}

	log.Printf("Debug: Loading configuration from '%s'", configPath)
	fileData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading configuration file '%s': %w", configPath, err)
	}

	// Unmarshal the JSON data into the Configuration struct.
	if err = json.Unmarshal(fileData, cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling configuration JSON from '%s': %w", configPath, err)
	}
	log.Printf("Debug: Configuration loaded successfully. Debug mode: %v", cfg.Debug)
	return cfg, nil
}

// LoadOrCreate attempts to load an existing configuration file. If the file
// doesn't exist (ErrConfigNotFound), it returns a new, empty (default)
// Configuration struct without error. For any other error during loading,
// it returns that error. This is useful for application startup where a missing
// config file is not a fatal error but means starting with defaults.
func LoadOrCreate() (*Configuration, error) {
	cfg, err := Load()
	if err != nil {
		// If the specific error is ErrConfigNotFound, it means no config file exists.
		// In this case, return a new, empty Configuration struct with defaults.
		if errors.Is(err, ErrConfigNotFound) {
			log.Println("Debug: No existing configuration file found. Creating new default configuration.")
			return &Configuration{
				HTTP:    DefaultHTTPConfig(),
				Polling: DefaultPollingConfig(),
			}, nil
		}
		// For any other error during Load, propagate it.
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Configuration loaded successfully. Ensure defaults for missing fields.
	// This provides backward compatibility for existing config files.
	if cfg.HTTP.Timeout == 0 {
		cfg.HTTP = DefaultHTTPConfig()
	}
	if cfg.Polling.InitialInterval == 0 {
		cfg.Polling = DefaultPollingConfig()
	}

	// Configuration loaded successfully.
	return cfg, nil
}

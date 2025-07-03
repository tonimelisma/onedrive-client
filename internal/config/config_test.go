package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tonimelisma/onedrive-client/pkg/onedrive"
)

func TestDefaultHTTPConfig(t *testing.T) {
	config := DefaultHTTPConfig()
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
	assert.Equal(t, 10*time.Second, config.MaxRetryDelay)
}

func TestDefaultPollingConfig(t *testing.T) {
	config := DefaultPollingConfig()
	assert.Equal(t, 2*time.Second, config.InitialInterval)
	assert.Equal(t, 30*time.Second, config.MaxInterval)
	assert.Equal(t, 1.5, config.Multiplier)
	assert.Equal(t, 0, config.MaxAttempts)
}

func TestLoadOrCreateWithDefaults(t *testing.T) {
	// Test creating new configuration (no existing file)
	tempDir := t.TempDir()
	tempConfigFile := filepath.Join(tempDir, "config.json")

	t.Setenv("ONEDRIVE_CONFIG_PATH", tempConfigFile)

	cfg, err := LoadOrCreate()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should have default HTTP config
	assert.Equal(t, 30*time.Second, cfg.HTTP.Timeout)
	assert.Equal(t, 3, cfg.HTTP.RetryAttempts)

	// Should have default polling config
	assert.Equal(t, 2*time.Second, cfg.Polling.InitialInterval)
	assert.Equal(t, 30*time.Second, cfg.Polling.MaxInterval)
	assert.Equal(t, 1.5, cfg.Polling.Multiplier)
}

func TestLoadOrCreateBackwardCompatibility(t *testing.T) {
	// Test loading existing configuration without new fields (backward compatibility)
	tempDir := t.TempDir()
	tempConfigFile := filepath.Join(tempDir, "config.json")

	// Create an old-style config file without HTTP/Polling fields
	oldConfig := map[string]interface{}{
		"token": map[string]interface{}{
			"access_token": "test-token",
		},
		"debug": false,
	}

	configData, err := json.MarshalIndent(oldConfig, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(tempConfigFile, configData, onedrive.PermSecureFile)
	require.NoError(t, err)

	t.Setenv("ONEDRIVE_CONFIG_PATH", tempConfigFile)

	cfg, err := LoadOrCreate()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should have loaded the token correctly
	assert.Equal(t, "test-token", cfg.Token.AccessToken)
	assert.False(t, cfg.Debug)

	// Should have applied default HTTP config for missing fields
	assert.Equal(t, 30*time.Second, cfg.HTTP.Timeout)
	assert.Equal(t, 3, cfg.HTTP.RetryAttempts)

	// Should have applied default polling config for missing fields
	assert.Equal(t, 2*time.Second, cfg.Polling.InitialInterval)
	assert.Equal(t, 30*time.Second, cfg.Polling.MaxInterval)
	assert.Equal(t, 1.5, cfg.Polling.Multiplier)
}

func TestConfigurationSaveWithNewFields(t *testing.T) {
	// Test saving configuration with new HTTP/Polling fields
	tempDir := t.TempDir()
	tempConfigFile := filepath.Join(tempDir, "config.json")

	t.Setenv("ONEDRIVE_CONFIG_PATH", tempConfigFile)

	cfg := &Configuration{
		Debug: true,
		HTTP: HTTPConfig{
			Timeout:       45 * time.Second,
			RetryAttempts: 5,
			RetryDelay:    2 * time.Second,
			MaxRetryDelay: 20 * time.Second,
		},
		Polling: PollingConfig{
			InitialInterval: 3 * time.Second,
			MaxInterval:     60 * time.Second,
			Multiplier:      2.0,
			MaxAttempts:     10,
		},
	}

	err := cfg.Save()
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, tempConfigFile)

	// Load the configuration back and verify fields
	loadedCfg, err := Load()
	require.NoError(t, err)

	assert.True(t, loadedCfg.Debug)
	assert.Equal(t, 45*time.Second, loadedCfg.HTTP.Timeout)
	assert.Equal(t, 5, loadedCfg.HTTP.RetryAttempts)
	assert.Equal(t, 2*time.Second, loadedCfg.HTTP.RetryDelay)
	assert.Equal(t, 20*time.Second, loadedCfg.HTTP.MaxRetryDelay)

	assert.Equal(t, 3*time.Second, loadedCfg.Polling.InitialInterval)
	assert.Equal(t, 60*time.Second, loadedCfg.Polling.MaxInterval)
	assert.Equal(t, 2.0, loadedCfg.Polling.Multiplier)
	assert.Equal(t, 10, loadedCfg.Polling.MaxAttempts)
}

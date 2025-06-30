package e2e

import (
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for E2E tests
type Config struct {
	TestDir     string
	Timeout     time.Duration
	Cleanup     bool
	MaxFileSize int64
	ChunkSize   int64
}

// LoadConfig loads E2E test configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		TestDir:     getEnvOrDefault("ONEDRIVE_E2E_TEST_DIR", "/E2E-Tests"),
		Timeout:     getTimeoutFromEnv("ONEDRIVE_E2E_TIMEOUT", 300*time.Second),
		Cleanup:     getBoolFromEnv("ONEDRIVE_E2E_CLEANUP", true),
		MaxFileSize: getInt64FromEnv("ONEDRIVE_E2E_MAX_FILE_SIZE", 100*1024*1024), // 100MB
		ChunkSize:   getInt64FromEnv("ONEDRIVE_E2E_CHUNK_SIZE", 320*1024*10),      // 3.2MB
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getTimeoutFromEnv parses timeout from environment variable
func getTimeoutFromEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

// getBoolFromEnv parses boolean from environment variable
func getBoolFromEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return result
}

// getInt64FromEnv parses int64 from environment variable
func getInt64FromEnv(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return result
}

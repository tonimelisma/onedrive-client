package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

const configDir = ".onedrive-client"
const configFile = "config.json"

type Configuration struct {
	Token oauth2.Token `json:"token"`
	Debug bool         `json:"debug"`
}

func loadConfiguration() (thisConfig Configuration, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return thisConfig, fmt.Errorf("getting home directory: %v", err)
	}

	configPath := filepath.Join(homeDir, configDir, configFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return thisConfig, err
	}

	fileHandle, err := os.ReadFile(configPath)
	if err != nil {
		return thisConfig, fmt.Errorf("reading file: %v", err)
	}

	err = json.Unmarshal([]byte(fileHandle), &thisConfig)
	if err != nil {
		return thisConfig, fmt.Errorf("unmarshalling json: %v", err)
	}

	return thisConfig, nil
}

func saveConfiguration(thisConfig Configuration) (err error) {
	jsonData, err := json.MarshalIndent(thisConfig, "", "")
	if err != nil {
		return fmt.Errorf("marshalling json: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %v", err)
	}

	configDirPath := filepath.Join(homeDir, configDir)
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		err = os.Mkdir(configDirPath, 0700)
		if err != nil {
			return fmt.Errorf("creating configuration directory: %v", err)
		}
	}

	configFilePath := filepath.Join(configDirPath, configFile)
	err = os.WriteFile(configFilePath, jsonData, 0600)
	if err != nil {
		return fmt.Errorf("writing configuration: %v", err)
	}

	return nil
}

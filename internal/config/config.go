// internal/config/config.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LoadConfig loads and validates configuration from the specified JSON file.
// If the file doesn't exist or fails to parse, returns default configuration.
func LoadConfig(path string) (*ProfileConfiguration, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return defaults
		return DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), err
	}

	// Parse JSON
	var config ProfileConfiguration
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig(), err
	}

	// Validate and normalize
	if err := config.Validate(); err != nil {
		return DefaultConfig(), err
	}

	return &config, nil
}

// LoadDefaultConfig loads configuration from default location "profiles.json"
// in the current working directory.
func LoadDefaultConfig() (*ProfileConfiguration, error) {
	// Try current directory first
	configPath := "profiles.json"
	if _, err := os.Stat(configPath); err == nil {
		return LoadConfig(configPath)
	}

	// Try config directory relative to executable
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		configPath = filepath.Join(exeDir, "profiles.json")
		if _, err := os.Stat(configPath); err == nil {
			return LoadConfig(configPath)
		}
	}

	// No config file found, return defaults
	return DefaultConfig(), nil
}

// SaveConfig writes configuration to the specified JSON file.
func SaveConfig(config *ProfileConfiguration, path string) error {
	// Validate before saving
	if err := config.Validate(); err != nil {
		return err
	}

	// Marshal with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Write file
	return os.WriteFile(path, data, 0644)
}

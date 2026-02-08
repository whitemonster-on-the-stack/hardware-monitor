package config

import (
	"encoding/json"
	"os"
)

// LoadConfig loads the configuration from the specified path.
// If the file does not exist or is invalid, it returns the default configuration.
func LoadConfig(path string) (*ProfileConfiguration, error) {
	if path == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// If file doesn't exist, return default without error (optional: return err if strict)
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	var cfg ProfileConfiguration
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// Validate critical fields
	if cfg.ColumnWidths == nil {
		cfg.ColumnWidths = DefaultConfig().ColumnWidths
	}
	if cfg.RefreshInterval < 100 {
		cfg.RefreshInterval = 100
	}

	return &cfg, nil
}

// SaveConfig saves the current configuration to the specified path.
func SaveConfig(path string, cfg *ProfileConfiguration) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// DefaultConfig returns the hardcoded default configuration.
func DefaultConfig() *ProfileConfiguration {
	return &ProfileConfiguration{
		Theme: "lich-king",
		ColumnWidths: map[string]float64{
			"gpu":     0.30,
			"process": 0.40,
			"cpu":     0.30,
		},
		RefreshInterval:  1000,
		MaxProcesses:     200,
		GPUHistoryLength: 100,
		ShowTooltips:     true,
		AlertThresholds: AlertThresholds{
			CPUUsagePercent:    90.0,
			CPUTempCelsius:     85.0,
			GPUUsagePercent:    98.0,
			GPUTempCelsius:     85.0,
			MemoryUsagePercent: 95.0,
			DiskUsagePercent:   90.0,
		},
	}
}

// LoadConfig reads the configuration from the specified path.
// If the file does not exist or is invalid, it returns the default configuration.
func LoadConfig(path string) (*ProfileConfiguration, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, try to write defaults
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err == nil {
			_ = ioutil.WriteFile(path, data, 0644)
		}
		return cfg, nil
	}

	// Read file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	// Parse JSON
	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// SaveConfig writes the configuration to the specified path.
func SaveConfig(path string, cfg *ProfileConfiguration) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

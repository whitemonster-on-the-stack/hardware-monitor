package config

// ProfileConfiguration defines the user-configurable settings for OmniTop.
type ProfileConfiguration struct {
	Theme            string             `json:"theme"`
	ColumnWidths     map[string]float64 `json:"column_widths"`
	RefreshInterval  int                `json:"refresh_interval"` // In milliseconds
	MaxProcesses     int                `json:"max_processes"`
	GPUHistoryLength int                `json:"gpu_history_length"`
	ShowTooltips     bool               `json:"show_tooltips"`
	EnabledModules   []string           `json:"enabled_modules"`
}

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
		GPUHistoryLength: 60,
		ShowTooltips:     true,
		EnabledModules:   []string{"gpu", "process", "cpu", "net", "disk"},
	}
}

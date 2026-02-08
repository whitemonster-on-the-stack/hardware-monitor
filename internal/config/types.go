// internal/config/types.go
package config

// ProfileConfiguration defines user-configurable settings for OmniTop.
type ProfileConfiguration struct {
	// Theme defines the color scheme.
	// Valid values: "lich-king", "default", "custom"
	Theme string `json:"theme"`

	// ColumnWidths defines the percentage widths for each column.
	// Keys: "gpu", "process", "cpu"
	// Values: percentage (0.0-1.0)
	ColumnWidths map[string]float64 `json:"columnWidths"`

	// RefreshInterval defines milliseconds between updates.
	// Minimum: 250, Maximum: 5000
	RefreshInterval int `json:"refreshInterval"`

	// MaxProcesses defines maximum processes to display in the process list.
	// Minimum: 10, Maximum: 1000
	MaxProcesses int `json:"maxProcesses"`

	// GPUHistoryLength defines number of historical GPU utilization points.
	// Minimum: 10, Maximum: 500
	GPUHistoryLength int `json:"gpuHistoryLength"`

	// ShowTooltips defines whether to display hover tooltips.
	ShowTooltips bool `json:"showTooltips"`
}

// Validate checks if configuration values are within acceptable ranges.
func (c *ProfileConfiguration) Validate() error {
	// Validate theme
	validThemes := map[string]bool{
		"lich-king": true,
		"default":   true,
		"custom":    true,
	}
	if !validThemes[c.Theme] {
		c.Theme = "lich-king"
	}

	// Validate column widths
	if c.ColumnWidths == nil {
		c.ColumnWidths = make(map[string]float64)
	}
	// Ensure all required columns exist
	requiredColumns := []string{"gpu", "process", "cpu"}
	for _, col := range requiredColumns {
		if _, exists := c.ColumnWidths[col]; !exists {
			// Set defaults if missing
			if col == "gpu" {
				c.ColumnWidths[col] = 0.30
			} else if col == "process" {
				c.ColumnWidths[col] = 0.40
			} else {
				c.ColumnWidths[col] = 0.30 // cpu gets remainder
			}
		}
		// Ensure values are within bounds
		if c.ColumnWidths[col] < 0.1 {
			c.ColumnWidths[col] = 0.1
		}
		if c.ColumnWidths[col] > 0.9 {
			c.ColumnWidths[col] = 0.9
		}
	}

	// Validate refresh interval
	if c.RefreshInterval < 250 {
		c.RefreshInterval = 250
	}
	if c.RefreshInterval > 5000 {
		c.RefreshInterval = 5000
	}

	// Validate max processes
	if c.MaxProcesses < 10 {
		c.MaxProcesses = 10
	}
	if c.MaxProcesses > 1000 {
		c.MaxProcesses = 1000
	}

	// Validate GPU history length
	if c.GPUHistoryLength < 10 {
		c.GPUHistoryLength = 10
	}
	if c.GPUHistoryLength > 500 {
		c.GPUHistoryLength = 500
	}

	return nil
}

// DefaultConfig returns the default configuration matching current behavior.
func DefaultConfig() *ProfileConfiguration {
	return &ProfileConfiguration{
		Theme:            "lich-king",
		ColumnWidths:     map[string]float64{"gpu": 0.30, "process": 0.40, "cpu": 0.30},
		RefreshInterval:  1000,
		MaxProcesses:     200,
		GPUHistoryLength: 100,
		ShowTooltips:     true,
	}
}

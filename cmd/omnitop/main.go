package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/omnitop/internal/config"
	"github.com/google/omnitop/internal/metrics"
	"github.com/google/omnitop/internal/ui"
)

func main() {
	// Parse flags
	mockMode := flag.Bool("mock", false, "Run in mock mode with simulated data")
	configPath := flag.String("config", "profiles.json", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig("profiles.json")
	if err != nil {
		log.Printf("Warning: Failed to load profiles.json: %v. Using defaults.", err)
		cfg = config.DefaultConfig()
	}

	// Initialize metrics provider
	var provider metrics.Provider
	if *mockMode {
		log.Println("Starting in MOCK mode...")
		provider = &metrics.MockProvider{}
	} else {
		log.Println("Starting in REAL mode...")
		provider = &metrics.RealProvider{}
	}

	if err := provider.Init(); err != nil {
		log.Fatalf("Failed to initialize metrics provider: %v", err)
	}
	defer provider.Shutdown()

	// Create root model
	root := ui.NewRootModel(provider, cfg)

	// Start Bubble Tea program
	p := tea.NewProgram(root, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running OmniTop: %v\n", err)
		os.Exit(1)
	}
}

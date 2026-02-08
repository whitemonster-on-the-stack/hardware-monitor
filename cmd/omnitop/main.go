package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/omnitop/internal/metrics"
	"github.com/google/omnitop/internal/ui"
)

func main() {
	// Parse flags
	mockMode := flag.Bool("mock", false, "Run in mock mode with simulated data")
	flag.Parse()

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
	root := ui.NewRootModel(provider)

	// Start Bubble Tea program
	p := tea.NewProgram(root, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running OmniTop: %v\n", err)
		os.Exit(1)
	}
}

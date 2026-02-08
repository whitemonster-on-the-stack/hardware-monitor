package metrics

import (
	"testing"
)

func TestMockProvider(t *testing.T) {
	provider := &MockProvider{}
	err := provider.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	stats, err := provider.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats is nil")
	}

	if stats.CPU.GlobalUsagePercent < 0 || stats.CPU.GlobalUsagePercent > 100 {
		t.Errorf("Invalid CPU usage: %f", stats.CPU.GlobalUsagePercent)
	}

	if len(stats.Processes) == 0 {
		t.Error("No processes returned in mock mode")
	}

	if !stats.GPU.Available {
		t.Error("GPU should be available in mock mode")
	}
}

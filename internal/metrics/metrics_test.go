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

	// Verify GPU.MemoryUtil is correctly computed from MemoryUsed/MemoryTotal
	if stats.GPU.MemoryUtil < 0 || stats.GPU.MemoryUtil > 100 {
		t.Errorf("GPU.MemoryUtil out of range: %d", stats.GPU.MemoryUtil)
	}
	
	// Verify MemoryUtil matches computed occupancy from MemoryUsed/MemoryTotal
	if stats.GPU.MemoryTotal > 0 {
		expectedUtil := uint32(float64(stats.GPU.MemoryUsed) / float64(stats.GPU.MemoryTotal) * 100.0)
		if stats.GPU.MemoryUtil != expectedUtil {
			t.Errorf("GPU.MemoryUtil mismatch: got %d, expected %d (from %d/%d)", 
				stats.GPU.MemoryUtil, expectedUtil, stats.GPU.MemoryUsed, stats.GPU.MemoryTotal)
		}
	}
}

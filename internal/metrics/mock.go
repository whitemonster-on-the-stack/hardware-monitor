package metrics

import (
	"math/rand"
	"time"
)

type MockProvider struct {
	lastStats SystemStats
}

func (m *MockProvider) Init() error {
	m.lastStats = SystemStats{
		Timestamp: time.Now(),
		Uptime:    3600,
		CPU: CPUStats{
			PerCoreUsage: make([]float64, 8), // Simulate 8 cores
			PerCoreTemp:  make([]float64, 8),
		},
		GPU: GPUStats{
			Available:      true,
			Name:           "NVIDIA GeForce RTX 4090",
			MemoryTotal:    24576 * 1024 * 1024,
			HistoricalUtil: make([]float64, 60),
			Processes:      make([]GPUProcess, 0),
		},
		Processes: make([]ProcessInfo, 50),
	}
	return nil
}

func (m *MockProvider) GetStats() (*SystemStats, error) {
	// Simulate metric updates with some randomness
	now := time.Now()
	m.lastStats.Timestamp = now
	m.lastStats.Uptime++

	// CPU
	m.lastStats.CPU.GlobalUsagePercent = 20 + rand.Float64()*10
	for i := range m.lastStats.CPU.PerCoreUsage {
		m.lastStats.CPU.PerCoreUsage[i] = 10 + rand.Float64()*30
		m.lastStats.CPU.PerCoreTemp[i] = 40 + rand.Float64()*10
	}
	m.lastStats.CPU.LoadAvg = [3]float64{1.5, 1.2, 0.8}

	// Memory
	m.lastStats.Memory.Total = 32 * 1024 * 1024 * 1024
	m.lastStats.Memory.Used = 12 * 1024 * 1024 * 1024
	m.lastStats.Memory.Free = m.lastStats.Memory.Total - m.lastStats.Memory.Used
	m.lastStats.Memory.UsedPercent = float64(m.lastStats.Memory.Used) / float64(m.lastStats.Memory.Total) * 100
	m.lastStats.Memory.SwapTotal = 8 * 1024 * 1024 * 1024
	m.lastStats.Memory.SwapUsed = 1 * 1024 * 1024 * 1024
	m.lastStats.Memory.SwapPercent = 12.5

	// GPU
	m.lastStats.GPU.Utilization = uint32(50 + rand.Intn(30))
	m.lastStats.GPU.Temperature = uint32(60 + rand.Intn(10))
	m.lastStats.GPU.MemoryUsed = uint64(8 * 1024 * 1024 * 1024)
	m.lastStats.GPU.FanSpeed = uint32(40 + rand.Intn(10))
	m.lastStats.GPU.GraphicsClock = 2500
	m.lastStats.GPU.MemoryClock = 10500
	m.lastStats.GPU.PowerUsage = 150000 // mW
	m.lastStats.GPU.PowerLimit = 450000 // mW
	// Compute VRAM utilization percentage
	if m.lastStats.GPU.MemoryTotal > 0 {
		m.lastStats.GPU.MemoryUtil = uint32(float64(m.lastStats.GPU.MemoryUsed) / float64(m.lastStats.GPU.MemoryTotal) * 100.0)
	}

	// Historical Graph
	m.lastStats.GPU.HistoricalUtil = append(m.lastStats.GPU.HistoricalUtil[1:], float64(m.lastStats.GPU.Utilization))

	// Fake Processes
	users := []string{"root", "jules", "systemd"}
	cmds := []string{"chrome", "code", "go", "kworker", "bash"}
	m.lastStats.Processes = make([]ProcessInfo, 50) // Reset to avoid appending forever
	for i := 0; i < len(m.lastStats.Processes); i++ {
		isGpu := i < 5
		pid := int32(1000 + i)
		m.lastStats.Processes[i] = ProcessInfo{
			PID:        pid,
			User:       users[rand.Intn(len(users))],
			Command:    cmds[rand.Intn(len(cmds))],
			State:      "R",
			CPUPercent: rand.Float64() * 5,
			MemPercent: rand.Float64() * 2,
			IsGPUUser:  isGpu,
			Threads:    int32(1 + rand.Intn(10)),
			Priority:   0,
		}

		if isGpu {
			// Add to GPU process list if not already there (simplified for mock)
			if len(m.lastStats.GPU.Processes) < 5 {
				m.lastStats.GPU.Processes = append(m.lastStats.GPU.Processes, GPUProcess{
					PID:        uint32(pid),
					MemoryUsed: uint64(rand.Int63n(1000) * 1024 * 1024),
				})
			}
		}
	}

	return &m.lastStats, nil
}

func (m *MockProvider) Shutdown() {}

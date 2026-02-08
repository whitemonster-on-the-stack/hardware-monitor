package metrics

import (
	"time"
)

// SystemStats holds the aggregated metrics for the entire system at a point in time.
type SystemStats struct {
	Timestamp time.Time
	Uptime    uint64 // Uptime in seconds
	CPU       CPUStats
	Memory    MemoryStats
	Disk      DiskStats
	Net       NetStats
	GPU       GPUStats
	Processes []ProcessInfo
}

// CPUStats holds CPU related metrics.
type CPUStats struct {
	GlobalUsagePercent float64
	PerCoreUsage       []float64  // Percent usage per core
	PerCoreTemp        []float64  // Temperature per core (if available)
	LoadAvg            [3]float64 // 1, 5, 15 min load average
}

// MemoryStats holds memory related metrics.
type MemoryStats struct {
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
	SwapTotal   uint64
	SwapUsed    uint64
	SwapPercent float64
}

// DiskStats holds disk I/O metrics.
type DiskStats struct {
	ReadBytes  uint64 // Total read bytes
	WriteBytes uint64 // Total write bytes
	ReadSpeed  uint64 // Bytes per second
	WriteSpeed uint64 // Bytes per second
}

// NetStats holds network I/O metrics.
type NetStats struct {
	BytesSent     uint64 // Total bytes sent
	BytesRecv     uint64 // Total bytes received
	UploadSpeed   uint64 // Bytes per second
	DownloadSpeed uint64 // Bytes per second
}

// GPUStats holds NVIDIA GPU metrics.
type GPUStats struct {
	Available      bool // True if GPU is present and accessible
	Name           string
	Utilization    uint32 // GPU Utilization in percent
	MemoryTotal    uint64 // Total VRAM in bytes
	MemoryUsed     uint64 // Used VRAM in bytes
	MemoryUtil     uint32 // Memory utilization in percent
	Temperature    uint32 // GPU Temperature in Celsius
	FanSpeed       uint32 // Fan speed in percent
	GraphicsClock  uint32 // Graphics clock in MHz
	MemoryClock    uint32 // Memory clock in MHz
	PowerUsage     uint32 // Power usage in milliwatts
	PowerLimit     uint32 // Power limit in milliwatts
	Processes      []GPUProcess
	HistoricalUtil []float64 // Last N data points for the big graph
}

// GPUProcess represents a process running on the GPU.
type GPUProcess struct {
	PID        uint32
	Name       string
	MemoryUsed uint64
}

// ProcessInfo represents a system process.
type ProcessInfo struct {
	PID        int32
	User       string
	Command    string
	State      string
	CPUPercent float64
	MemPercent float64
	Memory     uint64 // RSS
	Threads    int32
	Priority   int32 // Nice value
	ParentPID  int32
	IsGPUUser  bool // True if this process is using the GPU
}

// Provider defines the interface for fetching system metrics.
type Provider interface {
	Init() error
	GetStats() (*SystemStats, error)
	Shutdown()
}

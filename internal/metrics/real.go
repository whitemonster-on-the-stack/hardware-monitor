package metrics

import (
	"fmt"
	"log"
	"time"

	"github.com/mindprince/gonvml"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type RealProvider struct {
	hasGPU         bool
	gpuHistoryUtil []float64
	maxHistoryLen  int
	
	// GPU health tracking
	gpuDevice      gonvml.Device // Current GPU device handle
	healthStatus   GPUHealthStatus
	errorCount     int
	lastError      string
	retryAttempts  int
	lastSuccess    time.Time
	circuitOpen    bool // Circuit breaker state
	circuitOpenAt  time.Time
}

func (r *RealProvider) Init() error {
	// Initialize NVML
	if err := gonvml.Initialize(); err != nil {
		log.Printf("NVML initialization failed (GPU metrics unavailable): %v", err)
		r.hasGPU = false
		r.healthStatus = GPUHealthFailed
		r.lastError = fmt.Sprintf("NVML initialization failed: %v", err)
	} else {
		r.hasGPU = true
		r.healthStatus = GPUHealthHealthy
		r.lastSuccess = time.Now()
	}
	r.maxHistoryLen = 100 // Store last 100 data points
	r.gpuHistoryUtil = make([]float64, 0, r.maxHistoryLen)
	return nil
}

// checkCircuitBreaker returns true if circuit is open and should block operations
func (r *RealProvider) checkCircuitBreaker() bool {
	if !r.circuitOpen {
		return false
	}
	// Check if we should allow a retry (30 second cooldown)
	if time.Since(r.circuitOpenAt) > 30*time.Second {
		r.circuitOpen = false
		r.retryAttempts = 0
		return false
	}
	return true
}

// recordError updates error tracking and health status
func (r *RealProvider) recordError(err error, operation string) {
	r.errorCount++
	r.lastError = fmt.Sprintf("%s: %v", operation, err)
	
	// Update health status based on error pattern
	if r.errorCount > 5 {
		r.healthStatus = GPUHealthFailed
		r.circuitOpen = true
		r.circuitOpenAt = time.Now()
		log.Printf("GPU circuit breaker opened after %d errors: %v", r.errorCount, err)
	} else if r.errorCount > 2 {
		r.healthStatus = GPUHealthDegraded
	}
	
	log.Printf("GPU error (%s): %v (total errors: %d)", operation, err, r.errorCount)
}

// recordSuccess resets error tracking when operation succeeds
func (r *RealProvider) recordSuccess() {
	r.errorCount = 0
	r.lastSuccess = time.Now()
	r.retryAttempts = 0
	r.healthStatus = GPUHealthHealthy
	r.circuitOpen = false
}

// tryReinitialize attempts to reinitialize NVML if in degraded/failed state
func (r *RealProvider) tryReinitialize() bool {
	if r.retryAttempts >= 3 {
		return false // Too many retry attempts
	}
	
	r.retryAttempts++
	log.Printf("Attempting GPU re-initialization (attempt %d)", r.retryAttempts)
	
	// Shutdown first if needed
	if r.hasGPU {
		gonvml.Shutdown()
		r.hasGPU = false
	}
	
	// Reinitialize
	if err := gonvml.Initialize(); err != nil {
		r.recordError(err, "NVML re-initialization")
		return false
	}
	
	r.hasGPU = true
	r.recordSuccess()
	log.Printf("GPU re-initialization successful")
	return true
}

// safeGPUMetric executes a GPU metric function with error handling
func (r *RealProvider) safeGPUMetric(fn func() error, metricName string) bool {
	if !r.hasGPU || r.checkCircuitBreaker() {
		return false
	}
	
	if err := fn(); err != nil {
		r.recordError(err, metricName)
		
		// Try reinitialization on first error
		if r.errorCount == 1 {
			r.tryReinitialize()
		}
		return false
	}
	
	// Reset error count on successful operation streak
	if r.errorCount > 0 {
		r.errorCount = 0
	}
	return true
}

func (r *RealProvider) GetStats() (*SystemStats, error) {
	stats := &SystemStats{
		Timestamp: time.Now(),
	}

	// CPU
	cpuPercent, err := cpu.Percent(0, true)
	if err == nil {
		stats.CPU.PerCoreUsage = cpuPercent
		stats.CPU.GlobalUsagePercent = 0
		for _, p := range cpuPercent {
			stats.CPU.GlobalUsagePercent += p
		}
		if len(cpuPercent) > 0 {
			stats.CPU.GlobalUsagePercent /= float64(len(cpuPercent))
		}
	}

	// Memory
	vm, err := mem.VirtualMemory()
	if err == nil {
		stats.Memory.Total = vm.Total
		stats.Memory.Used = vm.Used
		stats.Memory.Free = vm.Free
		stats.Memory.UsedPercent = vm.UsedPercent
	}

	// Disk
	parts, _ := disk.Partitions(false)
	for _, part := range parts {
		u, err := disk.Usage(part.Mountpoint)
		if err == nil {
			// Basic sum logic or per-partition handling needed.
			// For MVP, just track totals if needed or specific mount points.
			// OmniTop MVP focuses on IO, not usage details yet.
		}
		_ = u
	}
	ioCounters, err := disk.IOCounters()
	if err == nil {
		for _, v := range ioCounters {
			stats.Disk.ReadBytes += v.ReadBytes
			stats.Disk.WriteBytes += v.WriteBytes
		}
	}

	// Network
	netCounters, err := net.IOCounters(false)
	if err == nil && len(netCounters) > 0 {
		stats.Net.BytesSent = netCounters[0].BytesSent
		stats.Net.BytesRecv = netCounters[0].BytesRecv
	}

	// GPU (if available)
	stats.GPU.HealthStatus = r.healthStatus
	stats.GPU.LastError = r.lastError
	stats.GPU.ErrorCount = r.errorCount
	stats.GPU.LastSuccessfulUpdate = r.lastSuccess
	stats.GPU.RetryAttempts = r.retryAttempts
	
	if r.hasGPU && !r.checkCircuitBreaker() {
		// Get device count
		count, err := gonvml.DeviceCount()
		if err != nil {
			r.recordError(err, "DeviceCount")
			stats.GPU.Available = false
		} else if count == 0 {
			stats.GPU.Available = false
		} else {
			// Get device handle
			dev, err := gonvml.DeviceHandleByIndex(0)
			if err != nil {
				r.recordError(err, "DeviceHandleByIndex")
				stats.GPU.Available = false
			} else {
				stats.GPU.Available = true
				r.gpuDevice = dev
				
				// Collect metrics with individual error handling
				var successCount int
				
				// GPU Name
				if name, err := dev.Name(); err == nil {
					stats.GPU.Name = name
					successCount++
				} else {
					r.recordError(err, "Name")
					stats.GPU.Name = "Unknown (Error)"
				}
				
				// GPU Utilization
				if util, _, err := dev.UtilizationRates(); err == nil {
					stats.GPU.Utilization = uint32(util)
					successCount++
				} else {
					r.recordError(err, "UtilizationRates")
				}
				
				// Memory Info
				if total, used, err := dev.MemoryInfo(); err == nil {
					stats.GPU.MemoryTotal = total
					stats.GPU.MemoryUsed = used
					if total > 0 {
						stats.GPU.MemoryUtil = uint32(float64(used) / float64(total) * 100.0)
					}
					successCount++
				} else {
					r.recordError(err, "MemoryInfo")
				}
				
				// Temperature
				if temp, err := dev.Temperature(); err == nil {
					stats.GPU.Temperature = uint32(temp)
					successCount++
				} else {
					r.recordError(err, "Temperature")
				}
				
				// Fan Speed
				if fan, err := dev.FanSpeed(); err == nil {
					stats.GPU.FanSpeed = uint32(fan)
					successCount++
				} else {
					r.recordError(err, "FanSpeed")
				}
				
				// Power Usage
				if power, err := dev.PowerUsage(); err == nil {
					stats.GPU.PowerUsage = uint32(power)
					successCount++
				} else {
					r.recordError(err, "PowerUsage")
				}
				
				// Update health based on success rate
				if successCount >= 4 { // At least 4 out of 6 metrics succeeded
					r.recordSuccess()
					stats.GPU.HealthStatus = GPUHealthHealthy
					
					// Store historical utilization if we have it
					if stats.GPU.Utilization > 0 {
						r.gpuHistoryUtil = append(r.gpuHistoryUtil, float64(stats.GPU.Utilization))
						if len(r.gpuHistoryUtil) > r.maxHistoryLen {
							r.gpuHistoryUtil = r.gpuHistoryUtil[1:]
						}
						stats.GPU.HistoricalUtil = make([]float64, len(r.gpuHistoryUtil))
						copy(stats.GPU.HistoricalUtil, r.gpuHistoryUtil)
					}
				} else if successCount > 0 {
					// Some metrics succeeded, some failed
					stats.GPU.HealthStatus = GPUHealthDegraded
				} else {
					// All metrics failed
					stats.GPU.HealthStatus = GPUHealthFailed
				}
			}
		}
	}

	// Processes
	procs, err := process.Processes()
	if err == nil {
		// Limit process list for MVP performance
		limit := 200
		count := 0
		for _, p := range procs {
			if count >= limit {
				break
			}
			name, _ := p.Name()
			user, _ := p.Username()
			cpuP, _ := p.CPUPercent()
			memP, _ := p.MemoryPercent()
			memInfo, _ := p.MemoryInfo()
			rss := uint64(0)
			if memInfo != nil {
				rss = memInfo.RSS
			}

			stats.Processes = append(stats.Processes, ProcessInfo{
				PID:        p.Pid,
				User:       user,
				Command:    name,
				CPUPercent: cpuP,
				MemPercent: float64(memP),
				Memory:     rss,
			})
			count++
		}
	}

	return stats, nil
}

func (r *RealProvider) Shutdown() {
	if r.hasGPU {
		gonvml.Shutdown()
	}
}

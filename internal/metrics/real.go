package metrics

import (
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
	hasGPU     bool
	gpuHistory []float64
}

func (r *RealProvider) Init() error {
	// Initialize NVML
	if err := gonvml.Initialize(); err != nil {
		log.Printf("NVML initialization failed (GPU metrics unavailable): %v", err)
		r.hasGPU = false
	} else {
		r.hasGPU = true
		// Initialize GPU history buffer
		r.gpuHistory = make([]float64, 0, 60)
	}
	return nil
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
	if r.hasGPU {
		count, err := gonvml.DeviceCount()
		if err == nil && count > 0 {
			dev, err := gonvml.DeviceHandleByIndex(0)
			if err == nil {
				stats.GPU.Available = true
				name, _ := dev.Name()
				stats.GPU.Name = name
				util, memUtil, _ := dev.UtilizationRates()
				stats.GPU.Utilization = uint32(util)
				stats.GPU.MemoryUtil = uint32(memUtil)
				total, used, _ := dev.MemoryInfo()
				stats.GPU.MemoryTotal = total
				stats.GPU.MemoryUsed = used
				temp, _ := dev.Temperature()
				stats.GPU.Temperature = uint32(temp)
				fan, _ := dev.FanSpeed()
				stats.GPU.FanSpeed = uint32(fan)
				power, _ := dev.PowerUsage()
				stats.GPU.PowerUsage = uint32(power)
				
				// Update GPU history
				r.gpuHistory = append(r.gpuHistory, float64(util))
				if len(r.gpuHistory) > 60 {
					r.gpuHistory = r.gpuHistory[1:]
				}
				stats.GPU.HistoricalUtil = make([]float64, len(r.gpuHistory))
				copy(stats.GPU.HistoricalUtil, r.gpuHistory)
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

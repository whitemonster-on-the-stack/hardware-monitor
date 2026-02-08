package metrics

import (
	"log"
	"time"

	"github.com/mindprince/gonvml"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

type RealProvider struct {
	hasGPU        bool
	gpuHistory    []float64
	lastDiskRead  uint64
	lastDiskWrite uint64
	lastNetSent   uint64
	lastNetRecv   uint64
	lastTime      time.Time

	// Cache for processes to calculate CPU deltas
	procCache map[int32]*process.Process
}

func (r *RealProvider) Init() error {
	// Initialize NVML
	if err := gonvml.Initialize(); err != nil {
		log.Printf("NVML initialization failed (GPU metrics unavailable): %v", err)
		r.hasGPU = false
	} else {
		r.hasGPU = true
		// Initialize GPU history buffer
		r.gpuHistory = make([]float64, 0, 100)
	}
	r.lastTime = time.Now()
	r.procCache = make(map[int32]*process.Process)
	return nil
}

func (r *RealProvider) GetStats() (*SystemStats, error) {
	now := time.Now()
	duration := now.Sub(r.lastTime).Seconds()
	if duration < 0.1 {
		duration = 0.1 // Prevent division by zero
	}

	stats := &SystemStats{
		Timestamp: now,
	}

	// Host Info (Uptime)
	if uptime, err := host.Uptime(); err == nil {
		stats.Uptime = uptime
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

	// Load Average
	if avg, err := load.Avg(); err == nil {
		stats.CPU.LoadAvg = [3]float64{avg.Load1, avg.Load5, avg.Load15}
	}

	// Memory & Swap
	vm, err := mem.VirtualMemory()
	if err == nil {
		stats.Memory.Total = vm.Total
		stats.Memory.Used = vm.Used
		stats.Memory.Free = vm.Free
		stats.Memory.UsedPercent = vm.UsedPercent
	}
	sw, err := mem.SwapMemory()
	if err == nil {
		stats.Memory.SwapTotal = sw.Total
		stats.Memory.SwapUsed = sw.Used
		stats.Memory.SwapPercent = sw.UsedPercent
	}

	// Disk
	ioCounters, err := disk.IOCounters()
	if err == nil {
		for _, v := range ioCounters {
			stats.Disk.ReadBytes += v.ReadBytes
			stats.Disk.WriteBytes += v.WriteBytes
		}
	}

	// Calculate Disk Speed
	if r.lastDiskRead > 0 {
		stats.Disk.ReadSpeed = uint64(float64(stats.Disk.ReadBytes-r.lastDiskRead) / duration)
		stats.Disk.WriteSpeed = uint64(float64(stats.Disk.WriteBytes-r.lastDiskWrite) / duration)
	}
	r.lastDiskRead = stats.Disk.ReadBytes
	r.lastDiskWrite = stats.Disk.WriteBytes


	// Network
	netCounters, err := net.IOCounters(false)
	if err == nil && len(netCounters) > 0 {
		stats.Net.BytesSent = netCounters[0].BytesSent
		stats.Net.BytesRecv = netCounters[0].BytesRecv
	}

	// Calculate Net Speed
	if r.lastNetSent > 0 {
		stats.Net.UploadSpeed = uint64(float64(stats.Net.BytesSent-r.lastNetSent) / duration)
		stats.Net.DownloadSpeed = uint64(float64(stats.Net.BytesRecv-r.lastNetRecv) / duration)
	}
	r.lastNetSent = stats.Net.BytesSent
	r.lastNetRecv = stats.Net.BytesRecv
	r.lastTime = now


	// GPU (if available)
	gpuPids := make(map[uint32]bool)
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
				if len(r.gpuHistory) > 100 {
					r.gpuHistory = r.gpuHistory[1:]
				}
				stats.GPU.HistoricalUtil = make([]float64, len(r.gpuHistory))
				copy(stats.GPU.HistoricalUtil, r.gpuHistory)

				// NOTE: mindprince/gonvml does not support process lists or power limits.
				// Leaving stats.GPU.Processes empty for RealProvider.
			}
		}
	}

	// Processes
	pids, err := process.Pids()
	if err == nil {
		// New cache for next iteration to clean up old processes
		newCache := make(map[int32]*process.Process)

		count := 0
		limit := 200 // Limit for MVP performance

		for _, pid := range pids {
			if count >= limit {
				break
			}

			// Reuse existing process struct if available
			var p *process.Process
			if existing, ok := r.procCache[pid]; ok {
				p = existing
			} else {
				p, err = process.NewProcess(pid)
				if err != nil {
					continue
				}
			}
			newCache[pid] = p

			// Gather metrics
			name, _ := p.Name()
			user, _ := p.Username()
			cpuP, _ := p.Percent(0) // This now works correctly with cached process
			memP, _ := p.MemoryPercent()
			memInfo, _ := p.MemoryInfo()
			rss := uint64(0)
			if memInfo != nil {
				rss = memInfo.RSS
			}

			// Detailed info
			ppid, _ := p.Ppid()
			threads, _ := p.NumThreads()
			nice, _ := p.Nice()
			state, _ := p.Status()

			// Handle state slice if it returns multiple characters
			stateStr := "U"
			if len(state) > 0 {
				stateStr = state[0]
			}

			// Check if using GPU
			isGpu := false
			if gpuPids[uint32(p.Pid)] {
				isGpu = true
			}

			stats.Processes = append(stats.Processes, ProcessInfo{
				PID:        p.Pid,
				User:       user,
				Command:    name,
				State:      stateStr,
				CPUPercent: cpuP,
				MemPercent: float64(memP),
				Memory:     rss,
				Threads:    threads,
				Priority:   nice,
				ParentPID:  ppid,
				IsGPUUser:  isGpu,
			})
			count++
		}

		// Update cache
		r.procCache = newCache
	}

	return stats, nil
}

func (r *RealProvider) Shutdown() {
	if r.hasGPU {
		gonvml.Shutdown()
	}
}

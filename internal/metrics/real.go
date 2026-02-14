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
	hasGPU     bool
	gpuHistory []float64
	lastNet    NetStats
	lastDisk   DiskStats
	lastTime   time.Time
	procCache  map[int32]*process.Process
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

	// Calculate duration since last check
	duration := 0.1
	if !r.lastTime.IsZero() {
		duration = now.Sub(r.lastTime).Seconds()
		if duration < 0.1 {
			duration = 0.1
		}
	}

	stats := &SystemStats{
		Timestamp: now,
	}

	// Host Info (Uptime)
	if uptime, err := host.Uptime(); err == nil {
		stats.Uptime = uptime
	}

	// CPU Usage
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

	// CPU Temps
	temps, err := host.SensorsTemperatures()
	if err == nil {
		// Simple heuristic: collect all 'core' temps
		var coreTemps []float64
		for _, t := range temps {
			if len(t.SensorKey) >= 4 && t.SensorKey[:4] == "core" {
				coreTemps = append(coreTemps, t.Temperature)
			}
		}
		stats.CPU.PerCoreTemp = coreTemps
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

	// Network
	netCounters, err := net.IOCounters(false)
	if err == nil && len(netCounters) > 0 {
		stats.Net.BytesSent = netCounters[0].BytesSent
		stats.Net.BytesRecv = netCounters[0].BytesRecv
	}

	// Calculate speeds
	if !r.lastTime.IsZero() {
		if stats.Disk.ReadBytes >= r.lastDisk.ReadBytes {
			stats.Disk.ReadSpeed = uint64(float64(stats.Disk.ReadBytes-r.lastDisk.ReadBytes) / duration)
		}
		if stats.Disk.WriteBytes >= r.lastDisk.WriteBytes {
			stats.Disk.WriteSpeed = uint64(float64(stats.Disk.WriteBytes-r.lastDisk.WriteBytes) / duration)
		}
		if stats.Net.BytesSent >= r.lastNet.BytesSent {
			stats.Net.UploadSpeed = uint64(float64(stats.Net.BytesSent-r.lastNet.BytesSent) / duration)
		}
		if stats.Net.BytesRecv >= r.lastNet.BytesRecv {
			stats.Net.DownloadSpeed = uint64(float64(stats.Net.BytesRecv-r.lastNet.BytesRecv) / duration)
		}
	}

	// Update state
	r.lastTime = now
	r.lastDisk = stats.Disk
	r.lastNet = stats.Net

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
			}
		}
	}

	// Processes
	pids, err := process.Pids()
	if err == nil {
		// New cache for next iteration to clean up old processes
		newCache := make(map[int32]*process.Process)

		count := 0
		limit := 1000 // Increased limit

		for _, pid := range pids {
			if count >= limit {
				break
			}

			// Reuse existing process struct if available
			var p *process.Process
			if existing, ok := r.procCache[pid]; ok {
				p = existing
			} else {
				var err error
				p, err = process.NewProcess(pid)
				if err != nil {
					continue
				}
			}
			newCache[pid] = p

			// Gather metrics
			name, _ := p.Name()
			user, _ := p.Username()
			cpuP, _ := p.Percent(0)
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

			// Handle state slice
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

	// Resolve GPU Process Names from System Process List
	if len(stats.GPU.Processes) > 0 {
		pidMap := make(map[int32]string)
		for _, p := range stats.Processes {
			pidMap[p.PID] = p.Command
		}
		for i := range stats.GPU.Processes {
			if name, ok := pidMap[int32(stats.GPU.Processes[i].PID)]; ok {
				stats.GPU.Processes[i].Name = name
			}
		}
	}

	return stats, nil
}

func (r *RealProvider) Shutdown() {
	if r.hasGPU {
		gonvml.Shutdown()
	}
}

# OmniTop - The Unified System Monitor

OmniTop is a production-ready Terminal User Interface (TUI) system monitoring tool that merges the best features of `nvtop` (GPU focus), `htop` (process management), and `btop` (visual density) into a single, cohesive dashboard. It is written in Go using the Bubble Tea framework.

![OmniTop Screenshot](omnitop.png)

## Features

-   **Unified Dashboard**: 3-column layout replicating a power-user workflow.
    -   **Left**: GPU History & Telemetry (NVTop style). Detailed utilization graph, VRAM/Temp/Fan/Power bars, and GPU process list.
    -   **Middle**: Process list (HTop style). Sortable, filterable, with kill/renice capabilities. Bottom stacked Memory/Swap/Net/Disk summary.
    -   **Right**: Per-core CPU bars (BTop style) with Load Averages and Uptime.
-   **GPU First**: Native NVIDIA GPU monitoring via NVML (temps, fans, clocks, power).
-   **Lich King Theme**: Midnight Black, Ice Blue, and Blood Crimson aesthetics.
-   **Alerting**: Visual and desktop notifications when thresholds are exceeded (CPU > 90%, GPU > 98%, Temp > 85°C).
-   **Educational Tooltips**: Mouse-over columns to see explanations of metrics in the footer.
-   **Mock Mode**: Run without hardware sensors for testing/demo purposes.
-   **Configurable**: Profiles saved to `profiles.json`.

## Installation

### From Source

Requirements: Go 1.24+

```bash
git clone https://github.com/google/omnitop.git
cd omnitop
go build -o omnitop cmd/omnitop/main.go
```

### Running

```bash
# Run with real sensors (requires NVIDIA GPU for full GPU metrics)
./omnitop

# Run in Mock Mode (simulated data for testing/demo)
./omnitop --mock
```

## Key Bindings

| Key | Action |
|---|---|
| `q` / `Ctrl+C` | Quit |
| `[` / `]` | Resize Left Column (GPU) |
| `{` / `}` | Resize Middle Column (Process) |
| `/` | Filter Processes (Type name/user/PID) |
| `s` | Cycle Sort Order (CPU -> MEM -> PID) |
| `k` / `F9` | Kill Selected Process (SIGTERM) |
| `Up` / `Down` | Navigate Process List |
| `Enter` / `Esc`| Confirm / Cancel Filter |

## Configuration

Configuration is stored in `profiles.json` in the current directory. It is automatically created on first run if missing.

Example `profiles.json`:
```json
{
  "theme": "lich-king",
  "column_widths": {
    "gpu": 0.3,
    "process": 0.4,
    "cpu": 0.3
  },
  "refresh_interval": 1000,
  "max_processes": 200,
  "gpu_history_length": 100,
  "show_tooltips": true,
  "alert_thresholds": {
    "cpu_usage_percent": 90,
    "cpu_temp_celsius": 85,
    "gpu_usage_percent": 98,
    "gpu_temp_celsius": 85,
    "memory_usage_percent": 95,
    "disk_usage_percent": 90
  }
}
```

## Building AppImage

To create a portable AppImage (requires `wget`):

1.  Ensure `appimagetool` is installed or let the script download it.
2.  Run the build script:
    ```bash
    ./build_appimage.sh
    ```

## Architecture

-   **cmd/omnitop**: Entry point.
-   **internal/metrics**: Data collection (Real via gopsutil/gonvml, Mock).
-   **internal/ui**: Bubble Tea models for UI (GPU, CPU, Process, Footer).
-   **internal/config**: Configuration management.

## License

Apache 2.0

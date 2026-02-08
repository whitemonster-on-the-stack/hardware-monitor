# OmniTop - The Unified System Monitor

OmniTop merges the best features of `nvtop` (GPU history), `htop` (process management), and `btop` (visual density) into a single, cohesive TUI dashboard. Inspired by the "Wrath of the Lich King" aesthetic, it provides high-density system metrics with a focus on GPU telemetry, process management, and per-core CPU visualization.

## Features

- **Unified Dashboard**: 3-column layout replicating your multi-window workflow.
  - **Left (GPU)**: Large historical utilization graph, VRAM/Fan/Power/Temp bars, and GPU process list.
  - **Middle (Process)**: Interactive process list with sorting (CPU/MEM/PID), kill/signal capabilities.
  - **Right (CPU)**: Per-core usage bars (compact 2-column view), load averages, memory/swap breakdown.
- **Visuals**: "Wrath of the Lich King" theme (Midnight Black, Ice Blue, Steel Gray, Blood Crimson).
- **Interactivity**:
  - Mouse support for selection and resizing.
  - Tooltips explaining metrics on hover (displayed in footer).
  - Configurable column widths (saved to `profiles.json`).
- **Alerting**: Visual and desktop notifications (`notify-send`) for high CPU/GPU/Memory usage or temperature.

## Installation

### From Source

Requirements: Go 1.24+

```bash
git clone https://github.com/google/omnitop.git
cd omnitop
go build -o omnitop ./cmd/omnitop
```

### Running

```bash
# Run with real sensors (requires NVIDIA GPU for GPU metrics, root for some process details)
sudo ./omnitop

# Run in Mock Mode (simulated data for testing/demo)
./omnitop --mock
```

## Key Bindings

| Key | Action |
|---|---|
| `q` / `Ctrl+C` | Quit (Saves layout) |
| `[` | Shrink Left Column (GPU) |
| `]` | Expand Left Column (GPU) |
| `{` | Shrink Middle Column (Process) |
| `}` | Expand Middle Column (Process) |
| `g` | Toggle GPU Process List / History Graph (Left Panel) |
| `c` | Sort Processes by CPU% (Middle Panel) |
| `m` | Sort Processes by Memory% (Middle Panel) |
| `p` | Sort Processes by PID (Middle Panel) |
| `k` | Kill selected process (Middle Panel) |
| `t` | Toggle Tooltips |
| `Up` / `Down` | Navigate Process List |
| `Mouse` | Click to select, hover for tooltips |

## Configuration

Configuration is saved in `profiles.json` in the working directory. It is automatically created/updated on exit.

```json
{
  "theme": "lich-king",
  "column_widths": {
    "gpu": 0.30,
    "process": 0.40,
    "cpu": 0.30
  },
  "refresh_interval": 1000,
  "show_tooltips": true
}
```

## Building AppImage

To create a portable AppImage (requires `wget`):

```bash
./build_appimage.sh
```

## Troubleshooting

- **No GPU Data**: Ensure `nvidia-smi` works and NVML library is accessible. Run with `sudo` if needed.
- **Missing Metrics**: Some process metrics require root privileges.
- **Rendering Issues**: Ensure your terminal supports true color and UTF-8 (for block characters).

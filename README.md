# OmniTop - The Unified System Monitor

OmniTop merges the best features of `nvtop`, `htop`, and `btop` into a single, cohesive TUI dashboard. Inspired by the "Wrath of the Lich King" aesthetic, it provides high-density system metrics with a focus on GPU telemetry, process management, and per-core CPU visualization.

## Features

- **Unified Dashboard**: 3-column layout replicating your multi-window workflow.
  - **Left**: GPU History & Telemetry (NVTop style).
  - **Middle**: Process list (HTop style).
  - **Right**: Per-core CPU bars & Load Averages (BTop style).
- **GPU First**: Native NVIDIA GPU monitoring via NVML (temps, fans, clocks, power).
- **Lich King Theme**: Midnight Black, Ice Blue, and Blood Crimson aesthetics.
- **Mock Mode**: Run without hardware sensors for testing/demo purposes.
- **Keyboard Resizing**: Adjust column widths dynamically.

## Installation

### From Source

Requirements: Go 1.24.3+

```bash
git clone https://github.com/google/omnitop.git
cd omnitop
go build -o omnitop ./cmd/omnitop
```

### Running

```bash
# Run with real sensors (requires NVIDIA GPU for GPU metrics)
./omnitop

# Run in Mock Mode (simulated data)
./omnitop --mock
```

## Key Bindings

| Key | Action |
|---|---|
| `q` / `Ctrl+C` | Quit |
| `[` | Shrink Left Column (GPU) |
| `]` | Expand Left Column (GPU) |
| `{` | Shrink Middle Column (Process) |
| `}` | Expand Middle Column (Process) |
| `Up` / `Down` | Navigate Process List |

## Building AppImage

To create a portable AppImage:

1. Ensure `appimagetool` is installed.
2. Run the build script:
   ```bash
   ./build_appimage.sh
   ```

## Configuration

Configuration is currently embedded in the source code. Default profiles and layout are defined in `internal/ui/root.go`, so customizing the layout requires editing that file and rebuilding the binary (any `profiles.json` file in the repo is not yet wired into the application).

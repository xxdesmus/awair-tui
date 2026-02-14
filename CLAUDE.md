# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
go build -o awair-tui .    # Build binary
go run .                    # Build and run
./awair-tui                 # Run pre-built binary
./awair-tui --help          # Show usage
```

No test suite is configured.

## Architecture

A Go terminal application that monitors Awair air sensors in real-time via their Local API. Single binary, flat package structure (all `package main`). Uses the Charm ecosystem (Bubbletea + Lipgloss + Bubbles) for the TUI.

### Module Overview

- **`main.go`** — Entry point. CLI flag parsing (`flag` stdlib), program setup, mDNS discovery goroutine launch.
- **`api.go`** — HTTP client for Awair Local API (`/air-data/latest`, `/settings/config/data`). Sensor data types, optimal range constants (temps in °F for rating), `CToF()` conversion, `RateSensorValue()` scoring logic.
- **`discovery.go`** — mDNS auto-discovery via `grandcat/zeroconf`. Browses `_http._tcp` services matching `awair*` prefix, filters for IPv4 addresses, returns a channel.
- **`config.go`** — Reads/writes `~/.awair-tui.json` for persistent device name mappings (IP → friendly name).
- **`ui.go`** — Bubbletea `Model`/`Update`/`View` implementation. Responsive device grid, sensor bars with color-coded ratings, log panel, status bar, text input prompts via `bubbles/textinput`.

### Data Flow

`main.go` creates the Bubbletea program → discovery goroutine sends `discoveredMsg` via `p.Send()` → `Update` handles `tickMsg` to dispatch parallel `pollCmd` per device → `pollResultMsg` updates device state → `View` re-renders the grid. All rendering is pure string output via Lipgloss.

## Key Patterns

- IPv6 addresses are bracketed in URLs via `formatHost()` in `api.go`
- Device grid layout divides available terminal height evenly across rows — no minimum height enforcement, to avoid pushing boxes off-screen
- Sensor bars gracefully degrade: when box width is too narrow, bars are hidden and only label + value are shown (barWidth clamped to 0)
- API returns temps in Celsius; rating always uses °F (via `DisplayValue()`), display respects `--fahrenheit` flag via `FormatValue()`
- Default temp display is Celsius; use `--fahrenheit` or `-f` to switch
- Device polling uses Bubbletea commands (goroutine per device), not sequential loops
- Config-defined names take priority over mDNS names, which take priority over device UUIDs

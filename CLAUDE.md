# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
npm run dev          # Build TypeScript and run the app
npm run build        # Build only (tsc → dist/)
npm start            # Run pre-built dist/index.js
```

No test suite is configured.

## Architecture

A Node.js terminal application that monitors Awair air sensors in real-time via their Local API. Uses ES modules with strict TypeScript (ES2022 target, Node16 module resolution).

### Module Overview

- **`src/index.ts`** — Entry point. CLI argument parsing, device management (`Map<ip, AwairDevice>`), polling loop (`Promise.allSettled` every N seconds), and keyboard shortcut wiring.
- **`src/api.ts`** — HTTP client for Awair Local API (`/air-data/latest`, `/settings/config/data`). Also contains sensor data types, optimal range constants (temps in °F), `cToF()` conversion, and `rateSensorValue()` scoring logic.
- **`src/discovery.ts`** — mDNS auto-discovery via `bonjour-service`. Browses `_http._tcp` services matching `awair-*` prefix, filters for IPv4 addresses.
- **`src/ui.ts`** — `Dashboard` class built on `blessed`. Manages screen layout (header, responsive device grid, log panel, status bar), sensor bar rendering with color-coded ratings, and text input prompts.

### Data Flow

`index.ts` orchestrates: discovery finds devices → `addDevice()` stores in Map → `pollDevice()` fetches from `api.ts` → `dashboard.updateDevices()` re-renders grid. The polling interval runs continuously via `setInterval`.

## Key Patterns

- IPv6 addresses are bracketed in URLs via `formatHost()` in `api.ts`
- `blessed` has no types for `tput` — accessed via `any` cast to suppress terminfo errors on exit
- `blessed.textbox.readInput()` handles its own focus; do NOT combine with `inputOnFocus: true` (causes deadlock)
- Device grid layout divides available terminal height evenly across rows — no minimum height enforcement, to avoid pushing boxes off-screen
- Sensor bars gracefully degrade: when box width is too narrow, bars are hidden and only label + value are shown (barWidth clamped to 0, not forced minimum)
- API returns temps in Celsius; conversion to Fahrenheit via `cToF()` happens at display time in `ui.ts`, keeping API data in native units

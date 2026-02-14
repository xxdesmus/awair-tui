# awair-tui

A terminal UI for monitoring Awair air quality sensors in real time via the [Local API](https://support.getawair.com/hc/en-us/articles/360049221014-Awair-Element-Local-API-Feature).

```
 ☁  Awair TUI   Real-time air quality monitoring

┌─ AWAIR-ELEM-1A2B3C (192.168.1.100) ─────────────────┐
│  Awair Score    86 Good                               │
│  ██████████████████████████████░░░░░░░░               │
│                                                       │
│  Temperature     74.5°F  ██████████████░░░░░░░░░░     │
│  Humidity        55.5%   █████████████████░░░░░░░     │
│  CO₂           965 ppm  █████████████████████░░░     │
│  VOC           276 ppb  ████████░░░░░░░░░░░░░░░░     │
│  PM2.5        2 µg/m³   █░░░░░░░░░░░░░░░░░░░░░░     │
│  Dew Point     51.1°F   ██████████░░░░░░░░░░░░░░     │
│  Abs Humidity  9.4g/m³   ████████████░░░░░░░░░░░     │
│  CO₂ (est)    595 ppm   ████████████████████░░░░     │
│  PM10 (est)  2 µg/m³    █░░░░░░░░░░░░░░░░░░░░░░     │
│                                                       │
│  Updated: 10:30:15 AM                                 │
└───────────────────────────────────────────────────────┘
```

## Prerequisites

- Node.js 18+
- Awair Element (or 2nd Edition) with Local API enabled via the Awair Home app

## Install

```sh
npm install
npm run build
```

## Usage

```sh
# Auto-discover Awair devices on your LAN via mDNS
node dist/index.js

# Connect to a specific device IP
node dist/index.js 192.168.1.100

# Multiple devices
node dist/index.js 192.168.1.100 192.168.1.101

# Custom polling interval (default: 10 seconds)
node dist/index.js --interval 5

# Skip mDNS discovery, only use specified IPs
node dist/index.js --no-discovery 192.168.1.100
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q` / `Esc` | Quit |
| `r` | Force refresh all devices |
| `a` | Add a device by IP address |
| `d` | Restart mDNS discovery |

## Sensors

| Sensor | Unit | Optimal Range |
|--------|------|---------------|
| Temperature | °F | 68 – 77 |
| Humidity | % RH | 40 – 50 |
| CO₂ | ppm | < 600 |
| VOC | ppb | < 300 |
| PM2.5 | µg/m³ | < 12 |
| Dew Point | °F | 50 – 65 |
| Abs Humidity | g/m³ | 4 – 12 |
| CO₂ (est) | ppm | < 600 |
| PM10 (est) | µg/m³ | < 50 |

Values are color-coded: **green** (good), **yellow** (fair), **red** (poor).

## How It Works

1. **Discovery** — Browses for `_http._tcp` mDNS services with names starting with `awair` (e.g. `awair-elem-1a2b3c`)
2. **Polling** — Fetches `GET http://<device-ip>/air-data/latest` every 10 seconds (configurable)
3. **Display** — Renders a responsive grid dashboard with score, sensor bars, and color ratings per Awair's scoring methodology. Bars gracefully hide in narrow columns.

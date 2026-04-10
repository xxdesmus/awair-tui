# Enhanced Device Cards Design Spec

**Date:** 2025-01-10  
**Status:** Ready for Implementation  
**Priority:** A → C → B

## Overview

Enhance device cards with connection status indicators, in-memory sparkline trends, and expandable detailed views to improve monitoring capabilities.

## Feature A: Connection Status Indicators

### Design
Show visual indicators for device connection state:

**States:**
- **Online** (green dot) - Device responding normally
- **Offline** (red dot) - Device not responding for >30s
- **Connecting** (yellow/amber dot) - Initial connection attempt
- **Error** (red border) - Last poll returned an error

**Visual Design:**
```
┌─ ● Device Name (192.168.1.100) ──────┐
│  ● = online (green), ○ = offline (red) │
│  Border turns red on error state       │
└────────────────────────────────────────┘
```

**Implementation:**
- Add `ConnectionStatus` enum: `online`, `offline`, `connecting`, `error`
- Track `LastSuccessfulPoll` timestamp
- Status dot appears left of device name in header
- Border color changes based on status (cyan=normal, red=error)

## Feature C: In-Memory Sparklines

### Design
Show trend graphs for each sensor using last 20 readings stored in memory.

**Visual Design:**
```
Temperature    23.6°C   ▃▄▆▇█░░░░░  [▂▄▆▇▅▃▂▄▆▇█]
```

**ASCII Chart Characters:**
- Use Unicode block elements: `▁▂▃▄▅▆▇█`
- Each character represents one reading
- Rightmost is most recent
- Scale to min/max of the visible data

**Implementation:**
- Store last 20 readings in `Device.History []SensorData`
- Add `renderSparkline(values []float64, width int) string` function
- Show sparkline to the right of progress bar (or below if narrow)
- History pruned automatically (keep only last 20)

## Feature B: Expandable Detailed View

### Design
Press Enter on a device to expand and see detailed information.

**Expanded View Content:**
```
┌─ Device Name (192.168.1.100) ────────┐
│  [Expanded]                          │
│                                      │
│  Score: 86 (Good)                    │
│                                      │
│  -- Raw Sensor Values --             │
│  Temperature: 23.6°C                 │
│  Humidity: 55.5%                     │
│  CO₂: 965 ppm                        │
│  ...                                 │
│                                      │
│  -- Device Info --                   │
│  UUID: awair-elem-1a2b3c             │
│  MAC: AA:BB:CC:DD:EE:FF              │
│  Firmware: 1.2.3                     │
│  SSID: MyNetwork                     │
│                                      │
│  -- Statistics --                    │
│  Last 20 readings:                   │
│  Temp: min 22.1, max 24.8, avg 23.4  │
│                                      │
│  [Press Enter to collapse]           │
└──────────────────────────────────────┘
```

**Implementation:**
- Add `selectedDevice string` to model
- When `selectedDevice != ""`, render expanded view instead of grid
- Show all available API fields
- Calculate min/max/avg from history
- Press Enter or Esc to return to grid view

## Technical Changes

### Data Structure Updates

```go
// ConnectionStatus represents device connection state
type ConnectionStatus int

const (
    StatusConnecting ConnectionStatus = iota
    StatusOnline
    StatusOffline
    StatusError
)

// Device updates
type Device struct {
    // ... existing fields ...
    Status           ConnectionStatus
    LastSuccessfulPoll time.Time
    History          []SensorData // Last 20 readings for sparklines
}
```

### New Functions

```go
// Update connection status based on poll results
func (dev *Device) UpdateStatus(err error)

// Render connection status indicator
func (m model) renderStatusIndicator(status ConnectionStatus) string

// Render sparkline for sensor history
func (m model) renderSparkline(values []float64, width int) string

// Render expanded device view
func (m model) renderExpandedDevice(dev *Device) string
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Toggle expanded view for selected device (Feature B) |
| `Esc` | Return to grid view from expanded mode |
| `↑/↓` or `j/k` | Navigate between devices in expanded view (optional) |

## Implementation Order

### Phase 1: Connection Status (A)
1. Add ConnectionStatus type and Device fields
2. Update status on poll results
3. Render status dot in device header
4. Change border color on error

### Phase 2: Sparklines (C)
1. Add History slice to Device
2. Store readings on successful poll
3. Create sparkline rendering function
4. Display sparklines next to sensor values

### Phase 3: Detailed View (B)
1. Add selectedDevice to model
2. Handle Enter key for toggle
3. Create expanded view renderer
4. Show raw data and statistics

## Testing Checklist

- [ ] Status indicator shows correct state (online/offline/connecting/error)
- [ ] Border color changes appropriately
- [ ] Sparklines render correctly for all sensors
- [ ] History maintains last 20 readings
- [ ] Expanded view shows all device information
- [ ] Can toggle between grid and expanded views
- [ ] Works across all color themes

## Future Considerations (Out of Scope)

- Persist history to disk for longer trends
- Prometheus/Grafana integration
- Alert thresholds and notifications
- Device grouping by room/location

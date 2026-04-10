# Visual Polish Design Spec

**Date:** 2025-01-10  
**Status:** Ready for Implementation  
**Scope:** Visual polish improvements for Awair TUI (Icons, Loading States, Animations, Empty States)

## Overview

Enhance the visual appeal of Awair TUI with icons, loading indicators, subtle update animations, and improved empty states while maintaining the clean, modern aesthetic established by the theme system.

## Design Decisions

### 1. Icons (Simple Unicode)

**Sensor Icon Mapping:**

| Sensor | Icon | Rationale |
|--------|------|-----------|
| Temperature | 🌡️ | Universal temperature symbol |
| Humidity | 💧 | Water drop represents moisture |
| CO₂ | ☁️ | Cloud evokes air/gas |
| VOC | ✨ | Sparkles suggest particles/volatiles |
| PM2.5 | 🌫️ | Fog represents particulate matter |
| Dew Point | 💨 | Wind/cloud symbol for condensation |
| Abs Humidity | 💦 | Sweat drops for absolute moisture |
| CO₂ (est) | ☁️ | Same as CO₂ |
| PM10 (est) | 🌫️ | Same as PM2.5 |

**Display Format:** `🌡️ Temperature    23.6°C`

**Implementation Notes:**
- Icons render left-aligned before the label
- Total width including icon and space: ~3 characters
- Works in all terminals without font dependencies
- Gracefully degrades if emoji rendering fails (shows as boxes)

### 2. Loading States (Dots Spinner)

**Locations:**
- Device card: Below header during connection
- Status bar: Global polling indicator
- Discovery: During mDNS search

**Spinner Style:** Dots (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`)

**Connection State Display:**
```
┌─ Device Name (192.168.1.100) ───────┐
│  ⠋ Connecting...                      │
└──────────────────────────────────────┘
```

**Status Bar Indicator:**
```
 q Quit  r Refresh  a Add  d Discover  ⠋ Polling...
```

### 3. Update Animations (Subtle)

**Trigger:** When sensor value changes from previous reading

**Animation:** Brief highlight flash (500ms)
- Text foreground brightens to `FgPrimary` color
- Returns to normal color on next tick
- Creates subtle "pulse" effect

**Implementation:** Store previous values in device state, compare on update

### 4. Enhanced Empty State

**Current:**
```
No Awair devices found

Searching via mDNS discovery...

Press a to manually add a device IP
Press d to restart discovery
Press q to quit
```

**New Design:**
```
☁️  No Awair Devices Found

🔍 Searching your network via mDNS...

📡 Or manually add a device:
   Press [a] to enter an IP address

💡 Tip: Ensure your Awair device has Local API enabled
       in the Awair Home app settings
```

**Visual Elements:**
- Header with cloud icon for branding
- Emoji bullet points for scanability
- Action items with bracketed keys: `[a]`
- Helpful tip section at bottom
- Centered layout with consistent padding

## Technical Implementation

### New Files
- `icons.go` - Icon definitions and mapping
- `spinner.go` - Spinner component wrapper

### Modified Files
- `ui.go` - Add icon rendering, spinner integration, update animations
- `api.go` - Store previous values for change detection
- `main.go` - (No changes needed)

### Data Structure Changes

```go
// Device struct additions
type Device struct {
    // ... existing fields ...
    PreviousData *SensorData  // For change detection
    IsConnecting bool         // For spinner state
}
```

### Spinner Integration

Use Bubbletea's `spinner` package from `github.com/charmbracelet/bubbles/spinner`:

```go
import "github.com/charmbracelet/bubbles/spinner"

// In model:
spinner spinner.Model

// Initialize:
s := spinner.New()
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(theme.FgSecondary)
```

### Change Detection Logic

```go
func (m *model) detectChanges(ip string, newData *SensorData) map[string]bool {
    dev := m.devices[ip]
    if dev.PreviousData == nil {
        return nil
    }
    
    changed := make(map[string]bool)
    if newData.Temp != dev.PreviousData.Temp {
        changed["temp"] = true
    }
    // ... check all sensors ...
    
    return changed
}
```

## Testing Checklist

- [ ] Icons display correctly in all six themes
- [ ] Spinner animates smoothly during connection
- [ ] Update highlight flashes briefly on value change
- [ ] Empty state displays with proper formatting
- [ ] Graceful degradation if emoji rendering fails
- [ ] No visual glitches when switching between states

## Acceptance Criteria

1. All sensor readings display with appropriate icons
2. Loading states show animated spinner (Option B - below header)
3. Value changes trigger subtle highlight animation (500ms)
4. Empty state shows branded, helpful message with action items
5. Changes work across all six color themes
6. No breaking changes to existing functionality

## Future Considerations (Out of Scope)

- Nerd Font icons (optional advanced feature)
- Different spinner styles (configurable)
- Sound notifications on alerts
- Configurable animation durations

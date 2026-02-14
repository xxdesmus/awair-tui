package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const maxResponseSize = 1 << 20 // 1 MB

// SensorData represents the JSON response from /air-data/latest.
type SensorData struct {
	Timestamp      string   `json:"timestamp"`
	Score          int      `json:"score"`
	DewPoint       *float64 `json:"dew_point"`
	Temp           float64  `json:"temp"`
	Humid          float64  `json:"humid"`
	AbsHumid       *float64 `json:"abs_humid"`
	CO2            float64  `json:"co2"`
	CO2Est         *float64 `json:"co2_est"`
	CO2EstBaseline *float64 `json:"co2_est_baseline"`
	VOC            float64  `json:"voc"`
	VOCBaseline    *float64 `json:"voc_baseline"`
	VOCH2Raw       *float64 `json:"voc_h2_raw"`
	VOCEthanolRaw  *float64 `json:"voc_ethanol_raw"`
	PM25           float64  `json:"pm25"`
	PM10Est        *float64 `json:"pm10_est"`
}

// DeviceConfig represents the JSON response from /settings/config/data.
type DeviceConfig struct {
	DeviceUUID string `json:"device_uuid"`
	WifiMAC    string `json:"wifi_mac"`
	SSID       string `json:"ssid"`
	IP         string `json:"ip"`
	Netmask    string `json:"netmask"`
	Gateway    string `json:"gateway"`
	FWVersion  string `json:"fw_version"`
	Timezone   string `json:"timezone"`
	Display    string `json:"display"`
}

// Device holds the state for a single Awair device.
type Device struct {
	IP         string
	Name       string
	Data       *SensorData
	Config     *DeviceConfig
	LastError  error
	LastUpdate time.Time
}

// SensorRange defines the optimal range for a sensor reading.
type SensorRange struct {
	Min   float64
	Max   float64
	Unit  string
	Label string
}

// OptimalRanges defines per-sensor optimal ranges (temps in °F for rating).
var OptimalRanges = map[string]SensorRange{
	"temp":      {Min: 68, Max: 77, Unit: "°F", Label: "Temperature"},
	"dew_point": {Min: 50, Max: 65, Unit: "°F", Label: "Dew Point"},
	"humid":     {Min: 40, Max: 50, Unit: "%", Label: "Humidity"},
	"abs_humid": {Min: 4, Max: 12, Unit: "g/m³", Label: "Abs Humidity"},
	"co2":       {Min: 0, Max: 600, Unit: "ppm", Label: "CO₂"},
	"co2_est":   {Min: 0, Max: 600, Unit: "ppm", Label: "CO₂ (est)"},
	"voc":       {Min: 0, Max: 300, Unit: "ppb", Label: "VOC"},
	"pm25":      {Min: 0, Max: 12, Unit: "µg/m³", Label: "PM2.5"},
	"pm10_est":  {Min: 0, Max: 50, Unit: "µg/m³", Label: "PM10 (est)"},
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

// formatHost wraps IPv6 addresses in brackets for use in URLs.
func formatHost(ip string) string {
	if strings.Contains(ip, ":") && !strings.HasPrefix(ip, "[") {
		return "[" + ip + "]"
	}
	return ip
}

// FetchAirData retrieves the latest sensor data from an Awair device.
func FetchAirData(ip string) (*SensorData, error) {
	url := fmt.Sprintf("http://%s/air-data/latest", formatHost(ip))
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	var data SensorData
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

// FetchDeviceConfig retrieves the device configuration.
func FetchDeviceConfig(ip string) (*DeviceConfig, error) {
	url := fmt.Sprintf("http://%s/settings/config/data", formatHost(ip))
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	var cfg DeviceConfig
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize)).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// CToF converts Celsius to Fahrenheit.
func CToF(c float64) float64 {
	return c*9.0/5.0 + 32.0
}

// RateSensorValue returns "good", "fair", or "poor" for a sensor value.
// For temp/dew_point, value should be in °F.
func RateSensorValue(key string, value float64) string {
	r, ok := OptimalRanges[key]
	if !ok {
		return "fair"
	}

	switch key {
	case "temp", "dew_point":
		if value >= r.Min && value <= r.Max {
			return "good"
		}
		dist := r.Min - value
		if value > r.Max {
			dist = value - r.Max
		}
		if dist <= 5 {
			return "fair"
		}
		return "poor"

	case "humid":
		if value >= r.Min && value <= r.Max {
			return "good"
		}
		dist := r.Min - value
		if value > r.Max {
			dist = value - r.Max
		}
		if dist <= 10 {
			return "fair"
		}
		return "poor"

	case "abs_humid":
		if value >= r.Min && value <= r.Max {
			return "good"
		}
		return "fair"

	default:
		// co2, co2_est, voc, pm25, pm10_est — lower is better
		if value <= r.Max {
			return "good"
		}
		if value <= r.Max*2 {
			return "fair"
		}
		return "poor"
	}
}

// FormatValue formats a sensor value for display.
func FormatValue(key string, value float64, fahrenheit bool) string {
	r := OptimalRanges[key]

	switch key {
	case "temp", "dew_point":
		if fahrenheit {
			return fmt.Sprintf("%.1f°F", CToF(value))
		}
		return fmt.Sprintf("%.1f°C", value)
	case "humid":
		return fmt.Sprintf("%.1f%s", value, r.Unit)
	case "abs_humid":
		return fmt.Sprintf("%.1f %s", value, r.Unit)
	default:
		return fmt.Sprintf("%.0f %s", math.Round(value), r.Unit)
	}
}

// DisplayValue returns the value used for rating and bar display.
// For temp/dew_point this is always °F (since ratings are defined in °F).
func DisplayValue(key string, rawCelsius float64) float64 {
	if key == "temp" || key == "dew_point" {
		return CToF(rawCelsius)
	}
	return rawCelsius
}

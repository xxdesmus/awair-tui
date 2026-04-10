package main

// SensorIcons maps sensor keys to their unicode icons.
var SensorIcons = map[string]string{
	"temp":      "🌡️",
	"humid":     "💧",
	"co2":       "☁️",
	"co2_est":   "☁️",
	"voc":       "✨",
	"pm25":      "🌫️",
	"pm10_est":  "🌫️",
	"dew_point": "💨",
	"abs_humid": "💦",
}

// GetIcon returns the icon for a sensor key, or empty string if not found.
func GetIcon(key string) string {
	if icon, ok := SensorIcons[key]; ok {
		return icon
	}
	return ""
}

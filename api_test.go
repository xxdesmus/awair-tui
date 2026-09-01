package main

import "testing"

func TestRateSensorValue(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		value  float64
		rating string
	}{
		{"temperature in range", "temp", 72, "good"},
		{"temperature near range", "temp", 80, "fair"},
		{"temperature far from range", "temp", 90, "poor"},
		{"humidity in range", "humid", 45, "good"},
		{"humidity near range", "humid", 55, "fair"},
		{"carbon dioxide in range", "co2", 600, "good"},
		{"carbon dioxide elevated", "co2", 900, "fair"},
		{"carbon dioxide poor", "co2", 1201, "poor"},
		{"unknown sensor", "unknown", 1, "fair"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RateSensorValue(tt.key, tt.value); got != tt.rating {
				t.Errorf("RateSensorValue(%q, %v) = %q, want %q", tt.key, tt.value, got, tt.rating)
			}
		})
	}
}

func TestTemperatureConversionAndFormatting(t *testing.T) {
	if got := CToF(0); got != 32 {
		t.Errorf("CToF(0) = %v, want 32", got)
	}
	if got := DisplayValue("temp", 20); got != 68 {
		t.Errorf("DisplayValue(temp, 20) = %v, want 68", got)
	}
	if got := FormatValue("temp", 20, true); got != "68.0°F" {
		t.Errorf("FormatValue(temp, 20, true) = %q, want %q", got, "68.0°F")
	}
}

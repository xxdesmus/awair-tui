package main

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestStartPollPreventsOverlappingRequests(t *testing.T) {
	m := initialModel(&Config{Devices: map[string]string{}}, nil, 10, false, false, "nord", false)
	m.addDevice("192.0.2.1", "test")

	if cmd := m.startPoll("192.0.2.1"); cmd == nil {
		t.Fatal("first poll should be scheduled")
	}
	if cmd := m.startPoll("192.0.2.1"); cmd != nil {
		t.Fatal("second poll should be skipped while the first is in flight")
	}

	updated, _ := m.Update(pollResultMsg{IP: "192.0.2.1", Data: &SensorData{}})
	m = updated.(model)
	if cmd := m.startPoll("192.0.2.1"); cmd == nil {
		t.Fatal("poll should be scheduled after its previous result is handled")
	}
}

func TestInitialModelPollsStartingDevices(t *testing.T) {
	cfg := &Config{Devices: map[string]string{
		"192.0.2.20": "second",
		"192.0.2.10": "first",
	}}
	m := initialModel(cfg, nil, 10, false, false, "nord", false)

	wantOrder := []string{"192.0.2.10", "192.0.2.20"}
	if strings.Join(m.deviceOrder, ",") != strings.Join(wantOrder, ",") {
		t.Errorf("device order = %v, want %v", m.deviceOrder, wantOrder)
	}
	for _, ip := range m.deviceOrder {
		if !m.devices[ip].PollInFlight {
			t.Errorf("starting device %s is not marked as polling", ip)
		}
	}
}

func TestDeviceHeaderTruncationPreservesANSIAndUnicode(t *testing.T) {
	label := "\x1b[31m●\x1b[0m 温度 sensor"
	truncated := ansi.Truncate(label, 5, "")

	if !utf8.ValidString(truncated) {
		t.Errorf("truncated label is not valid UTF-8: %q", truncated)
	}
	if got := lipgloss.Width(truncated); got > 5 {
		t.Errorf("truncated width = %d, want at most 5", got)
	}
	if !strings.Contains(truncated, "\x1b[0m") {
		t.Errorf("truncated label lost its ANSI reset: %q", truncated)
	}
}

package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ratingColor returns the appropriate color for a rating string using the theme.
func (m model) ratingColor(rating string) lipgloss.Color {
	return m.theme.ratingColor(rating)
}

// scoreColor returns the appropriate color for a score value using the theme.
func (m model) scoreColor(score int) lipgloss.Color {
	return m.theme.scoreColor(score)
}

func scoreLabel(score int) string {
	if score >= 80 {
		return "Good"
	}
	if score >= 60 {
		return "Fair"
	}
	return "Poor"
}

// logEntry is a timestamped log message.
type logEntry struct {
	Time    time.Time
	Message string
}

// Message types for bubbletea.
type tickMsg time.Time

type pollResultMsg struct {
	IP   string
	Data *SensorData
	Err  error
}

type configResultMsg struct {
	IP     string
	Config *DeviceConfig
}

type discoveredMsg DiscoveredDevice

// model is the bubbletea application state.
type model struct {
	devices     map[string]*Device
	deviceOrder []string // stable insertion order
	config      *Config
	logs        []logEntry
	width       int
	height      int
	fahrenheit  bool
	theme       Theme

	showPrompt  bool
	promptStep  string // "ip" or "name"
	promptInput textinput.Model
	pendingIP   string

	pollInterval time.Duration
	noDiscovery  bool
	discoveryCtx func() // cancel function for discovery

	spinner       spinner.Model
	showSpinner   bool                       // true when any device is polling
	changedFields map[string]map[string]bool // deviceIP -> sensorKey -> changed

	selectedDevice string // IP of device in expanded view, empty for grid view
}

func initialModel(cfg *Config, ips []string, interval int, noDiscovery, fahrenheit bool, themeName string) model {
	ti := textinput.New()
	ti.CharLimit = 64
	ti.Width = 40

	theme := GetTheme(themeName)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.FgSecondary)

	m := model{
		devices:       make(map[string]*Device),
		deviceOrder:   []string{},
		config:        cfg,
		logs:          []logEntry{},
		fahrenheit:    fahrenheit,
		theme:         GetTheme(themeName),
		promptInput:   ti,
		pollInterval:  time.Duration(interval) * time.Second,
		noDiscovery:   noDiscovery,
		spinner:       s,
		changedFields: make(map[string]map[string]bool),
	}

	// Load config-defined devices
	if len(cfg.Devices) > 0 {
		m.addLog(fmt.Sprintf("Loaded %d device name(s) from config", len(cfg.Devices)))
		for ip, name := range cfg.Devices {
			dev := m.addDevice(ip, name)
			m.addLog(fmt.Sprintf("Added config device: %s (%s)", dev.Name, ip))
		}
	}

	// Add CLI-specified devices
	for _, ip := range ips {
		if _, exists := cfg.Devices[ip]; !exists {
			dev := m.addDevice(ip, "")
			m.addLog(fmt.Sprintf("Added device: %s", dev.Name))
		}
	}

	return m
}

func (m *model) addLog(msg string) {
	m.logs = append(m.logs, logEntry{Time: time.Now(), Message: msg})
	if len(m.logs) > 100 {
		m.logs = m.logs[1:]
	}
}

// detectChanges compares new data with previous data to track which fields changed.
func (m *model) detectChanges(ip string, newData *SensorData) {
	dev := m.devices[ip]
	if dev.PreviousData == nil {
		return
	}

	if m.changedFields[ip] == nil {
		m.changedFields[ip] = make(map[string]bool)
	}

	old := dev.PreviousData
	if newData.Temp != old.Temp {
		m.changedFields[ip]["temp"] = true
	}
	if newData.Humid != old.Humid {
		m.changedFields[ip]["humid"] = true
	}
	if newData.CO2 != old.CO2 {
		m.changedFields[ip]["co2"] = true
	}
	if newData.VOC != old.VOC {
		m.changedFields[ip]["voc"] = true
	}
	if newData.PM25 != old.PM25 {
		m.changedFields[ip]["pm25"] = true
	}
	if newData.DewPoint != nil && old.DewPoint != nil && *newData.DewPoint != *old.DewPoint {
		m.changedFields[ip]["dew_point"] = true
	}
	if newData.AbsHumid != nil && old.AbsHumid != nil && *newData.AbsHumid != *old.AbsHumid {
		m.changedFields[ip]["abs_humid"] = true
	}
	if newData.CO2Est != nil && old.CO2Est != nil && *newData.CO2Est != *old.CO2Est {
		m.changedFields[ip]["co2_est"] = true
	}
	if newData.PM10Est != nil && old.PM10Est != nil && *newData.PM10Est != *old.PM10Est {
		m.changedFields[ip]["pm10_est"] = true
	}
}

// updateSpinnerState updates whether the spinner should be shown.
func (m *model) updateSpinnerState() {
	m.showSpinner = false
	for _, dev := range m.devices {
		if dev.IsConnecting {
			m.showSpinner = true
			return
		}
	}
}

// clearChangedField marks a field as no longer changed (after animation).
func (m *model) clearChangedField(ip, key string) {
	if m.changedFields[ip] != nil {
		delete(m.changedFields[ip], key)
	}
}

func (m *model) addDevice(ip, name string) *Device {
	// Config names take priority
	configName := m.config.Devices[ip]

	if existing, ok := m.devices[ip]; ok {
		if configName != "" {
			existing.Name = configName
		} else if name != "" && existing.Name == ip {
			existing.Name = name
		}
		return existing
	}

	displayName := ip
	if configName != "" {
		displayName = configName
	} else if name != "" {
		displayName = name
	}

	dev := &Device{
		IP:   ip,
		Name: displayName,
	}
	m.devices[ip] = dev
	m.deviceOrder = append(m.deviceOrder, ip)
	return dev
}

// orderedDevices returns devices in stable insertion order.
func (m *model) orderedDevices() []*Device {
	var devs []*Device
	for _, ip := range m.deviceOrder {
		if d, ok := m.devices[ip]; ok {
			devs = append(devs, d)
		}
	}
	return devs
}

func (m model) Init() tea.Cmd {
	// Start the first tick and poll all existing devices immediately
	cmds := []tea.Cmd{tickCmd(m.pollInterval)}
	for _, ip := range m.deviceOrder {

		cmds = append(cmds, pollCmd(ip), configCmd(ip))
	}
	return tea.Batch(cmds...)
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func pollCmd(ip string) tea.Cmd {
	return func() tea.Msg {
		data, err := FetchAirData(ip)
		return pollResultMsg{IP: ip, Data: data, Err: err}
	}
}

// discoverCmd runs a one-shot mDNS discovery and sends results as messages.
func discoverCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ch := StartDiscovery(ctx)
		// Collect all discovered devices from this query
		var found []DiscoveredDevice
		for dev := range ch {
			found = append(found, dev)
		}
		return discoveryBatchMsg(found)
	}
}

// discoveryBatchMsg carries all devices found in a single discovery pass.
type discoveryBatchMsg []DiscoveredDevice

func configCmd(ip string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := FetchDeviceConfig(ip)
		if err != nil {
			return nil
		}
		return configResultMsg{IP: ip, Config: cfg}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tickMsg:
		// Poll all devices
		var cmds []tea.Cmd
		for _, ip := range m.deviceOrder {
			if dev, ok := m.devices[ip]; ok {
				dev.IsConnecting = true
			}
			cmds = append(cmds, pollCmd(ip))
		}
		cmds = append(cmds, tickCmd(m.pollInterval))
		m.updateSpinnerState()
		return m, tea.Batch(cmds...)

	case pollResultMsg:
		if dev, ok := m.devices[msg.IP]; ok {
			dev.IsConnecting = false
			if msg.Err != nil {
				dev.LastError = msg.Err
				dev.Status = StatusError
				// Check if offline (no successful poll for 30s, but only if we've had a successful poll)
				if !dev.LastSuccessfulPoll.IsZero() && time.Since(dev.LastSuccessfulPoll) > 30*time.Second {
					dev.Status = StatusOffline
				}

			} else {
				// Detect changes for animation
				if dev.Data != nil && msg.Data != nil {
					m.detectChanges(msg.IP, msg.Data)
				}
				// Store in history for sparklines
				dev.History = append(dev.History, *msg.Data)
				if len(dev.History) > 20 {
					dev.History = dev.History[len(dev.History)-20:]
				}
				dev.PreviousData = dev.Data
				dev.Data = msg.Data
				dev.LastError = nil
				dev.LastUpdate = time.Now()
				dev.LastSuccessfulPoll = time.Now()
				dev.Status = StatusOnline
			}
		}
		m.updateSpinnerState()
		return m, nil

	case configResultMsg:
		if msg.Config == nil {
			return m, nil
		}
		if dev, ok := m.devices[msg.IP]; ok {
			dev.Config = msg.Config
			// Fall back to device_uuid if no better name exists
			if msg.Config.DeviceUUID != "" && dev.Name == dev.IP {
				dev.Name = msg.Config.DeviceUUID
			}
		}
		return m, nil

	case discoveredMsg:
		if _, exists := m.devices[msg.IP]; !exists {
			dev := m.addDevice(msg.IP, msg.Name)
			m.addLog(fmt.Sprintf("Discovered: %s at %s", dev.Name, msg.IP))
			return m, tea.Batch(pollCmd(msg.IP), configCmd(msg.IP))
		}
		return m, nil

	case discoveryBatchMsg:
		var cmds []tea.Cmd
		for _, d := range msg {
			if _, exists := m.devices[d.IP]; !exists {
				dev := m.addDevice(d.IP, d.Name)
				m.addLog(fmt.Sprintf("Discovered: %s at %s", dev.Name, d.IP))
				cmds = append(cmds, pollCmd(d.IP), configCmd(d.IP))
			}
		}
		if len(cmds) == 0 {
			m.addLog("No new devices found")
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showPrompt {
		return m.handlePromptKey(msg)
	}

	// Handle expanded view mode
	if m.selectedDevice != "" {
		switch msg.String() {
		case "esc", "enter":
			m.selectedDevice = ""
			return m, nil
		case "q", "ctrl+c":
			if m.discoveryCtx != nil {
				m.discoveryCtx()
			}
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		if m.discoveryCtx != nil {
			m.discoveryCtx()
		}
		return m, tea.Quit

	case "r":
		m.addLog("Refreshing...")
		var cmds []tea.Cmd
		for _, ip := range m.deviceOrder {

			cmds = append(cmds, pollCmd(ip))
		}
		return m, tea.Batch(cmds...)

	case "a":
		m.showPrompt = true
		m.promptStep = "ip"
		m.promptInput.Placeholder = "192.168.1.100"
		m.promptInput.SetValue("")
		m.promptInput.Focus()
		return m, textinput.Blink

	case "d":
		if m.noDiscovery {
			m.addLog("Discovery disabled (--no-discovery)")
			return m, nil
		}
		m.addLog("Restarting mDNS discovery...")
		return m, discoverCmd()

	case "enter":
		// Toggle expanded view for first device if any exist
		if len(m.deviceOrder) > 0 {
			m.selectedDevice = m.deviceOrder[0]
			return m, nil
		}
	}

	return m, nil
}

func (m model) handlePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showPrompt = false
		m.promptStep = ""
		m.pendingIP = ""
		m.promptInput.Blur()
		return m, nil

	case "enter":
		value := strings.TrimSpace(m.promptInput.Value())
		if m.promptStep == "ip" {
			if value == "" {
				m.showPrompt = false
				m.promptInput.Blur()
				return m, nil
			}
			if !isValidIP(value) {
				m.addLog(fmt.Sprintf("Invalid IP: %s", value))
				m.showPrompt = false
				m.promptInput.Blur()
				return m, nil
			}
			m.pendingIP = value
			m.promptStep = "name"
			m.promptInput.Placeholder = "(optional)"
			m.promptInput.SetValue("")
			return m, nil

		} else if m.promptStep == "name" {
			ip := m.pendingIP
			name := value
			if name != "" {
				m.config.Devices[ip] = name
				SaveConfig(m.config)
			}
			dev := m.addDevice(ip, name)
			m.addLog(fmt.Sprintf("Added device: %s (%s)", dev.Name, ip))
			m.showPrompt = false
			m.promptStep = ""
			m.pendingIP = ""
			m.promptInput.Blur()
			return m, tea.Batch(pollCmd(ip), configCmd(ip))
		}
		return m, nil
	}

	// Forward key to text input
	var cmd tea.Cmd
	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}

func isValidIP(s string) bool {
	return net.ParseIP(s) != nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	header := m.renderHeader()
	statusBar := m.renderStatusBar()
	logPanel := m.renderLogPanel()

	// Calculate available height for device grid
	headerHeight := 2
	logHeight := 6
	statusHeight := 1
	gridHeight := m.height - headerHeight - logHeight - statusHeight

	var content string
	if m.selectedDevice != "" {
		// Show expanded view for selected device
		if dev, ok := m.devices[m.selectedDevice]; ok {
			content = m.renderExpandedDevice(dev, gridHeight)
		} else {
			m.selectedDevice = ""
			content = m.renderDeviceGrid(gridHeight)
		}
	} else if len(m.devices) == 0 {
		content = m.renderEmptyState(gridHeight)
	} else {
		content = m.renderDeviceGrid(gridHeight)
	}

	// Overlay prompt if active
	if m.showPrompt {
		content = m.overlayPrompt(content, gridHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, content, logPanel, statusBar)
}

func (m model) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.AccentCyan).
		Render(" ☁  Awair TUI ")

	subtitle := lipgloss.NewStyle().
		Foreground(m.theme.FgMuted).
		Render("Real-time air quality monitoring")

	line := title + " " + subtitle

	return lipgloss.NewStyle().
		Width(m.width).
		Render(line + "\n")
}

func (m model) renderStatusBar() string {
	var content string
	if m.selectedDevice != "" {
		content = " Enter/Esc Back to grid  q Quit"
	} else {
		content = " q Quit  r Refresh  a Add device  d Discovery  Enter Details"
	}
	if m.showSpinner {
		content = m.spinner.View() + " Polling...  " + content
	}
	return lipgloss.NewStyle().
		Width(m.width).
		Background(m.theme.BgTertiary).
		Foreground(m.theme.FgSecondary).
		Render(content)
}

func (m model) renderLogPanel() string {
	border := lipgloss.NewStyle().
		Width(m.width-2).
		Height(4).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BgTertiary).
		Padding(0, 1)

	start := len(m.logs) - 4
	if start < 0 {
		start = 0
	}
	lines := make([]string, 0, 4)
	for _, entry := range m.logs[start:] {
		ts := lipgloss.NewStyle().Foreground(m.theme.FgMuted).Render(entry.Time.Format("15:04:05"))
		lines = append(lines, ts+" "+entry.Message)
	}

	content := strings.Join(lines, "\n")
	return border.Render(content)
}

func (m model) renderEmptyState(height int) string {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.FgPrimary).
		Render("No Awair Devices Found")

	searching := "Searching your network via mDNS..."

	manual := "Or manually add a device:\n" +
		"   Press [" + lipgloss.NewStyle().Bold(true).Render("a") + "] to enter an IP address"

	tip := "Tip: Ensure your Awair device has Local API enabled\n" +
		"     in the Awair Home app settings"

	content := header + "\n\n" +
		lipgloss.NewStyle().Foreground(m.theme.FgSecondary).Render(searching) + "\n\n" +
		lipgloss.NewStyle().Foreground(m.theme.FgSecondary).Render(manual) + "\n\n" +
		lipgloss.NewStyle().Foreground(m.theme.FgMuted).Render(tip)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)
}

// gridCols picks column count for the device grid.
func gridCols(n int) int {
	if n <= 2 {
		return n
	}
	if n == 4 {
		return 2 // 2x2 is better than 3+1
	}
	if n <= 6 {
		return 3
	}
	return 3
}

func (m model) renderDeviceGrid(height int) string {
	devs := m.orderedDevices()
	if len(devs) == 0 {
		return m.renderEmptyState(height)
	}

	cols := gridCols(len(devs))
	rows := (len(devs) + cols - 1) / cols
	boxWidth := m.width / cols
	boxHeight := height / rows

	var rowStrings []string

	for row := 0; row < rows; row++ {
		var colStrings []string
		for col := 0; col < cols; col++ {
			idx := row*cols + col

			w := boxWidth
			// Last column gets remaining width
			if col == cols-1 {
				w = m.width - (cols-1)*boxWidth
			}

			if idx >= len(devs) {
				// Empty cell
				colStrings = append(colStrings, lipgloss.NewStyle().Width(w).Height(boxHeight).Render(""))
				continue
			}

			dev := devs[idx]

			// Inner content width = box width - 2 (border) - 2 (padding)
			innerWidth := w - 4
			if innerWidth < 10 {
				innerWidth = 10
			}

			content := m.renderDeviceContent(dev, innerWidth)

			box := lipgloss.NewStyle().
				Width(w-2).
				MaxWidth(w).
				Height(boxHeight-2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(m.theme.AccentCyan).
				Padding(0, 1).
				Render(content)

			colStrings = append(colStrings, box)
		}
		rowStrings = append(rowStrings, lipgloss.JoinHorizontal(lipgloss.Top, colStrings...))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rowStrings...)
}

func (m model) renderDeviceContent(dev *Device, width int) string {
	// Device name header with status indicator
	statusDot := m.renderStatusIndicator(dev.Status)
	nameLabel := fmt.Sprintf("%s %s (%s)", statusDot, dev.Name, dev.IP)
	if lipgloss.Width(nameLabel) > width {
		nameLabel = nameLabel[:width]
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(m.theme.AccentCyan).Render(nameLabel)

	if dev.Data == nil {
		// Show spinner on first attempts, error only after multiple failures
		if dev.LastError != nil && len(dev.History) > 0 {
			// We've had data before, now getting errors
			errStyle := lipgloss.NewStyle().Foreground(m.theme.ColorPoor)
			errMsg := m.formatError(dev.LastError)
			return header + "\n\n" + errStyle.Render(errMsg) + "\n\nRetrying..."
		}
		// Show spinner below header
		spinnerStyle := lipgloss.NewStyle().Foreground(m.theme.FgSecondary)
		return header + "\n" + spinnerStyle.Render(m.spinner.View()+" Connecting...")
	}

	d := dev.Data
	barWidth := width - 30
	if barWidth < 0 {
		barWidth = 0
	}

	var lines []string
	lines = append(lines, header)

	// Awair Score
	sc := m.scoreColor(d.Score)
	sl := scoreLabel(d.Score)
	scoreStyle := lipgloss.NewStyle().Bold(true).Foreground(sc)
	lines = append(lines,
		fmt.Sprintf("%s    %s",
			lipgloss.NewStyle().Bold(true).Render("Awair Score"),
			scoreStyle.Render(fmt.Sprintf("%d %s", d.Score, sl))))

	if barWidth > 0 {
		lines = append(lines, m.renderGauge(d.Score, barWidth, sc))
	}
	lines = append(lines, "")

	// Sensor readings
	type sensorEntry struct {
		Key   string
		Value float64
	}

	sensors := []sensorEntry{
		{"temp", d.Temp},
		{"humid", d.Humid},
		{"co2", d.CO2},
		{"voc", d.VOC},
		{"pm25", d.PM25},
	}
	if d.DewPoint != nil {
		sensors = append(sensors, sensorEntry{"dew_point", *d.DewPoint})
	}
	if d.AbsHumid != nil {
		sensors = append(sensors, sensorEntry{"abs_humid", *d.AbsHumid})
	}
	if d.CO2Est != nil {
		sensors = append(sensors, sensorEntry{"co2_est", *d.CO2Est})
	}
	if d.PM10Est != nil {
		sensors = append(sensors, sensorEntry{"pm10_est", *d.PM10Est})
	}

	for _, s := range sensors {
		r := OptimalRanges[s.Key]
		ratingVal := DisplayValue(s.Key, s.Value)
		rating := RateSensorValue(s.Key, ratingVal)
		color := m.ratingColor(rating)
		valStr := FormatValue(s.Key, s.Value, m.fahrenheit)

		// Check if this field just changed for animation
		changed := false
		if devChanges, ok := m.changedFields[dev.IP]; ok {
			if devChanges[s.Key] {
				changed = true
				// Clear the changed flag (animation lasts one tick)
				m.clearChangedField(dev.IP, s.Key)
			}
		}

		label := visPadRight(r.Label, 14)
		valPad := visPadLeft(valStr, 12)

		valStyle := lipgloss.NewStyle().Foreground(color)
		// Apply subtle highlight if value just changed
		if changed {
			valStyle = valStyle.Background(m.theme.BgSecondary)
		}
		labelStyle := lipgloss.NewStyle().Bold(true)

		// Build sparkline from history
		sparkline := ""
		if len(dev.History) > 1 {
			historyValues := make([]float64, 0, len(dev.History))
			for _, h := range dev.History {
				switch s.Key {
				case "temp":
					historyValues = append(historyValues, h.Temp)
				case "humid":
					historyValues = append(historyValues, h.Humid)
				case "co2":
					historyValues = append(historyValues, h.CO2)
				case "voc":
					historyValues = append(historyValues, h.VOC)
				case "pm25":
					historyValues = append(historyValues, h.PM25)
				case "dew_point":
					if h.DewPoint != nil {
						historyValues = append(historyValues, *h.DewPoint)
					}
				case "abs_humid":
					if h.AbsHumid != nil {
						historyValues = append(historyValues, *h.AbsHumid)
					}
				case "co2_est":
					if h.CO2Est != nil {
						historyValues = append(historyValues, *h.CO2Est)
					}
				case "pm10_est":
					if h.PM10Est != nil {
						historyValues = append(historyValues, *h.PM10Est)
					}
				}
			}
			sparkWidth := 8
			if barWidth > 0 {
				sparkWidth = 6
			}
			sparkline = m.renderSparkline(historyValues, sparkWidth)
			if sparkline != "" {
				sparkline = " " + sparkline
			}
		}

		if barWidth > 0 {
			bar := m.renderSensorBar(s.Key, ratingVal, barWidth, color)
			lines = append(lines, fmt.Sprintf("%s %s  %s%s",
				labelStyle.Render(label),
				valStyle.Render(valPad),
				bar,
				sparkline))
		} else {
			lines = append(lines, fmt.Sprintf("%s %s%s",
				labelStyle.Render(label),
				valStyle.Render(valPad),
				sparkline))
		}
	}

	// Timestamp
	if !dev.LastUpdate.IsZero() {
		lines = append(lines, "")
		ts := lipgloss.NewStyle().Foreground(m.theme.FgMuted).
			Render("Updated: " + dev.LastUpdate.Format("15:04:05"))
		lines = append(lines, ts)
	}

	return strings.Join(lines, "\n")
}

func (m model) renderGauge(score int, width int, color lipgloss.Color) string {
	if width <= 0 {
		return ""
	}
	ratio := clamp01(float64(score) / 100.0)
	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}

	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(m.theme.BgTertiary)

	return filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", width-filled))
}

func (m model) renderSensorBar(key string, value float64, width int, color lipgloss.Color) string {
	if width <= 0 {
		return ""
	}

	var ratio float64
	switch key {
	case "temp":
		ratio = clamp01((value - 50) / 54) // 50-104°F
	case "dew_point":
		ratio = clamp01((value - 30) / 50) // 30-80°F
	case "humid":
		ratio = clamp01(value / 100)
	case "abs_humid":
		ratio = clamp01(value / 25)
	case "co2", "co2_est":
		ratio = clamp01(value / 2500)
	case "voc":
		ratio = clamp01(value / 1500)
	case "pm10_est":
		ratio = clamp01(value / 200)
	default: // pm25
		ratio = clamp01(value / 100)
	}

	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}

	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(m.theme.BgTertiary)

	return filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", width-filled))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// renderStatusIndicator returns a colored dot indicating connection status.
func (m model) renderStatusIndicator(status ConnectionStatus) string {
	switch status {
	case StatusOnline:
		return lipgloss.NewStyle().Foreground(m.theme.ColorGood).Render("●")
	case StatusOffline:
		return lipgloss.NewStyle().Foreground(m.theme.ColorPoor).Render("●")
	case StatusError:
		return lipgloss.NewStyle().Foreground(m.theme.ColorFair).Render("●")
	case StatusConnecting:
		return lipgloss.NewStyle().Foreground(m.theme.FgMuted).Render("○")
	default:
		return lipgloss.NewStyle().Foreground(m.theme.FgMuted).Render("○")
	}
}

// Sparkline characters (low to high)
var sparklineChars = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// renderSparkline creates an ASCII sparkline from a slice of values.
func (m model) renderSparkline(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}

	// Take last 'width' values
	start := 0
	if len(values) > width {
		start = len(values) - width
	}
	displayValues := values[start:]

	// Find min and max
	minVal, maxVal := displayValues[0], displayValues[0]
	for _, v := range displayValues {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Handle edge case where all values are the same
	if minVal == maxVal {
		return strings.Repeat(sparklineChars[len(sparklineChars)/2], len(displayValues))
	}

	// Build sparkline
	var result strings.Builder
	range_val := maxVal - minVal
	for _, v := range displayValues {
		normalized := (v - minVal) / range_val
		idx := int(normalized * float64(len(sparklineChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparklineChars) {
			idx = len(sparklineChars) - 1
		}
		result.WriteString(sparklineChars[idx])
	}

	return lipgloss.NewStyle().Foreground(m.theme.FgSecondary).Render(result.String())
}

// renderExpandedDevice shows detailed view for a single device.
func (m model) renderExpandedDevice(dev *Device, height int) string {
	// Header with back hint
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.AccentCyan).
		Render("Device Details: " + dev.Name)

	subheader := lipgloss.NewStyle().
		Foreground(m.theme.FgMuted).
		Render("Press [Enter] or [Esc] to return to grid view")

	var lines []string
	lines = append(lines, header)
	lines = append(lines, subheader)
	lines = append(lines, "")

	if dev.Data == nil {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.ColorFair).Render("No data available"))
		return lipgloss.NewStyle().
			Width(m.width).
			Height(height).
			Render(strings.Join(lines, "\n"))
	}

	d := dev.Data

	// Score section
	sc := m.scoreColor(d.Score)
	scoreLine := fmt.Sprintf("Awair Score: %s (%s)",
		lipgloss.NewStyle().Bold(true).Foreground(sc).Render(fmt.Sprintf("%d", d.Score)),
		scoreLabel(d.Score))
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render(scoreLine))
	lines = append(lines, "")

	// Raw sensor values section
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(m.theme.FgPrimary).Render("-- Sensor Readings --"))

	sensors := []struct {
		label string
		value string
	}{
		{"Temperature", FormatValue("temp", d.Temp, m.fahrenheit)},
		{"Humidity", FormatValue("humid", d.Humid, m.fahrenheit)},
		{"CO₂", FormatValue("co2", d.CO2, m.fahrenheit)},
		{"VOC", FormatValue("voc", d.VOC, m.fahrenheit)},
		{"PM2.5", FormatValue("pm25", d.PM25, m.fahrenheit)},
	}

	if d.DewPoint != nil {
		sensors = append(sensors, struct{ label, value string }{
			"Dew Point", FormatValue("dew_point", *d.DewPoint, m.fahrenheit)})
	}
	if d.AbsHumid != nil {
		sensors = append(sensors, struct{ label, value string }{
			"Abs Humidity", FormatValue("abs_humid", *d.AbsHumid, m.fahrenheit)})
	}
	if d.CO2Est != nil {
		sensors = append(sensors, struct{ label, value string }{
			"CO₂ (est)", FormatValue("co2_est", *d.CO2Est, m.fahrenheit)})
	}
	if d.PM10Est != nil {
		sensors = append(sensors, struct{ label, value string }{
			"PM10 (est)", FormatValue("pm10_est", *d.PM10Est, m.fahrenheit)})
	}

	for _, s := range sensors {
		lines = append(lines, fmt.Sprintf("  %s: %s", visPadRight(s.label, 15), s.value))
	}

	// Device info section
	if dev.Config != nil {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(m.theme.FgPrimary).Render("-- Device Info --"))
		cfg := dev.Config
		if cfg.DeviceUUID != "" {
			lines = append(lines, fmt.Sprintf("  UUID: %s", cfg.DeviceUUID))
		}
		if cfg.WifiMAC != "" {
			lines = append(lines, fmt.Sprintf("  MAC: %s", cfg.WifiMAC))
		}
		if cfg.FWVersion != "" {
			lines = append(lines, fmt.Sprintf("  Firmware: %s", cfg.FWVersion))
		}
		if cfg.SSID != "" {
			lines = append(lines, fmt.Sprintf("  WiFi: %s", cfg.SSID))
		}
		if cfg.IP != "" {
			lines = append(lines, fmt.Sprintf("  IP: %s", cfg.IP))
		}
	}

	// Statistics section
	if len(dev.History) > 1 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(m.theme.FgPrimary).Render("-- Recent Statistics (last "+fmt.Sprintf("%d", len(dev.History))+" readings) --"))

		// Calculate stats for each sensor
		stats := []struct {
			name   string
			key    string
			getVal func(SensorData) float64
		}{
			{"Temperature", "temp", func(s SensorData) float64 { return s.Temp }},
			{"Humidity", "humid", func(s SensorData) float64 { return s.Humid }},
			{"CO₂", "co2", func(s SensorData) float64 { return s.CO2 }},
			{"VOC", "voc", func(s SensorData) float64 { return s.VOC }},
			{"PM2.5", "pm25", func(s SensorData) float64 { return s.PM25 }},
		}

		for _, stat := range stats {
			values := make([]float64, 0, len(dev.History))
			for _, h := range dev.History {
				values = append(values, stat.getVal(h))
			}
			if len(values) > 0 {
				min, max, avg := calculateStats(values)
				lines = append(lines, fmt.Sprintf("  %s: min %.1f, max %.1f, avg %.1f",
					visPadRight(stat.name, 12), min, max, avg))
			}
		}
	}

	// Connection status
	lines = append(lines, "")
	statusText := "Unknown"
	switch dev.Status {
	case StatusOnline:
		statusText = "Online"
	case StatusOffline:
		statusText = "Offline"
	case StatusError:
		statusText = "Error"
	case StatusConnecting:
		statusText = "Connecting"
	}
	lines = append(lines, fmt.Sprintf("Status: %s", statusText))
	lines = append(lines, fmt.Sprintf("Last Update: %s", dev.LastUpdate.Format("15:04:05")))

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(m.width-4).
		Height(height-2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.AccentCyan).
		Padding(1, 2).
		Render(content)
}

// calculateStats returns min, max, and average of a slice of float64.
func calculateStats(values []float64) (min, max, avg float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	min = values[0]
	max = values[0]
	var sum float64
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}
	avg = sum / float64(len(values))
	return
}

func (m model) overlayPrompt(grid string, gridHeight int) string {
	var title string
	if m.promptStep == "ip" {
		title = "Enter device IP address"
	} else {
		title = "Friendly name (optional, Enter to skip)"
	}

	promptBox := lipgloss.NewStyle().
		Width(50).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.AccentCyan).
		Padding(0, 1).
		Render(title + "\n" + m.promptInput.View())

	return lipgloss.Place(m.width, gridHeight,
		lipgloss.Center, lipgloss.Center,
		promptBox)
}

// visPadRight pads s with spaces to visual width n using lipgloss.Width.
func visPadRight(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

// visPadLeft pads s with leading spaces to visual width n using lipgloss.Width.
func visPadLeft(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return strings.Repeat(" ", n-w) + s
}

// formatError converts ugly Go errors into user-friendly messages.
func (m model) formatError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Network connectivity errors
	if strings.Contains(errStr, "connect: host is down") {
		return "Device offline - check power and network connection"
	}
	if strings.Contains(errStr, "no route to host") {
		return "Device unreachable - check network"
	}
	if strings.Contains(errStr, "connection refused") {
		return "Connection refused - Local API may be disabled"
	}
	if strings.Contains(errStr, "timeout") {
		return "Request timed out - device may be slow to respond"
	}
	if strings.Contains(errStr, "dial tcp") && strings.Contains(errStr, "i/o timeout") {
		return "Network timeout - device not responding"
	}

	// HTTP errors
	if strings.Contains(errStr, "HTTP 404") {
		return "Device API not found - verify Local API is enabled"
	}
	if strings.Contains(errStr, "HTTP 500") {
		return "Device error - sensor may be initializing"
	}

	// Generic fallback
	return "Connection failed"
}

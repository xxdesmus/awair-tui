package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	noDiscovery := flag.Bool("no-discovery", false, "Disable mDNS auto-discovery")
	interval := flag.Int("interval", 10, "Polling interval in seconds")
	fahrenheit := flag.Bool("fahrenheit", false, "Display temperatures in Fahrenheit")
	theme := flag.String("theme", "nord", "Color theme (nord, catppuccin, tokyonight, gruvbox, dracula, classic)")

	// Short flags
	flag.IntVar(interval, "i", 10, "Polling interval in seconds (shorthand)")
	flag.BoolVar(fahrenheit, "f", false, "Display temperatures in Fahrenheit (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Awair TUI — Real-time air quality monitoring

Usage:
  awair-tui [options] [ip ...]

Options:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Available themes: nord, catppuccin, tokyonight, gruvbox, dracula, classic
Use --theme help to see all available themes

Examples:
  awair-tui                            Auto-discover devices
  awair-tui 192.168.1.100              Connect to specific device
  awair-tui -i 5 192.168.1.100        Poll every 5s
  awair-tui --fahrenheit               Show temps in °F
  awair-tui --theme tokyonight         Use Tokyo Night theme
`)
	}

	flag.Parse()
	ips := flag.Args()

	// Handle theme help request
	if *theme == "help" {
		fmt.Println("Available themes:")
		fmt.Println()
		for _, name := range ListThemes() {
			t := GetTheme(name)
			fmt.Printf("  %-12s %s\n", name, t.Description)
		}
		fmt.Println()
		fmt.Println("Use --theme <name> to select a theme")
		os.Exit(0)
	}

	cfg := LoadConfig()

	// Set up discovery context before model creation so the cancel func
	// is captured in the model's value copy passed to Bubbletea.
	var cancel context.CancelFunc
	var ctx context.Context
	if !*noDiscovery {
		ctx, cancel = context.WithCancel(context.Background())
	}

	m := initialModel(cfg, ips, *interval, *noDiscovery, *fahrenheit, *theme)
	if cancel != nil {
		m.discoveryCtx = cancel
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start mDNS discovery in a goroutine
	if !*noDiscovery {
		go func() {
			ch := StartDiscovery(ctx)
			for dev := range ch {
				p.Send(discoveredMsg(dev))
			}
		}()
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
		os.Exit(1)
	}
}

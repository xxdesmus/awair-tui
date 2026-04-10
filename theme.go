package main

import "github.com/charmbracelet/lipgloss"

// Theme defines a complete color scheme for the application.
type Theme struct {
	Name        string
	Description string

	// Backgrounds
	BgPrimary   lipgloss.Color
	BgSecondary lipgloss.Color
	BgTertiary  lipgloss.Color

	// Foregrounds
	FgPrimary   lipgloss.Color
	FgSecondary lipgloss.Color
	FgMuted     lipgloss.Color

	// Accents
	AccentCyan   lipgloss.Color
	AccentGreen  lipgloss.Color
	AccentYellow lipgloss.Color
	AccentRed    lipgloss.Color
	AccentPurple lipgloss.Color

	// Status colors
	ColorGood lipgloss.Color
	ColorFair lipgloss.Color
	ColorPoor lipgloss.Color
}

// Available themes
var themes = map[string]Theme{
	"nord":       nordTheme,
	"catppuccin": catppuccinTheme,
	"tokyonight": tokyoNightTheme,
	"gruvbox":    gruvboxTheme,
	"dracula":    draculaTheme,
	"classic":    classicTheme,
}

// GetTheme returns a theme by name, defaults to "nord" if not found.
func GetTheme(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return nordTheme
}

// ListThemes returns all available theme names.
func ListThemes() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	return names
}

// nordTheme - cool, icy blue-based palette
var nordTheme = Theme{
	Name:         "nord",
	Description:  "Cool, icy blue-based palette inspired by Nordic aesthetics",
	BgPrimary:    lipgloss.Color("#2e3440"),
	BgSecondary:  lipgloss.Color("#3b4252"),
	BgTertiary:   lipgloss.Color("#434c5e"),
	FgPrimary:    lipgloss.Color("#eceff4"),
	FgSecondary:  lipgloss.Color("#d8dee9"),
	FgMuted:      lipgloss.Color("#616e88"),
	AccentCyan:   lipgloss.Color("#88c0d0"),
	AccentGreen:  lipgloss.Color("#a3be8c"),
	AccentYellow: lipgloss.Color("#ebcb8b"),
	AccentRed:    lipgloss.Color("#bf616a"),
	AccentPurple: lipgloss.Color("#b48ead"),
	ColorGood:    lipgloss.Color("#a3be8c"),
	ColorFair:    lipgloss.Color("#ebcb8b"),
	ColorPoor:    lipgloss.Color("#bf616a"),
}

// catppuccinTheme - soft, pastel palette
var catppuccinTheme = Theme{
	Name:         "catppuccin",
	Description:  "Soft, pastel palette with gentle contrast",
	BgPrimary:    lipgloss.Color("#1e1e2e"),
	BgSecondary:  lipgloss.Color("#313244"),
	BgTertiary:   lipgloss.Color("#45475a"),
	FgPrimary:    lipgloss.Color("#cdd6f4"),
	FgSecondary:  lipgloss.Color("#a6adc8"),
	FgMuted:      lipgloss.Color("#6c7086"),
	AccentCyan:   lipgloss.Color("#89dceb"),
	AccentGreen:  lipgloss.Color("#a6e3a1"),
	AccentYellow: lipgloss.Color("#f9e2af"),
	AccentRed:    lipgloss.Color("#f38ba8"),
	AccentPurple: lipgloss.Color("#cba6f7"),
	ColorGood:    lipgloss.Color("#a6e3a1"),
	ColorFair:    lipgloss.Color("#f9e2af"),
	ColorPoor:    lipgloss.Color("#f38ba8"),
}

// tokyoNightTheme - deep purple-blue palette
var tokyoNightTheme = Theme{
	Name:         "tokyonight",
	Description:  "Deep purple-blue palette inspired by Tokyo at night",
	BgPrimary:    lipgloss.Color("#1a1b26"),
	BgSecondary:  lipgloss.Color("#24283b"),
	BgTertiary:   lipgloss.Color("#414868"),
	FgPrimary:    lipgloss.Color("#c0caf5"),
	FgSecondary:  lipgloss.Color("#a9b1d6"),
	FgMuted:      lipgloss.Color("#565f89"),
	AccentCyan:   lipgloss.Color("#7dcfff"),
	AccentGreen:  lipgloss.Color("#9ece6a"),
	AccentYellow: lipgloss.Color("#e0af68"),
	AccentRed:    lipgloss.Color("#f7768e"),
	AccentPurple: lipgloss.Color("#bb9af7"),
	ColorGood:    lipgloss.Color("#9ece6a"),
	ColorFair:    lipgloss.Color("#e0af68"),
	ColorPoor:    lipgloss.Color("#f7768e"),
}

// gruvboxTheme - warm, earthy palette
var gruvboxTheme = Theme{
	Name:         "gruvbox",
	Description:  "Warm, earthy palette with retro aesthetics",
	BgPrimary:    lipgloss.Color("#282828"),
	BgSecondary:  lipgloss.Color("#3c3836"),
	BgTertiary:   lipgloss.Color("#504945"),
	FgPrimary:    lipgloss.Color("#ebdbb2"),
	FgSecondary:  lipgloss.Color("#d5c4a1"),
	FgMuted:      lipgloss.Color("#928374"),
	AccentCyan:   lipgloss.Color("#8ec07c"),
	AccentGreen:  lipgloss.Color("#b8bb26"),
	AccentYellow: lipgloss.Color("#fabd2f"),
	AccentRed:    lipgloss.Color("#fb4934"),
	AccentPurple: lipgloss.Color("#d3869b"),
	ColorGood:    lipgloss.Color("#b8bb26"),
	ColorFair:    lipgloss.Color("#fabd2f"),
	ColorPoor:    lipgloss.Color("#fb4934"),
}

// draculaTheme - vibrant, high-contrast palette
var draculaTheme = Theme{
	Name:         "dracula",
	Description:  "Vibrant, high-contrast palette with neon accents",
	BgPrimary:    lipgloss.Color("#282a36"),
	BgSecondary:  lipgloss.Color("#44475a"),
	BgTertiary:   lipgloss.Color("#6272a4"),
	FgPrimary:    lipgloss.Color("#f8f8f2"),
	FgSecondary:  lipgloss.Color("#bfbfbf"),
	FgMuted:      lipgloss.Color("#6272a4"),
	AccentCyan:   lipgloss.Color("#8be9fd"),
	AccentGreen:  lipgloss.Color("#50fa7b"),
	AccentYellow: lipgloss.Color("#f1fa8c"),
	AccentRed:    lipgloss.Color("#ff5555"),
	AccentPurple: lipgloss.Color("#bd93f9"),
	ColorGood:    lipgloss.Color("#50fa7b"),
	ColorFair:    lipgloss.Color("#f1fa8c"),
	ColorPoor:    lipgloss.Color("#ff5555"),
}

// classicTheme - the original neon colors for those who prefer it
var classicTheme = Theme{
	Name:         "classic",
	Description:  "Original neon color scheme with bright, high-contrast colors",
	BgPrimary:    lipgloss.Color("#000000"),
	BgSecondary:  lipgloss.Color("#1a1a1a"),
	BgTertiary:   lipgloss.Color("#333333"),
	FgPrimary:    lipgloss.Color("#ffffff"),
	FgSecondary:  lipgloss.Color("#cccccc"),
	FgMuted:      lipgloss.Color("#888888"),
	AccentCyan:   lipgloss.Color("#00ffff"),
	AccentGreen:  lipgloss.Color("#00ff00"),
	AccentYellow: lipgloss.Color("#ffff00"),
	AccentRed:    lipgloss.Color("#ff0000"),
	AccentPurple: lipgloss.Color("#ff00ff"),
	ColorGood:    lipgloss.Color("#00ff00"),
	ColorFair:    lipgloss.Color("#ffff00"),
	ColorPoor:    lipgloss.Color("#ff0000"),
}

// ratingColor returns the appropriate color for a rating string.
func (t Theme) ratingColor(rating string) lipgloss.Color {
	switch rating {
	case "good":
		return t.ColorGood
	case "fair":
		return t.ColorFair
	default:
		return t.ColorPoor
	}
}

// scoreColor returns the appropriate color for a score value.
func (t Theme) scoreColor(score int) lipgloss.Color {
	if score >= 80 {
		return t.ColorGood
	}
	if score >= 60 {
		return t.ColorFair
	}
	return t.ColorPoor
}

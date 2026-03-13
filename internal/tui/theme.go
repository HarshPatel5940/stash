// Package tui provides themed terminal UI components using charmbracelet.
package tui

import (
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Color palette (Catppuccin Mocha inspired)
var (
	colorPrimary   = lipgloss.Color("205") // Pink
	colorSuccess   = lipgloss.Color("40")  // Green
	colorError     = lipgloss.Color("196") // Red
	colorWarning   = lipgloss.Color("214") // Orange
	colorInfo      = lipgloss.Color("75")  // Cyan
	colorDim       = lipgloss.Color("245") // Gray
	colorHighlight = lipgloss.Color("141") // Purple
)

// Styles for consistent output across the CLI
var (
	// TitleStyle for headers and section titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	// SuccessStyle for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	// ErrorStyle for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	// WarningStyle for warning messages
	WarningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	// InfoStyle for informational messages
	InfoStyle = lipgloss.NewStyle().
			Foreground(colorInfo)

	// DimStyle for secondary/muted text
	DimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// HighlightStyle for emphasized text
	HighlightStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true)

	// PathStyle for file paths
	PathStyle = lipgloss.NewStyle().
			Foreground(colorInfo).
			Italic(true)

	// NumberStyle for counts and numbers
	NumberStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true)
)

// Icons - minimal, text-based
var (
	IconSuccess  = SuccessStyle.Render("✓")
	IconError    = ErrorStyle.Render("✗")
	IconWarning  = WarningStyle.Render("!")
	IconInfo     = InfoStyle.Render("*")
	IconArrow    = DimStyle.Render("→")
	IconBullet   = DimStyle.Render("-")
	IconSpinner  = InfoStyle.Render("⠋")
	IconProgress = InfoStyle.Render("▪")
)

// StashTheme returns the custom theme for stash TUI forms
func StashTheme() *huh.Theme {
	return huh.ThemeCatppuccin()
}

// IsTerminal returns true if stdout is a terminal
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// IsColorDisabled returns true if NO_COLOR is set or terminal doesn't support color
func IsColorDisabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	if os.Getenv("TERM") == "dumb" {
		return true
	}
	return !IsTerminal()
}

// UseAccessibleMode returns true if forms should use accessible mode
func UseAccessibleMode() bool {
	return IsColorDisabled()
}

// ApplyTheme applies the stash theme to a form
func ApplyTheme(form *huh.Form) *huh.Form {
	if UseAccessibleMode() {
		return form.WithAccessible(true)
	}
	return form.WithTheme(StashTheme())
}

// Package tui provides interactive terminal UI components using charmbracelet/huh.
// It provides multi-select forms for restore options and file selection.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/harshpatel5940/stash/internal/metadata"
)

// RestoreOptions contains options for what to restore
type RestoreOptions struct {
	RestoreFiles         bool
	RestoreMacOSDefaults bool
	InstallHomebrew      bool
	InstallMAS           bool
	InstallVSCode        bool
	InstallNPM           bool
	RestoreShellHistory  bool
}

// AvailableOptions indicates which restore options are available
type AvailableOptions struct {
	HasBrewfile      bool
	HasMAS           bool
	HasVSCode        bool
	HasNPM           bool
	HasMacOSDefaults bool
	HasShellHistory  bool
}

// RestoreOptionsForm presents an interactive multi-select form for restore options
func RestoreOptionsForm(available AvailableOptions) (RestoreOptions, error) {
	opts := RestoreOptions{
		// All options start as false, will be set based on user selection
	}

	var selected []string

	// Build options dynamically based on what's available
	var options []huh.Option[string]

	// Files option (selected by default)
	options = append(options, huh.NewOption("Dotfiles, SSH, configs", "files").Selected(true))

	if available.HasMacOSDefaults {
		options = append(options, huh.NewOption("macOS defaults", "macos").Selected(true))
	}

	if available.HasShellHistory {
		options = append(options, huh.NewOption("Shell history", "history").Selected(true))
	}

	if available.HasBrewfile {
		options = append(options, huh.NewOption("Homebrew packages", "brew").Selected(true))
	}

	if available.HasMAS {
		options = append(options, huh.NewOption("Mac App Store apps", "mas").Selected(false))
	}

	if available.HasVSCode {
		options = append(options, huh.NewOption("VS Code extensions", "vscode").Selected(true))
	}

	if available.HasNPM {
		options = append(options, huh.NewOption("NPM globals", "npm").Selected(false))
	}

	form := ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select what to restore").
				Description("Space to toggle, Enter to confirm").
				Options(options...).
				Value(&selected),
		),
	))

	if err := form.Run(); err != nil {
		return opts, err
	}

	// Map selected options back to RestoreOptions
	for _, sel := range selected {
		switch sel {
		case "files":
			opts.RestoreFiles = true
		case "macos":
			opts.RestoreMacOSDefaults = true
		case "history":
			opts.RestoreShellHistory = true
		case "brew":
			opts.InstallHomebrew = true
		case "mas":
			opts.InstallMAS = true
		case "vscode":
			opts.InstallVSCode = true
		case "npm":
			opts.InstallNPM = true
		}
	}

	return opts, nil
}

// FilePickerForm presents an interactive multi-select form for picking files to restore
func FilePickerForm(files []metadata.FileInfo) ([]metadata.FileInfo, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// Group files by category for better organization
	type fileGroup struct {
		category string
		files    []metadata.FileInfo
	}

	groups := make(map[string][]metadata.FileInfo)
	for _, f := range files {
		category := getCategoryFromPath(f.OriginalPath)
		groups[category] = append(groups[category], f)
	}

	var selected []string
	fileMap := make(map[string]metadata.FileInfo)

	// Build options for each file
	var options []huh.Option[string]
	for _, f := range files {
		key := f.OriginalPath
		fileMap[key] = f

		label := formatFileLabel(f)
		options = append(options, huh.NewOption(label, key).Selected(true))
	}

	// If there are too many files, show a summary and confirm
	if len(files) > 50 {
		var confirm bool
		confirmForm := ApplyTheme(huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Restore all %d files?", len(files))).
					Description("Many files to restore. Yes = all, No = pick individually").
					Affirmative("Yes, restore all").
					Negative("No, let me pick").
					Value(&confirm),
			),
		))

		if err := confirmForm.Run(); err != nil {
			return nil, err
		}

		if confirm {
			return files, nil
		}
	}

	form := ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select files to restore").
				Description("Space to toggle, Enter to confirm").
				Options(options...).
				Height(20).
				Value(&selected),
		),
	))

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Map selected keys back to FileInfo
	var result []metadata.FileInfo
	for _, key := range selected {
		if f, ok := fileMap[key]; ok {
			result = append(result, f)
		}
	}

	return result, nil
}

// getCategoryFromPath extracts the category from a file path
func getCategoryFromPath(path string) string {
	// Expand ~ if present
	if strings.HasPrefix(path, "~/") {
		path = path[2:]
	}

	// Check for common categories
	switch {
	case strings.HasPrefix(path, ".ssh"):
		return "SSH"
	case strings.HasPrefix(path, ".gnupg") || strings.HasPrefix(path, ".gpg"):
		return "GPG"
	case strings.HasPrefix(path, ".aws"):
		return "AWS"
	case strings.HasPrefix(path, ".config"):
		return "Config"
	case strings.HasPrefix(path, "."):
		return "Dotfiles"
	case strings.Contains(path, ".env"):
		return "Environment"
	default:
		return "Other"
	}
}

// formatFileLabel creates a display label for a file
func formatFileLabel(f metadata.FileInfo) string {
	icon := "[F]"
	if f.IsDir {
		icon = "[D]"
	}

	// Get short path
	shortPath := f.OriginalPath
	if len(shortPath) > 45 {
		shortPath = "..." + shortPath[len(shortPath)-42:]
	}

	return fmt.Sprintf("%s %s (%s)", icon, shortPath, metadata.FormatSize(f.Size))
}

// ConfirmRestore shows a confirmation dialog before starting restore
func ConfirmRestore(fileCount int, opts RestoreOptions) (bool, error) {
	var confirm bool

	description := fmt.Sprintf("%d files", fileCount)
	if opts.InstallHomebrew {
		description += " + Homebrew"
	}
	if opts.InstallVSCode {
		description += " + VS Code"
	}

	form := ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Proceed with restore?").
				Description(description).
				Affirmative("Yes").
				Negative("Cancel").
				Value(&confirm),
		),
	))

	if err := form.Run(); err != nil {
		return false, err
	}

	return confirm, nil
}

// SelectBackup presents a selection form to choose a backup file
func SelectBackup(backups []string) (string, error) {
	if len(backups) == 0 {
		return "", fmt.Errorf("no backups available")
	}

	var selected string

	var options []huh.Option[string]
	for _, b := range backups {
		name := filepath.Base(b)
		options = append(options, huh.NewOption(name, b))
	}

	form := ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select backup").
				Options(options...).
				Value(&selected),
		),
	))

	if err := form.Run(); err != nil {
		return "", err
	}

	return selected, nil
}

// BrewPackagePickerForm presents an interactive multi-select form for picking brew packages
func BrewPackagePickerForm(items []BrewPackageItem) ([]BrewPackageItem, error) {
	if len(items) == 0 {
		return nil, nil
	}

	// Ask if user wants to pick packages or install all
	var pickIndividual bool
	confirmForm := ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Pick individual packages? (%d total)", len(items))).
				Description("Yes = pick specific packages, No = install all").
				Affirmative("Yes, let me choose").
				Negative("No, install all").
				Value(&pickIndividual),
		),
	))

	if err := confirmForm.Run(); err != nil {
		return nil, err
	}

	if !pickIndividual {
		return items, nil
	}

	var selected []string
	itemMap := make(map[string]BrewPackageItem)

	// Build options for each package
	var options []huh.Option[string]
	for i, item := range items {
		key := fmt.Sprintf("%d", i)
		itemMap[key] = item
		options = append(options, huh.NewOption(item.Label, key).Selected(true))
	}

	form := ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select packages to install").
				Description("Space to toggle, Enter to confirm, / to filter").
				Options(options...).
				Height(20).
				Value(&selected),
		),
	))

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Map selected keys back to BrewPackageItem
	var result []BrewPackageItem
	for _, key := range selected {
		if item, ok := itemMap[key]; ok {
			result = append(result, item)
		}
	}

	return result, nil
}

// BrewPackageItem represents a brew package for selection
type BrewPackageItem struct {
	Type    string // "tap", "brew", "cask", "mas"
	Name    string
	Label   string // display label
	RawLine string
}

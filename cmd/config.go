package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/harshpatel5940/stash/internal/config"
	stashTUI "github.com/harshpatel5940/stash/internal/tui"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage stash configuration",
	Long: `Manage the stash configuration file (~/.stash.yaml).

The config command provides subcommands to:
  - Initialize a default configuration file
  - Display the current configuration
  - Edit the configuration in your default editor
  - Show the config file path

Configuration controls:
  - Backup retention and cleanup
  - Dotfiles and ignored directories
  - Secret directories to backup
  - Shell history files
  - Git repository scanning settings
  - macOS defaults domains
  - Browser backup settings
  - Restore TUI preferences
  - Diff display limits`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration file",
	Long: `Creates a default configuration file at ~/.stash.yaml.

If the file already exists, it will not be overwritten unless --force is used.`,
	RunE: runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long:  `Displays the current configuration, including default values.`,
	RunE:  runConfigShow,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration with TUI or raw editor",
	Long: `Opens an interactive TUI editor for all configuration sections.

Use --raw to edit the YAML directly in your default editor.
The editor is determined by VISUAL/EDITOR, falling back to 'vim'.`,
	RunE: runConfigEdit,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Long:  `Displays the path to the configuration file.`,
	RunE:  runConfigPath,
}

var (
	configForce bool
	configRaw   bool
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configPathCmd)

	configInitCmd.Flags().BoolVarP(&configForce, "force", "f", false, "Overwrite existing configuration file")
	configEditCmd.Flags().BoolVar(&configRaw, "raw", false, "Open raw YAML in editor instead of TUI")
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".stash.yaml")

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil && !configForce {
		return fmt.Errorf("configuration file already exists at %s\nUse --force to overwrite", configPath)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Save to file
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	ui.PrintSuccess("Configuration file created at %s", configPath)
	fmt.Println()
	fmt.Println("📝 Default configuration includes:")
	fmt.Println("   • Backup retention: Keep 5 backups, auto-cleanup enabled")
	fmt.Println("   • Git scanning: Documents, Projects, Code, Dev, workspace, github")
	fmt.Println("   • Secret directories: .ssh, .gnupg, .aws")
	fmt.Println("   • Shell history: .zsh_history, .bash_history, .zhistory")
	fmt.Println("   • macOS defaults: Dock, Finder, and 20+ system preferences")
	fmt.Println("   • Browser data: skipped by default (enable in config)")
	fmt.Println("   • Restore UI: Interactive TUI enabled")
	fmt.Println()
	fmt.Println("Customize your settings:")
	fmt.Printf("  %s\n", ui.Info("stash config edit"))
	fmt.Println()
	fmt.Println("View current configuration:")
	fmt.Printf("  %s\n", ui.Info("stash config show"))

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	ui.PrintSectionHeader("⚙️", "Current Configuration")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	fmt.Println(string(data))

	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".stash.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("%s Using default configuration (no config file found)\n", ui.Info("📝"))
		fmt.Printf("   Create one with: %s\n", ui.Info("stash config init"))
	} else {
		fmt.Printf("%s Configuration loaded from: %s\n", ui.Info("📝"), configPath)
	}

	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".stash.yaml")

	// Create config file if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := config.DefaultConfig()
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("failed to create configuration file: %w", err)
		}
		fmt.Println(ui.Success("✓") + " Created new configuration file")
	}

	if configRaw || !stashTUI.IsTerminal() {
		if !configRaw && !stashTUI.IsTerminal() {
			ui.PrintInfo("Terminal is not interactive, using raw editor mode")
		}
		if err := openConfigInEditor(configPath); err != nil {
			return err
		}
	} else {
		ui.PrintInfo("Guided mode is available, but --raw is recommended for full control: stash config edit --raw")
		if err := runConfigEditTUI(configPath); err != nil {
			return err
		}
	}

	// Validate the edited config
	if _, err := config.Load(); err != nil {
		ui.PrintWarning("Configuration file may have syntax errors: %v", err)
		fmt.Println()
		fmt.Println("Fix the errors and run 'stash config show' to verify.")
		return nil
	}

	ui.PrintSuccess("Configuration saved successfully!")
	return nil
}

func openConfigInEditor(configPath string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vim"
	}

	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	fmt.Printf("Opening %s in %s...\n", configPath, editor)
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}
	return nil
}

func runConfigEditTUI(configPath string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration for editing: %w", err)
	}

	defaults := config.DefaultConfig()
	ensureConfigSections(cfg, defaults)

	backupDir := cfg.BackupDir
	encryptionKey := cfg.EncryptionKey
	keepCount := fmt.Sprintf("%d", cfg.Backup.KeepCount)
	autoCleanup := cfg.Backup.AutoCleanup
	incrementalEnabled := cfg.Incremental.Enabled
	fullBackupInterval := cfg.Incremental.FullBackupInterval
	autoMergeThreshold := fmt.Sprintf("%d", cfg.Incremental.AutoMergeThreshold)
	restoreTUI := cfg.Restore.UseTUI
	restoreThreshold := fmt.Sprintf("%d", cfg.GetRestoreFilePickerThreshold())
	gitMaxDepth := fmt.Sprintf("%d", cfg.GetGitMaxDepth())
	diffLimit := fmt.Sprintf("%d", cfg.GetDiffDisplayLimit())
	browserEnabled := cfg.IsBrowsersEnabled()
	macosEnabled := cfg.IsMacOSDefaultsEnabled()
	cloudEnabled := cfg.Cloud.Enabled
	cloudProvider := cfg.Cloud.Provider
	cloudBucket := cfg.Cloud.Bucket
	cloudRegion := cfg.Cloud.Region
	cloudEndpoint := cfg.Cloud.Endpoint
	cloudPrefix := cfg.Cloud.Prefix

	form := stashTUI.ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Backup directory").Value(&backupDir),
			huh.NewInput().Title("Encryption key path").Value(&encryptionKey),
			huh.NewInput().Title("Backups to keep").Description("Rotate old backups by count").Value(&keepCount),
			huh.NewConfirm().Title("Auto cleanup old backups").Value(&autoCleanup),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Enable incremental backups").Value(&incrementalEnabled),
			huh.NewInput().Title("Full backup interval (incremental)").Description("Examples: 7d, 14d").Value(&fullBackupInterval),
			huh.NewInput().Title("Auto-merge threshold").Description("Merge incrementals after this count").Value(&autoMergeThreshold),
			huh.NewInput().Title("Git scan max depth").Value(&gitMaxDepth),
			huh.NewInput().Title("Diff display limit").Value(&diffLimit),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Enable restore TUI").Value(&restoreTUI),
			huh.NewInput().Title("Restore file picker threshold").Description("Show file picker only below this file count").Value(&restoreThreshold),
			huh.NewConfirm().Title("Enable browser data backup").Value(&browserEnabled),
			huh.NewConfirm().Title("Enable macOS defaults backup").Value(&macosEnabled),
			huh.NewConfirm().Title("Enable cloud sync settings").Value(&cloudEnabled),
		),
		huh.NewGroup(
			huh.NewInput().Title("Cloud provider").Description("s3/minio/etc (used when cloud enabled)").Value(&cloudProvider),
			huh.NewInput().Title("Cloud bucket").Value(&cloudBucket),
			huh.NewInput().Title("Cloud region").Value(&cloudRegion),
			huh.NewInput().Title("Cloud endpoint (optional)").Value(&cloudEndpoint),
			huh.NewInput().Title("Cloud prefix (optional)").Value(&cloudPrefix),
		),
	))

	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to run config TUI: %w", err)
	}

	keepCountInt, err := parsePositiveInt(keepCount, "backup keep count")
	if err != nil {
		return err
	}
	restoreThresholdInt, err := parsePositiveInt(restoreThreshold, "restore file picker threshold")
	if err != nil {
		return err
	}
	gitMaxDepthInt, err := parsePositiveInt(gitMaxDepth, "git max depth")
	if err != nil {
		return err
	}
	diffLimitInt, err := parsePositiveInt(diffLimit, "diff display limit")
	if err != nil {
		return err
	}
	autoMergeInt, err := parsePositiveInt(autoMergeThreshold, "auto-merge threshold")
	if err != nil {
		return err
	}

	cfg.BackupDir = strings.TrimSpace(backupDir)
	cfg.EncryptionKey = strings.TrimSpace(encryptionKey)
	cfg.Backup.KeepCount = keepCountInt
	cfg.Backup.AutoCleanup = autoCleanup
	cfg.Incremental.Enabled = incrementalEnabled
	cfg.Incremental.FullBackupInterval = strings.TrimSpace(fullBackupInterval)
	cfg.Incremental.AutoMergeThreshold = autoMergeInt
	cfg.Restore.UseTUI = restoreTUI
	cfg.Restore.FilePickerThreshold = restoreThresholdInt
	cfg.Git.MaxDepth = gitMaxDepthInt
	cfg.Diff.DisplayLimit = diffLimitInt
	cfg.Browsers.Enabled = browserEnabled
	cfg.MacOSDefaults.Enabled = macosEnabled
	cfg.Cloud.Enabled = cloudEnabled
	cfg.Cloud.Provider = strings.TrimSpace(cloudProvider)
	cfg.Cloud.Bucket = strings.TrimSpace(cloudBucket)
	cfg.Cloud.Region = strings.TrimSpace(cloudRegion)
	cfg.Cloud.Endpoint = strings.TrimSpace(cloudEndpoint)
	cfg.Cloud.Prefix = strings.TrimSpace(cloudPrefix)

	searchPaths, err := editStringListWithTUI("Search paths", "Select paths to scan for .env/.pem files", cfg.SearchPaths, defaults.SearchPaths)
	if err != nil {
		return err
	}
	excludes, err := editStringListWithTUI("Exclude patterns", "Patterns to ignore during scans", cfg.Exclude, defaults.Exclude)
	if err != nil {
		return err
	}
	additionalDotfiles, err := editStringListWithTUI("Additional dotfiles", "Extra dotfiles to always include", cfg.AdditionalDotfiles, defaults.AdditionalDotfiles)
	if err != nil {
		return err
	}
	dotfileAdditional, err := editStringListWithTUI("Dotfiles.additional", "Extra dotfiles in dotfiles section", cfg.Dotfiles.Additional, defaults.Dotfiles.Additional)
	if err != nil {
		return err
	}
	dotfileIgnored, err := editStringListWithTUI("Dotfiles.ignored_dirs", "Ignored directories while scanning dotfiles", cfg.Dotfiles.IgnoredDirs, defaults.Dotfiles.IgnoredDirs)
	if err != nil {
		return err
	}
	secretDirs, err := editStringListWithTUI("Secret directories", "Sensitive directories to backup", cfg.Secrets.Directories, defaults.Secrets.Directories)
	if err != nil {
		return err
	}
	historyFiles, err := editStringListWithTUI("Shell history files", "Shell history files to include", cfg.ShellHistory.Files, defaults.ShellHistory.Files)
	if err != nil {
		return err
	}
	gitSearchDirs, err := editStringListWithTUI("Git search dirs", "Directories to scan for git repos", cfg.Git.SearchDirs, defaults.Git.SearchDirs)
	if err != nil {
		return err
	}
	gitSkipDirs, err := editStringListWithTUI("Git skip dirs", "Directory names to skip while scanning git repos", cfg.Git.SkipDirs, defaults.Git.SkipDirs)
	if err != nil {
		return err
	}
	macosDomains, err := editStringListWithTUI("macOS defaults domains", "Preference domains to backup and restore", cfg.MacOSDefaults.Domains, defaults.MacOSDefaults.Domains)
	if err != nil {
		return err
	}
	browserInclude, err := editStringListWithTUI("Browser include filter", "Leave empty for all supported browsers", cfg.Browsers.Include, defaults.Browsers.Include)
	if err != nil {
		return err
	}

	cfg.SearchPaths = searchPaths
	cfg.Exclude = excludes
	cfg.AdditionalDotfiles = additionalDotfiles
	cfg.Dotfiles.Additional = dotfileAdditional
	cfg.Dotfiles.IgnoredDirs = dotfileIgnored
	cfg.Secrets.Directories = secretDirs
	cfg.ShellHistory.Files = historyFiles
	cfg.Git.SearchDirs = gitSearchDirs
	cfg.Git.SkipDirs = gitSkipDirs
	cfg.MacOSDefaults.Domains = macosDomains
	cfg.Browsers.Include = browserInclude

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	return nil
}

func parsePositiveInt(raw, field string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", field, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", field)
	}
	return value, nil
}

func ensureConfigSections(cfg, defaults *config.Config) {
	if cfg.Backup == nil {
		cfg.Backup = defaults.Backup
	}
	if cfg.Restore == nil {
		cfg.Restore = defaults.Restore
	}
	if cfg.Git == nil {
		cfg.Git = defaults.Git
	}
	if cfg.Dotfiles == nil {
		cfg.Dotfiles = defaults.Dotfiles
	}
	if cfg.Secrets == nil {
		cfg.Secrets = defaults.Secrets
	}
	if cfg.ShellHistory == nil {
		cfg.ShellHistory = defaults.ShellHistory
	}
	if cfg.MacOSDefaults == nil {
		cfg.MacOSDefaults = defaults.MacOSDefaults
	}
	if cfg.Browsers == nil {
		cfg.Browsers = defaults.Browsers
	}
	if cfg.Diff == nil {
		cfg.Diff = defaults.Diff
	}
	if cfg.Incremental == nil {
		cfg.Incremental = defaults.Incremental
	}
	if cfg.Cloud == nil {
		cfg.Cloud = &config.CloudConfig{
			Enabled:  false,
			Provider: "s3",
			Bucket:   "",
			Region:   "us-east-1",
			Endpoint: "",
			Prefix:   "",
		}
	}
}

func editStringListWithTUI(title, description string, current, suggestions []string) ([]string, error) {
	candidates := mergeStringListCandidates(current, suggestions)
	var selected []string

	options := make([]huh.Option[string], 0, len(candidates))
	currentSet := make(map[string]struct{}, len(current))
	for _, v := range current {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			currentSet[trimmed] = struct{}{}
		}
	}
	for _, candidate := range candidates {
		_, isSelected := currentSet[candidate]
		options = append(options, huh.NewOption(candidate, candidate).Selected(isSelected))
	}

	if len(options) > 0 {
		form := stashTUI.ApplyTheme(huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title(title).
					Description(description + ". Space=toggle, Enter=continue").
					Options(options...).
					Height(12).
					Value(&selected),
			),
		))
		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("failed to edit %s: %w", title, err)
		}
	}

	resultSet := map[string]struct{}{}
	for _, item := range selected {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			resultSet[trimmed] = struct{}{}
		}
	}

	var addCustom bool
	customPrompt := stashTUI.ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Add custom entries for %s?", title)).
				Affirmative("Yes").
				Negative("No").
				Value(&addCustom),
		),
	))
	if err := customPrompt.Run(); err != nil {
		return nil, fmt.Errorf("failed to prompt custom entries for %s: %w", title, err)
	}

	for addCustom {
		var customValue string
		var addAnother bool
		customForm := stashTUI.ApplyTheme(huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("%s custom entry", title)).
					Description("Enter one value").
					Value(&customValue),
				huh.NewConfirm().
					Title("Add another custom entry?").
					Affirmative("Yes").
					Negative("No").
					Value(&addAnother),
			),
		))
		if err := customForm.Run(); err != nil {
			return nil, fmt.Errorf("failed to collect custom entries for %s: %w", title, err)
		}
		customValue = strings.TrimSpace(customValue)
		if customValue != "" {
			resultSet[customValue] = struct{}{}
		}
		addCustom = addAnother
	}

	result := make([]string, 0, len(resultSet))
	for value := range resultSet {
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func mergeStringListCandidates(primary, secondary []string) []string {
	seen := map[string]struct{}{}
	merged := make([]string, 0, len(primary)+len(secondary))
	appendUnique := func(values []string) {
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			merged = append(merged, trimmed)
		}
	}
	appendUnique(primary)
	appendUnique(secondary)
	return merged
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".stash.yaml")

	fmt.Println(configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "\n%s File does not exist. Create it with: stash config init\n", ui.Warning("⚠️"))
	}

	return nil
}

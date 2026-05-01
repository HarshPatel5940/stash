package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	Long: `Opens an interactive TUI editor for common configuration settings.

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
	fmt.Println("   • macOS defaults: Dock, Finder, and 12+ system preferences")
	fmt.Println("   • Browser data: Chrome, Firefox, Safari, Arc (enabled)")
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

	if cfg.Backup == nil {
		cfg.Backup = &config.BackupConfig{KeepCount: 5, AutoCleanup: true}
	}
	if cfg.Restore == nil {
		cfg.Restore = &config.RestoreConfig{UseTUI: true, FilePickerThreshold: 100}
	}
	if cfg.Git == nil {
		cfg.Git = &config.GitConfig{MaxDepth: 5}
	}

	backupDir := cfg.BackupDir
	encryptionKey := cfg.EncryptionKey
	keepCount := fmt.Sprintf("%d", cfg.Backup.KeepCount)
	autoCleanup := cfg.Backup.AutoCleanup
	restoreTUI := cfg.Restore.UseTUI
	restoreThreshold := fmt.Sprintf("%d", cfg.GetRestoreFilePickerThreshold())
	gitMaxDepth := fmt.Sprintf("%d", cfg.GetGitMaxDepth())
	skipBrowsers := !cfg.IsBrowsersEnabled()
	searchPaths := strings.Join(cfg.SearchPaths, ", ")

	form := stashTUI.ApplyTheme(huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Backup directory").Value(&backupDir),
			huh.NewInput().Title("Encryption key path").Value(&encryptionKey),
			huh.NewInput().Title("Backups to keep").Description("Rotate old backups by count").Value(&keepCount),
			huh.NewConfirm().Title("Auto cleanup old backups").Value(&autoCleanup),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Use interactive restore TUI").Value(&restoreTUI),
			huh.NewInput().Title("Restore file picker threshold").Description("Show file picker only below this file count").Value(&restoreThreshold),
			huh.NewInput().Title("Git scan max depth").Value(&gitMaxDepth),
			huh.NewConfirm().Title("Skip browser data in backup").Value(&skipBrowsers),
			huh.NewInput().Title("Search paths (comma-separated)").Value(&searchPaths),
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

	cfg.BackupDir = strings.TrimSpace(backupDir)
	cfg.EncryptionKey = strings.TrimSpace(encryptionKey)
	cfg.Backup.KeepCount = keepCountInt
	cfg.Backup.AutoCleanup = autoCleanup
	cfg.Restore.UseTUI = restoreTUI
	cfg.Restore.FilePickerThreshold = restoreThresholdInt
	cfg.Git.MaxDepth = gitMaxDepthInt

	if cfg.Browsers == nil {
		cfg.Browsers = &config.BrowsersConfig{Enabled: true}
	}
	cfg.Browsers.Enabled = !skipBrowsers
	cfg.SearchPaths = parseCSVValues(searchPaths)

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

func parseCSVValues(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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

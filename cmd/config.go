package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/harshpatel5940/stash/internal/config"
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
	Short: "Edit configuration in default editor",
	Long: `Opens the configuration file in your default editor.

The editor is determined by the EDITOR environment variable,
falling back to 'vim' if not set.`,
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
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configPathCmd)

	configInitCmd.Flags().BoolVarP(&configForce, "force", "f", false, "Overwrite existing configuration file")
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

	// Determine editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	// Open in editor
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	fmt.Printf("Opening %s in %s...\n", configPath, editor)
	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
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

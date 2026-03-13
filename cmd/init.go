package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var initVerbose bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize stash configuration and encryption key",
	Long: `Initialize stash by creating a default configuration file and generating
an encryption key if they don't already exist.

This will create:
  - ~/.stash.yaml (configuration file)
  - ~/.stash.key (encryption key)`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initVerbose, "verbose", "v", false, "Show detailed output")
}

func runInit(cmd *cobra.Command, args []string) error {
	ui.Verbose = initVerbose

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".stash.yaml")
	keyPath := filepath.Join(homeDir, ".stash.key")

	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	keyExists := false
	if _, err := os.Stat(keyPath); err == nil {
		keyExists = true
	}

	// Create config if needed
	if !configExists {
		cfg := config.DefaultConfig()
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
	}

	// Create key if needed
	if !keyExists {
		encryptor := crypto.NewEncryptor(keyPath)
		if err := encryptor.GenerateKey(); err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
	}

	// Output
	if configExists && keyExists {
		ui.PrintSuccess("Already initialized")
		ui.PrintDim("  Config: %s", configPath)
		ui.PrintDim("  Key: %s", keyPath)
	} else {
		ui.PrintSuccess("Initialized stash")
		ui.PrintDim("  Config: %s", configPath)
		ui.PrintDim("  Key: %s", keyPath)
		ui.PrintWarning("IMPORTANT: Backup your key to a password manager!")
	}

	// Verbose output
	if initVerbose {
		ui.PrintDivider()
		fmt.Println("Configuration includes:")
		fmt.Println("  - Backup retention: 5 backups, auto-cleanup")
		fmt.Println("  - Git scanning: ~/Documents, ~/Projects, ~/Code, etc.")
		fmt.Println("  - Secrets: .ssh, .gnupg, .aws")
		fmt.Println("  - Shell history: .zsh_history, .bash_history")
		fmt.Println("  - macOS defaults: Dock, Finder, trackpad, etc.")
		fmt.Println("  - Browser data: Chrome, Firefox, Safari")
		ui.PrintDivider()
		fmt.Println("Next steps:")
		fmt.Printf("  1. Backup key: Store %s safely\n", keyPath)
		fmt.Println("  2. Customize: stash config edit")
		fmt.Println("  3. First backup: stash backup")
	}

	return nil
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/spf13/cobra"
)

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
}

func runInit(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".stash.yaml")
	keyPath := filepath.Join(homeDir, ".stash.key")

	// Check if config already exists
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	// Check if key already exists
	keyExists := false
	if _, err := os.Stat(keyPath); err == nil {
		keyExists = true
	}

	// Create config if it doesn't exist
	if configExists {
		fmt.Printf("✓ Config already exists: %s\n", configPath)
	} else {
		cfg := config.DefaultConfig()
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
		fmt.Printf("✓ Created config: %s\n", configPath)
	}

	// Create encryption key if it doesn't exist
	if keyExists {
		fmt.Printf("✓ Encryption key already exists: %s\n", keyPath)
	} else {
		encryptor := crypto.NewEncryptor(keyPath)
		if err := encryptor.GenerateKey(); err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
		fmt.Printf("✓ Generated encryption key: %s\n", keyPath)
		fmt.Printf("\n⚠️  IMPORTANT: Keep this key safe! You'll need it to restore backups.\n")
	}

	if !configExists || !keyExists {
		fmt.Printf("\n✓ Initialization complete!\n")
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Review and customize ~/.stash.yaml if needed\n")
		fmt.Printf("  2. Run 'stash list' to preview what will be backed up\n")
		fmt.Printf("  3. Run 'stash backup' to create your first backup\n")
	} else {
		fmt.Printf("\n✓ Already initialized!\n")
	}

	return nil
}

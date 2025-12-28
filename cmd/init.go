package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	keyExists := false
	if _, err := os.Stat(keyPath); err == nil {
		keyExists = true
	}

	if configExists {
		fmt.Printf("✓ Config already exists: %s\n", configPath)
	} else {
		cfg := config.DefaultConfig()
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
		fmt.Printf("✓ Created config: %s\n", configPath)
	}

	if keyExists {
		fmt.Printf("✓ Encryption key already exists: %s\n", keyPath)
	} else {
		encryptor := crypto.NewEncryptor(keyPath)
		if err := encryptor.GenerateKey(); err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}
		fmt.Printf("✓ Generated encryption key: %s\n", keyPath)

		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("\n🔑 Key location: %s\n", keyPath)
		fmt.Println("\n📋 Action items:")
		fmt.Println("   1. Backup this key to a password manager (1Password, Bitwarden, etc.)")
		fmt.Println("   2. Store a copy on a USB drive in a secure location")
		fmt.Println("   3. Never commit this key to git or share it publicly")
		fmt.Println("\n⚠️  WITHOUT THIS KEY, YOU CANNOT RESTORE YOUR BACKUPS!")
		fmt.Println("⚠️  Losing this key means losing access to ALL encrypted backups!")
		fmt.Println()
	}

	if !configExists || !keyExists {
		fmt.Printf("✓ Initialization complete!\n")
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. SECURE YOUR KEY: Store %s in a password manager\n", keyPath)
		fmt.Printf("  2. Review and customize ~/.stash.yaml if needed\n")
		fmt.Printf("  3. Run 'stash list' to preview what will be backed up\n")
		fmt.Printf("  4. Run 'stash backup' to create your first backup\n")
		fmt.Printf("  5. Store backup files (.age) in cloud/external drive\n")
	} else {
		fmt.Printf("\n✓ Already initialized!\n")
		fmt.Printf("\n💡 Remember: Keep %s safe!\n", keyPath)
	}

	return nil
}

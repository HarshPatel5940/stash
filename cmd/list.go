package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/finder"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List what would be backed up (dry-run)",
	Long: `Shows a preview of all files and directories that would be included
in a backup without actually creating one.

This includes:
  - Dotfiles from your home directory
  - Secret directories (SSH, GPG, AWS)
  - .env files from your projects
  - .pem files
  - Package manager lists (if tools are installed)`,
	RunE: runList,
}

var listConfigPath string

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listConfigPath, "config", "c", "", "Config file path (default: ~/.stash.yaml)")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load config
	var cfg *config.Config
	var err error

	if listConfigPath != "" {
		// Load from specified path
		cfg = config.DefaultConfig()
		// TODO: implement load from specific path
		return fmt.Errorf("custom config path not yet implemented, use ~/.stash.yaml")
	} else {
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	cfg.ExpandPaths()

	fmt.Println("📋 Stash Backup Preview")
	fmt.Println("=======================")
	fmt.Println()

	// Find dotfiles
	dotfilesFinder, err := finder.NewDotfilesFinder()
	if err != nil {
		return fmt.Errorf("failed to create dotfiles finder: %w", err)
	}

	dotfiles, err := dotfilesFinder.Find(cfg.AdditionalDotfiles)
	if err != nil {
		fmt.Printf("⚠️  Warning: error finding dotfiles: %v\n", err)
	}

	if len(dotfiles) > 0 {
		fmt.Printf("📄 Dotfiles (%d found):\n", len(dotfiles))
		for _, file := range dotfiles {
			fmt.Printf("  ✓ %s\n", file)
		}
		fmt.Println()
	} else {
		fmt.Println("📄 Dotfiles: None found")
		fmt.Println()
	}

	// Find config directory
	configDir, found := dotfilesFinder.FindConfigDir()
	if found {
		fmt.Printf("⚙️  Config Directory:\n")
		fmt.Printf("  ✓ %s\n", configDir)
		fmt.Println()
	}

	// Find secret directories
	secretDirs := dotfilesFinder.FindSecretDirs()
	if len(secretDirs) > 0 {
		fmt.Printf("🔐 Secret Directories (%d found):\n", len(secretDirs))
		for name, path := range secretDirs {
			fmt.Printf("  ✓ %s: %s\n", name, path)
		}
		fmt.Println()
	} else {
		fmt.Println("🔐 Secret Directories: None found")
		fmt.Println()
	}

	// Find .env files
	envFinder := finder.NewEnvFilesFinder(cfg.SearchPaths, cfg.Exclude)

	envFiles, err := envFinder.FindEnvFiles()
	if err != nil {
		fmt.Printf("⚠️  Warning: error finding .env files: %v\n", err)
	}

	if len(envFiles) > 0 {
		fmt.Printf("🔑 Environment Files (%d found):\n", len(envFiles))
		for _, file := range envFiles {
			fmt.Printf("  ✓ %s\n", file)
		}
		fmt.Println()
	} else {
		fmt.Println("🔑 Environment Files: None found")
		fmt.Println()
	}

	// Find .pem files
	pemFiles, err := envFinder.FindPemFiles()
	if err != nil {
		fmt.Printf("⚠️  Warning: error finding .pem files: %v\n", err)
	}

	if len(pemFiles) > 0 {
		fmt.Printf("🔒 PEM Files (%d found):\n", len(pemFiles))
		for _, file := range pemFiles {
			fmt.Printf("  ✓ %s\n", file)
		}
		fmt.Println()
	} else {
		fmt.Println("🔒 PEM Files: None found")
		fmt.Println()
	}

	// Check for package managers
	fmt.Println("📦 Package Managers:")
	checkPackageManager("brew", "Homebrew")
	checkPackageManager("mas", "Mac App Store")
	checkPackageManager("code", "VS Code")
	checkPackageManager("npm", "NPM")
	fmt.Println()

	// Summary
	totalFiles := len(dotfiles) + len(envFiles) + len(pemFiles)
	totalDirs := len(secretDirs)
	if found {
		totalDirs++
	}

	fmt.Println("📊 Summary:")
	fmt.Printf("  Files: %d\n", totalFiles)
	fmt.Printf("  Directories: %d\n", totalDirs)
	fmt.Println()
	fmt.Println("💡 Run 'stash backup' to create a backup with these items")

	return nil
}

func checkPackageManager(cmd, name string) {
	path, exists := os.LookupEnv("PATH")
	if !exists {
		return
	}

	// Simple check - just see if command exists
	_, err := lookupCommand(cmd, path)
	if err == nil {
		fmt.Printf("  ✓ %s (installed)\n", name)
	} else {
		fmt.Printf("  ✗ %s (not installed)\n", name)
	}
}

func lookupCommand(cmd, pathEnv string) (string, error) {
	// Basic implementation - just use os/exec
	for _, dir := range filepath.SplitList(pathEnv) {
		path := filepath.Join(dir, cmd)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("command not found")
}

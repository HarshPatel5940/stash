package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "stash",
	Short: "A Mac backup CLI tool",
	Long: `Stash is a simple Go CLI tool that helps you quickly reset your Mac
and restore to a fresh state with all apps, configs, and secrets intact.

It backs up:
  - Dotfiles (.zshrc, .gitconfig, etc.)
  - Secret files (SSH, GPG, AWS credentials)
  - Application configs (~/.config)
  - .env and .pem files from your projects
  - Package lists (Homebrew, MAS, VS Code, npm)

All backups are encrypted using age for security.`,
	Version: "1.0.1",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

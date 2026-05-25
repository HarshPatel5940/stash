package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var initVerbose bool
var initSkipDeps bool

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
	initCmd.Flags().BoolVar(&initSkipDeps, "skip-deps", false, "Skip auto-installing Homebrew and required CLIs")
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

	if !shouldSkipDeps() {
		ui.PrintInfo("Checking Homebrew and required CLIs...")
		if err := ensureDependencies(); err != nil {
			return err
		}
	}

	// Output
	if configExists && keyExists {
		ui.PrintSuccess("Already initialized")
		ui.PrintDim("  Config: %s", configPath)
		ui.PrintDim("  Key: %s", keyPath)
		ui.PrintDim("  Edit config: stash config edit --raw")
		ui.PrintDim("  Explore commands: stash --help")
	} else {
		ui.PrintSuccess("Initialized stash")
		ui.PrintDim("  Config: %s", configPath)
		ui.PrintDim("  Key: %s", keyPath)
		ui.PrintDim("  Edit config: stash config edit --raw")
		ui.PrintDim("  Explore commands: stash --help")
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
		fmt.Println("  - Browser data: disabled by default (opt in via config)")
		ui.PrintDivider()
		fmt.Println("Next steps:")
		fmt.Printf("  1. Backup key: Store %s safely\n", keyPath)
		fmt.Println("  2. Customize: stash config edit")
		fmt.Println("  3. First backup: stash backup")
	}

	return nil
}

func shouldSkipDeps() bool {
	if initSkipDeps {
		return true
	}
	if v := strings.TrimSpace(os.Getenv("STASH_SKIP_DEPS")); v != "" && v != "0" && strings.ToLower(v) != "false" {
		return true
	}
	return false
}

func ensureDependencies() error {
	brewPath, err := findBrew()
	if err != nil {
		return err
	}
	if brewPath == "" {
		ui.PrintInfo("Homebrew not found. Installing...")
		if err := installHomebrew(); err != nil {
			return fmt.Errorf("failed to install Homebrew: %w", err)
		}
		brewPath, err = findBrew()
		if err != nil {
			return err
		}
		if brewPath == "" {
			return fmt.Errorf("Homebrew install completed but brew is still not available in PATH")
		}
	}

	var failures []string
	installBrew := func(label string, args ...string) {
		ui.PrintInfo("Installing %s...", label)
		if err := runBrew(brewPath, args...); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", label, err))
		}
	}

	if !commandExists("mas") {
		installBrew("mas", "install", "mas")
	}
	if !commandExists("dockutil") {
		installBrew("dockutil", "install", "dockutil")
	}
	if !commandExists("npm") {
		installBrew("node (npm)", "install", "node")
	}
	if !commandExists("code") {
		installBrew("Visual Studio Code", "install", "--cask", "visual-studio-code")
		if !commandExists("code") {
			failures = append(failures, "code CLI not found after installing VS Code (open VS Code and run 'Shell Command: Install code' from the Command Palette)")
		}
	}

	if len(failures) > 0 {
		for _, failure := range failures {
			ui.PrintWarning("%s", failure)
		}
		return fmt.Errorf("one or more dependencies failed to install")
	}

	ui.PrintSuccess("Dependencies ready")
	return nil
}

func findBrew() (string, error) {
	if path, err := exec.LookPath("brew"); err == nil {
		return path, nil
	}
	for _, candidate := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", nil
}

func installHomebrew() error {
	cmd := exec.Command("/bin/bash", "-c", "curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | /bin/bash")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runBrew(brewPath string, args ...string) error {
	cmd := exec.Command(brewPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

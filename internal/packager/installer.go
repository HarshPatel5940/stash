package packager

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/harshpatel5940/stash/internal/ui"
)

// Installer provides progress-wrapped package installation
type Installer struct {
	verbose bool
}

// NewInstaller creates a new package installer
func NewInstaller(verbose bool) *Installer {
	return &Installer{verbose: verbose}
}

// InstallBrewPackages installs Homebrew packages from a Brewfile with progress
func (i *Installer) InstallBrewPackages(brewfilePath string) error {
	if !commandExists("brew") {
		return fmt.Errorf("brew not installed")
	}

	// Count packages in Brewfile
	count := countBrewfilePackages(brewfilePath)
	if count == 0 {
		fmt.Println("  No packages found in Brewfile")
		return nil
	}

	fmt.Printf("  Installing %d packages from Brewfile...\n", count)

	// Create progress bar
	bar := ui.NewProgressBar(count, "Homebrew")

	// Run brew bundle and parse output
	cmd := exec.Command("brew", "bundle", "--file="+brewfilePath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start brew bundle: %w", err)
	}

	// Parse stdout for progress
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			// brew bundle outputs "Installing <package>" or "Using <package>"
			if strings.HasPrefix(line, "Installing") || strings.HasPrefix(line, "Using") ||
				strings.HasPrefix(line, "Brewing") || strings.Contains(line, "already installed") {
				bar.Add(1)
			}
			if i.verbose {
				fmt.Printf("    %s\n", line)
			}
		}
	}()

	// Capture stderr for errors
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if i.verbose {
				fmt.Printf("    [stderr] %s\n", line)
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		bar.Finish()
		return fmt.Errorf("brew bundle failed: %w", err)
	}

	bar.Finish()
	return nil
}

// InstallVSCodeExtensions installs VS Code extensions with progress
func (i *Installer) InstallVSCodeExtensions(extensionsPath string) (int, error) {
	if !commandExists("code") {
		return 0, fmt.Errorf("code command not found - install VS Code first")
	}

	// Read extensions list
	extensions, err := readNonEmptyLines(extensionsPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read extensions file: %w", err)
	}

	if len(extensions) == 0 {
		fmt.Println("  No extensions found in file")
		return 0, nil
	}

	fmt.Printf("  Installing %d VS Code extensions...\n", len(extensions))

	// Create progress bar
	bar := ui.NewProgressBar(len(extensions), "VS Code")

	installed := 0
	for _, ext := range extensions {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}

		cmd := exec.Command("code", "--install-extension", ext, "--force")
		output, err := cmd.CombinedOutput()
		if err != nil {
			if i.verbose {
				fmt.Printf("    Failed to install %s: %s\n", ext, string(output))
			}
		} else {
			installed++
		}
		bar.Add(1)
	}

	bar.Finish()
	return installed, nil
}

// InstallMASApps installs Mac App Store apps with progress
func (i *Installer) InstallMASApps(masFilePath string) (int, error) {
	if !commandExists("mas") {
		return 0, fmt.Errorf("mas not installed - install with: brew install mas")
	}

	// Read MAS apps list
	lines, err := readNonEmptyLines(masFilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read MAS file: %w", err)
	}

	if len(lines) == 0 {
		fmt.Println("  No apps found in file")
		return 0, nil
	}

	fmt.Printf("  Installing %d Mac App Store apps...\n", len(lines))

	// Create progress bar
	bar := ui.NewProgressBar(len(lines), "App Store")

	installed := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse app ID (first field)
		parts := strings.Fields(line)
		if len(parts) < 1 {
			bar.Add(1)
			continue
		}
		appID := parts[0]

		cmd := exec.Command("mas", "install", appID)
		output, err := cmd.CombinedOutput()
		if err != nil {
			if i.verbose {
				fmt.Printf("    Failed to install app %s: %s\n", appID, string(output))
			}
		} else {
			installed++
		}
		bar.Add(1)
	}

	bar.Finish()
	return installed, nil
}

// InstallNPMPackages shows the NPM package list (manual install recommended)
func (i *Installer) InstallNPMPackages(npmFilePath string) error {
	if !commandExists("npm") {
		return fmt.Errorf("npm not found - install Node.js first")
	}

	// Read NPM packages list
	lines, err := readNonEmptyLines(npmFilePath)
	if err != nil {
		return fmt.Errorf("failed to read NPM file: %w", err)
	}

	if len(lines) == 0 {
		fmt.Println("  No packages found in file")
		return nil
	}

	fmt.Printf("  Found %d NPM global packages\n", len(lines))
	fmt.Println("  NPM global packages list saved at:", npmFilePath)
	fmt.Println("  💡 Review and install manually with: npm install -g <package>")

	return nil
}

// countBrewfilePackages counts packages in a Brewfile
func countBrewfilePackages(brewfilePath string) int {
	lines, err := readNonEmptyLines(brewfilePath)
	if err != nil {
		return 0
	}

	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Count lines that start with brew, cask, tap, or mas
		if strings.HasPrefix(line, "brew ") ||
			strings.HasPrefix(line, "cask ") ||
			strings.HasPrefix(line, "tap ") ||
			strings.HasPrefix(line, "mas ") {
			count++
		}
	}
	return count
}

// readNonEmptyLines reads a file and returns non-empty, non-comment lines
func readNonEmptyLines(filePath string) ([]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

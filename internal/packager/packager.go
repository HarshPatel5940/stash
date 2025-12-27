package packager

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Packager handles collecting package lists from various package managers
type Packager struct {
	outputDir string
}

// NewPackager creates a new packager
func NewPackager(outputDir string) *Packager {
	return &Packager{
		outputDir: outputDir,
	}
}

// CollectAll collects all package lists
func (p *Packager) CollectAll() (map[string]int, error) {
	counts := make(map[string]int)

	// Homebrew
	if err := p.CollectHomebrew(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "Brewfile"))
		counts["homebrew"] = count
	}

	// MAS
	if err := p.CollectMAS(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "mas-apps.txt"))
		counts["mas"] = count
	}

	// VS Code
	if err := p.CollectVSCode(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "vscode-extensions.txt"))
		counts["vscode"] = count
	}

	// NPM
	if err := p.CollectNPM(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "npm-global.txt"))
		counts["npm"] = count
	}

	return counts, nil
}

// CollectHomebrew dumps Brewfile
func (p *Packager) CollectHomebrew() error {
	if !commandExists("brew") {
		return fmt.Errorf("brew not installed")
	}

	brewfile := filepath.Join(p.outputDir, "Brewfile")

	// Use brew bundle dump to create Brewfile
	cmd := exec.Command("brew", "bundle", "dump", "--file="+brewfile, "--force")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle dump failed: %v, %s", err, stderr.String())
	}

	return nil
}

// CollectMAS collects Mac App Store apps
func (p *Packager) CollectMAS() error {
	if !commandExists("mas") {
		return fmt.Errorf("mas not installed")
	}

	output, err := exec.Command("mas", "list").Output()
	if err != nil {
		return fmt.Errorf("mas list failed: %v", err)
	}

	masFile := filepath.Join(p.outputDir, "mas-apps.txt")
	return os.WriteFile(masFile, output, 0644)
}

// CollectVSCode collects VS Code extensions
func (p *Packager) CollectVSCode() error {
	if !commandExists("code") {
		return fmt.Errorf("code not installed")
	}

	output, err := exec.Command("code", "--list-extensions").Output()
	if err != nil {
		return fmt.Errorf("code --list-extensions failed: %v", err)
	}

	vscodeFile := filepath.Join(p.outputDir, "vscode-extensions.txt")
	return os.WriteFile(vscodeFile, output, 0644)
}

// CollectNPM collects global npm packages
func (p *Packager) CollectNPM() error {
	if !commandExists("npm") {
		return fmt.Errorf("npm not installed")
	}

	output, err := exec.Command("npm", "list", "-g", "--depth=0").Output()
	if err != nil {
		// npm list returns exit code 1 even on success sometimes
		// So we check if we got output
		if len(output) == 0 {
			return fmt.Errorf("npm list failed: %v", err)
		}
	}

	npmFile := filepath.Join(p.outputDir, "npm-global.txt")
	return os.WriteFile(npmFile, output, 0644)
}

// commandExists checks if a command is available
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// countLines counts non-empty lines in a file
func (p *Packager) countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	lines := strings.Split(string(data), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

package packager

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type AppInfo struct {
	Name       string
	Path       string
	InHomebrew bool
}

type Packager struct {
	outputDir string
}

func NewPackager(outputDir string) *Packager {
	return &Packager{
		outputDir: outputDir,
	}
}

func (p *Packager) CollectAll() (map[string]int, error) {
	counts := make(map[string]int)

	if err := p.CollectHomebrew(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "Brewfile"))
		counts["homebrew"] = count
	}

	if err := p.CollectMAS(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "mas-apps.txt"))
		counts["mas"] = count
	}

	if err := p.CollectVSCode(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "vscode-extensions.txt"))
		counts["vscode"] = count
	}

	if err := p.CollectNPM(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "npm-global.txt"))
		counts["npm"] = count
	}

	if err := p.CollectPip(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "pip-requirements.txt"))
		counts["pip"] = count
	}

	if err := p.CollectCargo(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "cargo-packages.txt"))
		counts["cargo"] = count
	}

	if err := p.CollectPnpm(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "pnpm-global.txt"))
		counts["pnpm"] = count
	}

	if err := p.CollectGem(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "gem-packages.txt"))
		counts["gem"] = count
	}

	if err := p.CollectComposer(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "composer-global.txt"))
		counts["composer"] = count
	}

	if err := p.DetectNonBrewApps(); err == nil {
		count := p.countLines(filepath.Join(p.outputDir, "non-brew-apps.txt"))
		counts["non-brew-apps"] = count
	}

	return counts, nil
}

func (p *Packager) CollectHomebrew() error {
	if !commandExists("brew") {
		return fmt.Errorf("brew not installed")
	}

	brewfile := filepath.Join(p.outputDir, "Brewfile")

	cmd := exec.Command("brew", "bundle", "dump", "--file="+brewfile, "--force")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle dump failed: %v, %s", err, stderr.String())
	}

	return nil
}

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

func (p *Packager) CollectNPM() error {
	if !commandExists("npm") {
		return fmt.Errorf("npm not installed")
	}

	output, err := exec.Command("npm", "list", "-g", "--depth=0").Output()
	if err != nil {

		if len(output) == 0 {
			return fmt.Errorf("npm list failed: %v", err)
		}
	}

	npmFile := filepath.Join(p.outputDir, "npm-global.txt")
	return os.WriteFile(npmFile, output, 0644)
}

func (p *Packager) CollectPip() error {
	// Try pip3 first (more common on macOS), then pip
	var cmd *exec.Cmd
	if commandExists("pip3") {
		cmd = exec.Command("pip3", "freeze")
	} else if commandExists("pip") {
		cmd = exec.Command("pip", "freeze")
	} else {
		return fmt.Errorf("pip/pip3 not installed")
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("pip freeze failed: %v", err)
	}

	pipFile := filepath.Join(p.outputDir, "pip-requirements.txt")
	return os.WriteFile(pipFile, output, 0644)
}

func (p *Packager) CollectCargo() error {
	if !commandExists("cargo") {
		return fmt.Errorf("cargo not installed")
	}

	output, err := exec.Command("cargo", "install", "--list").Output()
	if err != nil {
		return fmt.Errorf("cargo install --list failed: %v", err)
	}

	cargoFile := filepath.Join(p.outputDir, "cargo-packages.txt")
	return os.WriteFile(cargoFile, output, 0644)
}

func (p *Packager) CollectPnpm() error {
	if !commandExists("pnpm") {
		return fmt.Errorf("pnpm not installed")
	}

	output, err := exec.Command("pnpm", "list", "-g", "--depth=0").Output()
	if err != nil {
		// pnpm might return non-zero exit code even on success
		if len(output) == 0 {
			return fmt.Errorf("pnpm list failed: %v", err)
		}
	}

	pnpmFile := filepath.Join(p.outputDir, "pnpm-global.txt")
	return os.WriteFile(pnpmFile, output, 0644)
}

func (p *Packager) CollectGem() error {
	if !commandExists("gem") {
		return fmt.Errorf("gem not installed")
	}

	output, err := exec.Command("gem", "list").Output()
	if err != nil {
		return fmt.Errorf("gem list failed: %v", err)
	}

	gemFile := filepath.Join(p.outputDir, "gem-packages.txt")
	return os.WriteFile(gemFile, output, 0644)
}

func (p *Packager) CollectComposer() error {
	if !commandExists("composer") {
		return fmt.Errorf("composer not installed")
	}

	// Check if composer has global packages
	homeDir := os.Getenv("HOME")
	composerDir := filepath.Join(homeDir, ".composer", "composer.json")

	// If composer.json doesn't exist, there are no global packages
	if _, err := os.Stat(composerDir); os.IsNotExist(err) {
		// Create empty file to indicate composer is installed but no global packages
		composerFile := filepath.Join(p.outputDir, "composer-global.txt")
		return os.WriteFile(composerFile, []byte("# No global composer packages installed\n"), 0644)
	}

	output, err := exec.Command("composer", "global", "show").Output()
	if err != nil {
		// If command fails, create a note about it
		composerFile := filepath.Join(p.outputDir, "composer-global.txt")
		note := fmt.Sprintf("# composer global show failed: %v\n# composer.json exists at %s\n", err, composerDir)
		return os.WriteFile(composerFile, []byte(note), 0644)
	}

	composerFile := filepath.Join(p.outputDir, "composer-global.txt")
	return os.WriteFile(composerFile, output, 0644)
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func (p *Packager) DetectNonBrewApps() error {

	installedApps, err := p.getInstalledApps()
	if err != nil {
		return fmt.Errorf("failed to get installed apps: %w", err)
	}

	brewApps, err := p.getBrewCasks()
	if err != nil {

		brewApps = make(map[string]bool)
	}

	var nonBrewApps []AppInfo
	for _, app := range installedApps {
		appNameLower := strings.ToLower(strings.TrimSuffix(app.Name, ".app"))
		if !brewApps[appNameLower] {
			app.InHomebrew = false
			nonBrewApps = append(nonBrewApps, app)
		}
	}

	var output strings.Builder
	output.WriteString("# Applications not managed by Homebrew\n")
	output.WriteString("# These apps were likely installed manually (DMG, App Store, etc.)\n")
	output.WriteString("# You'll need to reinstall these manually after restore\n\n")

	for _, app := range nonBrewApps {
		output.WriteString(fmt.Sprintf("%s\n", app.Name))
	}

	outputFile := filepath.Join(p.outputDir, "non-brew-apps.txt")
	return os.WriteFile(outputFile, []byte(output.String()), 0644)
}

func (p *Packager) getInstalledApps() ([]AppInfo, error) {
	appDirs := []string{
		"/Applications",
		filepath.Join(os.Getenv("HOME"), "Applications"),
	}

	var apps []AppInfo
	for _, dir := range appDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".app") {
				apps = append(apps, AppInfo{
					Name: entry.Name(),
					Path: filepath.Join(dir, entry.Name()),
				})
			}
		}
	}

	return apps, nil
}

func (p *Packager) getBrewCasks() (map[string]bool, error) {
	if !commandExists("brew") {
		return nil, fmt.Errorf("brew not installed")
	}

	output, err := exec.Command("brew", "list", "--cask").Output()
	if err != nil {
		return nil, err
	}

	casks := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		cask := strings.TrimSpace(line)
		if cask != "" {
			casks[strings.ToLower(cask)] = true
		}
	}

	return casks, nil
}

func (p *Packager) countLines(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	lines := strings.Split(string(data), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			count++
		}
	}
	return count
}

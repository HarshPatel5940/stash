package packager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/harshpatel5940/stash/internal/ui"
)

// Installer provides progress-wrapped package installation
type Installer struct {
	verbose bool
}

// BrewInstallError captures brew bundle failure details for restore summaries.
type BrewInstallError struct {
	Err            error
	FailedPackages []string
	StderrTail     []string
}

func (e *BrewInstallError) Error() string {
	if len(e.FailedPackages) == 0 {
		return fmt.Sprintf("brew bundle failed: %v", e.Err)
	}
	return fmt.Sprintf("brew bundle failed: %v (failed: %s)", e.Err, strings.Join(e.FailedPackages, ", "))
}

func (e *BrewInstallError) Unwrap() error {
	return e.Err
}

// NewInstaller creates a new package installer
func NewInstaller(verbose bool) *Installer {
	return &Installer{verbose: verbose}
}

func (i *Installer) InstallBrewPackages(brewfilePath string) error {
	if !commandExists("brew") {
		return fmt.Errorf("brew not installed")
	}

	items, err := ParseBrewfile(brewfilePath)
	if err != nil {
		return fmt.Errorf("failed to parse Brewfile: %w", err)
	}
	if len(items) == 0 {
		fmt.Println("  No packages found in Brewfile")
		return nil
	}

	// Clear any stale update lock that can deadlock per-package installs.
	brewPrefix := os.Getenv("HOMEBREW_PREFIX")
	if brewPrefix == "" {
		if out, err := exec.Command("brew", "--prefix").Output(); err == nil {
			brewPrefix = strings.TrimSpace(string(out))
		}
	}
	if brewPrefix != "" {
		_ = os.Remove(filepath.Join(brewPrefix, "var", "homebrew", "locks", "update"))
	}

	env := append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1", "HOMEBREW_NO_ENV_HINTS=1")

	// Group counts for the user-facing summary line.
	var taps, brews, casks, masApps int
	for _, it := range items {
		switch it.Type {
		case "tap":
			taps++
		case "brew":
			brews++
		case "cask":
			casks++
		case "mas":
			masApps++
		}
	}
	fmt.Printf("  Installing %d Brewfile entries (%d taps, %d brews, %d casks, %d MAS)...\n",
		len(items), taps, brews, casks, masApps)

	bar := ui.NewProgressBar(len(items), "Homebrew")

	var (
		failed  []string
		skipped int
		ok      int
		errTail []string
	)

	for _, it := range items {
		bar.Describe(fmt.Sprintf("Homebrew · %s", truncate(it.Name, 32)))

		if i.brewItemInstalled(it) {
			skipped++
			bar.Add(1)
			continue
		}

		out, err := i.installBrewItemWithRetry(it, env, 2)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s %s", it.Type, it.Name))
			tail := lastNonEmptyLine(out)
			if tail != "" {
				errTail = append(errTail, fmt.Sprintf("  %s %s: %s", it.Type, it.Name, tail))
			}
			if i.verbose {
				fmt.Printf("\n    ✗ %s %s failed:\n%s\n", it.Type, it.Name, indent(out, "      "))
			}
		} else {
			ok++
		}
		bar.Add(1)
	}
	bar.Finish()

	fmt.Printf("  ✓ %d installed, ⤳ %d already present, ✗ %d failed\n", ok, skipped, len(failed))
	if len(failed) > 0 {
		fmt.Println("  Failed entries:")
		for _, line := range errTail {
			fmt.Println(line)
		}
		return &BrewInstallError{
			Err:            fmt.Errorf("%d package(s) failed", len(failed)),
			FailedPackages: failed,
			StderrTail:     errTail,
		}
	}
	return nil
}

// installBrewItemWithRetry runs the appropriate install command for one
// Brewfile entry, retrying transient (network / fetch) errors up to `retries`
// extra times with a short backoff.
func (i *Installer) installBrewItemWithRetry(it BrewfileItem, env []string, retries int) (string, error) {
	var lastOut []byte
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		args := brewInstallArgs(it)
		if args == nil {
			return "", fmt.Errorf("unsupported entry type: %s", it.Type)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err == nil {
			return string(out), nil
		}
		lastOut, lastErr = out, err
		if !isTransientBrewError(string(out)) {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	return string(lastOut), lastErr
}

func brewInstallArgs(it BrewfileItem) []string {
	switch it.Type {
	case "tap":
		return []string{"brew", "tap", it.Name}
	case "brew":
		return []string{"brew", "install", "--formula", it.Name}
	case "cask":
		return []string{"brew", "install", "--cask", it.Name}
	case "mas":
		id := extractMASID(it.RawLine)
		if id == "" {
			return nil
		}
		return []string{"mas", "install", id}
	}
	return nil
}

// brewItemInstalled does a cheap pre-check so we skip work and avoid network
// hits for the common case where most of the Brewfile is already satisfied.
func (i *Installer) brewItemInstalled(it BrewfileItem) bool {
	switch it.Type {
	case "brew":
		return exec.Command("brew", "list", "--formula", "--versions", it.Name).Run() == nil
	case "cask":
		return exec.Command("brew", "list", "--cask", "--versions", it.Name).Run() == nil
	case "tap":
		out, err := exec.Command("brew", "tap").Output()
		return err == nil && containsLine(string(out), it.Name)
	case "mas":
		id := extractMASID(it.RawLine)
		if id == "" || !commandExists("mas") {
			return false
		}
		out, err := exec.Command("mas", "list").Output()
		return err == nil && strings.Contains(string(out), id+" ")
	}
	return false
}

// isTransientBrewError matches the classic "retry-worthy" failures: HTTP
// fetch errors, GitHub rate limits, partial downloads. Compilation / linking
// failures are returned as-is so we don't waste minutes retrying them.
func isTransientBrewError(output string) bool {
	lower := strings.ToLower(output)
	needles := []string{
		"failed to fetch",
		"download failed",
		"connection reset",
		"connection refused",
		"could not resolve host",
		"network is unreachable",
		"operation timed out",
		"timeout was reached",
		"curl: (",
		"sha256 mismatch",
		"http 5",
		"http error 5",
		"rate limit",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

var masIDRe = regexp.MustCompile(`id:\s*(\d+)`)

func extractMASID(line string) string {
	m := masIDRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func containsLine(haystack, needle string) bool {
	for _, line := range strings.Split(haystack, "\n") {
		if strings.TrimSpace(line) == needle {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func lastNonEmptyLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for j := len(lines) - 1; j >= 0; j-- {
		if t := strings.TrimSpace(lines[j]); t != "" {
			return t
		}
	}
	return ""
}

func indent(s, prefix string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func parseBrewFailedPackages(lines []string) []string {
	pkgSet := map[string]struct{}{}
	re := regexp.MustCompile(`Failed to fetch (.+)$`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		matches := re.FindStringSubmatch(trimmed)
		if len(matches) != 2 {
			continue
		}

		parts := strings.Split(matches[1], ",")
		for _, part := range parts {
			pkg := strings.TrimSpace(part)
			if pkg != "" {
				pkgSet[pkg] = struct{}{}
			}
		}
	}

	if len(pkgSet) == 0 {
		return nil
	}

	failed := make([]string, 0, len(pkgSet))
	for pkg := range pkgSet {
		failed = append(failed, pkg)
	}
	sort.Strings(failed)
	return failed
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

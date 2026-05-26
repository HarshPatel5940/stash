// Package wallpaper backs up the current macOS desktop wallpaper image(s) so
// they survive a clean reset. macOS does not store wallpaper paths in any
// `defaults` domain we already capture — the picture lives in
// ~/Library/Application Support/Dock/desktoppicture.db plus the source image.
package wallpaper

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Entry struct {
	Desktop int    `json:"desktop"`
	Source  string `json:"source"`
	Saved   string `json:"saved"` // path within backup
}

type Manager struct {
	outputDir string
}

func NewManager(outputDir string) *Manager {
	return &Manager{outputDir: outputDir}
}

func (m *Manager) BackupAll() error {
	if err := os.MkdirAll(m.outputDir, 0755); err != nil {
		return err
	}

	paths, err := currentWallpaperPaths()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}

	var entries []Entry
	for i, p := range paths {
		if p == "" {
			continue
		}
		base := filepath.Base(p)
		dst := filepath.Join(m.outputDir, fmt.Sprintf("desktop-%d-%s", i+1, base))
		if err := copyFile(p, dst); err != nil {
			// Built-in wallpapers live under /System/Library/Desktop Pictures
			// and may be unreadable on some macOS versions — record the path
			// anyway so restore can re-point if the file is still present.
			entries = append(entries, Entry{Desktop: i + 1, Source: p})
			continue
		}
		entries = append(entries, Entry{Desktop: i + 1, Source: p, Saved: filepath.Base(dst)})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.outputDir, "wallpaper.json"), data, 0644)
}

func (m *Manager) Restore(backupFile string) error {
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	backupDir := filepath.Dir(backupFile)
	home, _ := os.UserHomeDir()
	wallpaperDir := filepath.Join(home, "Pictures", "Wallpapers")
	_ = os.MkdirAll(wallpaperDir, 0755)

	for _, e := range entries {
		target := e.Source
		// If the original path is still present (e.g. a built-in), use it
		// directly. Otherwise rehydrate from the backup copy.
		if _, err := os.Stat(target); err != nil {
			if e.Saved == "" {
				fmt.Printf("  ⚠️  Wallpaper %d: original missing and no saved copy\n", e.Desktop)
				continue
			}
			src := filepath.Join(backupDir, e.Saved)
			restored := filepath.Join(wallpaperDir, e.Saved)
			if err := copyFile(src, restored); err != nil {
				fmt.Printf("  ⚠️  Wallpaper %d: copy failed: %v\n", e.Desktop, err)
				continue
			}
			target = restored
		}

		script := fmt.Sprintf(`tell application "System Events" to tell every desktop to set picture to %q`, target)
		if err := exec.Command("osascript", "-e", script).Run(); err != nil {
			fmt.Printf("  ⚠️  Wallpaper %d: osascript failed: %v\n", e.Desktop, err)
		}
	}
	return nil
}

func currentWallpaperPaths() ([]string, error) {
	out, err := exec.Command("osascript", "-e",
		`tell application "System Events" to get picture of every desktop`).Output()
	if err != nil {
		return nil, err
	}
	// osascript prints a comma-separated list when there are multiple desktops.
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ", ")
	var paths []string
	for _, p := range parts {
		paths = append(paths, strings.TrimSpace(p))
	}
	return paths, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

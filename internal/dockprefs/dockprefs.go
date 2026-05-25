// Package dockprefs backs up and restores the macOS Dock in a portable way.
//
// `defaults import com.apple.dock` round-trips the raw plist but the
// persistent-apps array embeds bookmark NSData blobs that reference the
// source machine's inodes/volumes. After a clean restore those bookmarks are
// dangling, so the dock either appears empty or shows the wrong icons.
// We instead snapshot a portable description (bundle id + path + label)
// and rebuild dock entries at restore time against the current filesystem.
package dockprefs

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DockApp struct {
	Label            string `json:"label"`
	BundleIdentifier string `json:"bundle_identifier,omitempty"`
	Path             string `json:"path,omitempty"`
}

type DockPrefs struct {
	Orientation         string  `json:"orientation,omitempty"`
	TileSize            float64 `json:"tilesize,omitempty"`
	Autohide            bool    `json:"autohide"`
	AutohideDelay       float64 `json:"autohide_delay"`
	AutohideTimeMod     float64 `json:"autohide_time_modifier"`
	Magnification       bool    `json:"magnification"`
	LargeSize           float64 `json:"largesize,omitempty"`
	MinEffect           string  `json:"mineffect,omitempty"`
	ShowRecents         bool    `json:"show_recents"`
	PersistentApps      []DockApp `json:"persistent_apps"`
	PersistentOthers    []DockApp `json:"persistent_others"`
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

	prefs := DockPrefs{
		Orientation:     readDockString("orientation", "bottom"),
		TileSize:        readDockFloat("tilesize", 48),
		Autohide:        readDockBool("autohide"),
		AutohideDelay:   readDockFloat("autohide-delay", 0.5),
		AutohideTimeMod: readDockFloat("autohide-time-modifier", 0.5),
		Magnification:   readDockBool("magnification"),
		LargeSize:       readDockFloat("largesize", 64),
		MinEffect:       readDockString("mineffect", "genie"),
		ShowRecents:     readDockBool("show-recents"),
	}

	prefs.PersistentApps = readDockTiles("persistent-apps")
	prefs.PersistentOthers = readDockTiles("persistent-others")

	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.outputDir, "dock.json"), data, 0644)
}

func (m *Manager) Restore(backupFile string) error {
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return err
	}
	var prefs DockPrefs
	if err := json.Unmarshal(data, &prefs); err != nil {
		return err
	}

	// Scalar prefs apply cleanly via `defaults write` — these are the ones
	// the user noticed missing after a reset (left-side dock, auto-hide).
	writeDockString("orientation", prefs.Orientation)
	writeDockFloat("tilesize", prefs.TileSize)
	writeDockBool("autohide", prefs.Autohide)
	writeDockFloat("autohide-delay", prefs.AutohideDelay)
	writeDockFloat("autohide-time-modifier", prefs.AutohideTimeMod)
	writeDockBool("magnification", prefs.Magnification)
	writeDockFloat("largesize", prefs.LargeSize)
	writeDockString("mineffect", prefs.MinEffect)
	writeDockBool("show-recents", prefs.ShowRecents)

	if hasDockutil() {
		// dockutil regenerates bookmark data correctly against the live
		// filesystem; preferred when present.
		_ = exec.Command("dockutil", "--remove", "all", "--no-restart").Run()
		for _, app := range prefs.PersistentApps {
			path := resolveAppPath(app)
			if path == "" {
				fmt.Printf("  ⚠️  Dock app not found, skipping: %s\n", app.Label)
				continue
			}
			_ = exec.Command("dockutil", "--add", path, "--no-restart").Run()
		}
		for _, other := range prefs.PersistentOthers {
			if other.Path != "" {
				_ = exec.Command("dockutil", "--add", other.Path, "--no-restart").Run()
			}
		}
	} else {
		fmt.Println("  ⚠️  dockutil not installed — install via `brew install dockutil`")
		fmt.Println("     to restore pinned Dock apps. Layout settings (position,")
		fmt.Println("     autohide, size) have been applied.")
	}

	_ = exec.Command("killall", "cfprefsd").Run()
	_ = exec.Command("killall", "Dock").Run()
	return nil
}

func resolveAppPath(app DockApp) string {
	if app.Path != "" {
		if _, err := os.Stat(app.Path); err == nil {
			return app.Path
		}
	}
	// Try common rehydration locations using the label.
	base := strings.TrimSuffix(app.Label, ".app") + ".app"
	for _, dir := range []string{"/Applications", "/System/Applications", "/System/Applications/Utilities", "/Applications/Utilities"} {
		candidate := filepath.Join(dir, base)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Last resort: mdfind by bundle id.
	if app.BundleIdentifier != "" {
		out, err := exec.Command("mdfind", "kMDItemCFBundleIdentifier=="+app.BundleIdentifier).Output()
		if err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				if line != "" {
					return line
				}
			}
		}
	}
	return ""
}

func readDockTiles(key string) []DockApp {
	// Use plutil + defaults export to get a structured view.
	out, err := exec.Command("defaults", "export", "com.apple.dock", "-").Output()
	if err != nil {
		return nil
	}
	tmp, err := os.CreateTemp("", "dock-*.plist")
	if err != nil {
		return nil
	}
	defer os.Remove(tmp.Name())
	_, _ = tmp.Write(out)
	tmp.Close()

	jsonOut, err := exec.Command("plutil", "-convert", "json", "-o", "-", tmp.Name()).Output()
	if err != nil {
		return nil
	}
	var full map[string]any
	if err := json.Unmarshal(jsonOut, &full); err != nil {
		return nil
	}
	rawTiles, _ := full[key].([]any)
	apps := make([]DockApp, 0, len(rawTiles))
	for _, t := range rawTiles {
		tile, _ := t.(map[string]any)
		td, _ := tile["tile-data"].(map[string]any)
		if td == nil {
			continue
		}
		app := DockApp{}
		if label, ok := td["file-label"].(string); ok {
			app.Label = label
		}
		if bid, ok := td["bundle-identifier"].(string); ok {
			app.BundleIdentifier = bid
		}
		if fd, ok := td["file-data"].(map[string]any); ok {
			if cfurl, ok := fd["_CFURLString"].(string); ok {
				app.Path = decodeFileURL(cfurl)
			}
		}
		apps = append(apps, app)
	}
	return apps
}

func decodeFileURL(raw string) string {
	if !strings.HasPrefix(raw, "file://") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return strings.TrimSuffix(u.Path, "/")
}

func readDockString(key, fallback string) string {
	out, err := exec.Command("defaults", "read", "com.apple.dock", key).Output()
	if err != nil {
		return fallback
	}
	return strings.TrimSpace(string(out))
}

func readDockFloat(key string, fallback float64) float64 {
	out, err := exec.Command("defaults", "read", "com.apple.dock", key).Output()
	if err != nil {
		return fallback
	}
	var f float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &f); err != nil {
		return fallback
	}
	return f
}

func readDockBool(key string) bool {
	out, err := exec.Command("defaults", "read", "com.apple.dock", key).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "1"
}

func writeDockString(key, val string) {
	if val == "" {
		return
	}
	_ = exec.Command("defaults", "write", "com.apple.dock", key, "-string", val).Run()
}

func writeDockFloat(key string, val float64) {
	if val == 0 {
		return
	}
	_ = exec.Command("defaults", "write", "com.apple.dock", key, "-float", fmt.Sprintf("%g", val)).Run()
}

func writeDockBool(key string, val bool) {
	v := "false"
	if val {
		v = "true"
	}
	_ = exec.Command("defaults", "write", "com.apple.dock", key, "-bool", v).Run()
}

func hasDockutil() bool {
	_, err := exec.LookPath("dockutil")
	return err == nil
}

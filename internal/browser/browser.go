package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harshpatel5940/stash/internal/security"
)

type BrowserManager struct {
	outputDir string
}

type BrowserInfo struct {
	Name          string
	Path          string
	FilesToBackup []string
}

func NewBrowserManager(outputDir string) *BrowserManager {
	return &BrowserManager{
		outputDir: outputDir,
	}
}

func (bm *BrowserManager) GetBrowsers() []BrowserInfo {
	homeDir, _ := os.UserHomeDir()

	browsers := []BrowserInfo{
		{
			Name: "Chrome",
			Path: filepath.Join(homeDir, "Library/Application Support/Google/Chrome"),
			FilesToBackup: []string{
				"Default/Bookmarks",
				"Default/Preferences",
				"Default/Extensions",
				"Local State",
			},
		},
		{
			Name: "Brave",
			Path: filepath.Join(homeDir, "Library/Application Support/BraveSoftware/Brave-Browser"),
			FilesToBackup: []string{
				"Default/Bookmarks",
				"Default/Preferences",
				"Default/Extensions",
				"Local State",
			},
		},
		{
			Name: "Edge",
			Path: filepath.Join(homeDir, "Library/Application Support/Microsoft Edge"),
			FilesToBackup: []string{
				"Default/Bookmarks",
				"Default/Preferences",
				"Default/Extensions",
				"Local State",
			},
		},
		{
			Name: "Opera",
			Path: filepath.Join(homeDir, "Library/Application Support/com.operasoftware.Opera"),
			FilesToBackup: []string{
				"Bookmarks",
				"Preferences",
				"Extensions",
				"Local State",
			},
		},
		{
			Name: "Vivaldi",
			Path: filepath.Join(homeDir, "Library/Application Support/Vivaldi"),
			FilesToBackup: []string{
				"Default/Bookmarks",
				"Default/Preferences",
				"Default/Extensions",
				"Local State",
			},
		},
		{
			Name: "Firefox",
			Path: filepath.Join(homeDir, "Library/Application Support/Firefox"),
			FilesToBackup: []string{
				"profiles.ini",
			},
		},
		{
			Name: "Safari",
			Path: filepath.Join(homeDir, "Library/Safari"),
			FilesToBackup: []string{
				"Bookmarks.plist",
				"TopSites.plist",
			},
		},
		{
			Name: "Arc",
			Path: filepath.Join(homeDir, "Library/Application Support/Arc"),
			FilesToBackup: []string{
				"User Data/Default/Bookmarks",
				"User Data/Default/Preferences",
			},
		},
	}

	return browsers
}

func (bm *BrowserManager) BackupAll() (map[string]int, error) {
	if err := os.MkdirAll(bm.outputDir, 0755); err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	browsers := bm.GetBrowsers()

	for _, browser := range browsers {

		if _, err := os.Stat(browser.Path); os.IsNotExist(err) {
			continue
		}

		browserDir := filepath.Join(bm.outputDir, strings.ToLower(browser.Name))
		if err := os.MkdirAll(browserDir, 0755); err != nil {
			continue
		}

		fileCount := 0

		if browser.Name == "Firefox" {
			fileCount = bm.backupFirefoxProfiles(browser.Path, browserDir)
		} else {

			for _, file := range browser.FilesToBackup {
				srcPath := filepath.Join(browser.Path, file)

				info, err := os.Stat(srcPath)
				if err != nil {
					continue
				}

				destPath := filepath.Join(browserDir, filepath.Base(file))

				if info.IsDir() {
					if err := copyDir(srcPath, destPath); err != nil {
						continue
					}
				} else {
					if err := copyFile(srcPath, destPath); err != nil {
						continue
					}
				}
				fileCount++
			}
		}

		if fileCount > 0 {
			counts[browser.Name] = fileCount
		}
	}

	readmePath := filepath.Join(bm.outputDir, "README.txt")
	readme := `Browser Data Backup

This directory contains backups of browser data including:
- Bookmarks
- Extensions
- Preferences
- Settings

To restore:
1. Close all browser instances
2. Copy the backed up files to their original locations
3. Restart the browser

WARNING: This may overwrite your current browser data!
Consider exporting/merging bookmarks manually if needed.
`
	os.WriteFile(readmePath, []byte(readme), 0644)

	if len(counts) == 0 {
		return counts, fmt.Errorf("no browsers found")
	}

	return counts, nil
}

func (bm *BrowserManager) backupFirefoxProfiles(firefoxPath, outputDir string) int {
	profilesPath := filepath.Join(firefoxPath, "Profiles")
	if _, err := os.Stat(profilesPath); os.IsNotExist(err) {
		return 0
	}

	profilesIni := filepath.Join(firefoxPath, "profiles.ini")
	if _, err := os.Stat(profilesIni); err == nil {
		copyFile(profilesIni, filepath.Join(outputDir, "profiles.ini"))
	}

	entries, err := os.ReadDir(profilesPath)
	if err != nil {
		return 0
	}

	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if !strings.Contains(entry.Name(), "default") {
			continue
		}

		profilePath := filepath.Join(profilesPath, entry.Name())
		profileBackupDir := filepath.Join(outputDir, "profile-"+entry.Name())

		importantFiles := []string{
			"places.sqlite",
			"key4.db",
			"logins.json",
			"prefs.js",
			"extensions.json",
		}

		os.MkdirAll(profileBackupDir, 0755)

		for _, file := range importantFiles {
			src := filepath.Join(profilePath, file)
			if _, err := os.Stat(src); err == nil {
				dest := filepath.Join(profileBackupDir, file)
				if copyFile(src, dest) == nil {
					fileCount++
				}
			}
		}

		extensionsDir := filepath.Join(profilePath, "extensions")
		if _, err := os.Stat(extensionsDir); err == nil {
			destExtDir := filepath.Join(profileBackupDir, "extensions")
			if copyDir(extensionsDir, destExtDir) == nil {
				fileCount++
			}
		}
	}

	return fileCount
}

func copyFile(src, dst string) error {
	// Sanitize paths
	src = security.CleanPath(src)
	dst = security.CleanPath(dst)

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, info.Mode())
}

func copyDir(src, dst string) error {
	// Sanitize paths
	src = security.CleanPath(src)
	dst = security.CleanPath(dst)

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := security.CleanPath(filepath.Join(src, entry.Name()))
		dstPath := security.CleanPath(filepath.Join(dst, entry.Name()))

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				continue
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				continue
			}
		}
	}

	return nil
}

func (bm *BrowserManager) GetStats() (int, error) {
	entries, err := os.ReadDir(bm.outputDir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}

	return count, nil
}

package fonts

import (
	"fmt"
	"os"
	"path/filepath"
)

type FontsManager struct {
	outputDir string
}

func NewFontsManager(outputDir string) *FontsManager {
	return &FontsManager{
		outputDir: outputDir,
	}
}

func (fm *FontsManager) BackupAll() (int, error) {
	homeDir, _ := os.UserHomeDir()
	fontsDir := filepath.Join(homeDir, "Library", "Fonts")

	if _, err := os.Stat(fontsDir); os.IsNotExist(err) {
		return 0, fmt.Errorf("fonts directory not found")
	}

	if err := os.MkdirAll(fm.outputDir, 0755); err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(fontsDir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		validExtensions := map[string]bool{
			".ttf":   true,
			".otf":   true,
			".ttc":   true,
			".dfont": true,
		}

		if !validExtensions[ext] {
			continue
		}

		srcPath := filepath.Join(fontsDir, entry.Name())
		destPath := filepath.Join(fm.outputDir, entry.Name())

		if err := copyFile(srcPath, destPath); err != nil {
			continue
		}

		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("no custom fonts found")
	}

	readmePath := filepath.Join(fm.outputDir, "README.txt")
	readme := `Custom Fonts Backup

This directory contains your custom fonts from ~/Library/Fonts

To restore:
1. Copy all font files to ~/Library/Fonts/
2. Or double-click each font file to install via Font Book
3. Fonts will be available system-wide after installation

Font formats supported:
- .ttf (TrueType Font)
- .otf (OpenType Font)
- .ttc (TrueType Collection)
- .dfont (Mac DFONT)
`
	os.WriteFile(readmePath, []byte(readme), 0644)

	return count, nil
}

func (fm *FontsManager) RestoreAll(backupDir string) (int, error) {
	homeDir, _ := os.UserHomeDir()
	fontsDir := filepath.Join(homeDir, "Library", "Fonts")

	if err := os.MkdirAll(fontsDir, 0755); err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "README.txt" {
			continue
		}

		srcPath := filepath.Join(backupDir, entry.Name())
		destPath := filepath.Join(fontsDir, entry.Name())

		if err := copyFile(srcPath, destPath); err != nil {
			fmt.Printf("  ⚠️  Failed to restore font %s: %v\n", entry.Name(), err)
			continue
		}

		count++
	}

	return count, nil
}

func (fm *FontsManager) GetStats() (int, error) {
	entries, err := os.ReadDir(fm.outputDir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != "README.txt" {
			ext := filepath.Ext(entry.Name())
			validExtensions := map[string]bool{
				".ttf":   true,
				".otf":   true,
				".ttc":   true,
				".dfont": true,
			}
			if validExtensions[ext] {
				count++
			}
		}
	}

	return count, nil
}

func copyFile(src, dst string) error {
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

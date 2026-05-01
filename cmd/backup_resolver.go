package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func collectBackups(backupDir string) ([]backupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	var backups []backupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".tar.gz.age") && !strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, backupInfo{
			Path:      filepath.Join(backupDir, name),
			Name:      name,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			Encrypted: strings.HasSuffix(name, ".age"),
		})
	}

	sortBackups(backups)
	return backups, nil
}

func sortBackups(backups []backupInfo) {
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})
	for i := range backups {
		backups[i].Index = i + 1
	}
}

func resolveBackupInput(input, backupDir string) (backupInfo, error) {
	var zero backupInfo
	input = strings.TrimSpace(input)
	if input == "" {
		return zero, fmt.Errorf("backup reference is empty")
	}

	backups, err := collectBackups(backupDir)
	if err == nil {
		if idx, parseErr := strconv.Atoi(input); parseErr == nil {
			if idx < 1 || idx > len(backups) {
				return zero, fmt.Errorf("backup index out of range: %d (found %d backups)", idx, len(backups))
			}
			return backups[idx-1], nil
		}
	}

	if info, ok := resolveExistingPath(input); ok {
		info.Index = findBackupIndex(backups, info.Path)
		return info, nil
	}

	candidates := []string{
		filepath.Join(backupDir, input),
		filepath.Join(backupDir, input+".tar.gz.age"),
		filepath.Join(backupDir, input+".tar.gz"),
	}
	for _, candidate := range candidates {
		if info, ok := resolveExistingPath(candidate); ok {
			info.Index = findBackupIndex(backups, info.Path)
			return info, nil
		}
	}

	for _, backup := range backups {
		base := normalizeBackupKey(backup.Name)
		if input == backup.Name || input == base {
			return backup, nil
		}
	}

	return zero, fmt.Errorf("backup not found: %s", input)
}

func resolveExistingPath(path string) (backupInfo, bool) {
	var zero backupInfo
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return zero, false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	name := filepath.Base(path)
	return backupInfo{
		Path:      absPath,
		Name:      name,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		Encrypted: strings.HasSuffix(name, ".age"),
	}, true
}

func findBackupIndex(backups []backupInfo, path string) int {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	for _, backup := range backups {
		if samePath(backup.Path, absPath) {
			return backup.Index
		}
	}
	return 0
}

func samePath(a, b string) bool {
	aa, err := filepath.Abs(a)
	if err != nil {
		aa = a
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		bb = b
	}
	return aa == bb
}

func backupDate(metaTime time.Time, fallback time.Time) time.Time {
	if metaTime.IsZero() {
		return fallback
	}
	return metaTime
}

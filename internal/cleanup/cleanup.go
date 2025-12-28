package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type BackupFile struct {
	Path    string
	ModTime time.Time
	Size    int64
}

type CleanupManager struct {
	backupDir string
}

func NewCleanupManager(backupDir string) *CleanupManager {
	return &CleanupManager{
		backupDir: backupDir,
	}
}

func (cm *CleanupManager) GetBackups() ([]BackupFile, error) {
	if _, err := os.Stat(cm.backupDir); os.IsNotExist(err) {
		return []BackupFile{}, nil
	}

	entries, err := os.ReadDir(cm.backupDir)
	if err != nil {
		return nil, err
	}

	var backups []BackupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".age" && ext != ".gz" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		backups = append(backups, BackupFile{
			Path:    filepath.Join(cm.backupDir, name),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	return backups, nil
}

func (cm *CleanupManager) RotateByCount(keepCount int) (int, error) {
	backups, err := cm.GetBackups()
	if err != nil {
		return 0, err
	}

	if len(backups) <= keepCount {
		return 0, nil
	}

	deleted := 0
	for i := keepCount; i < len(backups); i++ {
		if err := os.Remove(backups[i].Path); err != nil {
			continue
		}
		deleted++
	}

	return deleted, nil
}

func (cm *CleanupManager) RotateByAge(maxAge time.Duration) (int, error) {
	backups, err := cm.GetBackups()
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-maxAge)
	deleted := 0

	for _, backup := range backups {
		if backup.ModTime.Before(cutoff) {
			if err := os.Remove(backup.Path); err != nil {
				continue
			}
			deleted++
		}
	}

	return deleted, nil
}

func (cm *CleanupManager) RotateBySize(maxSizeBytes int64) (int, error) {
	backups, err := cm.GetBackups()
	if err != nil {
		return 0, err
	}

	var totalSize int64
	deleted := 0

	for _, backup := range backups {
		if totalSize+backup.Size > maxSizeBytes {

			if err := os.Remove(backup.Path); err != nil {
				continue
			}
			deleted++
		} else {
			totalSize += backup.Size
		}
	}

	return deleted, nil
}

func (cm *CleanupManager) GetTotalSize() (int64, error) {
	backups, err := cm.GetBackups()
	if err != nil {
		return 0, err
	}

	var total int64
	for _, backup := range backups {
		total += backup.Size
	}

	return total, nil
}

func (cm *CleanupManager) GetStats() (map[string]interface{}, error) {
	backups, err := cm.GetBackups()
	if err != nil {
		return nil, err
	}

	totalSize, _ := cm.GetTotalSize()

	stats := map[string]interface{}{
		"count":      len(backups),
		"total_size": totalSize,
	}

	if len(backups) > 0 {
		stats["oldest"] = backups[len(backups)-1].ModTime
		stats["newest"] = backups[0].ModTime
	}

	return stats, nil
}

func (cm *CleanupManager) ListBackups() ([]string, error) {
	backups, err := cm.GetBackups()
	if err != nil {
		return nil, err
	}

	var list []string
	for _, backup := range backups {
		size := formatBytes(backup.Size)
		age := time.Since(backup.ModTime)
		name := filepath.Base(backup.Path)

		ageStr := formatDuration(age)
		list = append(list, fmt.Sprintf("%s - %s (%s ago)", name, size, ageStr))
	}

	return list, nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d min", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	if days < 30 {
		return fmt.Sprintf("%d days", days)
	}
	months := days / 30
	if months == 1 {
		return "1 month"
	}
	if months < 12 {
		return fmt.Sprintf("%d months", months)
	}
	years := months / 12
	if years == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", years)
}

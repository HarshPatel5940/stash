// Package incremental provides incremental backup functionality for stash.
// It determines when full vs incremental backups are needed, tracks changed
// files between backups, and manages the backup index for efficient operation.
//
// Incremental backups only include files that have changed since the last
// backup, reducing backup time and storage requirements significantly.
package incremental

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/index"
)

// Manager handles incremental backup operations
type Manager struct {
	index      *index.BackupIndex
	indexPath  string
	cfg        *config.Config
	baseBackup string
}

// NewManager creates a new incremental backup manager
func NewManager(cfg *config.Config) (*Manager, error) {
	indexPath := index.GetDefaultIndexPath()

	// Load existing index
	idx, err := index.Load(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	return &Manager{
		index:     idx,
		indexPath: indexPath,
		cfg:       cfg,
	}, nil
}

// ShouldDoFullBackup determines if a full backup is needed
func (m *Manager) ShouldDoFullBackup() bool {
	// Always do full backup if no previous backups
	if m.index.GetFileCount() == 0 {
		return true
	}

	// Get interval from config, default to 7 days
	interval := 7 * 24 * time.Hour
	if m.cfg != nil && m.cfg.Incremental != nil {
		if parsed := parseIntervalString(m.cfg.Incremental.FullBackupInterval); parsed > 0 {
			interval = parsed
		}
	}

	return m.index.NeedFullBackup(interval)
}

// parseIntervalString parses interval strings like "7d", "24h", "30d" into time.Duration
func parseIntervalString(s string) time.Duration {
	if s == "" {
		return 0
	}

	// Try standard duration format first (e.g., "24h", "168h")
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}

	// Handle day format (e.g., "7d", "30d")
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}

	return 0
}

// FindChangedFiles finds all files that have changed since last backup
func (m *Manager) FindChangedFiles(allFiles []string) ([]string, error) {
	changed := make([]string, 0)

	for _, file := range allFiles {
		hasChanged, err := m.index.HasChanged(file)
		if err != nil {
			// Skip files we can't check
			continue
		}

		if hasChanged {
			changed = append(changed, file)
		}
	}

	return changed, nil
}

// GetBaseBackup returns the most recent full backup name
func (m *Manager) GetBaseBackup() string {
	// First check if we have it cached
	if m.baseBackup != "" {
		return m.baseBackup
	}

	// Otherwise get from the index
	return m.index.GetLastFullBackupName()
}

// UpdateIndex updates the index with newly backed up files
func (m *Manager) UpdateIndex(backupName string, files []string, isFull bool) error {
	// Create fingerprints for all files
	for _, file := range files {
		fp, err := index.CreateFingerprint(file, backupName)
		if err != nil {
			continue // Skip files we can't fingerprint
		}

		m.index.AddFile(file, fp)
	}

	// Update timestamps
	now := time.Now()
	if isFull {
		m.index.MarkFullBackup(now, backupName)
		m.baseBackup = backupName
	} else {
		m.index.MarkIncrementalBackup(now)
	}

	// Save index
	if err := m.index.Save(m.indexPath); err != nil {
		return fmt.Errorf("failed to save index: %w", err)
	}

	return nil
}

// GetStats returns statistics about the index
func (m *Manager) GetStats() (fileCount int, totalSize int64, lastBackup time.Time) {
	return m.index.GetFileCount(), m.index.GetTotalSize(), m.index.LastBackup
}

// GetLastFullBackup returns the timestamp of the last full backup
func (m *Manager) GetLastFullBackup() time.Time {
	return m.index.LastFullBackup
}

// IsFirstBackup returns true if this is the first backup
func (m *Manager) IsFirstBackup() bool {
	return m.index.GetFileCount() == 0
}

// EstimateSavings estimates how much space/time will be saved by incremental backup
func (m *Manager) EstimateSavings(totalFiles int) (filesSkipped int, percentSaved float64) {
	if totalFiles == 0 {
		return 0, 0
	}

	// Count how many files are unchanged
	indexedCount := m.index.GetFileCount()
	if indexedCount == 0 {
		return 0, 0
	}

	// Rough estimate: assume most indexed files are unchanged
	// In practice, this depends on user's workflow
	filesSkipped = indexedCount
	if filesSkipped > totalFiles {
		filesSkipped = totalFiles
	}

	percentSaved = (float64(filesSkipped) / float64(totalFiles)) * 100

	return filesSkipped, percentSaved
}

// GetChangedFilesByPath scans specific paths for changes
func (m *Manager) GetChangedFilesByPath(paths []string) (changed []string, total int, err error) {
	allFiles := make([]string, 0)

	// Walk each path to find all files
	for _, searchPath := range paths {
		// Expand home directory
		if len(searchPath) > 0 && searchPath[0] == '~' {
			homeDir, _ := os.UserHomeDir()
			searchPath = filepath.Join(homeDir, searchPath[1:])
		}

		// Skip if path doesn't exist
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			// Skip directories and certain patterns
			if info.IsDir() {
				name := info.Name()
				// Skip hidden dirs and common excludes
				if name == ".git" || name == "node_modules" || name == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}

			allFiles = append(allFiles, path)
			return nil
		})

		if err != nil {
			return nil, 0, err
		}
	}

	// Find which files have changed
	changedFiles, err := m.FindChangedFiles(allFiles)
	if err != nil {
		return nil, 0, err
	}

	return changedFiles, len(allFiles), nil
}

// GetBackupType returns the recommended backup type
func (m *Manager) GetBackupType() string {
	if m.ShouldDoFullBackup() {
		return "full"
	}
	return "incremental"
}

// GetRecommendation returns a recommendation for the backup
func (m *Manager) GetRecommendation() string {
	if m.IsFirstBackup() {
		return "First backup - full backup required"
	}

	if m.ShouldDoFullBackup() {
		lastFull := m.GetLastFullBackup()
		daysSince := int(time.Since(lastFull).Hours() / 24)
		return fmt.Sprintf("Full backup recommended (last full backup was %d days ago)", daysSince)
	}

	return "Incremental backup recommended (only changed files will be backed up)"
}

// CleanupOldIndex removes files from index that no longer exist
func (m *Manager) CleanupOldIndex() (removed int, err error) {
	toRemove := make([]string, 0)

	// Check each indexed file
	for path := range m.index.Files {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			toRemove = append(toRemove, path)
		}
	}

	// Remove non-existent files
	for _, path := range toRemove {
		m.index.RemoveFile(path)
	}

	// Save updated index
	if len(toRemove) > 0 {
		if err := m.index.Save(m.indexPath); err != nil {
			return 0, fmt.Errorf("failed to save index after cleanup: %w", err)
		}
	}

	return len(toRemove), nil
}

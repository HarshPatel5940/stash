// Package diff provides backup comparison functionality.
// It compares two backups and reports added, removed, and modified files,
// as well as changes to package manager counts and overall size changes.
//
// Use this package to understand what changed between two points in time.
package diff

import (
	"fmt"
	"os"
	"sort"

	"github.com/harshpatel5940/stash/internal/backuputil"
	"github.com/harshpatel5940/stash/internal/metadata"
)

// FileChange represents a change to a file between backups
type FileChange struct {
	Path        string
	OldSize     int64
	NewSize     int64
	SizeDelta   int64
	OldChecksum string
	NewChecksum string
}

// BackupDiff represents the differences between two backups
type BackupDiff struct {
	OldBackup      string
	NewBackup      string
	OldSize        int64
	NewSize        int64
	AddedFiles     []metadata.FileInfo
	RemovedFiles   []metadata.FileInfo
	ModifiedFiles  []FileChange
	UnchangedCount int
	AddedSize      int64
	RemovedSize    int64
	ModifiedSize   int64
	PackageChanges map[string]PackageChange
}

// PackageChange represents changes in a package manager
type PackageChange struct {
	Name     string
	OldCount int
	NewCount int
	Delta    int
}

// CompareOptions contains options for comparing backups
type CompareOptions struct {
	KeyPath string // Path to decryption key (optional, defaults to ~/.stash.key)
}

// Compare compares two backups and returns the differences
func Compare(oldBackupPath, newBackupPath string) (*BackupDiff, error) {
	return CompareWithOptions(oldBackupPath, newBackupPath, CompareOptions{})
}

// CompareWithOptions compares two backups with custom options
func CompareWithOptions(oldBackupPath, newBackupPath string, opts CompareOptions) (*BackupDiff, error) {
	// Load metadata from both backups
	oldMeta, err := loadBackupMetadata(oldBackupPath, opts.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load old backup metadata: %w", err)
	}

	newMeta, err := loadBackupMetadata(newBackupPath, opts.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load new backup metadata: %w", err)
	}

	// Get backup sizes
	oldStat, _ := os.Stat(oldBackupPath)
	newStat, _ := os.Stat(newBackupPath)

	oldSize := int64(0)
	newSize := int64(0)
	if oldStat != nil {
		oldSize = oldStat.Size()
	}
	if newStat != nil {
		newSize = newStat.Size()
	}

	diff := &BackupDiff{
		OldBackup:      oldBackupPath,
		NewBackup:      newBackupPath,
		OldSize:        oldSize,
		NewSize:        newSize,
		AddedFiles:     []metadata.FileInfo{},
		RemovedFiles:   []metadata.FileInfo{},
		ModifiedFiles:  []FileChange{},
		PackageChanges: make(map[string]PackageChange),
	}

	// Create maps for quick lookup
	oldFiles := make(map[string]metadata.FileInfo)
	newFiles := make(map[string]metadata.FileInfo)

	for _, file := range oldMeta.Files {
		oldFiles[file.OriginalPath] = file
	}

	for _, file := range newMeta.Files {
		newFiles[file.OriginalPath] = file
	}

	// Find added and modified files
	for path, newFile := range newFiles {
		if oldFile, exists := oldFiles[path]; exists {
			// File exists in both - check if modified
			if !newFile.IsDir && (newFile.Size != oldFile.Size || newFile.Checksum != oldFile.Checksum) {
				diff.ModifiedFiles = append(diff.ModifiedFiles, FileChange{
					Path:        path,
					OldSize:     oldFile.Size,
					NewSize:     newFile.Size,
					SizeDelta:   newFile.Size - oldFile.Size,
					OldChecksum: oldFile.Checksum,
					NewChecksum: newFile.Checksum,
				})
				diff.ModifiedSize += (newFile.Size - oldFile.Size)
			} else {
				diff.UnchangedCount++
			}
		} else {
			// File added
			diff.AddedFiles = append(diff.AddedFiles, newFile)
			if !newFile.IsDir {
				diff.AddedSize += newFile.Size
			}
		}
	}

	// Find removed files
	for path, oldFile := range oldFiles {
		if _, exists := newFiles[path]; !exists {
			diff.RemovedFiles = append(diff.RemovedFiles, oldFile)
			if !oldFile.IsDir {
				diff.RemovedSize += oldFile.Size
			}
		}
	}

	// Compare package counts
	for pkgType, newCount := range newMeta.PackageCounts {
		oldCount := oldMeta.PackageCounts[pkgType]
		if oldCount != newCount {
			diff.PackageChanges[pkgType] = PackageChange{
				Name:     pkgType,
				OldCount: oldCount,
				NewCount: newCount,
				Delta:    newCount - oldCount,
			}
		}
	}

	// Check for removed packages
	for pkgType, oldCount := range oldMeta.PackageCounts {
		if _, exists := newMeta.PackageCounts[pkgType]; !exists {
			diff.PackageChanges[pkgType] = PackageChange{
				Name:     pkgType,
				OldCount: oldCount,
				NewCount: 0,
				Delta:    -oldCount,
			}
		}
	}

	// Sort results for consistent output
	sort.Slice(diff.AddedFiles, func(i, j int) bool {
		return diff.AddedFiles[i].OriginalPath < diff.AddedFiles[j].OriginalPath
	})

	sort.Slice(diff.RemovedFiles, func(i, j int) bool {
		return diff.RemovedFiles[i].OriginalPath < diff.RemovedFiles[j].OriginalPath
	})

	sort.Slice(diff.ModifiedFiles, func(i, j int) bool {
		return diff.ModifiedFiles[i].Path < diff.ModifiedFiles[j].Path
	})

	return diff, nil
}

// loadBackupMetadata loads metadata from a backup
func loadBackupMetadata(backupPath string, keyPath string) (*metadata.Metadata, error) {
	// First, try to find a sidecar metadata file (for backwards compatibility)
	metadataPath := backupPath + ".metadata.json"
	if _, err := os.Stat(metadataPath); err == nil {
		return metadata.Load(metadataPath)
	}

	// Extract metadata from the backup archive (handles both encrypted and unencrypted)
	return backuputil.ExtractMetadata(backupPath, keyPath)
}

// GetAddedFilesCount returns the number of added files (excluding directories)
func (d *BackupDiff) GetAddedFilesCount() int {
	count := 0
	for _, file := range d.AddedFiles {
		if !file.IsDir {
			count++
		}
	}
	return count
}

// GetRemovedFilesCount returns the number of removed files (excluding directories)
func (d *BackupDiff) GetRemovedFilesCount() int {
	count := 0
	for _, file := range d.RemovedFiles {
		if !file.IsDir {
			count++
		}
	}
	return count
}

// GetModifiedFilesCount returns the number of modified files
func (d *BackupDiff) GetModifiedFilesCount() int {
	return len(d.ModifiedFiles)
}

// HasChanges returns true if there are any changes between the backups
func (d *BackupDiff) HasChanges() bool {
	return len(d.AddedFiles) > 0 || len(d.RemovedFiles) > 0 || len(d.ModifiedFiles) > 0 || len(d.PackageChanges) > 0
}

// GetTotalFileChanges returns the total number of file changes
func (d *BackupDiff) GetTotalFileChanges() int {
	return d.GetAddedFilesCount() + d.GetRemovedFilesCount() + d.GetModifiedFilesCount()
}

// GetTopAddedFiles returns the N largest added files
func (d *BackupDiff) GetTopAddedFiles(n int) []metadata.FileInfo {
	files := make([]metadata.FileInfo, len(d.AddedFiles))
	copy(files, d.AddedFiles)

	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})

	if len(files) > n {
		return files[:n]
	}
	return files
}

// GetTopModifiedFiles returns the N files with the largest size changes
func (d *BackupDiff) GetTopModifiedFiles(n int) []FileChange {
	changes := make([]FileChange, len(d.ModifiedFiles))
	copy(changes, d.ModifiedFiles)

	sort.Slice(changes, func(i, j int) bool {
		absI := changes[i].SizeDelta
		absJ := changes[j].SizeDelta
		if absI < 0 {
			absI = -absI
		}
		if absJ < 0 {
			absJ = -absJ
		}
		return absI > absJ
	})

	if len(changes) > n {
		return changes[:n]
	}
	return changes
}

// GetSizeDelta returns the total size change between backups
func (d *BackupDiff) GetSizeDelta() int64 {
	return d.AddedSize - d.RemovedSize + d.ModifiedSize
}

// Summary returns a summary of the changes
func (d *BackupDiff) Summary() string {
	if !d.HasChanges() {
		return "No changes detected between backups"
	}

	summary := fmt.Sprintf("Changes: +%d added, -%d removed, ~%d modified files\n",
		d.GetAddedFilesCount(),
		d.GetRemovedFilesCount(),
		d.GetModifiedFilesCount())

	sizeDelta := d.GetSizeDelta()
	sign := "+"
	if sizeDelta < 0 {
		sign = ""
	}

	summary += fmt.Sprintf("Size change: %s%s\n", sign, formatBytes(sizeDelta))

	if len(d.PackageChanges) > 0 {
		summary += fmt.Sprintf("Package changes: %d package managers affected\n", len(d.PackageChanges))
	}

	return summary
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	absBytes := bytes
	if absBytes < 0 {
		absBytes = -absBytes
	}

	const unit = 1024
	if absBytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := absBytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	value := float64(bytes) / float64(div)
	return fmt.Sprintf("%.1f %cB", value, "KMGTPE"[exp])
}

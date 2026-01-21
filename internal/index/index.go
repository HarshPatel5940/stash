// Package index provides a backup index system for tracking file states.
// It maintains fingerprints (size, modification time, checksums) of backed-up
// files to enable efficient incremental backups by detecting changed files.
//
// The index is persisted as JSON to ~/.stash-index.json and supports
// thread-safe concurrent access through mutex protection.
package index

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileFingerprint represents a file's state for change detection
type FileFingerprint struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	Checksum   string    `json:"checksum"`
	BackupedIn string    `json:"backuped_in"` // which backup contains this version
}

// BackupIndex tracks all backed-up files and their states
type BackupIndex struct {
	Version            string                      `json:"version"`
	LastFullBackup     time.Time                   `json:"last_full_backup"`
	LastFullBackupName string                      `json:"last_full_backup_name,omitempty"`
	LastBackup         time.Time                   `json:"last_backup"`
	Files              map[string]*FileFingerprint `json:"files"`
	mu                 sync.RWMutex
}

// New creates a new backup index
func New() *BackupIndex {
	return &BackupIndex{
		Version: "1.0",
		Files:   make(map[string]*FileFingerprint),
	}
}

// Load loads a backup index from file
func Load(path string) (*BackupIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return New(), nil // Return empty index if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	var idx BackupIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	if idx.Files == nil {
		idx.Files = make(map[string]*FileFingerprint)
	}

	return &idx, nil
}

// Save saves the backup index to file
func (idx *BackupIndex) Save(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

// AddFile adds or updates a file in the index
func (idx *BackupIndex) AddFile(path string, fingerprint *FileFingerprint) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.Files[path] = fingerprint
}

// GetFile retrieves a file's fingerprint from the index
func (idx *BackupIndex) GetFile(path string) (*FileFingerprint, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	fp, exists := idx.Files[path]
	return fp, exists
}

// HasChanged checks if a file has changed since last backup
func (idx *BackupIndex) HasChanged(path string) (bool, error) {
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File was deleted
			return true, nil
		}
		return false, err
	}

	// Check if file exists in index
	idx.mu.RLock()
	previous, exists := idx.Files[path]
	idx.mu.RUnlock()

	if !exists {
		// New file
		return true, nil
	}

	// Quick check: size or mtime changed
	if info.Size() != previous.Size || !info.ModTime().Equal(previous.ModTime) {
		return true, nil
	}

	// If size and mtime are same, file is likely unchanged
	// For extra safety, we could verify checksum here, but that's expensive
	return false, nil
}

// GetChangedFiles returns a list of files that have changed
func (idx *BackupIndex) GetChangedFiles(paths []string) ([]string, error) {
	var changed []string

	for _, path := range paths {
		hasChanged, err := idx.HasChanged(path)
		if err != nil {
			continue // Skip files we can't read
		}

		if hasChanged {
			changed = append(changed, path)
		}
	}

	return changed, nil
}

// CreateFingerprint creates a fingerprint for a file
func CreateFingerprint(path string, backupName string) (*FileFingerprint, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	fp := &FileFingerprint{
		Path:       path,
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		BackupedIn: backupName,
	}

	// Calculate checksum for files (not directories)
	if !info.IsDir() && info.Size() > 0 {
		checksum, err := calculateChecksum(path)
		if err != nil {
			return nil, err
		}
		fp.Checksum = checksum
	}

	return fp, nil
}

// calculateChecksum calculates SHA256 checksum of a file
func calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// MarkFullBackup updates the last full backup timestamp and name
func (idx *BackupIndex) MarkFullBackup(timestamp time.Time, backupName string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.LastFullBackup = timestamp
	idx.LastFullBackupName = backupName
	idx.LastBackup = timestamp
}

// GetLastFullBackupName returns the name of the last full backup
func (idx *BackupIndex) GetLastFullBackupName() string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.LastFullBackupName
}

// MarkIncrementalBackup updates the last backup timestamp
func (idx *BackupIndex) MarkIncrementalBackup(timestamp time.Time) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.LastBackup = timestamp
}

// NeedFullBackup checks if a full backup is needed
func (idx *BackupIndex) NeedFullBackup(interval time.Duration) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if idx.LastFullBackup.IsZero() {
		return true
	}

	return time.Since(idx.LastFullBackup) >= interval
}

// GetFileCount returns the number of files in the index
func (idx *BackupIndex) GetFileCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.Files)
}

// GetTotalSize returns the total size of all indexed files
func (idx *BackupIndex) GetTotalSize() int64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var total int64
	for _, fp := range idx.Files {
		total += fp.Size
	}

	return total
}

// RemoveFile removes a file from the index
func (idx *BackupIndex) RemoveFile(path string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.Files, path)
}

// GetBackupedFiles returns files backed up in a specific backup
func (idx *BackupIndex) GetBackupedFiles(backupName string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var files []string
	for path, fp := range idx.Files {
		if fp.BackupedIn == backupName {
			files = append(files, path)
		}
	}

	return files
}

// UpdateFromBackup updates the index with files from a backup
func (idx *BackupIndex) UpdateFromBackup(backupName string, files []string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, path := range files {
		fp, err := CreateFingerprint(path, backupName)
		if err != nil {
			continue // Skip files we can't fingerprint
		}

		idx.Files[path] = fp
	}

	return nil
}

// GetDefaultIndexPath returns the default path for the index file
func GetDefaultIndexPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".stash-index.json")
}

// Package incremental provides incremental backup and restore functionality.
// This file contains the restore chain resolution logic for incremental backups.
package incremental

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/harshpatel5940/stash/internal/metadata"
)

// RestoreChain represents a chain of backups needed for restore
type RestoreChain struct {
	FullBackup         string   // Path to full backup
	IncrementalBackups []string // Paths to incremental backups in order
}

// BackupRegistryEntry stores information about a backup for chain resolution
type BackupRegistryEntry struct {
	BackupName string    `json:"backup_name"`
	BackupPath string    `json:"backup_path"`
	BackupType string    `json:"backup_type"` // "full" or "incremental"
	BaseBackup string    `json:"base_backup,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// BackupRegistry stores metadata about all backups for chain resolution
type BackupRegistry struct {
	Version string                          `json:"version"`
	Backups map[string]*BackupRegistryEntry `json:"backups"`
	mu      sync.RWMutex
}

// GetRegistryPath returns the path to the backup registry file
func GetRegistryPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".stash-registry.json")
}

// LoadRegistry loads the backup registry from disk
func LoadRegistry() (*BackupRegistry, error) {
	registryPath := GetRegistryPath()

	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &BackupRegistry{
				Version: "1.0",
				Backups: make(map[string]*BackupRegistryEntry),
			}, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var registry BackupRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	if registry.Backups == nil {
		registry.Backups = make(map[string]*BackupRegistryEntry)
	}

	return &registry, nil
}

// Save saves the backup registry to disk
func (r *BackupRegistry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	registryPath := GetRegistryPath()
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// RegisterBackup adds a backup to the registry
func (r *BackupRegistry) RegisterBackup(name, path, backupType, baseBackup string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Backups[name] = &BackupRegistryEntry{
		BackupName: name,
		BackupPath: path,
		BackupType: backupType,
		BaseBackup: baseBackup,
		Timestamp:  time.Now(),
	}
}

// GetBackup retrieves a backup entry from the registry
func (r *BackupRegistry) GetBackup(name string) (*BackupRegistryEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.Backups[name]
	return entry, exists
}

// RemoveBackup removes a backup from the registry
func (r *BackupRegistry) RemoveBackup(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.Backups, name)
}

// GetRestoreChain determines the chain of backups needed to restore
func GetRestoreChain(backupPath string) (*RestoreChain, error) {
	// Extract metadata from the backup
	meta, err := extractMetadata(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// If it's a full backup, just return it
	if meta.IsFull() {
		return &RestoreChain{
			FullBackup:         backupPath,
			IncrementalBackups: []string{},
		}, nil
	}

	// It's an incremental backup, need to find the chain
	chain := &RestoreChain{
		IncrementalBackups: []string{backupPath},
	}

	// Find the base backup
	backupDir := filepath.Dir(backupPath)
	baseBackupName := meta.BaseBackup

	if baseBackupName == "" {
		return nil, fmt.Errorf("incremental backup has no base backup reference")
	}

	// Look for the base backup in the same directory
	baseBackupPath := findBackupFile(backupDir, baseBackupName)
	if baseBackupPath == "" {
		return nil, fmt.Errorf("base backup not found: %s", baseBackupName)
	}

	// Recursively get the chain for the base backup
	// (in case the base is also incremental)
	baseChain, err := GetRestoreChain(baseBackupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get base backup chain: %w", err)
	}

	// Combine the chains
	chain.FullBackup = baseChain.FullBackup
	chain.IncrementalBackups = append(baseChain.IncrementalBackups, backupPath)

	return chain, nil
}

// findBackupFile finds a backup file by name in a directory
func findBackupFile(dir, name string) string {
	// Try with .tar.gz.age extension
	path := filepath.Join(dir, name+".tar.gz.age")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Try with .tar.gz extension
	path = filepath.Join(dir, name+".tar.gz")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Try exact name
	path = filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}

// extractMetadata extracts metadata from a backup file
func extractMetadata(backupPath string) (*metadata.Metadata, error) {
	// First, try to get metadata from the registry (works for encrypted backups)
	backupName := extractBackupName(backupPath)
	registry, err := LoadRegistry()
	if err == nil {
		if entry, exists := registry.GetBackup(backupName); exists {
			meta := metadata.New()
			meta.SetBackupType(entry.BackupType)
			meta.SetBaseBackup(entry.BaseBackup)
			return meta, nil
		}
	}

	// Check if it's encrypted - if so and not in registry, assume full backup
	isEncrypted := strings.HasSuffix(backupPath, ".age")
	if isEncrypted {
		// No registry entry found - assume full backup for safety
		// This maintains backwards compatibility with backups made before
		// the registry was implemented
		meta := metadata.New()
		meta.SetBackupType("full")
		return meta, nil
	}

	// For unencrypted backups, we could try to extract and read metadata.json
	// but for now, default to full backup
	return &metadata.Metadata{
		BackupType: "full", // Default to full for safety
	}, nil
}

// extractBackupName extracts the backup name from a path
func extractBackupName(backupPath string) string {
	name := filepath.Base(backupPath)
	// Remove extensions
	name = strings.TrimSuffix(name, ".age")
	name = strings.TrimSuffix(name, ".tar.gz")
	return name
}

// ValidateChain validates that all backups in the chain exist and are accessible
func (rc *RestoreChain) Validate() error {
	// Check full backup exists
	if _, err := os.Stat(rc.FullBackup); err != nil {
		return fmt.Errorf("full backup not found: %s", rc.FullBackup)
	}

	// Check all incremental backups exist
	for _, backup := range rc.IncrementalBackups {
		if _, err := os.Stat(backup); err != nil {
			return fmt.Errorf("incremental backup not found: %s", backup)
		}
	}

	return nil
}

// GetTotalBackups returns the total number of backups in the chain
func (rc *RestoreChain) GetTotalBackups() int {
	return 1 + len(rc.IncrementalBackups) // 1 full + N incrementals
}

// GetBackupsInOrder returns all backups in the order they should be restored
func (rc *RestoreChain) GetBackupsInOrder() []string {
	backups := []string{rc.FullBackup}
	backups = append(backups, rc.IncrementalBackups...)
	return backups
}

// Summary returns a human-readable summary of the restore chain
func (rc *RestoreChain) Summary() string {
	if len(rc.IncrementalBackups) == 0 {
		return "Full backup restore (1 file)"
	}

	return fmt.Sprintf("Restore chain: 1 full backup + %d incremental backup(s)", len(rc.IncrementalBackups))
}

// LoadMetadataFromExtractedBackup loads metadata from an already extracted backup directory
func LoadMetadataFromExtractedBackup(extractedDir string) (*metadata.Metadata, error) {
	metadataPath := filepath.Join(extractedDir, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata.json: %w", err)
	}

	var meta metadata.Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata.json: %w", err)
	}

	return &meta, nil
}

// GetIncrementalFiles returns files that should be restored from an incremental backup
func GetIncrementalFiles(meta *metadata.Metadata) []string {
	if !meta.IsIncremental() {
		return nil
	}

	// Return all files in the incremental backup
	files := make([]string, 0, len(meta.Files))
	for _, file := range meta.Files {
		files = append(files, file.BackupPath)
	}

	return files
}

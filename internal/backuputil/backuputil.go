// Package backuputil provides utilities for working with backup files.
// It handles extracting metadata from both encrypted (.age) and
// unencrypted (.tar.gz) backup archives.
package backuputil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harshpatel5940/stash/internal/archiver"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/harshpatel5940/stash/internal/metadata"
)

// ExtractMetadata extracts metadata.json from a backup file.
// Handles both encrypted (.age) and unencrypted (.tar.gz) backups.
// If keyPath is empty, it defaults to ~/.stash.key for encrypted backups.
func ExtractMetadata(backupPath, keyPath string) (*metadata.Metadata, error) {
	// Check if backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		return nil, fmt.Errorf("backup file not found: %w", err)
	}

	// Determine if backup is encrypted
	isEncrypted := strings.HasSuffix(backupPath, ".age")

	// Create temp directory for extraction
	tempDir, err := os.MkdirTemp("", "stash-metadata-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var archivePath string

	if isEncrypted {
		// Decrypt the backup first
		if keyPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			keyPath = filepath.Join(homeDir, ".stash.key")
		}

		// Check if key exists
		if _, err := os.Stat(keyPath); err != nil {
			return nil, fmt.Errorf("encryption key not found at %s: %w", keyPath, err)
		}

		// Decrypt to temp file
		decryptedPath := filepath.Join(tempDir, "backup.tar.gz")
		enc := crypto.NewEncryptor(keyPath)
		if err := enc.Decrypt(backupPath, decryptedPath); err != nil {
			return nil, fmt.Errorf("failed to decrypt backup: %w", err)
		}
		archivePath = decryptedPath
	} else {
		archivePath = backupPath
	}

	// Extract the archive
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extraction directory: %w", err)
	}

	arch := archiver.NewArchiver()
	if err := arch.Extract(archivePath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract backup: %w", err)
	}

	// Look for metadata.json in the extracted contents
	metadataPath := filepath.Join(extractDir, "metadata.json")
	if _, err := os.Stat(metadataPath); err != nil {
		return nil, fmt.Errorf("metadata.json not found in backup archive")
	}

	// Load and return the metadata
	return metadata.Load(metadataPath)
}

// IsEncrypted returns true if the backup file is encrypted (has .age extension)
func IsEncrypted(backupPath string) bool {
	return strings.HasSuffix(backupPath, ".age")
}

// GetBackupBaseName returns the backup filename without encryption extension
func GetBackupBaseName(backupPath string) string {
	name := filepath.Base(backupPath)
	if strings.HasSuffix(name, ".tar.gz.age") {
		return strings.TrimSuffix(name, ".age")
	}
	return name
}

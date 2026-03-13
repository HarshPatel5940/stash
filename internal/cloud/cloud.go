// Package cloud provides cloud storage integration for backup synchronization.
// It supports S3-compatible storage providers including AWS S3, Backblaze B2,
// MinIO, DigitalOcean Spaces, and Cloudflare R2.
package cloud

import (
	"fmt"
	"time"
)

// Provider defines the interface for cloud storage providers
type Provider interface {
	// Upload uploads a local file to the remote storage
	Upload(localPath, remotePath string) error

	// Download downloads a remote file to a local path
	Download(remotePath, localPath string) error

	// List lists all backups in the remote storage
	List(prefix string) ([]BackupEntry, error)

	// Delete deletes a file from remote storage
	Delete(remotePath string) error

	// Exists checks if a file exists in remote storage
	Exists(remotePath string) (bool, error)

	// GetName returns the provider name
	GetName() string
}

// BackupEntry represents a backup file in cloud storage
type BackupEntry struct {
	Name         string    // File name
	Key          string    // Full path/key in storage
	Size         int64     // File size in bytes
	LastModified time.Time // Last modification time
}

// Config holds cloud storage configuration
type Config struct {
	Provider string `yaml:"provider"` // "s3" (also works for B2, MinIO, R2, etc.)
	Bucket   string `yaml:"bucket"`
	Region   string `yaml:"region"`
	Endpoint string `yaml:"endpoint,omitempty"` // Custom endpoint for S3-compatible services
	Prefix   string `yaml:"prefix,omitempty"`   // Path prefix for backups
}

// NewProvider creates a new cloud storage provider based on configuration
func NewProvider(cfg Config) (Provider, error) {
	switch cfg.Provider {
	case "s3", "":
		return NewS3Provider(cfg)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", cfg.Provider)
	}
}

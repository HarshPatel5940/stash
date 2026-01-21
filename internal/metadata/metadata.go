// Package metadata handles backup manifest creation and management.
// It tracks all files included in a backup along with their properties
// (size, permissions, checksums), package manager counts, timing statistics,
// and backup type information for incremental backup support.
//
// The metadata is stored as metadata.json within each backup archive.
package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type FileInfo struct {
	OriginalPath string      `json:"original_path"`
	BackupPath   string      `json:"backup_path"`
	Size         int64       `json:"size"`
	Mode         os.FileMode `json:"mode"`
	ModTime      time.Time   `json:"mod_time"`
	Checksum     string      `json:"checksum"`
	IsDir        bool        `json:"is_dir"`
}

type CategoryTiming struct {
	Name      string        `json:"name"`
	FileCount int           `json:"file_count"`
	TotalSize int64         `json:"total_size"`
	Duration  time.Duration `json:"duration"`
}

type Metadata struct {
	Version          string                     `json:"version"`
	Timestamp        time.Time                  `json:"timestamp"`
	Hostname         string                     `json:"hostname"`
	Username         string                     `json:"username"`
	Files            []FileInfo                 `json:"files"`
	PackageCounts    map[string]int             `json:"package_counts"`
	BackupSize       int64                      `json:"backup_size"`
	CompressedSize   int64                      `json:"compressed_size,omitempty"`
	EncryptedSize    int64                      `json:"encrypted_size,omitempty"`
	TotalDuration    time.Duration              `json:"total_duration,omitempty"`
	Categories       map[string]*CategoryTiming `json:"categories,omitempty"`
	BackupType       string                     `json:"backup_type,omitempty"`        // "full" or "incremental"
	BaseBackup       string                     `json:"base_backup,omitempty"`        // reference to full backup
	ChangedFilesOnly bool                       `json:"changed_files_only,omitempty"` // true for incremental
	mu               sync.Mutex
}

func New() *Metadata {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")

	return &Metadata{
		Version:       "1.1.0",
		Timestamp:     time.Now(),
		Hostname:      hostname,
		Username:      username,
		Files:         []FileInfo{},
		PackageCounts: make(map[string]int),
		Categories:    make(map[string]*CategoryTiming),
		BackupSize:    0,
	}
}

func (m *Metadata) AddFile(originalPath, backupPath string) error {
	info, err := os.Stat(originalPath)
	if err != nil {
		return err
	}

	fileInfo := FileInfo{
		OriginalPath: originalPath,
		BackupPath:   backupPath,
		Size:         info.Size(),
		Mode:         info.Mode(),
		ModTime:      info.ModTime(),
		IsDir:        info.IsDir(),
	}

	if !info.IsDir() {
		checksum, err := calculateChecksum(originalPath)
		if err != nil {
			return err
		}
		fileInfo.Checksum = checksum
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !info.IsDir() {
		m.BackupSize += info.Size()
	}
	m.Files = append(m.Files, fileInfo)
	return nil
}

func (m *Metadata) AddFileInfo(fileInfo FileInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Files = append(m.Files, fileInfo)
	m.BackupSize += fileInfo.Size
}

func (m *Metadata) SetPackageCount(packageType string, count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PackageCounts[packageType] = count
}

func (m *Metadata) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Load(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

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

func (m *Metadata) Summary() string {
	fileCount := 0
	dirCount := 0

	categorySizes := make(map[string]int64)

	for _, f := range m.Files {
		if f.IsDir {
			dirCount++
		} else {
			fileCount++
		}

		if f.Size > 0 {
			parts := strings.Split(f.BackupPath, string(filepath.Separator))
			if len(parts) > 0 {
				category := parts[0]
				categorySizes[category] += f.Size
			}
		}
	}

	summary := fmt.Sprintf("Backup created: %s\n", m.Timestamp.Format("2006-01-02 15:04:05"))
	summary += fmt.Sprintf("Hostname: %s\n", m.Hostname)
	summary += fmt.Sprintf("Username: %s\n", m.Username)
	summary += fmt.Sprintf("Files: %d, Directories: %d\n", fileCount, dirCount)
	summary += fmt.Sprintf("Total size: %s\n", FormatSize(m.BackupSize))

	summary += "\nSize Breakdown:\n"

	type category struct {
		Name string
		Size int64
	}
	var categories []category
	for name, size := range categorySizes {
		categories = append(categories, category{name, size})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Size > categories[j].Size
	})

	for _, c := range categories {
		summary += fmt.Sprintf("  %s: %s\n", c.Name, FormatSize(c.Size))
	}

	if len(m.PackageCounts) > 0 {
		summary += "\nPackages:\n"
		for pkg, count := range m.PackageCounts {
			summary += fmt.Sprintf("  %s: %d\n", pkg, count)
		}
	}

	return summary
}

func FormatSize(bytes int64) string {
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

// AddCategoryTiming records timing and statistics for a backup category
func (m *Metadata) AddCategoryTiming(name string, fileCount int, totalSize int64, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Categories[name] = &CategoryTiming{
		Name:      name,
		FileCount: fileCount,
		TotalSize: totalSize,
		Duration:  duration,
	}
}

// SetCompressedSize records the compressed archive size
func (m *Metadata) SetCompressedSize(size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CompressedSize = size
}

// SetEncryptedSize records the final encrypted size
func (m *Metadata) SetEncryptedSize(size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EncryptedSize = size
}

// SetTotalDuration records the total backup duration
func (m *Metadata) SetTotalDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalDuration = duration
}

// GetLargestFiles returns the N largest files in the backup
func (m *Metadata) GetLargestFiles(n int) []FileInfo {
	files := make([]FileInfo, len(m.Files))
	copy(files, m.Files)

	// Sort by size descending
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})

	if len(files) > n {
		return files[:n]
	}
	return files
}

// GetCompressionRatio returns the compression ratio as a percentage
func (m *Metadata) GetCompressionRatio() float64 {
	if m.BackupSize == 0 {
		return 0
	}
	return (1.0 - float64(m.CompressedSize)/float64(m.BackupSize)) * 100
}

// GetFileCount returns total number of non-directory files
func (m *Metadata) GetFileCount() int {
	count := 0
	for _, f := range m.Files {
		if !f.IsDir {
			count++
		}
	}
	return count
}

// GetCategoryStats returns statistics organized by category
func (m *Metadata) GetCategoryStats() map[string]map[string]interface{} {
	stats := make(map[string]map[string]interface{})

	for name, cat := range m.Categories {
		stats[name] = map[string]interface{}{
			"files":    cat.FileCount,
			"size":     cat.TotalSize,
			"duration": formatDuration(cat.Duration),
		}
	}

	return stats
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(time.Second).String()
}

// SetBackupType sets the backup type (full or incremental)
func (m *Metadata) SetBackupType(backupType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BackupType = backupType
}

// SetBaseBackup sets the reference to the base full backup
func (m *Metadata) SetBaseBackup(baseBackup string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BaseBackup = baseBackup
}

// SetChangedFilesOnly marks this as an incremental backup
func (m *Metadata) SetChangedFilesOnly(changedOnly bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ChangedFilesOnly = changedOnly
}

// IsIncremental returns true if this is an incremental backup
func (m *Metadata) IsIncremental() bool {
	return m.BackupType == "incremental"
}

// IsFull returns true if this is a full backup
func (m *Metadata) IsFull() bool {
	return m.BackupType == "full" || m.BackupType == ""
}

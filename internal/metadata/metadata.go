package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
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

type Metadata struct {
	Version       string         `json:"version"`
	Timestamp     time.Time      `json:"timestamp"`
	Hostname      string         `json:"hostname"`
	Username      string         `json:"username"`
	Files         []FileInfo     `json:"files"`
	PackageCounts map[string]int `json:"package_counts"`
	BackupSize    int64          `json:"backup_size"`
}

func New() *Metadata {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")

	return &Metadata{
		Version:       "1.0.0",
		Timestamp:     time.Now(),
		Hostname:      hostname,
		Username:      username,
		Files:         []FileInfo{},
		PackageCounts: make(map[string]int),
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

	// Calculate checksum for files (not directories)
	if !info.IsDir() {
		checksum, err := calculateChecksum(originalPath)
		if err != nil {
			return err
		}
		fileInfo.Checksum = checksum
		m.BackupSize += info.Size()
	}

	m.Files = append(m.Files, fileInfo)
	return nil
}

func (m *Metadata) SetPackageCount(packageType string, count int) {
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
	for _, f := range m.Files {
		if f.IsDir {
			dirCount++
		} else {
			fileCount++
		}
	}

	summary := fmt.Sprintf("Backup created: %s\n", m.Timestamp.Format("2006-01-02 15:04:05"))
	summary += fmt.Sprintf("Hostname: %s\n", m.Hostname)
	summary += fmt.Sprintf("Username: %s\n", m.Username)
	summary += fmt.Sprintf("Files: %d, Directories: %d\n", fileCount, dirCount)
	summary += fmt.Sprintf("Total size: %.2f MB\n", float64(m.BackupSize)/(1024*1024))

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

// Package stats provides comprehensive backup statistics collection and reporting.
// It tracks file counts, sizes, compression ratios, processing speeds, and
// per-category breakdowns to give users insight into their backup operations.
package stats

import (
	"sort"
	"time"
)

// CategoryStats holds statistics for a backup category
type CategoryStats struct {
	Name        string
	FileCount   int
	TotalSize   int64
	LargestFile string
	TimeSpent   time.Duration
}

// FileStats holds information about a single file
type FileStats struct {
	Path string
	Size int64
}

// BackupStats aggregates all backup statistics
type BackupStats struct {
	Categories        map[string]*CategoryStats
	TotalFiles        int
	OriginalSize      int64
	CompressedSize    int64
	EncryptedSize     int64
	CompressionRatio  float64
	TotalTime         time.Duration
	LargestFiles      []FileStats
	FileTypeBreakdown map[string]int
	StartTime         time.Time
	EndTime           time.Time
}

// New creates a new BackupStats instance
func New() *BackupStats {
	return &BackupStats{
		Categories:        make(map[string]*CategoryStats),
		FileTypeBreakdown: make(map[string]int),
		StartTime:         time.Now(),
	}
}

// AddCategory adds statistics for a category
func (bs *BackupStats) AddCategory(name string, fileCount int, totalSize int64, largestFile string, timeSpent time.Duration) {
	bs.Categories[name] = &CategoryStats{
		Name:        name,
		FileCount:   fileCount,
		TotalSize:   totalSize,
		LargestFile: largestFile,
		TimeSpent:   timeSpent,
	}

	bs.TotalFiles += fileCount
	bs.OriginalSize += totalSize
}

// UpdateCategory updates an existing category or creates it if it doesn't exist
func (bs *BackupStats) UpdateCategory(name string, fileCount int, totalSize int64, timeSpent time.Duration) {
	if cat, exists := bs.Categories[name]; exists {
		cat.FileCount += fileCount
		cat.TotalSize += totalSize
		cat.TimeSpent += timeSpent
	} else {
		bs.Categories[name] = &CategoryStats{
			Name:      name,
			FileCount: fileCount,
			TotalSize: totalSize,
			TimeSpent: timeSpent,
		}
	}

	bs.TotalFiles += fileCount
	bs.OriginalSize += totalSize
}

// AddFile adds a file to the statistics
func (bs *BackupStats) AddFile(path string, size int64) {
	bs.LargestFiles = append(bs.LargestFiles, FileStats{
		Path: path,
		Size: size,
	})
}

// Finalize calculates final statistics
func (bs *BackupStats) Finalize(compressedSize, encryptedSize int64) {
	bs.EndTime = time.Now()
	bs.TotalTime = bs.EndTime.Sub(bs.StartTime)
	bs.CompressedSize = compressedSize
	bs.EncryptedSize = encryptedSize

	if bs.OriginalSize > 0 {
		bs.CompressionRatio = float64(bs.CompressedSize) / float64(bs.OriginalSize)
	}

	// Sort largest files by size
	sort.Slice(bs.LargestFiles, func(i, j int) bool {
		return bs.LargestFiles[i].Size > bs.LargestFiles[j].Size
	})

	// Keep only top 10 largest files
	if len(bs.LargestFiles) > 10 {
		bs.LargestFiles = bs.LargestFiles[:10]
	}
}

// GetCategoryStats returns statistics for a specific category
func (bs *BackupStats) GetCategoryStats(name string) *CategoryStats {
	return bs.Categories[name]
}

// GetCompressionRatio returns the compression ratio as a percentage
func (bs *BackupStats) GetCompressionRatio() float64 {
	if bs.OriginalSize == 0 {
		return 0
	}
	return (1.0 - bs.CompressionRatio) * 100
}

// GetSizeReduction returns the size reduction in bytes
func (bs *BackupStats) GetSizeReduction() int64 {
	return bs.OriginalSize - bs.CompressedSize
}

// GetAverageFileSize returns the average file size
func (bs *BackupStats) GetAverageFileSize() int64 {
	if bs.TotalFiles == 0 {
		return 0
	}
	return bs.OriginalSize / int64(bs.TotalFiles)
}

// GetProcessingSpeed returns files processed per second
func (bs *BackupStats) GetProcessingSpeed() float64 {
	if bs.TotalTime.Seconds() == 0 {
		return 0
	}
	return float64(bs.TotalFiles) / bs.TotalTime.Seconds()
}

// GetBytesPerSecond returns bytes processed per second
func (bs *BackupStats) GetBytesPerSecond() float64 {
	if bs.TotalTime.Seconds() == 0 {
		return 0
	}
	return float64(bs.OriginalSize) / bs.TotalTime.Seconds()
}

// ToMap converts statistics to a map for display
func (bs *BackupStats) ToMap() map[string]interface{} {
	categories := make(map[string]map[string]interface{})
	for name, cat := range bs.Categories {
		categories[name] = map[string]interface{}{
			"files":    cat.FileCount,
			"size":     cat.TotalSize,
			"duration": formatDuration(cat.TimeSpent),
		}
	}

	largestFiles := make([]map[string]interface{}, 0, len(bs.LargestFiles))
	for _, file := range bs.LargestFiles {
		largestFiles = append(largestFiles, map[string]interface{}{
			"path": file.Path,
			"size": file.Size,
		})
	}

	return map[string]interface{}{
		"categories":        categories,
		"total_files":       bs.TotalFiles,
		"original_size":     bs.OriginalSize,
		"compressed_size":   bs.CompressedSize,
		"encrypted_size":    bs.EncryptedSize,
		"compression_ratio": bs.GetCompressionRatio(),
		"size_reduction":    bs.GetSizeReduction(),
		"total_time":        formatDuration(bs.TotalTime),
		"largest_files":     largestFiles,
		"processing_speed":  bs.GetProcessingSpeed(),
		"bytes_per_second":  bs.GetBytesPerSecond(),
	}
}

// formatDuration formats a duration into human-readable form
func formatDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)

	if d < time.Second {
		return d.String()
	}

	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return formatWithUnits(int(h), "h", int(m), "m", int(s), "s")
	} else if m > 0 {
		return formatWithUnits(int(m), "m", int(s), "s", 0, "")
	}
	return formatWithUnits(int(s), "s", 0, "", 0, "")
}

func formatWithUnits(v1 int, u1 string, v2 int, u2 string, v3 int, u3 string) string {
	result := ""
	if v1 > 0 {
		result += string(rune('0'+v1/10)) + string(rune('0'+v1%10)) + u1
	}
	if v2 > 0 {
		if result != "" {
			result += " "
		}
		result += string(rune('0'+v2/10)) + string(rune('0'+v2%10)) + u2
	}
	if v3 > 0 {
		if result != "" {
			result += " "
		}
		result += string(rune('0'+v3/10)) + string(rune('0'+v3%10)) + u3
	}
	if result == "" {
		return "0s"
	}
	return result
}

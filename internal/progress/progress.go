// Package progress provides real-time progress tracking for backup operations.
// It supports tracking multiple categories (dotfiles, secrets, browsers, etc.)
// simultaneously with thread-safe concurrent updates and ETA calculation.
//
// The package includes both verbose text output and visual progress bars.
package progress

import (
	"fmt"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// CategoryProgress tracks progress for a specific backup category
type CategoryProgress struct {
	Name       string
	FilesTotal int
	FilesDone  int
	BytesTotal int64
	BytesDone  int64
	StartTime  time.Time
	mu         sync.Mutex
}

// ProgressTracker manages progress tracking across multiple categories
type ProgressTracker struct {
	categories map[string]*CategoryProgress
	startTime  time.Time
	mu         sync.RWMutex
	verbose    bool
	bar        *progressbar.ProgressBar
}

// New creates a new ProgressTracker
func New(verbose bool) *ProgressTracker {
	return &ProgressTracker{
		categories: make(map[string]*CategoryProgress),
		startTime:  time.Now(),
		verbose:    verbose,
	}
}

// AddCategory adds a new category to track
func (pt *ProgressTracker) AddCategory(name string, filesTotal int, bytesTotal int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.categories[name] = &CategoryProgress{
		Name:       name,
		FilesTotal: filesTotal,
		FilesDone:  0,
		BytesTotal: bytesTotal,
		BytesDone:  0,
		StartTime:  time.Now(),
	}
}

// UpdateCategory updates progress for a category
func (pt *ProgressTracker) UpdateCategory(name string, filesDone int, bytesDone int64) {
	pt.mu.RLock()
	cat, exists := pt.categories[name]
	pt.mu.RUnlock()

	if !exists {
		return
	}

	cat.mu.Lock()
	cat.FilesDone = filesDone
	cat.BytesDone = bytesDone
	cat.mu.Unlock()
}

// IncrementCategory increments progress for a category
func (pt *ProgressTracker) IncrementCategory(name string, files int, bytes int64) {
	pt.mu.RLock()
	cat, exists := pt.categories[name]
	pt.mu.RUnlock()

	if !exists {
		return
	}

	cat.mu.Lock()
	cat.FilesDone += files
	cat.BytesDone += bytes
	cat.mu.Unlock()
}

// CompleteCategory marks a category as complete
func (pt *ProgressTracker) CompleteCategory(name string) {
	pt.mu.RLock()
	cat, exists := pt.categories[name]
	pt.mu.RUnlock()

	if !exists {
		return
	}

	cat.mu.Lock()
	cat.FilesDone = cat.FilesTotal
	cat.BytesDone = cat.BytesTotal
	cat.mu.Unlock()
}

// GetCategoryProgress returns progress info for a category
func (pt *ProgressTracker) GetCategoryProgress(name string) (filesDone, filesTotal int, bytesDone, bytesTotal int64, duration time.Duration) {
	pt.mu.RLock()
	cat, exists := pt.categories[name]
	pt.mu.RUnlock()

	if !exists {
		return 0, 0, 0, 0, 0
	}

	cat.mu.Lock()
	defer cat.mu.Unlock()

	return cat.FilesDone, cat.FilesTotal, cat.BytesDone, cat.BytesTotal, time.Since(cat.StartTime)
}

// GetTotalProgress returns overall progress across all categories
func (pt *ProgressTracker) GetTotalProgress() (filesDone, filesTotal int, bytesDone, bytesTotal int64) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	for _, cat := range pt.categories {
		cat.mu.Lock()
		filesDone += cat.FilesDone
		filesTotal += cat.FilesTotal
		bytesDone += cat.BytesDone
		bytesTotal += cat.BytesTotal
		cat.mu.Unlock()
	}

	return
}

// GetSpeed calculates current processing speed
func (pt *ProgressTracker) GetSpeed() (filesPerSec float64, bytesPerSec float64) {
	filesDone, _, bytesDone, _ := pt.GetTotalProgress()
	elapsed := time.Since(pt.startTime).Seconds()

	if elapsed > 0 {
		filesPerSec = float64(filesDone) / elapsed
		bytesPerSec = float64(bytesDone) / elapsed
	}

	return
}

// GetETA estimates time remaining
func (pt *ProgressTracker) GetETA() time.Duration {
	filesDone, filesTotal, _, _ := pt.GetTotalProgress()

	if filesDone == 0 || filesTotal == 0 {
		return 0
	}

	elapsed := time.Since(pt.startTime)
	percentComplete := float64(filesDone) / float64(filesTotal)

	if percentComplete > 0 {
		totalEstimated := time.Duration(float64(elapsed) / percentComplete)
		return totalEstimated - elapsed
	}

	return 0
}

// GetElapsed returns elapsed time since start
func (pt *ProgressTracker) GetElapsed() time.Duration {
	return time.Since(pt.startTime)
}

// PrintProgress prints current progress to stdout
func (pt *ProgressTracker) PrintProgress(categoryName string) {
	if !pt.verbose {
		return
	}

	filesDone, filesTotal, bytesDone, _, duration := pt.GetCategoryProgress(categoryName)

	fmt.Printf("  %s: %d/%d files (%.2f MB) - %s\n",
		categoryName,
		filesDone,
		filesTotal,
		float64(bytesDone)/(1024*1024),
		duration.Round(time.Millisecond),
	)
}

// StartProgressBar initializes a progress bar for total progress
func (pt *ProgressTracker) StartProgressBar(bytesTotal int64) {
	if pt.verbose {
		return // Don't show progress bar in verbose mode
	}

	pt.bar = progressbar.NewOptions64(
		bytesTotal,
		progressbar.OptionSetDescription("Backing up"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
	)
}

// UpdateProgressBar updates the progress bar
func (pt *ProgressTracker) UpdateProgressBar(current int64) {
	if pt.bar != nil {
		pt.bar.Set64(current)
	}
}

// FinishProgressBar completes the progress bar
func (pt *ProgressTracker) FinishProgressBar() {
	if pt.bar != nil {
		pt.bar.Finish()
		fmt.Println() // Add newline after progress bar
	}
}

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes int64) string {
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

// FormatDuration formats duration into human-readable format
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

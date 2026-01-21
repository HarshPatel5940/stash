// Package ui provides terminal user interface utilities for stash.
// It includes colored output, progress bars, spinners, and formatted
// display of statistics, errors, and file changes.
//
// All output functions are designed to be user-friendly and informative.
package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

var (
	Success = color.New(color.FgGreen).SprintFunc()
	Error   = color.New(color.FgRed).SprintFunc()
	Warning = color.New(color.FgYellow).SprintFunc()
	Info    = color.New(color.FgCyan).SprintFunc()
	Bold    = color.New(color.Bold).SprintFunc()

	IconSuccess = "✓"
	IconError   = "✗"
	IconWarning = "⚠️"
	IconInfo    = "ℹ️"
)

func PrintSuccess(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", Success(IconSuccess), msg)
}

func PrintError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", Error(IconError), msg)
}

func PrintWarning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s  %s\n", Warning(IconWarning), msg)
}

func PrintInfo(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s  %s\n", Info(IconInfo), msg)
}

func PrintHeader(text string) {
	fmt.Println(Bold(text))
}

func PrintSectionHeader(emoji, text string) {
	fmt.Printf("\n%s %s\n", emoji, Bold(text))
}

func NewProgressBar(max int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stdout, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}

func NewSimpleProgressBar(max int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stdout, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}

type Spinner struct {
	writer  io.Writer
	message string
	active  bool
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		writer:  os.Stdout,
		message: message,
		active:  false,
	}
}

func (s *Spinner) Start() {
	s.active = true
	fmt.Fprintf(s.writer, "  %s %s...", Info("⏳"), s.message)
}

func (s *Spinner) Stop() {
	if s.active {
		fmt.Fprintf(s.writer, "\r  %s %s\n", Success(IconSuccess), s.message)
		s.active = false
	}
}

func (s *Spinner) Fail() {
	if s.active {
		fmt.Fprintf(s.writer, "\r  %s %s\n", Error(IconError), s.message)
		s.active = false
	}
}

func (s *Spinner) UpdateMessage(message string) {
	s.message = message
	if s.active {
		fmt.Fprintf(s.writer, "\r  %s %s...", Info("⏳"), s.message)
	}
}

func PrintDivider() {
	fmt.Println(color.New(color.Faint).Sprint("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
}

func PrintBox(title string) {
	width := 50
	fmt.Println("┌" + repeat("─", width-2) + "┐")
	padding := (width - len(title) - 2) / 2
	fmt.Printf("│%s%s%s│\n", repeat(" ", padding), Bold(title), repeat(" ", width-len(title)-padding-2))
	fmt.Println("└" + repeat("─", width-2) + "┘")
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

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
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func PrintSummaryTable(items map[string]string) {
	maxKeyLen := 0
	for key := range items {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}

	PrintDivider()
	for key, value := range items {
		padding := repeat(" ", maxKeyLen-len(key))
		fmt.Printf("  %s:%s %s\n", Bold(key), padding, value)
	}
	PrintDivider()
}

// PrintCategoryProgress prints progress for a single category
func PrintCategoryProgress(name string, filesDone, filesTotal int, bytesDone, bytesTotal int64) {
	percentage := 0.0
	if filesTotal > 0 {
		percentage = (float64(filesDone) / float64(filesTotal)) * 100
	}

	fmt.Printf("  %s %s: %d/%d files (%.1f%%) - %s\n",
		Info("▶"),
		name,
		filesDone,
		filesTotal,
		percentage,
		FormatBytes(bytesDone),
	)
}

// PrintStatistics prints detailed backup statistics
func PrintStatistics(stats map[string]interface{}) {
	fmt.Println()
	PrintSectionHeader("📊", "BACKUP STATISTICS")
	PrintDivider()

	if categories, ok := stats["categories"].(map[string]map[string]interface{}); ok {
		fmt.Println(Bold("  Category Breakdown:"))
		for name, data := range categories {
			files := data["files"].(int)
			size := data["size"].(int64)
			duration := data["duration"].(string)
			fmt.Printf("    %-20s %4d files    %10s    (%s)\n",
				name+":",
				files,
				FormatBytes(size),
				duration,
			)
		}
		fmt.Println()
	}

	if totalFiles, ok := stats["total_files"].(int); ok {
		fmt.Printf("  %s %d\n", Bold("Total Files:"), totalFiles)
	}

	if originalSize, ok := stats["original_size"].(int64); ok {
		fmt.Printf("  %s %s\n", Bold("Original Size:"), FormatBytes(originalSize))
	}

	if compressedSize, ok := stats["compressed_size"].(int64); ok {
		originalSize := stats["original_size"].(int64)
		reduction := 0.0
		if originalSize > 0 {
			reduction = (1.0 - float64(compressedSize)/float64(originalSize)) * 100
		}
		fmt.Printf("  %s %s (%.1f%% reduction)\n",
			Bold("Compressed:"),
			FormatBytes(compressedSize),
			reduction,
		)
	}

	if totalTime, ok := stats["total_time"].(string); ok {
		fmt.Printf("  %s %s\n", Bold("Total Time:"), totalTime)
	}

	if largestFiles, ok := stats["largest_files"].([]map[string]interface{}); ok && len(largestFiles) > 0 {
		fmt.Println()
		fmt.Println(Bold("  Top 5 Largest Files:"))
		for i, file := range largestFiles {
			if i >= 5 {
				break
			}
			path := file["path"].(string)
			size := file["size"].(int64)
			fmt.Printf("    %d. %-50s  %10s\n", i+1, truncatePath(path, 50), FormatBytes(size))
		}
	}

	PrintDivider()
}

// truncatePath truncates a file path to fit within maxLen
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

// PrintErrorWithSolution prints an error with a suggested solution
func PrintErrorWithSolution(problem, solution, alternative string) {
	fmt.Println()
	PrintError("Backup failed: %s", problem)
	fmt.Println()
	if solution != "" {
		fmt.Printf("📍 %s: %s\n", Bold("Problem"), problem)
		fmt.Printf("🔧 %s: %s\n", Bold("Solution"), solution)
	}
	if alternative != "" {
		fmt.Printf("💡 %s: %s\n", Bold("Alternative"), alternative)
	}
	fmt.Println()
}

// PrintComparisonHeader prints header for backup comparison
func PrintComparisonHeader(oldBackup, newBackup string, oldSize, newSize int64) {
	fmt.Println()
	PrintSectionHeader("📊", "Comparing backups")
	fmt.Printf("  Old: %s (%s)\n", oldBackup, FormatBytes(oldSize))
	fmt.Printf("  New: %s (%s)\n", newBackup, FormatBytes(newSize))
	fmt.Println()
}

// PrintFileChanges prints file changes in a diff
func PrintFileChanges(added, removed, modified, unchanged int, addedSize, removedSize, modifiedSize int64) {
	PrintSectionHeader("📁", "FILE CHANGES")
	if added > 0 {
		fmt.Printf("  %s %d files added     (+%s)\n", Success("+"), added, FormatBytes(addedSize))
	}
	if removed > 0 {
		fmt.Printf("  %s %d files removed   (-%s)\n", Error("-"), removed, FormatBytes(removedSize))
	}
	if modified > 0 {
		sizeChange := modifiedSize
		sign := "+"
		if sizeChange < 0 {
			sign = "-"
			sizeChange = -sizeChange
		}
		fmt.Printf("  %s %d files modified  (%s%s)\n", Warning("~"), modified, sign, FormatBytes(sizeChange))
	}
	if unchanged > 0 {
		fmt.Printf("  %s %d files unchanged\n", Info("="), unchanged)
	}
	fmt.Println()
}

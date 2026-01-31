// Package ui provides terminal user interface utilities for stash.
// It provides styled output using lipgloss, progress bars, spinners,
// and formatted display. Supports both minimal and verbose output modes.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/schollz/progressbar/v3"
)

// Verbose controls whether detailed output is shown
var Verbose = false

// Color palette
var (
	colorSuccess = lipgloss.Color("40")  // Green
	colorError   = lipgloss.Color("196") // Red
	colorWarning = lipgloss.Color("214") // Orange/Yellow
	colorInfo    = lipgloss.Color("75")  // Cyan
	colorDim     = lipgloss.Color("245") // Gray
	colorAccent  = lipgloss.Color("141") // Purple
)

// Styles
var (
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	warningStyle = lipgloss.NewStyle().Foreground(colorWarning)
	infoStyle    = lipgloss.NewStyle().Foreground(colorInfo)
	dimStyle     = lipgloss.NewStyle().Foreground(colorDim)
	accentStyle  = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	boldStyle    = lipgloss.NewStyle().Bold(true)
	pathStyle    = lipgloss.NewStyle().Foreground(colorInfo)
)

// Icons - minimal text-based
var (
	IconSuccess = successStyle.Render("✓")
	IconError   = errorStyle.Render("✗")
	IconWarning = warningStyle.Render("!")
	IconInfo    = infoStyle.Render("*")
)

// Legacy color functions for backwards compatibility
var (
	Success = func(s string) string { return successStyle.Render(s) }
	Error   = func(s string) string { return errorStyle.Render(s) }
	Warning = func(s string) string { return warningStyle.Render(s) }
	Info    = func(s string) string { return infoStyle.Render(s) }
	Bold    = func(s string) string { return boldStyle.Render(s) }
)

// ============================================================================
// Minimal Mode Output (single-line results)
// ============================================================================

// PrintResult prints a single-line result message (for normal mode)
func PrintResult(icon, message string) {
	fmt.Printf("%s %s\n", icon, message)
}

// PrintSuccess prints a success message
func PrintSuccess(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", IconSuccess, msg)
}

// PrintError prints an error message
func PrintError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", IconError, msg)
}

// PrintWarning prints a warning message
func PrintWarning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", IconWarning, msg)
}

// PrintInfo prints an info message
func PrintInfo(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", IconInfo, msg)
}

// PrintDim prints dimmed/secondary text
func PrintDim(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(dimStyle.Render(msg))
}

// ============================================================================
// Verbose Mode Output (detailed, only when Verbose=true)
// ============================================================================

// PrintVerbose prints only if verbose mode is enabled
func PrintVerbose(format string, a ...interface{}) {
	if Verbose {
		msg := fmt.Sprintf(format, a...)
		fmt.Println(dimStyle.Render("  " + msg))
	}
}

// PrintVerboseSuccess prints a success message only in verbose mode
func PrintVerboseSuccess(format string, a ...interface{}) {
	if Verbose {
		PrintSuccess(format, a...)
	}
}

// PrintVerboseInfo prints an info message only in verbose mode
func PrintVerboseInfo(format string, a ...interface{}) {
	if Verbose {
		PrintInfo(format, a...)
	}
}

// ============================================================================
// Headers and Sections
// ============================================================================

// PrintHeader prints a bold header
func PrintHeader(text string) {
	fmt.Println(boldStyle.Render(text))
}

// PrintSectionHeader prints a section header (only in verbose mode by default)
func PrintSectionHeader(emoji, text string) {
	if Verbose {
		fmt.Printf("\n%s %s\n", emoji, boldStyle.Render(text))
	}
}

// PrintDivider prints a horizontal divider line
func PrintDivider() {
	fmt.Println(dimStyle.Render(strings.Repeat("-", 50)))
}

// ============================================================================
// Tables
// ============================================================================

// TableRow represents a row in a table
type TableRow struct {
	Columns []string
}

// PrintTable prints a formatted table
func PrintTable(headers []string, rows [][]string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, col := range row {
			if i < len(widths) && len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}

	// Print header
	headerLine := ""
	for i, h := range headers {
		headerLine += fmt.Sprintf("%-*s  ", widths[i], h)
	}
	fmt.Println(dimStyle.Render(headerLine))

	// Print rows
	for _, row := range rows {
		line := ""
		for i, col := range row {
			if i < len(widths) {
				line += fmt.Sprintf("%-*s  ", widths[i], col)
			}
		}
		fmt.Println(line)
	}
}

// ============================================================================
// Progress Indicators
// ============================================================================

// Spinner provides a simple spinner for long operations
type Spinner struct {
	writer  io.Writer
	message string
	active  bool
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		writer:  os.Stdout,
		message: message,
		active:  false,
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	s.active = true
	fmt.Fprintf(s.writer, "%s %s...", infoStyle.Render("*"), s.message)
}

// Stop stops the spinner with success
func (s *Spinner) Stop() {
	if s.active {
		fmt.Fprintf(s.writer, "\r%s %s   \n", IconSuccess, s.message)
		s.active = false
	}
}

// Fail stops the spinner with failure
func (s *Spinner) Fail() {
	if s.active {
		fmt.Fprintf(s.writer, "\r%s %s   \n", IconError, s.message)
		s.active = false
	}
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.message = message
	if s.active {
		fmt.Fprintf(s.writer, "\r%s %s...", infoStyle.Render("*"), s.message)
	}
}

// NewProgressBar creates a progress bar
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

// NewSimpleProgressBar creates a simple progress bar
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

// ============================================================================
// Formatting Utilities
// ============================================================================

// FormatBytes formats bytes as human-readable string
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

// FormatPath formats a file path with styling
func FormatPath(path string) string {
	return pathStyle.Render(path)
}

// FormatNumber formats a number with accent styling
func FormatNumber(n int) string {
	return accentStyle.Render(fmt.Sprintf("%d", n))
}

// ============================================================================
// Detailed Output (for verbose mode)
// ============================================================================

// PrintBox prints text in a box
func PrintBox(title string) {
	width := 50
	fmt.Println("+" + strings.Repeat("-", width-2) + "+")
	padding := (width - len(title) - 2) / 2
	fmt.Printf("|%s%s%s|\n", strings.Repeat(" ", padding), boldStyle.Render(title), strings.Repeat(" ", width-len(title)-padding-2))
	fmt.Println("+" + strings.Repeat("-", width-2) + "+")
}

// PrintSummaryTable prints a summary table (verbose mode only)
func PrintSummaryTable(items map[string]string) {
	if !Verbose {
		return
	}

	maxKeyLen := 0
	for key := range items {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}

	PrintDivider()
	for key, value := range items {
		padding := strings.Repeat(" ", maxKeyLen-len(key))
		fmt.Printf("  %s:%s %s\n", boldStyle.Render(key), padding, value)
	}
	PrintDivider()
}

// PrintCategoryProgress prints progress for a category (verbose mode only)
func PrintCategoryProgress(name string, filesDone, filesTotal int, bytesDone, bytesTotal int64) {
	if !Verbose {
		return
	}

	percentage := 0.0
	if filesTotal > 0 {
		percentage = (float64(filesDone) / float64(filesTotal)) * 100
	}

	fmt.Printf("  %s %s: %d/%d files (%.0f%%) - %s\n",
		infoStyle.Render(">"),
		name,
		filesDone,
		filesTotal,
		percentage,
		FormatBytes(bytesDone),
	)
}

// PrintStatistics prints detailed backup statistics (verbose mode only)
func PrintStatistics(stats map[string]interface{}) {
	if !Verbose {
		return
	}

	fmt.Println()
	fmt.Printf("%s %s\n", "*", boldStyle.Render("STATISTICS"))
	PrintDivider()

	if categories, ok := stats["categories"].(map[string]map[string]interface{}); ok {
		fmt.Println(boldStyle.Render("  Categories:"))
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
		fmt.Printf("  %s %d\n", boldStyle.Render("Total Files:"), totalFiles)
	}

	if originalSize, ok := stats["original_size"].(int64); ok {
		fmt.Printf("  %s %s\n", boldStyle.Render("Original Size:"), FormatBytes(originalSize))
	}

	if compressedSize, ok := stats["compressed_size"].(int64); ok {
		originalSize := stats["original_size"].(int64)
		reduction := 0.0
		if originalSize > 0 {
			reduction = (1.0 - float64(compressedSize)/float64(originalSize)) * 100
		}
		fmt.Printf("  %s %s (%.0f%% reduction)\n",
			boldStyle.Render("Compressed:"),
			FormatBytes(compressedSize),
			reduction,
		)
	}

	if totalTime, ok := stats["total_time"].(string); ok {
		fmt.Printf("  %s %s\n", boldStyle.Render("Total Time:"), totalTime)
	}

	PrintDivider()
}

// PrintErrorWithSolution prints an error with a suggested solution
func PrintErrorWithSolution(problem, solution, alternative string) {
	fmt.Println()
	PrintError("%s", problem)
	if Verbose && solution != "" {
		fmt.Printf("  %s: %s\n", boldStyle.Render("Fix"), solution)
	}
	if Verbose && alternative != "" {
		fmt.Printf("  %s: %s\n", boldStyle.Render("Or"), alternative)
	}
}

// PrintComparisonHeader prints header for backup comparison
func PrintComparisonHeader(oldBackup, newBackup string, oldSize, newSize int64) {
	if !Verbose {
		return
	}
	fmt.Println()
	fmt.Printf("%s %s\n", "*", boldStyle.Render("Comparing backups"))
	fmt.Printf("  Old: %s (%s)\n", oldBackup, FormatBytes(oldSize))
	fmt.Printf("  New: %s (%s)\n", newBackup, FormatBytes(newSize))
	fmt.Println()
}

// PrintFileChanges prints file changes summary
func PrintFileChanges(added, removed, modified, unchanged int, addedSize, removedSize, modifiedSize int64) {
	// Minimal output: single line summary
	parts := []string{}
	if added > 0 {
		parts = append(parts, successStyle.Render(fmt.Sprintf("+%d", added)))
	}
	if removed > 0 {
		parts = append(parts, errorStyle.Render(fmt.Sprintf("-%d", removed)))
	}
	if modified > 0 {
		parts = append(parts, warningStyle.Render(fmt.Sprintf("~%d", modified)))
	}
	if len(parts) > 0 {
		fmt.Printf("Changes: %s files\n", strings.Join(parts, " "))
	} else {
		fmt.Println("No changes")
	}

	// Verbose: detailed breakdown
	if Verbose {
		if added > 0 {
			fmt.Printf("  %s %d added (+%s)\n", successStyle.Render("+"), added, FormatBytes(addedSize))
		}
		if removed > 0 {
			fmt.Printf("  %s %d removed (-%s)\n", errorStyle.Render("-"), removed, FormatBytes(removedSize))
		}
		if modified > 0 {
			fmt.Printf("  %s %d modified\n", warningStyle.Render("~"), modified)
		}
		if unchanged > 0 {
			fmt.Printf("  %s %d unchanged\n", dimStyle.Render("="), unchanged)
		}
	}
}

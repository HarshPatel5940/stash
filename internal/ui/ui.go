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

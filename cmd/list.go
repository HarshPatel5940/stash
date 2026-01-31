package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/harshpatel5940/stash/internal/backuputil"
	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	listDetails bool
	listVerbose bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available backups",
	Long: `Lists all backups found in the backup directory.

Shows backup details including:
  - Backup timestamp
  - File size
  - Encryption status
  - Number of files backed up (with --details)

Use this to find which backup to restore.`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listDetails, "details", "d", false, "Show detailed metadata (may be slow)")
	listCmd.Flags().BoolVarP(&listVerbose, "verbose", "v", false, "Show verbose output")
}

type backupInfo struct {
	Path      string
	Name      string
	Size      int64
	ModTime   time.Time
	Encrypted bool
	Metadata  *metadata.Metadata
}

func runList(cmd *cobra.Command, args []string) error {
	ui.Verbose = listVerbose

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg.ExpandPaths()

	if _, err := os.Stat(cfg.BackupDir); os.IsNotExist(err) {
		ui.PrintInfo("No backups found")
		ui.PrintDim("  Run: stash backup")
		return nil
	}

	backups, err := findBackups(cfg.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to find backups: %w", err)
	}

	if len(backups) == 0 {
		ui.PrintInfo("No backups found in %s", cfg.BackupDir)
		ui.PrintDim("  Run: stash backup")
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	// Build table
	headers := []string{"NAME", "SIZE", "DATE"}
	if listDetails {
		headers = append(headers, "FILES")
	}

	var rows [][]string
	for _, backup := range backups {
		// Shorten name for display
		name := backup.Name
		if len(name) > 35 {
			name = name[:32] + "..."
		}

		encIcon := ""
		if !backup.Encrypted {
			encIcon = " (!)"
		}

		row := []string{
			name + encIcon,
			metadata.FormatSize(backup.Size),
			backup.ModTime.Format("2006-01-02 15:04"),
		}

		if listDetails && backup.Metadata != nil {
			row = append(row, fmt.Sprintf("%d", len(backup.Metadata.Files)))
		} else if listDetails {
			row = append(row, "-")
		}

		rows = append(rows, row)
	}

	// Print table
	ui.PrintTable(headers, rows)

	// Summary
	fmt.Println()
	ui.PrintDim("%d backup(s) in %s", len(backups), cfg.BackupDir)

	// Verbose: show restore hint
	if listVerbose {
		fmt.Println()
		ui.PrintDim("Restore: stash restore %s", backups[0].Name)
	}

	return nil
}

func findBackups(backupDir string) ([]backupInfo, error) {
	var backups []backupInfo

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if !strings.HasSuffix(name, ".tar.gz.age") && !strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		path := filepath.Join(backupDir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		backup := backupInfo{
			Path:      path,
			Name:      name,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			Encrypted: strings.HasSuffix(name, ".age"),
		}

		// Load metadata if --details flag is set
		if listDetails {
			meta, err := readMetadataFromBackup(path)
			if err == nil {
				backup.Metadata = meta
			}
		}

		backups = append(backups, backup)
	}

	return backups, nil
}

func readMetadataFromBackup(backupPath string) (*metadata.Metadata, error) {
	return backuputil.ExtractMetadata(backupPath, "")
}

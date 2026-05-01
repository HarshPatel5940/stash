package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/harshpatel5940/stash/internal/backuputil"
	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/incremental"
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
  - Numeric ID for quick restore
  - File size and date
  - Note/type summary in INFO column
  - Number of files backed up (with --details)`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listDetails, "details", "d", false, "Show detailed metadata (may be slow)")
	listCmd.Flags().BoolVarP(&listVerbose, "verbose", "v", false, "Show verbose output")
}

type backupInfo struct {
	Path      string
	Index     int
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

	backups, err := collectBackups(cfg.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to find backups: %w", err)
	}

	if len(backups) == 0 {
		ui.PrintInfo("No backups found in %s", cfg.BackupDir)
		ui.PrintDim("  Run: stash backup")
		return nil
	}

	registry, _ := incremental.LoadRegistry()

	// Build table
	headers := []string{"ID", "NAME", "SIZE", "DATE", "INFO"}
	if listDetails {
		headers = append(headers, "FILES", "HOST")
	}

	var rows [][]string
	for _, backup := range backups {
		if listDetails {
			meta, err := readMetadataFromBackup(backup.Path, cfg.EncryptionKey)
			if err == nil {
				backup.Metadata = meta
			}
		}

		name := backup.Name
		if len(name) > 35 {
			name = name[:32] + "..."
		}

		encIcon := ""
		if !backup.Encrypted {
			encIcon = " (!)"
		}

		row := []string{
			fmt.Sprintf("%d", backup.Index),
			name + encIcon,
			metadata.FormatSize(backup.Size),
			backup.ModTime.Format("2006-01-02 15:04"),
			buildListInfo(backup, registry),
		}

		if listDetails {
			if backup.Metadata != nil {
				host := backup.Metadata.Hostname
				if host == "" {
					host = "-"
				}
				row = append(row, fmt.Sprintf("%d", len(backup.Metadata.Files)), host)
			} else {
				row = append(row, "-", "-")
			}
		}

		rows = append(rows, row)
	}

	ui.PrintTable(headers, rows)

	fmt.Println()
	ui.PrintDim("%d backup(s) in %s", len(backups), cfg.BackupDir)

	if listVerbose {
		fmt.Println()
		ui.PrintDim("Restore latest: stash restore 1")
	}

	return nil
}

func readMetadataFromBackup(backupPath string, keyPath string) (*metadata.Metadata, error) {
	return backuputil.ExtractMetadata(backupPath, keyPath)
}

func buildListInfo(backup backupInfo, registry *incremental.BackupRegistry) string {
	var parts []string

	if registry != nil {
		if entry, ok := registry.GetBackup(normalizeBackupKey(backup.Name)); ok && entry.BackupType != "" {
			parts = append(parts, entry.BackupType)
		}
	}

	if note, err := loadBackupNote(backup.Name); err == nil && note != "" {
		parts = append(parts, truncateInfo(note, 40))
	}

	if !backup.Encrypted {
		parts = append(parts, "plain")
	}

	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " | ")
}

func truncateInfo(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

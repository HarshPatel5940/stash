package cmd

import (
	"fmt"
	"time"

	"github.com/harshpatel5940/stash/internal/cleanup"
	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cleanupKeepCount int
	cleanupMaxAge    int
	cleanupDryRun    bool
	cleanupVerbose   bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup old backups",
	Long: `Remove old backups based on count or age criteria.

Examples:
  stash cleanup --keep 10       # Keep 10 most recent
  stash cleanup --max-age 30    # Delete older than 30 days
  stash cleanup --dry-run       # Preview deletions`,
	RunE: runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().IntVarP(&cleanupKeepCount, "keep", "k", 5, "Number of backups to keep (0 = disable)")
	cleanupCmd.Flags().IntVarP(&cleanupMaxAge, "max-age", "a", 0, "Delete backups older than N days (0 = disable)")
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Preview deletions")
	cleanupCmd.Flags().BoolVarP(&cleanupVerbose, "verbose", "v", false, "Show detailed output")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	ui.Verbose = cleanupVerbose

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cm := cleanup.NewCleanupManager(cfg.BackupDir)

	stats, err := cm.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get backup stats: %w", err)
	}

	count := stats["count"].(int)
	if count == 0 {
		ui.PrintInfo("No backups found")
		return nil
	}

	totalSize := stats["total_size"].(int64)

	// Verbose: show all backups
	if cleanupVerbose {
		backups, _ := cm.ListBackups()
		for i, backup := range backups {
			wouldDelete := cleanupKeepCount > 0 && i >= cleanupKeepCount
			if wouldDelete {
				fmt.Printf("  %s %s (delete)\n", ui.IconError, backup)
			} else {
				fmt.Printf("  %s %s (keep)\n", ui.IconSuccess, backup)
			}
		}
		fmt.Println()
	}

	// Dry run mode
	if cleanupDryRun {
		toDelete := 0
		if cleanupKeepCount > 0 && count > cleanupKeepCount {
			toDelete = count - cleanupKeepCount
		}
		if toDelete > 0 {
			ui.PrintInfo("DRY RUN: Would delete %d backup(s), keep %d", toDelete, cleanupKeepCount)
		} else {
			ui.PrintInfo("DRY RUN: Nothing to delete (keeping %d)", cleanupKeepCount)
		}
		return nil
	}

	var deleted int

	if cleanupKeepCount > 0 {
		deleted, err = cm.RotateByCount(cleanupKeepCount)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
	}

	if cleanupMaxAge > 0 {
		maxAge := time.Duration(cleanupMaxAge) * 24 * time.Hour
		ageDeleted, err := cm.RotateByAge(maxAge)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
		deleted += ageDeleted
	}

	// Result
	if deleted > 0 {
		newStats, _ := cm.GetStats()
		newSize := newStats["total_size"].(int64)
		freed := totalSize - newSize
		ui.PrintSuccess("Deleted %d backup(s), freed %s", deleted, ui.FormatBytes(freed))
	} else {
		ui.PrintSuccess("No cleanup needed (%d backups)", count)
	}

	return nil
}

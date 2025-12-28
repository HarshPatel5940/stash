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
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup old backups",
	Long: `Remove old backups based on count or age criteria.

By default, keeps the 5 most recent backups and deletes older ones.

Examples:
  stash cleanup --keep 10          # Keep 10 most recent backups
  stash cleanup --max-age 30       # Delete backups older than 30 days
  stash cleanup --dry-run          # Preview what would be deleted`,
	RunE: runCleanup,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().IntVarP(&cleanupKeepCount, "keep", "k", 5, "Number of backups to keep (0 = disable)")
	cleanupCmd.Flags().IntVarP(&cleanupMaxAge, "max-age", "a", 0, "Delete backups older than N days (0 = disable)")
	cleanupCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Preview what would be deleted without deleting")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ui.PrintSectionHeader("🧹", "Backup Cleanup")

	cm := cleanup.NewCleanupManager(cfg.BackupDir)

	stats, err := cm.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get backup stats: %w", err)
	}

	count := stats["count"].(int)
	if count == 0 {
		ui.PrintInfo("No backups found in %s", cfg.BackupDir)
		return nil
	}

	totalSize := stats["total_size"].(int64)
	ui.PrintInfo("Found %d backup(s), total size: %s", count, ui.FormatBytes(totalSize))

	fmt.Println()
	backups, _ := cm.ListBackups()
	for i, backup := range backups {
		if cleanupDryRun {

			wouldDelete := false
			if cleanupKeepCount > 0 && i >= cleanupKeepCount {
				wouldDelete = true
			}
			if wouldDelete {
				fmt.Printf("  %s %s\n", ui.Error("✗"), backup)
			} else {
				fmt.Printf("  %s %s\n", ui.Success("✓"), backup)
			}
		} else {
			fmt.Printf("  • %s\n", backup)
		}
	}

	if cleanupDryRun {
		fmt.Println()
		ui.PrintInfo("DRY RUN MODE - No files will be deleted")
	}

	var deleted int

	if cleanupKeepCount > 0 {
		fmt.Println()
		if cleanupDryRun {
			toDelete := count - cleanupKeepCount
			if toDelete > 0 {
				ui.PrintInfo("Would delete %d backup(s), keeping %d most recent", toDelete, cleanupKeepCount)
			} else {
				ui.PrintInfo("No cleanup needed (keeping %d backups)", cleanupKeepCount)
			}
			return nil
		}

		ui.PrintSectionHeader("🗑️", "Deleting old backups...")
		deleted, err = cm.RotateByCount(cleanupKeepCount)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
	}

	if cleanupMaxAge > 0 {
		fmt.Println()
		maxAge := time.Duration(cleanupMaxAge) * 24 * time.Hour

		if cleanupDryRun {
			ui.PrintInfo("Would delete backups older than %d days", cleanupMaxAge)
			return nil
		}

		ui.PrintSectionHeader("🗑️", "Deleting old backups...")
		ageDeleted, err := cm.RotateByAge(maxAge)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
		deleted += ageDeleted
	}

	fmt.Println()
	ui.PrintDivider()
	if deleted > 0 {
		ui.PrintSuccess("Deleted %d old backup(s)", deleted)

		newStats, _ := cm.GetStats()
		newCount := newStats["count"].(int)
		newSize := newStats["total_size"].(int64)
		freed := totalSize - newSize

		ui.PrintInfo("Remaining: %d backup(s), %s", newCount, ui.FormatBytes(newSize))
		ui.PrintInfo("Space freed: %s", ui.FormatBytes(freed))
	} else {
		ui.PrintInfo("No backups were deleted")
	}
	ui.PrintDivider()

	return nil
}

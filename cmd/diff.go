package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/harshpatel5940/stash/internal/diff"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	diffVerbose      bool
	diffShowPackages bool
)

var diffCmd = &cobra.Command{
	Use:   "diff <backup1> <backup2>",
	Short: "Compare two backups and show differences",
	Long: `Compare two backup files and display the differences between them.

Shows:
  - Files that were added, removed, or modified
  - Size changes for each category
  - Package manager changes (Homebrew, npm, etc.)
  - Visual colored diff output

Examples:
  stash diff backup-old.tar.gz.age backup-new.tar.gz.age
  stash diff backup-2024-01-01.tar.gz.age backup-2024-01-15.tar.gz.age --verbose`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().BoolVarP(&diffVerbose, "verbose", "v", false, "Show detailed file-by-file changes")
	diffCmd.Flags().BoolVar(&diffShowPackages, "packages", true, "Show package manager changes")
}

func runDiff(cmd *cobra.Command, args []string) error {
	oldBackup := args[0]
	newBackup := args[1]

	// Make paths absolute
	oldBackup, _ = filepath.Abs(oldBackup)
	newBackup, _ = filepath.Abs(newBackup)

	ui.PrintSectionHeader("📊", "Comparing Backups")

	// Perform the comparison
	result, err := diff.Compare(oldBackup, newBackup)
	if err != nil {
		ui.PrintError("Failed to compare backups: %v", err)
		fmt.Println()
		ui.PrintInfo("Note: To compare encrypted backups, metadata must be accessible")
		ui.PrintInfo("Consider saving metadata alongside backups for easier comparison")
		return err
	}

	// Print comparison header
	ui.PrintComparisonHeader(filepath.Base(oldBackup), filepath.Base(newBackup), result.OldSize, result.NewSize)

	// Print file changes summary
	ui.PrintFileChanges(
		result.GetAddedFilesCount(),
		result.GetRemovedFilesCount(),
		result.GetModifiedFilesCount(),
		result.UnchangedCount,
		result.AddedSize,
		result.RemovedSize,
		result.ModifiedSize,
	)

	// Show added files
	if len(result.AddedFiles) > 0 {
		fmt.Println(ui.Bold("Added Files:"))
		limit := 10
		if diffVerbose {
			limit = len(result.AddedFiles)
		}
		for i, file := range result.AddedFiles {
			if i >= limit {
				fmt.Printf("  %s ... and %d more\n", ui.Info("▶"), len(result.AddedFiles)-limit)
				break
			}
			if !file.IsDir {
				fmt.Printf("  %s %s (%s)\n", ui.Success("+"), file.OriginalPath, ui.FormatBytes(file.Size))
			}
		}
		fmt.Println()
	}

	// Show removed files
	if len(result.RemovedFiles) > 0 {
		fmt.Println(ui.Bold("Removed Files:"))
		limit := 10
		if diffVerbose {
			limit = len(result.RemovedFiles)
		}
		for i, file := range result.RemovedFiles {
			if i >= limit {
				fmt.Printf("  %s ... and %d more\n", ui.Info("▶"), len(result.RemovedFiles)-limit)
				break
			}
			if !file.IsDir {
				fmt.Printf("  %s %s (%s)\n", ui.Error("-"), file.OriginalPath, ui.FormatBytes(file.Size))
			}
		}
		fmt.Println()
	}

	// Show modified files
	if len(result.ModifiedFiles) > 0 {
		fmt.Println(ui.Bold("Modified Files:"))
		limit := 10
		if diffVerbose {
			limit = len(result.ModifiedFiles)
		}

		topModified := result.GetTopModifiedFiles(limit)
		for _, change := range topModified {
			sign := "+"
			sizeDelta := change.SizeDelta
			if sizeDelta < 0 {
				sign = ""
			}
			fmt.Printf("  %s %-50s  %10s → %10s (%s%s)\n",
				ui.Warning("~"),
				truncatePath(change.Path, 50),
				ui.FormatBytes(change.OldSize),
				ui.FormatBytes(change.NewSize),
				sign,
				ui.FormatBytes(sizeDelta),
			)
		}
		if !diffVerbose && len(result.ModifiedFiles) > limit {
			fmt.Printf("  %s ... and %d more\n", ui.Info("▶"), len(result.ModifiedFiles)-limit)
		}
		fmt.Println()
	}

	// Show package changes
	if diffShowPackages && len(result.PackageChanges) > 0 {
		ui.PrintSectionHeader("📦", "PACKAGE CHANGES")
		for pkgType, change := range result.PackageChanges {
			sign := "+"
			if change.Delta < 0 {
				sign = ""
			}
			delta := change.Delta
			if delta < 0 {
				delta = -delta
			}

			fmt.Printf("  %-15s %3d → %3d (%s%d)\n",
				pkgType+":",
				change.OldCount,
				change.NewCount,
				sign,
				delta,
			)
		}
		fmt.Println()
	}

	// Print summary
	if !result.HasChanges() {
		ui.PrintSuccess("No changes detected between backups")
	} else {
		fmt.Println(ui.Bold("Summary:"))
		fmt.Println(result.Summary())
	}

	return nil
}

// truncatePath truncates a path to fit within maxLen characters
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

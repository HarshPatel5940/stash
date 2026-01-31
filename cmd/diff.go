package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/diff"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	diffVerbose      bool
	diffShowPackages bool
	diffDecryptKey   string
)

var diffCmd = &cobra.Command{
	Use:   "diff <backup1> <backup2>",
	Short: "Compare two backups and show differences",
	Long: `Compare two backup files and display the differences between them.

Shows:
  - Files that were added, removed, or modified
  - Size changes for each category
  - Package manager changes (Homebrew, npm, etc.)

Examples:
  stash diff backup-old.tar.gz.age backup-new.tar.gz.age
  stash diff backup-2024-01-01.tar.gz.age backup-2024-01-15.tar.gz.age -v`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.Flags().BoolVarP(&diffVerbose, "verbose", "v", false, "Show detailed file-by-file changes")
	diffCmd.Flags().BoolVar(&diffShowPackages, "packages", true, "Show package manager changes")
	diffCmd.Flags().StringVarP(&diffDecryptKey, "decrypt-key", "k", "", "Path to decryption key (default: ~/.stash.key)")
}

func runDiff(cmd *cobra.Command, args []string) error {
	ui.Verbose = diffVerbose

	oldBackup := args[0]
	newBackup := args[1]

	oldBackup, _ = filepath.Abs(oldBackup)
	newBackup, _ = filepath.Abs(newBackup)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Perform the comparison
	opts := diff.CompareOptions{
		KeyPath: diffDecryptKey,
	}
	result, err := diff.CompareWithOptions(oldBackup, newBackup, opts)
	if err != nil {
		ui.PrintError("Failed to compare: %v", err)
		ui.PrintDim("  Ensure key is at ~/.stash.key or use --decrypt-key")
		return err
	}

	// Minimal output: single line summary
	if !result.HasChanges() {
		ui.PrintSuccess("No changes")
		return nil
	}

	// Print file changes summary (always shown)
	ui.PrintFileChanges(
		result.GetAddedFilesCount(),
		result.GetRemovedFilesCount(),
		result.GetModifiedFilesCount(),
		result.UnchangedCount,
		result.AddedSize,
		result.RemovedSize,
		result.ModifiedSize,
	)

	// Verbose: detailed file lists
	if diffVerbose {
		// Added files
		if len(result.AddedFiles) > 0 {
			fmt.Printf("\n%s:\n", ui.Bold("Added"))
			limit := cfg.GetDiffDisplayLimit()
			for i, file := range result.AddedFiles {
				if i >= limit {
					ui.PrintDim("  ... and %d more", len(result.AddedFiles)-limit)
					break
				}
				if !file.IsDir {
					fmt.Printf("  %s %s (%s)\n", ui.Success("+"), file.OriginalPath, ui.FormatBytes(file.Size))
				}
			}
		}

		// Removed files
		if len(result.RemovedFiles) > 0 {
			fmt.Printf("\n%s:\n", ui.Bold("Removed"))
			limit := cfg.GetDiffDisplayLimit()
			for i, file := range result.RemovedFiles {
				if i >= limit {
					ui.PrintDim("  ... and %d more", len(result.RemovedFiles)-limit)
					break
				}
				if !file.IsDir {
					fmt.Printf("  %s %s (%s)\n", ui.Error("-"), file.OriginalPath, ui.FormatBytes(file.Size))
				}
			}
		}

		// Modified files
		if len(result.ModifiedFiles) > 0 {
			fmt.Printf("\n%s:\n", ui.Bold("Modified"))
			limit := cfg.GetDiffDisplayLimit()
			topModified := result.GetTopModifiedFiles(limit)
			for _, change := range topModified {
				sign := "+"
				sizeDelta := change.SizeDelta
				if sizeDelta < 0 {
					sign = ""
				}
				fmt.Printf("  %s %s (%s%s)\n",
					ui.Warning("~"),
					truncateDiffPath(change.Path, 50),
					sign,
					ui.FormatBytes(sizeDelta),
				)
			}
			if len(result.ModifiedFiles) > limit {
				ui.PrintDim("  ... and %d more", len(result.ModifiedFiles)-limit)
			}
		}

		// Package changes
		if diffShowPackages && len(result.PackageChanges) > 0 {
			fmt.Printf("\n%s:\n", ui.Bold("Packages"))
			for pkgType, change := range result.PackageChanges {
				if change.Delta != 0 {
					sign := "+"
					if change.Delta < 0 {
						sign = ""
					}
					fmt.Printf("  %s: %d → %d (%s%d)\n",
						pkgType,
						change.OldCount,
						change.NewCount,
						sign,
						change.Delta,
					)
				}
			}
		}
	}

	return nil
}

func truncateDiffPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

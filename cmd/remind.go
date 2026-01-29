package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harshpatel5940/stash/internal/gittracker"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	remindVerbose bool
)

var remindCmd = &cobra.Command{
	Use:   "remind",
	Short: "Show git repos needing attention",
	Long: `Scans your common project directories for git repositories and shows
which ones have uncommitted changes or unpushed commits.

This is useful to run before backup or at the end of the day to ensure
all your work is safely committed and pushed.

Scanned directories:
  - ~/Documents
  - ~/Projects
  - ~/Code
  - ~/Dev
  - ~/workspace
  - ~/github`,
	RunE: runRemind,
}

func init() {
	rootCmd.AddCommand(remindCmd)
	remindCmd.Flags().BoolVarP(&remindVerbose, "verbose", "v", false, "Show all repos, not just those needing attention")
}

func runRemind(cmd *cobra.Command, args []string) error {
	ui.PrintSectionHeader("🔍", "Scanning Git Repositories")

	gt := gittracker.NewGitTracker("")

	homeDir, _ := os.UserHomeDir()
	searchDirs := []string{
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Projects"),
		filepath.Join(homeDir, "Code"),
		filepath.Join(homeDir, "Dev"),
		filepath.Join(homeDir, "workspace"),
		filepath.Join(homeDir, "github"),
	}

	if err := gt.ScanDirectories(searchDirs); err != nil {
		return fmt.Errorf("failed to scan directories: %w", err)
	}

	allRepos := gt.GetRepos()
	if len(allRepos) == 0 {
		fmt.Println("\nNo git repositories found in common directories.")
		return nil
	}

	needsAttention := gt.GetReposNeedingAttention()

	if remindVerbose {
		// Show all repos with their status
		fmt.Printf("\n📁 Found %d repositories:\n\n", len(allRepos))

		for _, repo := range allRepos {
			shortPath := shortenPath(repo.Path, homeDir)
			status := repo.GetStatusSummary()

			if repo.NeedsAttention() {
				fmt.Printf("  ⚠️  %s\n", shortPath)
				fmt.Printf("      Branch: %s | Status: %s\n\n", repo.Branch, ui.Warning(status))
			} else {
				fmt.Printf("  ✓  %s\n", shortPath)
				fmt.Printf("      Branch: %s | Status: %s\n\n", repo.Branch, ui.Success(status))
			}
		}
	}

	if len(needsAttention) == 0 {
		fmt.Println()
		ui.PrintSuccess("All %d repositories are clean and synced!", len(allRepos))
		return nil
	}

	// Show repos needing attention
	fmt.Printf("\n⚠️  %d of %d repositories need attention:\n\n", len(needsAttention), len(allRepos))

	for _, repo := range needsAttention {
		shortPath := shortenPath(repo.Path, homeDir)
		fmt.Printf("  %s\n", ui.Bold(shortPath))
		fmt.Printf("    Branch: %s\n", repo.Branch)

		if repo.Dirty {
			fmt.Printf("    %s Uncommitted changes\n", ui.Warning("•"))
		}
		if repo.UnpushedCount > 0 {
			fmt.Printf("    %s %d unpushed commit(s)\n", ui.Warning("•"), repo.UnpushedCount)
		}
		if repo.Behind > 0 {
			fmt.Printf("    %s %d commit(s) behind remote\n", ui.Info("•"), repo.Behind)
		}
		fmt.Println()
	}

	// Print suggestions
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println("\n💡 Suggestions:")

	hasUncommitted := false
	hasUnpushed := false
	for _, repo := range needsAttention {
		if repo.Dirty {
			hasUncommitted = true
		}
		if repo.UnpushedCount > 0 {
			hasUnpushed = true
		}
	}

	if hasUncommitted {
		fmt.Println("   • Commit your changes: git add . && git commit -m \"message\"")
	}
	if hasUnpushed {
		fmt.Println("   • Push your commits: git push")
	}
	fmt.Println("   • Run 'stash backup' after syncing to create a backup")

	return nil
}

// shortenPath replaces home directory with ~
func shortenPath(path, homeDir string) string {
	if strings.HasPrefix(path, homeDir) {
		return "~" + path[len(homeDir):]
	}
	return path
}

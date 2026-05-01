package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/harshpatel5940/stash/internal/config"
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
	Long: `Scans your project directories for git repositories and shows
which ones have uncommitted changes or unpushed commits.

Run before backup or at end of day to ensure all work is committed.`,
	RunE: runRemind,
}

func init() {
	rootCmd.AddCommand(remindCmd)
	remindCmd.Flags().BoolVarP(&remindVerbose, "verbose", "v", false, "Show all repos, not just those needing attention")
}

func runRemind(cmd *cobra.Command, args []string) error {
	ui.Verbose = remindVerbose

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	cfg.ExpandPaths()

	gt := gittracker.NewGitTrackerWithConfig(
		"",
		cfg.GetGitMaxDepth(),
		cfg.GetGitSkipDirs(),
	)

	searchDirs := cfg.GetGitSearchDirs()

	if err := gt.ScanDirectories(searchDirs); err != nil {
		return fmt.Errorf("failed to scan: %w", err)
	}

	homeDir, _ := os.UserHomeDir()

	allRepos := gt.GetRepos()
	if len(allRepos) == 0 {
		ui.PrintInfo("No git repositories found")
		return nil
	}

	needsAttention := gt.GetReposNeedingAttention()

	// Minimal: all clean
	if len(needsAttention) == 0 {
		ui.PrintSuccess("All %d repos clean", len(allRepos))
		return nil
	}

	// Show repos needing attention
	fmt.Printf("%d of %d repos need attention:\n", len(needsAttention), len(allRepos))
	uncommittedCount := 0
	unpushedCount := 0
	noUpstreamCount := 0

	for _, repo := range needsAttention {
		shortPath := truncateRemindPath(shortenRemindPath(repo.Path, homeDir), 80)
		issues := []string{}

		if repo.Dirty {
			issues = append(issues, "uncommitted")
			uncommittedCount++
		}
		if repo.UnpushedCount > 0 {
			issues = append(issues, fmt.Sprintf("%d unpushed", repo.UnpushedCount))
			unpushedCount++
		}
		if !repo.HasUpstream && len(repo.Remotes) > 0 {
			issues = append(issues, "no upstream")
			noUpstreamCount++
		}
		if repo.Behind > 0 {
			issues = append(issues, fmt.Sprintf("%d behind", repo.Behind))
		}

		fmt.Printf("  %s %s (%s)\n", ui.IconWarning, shortPath, strings.Join(issues, ", "))
	}
	fmt.Printf("\n  Uncommitted: %d  Unpushed: %d  No upstream: %d\n", uncommittedCount, unpushedCount, noUpstreamCount)

	// Verbose: show all repos
	if remindVerbose {
		fmt.Println()
		ui.PrintDivider()
		fmt.Printf("All repositories (%d):\n", len(allRepos))
		for _, repo := range allRepos {
			shortPath := truncateRemindPath(shortenRemindPath(repo.Path, homeDir), 80)
			if repo.NeedsAttention() {
				fmt.Printf("  %s %s\n", ui.IconWarning, shortPath)
			} else {
				fmt.Printf("  %s %s\n", ui.IconSuccess, shortPath)
			}
		}
	}

	return nil
}

func shortenRemindPath(path, homeDir string) string {
	if strings.HasPrefix(path, homeDir) {
		return "~" + path[len(homeDir):]
	}
	return path
}

func truncateRemindPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	if maxLen <= 3 {
		return path[:maxLen]
	}
	keepTail := maxLen - 6
	if keepTail < 1 {
		keepTail = 1
	}
	return path[:3] + "..." + path[len(path)-keepTail:]
}

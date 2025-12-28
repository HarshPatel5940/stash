package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available backups",
	Long: `Lists all backups found in the backup directory.

Shows backup details including:
  - Backup timestamp
  - File size
  - Encryption status
  - Number of files backed up

Use this to find which backup to restore.`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
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

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg.ExpandPaths()

	if _, err := os.Stat(cfg.BackupDir); os.IsNotExist(err) {
		fmt.Printf("📁 No backups found\n")
		fmt.Printf("\n💡 Backup directory doesn't exist: %s\n", cfg.BackupDir)
		fmt.Println("💡 Run 'stash backup' to create your first backup")
		return nil
	}

	backups, err := findBackups(cfg.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to find backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Printf("📁 No backups found in %s\n", cfg.BackupDir)
		fmt.Println("\n💡 Run 'stash backup' to create your first backup")
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	fmt.Println("📦 Available Backups")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	for i, backup := range backups {
		fmt.Printf("%d. %s\n", i+1, backup.Name)
		fmt.Printf("   📅 Created: %s\n", backup.ModTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("   💾 Size: %s\n", metadata.FormatSize(backup.Size))

		if backup.Encrypted {
			fmt.Printf("   🔐 Encrypted: Yes\n")
		} else {
			fmt.Printf("   ⚠️  Encrypted: No\n")
		}

		if backup.Metadata != nil {
			totalPackages := 0
			for _, count := range backup.Metadata.PackageCounts {
				totalPackages += count
			}
			fmt.Printf("   📊 Files: %d | Packages: %d\n",
				len(backup.Metadata.Files),
				totalPackages)
		}

		fmt.Printf("   📍 Path: %s\n", backup.Path)
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Total: %d backup(s) found\n", len(backups))
	fmt.Println()
	fmt.Println("💡 To restore a backup:")
	fmt.Printf("   stash restore %s\n", backups[0].Path)
	fmt.Println()
	fmt.Println("💡 To preview what would be backed up:")
	fmt.Println("   stash backup --dry-run")

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

		if strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".age") {

		}

		backups = append(backups, backup)
	}

	return backups, nil
}

func readMetadataFromBackup(backupPath string) (*metadata.Metadata, error) {

	return nil, nil
}

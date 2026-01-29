package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/harshpatel5940/stash/internal/cloud"
	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	syncBucket   string
	syncRegion   string
	syncEndpoint string
	syncPrefix   string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync backups with cloud storage",
	Long: `Synchronize backups with S3-compatible cloud storage.

Supports:
  - AWS S3
  - Backblaze B2
  - MinIO
  - DigitalOcean Spaces
  - Cloudflare R2
  - Any S3-compatible service

Configure in ~/.stash.yaml or use flags:
  cloud:
    enabled: true
    provider: s3
    bucket: my-backups
    region: us-east-1
    endpoint: ""  # Optional, for non-AWS S3 services
    prefix: stash/  # Optional path prefix`,
}

var syncUpCmd = &cobra.Command{
	Use:   "up [backup-file]",
	Short: "Upload backup(s) to cloud",
	Long: `Upload one or all local backups to cloud storage.

Examples:
  stash sync up                           # Upload all local backups
  stash sync up backup-2024-01-15.tar.gz.age  # Upload specific backup`,
	RunE: runSyncUp,
}

var syncDownCmd = &cobra.Command{
	Use:   "down <backup-name>",
	Short: "Download backup from cloud",
	Long: `Download a backup from cloud storage.

Examples:
  stash sync down backup-2024-01-15.tar.gz.age`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncDown,
}

var syncListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups in cloud storage",
	Long:  `List all backups stored in cloud storage.`,
	RunE:  runSyncList,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.AddCommand(syncUpCmd)
	syncCmd.AddCommand(syncDownCmd)
	syncCmd.AddCommand(syncListCmd)

	// Global flags for sync commands
	syncCmd.PersistentFlags().StringVar(&syncBucket, "bucket", "", "S3 bucket name")
	syncCmd.PersistentFlags().StringVar(&syncRegion, "region", "", "AWS region")
	syncCmd.PersistentFlags().StringVar(&syncEndpoint, "endpoint", "", "Custom S3 endpoint")
	syncCmd.PersistentFlags().StringVar(&syncPrefix, "prefix", "", "Path prefix in bucket")
}

func getCloudProvider() (cloud.Provider, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}
	cfg.ExpandPaths()

	// Build cloud config from flags or config file
	cloudCfg := cloud.Config{
		Provider: "s3",
	}

	if cfg.Cloud != nil {
		cloudCfg.Bucket = cfg.Cloud.Bucket
		cloudCfg.Region = cfg.Cloud.Region
		cloudCfg.Endpoint = cfg.Cloud.Endpoint
		cloudCfg.Prefix = cfg.Cloud.Prefix
	}

	// Override with flags
	if syncBucket != "" {
		cloudCfg.Bucket = syncBucket
	}
	if syncRegion != "" {
		cloudCfg.Region = syncRegion
	}
	if syncEndpoint != "" {
		cloudCfg.Endpoint = syncEndpoint
	}
	if syncPrefix != "" {
		cloudCfg.Prefix = syncPrefix
	}

	// Validate
	if cloudCfg.Bucket == "" {
		return nil, nil, fmt.Errorf("bucket not configured. Set in ~/.stash.yaml or use --bucket flag")
	}
	if cloudCfg.Region == "" {
		return nil, nil, fmt.Errorf("region not configured. Set in ~/.stash.yaml or use --region flag")
	}

	provider, err := cloud.NewProvider(cloudCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cloud provider: %w", err)
	}

	return provider, cfg, nil
}

func runSyncUp(cmd *cobra.Command, args []string) error {
	provider, cfg, err := getCloudProvider()
	if err != nil {
		return err
	}

	ui.PrintSectionHeader("☁️", fmt.Sprintf("Uploading to %s", provider.GetName()))

	if len(args) > 0 {
		// Upload specific file
		backupFile := args[0]
		if !filepath.IsAbs(backupFile) {
			backupFile = filepath.Join(cfg.BackupDir, backupFile)
		}

		if _, err := os.Stat(backupFile); err != nil {
			return fmt.Errorf("backup file not found: %s", backupFile)
		}

		return uploadBackup(provider, backupFile)
	}

	// Upload all local backups
	entries, err := os.ReadDir(cfg.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tar.gz.age") || strings.HasSuffix(name, ".tar.gz") {
			backups = append(backups, filepath.Join(cfg.BackupDir, name))
		}
	}

	if len(backups) == 0 {
		fmt.Println("\nNo backups found to upload")
		return nil
	}

	fmt.Printf("\nFound %d local backup(s)\n\n", len(backups))

	uploaded := 0
	skipped := 0
	for _, backup := range backups {
		name := filepath.Base(backup)
		exists, err := provider.Exists(name)
		if err != nil {
			fmt.Printf("  ⚠️  Error checking %s: %v\n", name, err)
			continue
		}

		if exists {
			fmt.Printf("  ⏭️  %s (already exists)\n", name)
			skipped++
			continue
		}

		if err := uploadBackup(provider, backup); err != nil {
			fmt.Printf("  ✗ Failed to upload %s: %v\n", name, err)
		} else {
			uploaded++
		}
	}

	fmt.Printf("\n✓ Uploaded: %d, Skipped: %d\n", uploaded, skipped)
	return nil
}

func uploadBackup(provider cloud.Provider, backupPath string) error {
	name := filepath.Base(backupPath)
	info, err := os.Stat(backupPath)
	if err != nil {
		return err
	}

	fmt.Printf("  ⬆️  Uploading %s (%s)...", name, metadata.FormatSize(info.Size()))

	if err := provider.Upload(backupPath, name); err != nil {
		fmt.Printf(" ✗\n")
		return err
	}

	fmt.Printf(" ✓\n")
	return nil
}

func runSyncDown(cmd *cobra.Command, args []string) error {
	provider, cfg, err := getCloudProvider()
	if err != nil {
		return err
	}

	backupName := args[0]

	ui.PrintSectionHeader("☁️", fmt.Sprintf("Downloading from %s", provider.GetName()))

	// Check if file exists in cloud
	exists, err := provider.Exists(backupName)
	if err != nil {
		return fmt.Errorf("failed to check cloud: %w", err)
	}
	if !exists {
		return fmt.Errorf("backup not found in cloud: %s", backupName)
	}

	localPath := filepath.Join(cfg.BackupDir, backupName)

	// Check if already exists locally
	if _, err := os.Stat(localPath); err == nil {
		fmt.Printf("\n⚠️  File already exists locally: %s\n", localPath)
		fmt.Println("   Delete it first if you want to re-download")
		return nil
	}

	fmt.Printf("\n⬇️  Downloading %s...", backupName)

	if err := provider.Download(backupName, localPath); err != nil {
		fmt.Printf(" ✗\n")
		return fmt.Errorf("failed to download: %w", err)
	}

	fmt.Printf(" ✓\n")
	fmt.Printf("\n✓ Downloaded to: %s\n", localPath)
	return nil
}

func runSyncList(cmd *cobra.Command, args []string) error {
	provider, _, err := getCloudProvider()
	if err != nil {
		return err
	}

	ui.PrintSectionHeader("☁️", fmt.Sprintf("Cloud Backups (%s)", provider.GetName()))

	entries, err := provider.List("")
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("\nNo backups found in cloud storage")
		fmt.Println("\n💡 Upload backups with: stash sync up")
		return nil
	}

	// Sort by date descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastModified.After(entries[j].LastModified)
	})

	fmt.Printf("\n%d backup(s) found:\n\n", len(entries))

	for i, entry := range entries {
		fmt.Printf("%d. %s\n", i+1, entry.Name)
		fmt.Printf("   📅 Modified: %s\n", entry.LastModified.Format("2006-01-02 15:04:05"))
		fmt.Printf("   💾 Size: %s\n\n", metadata.FormatSize(entry.Size))
	}

	fmt.Println("💡 Download with: stash sync down <backup-name>")
	return nil
}

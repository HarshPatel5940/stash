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
	syncVerbose  bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync backups with cloud storage",
	Long: `Synchronize backups with S3-compatible cloud storage.

Supports AWS S3, Backblaze B2, MinIO, DigitalOcean Spaces, Cloudflare R2.

Configure in ~/.stash.yaml:
  cloud:
    enabled: true
    bucket: my-backups
    region: us-east-1`,
}

var syncUpCmd = &cobra.Command{
	Use:   "up [backup-file]",
	Short: "Upload backup(s) to cloud",
	Long: `Upload one or all local backups to cloud storage.

Examples:
  stash sync up                    # Upload all
  stash sync up backup.tar.gz.age  # Upload specific`,
	RunE: runSyncUp,
}

var syncDownCmd = &cobra.Command{
	Use:   "down <backup-name>",
	Short: "Download backup from cloud",
	Args:  cobra.ExactArgs(1),
	RunE:  runSyncDown,
}

var syncListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups in cloud storage",
	RunE:  runSyncList,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.AddCommand(syncUpCmd)
	syncCmd.AddCommand(syncDownCmd)
	syncCmd.AddCommand(syncListCmd)

	syncCmd.PersistentFlags().StringVar(&syncBucket, "bucket", "", "S3 bucket name")
	syncCmd.PersistentFlags().StringVar(&syncRegion, "region", "", "AWS region")
	syncCmd.PersistentFlags().StringVar(&syncEndpoint, "endpoint", "", "Custom S3 endpoint")
	syncCmd.PersistentFlags().StringVar(&syncPrefix, "prefix", "", "Path prefix in bucket")
	syncCmd.PersistentFlags().BoolVarP(&syncVerbose, "verbose", "v", false, "Show detailed output")
}

func getCloudProvider() (cloud.Provider, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}
	cfg.ExpandPaths()

	cloudCfg := cloud.Config{Provider: "s3"}

	if cfg.Cloud != nil {
		cloudCfg.Bucket = cfg.Cloud.Bucket
		cloudCfg.Region = cfg.Cloud.Region
		cloudCfg.Endpoint = cfg.Cloud.Endpoint
		cloudCfg.Prefix = cfg.Cloud.Prefix
	}

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

	if cloudCfg.Bucket == "" {
		return nil, nil, fmt.Errorf("bucket not configured (use --bucket or ~/.stash.yaml)")
	}
	if cloudCfg.Region == "" {
		return nil, nil, fmt.Errorf("region not configured (use --region or ~/.stash.yaml)")
	}

	provider, err := cloud.NewProvider(cloudCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cloud provider: %w", err)
	}

	return provider, cfg, nil
}

func runSyncUp(cmd *cobra.Command, args []string) error {
	ui.Verbose = syncVerbose

	provider, cfg, err := getCloudProvider()
	if err != nil {
		return err
	}

	if len(args) > 0 {
		// Upload specific file
		backupFile := args[0]
		if !filepath.IsAbs(backupFile) {
			backupFile = filepath.Join(cfg.BackupDir, backupFile)
		}

		if _, err := os.Stat(backupFile); err != nil {
			return fmt.Errorf("backup not found: %s", backupFile)
		}

		return uploadBackup(provider, backupFile)
	}

	// Upload all local backups
	entries, err := os.ReadDir(cfg.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup dir: %w", err)
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
		ui.PrintInfo("No backups to upload")
		return nil
	}

	uploaded := 0
	skipped := 0
	for _, backup := range backups {
		name := filepath.Base(backup)
		exists, err := provider.Exists(name)
		if err != nil {
			ui.PrintVerbose("Error checking %s: %v", name, err)
			continue
		}

		if exists {
			ui.PrintVerbose("Skipped %s (exists)", name)
			skipped++
			continue
		}

		if err := uploadBackup(provider, backup); err != nil {
			ui.PrintError("Failed: %s - %v", name, err)
		} else {
			uploaded++
		}
	}

	ui.PrintSuccess("Uploaded %d, skipped %d", uploaded, skipped)
	return nil
}

func uploadBackup(provider cloud.Provider, backupPath string) error {
	name := filepath.Base(backupPath)
	info, _ := os.Stat(backupPath)

	spinner := ui.NewSpinner(fmt.Sprintf("Uploading %s (%s)", name, metadata.FormatSize(info.Size())))
	spinner.Start()

	if err := provider.Upload(backupPath, name); err != nil {
		spinner.Fail()
		return err
	}

	spinner.Stop()
	return nil
}

func runSyncDown(cmd *cobra.Command, args []string) error {
	ui.Verbose = syncVerbose

	provider, cfg, err := getCloudProvider()
	if err != nil {
		return err
	}

	backupName := args[0]

	exists, err := provider.Exists(backupName)
	if err != nil {
		return fmt.Errorf("failed to check cloud: %w", err)
	}
	if !exists {
		return fmt.Errorf("not found in cloud: %s", backupName)
	}

	localPath := filepath.Join(cfg.BackupDir, backupName)

	if _, err := os.Stat(localPath); err == nil {
		ui.PrintWarning("Already exists locally: %s", localPath)
		return nil
	}

	spinner := ui.NewSpinner(fmt.Sprintf("Downloading %s", backupName))
	spinner.Start()

	if err := provider.Download(backupName, localPath); err != nil {
		spinner.Fail()
		return fmt.Errorf("download failed: %w", err)
	}

	spinner.Stop()
	ui.PrintDim("  Saved: %s", localPath)
	return nil
}

func runSyncList(cmd *cobra.Command, args []string) error {
	ui.Verbose = syncVerbose

	provider, _, err := getCloudProvider()
	if err != nil {
		return err
	}

	entries, err := provider.List("")
	if err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	if len(entries) == 0 {
		ui.PrintInfo("No backups in cloud")
		ui.PrintDim("  Upload: stash sync up")
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastModified.After(entries[j].LastModified)
	})

	// Build table
	headers := []string{"NAME", "SIZE", "DATE"}
	var rows [][]string
	for _, entry := range entries {
		name := entry.Name
		if len(name) > 35 {
			name = name[:32] + "..."
		}
		rows = append(rows, []string{
			name,
			metadata.FormatSize(entry.Size),
			entry.LastModified.Format("2006-01-02 15:04"),
		})
	}

	ui.PrintTable(headers, rows)
	fmt.Println()
	ui.PrintDim("%d backup(s) in %s", len(entries), provider.GetName())

	return nil
}

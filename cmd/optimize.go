package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harshpatel5940/stash/internal/archiver"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/harshpatel5940/stash/internal/incremental"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	optimizeDryRun    bool
	optimizeKeepChain bool
	optimizeOutput    string
)

var optimizeCmd = &cobra.Command{
	Use:   "optimize <backup-file>",
	Short: "Merge incremental backups into a full backup",
	Long: `Merges an incremental backup chain into a single full backup.

This command:
  1. Identifies the full backup chain (base + incrementals)
  2. Decrypts and extracts all backups in the chain
  3. Merges them into a single full backup
  4. Optionally removes the old incremental backups

This is useful for:
  - Reducing restore time (no need to process multiple backups)
  - Cleaning up after many incremental backups
  - Creating a portable single-file backup

Example:
  stash optimize backup-2024-01-15-120000.tar.gz.age

Options:
  --dry-run         Preview what would be done without making changes
  --keep-chain      Keep the original backup chain after optimization
  --output          Output directory for the optimized backup`,
	Args: cobra.ExactArgs(1),
	RunE: runOptimize,
}

func init() {
	rootCmd.AddCommand(optimizeCmd)
	optimizeCmd.Flags().BoolVar(&optimizeDryRun, "dry-run", false, "Preview optimization without making changes")
	optimizeCmd.Flags().BoolVar(&optimizeKeepChain, "keep-chain", false, "Keep original backup chain after optimization")
	optimizeCmd.Flags().StringVarP(&optimizeOutput, "output", "o", "", "Output directory for optimized backup")
}

func runOptimize(cmd *cobra.Command, args []string) error {
	backupFile := args[0]

	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		// Try checking in default backup directory
		if !filepath.IsAbs(backupFile) {
			homeDir, _ := os.UserHomeDir()
			altPath := filepath.Join(homeDir, "stash-backups", backupFile)
			if _, err := os.Stat(altPath); err == nil {
				backupFile = altPath
			} else {
				return fmt.Errorf("backup file not found: %s", args[0])
			}
		} else {
			return fmt.Errorf("backup file not found: %s", backupFile)
		}
	}

	if optimizeDryRun {
		fmt.Println("🔍 DRY RUN MODE - No files will be modified")
		fmt.Println()
	}

	fmt.Println("🔧 Starting backup optimization...")
	fmt.Println()

	// Get the restore chain
	chain, err := incremental.GetRestoreChain(backupFile)
	if err != nil {
		return fmt.Errorf("failed to get restore chain: %w", err)
	}

	// Validate chain
	if err := chain.Validate(); err != nil {
		return fmt.Errorf("backup chain validation failed: %w", err)
	}

	fmt.Printf("📚 Backup chain: %s\n", chain.Summary())
	fmt.Println()

	// If it's already a full backup with no incrementals, nothing to do
	if len(chain.IncrementalBackups) == 0 {
		fmt.Println("✓ This is already a full backup with no incrementals to merge")
		return nil
	}

	// Display chain details
	fmt.Println("📋 Backup chain details:")
	fmt.Printf("  Full backup:   %s\n", filepath.Base(chain.FullBackup))
	for i, incr := range chain.IncrementalBackups {
		fmt.Printf("  Incremental %d: %s\n", i+1, filepath.Base(incr))
	}
	fmt.Println()

	if optimizeDryRun {
		fmt.Println("🔍 Dry run summary:")
		fmt.Printf("  Would merge %d backup(s) into 1 full backup\n", chain.GetTotalBackups())
		fmt.Printf("  Output directory: %s\n", getOptimizeOutputDir(backupFile))
		if !optimizeKeepChain {
			fmt.Printf("  Would delete original chain after successful merge\n")
		}
		return nil
	}

	// Create temp directory for extraction
	tempDir, err := os.MkdirTemp("", "stash-optimize-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	extractDir := filepath.Join(tempDir, "merged")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	// Get encryption key path
	homeDir, _ := os.UserHomeDir()
	encryptionKey := filepath.Join(homeDir, ".stash.key")

	arch := archiver.NewArchiver()
	encryptor := crypto.NewEncryptor(encryptionKey)

	// Extract and merge all backups in the chain
	fmt.Println("📦 Extracting and merging backups...")
	for i, backupPath := range chain.GetBackupsInOrder() {
		fmt.Printf("  [%d/%d] Processing %s...\n", i+1, chain.GetTotalBackups(), filepath.Base(backupPath))

		// Decrypt if needed
		var archivePath string
		if strings.HasSuffix(backupPath, ".age") {
			archivePath = filepath.Join(tempDir, fmt.Sprintf("backup-%d.tar.gz", i))
			if err := encryptor.Decrypt(backupPath, archivePath); err != nil {
				return fmt.Errorf("failed to decrypt %s: %w", backupPath, err)
			}
		} else {
			archivePath = backupPath
		}

		// Extract (later backups override earlier ones)
		if err := arch.Extract(archivePath, extractDir); err != nil {
			return fmt.Errorf("failed to extract %s: %w", backupPath, err)
		}
	}
	fmt.Println("  ✓ All backups merged")

	// Update metadata to reflect this is now a full backup
	fmt.Println("\n📝 Updating metadata...")
	metadataPath := filepath.Join(extractDir, "metadata.json")
	meta, err := metadata.Load(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	// Update to full backup
	meta.SetBackupType("full")
	meta.SetBaseBackup("")
	meta.SetChangedFilesOnly(false)
	meta.Timestamp = time.Now()

	if err := meta.Save(metadataPath); err != nil {
		return fmt.Errorf("failed to save updated metadata: %w", err)
	}

	// Create new backup archive
	fmt.Println("\n📦 Creating optimized backup...")
	outputDir := getOptimizeOutputDir(backupFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02-150405")
	backupName := fmt.Sprintf("backup-%s-optimized", timestamp)
	archivePath := filepath.Join(outputDir, backupName+".tar.gz")

	if err := arch.Create(extractDir, archivePath); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	// Encrypt the new backup
	fmt.Println("🔐 Encrypting optimized backup...")
	encryptedPath := archivePath + ".age"
	if err := encryptor.Encrypt(archivePath, encryptedPath); err != nil {
		return fmt.Errorf("failed to encrypt backup: %w", err)
	}

	// Remove unencrypted archive
	os.Remove(archivePath)

	// Get file sizes
	fileInfo, _ := os.Stat(encryptedPath)
	var newSize int64
	if fileInfo != nil {
		newSize = fileInfo.Size()
	}

	// Calculate total size of original chain
	var originalSize int64
	for _, backupPath := range chain.GetBackupsInOrder() {
		if info, err := os.Stat(backupPath); err == nil {
			originalSize += info.Size()
		}
	}

	// Display results
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ Optimization completed!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("\n📁 Optimized backup: %s\n", encryptedPath)
	fmt.Printf("📊 Original chain:   %s (%d files)\n", ui.FormatBytes(originalSize), chain.GetTotalBackups())
	fmt.Printf("📦 Optimized backup: %s (1 file)\n", ui.FormatBytes(newSize))

	if newSize < originalSize {
		savings := float64(originalSize-newSize) / float64(originalSize) * 100
		fmt.Printf("💾 Space saved:      %s (%.1f%%)\n", ui.FormatBytes(originalSize-newSize), savings)
	}

	// Delete original chain if requested
	if !optimizeKeepChain {
		fmt.Println("\n🗑️  Deleting original backup chain...")
		deletedCount := 0
		for _, backupPath := range chain.GetBackupsInOrder() {
			if err := os.Remove(backupPath); err != nil {
				fmt.Printf("  ⚠️  Failed to delete %s: %v\n", filepath.Base(backupPath), err)
			} else {
				deletedCount++
			}
		}
		fmt.Printf("  ✓ Deleted %d backup file(s)\n", deletedCount)
	} else {
		fmt.Println("\n💡 Original backup chain preserved (use --keep-chain=false to delete)")
	}

	return nil
}

func getOptimizeOutputDir(backupFile string) string {
	if optimizeOutput != "" {
		return optimizeOutput
	}

	// Use the same directory as the input backup
	return filepath.Dir(backupFile)
}

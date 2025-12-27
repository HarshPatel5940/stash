package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harshpatel5940/stash/internal/archiver"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/spf13/cobra"
)

var (
	restoreDecryptKey  string
	restoreDryRun      bool
	restoreInteractive bool
	restoreNoDecrypt   bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore from a backup",
	Long: `Restores files from an encrypted backup to their original locations.

The restore process:
  1. Decrypts the backup file (if encrypted)
  2. Extracts the archive
  3. Reads metadata to find original file paths
  4. Restores files to their original locations

Use --dry-run to preview what would be restored without making changes.
Use --interactive to confirm each file before restoring.`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().StringVarP(&restoreDecryptKey, "decrypt-key", "k", "", "Path to decryption key (default: ~/.stash.key)")
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Preview what would be restored without making changes")
	restoreCmd.Flags().BoolVar(&restoreInteractive, "interactive", false, "Ask before restoring each file")
	restoreCmd.Flags().BoolVar(&restoreNoDecrypt, "no-decrypt", false, "Backup is not encrypted")
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupFile := args[0]

	// Check if backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupFile)
	}

	if restoreDryRun {
		fmt.Println("🔍 DRY RUN MODE - No files will be modified")
		fmt.Println()
	}

	fmt.Println("🔄 Starting restore process...")
	fmt.Println()

	// Set up decryption key path
	if restoreDecryptKey == "" {
		homeDir, _ := os.UserHomeDir()
		restoreDecryptKey = filepath.Join(homeDir, ".stash.key")
	}

	// Create temp directory for extraction
	tempDir, err := os.MkdirTemp("", "stash-restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var archivePath string

	// Decrypt if needed
	if restoreNoDecrypt {
		archivePath = backupFile
		fmt.Println("⚠️  Skipping decryption (--no-decrypt was used)")
	} else if strings.HasSuffix(backupFile, ".age") {
		fmt.Println("🔐 Decrypting backup...")

		encryptor := crypto.NewEncryptor(restoreDecryptKey)
		if !encryptor.KeyExists() {
			return fmt.Errorf("decryption key not found: %s", restoreDecryptKey)
		}

		archivePath = filepath.Join(tempDir, "backup.tar.gz")
		if err := encryptor.Decrypt(backupFile, archivePath); err != nil {
			return fmt.Errorf("failed to decrypt backup: %w", err)
		}
		fmt.Println("  ✓ Decryption successful")
	} else {
		archivePath = backupFile
		fmt.Println("⚠️  Backup does not appear to be encrypted (.age extension not found)")
	}

	// Extract archive
	fmt.Println("\n📦 Extracting backup...")
	extractDir := filepath.Join(tempDir, "extracted")
	arch := archiver.NewArchiver()

	if err := arch.Extract(archivePath, extractDir); err != nil {
		return fmt.Errorf("failed to extract backup: %w", err)
	}
	fmt.Println("  ✓ Extraction successful")

	// Load metadata
	fmt.Println("\n📋 Reading backup metadata...")
	metadataPath := filepath.Join(extractDir, "metadata.json")
	meta, err := metadata.Load(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	fmt.Printf("  Backup created: %s\n", meta.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Hostname: %s\n", meta.Hostname)
	fmt.Printf("  Username: %s\n", meta.Username)
	fmt.Printf("  Files: %d\n", len(meta.Files))

	// Show README if in dry-run mode
	readmePath := filepath.Join(extractDir, "README.txt")
	if restoreDryRun {
		if content, err := os.ReadFile(readmePath); err == nil {
			fmt.Println("\n📄 Backup README:")
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println(string(content))
			fmt.Println(strings.Repeat("-", 50))
		}
	}

	// Preview files to be restored
	fmt.Println("\n📂 Files to be restored:")
	fmt.Println(strings.Repeat("-", 80))

	fileCount := 0
	dirCount := 0
	skippedCount := 0

	for _, fileInfo := range meta.Files {
		if fileInfo.IsDir {
			dirCount++
			fmt.Printf("  [DIR]  %s\n", fileInfo.OriginalPath)
		} else {
			fileCount++
			fmt.Printf("  [FILE] %s (%s)\n", fileInfo.OriginalPath, metadata.FormatSize(fileInfo.Size))
		}
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Total: %d files, %d directories\n", fileCount, dirCount)

	if restoreDryRun {
		fmt.Println("\n✓ Dry run complete - no files were modified")
		return nil
	}

	// Confirm restore
	fmt.Println("\n⚠️  WARNING: This will restore files to their original locations!")
	fmt.Println("   Existing files may be overwritten.")
	fmt.Print("\nDo you want to continue? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Restore cancelled.")
		return nil
	}

	// Restore files
	fmt.Println("\n🔄 Restoring files...")

	for _, fileInfo := range meta.Files {
		backupFilePath := filepath.Join(extractDir, fileInfo.BackupPath)
		destPath := fileInfo.OriginalPath

		// Expand home directory if needed
		if strings.HasPrefix(destPath, "~") {
			homeDir, _ := os.UserHomeDir()
			destPath = filepath.Join(homeDir, destPath[1:])
		}

		// Interactive mode - ask for confirmation
		if restoreInteractive {
			fmt.Printf("\nRestore %s? [y/N]: ", fileInfo.OriginalPath)
			response, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("  ⚠️  Skipping %s: %v\n", fileInfo.OriginalPath, err)
				skippedCount++
				continue
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Printf("  ⊘ Skipped %s\n", fileInfo.OriginalPath)
				skippedCount++
				continue
			}
		}

		// Check if file exists
		if _, err := os.Stat(destPath); err == nil {
			if !restoreInteractive {
				// In non-interactive mode, ask once about overwrites
				fmt.Printf("  ⚠️  %s already exists\n", destPath)
				fmt.Print("     Overwrite? [y/N]: ")
				response, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("  ⊘ Skipped %s\n", fileInfo.OriginalPath)
					skippedCount++
					continue
				}

				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Printf("  ⊘ Skipped %s\n", fileInfo.OriginalPath)
					skippedCount++
					continue
				}
			}
		}

		// Restore based on type
		if fileInfo.IsDir {
			if err := arch.CopyDir(backupFilePath, destPath); err != nil {
				fmt.Printf("  ⚠️  Failed to restore directory %s: %v\n", fileInfo.OriginalPath, err)
				skippedCount++
				continue
			}
		} else {
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				fmt.Printf("  ⚠️  Failed to create parent directory for %s: %v\n", fileInfo.OriginalPath, err)
				skippedCount++
				continue
			}

			if err := arch.CopyFile(backupFilePath, destPath); err != nil {
				fmt.Printf("  ⚠️  Failed to restore %s: %v\n", fileInfo.OriginalPath, err)
				skippedCount++
				continue
			}

			// Restore permissions
			if err := os.Chmod(destPath, fileInfo.Mode); err != nil {
				fmt.Printf("  ⚠️  Failed to restore permissions for %s: %v\n", fileInfo.OriginalPath, err)
			}
		}

		fmt.Printf("  ✓ Restored %s\n", fileInfo.OriginalPath)
	}

	// Print summary
	successCount := len(meta.Files) - skippedCount
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ Restore completed!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("\n✓ Successfully restored: %d items\n", successCount)
	if skippedCount > 0 {
		fmt.Printf("⊘ Skipped: %d items\n", skippedCount)
	}

	// Show next steps
	fmt.Println("\n💡 Next steps:")
	fmt.Println("   1. Review restored files")
	fmt.Println("   2. Install packages from packages/ directory:")

	packagesDir := filepath.Join(extractDir, "packages")
	if _, err := os.Stat(filepath.Join(packagesDir, "Brewfile")); err == nil {
		fmt.Println("      - brew bundle --file=" + filepath.Join(packagesDir, "Brewfile"))
	}
	if _, err := os.Stat(filepath.Join(packagesDir, "vscode-extensions.txt")); err == nil {
		fmt.Println("      - cat " + filepath.Join(packagesDir, "vscode-extensions.txt") + " | xargs -L 1 code --install-extension")
	}
	fmt.Println("   3. Restart your terminal/shell")
	fmt.Println("   4. Test SSH connections and other credentials")

	return nil
}

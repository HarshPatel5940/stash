package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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
Use --interactive to pick/drop files in your editor (git-rebase style).`,
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

	// Interactive mode - let user pick/drop files
	filesToRestore := meta.Files
	if restoreInteractive {
		selected, err := interactivePickFiles(meta.Files, tempDir)
		if err != nil {
			return fmt.Errorf("interactive selection failed: %w", err)
		}
		if len(selected) == 0 {
			fmt.Println("No files selected. Restore cancelled.")
			return nil
		}
		filesToRestore = selected
		fmt.Printf("\n✓ Selected %d files to restore\n", len(filesToRestore))
	} else {
		// Non-interactive - confirm restore
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
	}

	// Restore files
	fmt.Println("\n🔄 Restoring files...")

	for _, fileInfo := range filesToRestore {
		backupFilePath := filepath.Join(extractDir, fileInfo.BackupPath)
		destPath := fileInfo.OriginalPath

		// Expand home directory if needed
		if strings.HasPrefix(destPath, "~") {
			homeDir, _ := os.UserHomeDir()
			destPath = filepath.Join(homeDir, destPath[1:])
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

func interactivePickFiles(files []metadata.FileInfo, tempDir string) ([]metadata.FileInfo, error) {
	// Create restore plan file
	planPath := filepath.Join(tempDir, "RESTORE_PLAN")

	var content strings.Builder
	content.WriteString("# Stash Restore Plan\n")
	content.WriteString("# \n")
	content.WriteString("# Commands:\n")
	content.WriteString("#   pick = restore this file\n")
	content.WriteString("#   drop = skip this file\n")
	content.WriteString("# \n")
	content.WriteString("# Lines starting with # are ignored\n")
	content.WriteString("#\n\n")

	for _, fileInfo := range files {
		fileType := "FILE"
		if fileInfo.IsDir {
			fileType = "DIR "
		}
		size := metadata.FormatSize(fileInfo.Size)
		content.WriteString(fmt.Sprintf("pick [%s] %s (%s)\n", fileType, fileInfo.OriginalPath, size))
	}

	if err := os.WriteFile(planPath, []byte(content.String()), 0644); err != nil {
		return nil, fmt.Errorf("failed to create restore plan: %w", err)
	}

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vim"
	}

	fmt.Println("\n📝 Opening restore plan in editor...")
	fmt.Printf("   Editor: %s\n", editor)
	fmt.Println("   Change 'pick' to 'drop' to skip files")
	fmt.Println("   Save and close when done")

	// Open editor
	cmd := exec.Command(editor, planPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor failed: %w", err)
	}

	// Parse edited file
	planContent, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read restore plan: %w", err)
	}

	// Build map of original paths to file info
	fileMap := make(map[string]metadata.FileInfo)
	for _, f := range files {
		fileMap[f.OriginalPath] = f
	}

	// Parse selections
	var selected []metadata.FileInfo
	scanner := bufio.NewScanner(strings.NewReader(string(planContent)))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line: "pick [TYPE] /path/to/file (size)"
		parts := strings.Fields(line)
		if len(parts) < 3 {
			fmt.Printf("⚠️  Warning: skipping malformed line %d: %s\n", lineNum, line)
			continue
		}

		action := parts[0]
		if action != "pick" && action != "drop" {
			fmt.Printf("⚠️  Warning: unknown action '%s' on line %d, treating as 'drop'\n", action, lineNum)
			continue
		}

		// Extract path - everything between "]" and "("
		restOfLine := strings.Join(parts[1:], " ")
		startIdx := strings.Index(restOfLine, "]")
		endIdx := strings.LastIndex(restOfLine, "(")

		if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
			fmt.Printf("⚠️  Warning: couldn't parse path on line %d\n", lineNum)
			continue
		}

		path := strings.TrimSpace(restOfLine[startIdx+1 : endIdx])

		if action == "pick" {
			if fileInfo, ok := fileMap[path]; ok {
				selected = append(selected, fileInfo)
			} else {
				fmt.Printf("⚠️  Warning: file not found in backup: %s\n", path)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse restore plan: %w", err)
	}

	return selected, nil
}

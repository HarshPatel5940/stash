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
	"github.com/harshpatel5940/stash/internal/defaults"
	"github.com/harshpatel5940/stash/internal/incremental"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/packager"
	"github.com/harshpatel5940/stash/internal/tui"
	"github.com/spf13/cobra"
)

var (
	restoreDecryptKey string
	restoreDryRun     bool
	restoreEditor     bool
	restoreNoDecrypt  bool
	restoreNoTUI      bool
)

type RestoreOptions struct {
	RestoreFiles         bool
	RestoreMacOSDefaults bool
	InstallHomebrew      bool
	InstallMAS           bool
	InstallVSCode        bool
	InstallNPM           bool
	RestoreShellHistory  bool
}

var restoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore from a backup",
	Long: `Restores files from an encrypted backup to their original locations.

The restore process:
  1. Decrypts the backup file (if encrypted)
  2. Extracts the archive
  3. Interactive multi-select menu to choose what to restore/install
  4. Optional file-by-file selection
  5. Executes selected actions automatically:
     - Restore files (dotfiles, SSH, GPG, configs)
     - Restore macOS system preferences
     - Install Homebrew packages
     - Install Mac App Store apps
     - Install VS Code extensions
     - Install NPM global packages

Use --dry-run to preview what would be restored without making changes.
Use --editor to pick/drop individual files in your editor (git-rebase style).
Use --no-tui for simple Y/n prompts instead of interactive multi-select.`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().StringVarP(&restoreDecryptKey, "decrypt-key", "k", "", "Path to decryption key (default: ~/.stash.key)")
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Preview what would be restored without making changes")
	restoreCmd.Flags().BoolVar(&restoreEditor, "editor", false, "Pick/drop individual files in your editor (git-rebase style)")
	restoreCmd.Flags().BoolVar(&restoreNoDecrypt, "no-decrypt", false, "Backup is not encrypted")
	restoreCmd.Flags().BoolVar(&restoreNoTUI, "no-tui", false, "Use simple Y/n prompts instead of interactive multi-select")
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupFile := args[0]

	if _, err := os.Stat(backupFile); os.IsNotExist(err) {

		if !filepath.IsAbs(backupFile) {
			homeDir, _ := os.UserHomeDir()
			altPath := filepath.Join(homeDir, "stash-backups", backupFile)
			if _, err := os.Stat(altPath); err == nil {
				backupFile = altPath
				fmt.Printf("📂 Using backup from: %s\n", altPath)
			} else {
				return fmt.Errorf("backup file not found: %s (also checked %s)", args[0], altPath)
			}
		} else {
			return fmt.Errorf("backup file not found: %s", backupFile)
		}
	}

	if restoreDryRun {
		fmt.Println("🔍 DRY RUN MODE - No files will be modified")
		fmt.Println()
	}

	fmt.Println("🔄 Starting restore process...")
	fmt.Println()

	if restoreDecryptKey == "" {
		homeDir, _ := os.UserHomeDir()
		restoreDecryptKey = filepath.Join(homeDir, ".stash.key")
	}

	tempDir, err := os.MkdirTemp("", "stash-restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var archivePath string

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

	fmt.Println("\n📦 Extracting backup...")
	extractDir := filepath.Join(tempDir, "extracted")
	arch := archiver.NewArchiver()

	if err := arch.Extract(archivePath, extractDir); err != nil {
		return fmt.Errorf("failed to extract backup: %w", err)
	}
	fmt.Println("  ✓ Extraction successful")

	fmt.Println("\n📋 Reading backup metadata...")
	metadataPath := filepath.Join(extractDir, "metadata.json")
	meta, err := metadata.Load(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	fmt.Printf("  Backup created: %s\n", meta.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Hostname: %s\n", meta.Hostname)
	fmt.Printf("  Username: %s\n", meta.Username)

	// Check if this is an incremental backup
	if meta.IsIncremental() {
		fmt.Printf("  Backup type: Incremental (requires base backup)\n")
		fmt.Printf("  Base backup: %s\n", meta.BaseBackup)

		// Get the restore chain
		chain, err := incremental.GetRestoreChain(backupFile)
		if err != nil {
			return fmt.Errorf("failed to resolve incremental backup chain: %w", err)
		}

		if err := chain.Validate(); err != nil {
			return fmt.Errorf("backup chain validation failed: %w", err)
		}

		fmt.Printf("\n📚 Restore chain: %s\n", chain.Summary())

		// Extract and merge all backups in the chain
		fmt.Println("\n📦 Extracting backup chain...")

		for i, backupPath := range chain.GetBackupsInOrder() {
			fmt.Printf("  [%d/%d] Extracting %s...\n", i+1, chain.GetTotalBackups(), filepath.Base(backupPath))

			// Decrypt if needed
			var chainArchivePath string
			if strings.HasSuffix(backupPath, ".age") {
				encryptor := crypto.NewEncryptor(restoreDecryptKey)
				chainArchivePath = filepath.Join(tempDir, fmt.Sprintf("backup-%d.tar.gz", i))
				if err := encryptor.Decrypt(backupPath, chainArchivePath); err != nil {
					return fmt.Errorf("failed to decrypt backup %s: %w", backupPath, err)
				}
			} else {
				chainArchivePath = backupPath
			}

			// Extract to the same directory (later backups override earlier ones)
			if err := arch.Extract(chainArchivePath, extractDir); err != nil {
				return fmt.Errorf("failed to extract backup %s: %w", backupPath, err)
			}
		}

		fmt.Println("  ✓ All backups in chain extracted and merged")

		// Reload metadata from the final (incremental) backup
		meta, err = metadata.Load(metadataPath)
		if err != nil {
			return fmt.Errorf("failed to reload metadata: %w", err)
		}
	} else {
		fmt.Printf("  Backup type: Full\n")
	}

	fmt.Printf("  Files: %d\n", len(meta.Files))

	packagesDir := filepath.Join(extractDir, "packages")
	macosDefaultsFile := filepath.Join(extractDir, "macos-defaults", "macos-defaults.json")

	hasBrewfile := fileExists(filepath.Join(packagesDir, "Brewfile"))
	hasMAS := fileExists(filepath.Join(packagesDir, "mas-apps.txt"))
	hasVSCode := fileExists(filepath.Join(packagesDir, "vscode-extensions.txt"))
	hasNPM := fileExists(filepath.Join(packagesDir, "npm-global.txt"))
	hasMacOSDefaults := fileExists(macosDefaultsFile)
	hasShellHistory := fileExists(filepath.Join(extractDir, "shell-history"))

	var options RestoreOptions
	if !restoreDryRun {
		available := tui.AvailableOptions{
			HasBrewfile:      hasBrewfile,
			HasMAS:           hasMAS,
			HasVSCode:        hasVSCode,
			HasNPM:           hasNPM,
			HasMacOSDefaults: hasMacOSDefaults,
			HasShellHistory:  hasShellHistory,
		}

		if restoreNoTUI {
			// Use simple Y/n prompts
			var err error
			options, err = promptRestoreOptions(hasBrewfile, hasMAS, hasVSCode, hasNPM, hasMacOSDefaults, hasShellHistory)
			if err != nil {
				return fmt.Errorf("failed to get restore options: %w", err)
			}
		} else {
			// Use TUI multi-select
			tuiOpts, err := tui.RestoreOptionsForm(available)
			if err != nil {
				return fmt.Errorf("failed to get restore options: %w", err)
			}
			options = RestoreOptions{
				RestoreFiles:         tuiOpts.RestoreFiles,
				RestoreMacOSDefaults: tuiOpts.RestoreMacOSDefaults,
				InstallHomebrew:      tuiOpts.InstallHomebrew,
				InstallMAS:           tuiOpts.InstallMAS,
				InstallVSCode:        tuiOpts.InstallVSCode,
				InstallNPM:           tuiOpts.InstallNPM,
				RestoreShellHistory:  tuiOpts.RestoreShellHistory,
			}
		}
	} else {
		// Dry run - use default options
		options = RestoreOptions{
			RestoreFiles:         true,
			RestoreMacOSDefaults: hasMacOSDefaults,
			InstallHomebrew:      hasBrewfile,
			InstallMAS:           hasMAS,
			InstallVSCode:        hasVSCode,
			InstallNPM:           hasNPM,
			RestoreShellHistory:  hasShellHistory,
		}
	}

	readmePath := filepath.Join(extractDir, "README.txt")
	if restoreDryRun {
		if content, err := os.ReadFile(readmePath); err == nil {
			fmt.Println("\n📄 Backup README:")
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println(string(content))
			fmt.Println(strings.Repeat("-", 50))
		}
	}

	fmt.Println("\n📂 Files to be restored:")
	fmt.Println(strings.Repeat("-", 80))

	fileCount := 0
	dirCount := 0

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

	filesToRestore := meta.Files
	if restoreEditor {
		// Use editor-based pick/drop selection
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
	} else if !restoreNoTUI && !restoreDryRun {
		// Use TUI multi-select for file selection (only for smaller backups)
		if len(meta.Files) <= 100 {
			selected, err := tui.FilePickerForm(meta.Files)
			if err != nil {
				return fmt.Errorf("file selection failed: %w", err)
			}
			if len(selected) == 0 {
				fmt.Println("No files selected. Restore cancelled.")
				return nil
			}
			filesToRestore = selected
			fmt.Printf("\n✓ Selected %d files to restore\n", len(filesToRestore))
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🚀 Executing restore actions...")
	fmt.Println(strings.Repeat("=", 50))

	successCount := 0
	skippedCount := 0

	if options.RestoreFiles {
		fmt.Println("\n🔄 Restoring files...")

		for _, fileInfo := range filesToRestore {
			backupFilePath := filepath.Join(extractDir, fileInfo.BackupPath)
			destPath := fileInfo.OriginalPath

			if strings.HasPrefix(destPath, "~") {
				homeDir, _ := os.UserHomeDir()
				destPath = filepath.Join(homeDir, destPath[1:])
			}

			if fileInfo.IsDir {
				if err := arch.CopyDir(backupFilePath, destPath); err != nil {
					fmt.Printf("  ⚠️  Failed to restore directory %s: %v\n", fileInfo.OriginalPath, err)
					skippedCount++
					continue
				}
			} else {

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

				if err := os.Chmod(destPath, fileInfo.Mode); err != nil {
					fmt.Printf("  ⚠️  Failed to restore permissions for %s: %v\n", fileInfo.OriginalPath, err)
				}
			}

			fmt.Printf("  ✓ Restored %s\n", fileInfo.OriginalPath)
		}

		successCount = len(filesToRestore) - skippedCount
		fmt.Printf("\n✓ Successfully restored: %d files\n", successCount)
		if skippedCount > 0 {
			fmt.Printf("⊘ Skipped: %d items\n", skippedCount)
		}
	}

	homeDir, _ := os.UserHomeDir()
	stashBackupsDir := filepath.Join(homeDir, "stash-backups")
	persistentPackagesDir := filepath.Join(stashBackupsDir, "packages")

	if _, err := os.Stat(packagesDir); err == nil {
		fmt.Println("\n📦 Copying packages to persistent location...")

		if err := os.MkdirAll(stashBackupsDir, 0755); err != nil {
			fmt.Printf("  ⚠️  Failed to create %s: %v\n", stashBackupsDir, err)
		} else {

			os.RemoveAll(persistentPackagesDir)

			arch := archiver.NewArchiver()
			if err := arch.CopyDir(packagesDir, persistentPackagesDir); err != nil {
				fmt.Printf("  ⚠️  Failed to copy packages: %v\n", err)
			} else {
				fmt.Printf("  ✓ Packages saved to %s\n", persistentPackagesDir)
			}
		}
	}

	if options.RestoreMacOSDefaults && fileExists(macosDefaultsFile) {
		fmt.Println("\n🔧 Restoring macOS defaults...")
		dm := defaults.NewDefaultsManager("")
		if err := dm.RestoreAll(macosDefaultsFile); err != nil {
			fmt.Printf("  ⚠️  Failed to restore macOS defaults: %v\n", err)
		}
	}

	// Create installer with progress bars
	installer := packager.NewInstaller(false)

	if options.InstallHomebrew && fileExists(filepath.Join(persistentPackagesDir, "Brewfile")) {
		fmt.Println("\n🍺 Installing Homebrew packages...")
		if err := installer.InstallBrewPackages(filepath.Join(persistentPackagesDir, "Brewfile")); err != nil {
			fmt.Printf("  ⚠️  Failed to install Homebrew packages: %v\n", err)
			fmt.Println("  💡 Run manually: brew bundle --file=" + filepath.Join(persistentPackagesDir, "Brewfile"))
		} else {
			fmt.Println("  ✓ Homebrew packages installed")
		}
	}

	if options.InstallMAS && fileExists(filepath.Join(persistentPackagesDir, "mas-apps.txt")) {
		fmt.Println("\n🏪 Installing Mac App Store apps...")
		installed, err := installer.InstallMASApps(filepath.Join(persistentPackagesDir, "mas-apps.txt"))
		if err != nil {
			fmt.Printf("  ⚠️  %v\n", err)
		} else {
			fmt.Printf("  ✓ Installed %d Mac App Store apps\n", installed)
		}
	}

	if options.InstallVSCode && fileExists(filepath.Join(persistentPackagesDir, "vscode-extensions.txt")) {
		fmt.Println("\n💻 Installing VS Code extensions...")
		installed, err := installer.InstallVSCodeExtensions(filepath.Join(persistentPackagesDir, "vscode-extensions.txt"))
		if err != nil {
			fmt.Printf("  ⚠️  %v\n", err)
		} else {
			fmt.Printf("  ✓ Installed %d VS Code extensions\n", installed)
		}
	}

	if options.InstallNPM && fileExists(filepath.Join(persistentPackagesDir, "npm-global.txt")) {
		fmt.Println("\n📦 NPM global packages...")
		if err := installer.InstallNPMPackages(filepath.Join(persistentPackagesDir, "npm-global.txt")); err != nil {
			fmt.Printf("  ⚠️  %v\n", err)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ Restore completed!")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Println("\n💡 Next steps:")
	if fileExists(filepath.Join(persistentPackagesDir, "non-brew-apps.txt")) {
		fmt.Println("   • Review non-Homebrew apps: cat " + filepath.Join(persistentPackagesDir, "non-brew-apps.txt"))
	}
	if options.RestoreMacOSDefaults {
		fmt.Println("   • Logout/restart for macOS defaults to fully take effect")
		fmt.Println("   • Or run: killall Dock Finder SystemUIServer")
	}
	if options.RestoreShellHistory {
		fmt.Println("   • Restart terminal to load shell history")
	}
	fmt.Println("   • Test SSH connections and other credentials")

	return nil
}

func interactivePickFiles(files []metadata.FileInfo, tempDir string) ([]metadata.FileInfo, error) {

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

	cmd := exec.Command(editor, planPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor failed: %w", err)
	}

	planContent, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read restore plan: %w", err)
	}

	fileMap := make(map[string]metadata.FileInfo)
	for _, f := range files {
		fileMap[f.OriginalPath] = f
	}

	var selected []metadata.FileInfo
	scanner := bufio.NewScanner(strings.NewReader(string(planContent)))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

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

func promptRestoreOptions(hasBrewfile, hasMAS, hasVSCode, hasNPM, hasMacOSDefaults, hasShellHistory bool) (RestoreOptions, error) {
	reader := bufio.NewReader(os.Stdin)
	options := RestoreOptions{}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🎯 Restore Options")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\nSelect what to restore/install:")

	options.RestoreFiles = true
	fmt.Println("\n✓ Files (dotfiles, SSH, GPG, configs, etc.) - Always included")

	if hasMacOSDefaults {
		fmt.Print("\n🔧 Restore macOS defaults (Dock, Finder, trackpad, etc.)? [Y/n]: ")
		response, _ := reader.ReadString('\n')
		options.RestoreMacOSDefaults = !strings.EqualFold(strings.TrimSpace(response), "n")
	}

	if hasShellHistory {
		fmt.Print("\n📜 Restore shell history? [Y/n]: ")
		response, _ := reader.ReadString('\n')
		options.RestoreShellHistory = !strings.EqualFold(strings.TrimSpace(response), "n")
	}

	if hasBrewfile {
		fmt.Print("\n🍺 Install Homebrew packages (this may take a while)? [Y/n]: ")
		response, _ := reader.ReadString('\n')
		options.InstallHomebrew = !strings.EqualFold(strings.TrimSpace(response), "n")
	}

	if hasMAS {
		fmt.Print("\n🏪 Install Mac App Store apps? [y/N]: ")
		response, _ := reader.ReadString('\n')
		options.InstallMAS = strings.EqualFold(strings.TrimSpace(response), "y") || strings.EqualFold(strings.TrimSpace(response), "yes")
	}

	if hasVSCode {
		fmt.Print("\n💻 Install VS Code extensions? [Y/n]: ")
		response, _ := reader.ReadString('\n')
		options.InstallVSCode = !strings.EqualFold(strings.TrimSpace(response), "n")
	}

	if hasNPM {
		fmt.Print("\n📦 Install NPM global packages? [y/N]: ")
		response, _ := reader.ReadString('\n')
		options.InstallNPM = strings.EqualFold(strings.TrimSpace(response), "y") || strings.EqualFold(strings.TrimSpace(response), "yes")
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	return options, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

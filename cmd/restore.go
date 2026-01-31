package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harshpatel5940/stash/internal/archiver"
	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/harshpatel5940/stash/internal/defaults"
	"github.com/harshpatel5940/stash/internal/incremental"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/packager"
	"github.com/harshpatel5940/stash/internal/tui"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	restoreDecryptKey string
	restoreDryRun     bool
	restoreEditor     bool
	restoreNoDecrypt  bool
	restoreNoTUI      bool
	restoreVerbose    bool
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
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Preview what would be restored")
	restoreCmd.Flags().BoolVar(&restoreEditor, "editor", false, "Pick files in editor (git-rebase style)")
	restoreCmd.Flags().BoolVar(&restoreNoDecrypt, "no-decrypt", false, "Skip decryption")
	restoreCmd.Flags().BoolVar(&restoreNoTUI, "no-tui", false, "Use Y/n prompts instead of TUI")
	restoreCmd.Flags().BoolVarP(&restoreVerbose, "verbose", "v", false, "Show detailed output")
}

func runRestore(cmd *cobra.Command, args []string) error {
	ui.Verbose = restoreVerbose
	backupFile := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		if !filepath.IsAbs(backupFile) {
			homeDir, _ := os.UserHomeDir()
			altPath := filepath.Join(homeDir, "stash-backups", backupFile)
			if _, err := os.Stat(altPath); err == nil {
				backupFile = altPath
				ui.PrintVerbose("Using: %s", altPath)
			} else {
				return fmt.Errorf("backup not found: %s", args[0])
			}
		} else {
			return fmt.Errorf("backup not found: %s", backupFile)
		}
	}

	if restoreDryRun {
		ui.PrintInfo("DRY RUN - No changes will be made")
	}

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
		ui.PrintVerbose("Skipping decryption")
	} else if strings.HasSuffix(backupFile, ".age") {
		ui.PrintVerbose("Decrypting...")
		encryptor := crypto.NewEncryptor(restoreDecryptKey)
		if !encryptor.KeyExists() {
			return fmt.Errorf("decryption key not found: %s", restoreDecryptKey)
		}

		archivePath = filepath.Join(tempDir, "backup.tar.gz")
		if err := encryptor.Decrypt(backupFile, archivePath); err != nil {
			return fmt.Errorf("failed to decrypt: %w", err)
		}
	} else {
		archivePath = backupFile
		ui.PrintVerbose("Not encrypted")
	}

	ui.PrintVerbose("Extracting...")
	extractDir := filepath.Join(tempDir, "extracted")
	arch := archiver.NewArchiver()

	if err := arch.Extract(archivePath, extractDir); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	metadataPath := filepath.Join(extractDir, "metadata.json")
	meta, err := metadata.Load(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	ui.PrintVerbose("Backup: %s (%s)", meta.Timestamp.Format("2006-01-02"), meta.Hostname)

	// Check if this is an incremental backup
	if meta.IsIncremental() {
		ui.PrintVerbose("Incremental backup, resolving chain...")

		chain, err := incremental.GetRestoreChain(backupFile)
		if err != nil {
			return fmt.Errorf("failed to resolve backup chain: %w", err)
		}

		if err := chain.Validate(); err != nil {
			return fmt.Errorf("backup chain validation failed: %w", err)
		}

		ui.PrintVerbose("Chain: %s", chain.Summary())

		// Extract and merge all backups in the chain
		for i, backupPath := range chain.GetBackupsInOrder() {
			ui.PrintVerbose("Extracting %d/%d: %s", i+1, chain.GetTotalBackups(), filepath.Base(backupPath))

			var chainArchivePath string
			if strings.HasSuffix(backupPath, ".age") {
				encryptor := crypto.NewEncryptor(restoreDecryptKey)
				chainArchivePath = filepath.Join(tempDir, fmt.Sprintf("backup-%d.tar.gz", i))
				if err := encryptor.Decrypt(backupPath, chainArchivePath); err != nil {
					return fmt.Errorf("failed to decrypt %s: %w", filepath.Base(backupPath), err)
				}
			} else {
				chainArchivePath = backupPath
			}

			if err := arch.Extract(chainArchivePath, extractDir); err != nil {
				return fmt.Errorf("failed to extract %s: %w", filepath.Base(backupPath), err)
			}
		}

		meta, err = metadata.Load(metadataPath)
		if err != nil {
			return fmt.Errorf("failed to reload metadata: %w", err)
		}
	}

	ui.PrintVerbose("Files: %d", len(meta.Files))

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

	// Dry run: show summary and exit
	if restoreDryRun {
		fileCount := 0
		for _, f := range meta.Files {
			if !f.IsDir {
				fileCount++
			}
		}
		ui.PrintInfo("DRY RUN: Would restore %d files", fileCount)
		if restoreVerbose {
			for _, f := range meta.Files {
				fmt.Printf("  %s\n", f.OriginalPath)
			}
		}
		return nil
	}

	filesToRestore := meta.Files
	if restoreEditor {
		// Interactive editor mode - pick files AND packages/actions
		selected, editorOptions, err := interactivePickAll(meta.Files, tempDir, hasBrewfile, hasMAS, hasVSCode, hasNPM, hasMacOSDefaults, hasShellHistory)
		if err != nil {
			return fmt.Errorf("interactive selection failed: %w", err)
		}
		if len(selected) == 0 && editorOptions.RestoreFiles {
			ui.PrintInfo("No files selected")
			return nil
		}
		filesToRestore = selected
		// Override options from editor
		options = editorOptions
	} else if !restoreNoTUI && options.RestoreFiles {
		// Use TUI multi-select for file selection (only for smaller backups and if user chose to restore files)
		if len(meta.Files) <= cfg.GetRestoreFilePickerThreshold() {
			selected, err := tui.FilePickerForm(meta.Files)
			if err != nil {
				return fmt.Errorf("file selection failed: %w", err)
			}
			if len(selected) == 0 {
				ui.PrintInfo("No files selected, but other restore options will continue")
			} else {
				filesToRestore = selected
			}
		}
	}

	// Start restore
	ui.PrintVerbose("Restoring %d files...", len(filesToRestore))

	successCount := 0
	skippedCount := 0

	if options.RestoreFiles {
		for _, fileInfo := range filesToRestore {
			backupFilePath := filepath.Join(extractDir, fileInfo.BackupPath)
			destPath := fileInfo.OriginalPath

			if strings.HasPrefix(destPath, "~") {
				homeDir, _ := os.UserHomeDir()
				destPath = filepath.Join(homeDir, destPath[1:])
			}

			if fileInfo.IsDir {
				if err := arch.CopyDir(backupFilePath, destPath); err != nil {
					ui.PrintVerbose("Failed: %s - %v", fileInfo.OriginalPath, err)
					skippedCount++
					continue
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					ui.PrintVerbose("Failed to create dir for %s", fileInfo.OriginalPath)
					skippedCount++
					continue
				}

				if err := arch.CopyFile(backupFilePath, destPath); err != nil {
					ui.PrintVerbose("Failed: %s - %v", fileInfo.OriginalPath, err)
					skippedCount++
					continue
				}

				_ = os.Chmod(destPath, fileInfo.Mode)
			}

			ui.PrintVerbose("Restored: %s", fileInfo.OriginalPath)
		}

		successCount = len(filesToRestore) - skippedCount
	}

	homeDir, _ := os.UserHomeDir()
	stashBackupsDir := filepath.Join(homeDir, "stash-backups")
	persistentPackagesDir := filepath.Join(stashBackupsDir, "packages")

	if _, err := os.Stat(packagesDir); err == nil {
		_ = os.MkdirAll(stashBackupsDir, 0755)
		os.RemoveAll(persistentPackagesDir)
		arch := archiver.NewArchiver()
		_ = arch.CopyDir(packagesDir, persistentPackagesDir)
	}

	if options.RestoreMacOSDefaults && fileExists(macosDefaultsFile) {
		ui.PrintVerbose("Restoring macOS defaults...")
		dm := defaults.NewDefaultsManager("")
		if err := dm.RestoreAll(macosDefaultsFile); err != nil {
			ui.PrintVerbose("macOS defaults failed: %v", err)
		}
	}

	installer := packager.NewInstaller(false)

	if options.InstallHomebrew && fileExists(filepath.Join(persistentPackagesDir, "Brewfile")) {
		brewfilePath := filepath.Join(persistentPackagesDir, "Brewfile")
		
		// Parse Brewfile into individual items
		items, err := packager.ParseBrewfile(brewfilePath)
		if err != nil {
			ui.PrintWarning("Failed to parse Brewfile: %v", err)
		} else if !restoreNoTUI && len(items) > 0 {
			// Show package picker if not in no-TUI mode
			var tuiItems []tui.BrewPackageItem
			for _, item := range items {
				tuiItems = append(tuiItems, tui.BrewPackageItem{
					Type:    item.Type,
					Name:    item.Name,
					Label:   packager.FormatBrewfileItem(item),
					RawLine: item.RawLine,
				})
			}
			
			selectedItems, err := tui.BrewPackagePickerForm(tuiItems)
			if err != nil {
				ui.PrintWarning("Package selection failed: %v", err)
			} else if len(selectedItems) == 0 {
				ui.PrintInfo("No packages selected, skipping Homebrew installation")
			} else {
				// Create filtered Brewfile with selected packages
				var filteredItems []packager.BrewfileItem
				for _, tuiItem := range selectedItems {
					filteredItems = append(filteredItems, packager.BrewfileItem{
						Type:    tuiItem.Type,
						Name:    tuiItem.Name,
						RawLine: tuiItem.RawLine,
					})
				}
				
				tempBrewfile := filepath.Join(tempDir, "Brewfile.filtered")
				if err := packager.CreateFilteredBrewfile(filteredItems, tempBrewfile); err != nil {
					ui.PrintWarning("Failed to create filtered Brewfile: %v", err)
				} else {
					ui.PrintVerbose("Installing %d selected packages...", len(selectedItems))
					if err := installer.InstallBrewPackages(tempBrewfile); err != nil {
						ui.PrintWarning("Homebrew failed: %v", err)
					}
				}
			}
		} else {
			// Install all packages (no-TUI mode or parse error)
			ui.PrintVerbose("Installing Homebrew packages...")
			if err := installer.InstallBrewPackages(brewfilePath); err != nil {
				ui.PrintWarning("Homebrew failed: %v", err)
			}
		}
	}

	if options.InstallMAS && fileExists(filepath.Join(persistentPackagesDir, "mas-apps.txt")) {
		ui.PrintVerbose("Installing Mac App Store apps...")
		_, _ = installer.InstallMASApps(filepath.Join(persistentPackagesDir, "mas-apps.txt"))
	}

	if options.InstallVSCode && fileExists(filepath.Join(persistentPackagesDir, "vscode-extensions.txt")) {
		ui.PrintVerbose("Installing VS Code extensions...")
		_, _ = installer.InstallVSCodeExtensions(filepath.Join(persistentPackagesDir, "vscode-extensions.txt"))
	}

	if options.InstallNPM && fileExists(filepath.Join(persistentPackagesDir, "npm-global.txt")) {
		ui.PrintVerbose("Installing NPM packages...")
		_ = installer.InstallNPMPackages(filepath.Join(persistentPackagesDir, "npm-global.txt"))
	}

	// Final output
	ui.PrintSuccess("Restored %d files", successCount)
	if skippedCount > 0 {
		ui.PrintDim("  Skipped: %d", skippedCount)
	}

	return nil
}

func interactivePickAll(files []metadata.FileInfo, tempDir string, hasBrewfile, hasMAS, hasVSCode, hasNPM, hasMacOSDefaults, hasShellHistory bool) ([]metadata.FileInfo, RestoreOptions, error) {
	planPath := filepath.Join(tempDir, "RESTORE_PLAN")

	var content strings.Builder
	content.WriteString("# Stash Restore Plan\n")
	content.WriteString("# \n")
	content.WriteString("# Commands:\n")
	content.WriteString("#   pick = restore/install this item\n")
	content.WriteString("#   drop = skip this item\n")
	content.WriteString("# \n")
	content.WriteString("# Lines starting with # are ignored\n")
	content.WriteString("#\n\n")

	// Add package installation options
	content.WriteString("# === PACKAGES & SETTINGS ===\n\n")
	
	if hasBrewfile {
		content.WriteString("pick [BREW] Install Homebrew packages (may take a while)\n")
	}
	if hasMAS {
		content.WriteString("drop [MAS ] Install Mac App Store apps\n")
	}
	if hasVSCode {
		content.WriteString("pick [CODE] Install VS Code extensions\n")
	}
	if hasNPM {
		content.WriteString("drop [NPM ] Install NPM global packages\n")
	}
	if hasMacOSDefaults {
		content.WriteString("pick [PREF] Restore macOS defaults (Dock, Finder, etc.)\n")
	}
	if hasShellHistory {
		content.WriteString("pick [HIST] Restore shell history\n")
	}

	content.WriteString("\n# === FILES & DIRECTORIES ===\n\n")

	for _, fileInfo := range files {
		fileType := "FILE"
		if fileInfo.IsDir {
			fileType = "DIR "
		}
		size := metadata.FormatSize(fileInfo.Size)
		content.WriteString(fmt.Sprintf("pick [%s] %s (%s)\n", fileType, fileInfo.OriginalPath, size))
	}

	if err := os.WriteFile(planPath, []byte(content.String()), 0644); err != nil {
		return nil, RestoreOptions{}, fmt.Errorf("failed to create restore plan: %w", err)
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
	fmt.Println("   Change 'pick' to 'drop' to skip items")
	fmt.Println("   Save and close when done")

	cmd := exec.Command(editor, planPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, RestoreOptions{}, fmt.Errorf("editor failed: %w", err)
	}

	planContent, err := os.ReadFile(planPath)
	if err != nil {
		return nil, RestoreOptions{}, fmt.Errorf("failed to read restore plan: %w", err)
	}

	fileMap := make(map[string]metadata.FileInfo)
	for _, f := range files {
		fileMap[f.OriginalPath] = f
	}

	var selected []metadata.FileInfo
	options := RestoreOptions{RestoreFiles: true}
	
	scanner := bufio.NewScanner(strings.NewReader(string(planContent)))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			fmt.Printf("⚠️  Warning: skipping malformed line %d: %s\n", lineNum, line)
			continue
		}

		action := parts[0]
		if action != "pick" && action != "drop" {
			fmt.Printf("⚠️  Warning: unknown action '%s' on line %d, treating as 'drop'\n", action, lineNum)
			continue
		}

		itemType := strings.Trim(parts[1], "[]")
		
		// Handle package/settings items
		switch itemType {
		case "BREW":
			options.InstallHomebrew = (action == "pick")
			continue
		case "MAS":
			options.InstallMAS = (action == "pick")
			continue
		case "CODE":
			options.InstallVSCode = (action == "pick")
			continue
		case "NPM":
			options.InstallNPM = (action == "pick")
			continue
		case "PREF":
			options.RestoreMacOSDefaults = (action == "pick")
			continue
		case "HIST":
			options.RestoreShellHistory = (action == "pick")
			continue
		}

		// Handle file items
		if len(parts) < 3 {
			fmt.Printf("⚠️  Warning: skipping malformed file line %d: %s\n", lineNum, line)
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
		return nil, RestoreOptions{}, fmt.Errorf("failed to parse restore plan: %w", err)
	}

	return selected, options, nil
}

func interactivePickFiles(files []metadata.FileInfo, tempDir string) ([]metadata.FileInfo, error) {
	// Kept for backwards compatibility - just calls the new function
	selected, _, err := interactivePickAll(files, tempDir, false, false, false, false, false, false)
	return selected, err
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

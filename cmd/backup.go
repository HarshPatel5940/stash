package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/harshpatel5940/stash/internal/archiver"
	"github.com/harshpatel5940/stash/internal/browser"
	"github.com/harshpatel5940/stash/internal/cleanup"
	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/harshpatel5940/stash/internal/defaults"
	"github.com/harshpatel5940/stash/internal/docker"
	stasherrors "github.com/harshpatel5940/stash/internal/errors"
	"github.com/harshpatel5940/stash/internal/finder"
	"github.com/harshpatel5940/stash/internal/fonts"
	"github.com/harshpatel5940/stash/internal/gittracker"
	"github.com/harshpatel5940/stash/internal/incremental"
	"github.com/harshpatel5940/stash/internal/kubernetes"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/packager"
	"github.com/harshpatel5940/stash/internal/recovery"
	"github.com/harshpatel5940/stash/internal/stats"
	"github.com/harshpatel5940/stash/internal/ui"
	"github.com/spf13/cobra"
)

var (
	backupOutput       string
	backupEncryptKey   string
	backupNoEncrypt    bool
	backupDryRun       bool
	backupVerbose      bool
	backupKeepCount    int
	backupSkipBrowsers bool
	backupIncremental  bool
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a new backup",
	Long: `Creates a timestamped backup of your dotfiles, secrets, configs, and package lists.

The backup will include:
  - Dotfiles from home directory
  - Secret directories (SSH, GPG, AWS)
  - .env and .pem files from your projects
  - Application configs (~/.config)
  - Package manager lists (Brewfile, MAS, VS Code, npm)
  - Non-Homebrew apps detection (warns about manually installed apps)
  - macOS system defaults/preferences (Dock, Finder, trackpad, etc.)
  - Shell history (.zsh_history, .bash_history)
  - Browser data (Chrome, Firefox, Safari bookmarks & settings)
  - Git repositories tracking (list of all repos with clone scripts)
  - Custom fonts from ~/Library/Fonts

The backup is compressed as tar.gz and encrypted with age.
Perfect for quickly restoring your Mac anywhere.`,
	RunE: runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Output directory for backups (default: ~/stash-backups)")
	backupCmd.Flags().StringVarP(&backupEncryptKey, "encrypt-key", "k", "", "Path to encryption key (default: ~/.stash.key)")
	backupCmd.Flags().BoolVar(&backupNoEncrypt, "no-encrypt", false, "Skip encryption (not recommended)")
	backupCmd.Flags().BoolVar(&backupDryRun, "dry-run", false, "Preview what would be backed up without creating backup")
	backupCmd.Flags().BoolVarP(&backupVerbose, "verbose", "v", false, "Show detailed output for debugging")
	backupCmd.Flags().IntVar(&backupKeepCount, "keep", 5, "Number of backups to keep (older ones auto-deleted)")
	backupCmd.Flags().BoolVar(&backupSkipBrowsers, "skip-browsers", false, "Skip browser data backup")
	backupCmd.Flags().BoolVarP(&backupIncremental, "incremental", "i", false, "Perform incremental backup (only changed files)")
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Set verbose mode for ui package
	ui.Verbose = backupVerbose

	// Initialize statistics tracking
	backupStats := stats.New()

	// Show spinner for non-dry-run, non-verbose mode
	var spinner *ui.Spinner
	if !backupDryRun && !backupVerbose {
		spinner = ui.NewSpinner("Backing up")
		spinner.Start()
	} else if backupDryRun {
		ui.PrintInfo("DRY RUN - No files will be backed up")
	}

	cfg, err := config.Load()
	if err != nil {
		if spinner != nil {
			spinner.Fail()
		}
		return stasherrors.WrapWithDetection(err, "Failed to load configuration")
	}

	ui.PrintVerbose("Starting backup...")
	cfg.ExpandPaths()

	if backupOutput != "" {
		cfg.BackupDir = backupOutput
	}
	if backupEncryptKey != "" {
		cfg.EncryptionKey = backupEncryptKey
	}

	// Initialize incremental backup manager
	var incrMgr *incremental.Manager
	var doIncrementalBackup bool
	if cfg.Incremental != nil && (cfg.Incremental.Enabled || backupIncremental) {
		mgr, err := incremental.NewManager(cfg)
		if err != nil {
			ui.PrintWarning("Failed to initialize incremental backup: %v", err)
		} else {
			incrMgr = mgr

			// Determine backup type
			if backupIncremental && !incrMgr.ShouldDoFullBackup() {
				doIncrementalBackup = true
			} else if cfg.Incremental.Enabled && !incrMgr.ShouldDoFullBackup() {
				doIncrementalBackup = true
			}

			// Show recommendation (verbose only)
			recommendation := incrMgr.GetRecommendation()
			ui.PrintVerbose("%s", recommendation)
		}
	}

	// Initialize recovery manager
	recoveryMgr := recovery.NewManager(cfg.BackupDir)

	if !backupNoEncrypt {
		encryptor := crypto.NewEncryptor(cfg.EncryptionKey)
		if !encryptor.KeyExists() {
			return stasherrors.NewEncryptionError(cfg.EncryptionKey, nil)
		}
	}

	timestamp := time.Now().Format("2006-01-02-150405")
	backupName := fmt.Sprintf("backup-%s", timestamp)
	tempDir := filepath.Join(os.TempDir(), backupName)

	if !backupDryRun {
		os.RemoveAll(tempDir)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tempDir)
	} else if backupVerbose {
		fmt.Printf("📁 Would create temp directory: %s\n", tempDir)
	}

	meta := metadata.New()

	// Set backup type in metadata
	if doIncrementalBackup {
		meta.SetBackupType("incremental")
		meta.SetChangedFilesOnly(true)
		if incrMgr != nil {
			baseBackup := incrMgr.GetBaseBackup()
			if baseBackup != "" {
				meta.SetBaseBackup(baseBackup)
			}
		}
	} else {
		meta.SetBackupType("full")
	}

	dirs := []string{
		"dotfiles",
		"ssh",
		"gpg",
		"aws",
		"config",
		"env-files",
		"pem-files",
		"packages",
		"macos-defaults",
		"shell-history",
		"browser-data",
		"git-repos",
		"fonts",
		"docker",
		"kubernetes",
	}

	if !backupDryRun {
		for _, dir := range dirs {
			if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
				return fmt.Errorf("failed to create subdirectory %s: %w", dir, err)
			}
		}
	}

	arch := archiver.NewArchiver()

	type backupTask struct {
		Name string
		Func func() error
	}

	tasks := []backupTask{
		{"Dotfiles", func() error { return backupDotfiles(tempDir, meta, arch, cfg, incrMgr, doIncrementalBackup) }},
		{"Secrets", func() error { return backupSecrets(tempDir, meta, arch, incrMgr, doIncrementalBackup, cfg) }},
		{"EnvFiles", func() error { return backupEnvFiles(tempDir, meta, arch, cfg, incrMgr, doIncrementalBackup) }},
		{"PemFiles", func() error { return backupPemFiles(tempDir, meta, arch, cfg, incrMgr, doIncrementalBackup) }},
		{"Packages", func() error { return backupPackages(tempDir, meta) }},
		{"MacOSDefaults", func() error { return backupMacOSDefaults(tempDir, meta, cfg) }},
		{"ShellHistory", func() error { return backupShellHistory(tempDir, meta, arch, incrMgr, doIncrementalBackup, cfg) }},
		{"GitRepos", func() error { return backupGitRepos(tempDir, meta, cfg) }},
		{"Fonts", func() error { return backupFonts(tempDir, meta) }},
		{"Docker", func() error { return backupDocker(tempDir, meta, cfg) }},
		{"Kubernetes", func() error { return backupKubernetes(tempDir, meta) }},
	}

	if !backupSkipBrowsers {
		tasks = append(tasks, backupTask{"BrowserData", func() error { return backupBrowserData(tempDir, meta, incrMgr, doIncrementalBackup) }})
	} else {
		ui.PrintVerbose("Skipping browser data")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks))
	statusChan := make(chan string, len(tasks))
	var errors []error
	var errorsMu sync.Mutex

	doneChan := make(chan bool)
	// Progress tracking only in verbose mode (spinner handles non-verbose)
	go func() {
		for taskName := range statusChan {
			ui.PrintVerbose("Completed: %s", taskName)
		}
		doneChan <- true
	}()

	for _, task := range tasks {
		wg.Add(1)
		go func(t backupTask) {
			defer wg.Done()

			taskStart := time.Now()
			ui.PrintVerbose("Started: %s", t.Name)

			if err := t.Func(); err != nil {
				// Convert to structured error if needed
				var stashErr *stasherrors.StashError
				if se, ok := err.(*stasherrors.StashError); ok {
					stashErr = se
				} else {
					stashErr = stasherrors.WrapWithDetection(err, fmt.Sprintf("Failed during %s", t.Name))
				}

				ui.PrintVerbose("Failed: %s - %v", t.Name, err)

				errorsMu.Lock()
				errors = append(errors, stashErr)
				errorsMu.Unlock()

				errChan <- fmt.Errorf("%s: %w", t.Name, err)

				// Mark task as failed in recovery system
				if !backupDryRun {
					recoveryMgr.MarkTaskFailed(filepath.Join(cfg.BackupDir, backupName), t.Name, err.Error())
				}
			} else {
				// Mark task as complete
				if !backupDryRun {
					recoveryMgr.MarkTaskComplete(filepath.Join(cfg.BackupDir, backupName), t.Name)
				}
			}

			taskDuration := time.Since(taskStart)
			ui.PrintVerbose("Done: %s (%.1fs)", t.Name, taskDuration.Seconds())
			statusChan <- t.Name
		}(task)
	}

	wg.Wait()
	close(errChan)
	close(statusChan)
	<-doneChan

	// Handle errors with better messages
	for err := range errChan {
		if stashErr, ok := err.(*stasherrors.StashError); ok {
			ui.PrintErrorWithSolution(stashErr.Message, stashErr.Suggestion, stashErr.Alternative)
		} else {
			ui.PrintWarning("%v", err)
		}
	}

	readmePath := filepath.Join(tempDir, "README.txt")
	if err := createReadme(readmePath, meta); err != nil {
		ui.PrintVerbose("Warning: failed to create README: %v", err)
	}

	if backupDryRun {
		ui.PrintInfo("DRY RUN - Would create: %s/%s.tar.gz.age (%d files)",
			cfg.BackupDir, backupName, meta.GetFileCount())
		if backupVerbose {
			fmt.Println(meta.Summary())
		}
		return nil
	}

	metadataPath := filepath.Join(tempDir, "metadata.json")
	if err := meta.Save(metadataPath); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	if err := os.MkdirAll(cfg.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	ui.PrintVerbose("Creating archive...")
	archivePath := filepath.Join(cfg.BackupDir, backupName+".tar.gz")
	ui.PrintVerbose("Archive path: %s", archivePath)
	if err := arch.Create(tempDir, archivePath); err != nil {
		if spinner != nil {
			spinner.Fail()
		}
		return fmt.Errorf("failed to create archive: %w", err)
	}

	var finalPath string
	if backupNoEncrypt {
		finalPath = archivePath
		ui.PrintWarning("Backup is NOT encrypted (--no-encrypt)")
	} else {
		ui.PrintVerbose("Encrypting backup...")
		encryptor := crypto.NewEncryptor(cfg.EncryptionKey)
		encryptedPath := archivePath + ".age"

		ui.PrintVerbose("Using key: %s", cfg.EncryptionKey)

		if err := encryptor.Encrypt(archivePath, encryptedPath); err != nil {
			if spinner != nil {
				spinner.Fail()
			}
			return fmt.Errorf("failed to encrypt backup: %w", err)
		}

		os.Remove(archivePath)
		finalPath = encryptedPath
	}

	// Finalize statistics
	fileInfo, _ := os.Stat(finalPath)
	var finalSize int64
	if fileInfo != nil {
		finalSize = fileInfo.Size()
	}

	archiveInfo, _ := os.Stat(archivePath)
	var compressedSize int64
	if archiveInfo != nil {
		compressedSize = archiveInfo.Size()
	}

	backupStats.Finalize(compressedSize, finalSize)

	// Add metadata statistics
	meta.SetCompressedSize(compressedSize)
	meta.SetEncryptedSize(finalSize)
	meta.SetTotalDuration(backupStats.TotalTime)

	// Stop spinner before final output
	if spinner != nil {
		spinner.Stop()
	}

	// Minimal output: single line result
	backupType := "full"
	if doIncrementalBackup {
		backupType = "incremental"
	}
	ui.PrintSuccess("Backup created: %s (%s, %d files, %s)",
		filepath.Base(finalPath),
		ui.FormatBytes(finalSize),
		meta.GetFileCount(),
		backupType,
	)

	// Verbose: detailed statistics
	if backupVerbose {
		ui.PrintDivider()
		fmt.Printf("  Path: %s\n", finalPath)
		fmt.Printf("  Original: %s\n", ui.FormatBytes(meta.BackupSize))
		fmt.Printf("  Compressed: %s (%.0f%% reduction)\n", ui.FormatBytes(compressedSize), meta.GetCompressionRatio())
		fmt.Printf("  Time: %s\n", backupStats.TotalTime.Round(time.Second))
		ui.PrintStatistics(backupStats.ToMap())
	}

	ui.PrintDim("  Restore: stash restore %s", filepath.Base(finalPath))

	// Update incremental index after successful backup
	if incrMgr != nil && !backupDryRun {
		var backedUpFiles []string
		for _, fileInfo := range meta.Files {
			backedUpFiles = append(backedUpFiles, fileInfo.OriginalPath)
		}

		isFull := !doIncrementalBackup
		if err := incrMgr.UpdateIndex(backupName, backedUpFiles, isFull); err != nil {
			ui.PrintVerbose("Warning: failed to update incremental index: %v", err)
		}
	}

	// Register backup in the registry
	if !backupDryRun {
		registry, err := incremental.LoadRegistry()
		if err == nil {
			regType := "full"
			baseBackup := ""
			if doIncrementalBackup {
				regType = "incremental"
				if incrMgr != nil {
					baseBackup = incrMgr.GetBaseBackup()
				}
			}
			registry.RegisterBackup(backupName, finalPath, regType, baseBackup)
			_ = registry.Save()
		}
	}

	// Clean up recovery state on successful completion
	if !backupDryRun {
		recoveryMgr.DeleteState(filepath.Join(cfg.BackupDir, backupName))
	}

	if backupKeepCount > 0 {
		ui.PrintVerbose("Cleaning up old backups...")
		cm := cleanup.NewCleanupManager(cfg.BackupDir)
		deleted, err := cm.RotateByCount(backupKeepCount)
		if err != nil {
			ui.PrintWarning("Cleanup failed: %v", err)
		} else if deleted > 0 {
			ui.PrintVerboseSuccess("Deleted %d old backup(s)", deleted)
		}
	}

	return nil
}

// shouldBackupFile checks if a file should be backed up in incremental mode
func shouldBackupFile(incrMgr *incremental.Manager, doIncremental bool, filePath string) bool {
	// Always backup in full mode
	if !doIncremental || incrMgr == nil {
		return true
	}

	// Check if file has changed
	changed, err := incrMgr.FindChangedFiles([]string{filePath})
	if err != nil {
		// If we can't determine, backup to be safe
		return true
	}

	// Backup if file is in the changed list
	return len(changed) > 0
}

func backupDotfiles(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver, cfg *config.Config, incrMgr *incremental.Manager, doIncremental bool) error {
	dotfilesFinder, err := finder.NewDotfilesFinderWithConfig(cfg)
	if err != nil {
		return err
	}

	dotfiles, err := dotfilesFinder.Find(cfg.AdditionalDotfiles)
	if err != nil {
		return err
	}

	count := 0
	skipped := 0
	for _, file := range dotfiles {
		// Skip unchanged files in incremental mode
		if !shouldBackupFile(incrMgr, doIncremental, file) {
			if backupVerbose {
				fmt.Printf("  ⏭  Skipping unchanged: %s\n", file)
			}
			skipped++
			continue
		}

		fileName := filepath.Base(file)
		destPath := filepath.Join(tempDir, "dotfiles", fileName)

		if backupVerbose {
			fmt.Printf("  📄 %s\n", file)
		}

		if !backupDryRun {
			if err := arch.CopyFile(file, destPath); err != nil {
				if backupVerbose {
					fmt.Printf("  ⚠️  Failed to copy %s: %v\n", file, err)
				}
				continue
			}
		}

		if err := meta.AddFile(file, filepath.Join("dotfiles", fileName)); err != nil {
			if backupVerbose {
				fmt.Printf("  ⚠️  Failed to add metadata for %s: %v\n", file, err)
			}
		}
		count++
	}

	if doIncremental && skipped > 0 && backupVerbose {
		fmt.Printf("  ⏭  Skipped %d unchanged dotfiles\n", skipped)
	}

	if configDir, found := dotfilesFinder.FindConfigDir(); found {
		destPath := filepath.Join(tempDir, "config")
		if backupVerbose {
			fmt.Printf("  📂 %s (excluding node_modules, cache, etc.)\n", configDir)
		}

		if !backupDryRun {
			if err := arch.CopyDir(configDir, destPath); err != nil {
				if backupVerbose {
					fmt.Printf("  ⚠️  Warning: Some .config files skipped: %v\n", err)
				}
			}
		}

		if err := meta.AddFile(configDir, "config"); err != nil {
			if backupVerbose {
				fmt.Printf("  ⚠️  Failed to add metadata for .config: %v\n", err)
			}
		}
		count++
	}

	if backupVerbose {
		fmt.Printf("  ✓ Backed up %d dotfiles/config\n", count)
	}
	return nil
}

func backupSecrets(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver, incrMgr *incremental.Manager, doIncremental bool, cfg *config.Config) error {
	dotfilesFinder, err := finder.NewDotfilesFinderWithConfig(cfg)
	if err != nil {
		return err
	}

	secretDirs := dotfilesFinder.FindSecretDirs()
	count := 0

	for name, path := range secretDirs {
		destPath := filepath.Join(tempDir, name)

		if backupVerbose {
			fmt.Printf("  🔐 %s → %s\n", path, name)
		}

		if !backupDryRun {
			if err := arch.CopyDir(path, destPath); err != nil {
				if backupVerbose {
					fmt.Printf("  ⚠️  Failed to copy %s directory: %v\n", name, err)
				}
				continue
			}
		}

		if err := meta.AddFile(path, name); err != nil {
			if backupVerbose {
				fmt.Printf("  ⚠️  Failed to add metadata for %s: %v\n", name, err)
			}
		}
		count++
	}

	if backupVerbose {
		fmt.Printf("  ✓ Backed up %d secret directories\n", count)
	}
	return nil
}

func backupEnvFiles(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver, cfg *config.Config, incrMgr *incremental.Manager, doIncremental bool) error {
	envFinder := finder.NewEnvFilesFinder(cfg.SearchPaths, cfg.Exclude)
	envFiles, err := envFinder.FindEnvFiles()
	if err != nil {
		return err
	}

	count := 0
	skipped := 0
	for _, file := range envFiles {
		// Skip unchanged files in incremental mode
		if !shouldBackupFile(incrMgr, doIncremental, file) {
			if backupVerbose {
				fmt.Printf("  ⏭  Skipping unchanged: %s\n", file)
			}
			skipped++
			continue
		}

		relPath := strings.TrimPrefix(file, filepath.Dir(cfg.SearchPaths[0]))
		relPath = strings.TrimPrefix(relPath, "/")
		safeName := strings.ReplaceAll(relPath, "/", "-")

		destPath := filepath.Join(tempDir, "env-files", safeName)

		if backupVerbose {
			fmt.Printf("  🔑 %s\n", file)
		}

		if !backupDryRun {
			if err := arch.CopyFile(file, destPath); err != nil {
				if backupVerbose {
					fmt.Printf("  ⚠️  Failed to copy %s: %v\n", file, err)
				}
				continue
			}
		}

		if err := meta.AddFile(file, filepath.Join("env-files", safeName)); err != nil {
			if backupVerbose {
				fmt.Printf("  ⚠️  Failed to add metadata for %s: %v\n", file, err)
			}
		}
		count++
	}

	if backupVerbose {
		fmt.Printf("  ✓ Backed up %d .env files", count)
		if doIncremental && skipped > 0 {
			fmt.Printf(" (skipped %d unchanged)", skipped)
		}
		fmt.Println()
	}
	return nil
}

func backupPemFiles(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver, cfg *config.Config, incrMgr *incremental.Manager, doIncremental bool) error {
	envFinder := finder.NewEnvFilesFinder(cfg.SearchPaths, cfg.Exclude)
	pemFiles, err := envFinder.FindPemFiles()
	if err != nil {
		return err
	}

	count := 0
	skipped := 0
	for _, file := range pemFiles {
		// Skip unchanged files in incremental mode
		if !shouldBackupFile(incrMgr, doIncremental, file) {
			if backupVerbose {
				fmt.Printf("  ⏭  Skipping unchanged: %s\n", file)
			}
			skipped++
			continue
		}

		relPath := strings.TrimPrefix(file, filepath.Dir(cfg.SearchPaths[0]))
		relPath = strings.TrimPrefix(relPath, "/")
		safeName := strings.ReplaceAll(relPath, "/", "-")

		destPath := filepath.Join(tempDir, "pem-files", safeName)

		if backupVerbose {
			fmt.Printf("  🔒 %s\n", file)
		}

		if !backupDryRun {
			if err := arch.CopyFile(file, destPath); err != nil {
				if backupVerbose {
					fmt.Printf("  ⚠️  Failed to copy %s: %v\n", file, err)
				}
				continue
			}
		}

		if err := meta.AddFile(file, filepath.Join("pem-files", safeName)); err != nil {
			if backupVerbose {
				fmt.Printf("  ⚠️  Failed to add metadata for %s: %v\n", file, err)
			}
		}
		count++
	}

	if backupVerbose {
		fmt.Printf("  ✓ Backed up %d .pem files", count)
		if doIncremental && skipped > 0 {
			fmt.Printf(" (skipped %d unchanged)", skipped)
		}
		fmt.Println()
	}
	return nil
}

func backupPackages(tempDir string, meta *metadata.Metadata) error {
	packagesDir := filepath.Join(tempDir, "packages")
	pkg := packager.NewPackager(packagesDir)

	var counts map[string]int
	var err error

	if backupDryRun {

		counts = make(map[string]int)
		counts["Homebrew"] = 0
		counts["MAS"] = 0
		counts["VSCode"] = 0
		counts["NPM"] = 0
		if backupVerbose {
			fmt.Println("  ℹ️  Would collect package lists (skipped in dry-run)")
		}
	} else {
		counts, err = pkg.CollectAll()
		if err != nil {
			return err
		}
	}

	total := 0
	for name, count := range counts {
		meta.SetPackageCount(name, count)
		total += count
		if backupVerbose {
			fmt.Printf("  📦 %s: %d packages\n", name, count)
		}
	}

	if total == 0 {
		if backupVerbose {
			fmt.Println("  ⚠️  No package managers found")
		}
	}

	return nil
}

func backupMacOSDefaults(tempDir string, meta *metadata.Metadata, cfg *config.Config) error {
	defaultsDir := filepath.Join(tempDir, "macos-defaults")
	dm := defaults.NewDefaultsManagerWithConfig(defaultsDir, cfg)

	if backupDryRun {
		domains := dm.ImportantDomains()
		if backupVerbose {
			fmt.Printf("  Would backup %d macOS preference domains\n", len(domains))
		}
		return nil
	}

	if err := dm.BackupAll(); err != nil {
		return fmt.Errorf("failed to backup macOS defaults: %w", err)
	}

	meta.AddFileInfo(metadata.FileInfo{
		OriginalPath: "~/Library/Preferences",
		BackupPath:   "macos-defaults/macos-defaults.json",
		Size:         0,
		Mode:         0644,
		IsDir:        false,
	})

	count, _ := dm.GetStats(filepath.Join(defaultsDir, "macos-defaults.json"))
	if backupVerbose {
		fmt.Printf("  ✓ Backed up %d preference domains\n", count)
	}

	return nil
}

func backupShellHistory(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver, incrMgr *incremental.Manager, doIncremental bool, cfg *config.Config) error {
	homeDir, _ := os.UserHomeDir()
	historyDir := filepath.Join(tempDir, "shell-history")

	historyFiles := cfg.GetShellHistoryFiles()

	count := 0
	skipped := 0
	for _, histFile := range historyFiles {
		srcPath := filepath.Join(homeDir, histFile)
		if _, err := os.Stat(srcPath); err != nil {
			continue
		}

		// Skip unchanged files in incremental mode
		if !shouldBackupFile(incrMgr, doIncremental, srcPath) {
			if backupVerbose {
				fmt.Printf("  ⏭  Skipping unchanged: %s\n", histFile)
			}
			skipped++
			continue
		}

		if backupDryRun {
			count++
			continue
		}

		destPath := filepath.Join(historyDir, histFile)
		if err := arch.CopyFile(srcPath, destPath); err != nil {
			if backupVerbose {
				fmt.Printf("  ⚠️  Failed to backup %s: %v\n", histFile, err)
			}
			continue
		}

		info, _ := os.Stat(srcPath)
		meta.AddFileInfo(metadata.FileInfo{
			OriginalPath: "~/" + histFile,
			BackupPath:   "shell-history/" + histFile,
			Size:         info.Size(),
			Mode:         info.Mode(),
			IsDir:        false,
		})

		count++
	}

	if count > 0 {
		if backupVerbose {
			fmt.Printf("  ✓ Backed up %d history file(s)\n", count)
		}
	} else if backupVerbose {
		fmt.Println("  ℹ️  No shell history files found")
	}

	return nil
}

func backupBrowserData(tempDir string, meta *metadata.Metadata, incrMgr *incremental.Manager, doIncremental bool) error {
	browserDir := filepath.Join(tempDir, "browser-data")
	bm := browser.NewBrowserManager(browserDir)

	if backupDryRun {
		if backupVerbose {
			fmt.Println("  Would backup browser data (Chrome, Firefox, Safari)")
		}
		return nil
	}

	counts, err := bm.BackupAll()
	if err != nil {
		return err
	}

	for browserName, count := range counts {
		relPath := "browser-data/" + strings.ToLower(browserName)
		fullPath := filepath.Join(tempDir, relPath)

		size, _ := getDirSize(fullPath)

		meta.AddFileInfo(metadata.FileInfo{
			OriginalPath: "~/Library/Application Support/" + browserName,
			BackupPath:   relPath,
			Size:         size,
			Mode:         0755,
			IsDir:        true,
		})
		if backupVerbose {
			fmt.Printf("  ✓ Backed up %s (%d items)\n", browserName, count)
		}
	}

	return nil
}

func backupGitRepos(tempDir string, meta *metadata.Metadata, cfg *config.Config) error {
	gitDir := filepath.Join(tempDir, "git-repos")
	gt := gittracker.NewGitTrackerWithConfig(
		gitDir,
		cfg.GetGitMaxDepth(),
		cfg.GetGitSkipDirs(),
	)

	searchDirs := cfg.GetGitSearchDirs()

	if backupDryRun {
		if backupVerbose {
			fmt.Println("  Would scan for git repositories in common directories")
		}
		return nil
	}

	if err := gt.ScanDirectories(searchDirs); err != nil {
		return err
	}

	// Warn about repos needing attention
	reposNeedingAttention := gt.GetReposNeedingAttention()
	for _, repo := range reposNeedingAttention {
		if repo.Dirty && repo.UnpushedCount > 0 {
			fmt.Printf("  ⚠️  %s: uncommitted changes + %d unpushed commit(s)\n", repo.Path, repo.UnpushedCount)
		} else if repo.Dirty {
			fmt.Printf("  ⚠️  %s: uncommitted changes\n", repo.Path)
		} else if repo.UnpushedCount > 0 {
			fmt.Printf("  ⚠️  %s: %d unpushed commit(s)\n", repo.Path, repo.UnpushedCount)
		}
	}

	if err := gt.Save(); err != nil {
		return err
	}

	count := gt.GetCount()
	if count > 0 {
		meta.AddFileInfo(metadata.FileInfo{
			OriginalPath: "~/Projects (git repos)",
			BackupPath:   "git-repos/git-repos.json",
			Size:         0,
			Mode:         0644,
			IsDir:        false,
		})
		if backupVerbose {
			fmt.Printf("  ✓ Tracked %d git repositories\n", count)
		}
	} else {
		if backupVerbose {
			fmt.Println("  ℹ️  No git repositories found")
		}
	}

	return nil
}

func backupFonts(tempDir string, meta *metadata.Metadata) error {
	fontsDir := filepath.Join(tempDir, "fonts")
	fm := fonts.NewFontsManager(fontsDir)

	if backupDryRun {
		if backupVerbose {
			fmt.Println("  Would backup custom fonts from ~/Library/Fonts")
		}
		return nil
	}

	count, err := fm.BackupAll()
	if err != nil {
		return err
	}

	size, _ := getDirSize(fontsDir)

	meta.AddFileInfo(metadata.FileInfo{
		OriginalPath: "~/Library/Fonts",
		BackupPath:   "fonts/",
		Size:         size,
		Mode:         0755,
		IsDir:        true,
	})
	if backupVerbose {
		fmt.Printf("  ✓ Backed up %d custom fonts\n", count)
	}

	return nil
}

func backupDocker(tempDir string, meta *metadata.Metadata, cfg *config.Config) error {
	dockerDir := filepath.Join(tempDir, "docker")

	dockerMgr := docker.NewDockerManager(dockerDir, cfg.SearchPaths)

	if backupDryRun {
		if backupVerbose {
			fmt.Println("  [DRY RUN] Would backup Docker configuration")
		}
		return nil
	}

	count, err := dockerMgr.BackupAll()
	if err != nil {
		if backupVerbose {
			fmt.Printf("  ⚠️  Docker backup: %v\n", err)
		}
		return nil // Don't fail backup if Docker isn't configured
	}

	if count > 0 {
		meta.SetPackageCount("docker-config", count)
	}

	if backupVerbose {
		fmt.Printf("  ✓ Backed up Docker configuration (%d files)\n", count)
	}

	return nil
}

func backupKubernetes(tempDir string, meta *metadata.Metadata) error {
	k8sDir := filepath.Join(tempDir, "kubernetes")

	k8sMgr := kubernetes.NewKubernetesManager(k8sDir)

	if backupDryRun {
		if backupVerbose {
			fmt.Println("  [DRY RUN] Would backup Kubernetes configuration")
		}
		return nil
	}

	count, err := k8sMgr.BackupAll()
	if err != nil {
		if backupVerbose {
			fmt.Printf("  ⚠️  Kubernetes backup: %v\n", err)
		}
		return nil // Don't fail backup if Kubernetes isn't configured
	}

	if count > 0 {
		meta.SetPackageCount("kubernetes-config", count)
	}

	if backupVerbose {
		fmt.Printf("  ✓ Backed up Kubernetes configuration (%d files)\n", count)
	}

	return nil
}

func createReadme(path string, meta *metadata.Metadata) error {
	content := fmt.Sprintf(`Stash Backup - %s
========================================

This backup was created by Stash on %s

Backup Contents:
- Dotfiles: Shell configs, git configs, etc.
- Secrets: SSH keys, GPG keys, AWS credentials
- Environment Files: .env files from your projects
- PEM Files: Certificate and key files
- Package Lists: Homebrew, MAS, VS Code, NPM

Metadata:
- Hostname: %s
- Username: %s
- Timestamp: %s

To restore this backup:
1. Install Stash on your new Mac
2. Copy your encryption key (~/.stash.key) to the new Mac
3. Run: stash restore <backup-file>

For more information, visit: https://github.com/harshpatel5940/stash
`, meta.Version, meta.Timestamp.Format("2006-01-02 15:04:05"),
		meta.Hostname, meta.Username, meta.Timestamp.Format(time.RFC3339))

	return os.WriteFile(path, []byte(content), 0644)
}

func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

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
	"github.com/harshpatel5940/stash/internal/finder"
	"github.com/harshpatel5940/stash/internal/fonts"
	"github.com/harshpatel5940/stash/internal/gittracker"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/packager"
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
}

func runBackup(cmd *cobra.Command, args []string) error {
	if backupDryRun {
		fmt.Println("🔍 DRY RUN MODE - No files will be backed up")
	} else {
		fmt.Println("🚀 Starting backup process...")
	}
	fmt.Println()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ui.PrintSectionHeader("📦", "Starting Backup")
	cfg.ExpandPaths()

	if backupOutput != "" {
		cfg.BackupDir = backupOutput
	}
	if backupEncryptKey != "" {
		cfg.EncryptionKey = backupEncryptKey
	}

	if !backupNoEncrypt {
		encryptor := crypto.NewEncryptor(cfg.EncryptionKey)
		if !encryptor.KeyExists() {
			fmt.Printf("❌ Encryption key not found: %s\n", cfg.EncryptionKey)
			fmt.Println("\n💡 Run 'stash init' to generate an encryption key")
			return fmt.Errorf("encryption key not found")
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
		{"Dotfiles", func() error { return backupDotfiles(tempDir, meta, arch, cfg) }},
		{"Secrets", func() error { return backupSecrets(tempDir, meta, arch) }},
		{"EnvFiles", func() error { return backupEnvFiles(tempDir, meta, arch, cfg) }},
		{"PemFiles", func() error { return backupPemFiles(tempDir, meta, arch, cfg) }},
		{"Packages", func() error { return backupPackages(tempDir, meta) }},
		{"MacOSDefaults", func() error { return backupMacOSDefaults(tempDir, meta) }},
		{"ShellHistory", func() error { return backupShellHistory(tempDir, meta, arch) }},
		{"GitRepos", func() error { return backupGitRepos(tempDir, meta) }},
		{"Fonts", func() error { return backupFonts(tempDir, meta) }},
	}

	if !backupSkipBrowsers {
		tasks = append(tasks, backupTask{"BrowserData", func() error { return backupBrowserData(tempDir, meta) }})
	} else if backupVerbose {
		fmt.Println("🚫 Skipping browser data backup")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks))
	statusChan := make(chan string, len(tasks))

	doneChan := make(chan bool)
	if !backupVerbose {
		go func() {
			completed := 0
			total := len(tasks)
			var lastTask string

			fmt.Printf("\r⏳ Backing up... (0/%d)", total)

			for taskName := range statusChan {
				completed++
				lastTask = taskName
				fmt.Printf("\r⏳ Backing up... (%d/%d) - Finished: %s     ", completed, total, lastTask)
			}
			fmt.Println()
			doneChan <- true
		}()
	}

	for _, task := range tasks {
		wg.Add(1)
		go func(t backupTask) {
			defer wg.Done()
			if backupVerbose {
				fmt.Printf("Started: %s\n", t.Name)
			}
			if err := t.Func(); err != nil {

				if backupVerbose {
					fmt.Printf("⚠️  %s: %v\n", t.Name, err)
				}
				errChan <- fmt.Errorf("%s: %w", t.Name, err)
			}
			if !backupVerbose {
				statusChan <- t.Name
			}
		}(task)
	}

	wg.Wait()
	close(errChan)
	close(statusChan)

	if !backupVerbose {
		<-doneChan
	}

	for err := range errChan {
		ui.PrintWarning("%v", err)
	}

	readmePath := filepath.Join(tempDir, "README.txt")
	if err := createReadme(readmePath, meta); err != nil {
		fmt.Printf("⚠️  Warning: failed to create README: %v\n", err)
	}

	if backupDryRun {
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("🔍 DRY RUN SUMMARY - No backup created")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("\n" + meta.Summary())
		fmt.Printf("\n📁 Would create backup at: %s/%s.tar.gz.age\n", cfg.BackupDir, backupName)
		fmt.Println("\n💡 Run without --dry-run to create actual backup")
		return nil
	}

	metadataPath := filepath.Join(tempDir, "metadata.json")
	if err := meta.Save(metadataPath); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	if err := os.MkdirAll(cfg.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	fmt.Println("\n📦 Creating archive...")
	archivePath := filepath.Join(cfg.BackupDir, backupName+".tar.gz")
	if backupVerbose {
		fmt.Printf("  📝 Archive path: %s\n", archivePath)
	}
	if err := arch.Create(tempDir, archivePath); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	var finalPath string
	if backupNoEncrypt {
		finalPath = archivePath
		fmt.Println("⚠️  Backup is NOT encrypted (--no-encrypt was used)")
	} else {
		fmt.Println("🔐 Encrypting backup...")
		encryptor := crypto.NewEncryptor(cfg.EncryptionKey)
		encryptedPath := archivePath + ".age"

		if backupVerbose {
			fmt.Printf("  🔑 Using key: %s\n", cfg.EncryptionKey)
			fmt.Printf("  📝 Encrypted output: %s\n", encryptedPath)
		}

		if err := encryptor.Encrypt(archivePath, encryptedPath); err != nil {
			return fmt.Errorf("failed to encrypt backup: %w", err)
		}

		os.Remove(archivePath)
		finalPath = encryptedPath
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ Backup completed successfully!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("\n📁 Backup location: %s\n", finalPath)

	fileInfo, _ := os.Stat(finalPath)
	if fileInfo != nil {
		fmt.Printf("💾 Backup size: %s\n", metadata.FormatSize(fileInfo.Size()))
	}

	if backupVerbose {
		fmt.Println("\n" + meta.Summary())
	}

	fmt.Println("\n📖 To restore this backup on a new Mac:")
	fmt.Printf("   stash restore %s\n", filepath.Base(finalPath))

	if backupKeepCount > 0 {
		ui.PrintSectionHeader("🧹", "Cleaning up old backups...")
		cm := cleanup.NewCleanupManager(cfg.BackupDir)
		deleted, err := cm.RotateByCount(backupKeepCount)
		if err != nil {
			ui.PrintWarning("Failed to cleanup old backups: %v", err)
		} else if deleted > 0 {
			ui.PrintSuccess("Deleted %d old backup(s), keeping %d most recent", deleted, backupKeepCount)
		} else {
			ui.PrintInfo("No cleanup needed (keeping %d backups)", backupKeepCount)
		}
	}

	return nil
}

func backupDotfiles(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver, cfg *config.Config) error {
	dotfilesFinder, err := finder.NewDotfilesFinder()
	if err != nil {
		return err
	}

	dotfiles, err := dotfilesFinder.Find(cfg.AdditionalDotfiles)
	if err != nil {
		return err
	}

	count := 0
	for _, file := range dotfiles {
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

func backupSecrets(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver) error {
	dotfilesFinder, err := finder.NewDotfilesFinder()
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

func backupEnvFiles(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver, cfg *config.Config) error {
	envFinder := finder.NewEnvFilesFinder(cfg.SearchPaths, cfg.Exclude)
	envFiles, err := envFinder.FindEnvFiles()
	if err != nil {
		return err
	}

	count := 0
	for _, file := range envFiles {

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
		fmt.Printf("  ✓ Backed up %d .env files\n", count)
	}
	return nil
}

func backupPemFiles(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver, cfg *config.Config) error {
	envFinder := finder.NewEnvFilesFinder(cfg.SearchPaths, cfg.Exclude)
	pemFiles, err := envFinder.FindPemFiles()
	if err != nil {
		return err
	}

	count := 0
	for _, file := range pemFiles {

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
		fmt.Printf("  ✓ Backed up %d .pem files\n", count)
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

func backupMacOSDefaults(tempDir string, meta *metadata.Metadata) error {
	defaultsDir := filepath.Join(tempDir, "macos-defaults")
	dm := defaults.NewDefaultsManager(defaultsDir)

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

func backupShellHistory(tempDir string, meta *metadata.Metadata, arch *archiver.Archiver) error {
	homeDir, _ := os.UserHomeDir()
	historyDir := filepath.Join(tempDir, "shell-history")

	historyFiles := []string{
		".zsh_history",
		".bash_history",
		".zhistory",
	}

	count := 0
	for _, histFile := range historyFiles {
		srcPath := filepath.Join(homeDir, histFile)
		if _, err := os.Stat(srcPath); err != nil {
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

func backupBrowserData(tempDir string, meta *metadata.Metadata) error {
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

func backupGitRepos(tempDir string, meta *metadata.Metadata) error {
	gitDir := filepath.Join(tempDir, "git-repos")
	gt := gittracker.NewGitTracker(gitDir)

	homeDir, _ := os.UserHomeDir()
	searchDirs := []string{
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Projects"),
		filepath.Join(homeDir, "Code"),
		filepath.Join(homeDir, "Dev"),
		filepath.Join(homeDir, "workspace"),
		filepath.Join(homeDir, "github"),
	}

	if backupDryRun {
		if backupVerbose {
			fmt.Println("  Would scan for git repositories in common directories")
		}
		return nil
	}

	if err := gt.ScanDirectories(searchDirs); err != nil {
		return err
	}

	for _, repo := range gt.GetRepos() {
		if repo.Dirty {
			fmt.Printf("  ⚠️  Uncommitted changes in %s\n", repo.Path)
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

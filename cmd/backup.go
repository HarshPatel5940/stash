package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harshpatel5940/stash/internal/archiver"
	"github.com/harshpatel5940/stash/internal/config"
	"github.com/harshpatel5940/stash/internal/crypto"
	"github.com/harshpatel5940/stash/internal/finder"
	"github.com/harshpatel5940/stash/internal/metadata"
	"github.com/harshpatel5940/stash/internal/packager"
	"github.com/spf13/cobra"
)

var (
	backupOutput     string
	backupEncryptKey string
	backupNoEncrypt  bool
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

The backup is compressed as tar.gz and encrypted with age.`,
	RunE: runBackup,
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Output directory for backups (default: ~/stash-backups)")
	backupCmd.Flags().StringVarP(&backupEncryptKey, "encrypt-key", "k", "", "Path to encryption key (default: ~/.stash.key)")
	backupCmd.Flags().BoolVar(&backupNoEncrypt, "no-encrypt", false, "Skip encryption (not recommended)")
}

func runBackup(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Starting backup process...")
	fmt.Println()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg.ExpandPaths()

	// Override with flags if provided
	if backupOutput != "" {
		cfg.BackupDir = backupOutput
	}
	if backupEncryptKey != "" {
		cfg.EncryptionKey = backupEncryptKey
	}

	// Check encryption key exists
	if !backupNoEncrypt {
		encryptor := crypto.NewEncryptor(cfg.EncryptionKey)
		if !encryptor.KeyExists() {
			fmt.Printf("❌ Encryption key not found: %s\n", cfg.EncryptionKey)
			fmt.Println("\n💡 Run 'stash init' to generate an encryption key")
			return fmt.Errorf("encryption key not found")
		}
	}

	// Create backup directory structure
	timestamp := time.Now().Format("2006-01-02-150405")
	backupName := fmt.Sprintf("backup-%s", timestamp)
	tempDir := filepath.Join(os.TempDir(), backupName)

	// Ensure temp directory is clean
	os.RemoveAll(tempDir)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup temp dir

	// Initialize metadata
	meta := metadata.New()

	// Create subdirectories
	dirs := []string{
		"dotfiles",
		"ssh",
		"gpg",
		"aws",
		"config",
		"env-files",
		"pem-files",
		"packages",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			return fmt.Errorf("failed to create subdirectory %s: %w", dir, err)
		}
	}

	arch := archiver.NewArchiver()

	// Backup dotfiles
	fmt.Println("📄 Backing up dotfiles...")
	if err := backupDotfiles(tempDir, meta, arch, cfg); err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
	}

	// Backup secret directories
	fmt.Println("🔐 Backing up secret directories...")
	if err := backupSecrets(tempDir, meta, arch); err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
	}

	// Backup .env files
	fmt.Println("🔑 Backing up .env files...")
	if err := backupEnvFiles(tempDir, meta, arch, cfg); err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
	}

	// Backup .pem files
	fmt.Println("🔒 Backing up .pem files...")
	if err := backupPemFiles(tempDir, meta, arch, cfg); err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
	}

	// Backup package lists
	fmt.Println("📦 Backing up package lists...")
	if err := backupPackages(tempDir, meta); err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
	}

	// Create README
	readmePath := filepath.Join(tempDir, "README.txt")
	if err := createReadme(readmePath, meta); err != nil {
		fmt.Printf("⚠️  Warning: failed to create README: %v\n", err)
	}

	// Save metadata
	metadataPath := filepath.Join(tempDir, "metadata.json")
	if err := meta.Save(metadataPath); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(cfg.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Archive the backup
	fmt.Println("\n📦 Creating archive...")
	archivePath := filepath.Join(cfg.BackupDir, backupName+".tar.gz")
	if err := arch.Create(tempDir, archivePath); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	// Encrypt the archive
	var finalPath string
	if backupNoEncrypt {
		finalPath = archivePath
		fmt.Println("⚠️  Backup is NOT encrypted (--no-encrypt was used)")
	} else {
		fmt.Println("🔐 Encrypting backup...")
		encryptor := crypto.NewEncryptor(cfg.EncryptionKey)
		encryptedPath := archivePath + ".age"

		if err := encryptor.Encrypt(archivePath, encryptedPath); err != nil {
			return fmt.Errorf("failed to encrypt backup: %w", err)
		}

		// Remove unencrypted archive
		os.Remove(archivePath)
		finalPath = encryptedPath
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✅ Backup completed successfully!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("\n📁 Backup location: %s\n", finalPath)

	fileInfo, _ := os.Stat(finalPath)
	if fileInfo != nil {
		fmt.Printf("💾 Backup size: %s\n", metadata.FormatSize(fileInfo.Size()))
	}

	fmt.Println("\n" + meta.Summary())
	fmt.Println("\n💡 To restore this backup on a new Mac:")
	fmt.Printf("   stash restore %s\n", filepath.Base(finalPath))

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

		if err := arch.CopyFile(file, destPath); err != nil {
			fmt.Printf("  ⚠️  Failed to copy %s: %v\n", file, err)
			continue
		}

		if err := meta.AddFile(file, filepath.Join("dotfiles", fileName)); err != nil {
			fmt.Printf("  ⚠️  Failed to add metadata for %s: %v\n", file, err)
		}
		count++
	}

	// Backup .config directory (with smart exclusions)
	if configDir, found := dotfilesFinder.FindConfigDir(); found {
		destPath := filepath.Join(tempDir, "config")
		fmt.Printf("  📂 Backing up .config (excluding node_modules, cache, etc.)...\n")
		if err := arch.CopyDir(configDir, destPath); err != nil {
			fmt.Printf("  ⚠️  Warning: Some .config files skipped: %v\n", err)
		}
		// Always add metadata even if some files were skipped
		if err := meta.AddFile(configDir, "config"); err != nil {
			fmt.Printf("  ⚠️  Failed to add metadata for .config: %v\n", err)
		}
		count++
	}

	fmt.Printf("  ✓ Backed up %d dotfiles/config\n", count)
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
		if err := arch.CopyDir(path, destPath); err != nil {
			fmt.Printf("  ⚠️  Failed to copy %s directory: %v\n", name, err)
			continue
		}

		if err := meta.AddFile(path, name); err != nil {
			fmt.Printf("  ⚠️  Failed to add metadata for %s: %v\n", name, err)
		}
		count++
	}

	fmt.Printf("  ✓ Backed up %d secret directories\n", count)
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
		// Create a safe filename that preserves path info
		relPath := strings.TrimPrefix(file, filepath.Dir(cfg.SearchPaths[0]))
		relPath = strings.TrimPrefix(relPath, "/")
		safeName := strings.ReplaceAll(relPath, "/", "-")

		destPath := filepath.Join(tempDir, "env-files", safeName)
		if err := arch.CopyFile(file, destPath); err != nil {
			fmt.Printf("  ⚠️  Failed to copy %s: %v\n", file, err)
			continue
		}

		if err := meta.AddFile(file, filepath.Join("env-files", safeName)); err != nil {
			fmt.Printf("  ⚠️  Failed to add metadata for %s: %v\n", file, err)
		}
		count++
	}

	fmt.Printf("  ✓ Backed up %d .env files\n", count)
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
		// Create a safe filename that preserves path info
		relPath := strings.TrimPrefix(file, filepath.Dir(cfg.SearchPaths[0]))
		relPath = strings.TrimPrefix(relPath, "/")
		safeName := strings.ReplaceAll(relPath, "/", "-")

		destPath := filepath.Join(tempDir, "pem-files", safeName)
		if err := arch.CopyFile(file, destPath); err != nil {
			fmt.Printf("  ⚠️  Failed to copy %s: %v\n", file, err)
			continue
		}

		if err := meta.AddFile(file, filepath.Join("pem-files", safeName)); err != nil {
			fmt.Printf("  ⚠️  Failed to add metadata for %s: %v\n", file, err)
		}
		count++
	}

	fmt.Printf("  ✓ Backed up %d .pem files\n", count)
	return nil
}

func backupPackages(tempDir string, meta *metadata.Metadata) error {
	packagesDir := filepath.Join(tempDir, "packages")
	pkg := packager.NewPackager(packagesDir)

	counts, err := pkg.CollectAll()
	if err != nil {
		return err
	}

	total := 0
	for name, count := range counts {
		meta.SetPackageCount(name, count)
		total += count
		fmt.Printf("  ✓ %s: %d packages\n", name, count)
	}

	if total == 0 {
		fmt.Println("  ⚠️  No package managers found")
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

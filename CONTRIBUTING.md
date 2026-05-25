# Contributing to Stash

Thanks for your interest in contributing! This guide covers setup, architecture, development workflows, and contribution guidelines.

---

## 📥 Installation Methods

### Using Go (Recommended for Development)

```bash
go install github.com/harshpatel5940/stash@latest
```

Ensure `$GOPATH/bin` is in your `PATH`:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Using Homebrew

```bash
brew install harshpatel5940/tap/stash
```

### From Source

```bash
git clone https://github.com/harshpatel5940/stash.git
cd stash
make build
./stash --version
```

---

## 🚀 Development Quick Start

```bash
# Clone
git clone https://github.com/harshpatel5940/stash.git
cd stash

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run locally
./stash --help
./stash init
./stash backup --dry-run
```

---

## 📁 Project Structure

```
stash/
├── cmd/                          # CLI commands (cobra)
│   ├── root.go                   # Root command setup
│   ├── init.go                   # Initialize config + key
│   ├── backup.go                 # Create backups
│   ├── backup_notes.go           # Add/edit notes on backups
│   ├── backup_resolver.go        # Resolve backup sources
│   ├── restore.go                # Restore from backup
│   ├── list.go                   # List available backups
│   ├── info.go                   # Show backup metadata
│   ├── diff.go                   # Compare two backups
│   ├── cleanup.go                # Delete old backups
│   ├── optimize.go               # Merge incremental backups
│   ├── remind.go                 # List repos needing attention
│   ├── config.go                 # Manage configuration
│   ├── sync.go                   # Sync with cloud storage
│   └── cmd_test.go               # Command tests
│
├── internal/
│   ├── archiver/                 # Tar.gz operations
│   │   ├── archiver.go           # Create/extract archives
│   │   └── archiver_test.go
│   │
│   ├── backuputil/               # Backup utilities
│   │   └── *.go                  # Shared backup helpers
│   │
│   ├── browser/                  # Browser data backup
│   │   ├── browser.go            # Chrome/Safari backups
│   │   └── browser_test.go
│   │
│   ├── cleanup/                  # Backup cleanup/retention
│   │   ├── cleanup.go            # Delete old backups
│   │   └── cleanup_test.go
│   │
│   ├── cloud/                    # Cloud storage integration
│   │   ├── cloud.go              # S3, cloud adapters
│   │   └── cloud_test.go
│   │
│   ├── config/                   # Configuration management
│   │   ├── config.go             # Load/save .stash.yaml
│   │   └── config_test.go
│   │
│   ├── crypto/                   # Age encryption
│   │   ├── crypto.go             # Encrypt/decrypt operations
│   │   └── crypto_test.go
│   │
│   ├── defaults/                 # macOS defaults backup
│   │   ├── defaults.go           # System preferences
│   │   └── defaults_test.go
│   │
│   ├── diff/                     # Backup comparison
│   │   ├── diff.go               # Compare backups
│   │   └── diff_test.go
│   │
│   ├── docker/                   # Docker config backup
│   │   ├── docker.go             # Docker configs
│   │   └── docker_test.go
│   │
│   ├── dockprefs/                # Dock preferences backup
│   │   ├── dockprefs.go          # Dock layout, apps
│   │   └── dockprefs_test.go
│   │
│   ├── errors/                   # Error handling
│   │   └── errors.go             # Custom error types
│   │
│   ├── finder/                   # File discovery
│   │   ├── finder.go             # Find dotfiles, secrets
│   │   └── finder_test.go
│   │
│   ├── firewall/                 # Firewall rules backup
│   │   ├── firewall.go           # macOS firewall config
│   │   └── firewall_test.go
│   │
│   ├── fonts/                    # Custom fonts backup
│   │   ├── fonts.go              # ~/Library/Fonts
│   │   └── fonts_test.go
│   │
│   ├── gittracker/               # Git repo tracking
│   │   ├── gittracker.go         # Find/list git repos
│   │   └── gittracker_test.go
│   │
│   ├── incremental/              # Incremental backup support
│   │   ├── incremental.go        # Diff-based backups
│   │   └── incremental_test.go
│   │
│   ├── index/                    # Backup indexing
│   │   ├── index.go              # Index for fast lookup
│   │   └── index_test.go
│   │
│   ├── kubernetes/               # Kubernetes config backup
│   │   ├── kubernetes.go         # ~/.kube configs
│   │   └── kubernetes_test.go
│   │
│   ├── metadata/                 # Backup metadata
│   │   ├── metadata.go           # File info, checksums
│   │   └── metadata_test.go
│   │
│   ├── packager/                 # Package managers
│   │   └── packager.go           # Homebrew, MAS, VSCode, npm
│   │
│   ├── progress/                 # Progress indication
│   │   ├── progress.go           # Progress bars
│   │   └── progress_test.go
│   │
│   ├── recovery/                 # Recovery utilities
│   │   ├── recovery.go           # Restore helpers
│   │   └── recovery_test.go
│   │
│   ├── security/                 # Security utilities
│   │   ├── security.go           # Path validation, etc.
│   │   └── security_test.go
│   │
│   ├── stats/                    # Backup statistics
│   │   ├── stats.go              # Calculate sizes, counts
│   │   └── stats_test.go
│   │
│   ├── tui/                      # Text UI (interactive modes)
│   │   ├── tui.go                # Interactive restore picker
│   │   └── tui_test.go
│   │
│   ├── ui/                       # User interface
│   │   ├── ui.go                 # Output formatting
│   │   └── ui_test.go
│   │
│   └── wallpaper/                # Wallpaper backup
│       ├── wallpaper.go          # Desktop background
│       └── wallpaper_test.go
│
├── .github/workflows/            # CI/CD
│   ├── test.yml                  # Run tests on PR/push
│   └── release.yml               # GoReleaser on tag
│
├── .goreleaser.yml               # Release configuration
├── Makefile                      # Build automation
├── go.mod                        # Dependencies
├── main.go                       # Entry point
└── stash.rb                      # Homebrew formula
```

---

## 🏗️ Architecture

### Backup Flow

```
init → backup → list → diff → sync/cleanup → restore
```

**1. Init** (`cmd/init.go`)
- Generate age key → `~/.stash.key` (0600 perms)
- Create config → `~/.stash.yaml`
- Auto-install dependencies (Homebrew, mas, dockutil)

**2. Backup** (`cmd/backup.go`)
- Load config → `internal/config`
- Find dotfiles/secrets → `internal/finder`
- Backup browser data → `internal/browser` (optional)
- Backup docker/k8s → `internal/docker`, `internal/kubernetes`
- Backup system prefs → `internal/defaults`, `internal/dockprefs`, `internal/firewall`, `internal/wallpaper`, `internal/fonts`
- Collect packages → `internal/packager`
- Track git repos → `internal/gittracker`
- Create metadata → `internal/metadata`
- Archive → tar.gz via `internal/archiver`
- Encrypt → age via `internal/crypto`
- Incremental support → `internal/incremental`
- Show stats → `internal/stats`
- Optional cloud upload → `internal/cloud`
- Cleanup old backups → `internal/cleanup`
- Output: `backup-TIMESTAMP.tar.gz.age`

**3. Restore** (`cmd/restore.go`)
- Decrypt `.age` file → `internal/crypto`
- Extract tar.gz → `internal/archiver`
- Load metadata → `internal/metadata`
- Interactive TUI mode: pick/drop files, packages, categories → `internal/tui`
- Copy files to original paths
- Restore permissions, symlinks
- Restore packages → `internal/recovery`

**4. Advanced**
- Compare backups → `cmd/diff.go`, `internal/diff`
- Cloud sync → `cmd/sync.go`, `internal/cloud`
- Merge incremental → `cmd/optimize.go`, `internal/incremental`
- List git repos needing attention → `cmd/remind.go`, `internal/gittracker`
- Manage notes → `cmd/backup_notes.go`, `internal/metadata`

### Core Components

**Encryption (`internal/crypto`)**
- Uses [age](https://github.com/FiloSottile/age) (X25519 + ChaCha20-Poly1305)
- Key generation, encryption, decryption
- Key stored with 0600 permissions
- Tested with large files, wrong keys, missing keys

**Archiver (`internal/archiver`)**
- Tar.gz creation/extraction
- Smart exclusions (node_modules, cache, logs, symlinks)
- Path traversal protection
- Permission preservation

**Metadata (`internal/metadata`)**
- JSON manifest of all files
- SHA256 checksums
- Original paths, permissions, sizes
- Package counts, backup size
- Backup notes (for `backup_notes.go`)

**Finder (`internal/finder`)**
- Dotfiles discovery (starts with `.`)
- Secret dirs (`~/.ssh`, `~/.gnupg`, `~/.aws`)
- `.env` and `.pem` file search with exclusions
- Symlink handling
- Config discovery (`~/.config`)

**Packager (`internal/packager`)**
- Homebrew → `Brewfile`
- Mac App Store → `mas list`
- VS Code → extension list
- npm → global packages

**Config (`internal/config`)**
- Load/save `~/.stash.yaml`
- Path expansion (`~` → home dir)
- Validate exclude patterns
- Default values for missing fields

**TUI (`internal/tui`)**
- Interactive restore picker
- Multi-select categories
- File/package pick/drop
- Git-rebase style editor integration

**Cloud (`internal/cloud`)**
- S3 and cloud storage adapters
- Upload/download backups
- Stream large files

**Incremental (`internal/incremental`)**
- Diff-based backups (faster, smaller)
- Merge incremental into full backups
- Maintain backup chains

---

## 🧪 Testing

### Running Tests

```bash
# All tests
make test

# Specific package
go test ./internal/crypto -v

# With coverage
go test ./... -cover

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Structure

- **48+ packages** with dedicated test files
- **180+ test cases** across codebase
- Unit tests for all core logic (crypto, archiver, finder, config, metadata, etc.)
- Integration tests for backup/restore flows
- Mock file systems using `t.TempDir()`

### Writing Tests

Follow existing patterns:

```go
func TestFeature(t *testing.T) {
    // Setup
    tempDir := t.TempDir()
    
    // Create test fixtures
    testFile := filepath.Join(tempDir, "test.txt")
    os.WriteFile(testFile, []byte("content"), 0644)
    
    // Execute
    result, err := YourFunction(testFile)
    
    // Assert
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

---

## 📂 Backup Structure

```
backup-2024-12-27-153045.tar.gz.age (encrypted)
└── backup-2024-12-27-153045.tar.gz
    └── backup-2024-12-27-153045/
        ├── metadata.json              # File manifest + notes
        ├── README.txt                 # Backup info
        ├── dotfiles/                  # Home dotfiles
        │   ├── .bashrc
        │   ├── .zshrc
        │   └── .gitconfig
        ├── ssh/                       # ~/.ssh
        ├── gpg/                       # ~/.gnupg
        ├── aws/                       # ~/.aws
        ├── config/                    # ~/.config (smart exclusions)
        ├── env-files/                 # All .env files
        ├── pem-files/                 # All .pem files
        ├── docker/                    # ~/.docker
        ├── kubernetes/                # ~/.kube
        ├── fonts/                     # ~/Library/Fonts
        ├── browser-data/              # Chrome/Safari data (optional)
        ├── git-repos/                 # Git repo list
        └── packages/
            ├── Brewfile
            ├── mas-apps.txt
            ├── vscode-extensions.txt
            └── npm-global.txt
```

### metadata.json Structure

```json
{
  "version": "1.4.0",
  "timestamp": "2024-12-27T15:30:45Z",
  "hostname": "macbook-pro",
  "username": "user",
  "encrypted": true,
  "notes": "Manual backup before reset",
  "files": [
    {
      "original_path": "/Users/user/.bashrc",
      "backup_path": "dotfiles/.bashrc",
      "size": 2048,
      "mode": 420,
      "mod_time": "2024-12-27T10:00:00Z",
      "checksum": "sha256-hash",
      "is_dir": false
    }
  ],
  "package_counts": {
    "Homebrew": 42,
    "MAS": 5,
    "VSCode": 10,
    "npm": 8
  },
  "backup_size": 15728640
}
```

---

## 🛠️ Development

### Dependencies

```go
require (
    filippo.io/age v1.2.1              # Encryption
    github.com/spf13/cobra v1.10.2     # CLI framework
    github.com/spf13/viper v1.21.0     # Config management
    gopkg.in/yaml.v3 v3.0.1            # YAML parsing
)
```

### Build Tags

None. macOS-only features use runtime checks.

### Environment Variables

- `EDITOR` / `VISUAL` - Used for interactive restore and config editing
- `HOME` - User home directory
- `STASH_INSTALL_DIR` - Override installation directory
- `AWS_*` - AWS credentials (for S3 cloud storage)

### Common Tasks

```bash
# Format code
go fmt ./...

# Lint
go vet ./...

# Check for issues
staticcheck ./...

# Build for current platform
make build

# Build for release
goreleaser build --snapshot --clean

# Install locally
go install

# Clean
make clean

# Run with verbose output
./stash backup --verbose

# Dry run (preview changes)
./stash backup --dry-run
./stash restore 1 --dry-run
```

---

## 🚢 Release Process

1. Update version in `cmd/root.go` (Version field)
2. Commit changes:
   ```bash
   git add -A
   git commit -m "feat: description of changes"
   ```
3. Create annotated tag:
   ```bash
   git tag -a v1.5.0 -m "Release v1.5.0"
   ```
4. Push to GitHub:
   ```bash
   git push origin v1.5.0
   ```
5. GitHub Actions runs tests + GoReleaser
6. Release created with binaries, checksums, release notes

### GoReleaser Configuration

Configured in `.goreleaser.yml`:
- Builds for `darwin/amd64` (Intel Macs)
- Builds for `darwin/arm64` (Apple Silicon)
- Archives include binary, README, LICENSE, CONTRIBUTING
- Auto-updates Homebrew tap: `harshpatel5940/tap/stash`

---

## 🎨 Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write tests for new features
- Keep functions small and focused
- Document exported functions with comments

### Naming Conventions

- **Commands**: lowercase (`backup`, `restore`, `list`)
- **Packages**: lowercase single word (`crypto`, `finder`)
- **Exported**: PascalCase (`FindFiles`, `NewConfig`)
- **Internal**: camelCase (`findFiles`, `newConfig`)
- **Constants**: ALL_CAPS (`MAX_BACKUP_SIZE`, `DEFAULT_KEEP_COUNT`)

### Error Handling

Use custom error types from `internal/errors`:
```go
import stasherrors "github.com/harshpatel5940/stash/internal/errors"

// Return descriptive errors
if err := someOp(); err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

---

## 🔄 Commands Reference

All commands implemented in `cmd/`:

| Command | File | Purpose |
|---------|------|---------|
| `init` | `init.go` | Setup config, encryption key, dependencies |
| `backup` | `backup.go` | Create encrypted backup |
| `backup-notes` | `backup_notes.go` | Add/edit notes on backup |
| `restore` | `restore.go` | Restore from encrypted backup |
| `list` | `list.go` | List available backups |
| `info` | `info.go` | Show backup metadata |
| `diff` | `diff.go` | Compare two backups |
| `cleanup` | `cleanup.go` | Delete old backups by retention policy |
| `optimize` | `optimize.go` | Merge incremental backups |
| `remind` | `remind.go` | List git repos needing attention |
| `config` | `config.go` | Edit/view configuration |
| `sync` | `sync.go` | Upload/download from cloud |

---

## 🐛 Debugging

### Enable Verbose Logging

```bash
stash backup --verbose
stash restore 1 --verbose
```

### Common Issues

**Encryption fails:**
```bash
# Check key permissions
ls -la ~/.stash.key

# Regenerate key (careful!)
rm ~/.stash.key && stash init
```

**Backup too large:**
- Review `exclude` patterns in `~/.stash.yaml`
- Add patterns for cache/build dirs: `*/cache/*`, `*/.next/*`

**Symlinks cause issues:**
- Symlinks are skipped by default
- Check `internal/archiver` for exclusion logic

**Permission denied errors:**
- Run `stash init --skip-deps` to install without sudo
- Check `~/.stash.yaml` paths are readable

**Cloud sync fails:**
- Verify AWS credentials (for S3)
- Check `internal/cloud` for adapter setup

---

## 📝 Adding New Features

### Example: Add New Backup Source

**1. Create internal package** (`internal/newsource/newsource.go`):
```go
package newsource

type Finder struct {
    homeDir string
}

func NewFinder(homeDir string) *Finder {
    return &Finder{homeDir: homeDir}
}

func (f *Finder) Find() ([]string, error) {
    // Discover and return paths
    return paths, nil
}
```

**2. Create tests** (`internal/newsource/newsource_test.go`):
```go
func TestFind(t *testing.T) {
    tempDir := t.TempDir()
    finder := NewFinder(tempDir)
    
    paths, err := finder.Find()
    if err != nil {
        t.Fatalf("Find failed: %v", err)
    }
    if len(paths) == 0 {
        t.Error("Expected to find paths")
    }
}
```

**3. Integrate into backup** (`cmd/backup.go`):
```go
import "github.com/harshpatel5940/stash/internal/newsource"

// In backup function:
finder := newsource.NewFinder(homeDir)
newPaths, err := finder.Find()
// Add newPaths to archive...
```

**4. Update README.md** with new feature

**5. Add config option** to `~/.stash.yaml` if needed

### Example: Add New Command

**1. Create command file** (`cmd/newcommand.go`):
```go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
    Use:   "newcommand",
    Short: "Brief description",
    Long:  "Detailed description...",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
        return nil
    },
}

func init() {
    rootCmd.AddCommand(newCmd)
    newCmd.Flags().StringVar(&someFlag, "flag", "default", "Flag help")
}
```

**2. Add tests** (`cmd/newcommand_test.go` or `cmd_test.go`)

**3. Register in `cmd/root.go`** if not auto-registered

---

## 🤝 Pull Request Guidelines

1. **One feature/fix per PR**
2. **Write tests** for new code (aim for >80% coverage)
3. **Update docs** if behavior changes
4. **Follow commit conventions**:
   - `feat: add new feature`
   - `fix: resolve issue`
   - `docs: update documentation`
   - `test: add tests`
   - `refactor: improve code structure`
   - `chore: maintenance tasks`
5. **Keep PRs small** (<500 lines of code)
6. **Describe changes** in PR description with context
7. **Test locally** before pushing:
   ```bash
   make test
   make build
   ./stash --help
   ```

---

## 🔐 Security

- **Never commit** `.stash.key` (add to `.gitignore`)
- **Use 0600** for sensitive files (done by `crypto.go`)
- **Validate all paths** to prevent traversal attacks (use `security.go`)
- **Skip symlinks** to avoid circular references (archiver.go)
- **Age encryption** is audited and battle-tested; don't implement custom crypto
- **Never log** sensitive data (keys, credentials, paths)

---

## ⚠️ Known Limitations

- **macOS only**: Uses `mas`, BSD tar/gzip, macOS-specific APIs
- **No incremental (yet)**: Full backups each time (incremental support in `internal/incremental`)
- **Manual package install**: Restore doesn't auto-install packages during restore
- **No cloud sync by default**: User can opt-in via config
- **Keychain not backed up**: Requires manual export/import
- **TCC permissions manual**: Full Disk Access must be granted manually

---

## 🔮 Future Ideas

- Auto-sync to S3/Dropbox
- Incremental backups enabled by default
- Backup verification (checksum validation)
- Linux support
- Compression options (zstd, xz)
- Backup rotation/retention UI
- Remote backup restores
- Backup encryption with passphrase (not just key)
- Automated scheduled backups

---

## 📊 Project Stats

- **Version**: Check `cmd/root.go` for current version
- **Languages**: Go
- **Platforms**: macOS (Intel + Apple Silicon)
- **Test Coverage**: 180+ tests across 48+ packages
- **Dependencies**: 3 main dependencies (age, cobra, viper)

---

## 📧 Getting Help

- **Issues**: [Bug reports, feature requests](https://github.com/harshpatel5940/stash/issues)
- **Discussions**: Questions, ideas, design feedback
- **Pull Requests**: Contributions welcome!

---

## 📜 License

MIT License - see [LICENSE](LICENSE)

Contributions are licensed under same terms.

---

**Happy Contributing! 🎉**

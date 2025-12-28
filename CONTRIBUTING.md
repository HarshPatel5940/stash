# Contributing to Stash

Thanks for your interest in contributing! This guide covers installation, development setup, architecture, and guidelines.

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
# Add tap (once available)
brew tap harshpatel5940/stash

# Install
brew install stash

# Update
brew upgrade stash
```

**Setting up Homebrew Tap:**

See the [Release Process](#-release-process) section for details on how releases automatically update the Homebrew formula.

### From Binary (Release)

Download from [GitHub Releases](https://github.com/harshpatel5940/stash/releases):

- `stash_VERSION_darwin_amd64.tar.gz` - Intel Macs
- `stash_VERSION_darwin_arm64.tar.gz` - Apple Silicon (M1/M2/M3+)

```bash
# Extract
tar -xzf stash_*.tar.gz

# Move to PATH
sudo mv stash /usr/local/bin/

# Verify
stash --version
```

### From Source

```bash
git clone https://github.com/harshpatel5940/stash.git
cd stash
make build
sudo make install
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
```

---

## 📁 Project Structure

```
stash/
├── cmd/                    # CLI commands (cobra)
│   ├── root.go            # Root command setup
│   ├── init.go            # Initialize config + key
│   ├── backup.go          # Create backups
│   ├── restore.go         # Restore from backup
│   └── list.go            # List available backups
├── internal/
│   ├── config/            # Config management
│   │   ├── config.go      # Load/save .stash.yaml
│   │   └── config_test.go
│   ├── crypto/            # Age encryption
│   │   ├── crypto.go      # Encrypt/decrypt operations
│   │   └── crypto_test.go
│   ├── archiver/          # Tar.gz operations
│   │   ├── archiver.go    # Create/extract archives
│   │   └── archiver_test.go
│   ├── metadata/          # Backup metadata
│   │   ├── metadata.go    # File info, checksums
│   │   └── metadata_test.go
│   ├── finder/            # File discovery
│   │   ├── finder.go      # Find dotfiles, .env, .pem
│   │   └── finder_test.go
│   └── packager/          # Package lists
│       └── packager.go    # Homebrew, MAS, VSCode, npm
├── .github/workflows/     # CI/CD
│   ├── test.yml           # Run tests on PR/push
│   └── release.yml        # GoReleaser on tag
├── .goreleaser.yml        # Release configuration
├── Makefile               # Build automation
├── go.mod                 # Dependencies
└── main.go                # Entry point
```

---

## 🏗️ Architecture

### Backup Flow

1. **Init** (`cmd/init.go`)
   - Generate age key → `~/.stash.key`
   - Create config → `~/.stash.yaml`

2. **Backup** (`cmd/backup.go`)
   - Load config (`internal/config`)
   - Find files (`internal/finder`)
   - Collect packages (`internal/packager`)
   - Create metadata (`internal/metadata`)
   - Archive to tar.gz (`internal/archiver`)
   - Encrypt with age (`internal/crypto`)
   - Output: `backup-TIMESTAMP.tar.gz.age`

3. **Restore** (`cmd/restore.go`)
   - Decrypt `.age` file (`internal/crypto`)
   - Extract tar.gz (`internal/archiver`)
   - Load metadata (`internal/metadata`)
   - Interactive mode: open editor for pick/drop
   - Copy files to original paths
   - Restore permissions

4. **List** (`cmd/list.go`)
   - Scan backup directory
   - Show backups with size, date, encryption status

### Key Components

**Encryption (`internal/crypto`)**
- Uses [age](https://github.com/FiloSottile/age) (X25519 + ChaCha20-Poly1305)
- Key generation, encrypt, decrypt
- Key stored with 0600 permissions

**Archiver (`internal/archiver`)**
- Tar.gz creation/extraction
- Smart exclusions (node_modules, cache, logs, symlinks)
- Path traversal protection
- Permission preservation

**Metadata (`internal/metadata`)**
- JSON manifest of all files
- SHA256 checksums
- Original paths, permissions, sizes
- Package counts

**Finder (`internal/finder`)**
- Dotfiles discovery (starts with `.`)
- Secret dirs (`~/.ssh`, `~/.gnupg`, `~/.aws`)
- `.env` and `.pem` file search with exclusions
- Symlink handling

**Packager (`internal/packager`)**
- Homebrew → `Brewfile`
- Mac App Store → `mas list`
- VS Code → extension list
- npm → global packages

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
```

### Test Coverage

- **archiver**: 10 tests (tar.gz, exclusions, symlinks, path traversal)
- **config**: 4 tests (defaults, save, expand paths, exclusions)
- **crypto**: 8 tests (key gen, encrypt/decrypt, large files, wrong key)
- **finder**: 13 tests (dotfiles, secrets, .env, .pem, exclusions)
- **metadata**: 13 tests (add files, checksums, save/load, formatting)

**Total: 48 tests**

### Writing Tests

Follow existing patterns:

```go
func TestFeature(t *testing.T) {
    tempDir := t.TempDir() 
    
    
    testFile := filepath.Join(tempDir, "test.txt")
    os.WriteFile(testFile, []byte("content"), 0644)
    
    
    result, err := YourFunction(testFile)
    
    
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
        ├── metadata.json          # File manifest
        ├── README.txt             # Backup info
        ├── dotfiles/              # Home dotfiles
        │   ├── .bashrc
        │   ├── .zshrc
        │   └── .gitconfig
        ├── ssh/                   # ~/.ssh
        ├── gpg/                   # ~/.gnupg
        ├── aws/                   # ~/.aws
        ├── config/                # ~/.config (smart exclusions)
        ├── env-files/             # All .env files
        ├── pem-files/             # All .pem files
        └── packages/
            ├── Brewfile
            ├── mas-apps.txt
            ├── vscode-extensions.txt
            └── npm-global.txt
```

### metadata.json

```json
{
  "version": "1.0.0",
  "timestamp": "2024-12-27T15:30:45Z",
  "hostname": "macbook-pro",
  "username": "user",
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
    "NPM": 8
  },
  "backup_size": 15728640
}
```

---

## 🛠️ Development

### Dependencies

```go
require (
    filippo.io/age v1.2.1              
    github.com/spf13/cobra v1.10.2     
    github.com/spf13/viper v1.21.0     
    gopkg.in/yaml.v3 v3.0.1            
)
```

### Build Tags

None currently. macOS-only features use runtime checks.

### Environment Variables

- `EDITOR` / `VISUAL` - Used for interactive restore
- `HOME` - User home directory

### Common Tasks

```bash
# Format code
go fmt ./...

# Lint
go vet ./...

# Build for release
goreleaser build --snapshot --clean

# Install locally
go install

# Clean
make clean
```

---

## 🚢 Release Process

1. Update version in code (if needed)
2. Commit changes: `git commit -am "feat: new feature"`
3. Tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
4. Push: `git push origin v1.0.0`
5. GitHub Actions runs tests + GoReleaser
6. Release created with binaries

### GoReleaser

Builds for:
- `darwin/amd64` (Intel Macs)
- `darwin/arm64` (Apple Silicon)

Archives include:
- Binary
- README.md
- LICENSE
- CONTRIBUTING.md

---

## 🎨 Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write tests for new features
- Keep functions small and focused
- Document exported functions

### Naming

- Commands: lowercase (backup, restore, list)
- Packages: lowercase single word (crypto, finder)
- Exported: PascalCase
- Internal: camelCase

---

## 🐛 Debugging

### Enable Verbose Logging

```bash
stash backup --verbose
```

### Common Issues

**Encryption fails:**
- Check `~/.stash.key` exists
- Verify 0600 permissions
- Regenerate: `rm ~/.stash.key && stash init`

**Backup too large:**
- Review `exclude` patterns in `~/.stash.yaml`
- Add more patterns for cache/build dirs

**Symlinks cause issues:**
- Symlinks are skipped by default
- Check `internal/archiver` exclusion logic

---

## 🔄 Interactive Restore

Git-rebase style pick/drop:

```
# Opens in $EDITOR
pick [FILE] ~/.bashrc (2.3 KB)
drop [FILE] ~/.ssh/id_rsa (3.2 KB)
pick [DIR ] ~/.config (0 B)
```

Implementation: `cmd/restore.go` → `interactivePickFiles()`

Parse logic:
1. Generate plan file with all files as `pick`
2. Open in editor
3. User edits (pick → drop)
4. Parse edited file
5. Restore only `pick` lines

---

## 📝 Adding New Features

### Example: Add new backup source

1. **Finder** (`internal/finder/finder.go`)
   ```go
   func (f *Finder) FindNewFiles() ([]string, error) {
       
   }
   ```

2. **Backup** (`cmd/backup.go`)
   ```go
   func backupNewFiles(tempDir string, meta *metadata.Metadata, ...) error {
       files := finder.FindNewFiles()
       
       
   }
   ```

3. **Test** (`internal/finder/finder_test.go`)
   ```go
   func TestFindNewFiles(t *testing.T) {
       
   }
   ```

4. **Update README** with new feature

---

## 🤝 Pull Request Guidelines

1. **One feature per PR**
2. **Write tests** for new code
3. **Update docs** if needed
4. **Follow commit conventions**:
   - `feat:` new feature
   - `fix:` bug fix
   - `docs:` documentation
   - `test:` tests
   - `refactor:` code refactoring
5. **Keep PR small** (<500 lines)
6. **Describe changes** in PR description

---

## 🔐 Security

- Never commit `.stash.key`
- Use `0600` for sensitive files
- Validate all file paths (prevent traversal)
- Skip symlinks to avoid circular refs
- Age encryption is audited, don't roll custom crypto

---

## ⚠️ Known Limitations

- **macOS only**: Uses `mas`, assumes BSD tar/gzip
- **No incremental**: Full backup every time
- **Manual package install**: Restore doesn't auto-install packages
- **No cloud sync**: User handles upload/download

---

## 🔮 Future Ideas

- Incremental backups (rsync-style)
- Cloud backends (S3, Dropbox)
- Selective restore (only SSH, only .env)
- Backup verification (checksum validation)
- Linux support
- Compression options (zstd, xz)
- Backup rotation/retention

---

## 📧 Getting Help

- **Issues**: Bug reports, feature requests
- **Discussions**: Questions, ideas
- **Discord/Slack**: (if created)

---

## 📜 License

MIT License - see [LICENSE](LICENSE)

Contributions are licensed under same terms.

---

**Happy Contributing! 🎉**

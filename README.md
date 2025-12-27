# 📦 Stash

[![Test and Build](https://github.com/harshpatel5940/stash/actions/workflows/test.yml/badge.svg)](https://github.com/harshpatel5940/stash/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Stash your Mac, restore anywhere.**

A simple Go CLI tool that helps you quickly reset your Mac and restore to a fresh state with all apps, configs, and secrets intact.

## 🚀 Features

- **Dotfiles Backup**: `.zshrc`, `.gitconfig`, `.vimrc`, and all other dotfiles
- **Secret Files**: SSH keys, GPG keys, AWS credentials
- **App Configs**: `~/.config` directory with smart exclusions (skips `node_modules`, cache, etc.)
- **Environment Files**: Recursive search for `.env` files across your projects
- **Certificate Files**: Find and backup `.pem` files with path preservation
- **Package Lists**: Homebrew Brewfile, Mac App Store apps, VS Code extensions, npm globals
- **Encrypted Backups**: All backups encrypted with [age](https://github.com/FiloSottile/age)
- **Smart Restore**: Files restored to original paths with proper permissions

## 📥 Installation

### Using Go Install (Recommended)

```bash
go install github.com/harshpatel5940/stash@latest
```

Make sure `$GOPATH/bin` is in your `PATH`:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Using Homebrew

```bash
# Coming soon - will be available via homebrew tap
brew tap harshpatel5940/stash
brew install stash
```

### From Source

```bash
git clone https://github.com/harshpatel5940/stash.git
cd stash
make build
sudo make install
```

## 🎯 Quick Start

### 1. Initialize

Generate encryption key and config file:

```bash
stash init
```

This creates:
- `~/.stash.yaml` - Configuration file
- `~/.stash.key` - Encryption key (keep this safe!)

### 2. Preview

See what would be backed up:

```bash
stash list
```

### 3. Backup

Create your first backup:

```bash
stash backup
```

Backups are saved to `~/stash-backups/` by default as encrypted `.tar.gz.age` files.

### 4. Restore

On your new Mac:

```bash
# Copy your .stash.key to the new Mac first!
stash restore ~/Downloads/backup-2024-12-27-153045.tar.gz.age
```

## 📚 Commands

### `stash init`

Initialize stash with default config and generate encryption key.

```bash
stash init
```

### `stash list`

Preview what would be backed up (dry-run mode).

```bash
stash list [--config ~/.stash.yaml]
```

### `stash backup`

Create a new encrypted backup.

```bash
stash backup [options]

Options:
  -o, --output string             Output directory (default: ~/stash-backups)
  -k, --encrypt-key string        Encryption key path (default: ~/.stash.key)
      --no-encrypt          Skip encryption (not recommended)
```

### `stash restore`

Restore from an encrypted backup.

```bash
stash restore <backup-file> [options]

Options:
  -k, --decrypt-key string  Decryption key path (default: ~/.stash.key)
      --dry-run            Preview what would be restored
      --interactive        Ask before restoring each file
      --no-decrypt         Backup is not encrypted
```

## ⚙️ Configuration

Edit `~/.stash.yaml` to customize:

```yaml
# Directories to search for .env and .pem files
search_paths:
  - ~/projects
  - ~/work
  - ~/Documents

# Exclude patterns (glob-style)
exclude:
  - "*/node_modules/*"
  - "*/vendor/*"
  - "*/.git/*"
  - "*/.next/*"
  - "*/dist/*"
  - "*/build/*"

# Additional dotfiles to backup
additional_dotfiles:
  - .custom_aliases
  - .local_config

# Backup output directory
backup_dir: ~/stash-backups

# Encryption key location
encryption_key: ~/.stash.key
```

## 📂 Backup Structure

```
backup-2024-12-27-153045/
├── metadata.json          # Paths, permissions, checksums
├── dotfiles/              # Home directory dotfiles
│   ├── .zshrc
│   ├── .gitconfig
│   └── ...
├── ssh/                   # SSH keys and config
├── gpg/                   # GPG keys
├── aws/                   # AWS credentials
├── config/                # ~/.config (smart: excludes node_modules, cache)
├── env-files/             # All .env files found
├── pem-files/             # All .pem files found
├── packages/              # Package manager dumps
│   ├── Brewfile
│   ├── mas-apps.txt
│   ├── vscode-extensions.txt
│   └── npm-global.txt
└── README.txt             # Backup information
```

## 📂 Smart .config Backup

Stash intelligently backs up your `~/.config` directory while automatically excluding:

- `node_modules/` - Node.js dependencies
- `cache/`, `Cache/` - Application caches
- `tmp/`, `temp/` - Temporary files
- `logs/`, `log/` - Log files
- Symlinks - Avoided to prevent circular references
- `.git/` - Git repositories
- Virtual environments (`venv/`, `.venv/`, `__pycache__/`)

**What gets backed up:**
- Fish, Zsh, Bash configurations
- Neovim/Vim settings
- Git configurations (gh, git-credential-manager)
- Editor configs (VS Code, Zed, etc.)
- Terminal configs
- Application settings (without bloat)

This keeps backups small and fast while preserving important configurations.

## 🔐 Security

⚠️ **CRITICAL: You need BOTH to restore:**
- 🔑 `~/.stash.key` - encryption key 
- 📦 `.tar.gz.age` backup file(s)

**Without BOTH, restore is IMPOSSIBLE!**

### Encryption Options

**Age encryption (default):**
- All backups encrypted using **age** (modern encryption tool)
- Private key stored at `~/.stash.key` (600 permissions)
- Generated automatically with `stash init`

**GPG encryption (optional):**
```bash
stash backup --use-gpg --gpg-recipient your-email@example.com
```
- Uses your existing GPG keys (e.g., from git commit signing)
- No need for separate `.stash.key`
- Restore requires GPG private key on new system

### Storage Best Practices

**Encryption key:**
- Password manager (1Password, Bitwarden, etc.)
- Secure cloud vault (separate from backup files)
- USB drive in safe location
- Multiple copies in different secure locations

**Backup files (.age):**
- Cloud storage (Dropbox, Google Drive, iCloud, OneDrive)
- External hard drive
- NAS (Network Attached Storage)
- Multiple locations for redundancy

**Never store both in same single location!**

## 🔄 Restore Workflow

After restoring on a new Mac:

1. **Install Homebrew** (if not already installed):
   ```bash
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   ```

2. **Restore packages** from the backup's `packages/` directory:
   ```bash
   # Homebrew packages
   brew bundle --file=~/stash-backups/backup-*/packages/Brewfile
   
   # VS Code extensions
   cat packages/vscode-extensions.txt | xargs -L 1 code --install-extension
   
   # npm global packages (review the list first)
   # Edit npm-global.txt to extract package names, then:
   npm install -g <package-name>
   ```

3. **Restart terminal** to load new configs

4. **Test connections**: SSH, AWS CLI, etc.

## 🛠️ Development

### Build

```bash
go build -o stash
```

### Run Tests

```bash
go test ./...
```

### Project Structure

```
stash/
├── cmd/                    # CLI commands
│   ├── root.go
│   ├── backup.go
│   ├── restore.go
│   ├── list.go
│   └── init.go
├── internal/
│   ├── finder/            # Find dotfiles, .env, .pem files
│   ├── packager/          # Collect package lists
│   ├── archiver/          # tar.gz operations
│   ├── crypto/            # age encryption
│   ├── metadata/          # Backup metadata
│   └── config/            # Config management
├── go.mod
├── go.sum
└── main.go
```

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## 📝 License

MIT License - see [LICENSE](LICENSE) file

## 💡 Tips

- Run `stash list` before backup to see what will be included
- Store encryption key in multiple secure locations
- Test restore process before wiping your Mac
- Review excluded patterns in config for your workflow
- Use `--dry-run` with restore to preview changes

## ⚠️ Known Limitations

- macOS only (uses Mac-specific tools like `mas`)
- Requires manual package installation after restore
- Large `.config` directories may slow down backup
- No cloud sync (manual file transfer required)

## 🔮 Roadmap

- [ ] Cloud sync (S3, Dropbox, iCloud)
- [ ] Incremental backups
- [ ] Selective restore (only SSH, only .env, etc.)
- [ ] Backup diff viewer
- [ ] Auto-backup scheduling
- [ ] Linux support
- [ ] Backup verification/validation

## 📧 Support

- Issues: [GitHub Issues](https://github.com/harshpatel5940/stash/issues)
- Discussions: [GitHub Discussions](https://github.com/harshpatel5940/stash/discussions)

---

Made with ❤️ by [Harsh Patel](https://github.com/harshpatel5940)

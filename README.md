# 📦 Stash

[![Test and Build](https://github.com/harshpatel5940/stash/actions/workflows/test.yml/badge.svg)](https://github.com/harshpatel5940/stash/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Backup your Mac. Restore anywhere.**

Encrypted backup CLI for macOS dotfiles, secrets, configs, and package lists.

## ⚠️ CRITICAL: Backup Safety

**You need BOTH to restore:**
- 🔑 `~/.stash.key` (encryption key)
- 📦 `.tar.gz.age` (backup file)

**Storage:**
- Key → Password manager, USB drive, secure vault
- Backup → Cloud storage, external drive, NAS

**Store separately. Without both, restore is impossible.**

---

## 🚀 Quick Start

### Install

```bash
go install github.com/harshpatel5940/stash@latest
```

### Usage

```bash
# 1. Initialize (creates key + config)
stash init

# 2. Preview what gets backed up
stash backup --dry-run

# 3. Create backup
stash backup

# 4. List backups
stash list

# 5. Restore (on new Mac)
stash restore backup-2024-12-27-153045.tar.gz.age --interactive
```

---

## 📚 Commands

### `stash init`
Generate encryption key and config.

### `stash backup`
Create encrypted backup.

**Flags:**
- `--dry-run` - Preview without creating backup
- `--verbose` - Detailed output for debugging
- `--no-encrypt` - Skip encryption (not recommended)
- `-o` - Output directory (default: `~/stash-backups`)

### `stash list`
Show all available backups with details.

### `stash restore <file>`
Restore from backup.

**Flags:**
- `--dry-run` - Preview without restoring
- `--interactive` - Pick/drop files in editor (git-rebase style)
- `--no-decrypt` - Backup not encrypted

**Interactive mode:**
Opens editor with pick/drop list. Change `pick` → `drop` to skip files.

```
pick [FILE] ~/.bashrc (2.3 KB)
drop [FILE] ~/.ssh/id_rsa (3.2 KB)  # Changed to drop
pick [DIR ] ~/.config (0 B)
```

---

## 📦 What Gets Backed Up

- **Dotfiles**: `.zshrc`, `.bashrc`, `.gitconfig`, `.vimrc`, etc.
- **Secrets**: `~/.ssh`, `~/.gnupg`, `~/.aws`
- **Configs**: `~/.config` (smart exclusions: no `node_modules`, cache, logs)
- **Environment**: `.env` files from projects
- **Certificates**: `.pem` files
- **Packages**: Homebrew, MAS, VS Code, npm lists

---

## ⚙️ Configuration

Edit `~/.stash.yaml`:

```yaml
search_paths:
  - ~/projects
  - ~/work

exclude:
  - "*/node_modules/*"
  - "*/vendor/*"
  - "*/.git/*"

additional_dotfiles:
  - .custom_aliases

backup_dir: ~/stash-backups
encryption_key: ~/.stash.key
```

---

## 🔐 Security

- **Encryption**: age (modern, simple, secure)
- **Key permissions**: 600 (owner read/write only)
- **No plaintext**: All backups encrypted by default

---

## 🔄 After Restore

```bash
# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Restore packages
brew bundle --file=packages/Brewfile
cat packages/vscode-extensions.txt | xargs -L 1 code --install-extension

# Restart terminal
# Test SSH, AWS, etc.
```

---

## 🧪 Testing

48 tests covering crypto, archiver, metadata, finder, config.

```bash
make test
```

---

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, architecture, and guidelines.

---

## 📝 License

MIT - see [LICENSE](LICENSE)

---

**Made with ❤️ by [Harsh Patel](https://github.com/harshpatel5940)**
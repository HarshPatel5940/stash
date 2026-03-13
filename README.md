# 📦 Stash

[![Test](https://github.com/harshpatel5940/stash/actions/workflows/test.yml/badge.svg)](https://github.com/harshpatel5940/stash/actions/workflows/test.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Encrypted backup for macOS dotfiles, secrets, and configs.

---

## ⚠️ Critical

**Need BOTH to restore:**
- 🔑 `~/.stash.key` → Password manager
- 📦 `.tar.gz.age` → Cloud/external drive

**Store separately. Lose either = lose everything.**

---

## Install

**Homebrew:**
```bash
brew install harshpatel5940/tap/stash
```

**Go:**
```bash
go install github.com/harshpatel5940/stash@latest
```

---

## Usage

```bash
# Setup
stash init

# Backup
stash backup

# List backups
stash list

# Restore (copy .stash.key first!)
stash restore backup.tar.gz.age
```

---

## What Gets Backed Up

- **Dotfiles**: Shell configs (`.zshrc`), git configs, etc.
- **Secrets**: SSH keys, GPG keys, AWS credentials.
- **Dev Secrets**: `.env` and `.pem` files from your projects.
- **Configs**: `~/.config` (with smart exclusions like `node_modules`).
- **Packages**: Homebrew, npm, VS Code extensions, Mac App Store apps.
- **Browser Data**: Bookmarks, extensions, settings (Chrome, Firefox, Safari, Arc).
- **Git Repos**: Tracks all your git repositories for easy re-cloning.
- **System**: macOS defaults/preferences, custom fonts, shell history.

---

## Flags

**Backup:**
- `--skip-browsers` - Skip browser data (saves space)
- `--keep <n>` - Keep only last N backups (default: 5)
- `--dry-run` - Preview what will be backed up
- `--verbose` - Detailed output
- `--no-encrypt` - Skip encryption (not recommended)

**Restore:**
- `--dry-run` - Preview
- `--editor` - Pick/drop files and packages in editor (git-rebase style)
- `--no-tui` - Use Y/n prompts instead of interactive TUI
- `--no-decrypt` - Unencrypted backup

---

## Interactive Restore

By default, restore opens an interactive TUI to select what to restore:

1. **Choose categories**: multi-select across dotfiles, Homebrew, VS Code, macOS defaults, etc.
2. **Pick files**: if dotfiles selected, choose individual files to restore
3. **Pick packages**: if Homebrew selected, choose to install all or pick individual packages

Use `--editor` for a git-rebase style text editor instead:

```
pick [BREW] Install Homebrew packages
drop [MAS ] Install Mac App Store apps
pick [CODE] Install VS Code extensions

pick [FILE] ~/.bashrc (2.3 KB)
drop [FILE] ~/.ssh/id_rsa (skip this)
pick [DIR ] ~/.config
```

Change `pick` → `drop` to skip. Save & close.

---

## Config

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

## After Restore

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

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md)

```bash
make build
make test
```

---

## License

MIT - see [LICENSE](LICENSE)

---

**[GitHub](https://github.com/harshpatel5940/stash)** • **[Issues](https://github.com/harshpatel5940/stash/issues)**

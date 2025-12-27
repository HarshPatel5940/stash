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
stash restore backup.tar.gz.age --interactive
```

---

## What Gets Backed Up

- Dotfiles (`.zshrc`, `.gitconfig`, `.vimrc`, etc.)
- Secrets (`~/.ssh`, `~/.gnupg`, `~/.aws`)
- Configs (`~/.config` - smart exclusions)
- `.env` and `.pem` files
- Package lists (Homebrew, npm, VS Code, MAS)

---

## Flags

**Backup:**
- `--dry-run` - Preview
- `--verbose` - Debug output
- `--no-encrypt` - Skip encryption

**Restore:**
- `--dry-run` - Preview
- `--interactive` - Pick/drop files (git-rebase style)
- `--no-decrypt` - Unencrypted backup

---

## Interactive Restore

Opens editor with pick/drop list:

```
pick [FILE] ~/.bashrc (2.3 KB)
drop [FILE] ~/.ssh/id_rsa (skip this)
pick [DIR ] ~/.config
```

Change `pick` → `drop` to skip files. Save & close.

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
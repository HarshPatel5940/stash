# 📦 Stash

[![Test](https://github.com/harshpatel5940/stash/actions/workflows/test.yml/badge.svg)](https://github.com/harshpatel5940/stash/actions/workflows/test.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Encrypted backup for macOS dotfiles, secrets, and configs.

---

## why?

No unified tool. Pieces exist (Homebrew recovery, dotfile repos, secret managers) but nothing automates everything together.

## How?

**Stash** encrypts & backs up your entire macOS setup (dotfiles, secrets, packages, system prefs, git repos). Restore selectively, if needed. Encryption with `age`, split key storage for security.

## What's Covered

- **Dotfiles**: `.zshrc`, git configs, shell aliases
- **Secrets**: SSH keys, GPG keys, AWS credentials
- **Dev Secrets**: `.env`, `.pem` files from projects
- **Configs**: `~/.config` (smart exclusions: `node_modules`, `.git`, `vendor`)
- **Packages**: Homebrew, npm, VS Code extensions, Mac App Store
- **Browser Data**: Bookmarks, extensions, settings (optional, disabled by default)
- **Git Repos**: All repos tracked for easy re-cloning
- **System**: macOS defaults, fonts, shell history

---

## Install

**Curl (no Homebrew required):**
```bash
curl -fsSL https://raw.githubusercontent.com/harshpatel5940/stash/main/install.sh | bash
```
Defaults to `/opt/homebrew/bin` on Apple Silicon (if present), otherwise `/usr/local/bin`.
Set `STASH_INSTALL_DIR` to override.

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
# Skip dependency installs:
# stash init --skip-deps

# Backup
stash backup

# List backups
stash list

# Show metadata + note
stash info 1

# Restore by ID or name (copy .stash.key first!)
stash restore 1
```

---

## Flags

**Backup:**
- `--skip-browsers` - Skip browser data for this run
- `--keep <n>` - Keep only last N backups (default: 5)
- `-m, --message` - Add note/message to backup
- `--dry-run` - Preview what will be backed up
- `--verbose` - Detailed output
- `--no-encrypt` - Skip encryption (not recommended)

**Restore:**
- `--dry-run` - Preview
- `--editor` - Pick/drop files and packages in editor (git-rebase style)
- `--no-tui` - Use Y/n prompts instead of interactive TUI
- `--no-decrypt` - Unencrypted backup

**Info:**
- `stash info <id|name>` - Show backup metadata and note
- `stash info <id|name> -m "..."` - Update note for a backup

**Init:**
- `stash init --skip-deps` - Skip auto-installing Homebrew and helper CLIs

**Config:**
- `stash config edit` - Interactive TUI editor for common settings
- `stash config edit --raw` - Open raw YAML in VISUAL/EDITOR/vim

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

browsers:
  enabled: true
```

---

## Reset Recovery Coverage

**Covered by stash:**
- Homebrew restore is resilient (per-package retries + progress).
- Finder defaults (hidden files, file extensions) and menu bar clock.
- Dock layout (position, autohide, pinned apps with `dockutil`).
- Desktop wallpaper restore.
- Application Firewall rules.

**Requires dependencies (auto-installed by `stash init` when missing):**
- Homebrew, `mas`, `dockutil`, Node.js/npm, and VS Code (for `code` CLI).
  - Note: the `code` CLI may still require running **"Shell Command: Install 'code'"** inside VS Code.

**Common reset gaps (manual today):**
- Keychain passwords/certificates.
- Login Items/LaunchAgents.
- TCC privacy permissions (Full Disk Access, Accessibility, etc.).
- Wi‑Fi/VPN/Proxy profiles.
- Printers and drivers.
- Apple ID/iCloud sign-in + service re‑enable.

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

## ⚠️ Critical Warning

**Need BOTH to restore:**
- 🔑 `~/.stash.key` → Store in password manager or secure location
- 📦 `.tar.gz.age` → Store in cloud or external drive

**Store separately. Lose either one = lose everything.** Key without backup is useless. Backup without key is inaccessible.

---

## Development

```bash
make build
make test
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

**License:** MIT ([LICENSE](LICENSE)) | **Security:** [SECURITY.md](SECURITY.md) | **[Issues](https://github.com/harshpatel5940/stash/issues)**

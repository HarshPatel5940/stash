# Stash CLI Improvements Plan

## Summary
Comprehensive improvements to the stash CLI tool including bug fixes, UX overhaul, and new features.

## Confirmed Choices
- **TUI Library**: `charmbracelet/huh` - simpler API, built-in multi-select forms
- **Cloud Provider**: S3-compatible only (works with AWS, Backblaze B2, MinIO, R2, etc.)
- **Scope**: All phases - complete overhaul

---

## Phase 1: Bug Fixes (Quick Wins)

### 1.1 Version Mismatch Bug
**Problem**: `--version` shows `1.1.1` instead of `1.2.1`
- `main.go` has `const Version = "1.2.1"` (unused)
- `cmd/root.go` has `var Version = "1.1.1"` (actual CLI version)

**Fix**: Update `cmd/root.go` to `1.2.1` and remove unused constant from `main.go`

**Files**:
- `/Users/harshnpatel/Documents/github/stash/cmd/root.go` - Update version
- `/Users/harshnpatel/Documents/github/stash/main.go` - Remove unused constant

---

## Phase 2: Fix Broken Features

### 2.1 Diff Command (Broken)
**Problem**: `loadBackupMetadata()` in `internal/diff/diff.go` can't handle encrypted `.age` files - it only looks for sidecar `.metadata.json` files which don't exist.

**Fix**: Create utility to extract metadata from encrypted backups:
1. Create `/internal/backuputil/backuputil.go` with `ExtractMetadata()` function
2. Decrypt `.age` file to temp directory
3. Extract tar.gz and read `metadata.json`
4. Add `--decrypt-key` flag to diff command

**Files**:
- Create: `/internal/backuputil/backuputil.go`
- Modify: `/internal/diff/diff.go`
- Modify: `/cmd/diff.go`

### 2.2 List Command (Incomplete)
**Problem**: `readMetadataFromBackup()` in `cmd/list.go` is a stub returning `nil`

**Fix**: Reuse `backuputil.ExtractMetadata()` from Phase 2.1
- Add metadata caching to avoid re-extracting
- Show file counts, package info for each backup

**Files**:
- Modify: `/cmd/list.go`

---

## Phase 3: Restore UX Overhaul

### 3.1 Replace Y/n Prompts with Multi-Select
**Current**: Sequential Y/n prompts for each restore option (tedious)
**New**: Single multi-select form to choose all options at once

### 3.2 Replace Editor-Based File Selection with TUI Multi-Select
**Current**: Opens vim/editor with "pick/drop" lines (git-rebase style)
**New**: Terminal-based multi-select with checkboxes

**Library Choice**: `charmbracelet/huh` (simpler than bubbletea, actively maintained)

**Implementation**:
1. Add `github.com/charmbracelet/huh` dependency
2. Create `/internal/tui/tui.go` with:
   - `RestoreOptionsForm()` - multi-select for restore options
   - `FilePickerForm()` - multi-select for individual files
3. Update `/cmd/restore.go` to use TUI forms
4. Keep `--editor` flag as fallback for editor-based selection

**Files**:
- Modify: `go.mod`
- Create: `/internal/tui/tui.go`
- Modify: `/cmd/restore.go`

---

## Phase 4: Git Tracking Enhancements

### 4.1 Detect Unpushed Commits
**Current**: Only detects uncommitted changes (`Dirty` field)
**New**: Also detect unpushed commits and ahead/behind status

**Implementation**:
1. Extend `GitRepo` struct with new fields:
   - `Ahead int` - commits ahead of remote
   - `Behind int` - commits behind remote
   - `UnpushedCount int` - unpushed commits
   - `HasUpstream bool` - has tracking branch
2. Add detection using:
   - `git rev-list --left-right --count @{upstream}...HEAD`
   - `git log @{upstream}..HEAD --oneline`
3. Show warnings during backup for repos with issues

**Files**:
- Modify: `/internal/gittracker/gittracker.go`
- Modify: `/cmd/backup.go`

### 4.2 Git Remind Command
**New command**: `stash remind` - shows repos needing attention

**Files**:
- Create: `/cmd/remind.go`

---

## Phase 5: Package Install Progress

### 5.1 Better Progress Indicators
**Current**: Basic stdout piping during installs
**New**: Progress bars with package counts

**Implementation**:
1. Create `/internal/packager/installer.go` with progress-wrapped installers
2. Parse brew bundle output to track progress
3. Show per-extension progress for VS Code

**Files**:
- Create: `/internal/packager/installer.go`
- Modify: `/cmd/restore.go`

---

## Phase 6: Cloud Sync (New Feature)

### 6.1 S3-Compatible Cloud Storage
**New command**: `stash sync` with subcommands:
- `stash sync up [backup]` - upload to cloud
- `stash sync down [backup]` - download from cloud
- `stash sync list` - list cloud backups

**Implementation**:
1. Add AWS SDK dependency
2. Create `/internal/cloud/` package with Provider interface
3. Add cloud config to `~/.stash.yaml`
4. Optional `--sync` flag on backup command

**Files**:
- Modify: `go.mod`
- Create: `/internal/cloud/cloud.go`
- Create: `/internal/cloud/s3.go`
- Create: `/cmd/sync.go`
- Modify: `/internal/config/config.go`

---

## Phase 7: Config System Overhaul

### 7.1 Expand Config Structure
Make all hardcoded values configurable in `~/.stash.yaml`:

**Files to modify**: `internal/config/config.go`

```yaml
# Proposed new config structure
backup:
  keep_count: 5              # Currently hardcoded flag default
  auto_cleanup: true

dotfiles:
  additional: []             # Extra dotfiles to include
  ignored_dirs:              # Currently hardcoded in dotfiles.go
    - .cache
    - .local
    - .npm
    - .vscode
    - .cargo
    - .rustup

secrets:
  directories:               # Currently hardcoded: .ssh, .gnupg, .aws
    - .ssh
    - .gnupg
    - .aws
    - .kube                  # Allow adding more

shell_history:
  files:                     # Currently hardcoded in backup.go
    - .zsh_history
    - .bash_history
    - .zhistory

git:
  search_dirs:               # Currently hardcoded in backup.go, remind.go
    - ~/Documents
    - ~/Projects
    - ~/Code
    - ~/Dev
    - ~/workspace
    - ~/github
  max_depth: 5               # Currently hardcoded in gittracker.go
  skip_dirs:                 # Currently hardcoded in gittracker.go
    - node_modules
    - vendor
    - venv
    - build

macos_defaults:
  domains:                   # Currently hardcoded in defaults.go
    - com.apple.dock
    - com.apple.finder
    - NSGlobalDomain

browsers:
  enabled: true
  include: []                # Allow custom browser configs

restore:
  use_tui: true
  file_picker_threshold: 100 # Currently hardcoded in restore.go

diff:
  display_limit: 10          # Currently hardcoded in diff.go
```

### 7.2 Implementation Steps

1. **Extend Config struct** (`internal/config/config.go`):
   - Add nested structs for each category
   - Provide sensible defaults in `DefaultConfig()`

2. **Update consumers** to read from config instead of hardcoded values:
   - `internal/finder/dotfiles.go` - ignored dirs, common dotfiles
   - `internal/gittracker/gittracker.go` - max depth, skip dirs
   - `internal/defaults/defaults.go` - domains list
   - `cmd/backup.go` - shell history files, git search dirs
   - `cmd/restore.go` - file picker threshold, TUI preference
   - `cmd/diff.go` - display limit
   - `cmd/remind.go` - git search dirs

3. **Add `stash config` command** for easy config management:
   - `stash config init` - create default config
   - `stash config show` - display current config
   - `stash config edit` - open in editor

**Files**:
- Modify: `internal/config/config.go`
- Modify: `internal/finder/dotfiles.go`
- Modify: `internal/gittracker/gittracker.go`
- Modify: `internal/defaults/defaults.go`
- Modify: `cmd/backup.go`
- Modify: `cmd/restore.go`
- Modify: `cmd/diff.go`
- Modify: `cmd/remind.go`
- Create: `cmd/config.go`

---

## Phase 8: GitHub Actions Optimization

### 8.1 Fix test.yml

**Issues**:
- Go version mismatch (1.21 vs go.mod 1.25.5)
- Missing Go module caching
- Missing permissions block
- Redundant `go mod download`

**Fix**:
```yaml
permissions:
  contents: read

jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'  # Match go.mod
          cache: true         # Enable module caching
      # Remove 'go mod download' - redundant with cache
      - name: Run tests
        run: go test -v ./...
```

### 8.2 Fix release.yml

**Issues**:
- Same Go version mismatch
- Missing caching in test job
- Duplicated test logic (should use reusable workflow)

**Fix**: Same caching improvements + consider reusable workflow

### 8.3 Optional Enhancements

- Add golangci-lint job for code quality
- Add multi-platform test matrix (Linux, Windows)
- Add test coverage reporting

**Files**:
- Modify: `.github/workflows/test.yml`
- Modify: `.github/workflows/release.yml`

---

## Verification Plan

1. **Version fix**: Run `stash --version` and verify output is `1.2.1`
2. **Diff command**: Run `stash diff backup1.tar.gz.age backup2.tar.gz.age`
3. **List command**: Run `stash list` and verify metadata shown
4. **Restore UX**: Run `stash restore backup.tar.gz.age` and verify multi-select UI
5. **Git tracking**: Run `stash backup` in a repo with unpushed commits, verify warnings
6. **Package progress**: Run restore with Homebrew packages, verify progress bars
7. **Cloud sync**: Configure S3 bucket, run `stash sync up`, verify upload
8. **Config system**: Create custom `~/.stash.yaml` with non-default values, verify they're used
9. **GitHub Actions**: Push to trigger workflow, verify caching works and tests pass

---

## Implementation Order

### Completed (Phase 1-6)
| Priority | Phase | Status |
|----------|-------|--------|
| 1 | Version bug fix | ✅ Done |
| 2 | Diff command fix | ✅ Done |
| 3 | List command fix | ✅ Done |
| 4 | Restore UX overhaul | ✅ Done |
| 5 | Git tracking | ✅ Done |
| 6 | Git remind command | ✅ Done |
| 7 | Package progress | ✅ Done |
| 8 | Cloud sync | ✅ Done |

### New Work (Phase 7-8)
| Priority | Phase | Scope |
|----------|-------|-------|
| 9 | Config system overhaul (FULL) | High - all 15+ configurable values |
| 10 | GitHub Actions optimization | Low - 2 workflow files |

## Confirmed Choices (Phase 7-8)
- **Config Scope**: Full expansion - all configurable values
- **GitHub Actions**: Fix both test.yml and release.yml

# Contributing to Stash

Thanks for your interest in contributing to Stash! This document provides guidelines and instructions for contributing.

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- macOS (for full testing)
- Git

### Setting Up Development Environment

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR_USERNAME/stash.git
   cd stash
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Build**
   ```bash
   make build
   # or
   go build -o stash
   ```

4. **Run Tests**
   ```bash
   make test
   # or
   go test ./...
   ```

## 📝 Development Workflow

1. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bugfix-name
   ```

2. **Make Changes**
   - Write clean, readable code
   - Follow Go conventions and idioms
   - Add tests for new features
   - Update documentation as needed

3. **Test Your Changes**
   ```bash
   # Run tests
   make test
   
   # Run formatting
   make fmt
   
   # Run linting
   make vet
   ```

4. **Commit**
   ```bash
   git add .
   git commit -m "feat: add amazing feature"
   ```
   
   Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `style:` - Code style changes (formatting, etc.)
   - `refactor:` - Code refactoring
   - `test:` - Adding or updating tests
   - `chore:` - Maintenance tasks

5. **Push and Create PR**
   ```bash
   git push origin feature/your-feature-name
   ```
   Then open a Pull Request on GitHub.

## 🏗️ Project Structure

```
stash/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command setup
│   ├── backup.go          # Backup command
│   ├── restore.go         # Restore command
│   ├── list.go            # List/preview command
│   └── init.go            # Init command
├── internal/
│   ├── finder/            # File discovery logic
│   │   ├── dotfiles.go    # Find dotfiles
│   │   └── envfiles.go    # Find .env and .pem files
│   ├── packager/          # Package manager integration
│   │   └── packager.go    # Brew, MAS, VSCode, npm
│   ├── archiver/          # Archive operations
│   │   └── archiver.go    # tar.gz creation/extraction
│   ├── crypto/            # Encryption/decryption
│   │   └── crypto.go      # age implementation
│   ├── metadata/          # Backup metadata
│   │   └── metadata.go    # Metadata structure
│   └── config/            # Configuration
│       └── config.go      # Config loading/saving
├── go.mod                 # Go module file
├── go.sum                 # Go dependencies
├── main.go                # Entry point
├── Makefile               # Build automation
└── README.md              # Documentation
```

## 🧪 Testing Guidelines

### Writing Tests

- Place test files next to the code they test (`*_test.go`)
- Use table-driven tests for multiple test cases
- Mock external dependencies
- Aim for meaningful test coverage

Example:
```go
func TestFindDotfiles(t *testing.T) {
    tests := []struct {
        name     string
        input    []string
        expected []string
        wantErr  bool
    }{
        {
            name:     "finds common dotfiles",
            input:    []string{".zshrc", ".gitconfig"},
            expected: []string{".zshrc", ".gitconfig"},
            wantErr:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Specific package
go test ./internal/finder/
```

## 📋 Code Style

### Go Conventions

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting (automatically done with `make fmt`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused

### Example

```go
// FindDotfiles discovers dotfiles in the home directory.
// It returns a slice of absolute paths to found dotfiles.
func FindDotfiles(homeDir string, additional []string) ([]string, error) {
    var dotfiles []string
    
    // Implementation here
    
    return dotfiles, nil
}
```

## 🐛 Bug Reports

When filing a bug report, please include:

1. **Description**: Clear description of the issue
2. **Steps to Reproduce**: Exact steps to reproduce the bug
3. **Expected Behavior**: What you expected to happen
4. **Actual Behavior**: What actually happened
5. **Environment**:
   - OS version (macOS version)
   - Go version
   - Stash version
6. **Logs**: Relevant error messages or logs

## 💡 Feature Requests

When suggesting a feature:

1. **Use Case**: Describe the problem you're trying to solve
2. **Proposed Solution**: Your suggested approach
3. **Alternatives**: Other solutions you've considered
4. **Additional Context**: Any other relevant information

## 🔍 Pull Request Guidelines

### Before Submitting

- [ ] Code builds without errors (`make build`)
- [ ] All tests pass (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] No linting errors (`make vet`)
- [ ] Documentation updated (if needed)
- [ ] CHANGELOG updated (for significant changes)

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
How has this been tested?

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

## 🎯 Areas to Contribute

### Easy Wins (Good First Issues)

- Improve error messages
- Add more dotfile patterns
- Enhance documentation
- Add more package managers support
- Improve CLI help text

### Medium Complexity

- Add configuration validation
- Improve backup verification
- Add progress indicators
- Enhance restore preview
- Add backup rotation/cleanup

### Advanced Features

- Cloud sync integration
- Incremental backups
- Selective restore
- Backup diff viewer
- Cross-platform support (Linux)

## 📜 License

By contributing, you agree that your contributions will be licensed under the MIT License.

## 🤝 Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inclusive environment for all contributors.

### Standards

- Be respectful and inclusive
- Accept constructive criticism gracefully
- Focus on what's best for the project
- Show empathy towards other contributors

### Unacceptable Behavior

- Harassment or discriminatory comments
- Trolling or insulting comments
- Publishing others' private information
- Other unprofessional conduct

## 📞 Getting Help

- **Questions**: Open a [GitHub Discussion](https://github.com/harshpatel5940/stash/discussions)
- **Bugs**: File an [Issue](https://github.com/harshpatel5940/stash/issues)
- **Chat**: Join discussions in issues and PRs

## 🙏 Recognition

All contributors will be recognized in the project's README and release notes.

---

Thank you for contributing to Stash! 🎉
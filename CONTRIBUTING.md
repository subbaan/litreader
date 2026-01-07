# Contributing to litreader

Thank you for your interest in contributing to litreader! This is primarily a personal project, but contributions are welcome.

## How to Contribute

### Reporting Bugs

If you find a bug, please open an issue on GitHub with:
- A clear description of the problem
- Steps to reproduce the issue
- Your operating system and Go version
- Any relevant error messages or logs

### Suggesting Features

Feature suggestions are welcome! Please open an issue describing:
- What you'd like to see added
- Why it would be useful
- How you envision it working

### Code Contributions

If you'd like to contribute code:

1. **Fork the repository** on GitHub
2. **Clone your fork** to your local machine
3. **Create a new branch** for your changes
4. **Make your changes** following the guidelines below
5. **Test your changes** thoroughly
6. **Submit a pull request** with a clear description of what you've changed and why

#### Code Guidelines

- Follow standard Go conventions and formatting (`gofmt`, `go vet`)
- Add comments for complex logic
- Write tests for new functionality when possible
- Keep commits focused on a single change
- Update the README if you add new features or change behavior

#### Testing

Before submitting a pull request, please:
```bash
# Run tests
go test ./...

# Build the application
go build -o litreader ./cmd/litreader

# Test the application works as expected
./litreader
```

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/litreader.git
cd litreader

# Build and run
go build -o litreader ./cmd/litreader
./litreader
```

## Project Structure

- `cmd/litreader/` - Main application entry point
- `internal/config/` - Configuration management
- `internal/cache/` - File and search caching
- `internal/ui/` - Terminal UI components (Bubbletea)
- `internal/library/` - File scanning and library management
- `internal/external/` - External tool integrations (pandoc, ripgrep)
- `internal/models/` - Data models
- `internal/state/` - Application state management

## Questions?

If you have questions about contributing, feel free to open an issue to discuss!

## License

By contributing to litreader, you agree that your contributions will be licensed under the MIT License.

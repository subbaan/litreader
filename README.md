# litreader

A terminal-based text file reader and library manager for text stories, rewritten in Go using the Bubbletea framework.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue.svg)](https://golang.org/)

## Project Status

**Phase 1: Foundation - ✅ COMPLETE**

- ✅ Project structure created
- ✅ Go module initialized
- ✅ XDG-compliant path helpers
- ✅ Config management with Python compatibility
- ✅ Cache system (JSON-based)
- ✅ Model types (Favorite, Bookmark, FileInfo)
- ✅ Unit tests passing
- ✅ Verified compatibility with existing Python config

**Phase 2: Basic TUI - ✅ COMPLETE**

- ✅ Bubbletea application skeleton
- ✅ State management and navigation stack
- ✅ Dashboard view with library stats
- ✅ Lipgloss styling (configurable colors from config)
- ✅ Responsive layout and window resize handling
- ✅ Menu navigation (arrow keys, vim keys)

**Phase 3: Core Views - ✅ COMPLETE**

- ✅ Library file scanner
- ✅ File viewer with pandoc rendering
- ✅ Search view with ripgrep integration
- ✅ Favorites view with sorting
- ✅ Bookmarks view
- ✅ Author favorites view
- ✅ Full navigation between all views
- ✅ Keyboard shortcut navigation (s, f, b, a, l, q)

**Phase 4: Advanced Features - ✅ COMPLETE**

- ✅ Add to favorites from viewer ('f' key)
- ✅ Add bookmark from viewer ('b' key)
- ✅ Configuration editor ('c' key from dashboard)
- ✅ All views styled to match original Python version
- ✅ Author files browser (press Enter on an author to view their files)
- ✅ File scanning on startup (enables search functionality)

## Installation

### Prerequisites

#### Required
- **Go 1.24 or later** - [Download Go](https://golang.org/dl/)

#### Optional (gracefully handled if missing)
- **pandoc** - For rendering markdown files (falls back to plain text if unavailable)
- **ripgrep (rg)** - For fast file searching (search functionality disabled if unavailable)

### From Source

```bash
# Clone the repository
git clone https://github.com/subbass/litreader.git
cd litreader

# Build the application
go build -o litreader ./cmd/litreader

# Install to your system (optional)
sudo mv litreader /usr/local/bin/
# Or add to your PATH
```

### First Run

On first run, litreader will automatically create:
- Config directory: `~/.config/litreader/`
- Config file: `~/.config/litreader/litreader.conf`
- Cache directory: `~/.cache/litreader/`

You'll need to edit the config file to set your library directory. See the [Configuration](#configuration) section below.

## Building

```bash
go build -o litreader ./cmd/litreader
```

## Testing

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/config/... -v
```

## Project Structure

```
litreader-go/
├── cmd/
│   ├── litreader/          # Main application
│   └── test-config/        # Config testing tool
├── internal/
│   ├── config/             # Configuration management
│   ├── cache/              # File and search caching
│   ├── models/             # Data models
│   ├── external/           # External command wrappers (TODO)
│   ├── library/            # File scanning and stats (TODO)
│   ├── state/              # Application state (TODO)
│   └── ui/                 # Bubbletea UI components (TODO)
└── go.mod
```

## Configuration

The configuration file is located at `~/.config/litreader/litreader.conf` and uses a simple key-value format.

### Configuration Options

```ini
# Library paths
search_dir = /path/to/your/story/library
export_dir = ~/Documents/litreader_faves

# UI Colors
ui_color = blue
selector_color = yellow
selector_reverse = true
selector_bold = true
selector_reverse_color = black
content_color = white
content_bold = false

# Cache settings
cache_expiry_days = 7

# Editor (for config editing from within app)
editor = nano
```

### Usage
- Press 'c' from the dashboard to edit the configuration file
- Changes take effect after restarting the application

## Version

Current: 2.4.5 (Go rewrite)

### Changelog
- 2.4.5: Repository prepared for public release (MIT license, documentation updates)
- 2.2.0: Fixed file crash issues (encoding, pandoc timeout, rendering safeguards)
- 2.1.0: Dynamic app name support + library display
- 2.0.0: Initial Go rewrite from Python version 1.78.0

# Development Guide

Technical documentation for litreader development.

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
litreader/
├── cmd/
│   ├── litreader/          # Main application
│   └── test-config/        # Config testing tool
├── internal/
│   ├── config/             # Configuration management
│   ├── cache/              # File and search caching
│   ├── models/             # Data models
│   ├── external/           # External command wrappers
│   ├── library/            # File scanning and stats
│   ├── state/              # Application state
│   └── ui/                 # Bubbletea UI components
└── go.mod
```

## Development History

### Phase 1: Foundation
- Project structure created
- Go module initialized
- XDG-compliant path helpers
- Config management with Python compatibility
- Cache system (JSON-based)
- Model types (Favorite, Bookmark, FileInfo)
- Unit tests passing
- Verified compatibility with existing Python config

### Phase 2: Basic TUI
- Bubbletea application skeleton
- State management and navigation stack
- Dashboard view with library stats
- Lipgloss styling (configurable colors from config)
- Responsive layout and window resize handling
- Menu navigation (arrow keys, vim keys)

### Phase 3: Core Views
- Library file scanner
- File viewer with pandoc rendering
- Search view with ripgrep integration
- Favorites view with sorting
- Bookmarks view
- Author favorites view
- Full navigation between all views
- Keyboard shortcut navigation (s, f, b, a, l, q)

### Phase 4: Advanced Features
- Add to favorites from viewer ('f' key)
- Add bookmark from viewer ('b' key)
- Configuration editor ('c' key from dashboard)
- All views styled to match original Python version
- Author files browser (press Enter on an author to view their files)
- File scanning on startup (enables search functionality)

### Changelog

#### v2.6.3
- Redesigned viewer help bar layout: items are now semantically grouped with pipe separators instead of greedy flow wrapping
- Help bar uses 3 lines by default (wide/medium terminals) and 4 lines for narrow terminals
- Improved readability with logically grouped keybindings (navigation, actions, utilities)

#### v2.6.2
- Fix juddering scroll in word-wrap mode

#### v2.6.1
- Add word-wrap toggle and reflowing help bar

## Configuration File Format

The configuration file is located at `~/.config/litreader/litreader.conf` and uses a simple key-value format:

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

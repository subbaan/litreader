# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

litreader is a terminal-based (Bubbletea/Lipgloss TUI) ebook/text-file reader and library manager, written in Go. It scans a directory of `.txt` files organized by author, tracks reading progress, favorites, bookmarks, and ratings, and provides a full-text search (via ripgrep) and an in-app editor and config editor.

## Commands

```bash
# Build
go build -o litreader ./cmd/litreader

# Run all tests
go test ./...

# Run a single package's tests verbosely
go test ./internal/config/... -v

# Run a single test
go test ./internal/config/... -run TestName -v
```

There is no Makefile/linter config; use standard `gofmt`/`go vet`.

## Versioning and releases

- The app version lives in one place: `internal/version/version.go` (`AppVersion` const). Bump it whenever you make a user-facing change.
- Commit message subjects follow the pattern `<Description> (vX.Y.Z)`, matching the version bump in that commit (see `git log` for examples).
- Per user's global instructions: after editing source, rebuild the `litreader` executable and bump the version; the built executable should be part of a release.
- `-h`/`--help` and `-v`/`--version` are implemented directly in `cmd/litreader/main.go` — keep them in sync with new flags/features.

## Architecture

**Entry point**: `cmd/litreader/main.go` parses flags, loads config (`internal/config`) and cache (`internal/cache`), builds a `state.State`, and hands it to `ui.NewApp` which is run as a `tea.Program` (Bubbletea). A bare file argument (`litreader somefile.txt`) triggers "direct file mode", bypassing the library/dashboard entirely.

**UI is a single Bubbletea model (`internal/ui/app.go`) acting as a router.** `App` holds one sub-model per screen (`DashboardModel`, `SearchModel`, `ViewerModel`, `FavoritesModel`, `BookmarksModel`, `AuthorsModel`, `AuthorFilesModel`, `ConfigEditorModel`, `ExploreAuthorsModel`, `EditorModel`) plus a `currentView ViewType` and a `viewStack []ViewType` for back-navigation. `App.Update` does global key handling and message routing, then delegates to `updateCurrentView`, which forwards the `tea.Msg` to the active sub-model and, for non-editor views, also runs `handleNavigation` to catch app-level shortcuts and back/quit keys (`q`, `left`). Adding a new screen means: add a `ViewType` constant, a model field, wiring in `Update`/`View`/`navigateTo`, and window-resize propagation (all four are updated in lockstep in `app.go` — easy to forget one).

**Shared state**: `internal/state.State` is the one object passed to every screen's constructor — it wraps `*config.Config`, `*cache.Cache`, the scanned file list/metadata, and search results. Screens mutate it directly rather than passing data back through messages, except where a `tea.Cmd`/`tea.Msg` round-trip is needed for async work (e.g. `scanLibraryFiles`, `openEditorMsg`/`editorDoneMsg`).

**Multi-instance design**: the app name is derived from `os.Args[0]` (`internal/config/xdg.go` → `getAppName`), and config/cache paths are namespaced by that name (`~/.config/<name>/<name>.conf`, `~/.cache/<name>/`). Copying/renaming the `litreader` binary (e.g. to `ficreader`) produces a fully independent instance with its own config, cache, and library — there is no other mechanism for multiple libraries. Keep this name-derivation intact when touching config/cache path logic.

**Config** (`internal/config`): `Load()`/`Save()` read/write a simple `key = value` INI-style file (see `example.conf`); `loader.go` parses it and `saver.go` writes it back preserving comments/order where possible. The format is kept compatible with an earlier Python version of this tool, so field names and value formats (e.g. bool as `true`/`false`) shouldn't be changed casually.

**Cache** (`internal/cache`): JSON-based, split into the file-list scan cache (`filelist.go`, keyed by search dir + mtime, expires after `cache_expiry_days`) and the favorites/bookmarks/ratings cache (`cache.go`). `search.go` shells out to ripgrep for full-text search and caches results.

**External tools** (`internal/external`): thin wrappers that shell out to `pandoc` (markdown rendering, with plain-text fallback if pandoc is missing) and `ripgrep` (search, disabled if `rg` isn't installed). Treat both as optional dependencies — code paths must degrade gracefully without them.

**Library** (`internal/library`): filesystem scanning (`scanner.go`) and per-author/library statistics (`stats.go`, `rating.go`) over the author-folder/`.txt`-file layout.

**Viewer/editor**: `internal/ui/viewer.go` and `internal/ui/editor.go` are the two largest, most stateful screens — the viewer handles pandoc-rendered display, percentage-based jumping, bookmarks/search-in-file, and mouse-wheel scrolling (including XInput2 burst handling); the editor is a built-in text editor with word-wrap and visual-line cursor movement. Changes to scrolling/wrapping/cursor math should be tested interactively (`go run ./cmd/litreader` against a real `.txt` file), since these are hard to unit-test.

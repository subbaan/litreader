package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/cache"
	"github.com/subbass/litreader/internal/library"
	"github.com/subbass/litreader/internal/state"
)

// ViewType represents different views in the application
type ViewType int

const (
	ViewDashboard ViewType = iota
	ViewSearch
	ViewFile
	ViewFavorites
	ViewBookmarks
	ViewAuthors
	ViewAuthorFiles
	ViewConfigEditor
	ViewExploreAuthors
)

// App is the main bubbletea model
type App struct {
	// Window dimensions
	width  int
	height int

	// Current view and navigation
	currentView ViewType
	viewStack   []ViewType

	// Global state
	state  *state.State
	styles *Styles

	// View-specific models
	dashboard      *DashboardModel
	search         *SearchModel
	viewer         *ViewerModel
	favorites      *FavoritesModel
	bookmarks      *BookmarksModel
	authors        *AuthorsModel
	authorFiles    *AuthorFilesModel
	configEditor   *ConfigEditorModel
	exploreAuthors *ExploreAuthorsModel

	// Modal/popup state
	showPopup bool
	popupMsg  string

	// Loading state
	loading    bool
	loadingMsg string

	// Error state
	err error
}

// NewApp creates a new App model
func NewApp(s *state.State) *App {
	styles := NewStyles(s.Config)

	app := &App{
		currentView: ViewDashboard,
		viewStack:   []ViewType{},
		state:       s,
		styles:      styles,
	}

	// Initialize dashboard
	app.dashboard = NewDashboardModel(s, styles)

	return app
}

// Init initializes the app (bubbletea interface)
func (a *App) Init() tea.Cmd {
	// Scan library files in background
	return a.scanLibraryFiles
}

// Update handles messages (bubbletea interface)
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key handling
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Update all view models with new size
		if a.dashboard != nil {
			a.dashboard.width = msg.Width
			a.dashboard.height = msg.Height
		}
		if a.search != nil {
			a.search.width = msg.Width
			a.search.height = msg.Height
		}
		if a.viewer != nil {
			a.viewer.width = msg.Width
			a.viewer.height = msg.Height
		}
		if a.favorites != nil {
			a.favorites.width = msg.Width
			a.favorites.height = msg.Height
		}
		if a.bookmarks != nil {
			a.bookmarks.width = msg.Width
			a.bookmarks.height = msg.Height
		}
		if a.authors != nil {
			a.authors.width = msg.Width
			a.authors.height = msg.Height
		}
		if a.authorFiles != nil {
			a.authorFiles.width = msg.Width
			a.authorFiles.height = msg.Height
		}
		if a.configEditor != nil {
			a.configEditor.width = msg.Width
			a.configEditor.height = msg.Height
		}
		if a.exploreAuthors != nil {
			a.exploreAuthors.width = msg.Width
			a.exploreAuthors.height = msg.Height
		}
		return a, nil

	case errMsg:
		a.err = msg.err
		return a, nil

	case openFileMsg:
		return a, a.openFile(msg.filename, msg.position, msg.searchText)

	case openAuthorFilesMsg:
		return a, a.openAuthorFiles(msg.authorFolder)

	case filesScannedMsg:
		a.state.AllFiles = msg.files
		// Refresh dashboard with scanned files
		if a.dashboard != nil {
			a.dashboard.calculateStatistics()
			a.dashboard.calculateInProgress()
		}
		return a, nil

	case filesScannedWithMetadataMsg:
		// Populate AllFiles (just the paths)
		a.state.AllFiles = make([]string, len(msg.metadata))
		for i, meta := range msg.metadata {
			a.state.AllFiles[i] = meta.Path
			a.state.FileMetadata[meta.Path] = &msg.metadata[i]
		}
		// Refresh dashboard with scanned files
		if a.dashboard != nil {
			a.dashboard.calculateStatistics()
			a.dashboard.calculateInProgress()
		}
		return a, nil

	case stylesUpdatedMsg:
		// Rebuild styles from updated config
		a.styles = NewStyles(a.state.Config)

		// Update all view models with new styles
		if a.dashboard != nil {
			a.dashboard.styles = a.styles
		}
		if a.search != nil {
			a.search.styles = a.styles
		}
		if a.viewer != nil {
			a.viewer.styles = a.styles
		}
		if a.favorites != nil {
			a.favorites.styles = a.styles
		}
		if a.bookmarks != nil {
			a.bookmarks.styles = a.styles
		}
		if a.authors != nil {
			a.authors.styles = a.styles
		}
		if a.authorFiles != nil {
			a.authorFiles.styles = a.styles
		}
		if a.configEditor != nil {
			a.configEditor.styles = a.styles
		}
		if a.exploreAuthors != nil {
			a.exploreAuthors.styles = a.styles
		}
		return a, nil
	}

	// Delegate to current view
	return a.updateCurrentView(msg)
}

// updateCurrentView delegates updates to the current view
func (a *App) updateCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// First, let the current view handle the message
	switch a.currentView {
	case ViewDashboard:
		if a.dashboard != nil {
			var m tea.Model
			m, cmd = a.dashboard.Update(msg)
			if updated, ok := m.(*DashboardModel); ok {
				a.dashboard = updated
			}
		}
	case ViewSearch:
		if a.search != nil {
			var m tea.Model
			m, cmd = a.search.Update(msg)
			if updated, ok := m.(*SearchModel); ok {
				a.search = updated
			}
		}
	case ViewFile:
		if a.viewer != nil {
			var m tea.Model
			m, cmd = a.viewer.Update(msg)
			if updated, ok := m.(*ViewerModel); ok {
				a.viewer = updated
			}
		}
	case ViewFavorites:
		if a.favorites != nil {
			var m tea.Model
			m, cmd = a.favorites.Update(msg)
			if updated, ok := m.(*FavoritesModel); ok {
				a.favorites = updated
			}
		}
	case ViewBookmarks:
		if a.bookmarks != nil {
			var m tea.Model
			m, cmd = a.bookmarks.Update(msg)
			if updated, ok := m.(*BookmarksModel); ok {
				a.bookmarks = updated
			}
		}
	case ViewAuthors:
		if a.authors != nil {
			var m tea.Model
			m, cmd = a.authors.Update(msg)
			if updated, ok := m.(*AuthorsModel); ok {
				a.authors = updated
			}
		}
	case ViewConfigEditor:
		if a.configEditor != nil {
			var m tea.Model
			m, cmd = a.configEditor.Update(msg)
			if updated, ok := m.(*ConfigEditorModel); ok {
				a.configEditor = updated
			}
		}
	case ViewAuthorFiles:
		if a.authorFiles != nil {
			var m tea.Model
			m, cmd = a.authorFiles.Update(msg)
			if updated, ok := m.(*AuthorFilesModel); ok {
				a.authorFiles = updated
			}
		}
	case ViewExploreAuthors:
		if a.exploreAuthors != nil {
			var m tea.Model
			m, cmd = a.exploreAuthors.Update(msg)
			if updated, ok := m.(*ExploreAuthorsModel); ok {
				a.exploreAuthors = updated
			}
		}
	}

	// After view handles it, check if we should navigate
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if navCmd := a.handleNavigation(keyMsg); navCmd != nil {
			return a, navCmd
		}
	}

	return a, cmd
}

// View renders the current view (bubbletea interface)
func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	if a.err != nil {
		return a.styles.RenderTitle("Error: "+a.err.Error(), a.width)
	}

	if a.loading {
		return a.styles.RenderTitle(a.loadingMsg, a.width)
	}

	// Render current view
	switch a.currentView {
	case ViewDashboard:
		if a.dashboard != nil {
			return a.dashboard.View()
		}
	case ViewSearch:
		if a.search != nil {
			return a.search.View()
		}
	case ViewFile:
		if a.viewer != nil {
			return a.viewer.View()
		}
	case ViewFavorites:
		if a.favorites != nil {
			return a.favorites.View()
		}
	case ViewBookmarks:
		if a.bookmarks != nil {
			return a.bookmarks.View()
		}
	case ViewAuthors:
		if a.authors != nil {
			return a.authors.View()
		}
	case ViewConfigEditor:
		if a.configEditor != nil {
			return a.configEditor.View()
		}
	case ViewAuthorFiles:
		if a.authorFiles != nil {
			return a.authorFiles.View()
		}
	case ViewExploreAuthors:
		if a.exploreAuthors != nil {
			return a.exploreAuthors.View()
		}
	}

	return "Unknown view"
}

// Navigation methods

// handleNavigation processes navigation commands from any view
func (a *App) handleNavigation(key tea.KeyMsg) tea.Cmd {
	keyStr := key.String()

	// Dashboard shortcuts (work from dashboard only)
	if a.currentView == ViewDashboard {
		switch keyStr {
		case "s":
			return a.navigateTo(ViewSearch)
		case "f":
			return a.navigateTo(ViewFavorites)
		case "b":
			return a.navigateTo(ViewBookmarks)
		case "a":
			return a.navigateTo(ViewAuthors)
		case "e":
			return a.navigateTo(ViewExploreAuthors)
		case "c":
			return a.navigateTo(ViewConfigEditor)
		case "l":
			// Open last read file
			if a.state.Config.LastFile != "" {
				return a.openFile(a.state.Config.LastFile, a.state.Config.Position, a.state.Config.SearchText)
			}
		case "enter", "right":
			// Open selected in-progress story
			if len(a.dashboard.inProgress) > 0 && a.dashboard.cursor < len(a.dashboard.inProgress) {
				item := a.dashboard.inProgress[a.dashboard.cursor]
				return a.openFile(item.Favorite.Filename, item.Favorite.Position, item.Favorite.SearchText)
			}
		}
	}

	// Handle 'left' arrow from non-dashboard views (go back)
	if keyStr == "left" && a.currentView != ViewDashboard {
		// Check if viewer is in a special mode that should handle left arrow
		if a.currentView == ViewFile && a.viewer != nil {
			if a.viewer.ViewingBookmarks || a.viewer.EditingSearch || a.viewer.EditingBookmark {
				// Don't navigate - viewer is handling the key
				return nil
			}
		}
		// Don't navigate back from config editor - left arrow is used for adjusting parameters
		if a.currentView == ViewConfigEditor {
			return nil
		}
		// Reload dashboard data after returning
		if a.currentView == ViewFile {
			a.dashboard.calculateInProgress()
		}
		return a.navigateBack()
	}

	// Support 'q' from non-viewer views for backwards compatibility
	if keyStr == "q" && a.currentView != ViewDashboard && a.currentView != ViewFile {
		return a.navigateBack()
	}

	// In viewer, allow 'q' only if not in special modes (for quick quit comfort)
	if keyStr == "q" && a.currentView == ViewFile && a.viewer != nil {
		if !a.viewer.ViewingBookmarks && !a.viewer.EditingSearch && !a.viewer.EditingBookmark {
			// Save and go back
			a.viewer.savePosition()
			if a.dashboard != nil {
				a.dashboard.calculateInProgress()
			}
			return a.navigateBack()
		}
	}

	return nil
}

func (a *App) navigateTo(view ViewType) tea.Cmd {
	// Push current view to stack
	a.viewStack = append(a.viewStack, a.currentView)
	a.currentView = view

	// Initialize new view if needed
	switch view {
	case ViewSearch:
		a.search = NewSearchModel(a.state, a.styles, "")
		a.search.width = a.width
		a.search.height = a.height
	case ViewFavorites:
		a.favorites = NewFavoritesModel(a.state, a.styles)
		a.favorites.width = a.width
		a.favorites.height = a.height
	case ViewBookmarks:
		a.bookmarks = NewBookmarksModel(a.state, a.styles)
		a.bookmarks.width = a.width
		a.bookmarks.height = a.height
	case ViewAuthors:
		a.authors = NewAuthorsModel(a.state, a.styles)
		a.authors.width = a.width
		a.authors.height = a.height
	case ViewConfigEditor:
		a.configEditor = NewConfigEditorModel(a.state, a.styles)
		a.configEditor.width = a.width
		a.configEditor.height = a.height
	case ViewExploreAuthors:
		a.exploreAuthors = NewExploreAuthorsModel(a.state, a.styles)
		a.exploreAuthors.width = a.width
		a.exploreAuthors.height = a.height
	}

	return nil
}

func (a *App) openFile(filename string, position int, searchText string) tea.Cmd {
	// Push current view to stack
	a.viewStack = append(a.viewStack, a.currentView)
	a.currentView = ViewFile

	// Create viewer
	a.viewer = NewViewerModel(a.state, a.styles, filename, position, searchText)
	a.viewer.width = a.width
	a.viewer.height = a.height

	return nil
}

func (a *App) openAuthorFiles(authorFolder string) tea.Cmd {
	// Push current view to stack
	a.viewStack = append(a.viewStack, a.currentView)
	a.currentView = ViewAuthorFiles

	// Create author files view
	a.authorFiles = NewAuthorFilesModel(a.state, a.styles, authorFolder)
	a.authorFiles.width = a.width
	a.authorFiles.height = a.height

	return nil
}

func (a *App) navigateBack() tea.Cmd {
	if len(a.viewStack) > 0 {
		a.currentView = a.viewStack[len(a.viewStack)-1]
		a.viewStack = a.viewStack[:len(a.viewStack)-1]
	}
	if a.currentView == ViewAuthorFiles && a.authorFiles != nil {
		a.authorFiles.refreshFavorites()
	}
	return nil
}

// Message types

type errMsg struct {
	err error
}

type openFileMsg struct {
	filename   string
	position   int
	searchText string
}

type openAuthorFilesMsg struct {
	authorFolder string
}

type filesScannedMsg struct {
	files []string
}

type filesScannedWithMetadataMsg struct {
	metadata []cache.FileMetadata
}

type stylesUpdatedMsg struct {
	// Empty struct - just a signal to rebuild styles
}

// Commands

func (a *App) scanLibraryFiles() tea.Msg {
	// Try to load from cache first
	cachedMetadata, err := cache.LoadFileListCache(a.state.Config.SearchDir, a.state.Config.CacheExpiryDays)
	if err == nil && cachedMetadata != nil {
		// Cache hit - return cached metadata
		return filesScannedWithMetadataMsg{metadata: cachedMetadata}
	}

	// Cache miss or error - scan directory with metadata collection
	metadata, err := library.ScanFilesWithMetadata(a.state.Config.SearchDir)
	if err != nil {
		return errMsg{err: err}
	}

	// Save to cache for next time
	cache.SaveFileListCache(a.state.Config.SearchDir, metadata)

	return filesScannedWithMetadataMsg{metadata: metadata}
}

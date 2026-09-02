package ui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/config"
	"github.com/subbass/litreader/internal/state"
)

func newExploreAuthorsTestModel(t *testing.T) *ExploreAuthorsModel {
	t.Helper()

	searchDir := t.TempDir()
	cfg := config.NewDefaultConfig()
	cfg.SearchDir = searchDir
	s := state.NewState(cfg)
	s.AllFiles = []string{
		filepath.Join(searchDir, "Alice", "story.txt"),
		filepath.Join(searchDir, "Bob", "story.txt"),
	}

	model := NewExploreAuthorsModel(s, NewStyles(cfg))
	model.height = 20
	return model
}

func TestExploreAuthorsFilterAcceptsNavigationLetters(t *testing.T) {
	model := newExploreAuthorsTestModel(t)
	model.filterMode = true
	model.filterInput.Focus()

	for _, r := range "hjklq" {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = updated.(*ExploreAuthorsModel)
	}

	if got, want := model.filterInput.Value(), "hjklq"; got != want {
		t.Fatalf("filter value = %q, want %q", got, want)
	}
}

func TestExploreAuthorsFilterUsesArrowKeysForListNavigation(t *testing.T) {
	model := newExploreAuthorsTestModel(t)
	model.filterMode = true
	model.filterInput.Focus()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(*ExploreAuthorsModel)

	if got, want := model.cursor, 1; got != want {
		t.Fatalf("cursor = %d, want %d", got, want)
	}
	if got := model.filterInput.Value(); got != "" {
		t.Fatalf("filter value = %q, want empty", got)
	}
}

func TestAppDoesNotNavigateWhileAuthorFilterIsActive(t *testing.T) {
	model := newExploreAuthorsTestModel(t)
	app := NewApp(model.state)
	app.currentView = ViewExploreAuthors
	app.viewStack = []ViewType{ViewDashboard}
	app.exploreAuthors = model
	app.exploreAuthors.filterMode = true
	app.exploreAuthors.filterInput.Focus()

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = updated.(*App)

	if app.currentView != ViewExploreAuthors {
		t.Fatalf("current view = %v, want ViewExploreAuthors", app.currentView)
	}
	if got, want := app.exploreAuthors.filterInput.Value(), "q"; got != want {
		t.Fatalf("filter value = %q, want %q", got, want)
	}
}

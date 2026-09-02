package ui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/subbass/litreader/internal/config"
	"github.com/subbass/litreader/internal/state"
)

func newAuthorFilesTestModel(t *testing.T) *AuthorFilesModel {
	t.Helper()

	searchDir := t.TempDir()
	authorFolder := filepath.Join(searchDir, "Alice")
	if err := os.MkdirAll(authorFolder, 0755); err != nil {
		t.Fatal(err)
	}

	files := []string{
		filepath.Join(authorFolder, "dragon story.txt"),
		filepath.Join(authorFolder, "space opera.txt"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.NewDefaultConfig()
	cfg.SearchDir = searchDir
	s := state.NewState(cfg)
	s.AllFiles = files

	model := NewAuthorFilesModel(s, NewStyles(cfg), authorFolder)
	model.height = 20
	return model
}

func TestAuthorFilesFilterAcceptsNavigationLetters(t *testing.T) {
	model := newAuthorFilesTestModel(t)
	model.filterMode = true
	model.filterInput.Focus()

	for _, r := range "hjklq" {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = updated.(*AuthorFilesModel)
	}

	if got, want := model.filterInput.Value(), "hjklq"; got != want {
		t.Fatalf("filter value = %q, want %q", got, want)
	}
}

func TestAuthorFilesFilterNarrowsListByName(t *testing.T) {
	model := newAuthorFilesTestModel(t)
	model.filterMode = true
	model.filterInput.Focus()

	for _, r := range "dragon" {
		updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		model = updated.(*AuthorFilesModel)
	}

	if got, want := len(model.filtered), 1; got != want {
		t.Fatalf("filtered count = %d, want %d", got, want)
	}
	if got, want := filepath.Base(model.filtered[0]), "dragon story.txt"; got != want {
		t.Fatalf("filtered file = %q, want %q", got, want)
	}
}

func TestAuthorFilesFilterUsesArrowKeysForListNavigation(t *testing.T) {
	model := newAuthorFilesTestModel(t)
	model.filterMode = true
	model.filterInput.Focus()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(*AuthorFilesModel)

	if got, want := model.cursor, 1; got != want {
		t.Fatalf("cursor = %d, want %d", got, want)
	}
	if got := model.filterInput.Value(); got != "" {
		t.Fatalf("filter value = %q, want empty", got)
	}
}

func TestAppDoesNotNavigateWhileAuthorFilesFilterIsActive(t *testing.T) {
	model := newAuthorFilesTestModel(t)
	app := NewApp(model.state)
	app.currentView = ViewAuthorFiles
	app.viewStack = []ViewType{ViewExploreAuthors}
	app.authorFiles = model
	app.authorFiles.filterMode = true
	app.authorFiles.filterInput.Focus()

	updated, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	app = updated.(*App)

	if app.currentView != ViewAuthorFiles {
		t.Fatalf("current view = %v, want ViewAuthorFiles", app.currentView)
	}
	if got, want := app.authorFiles.filterInput.Value(), "q"; got != want {
		t.Fatalf("filter value = %q, want %q", got, want)
	}
}

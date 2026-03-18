package screensaver

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCenteredStart(t *testing.T) {
	if got := centeredStart(80, 40); got != 20 {
		t.Fatalf("centeredStart(80,40) = %d, want 20", got)
	}
}

func TestCenteredStartClampsAtZero(t *testing.T) {
	if got := centeredStart(20, 40); got != 0 {
		t.Fatalf("centeredStart(20,40) = %d, want 0", got)
	}
}

func TestUpdateQuitsOnAnyKeyPress(t *testing.T) {
	m := New()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatalf("expected quit command on arbitrary keypress")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestUpdateNKeyFetchesNextQuoteInsteadOfQuit(t *testing.T) {
	m := New()
	m.nextQuoteCmd = func() tea.Cmd {
		return func() tea.Msg {
			return quoteLoadedMsg{Quote: Quote{Text: "next", Author: "tester"}}
		}
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatalf("expected next-quote command on n key")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); ok {
		t.Fatalf("n key should not quit")
	}
	if _, ok := msg.(quoteLoadedMsg); !ok {
		t.Fatalf("expected quoteLoadedMsg, got %T", msg)
	}
}

func TestUpdateQuitsOnMouseInput(t *testing.T) {
	m := New()
	_, cmd := m.Update(tea.MouseMsg{})
	if cmd == nil {
		t.Fatalf("expected quit command on mouse input")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestUpdateAdvancesQuoteShine(t *testing.T) {
	m := New()
	updated, cmd := m.Update(quoteShineTickMsg{})
	got, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected updated model type Model, got %T", updated)
	}
	if got.quoteShinePos <= m.quoteShinePos {
		t.Fatalf("quoteShinePos did not advance: before=%d after=%d", m.quoteShinePos, got.quoteShinePos)
	}
	if cmd == nil {
		t.Fatalf("expected quote shine to schedule next tick")
	}
}

func TestRenderShinyQuoteNoHighlightBeforeStart(t *testing.T) {
	content := "abc"
	got := renderShinyQuote(content, 0, false)
	want := quoteBaseStyle.Render("a") + quoteBaseStyle.Render("b") + quoteBaseStyle.Render("c")
	if got != want {
		t.Fatalf("renderShinyQuote() before start should have no highlight")
	}
}

func TestRenderShinyQuoteHighlightsAfterStart(t *testing.T) {
	content := "abc"
	got := renderShinyQuote(content, 0, true)
	if !strings.Contains(got, quoteShineStyle.Render("a")) {
		t.Fatalf("renderShinyQuote() after start should include highlighted content")
	}
}

func TestUpdateQuitsWhenWindowTooSmallForQuote(t *testing.T) {
	m := New()
	m.quoteLoaded = true
	m.quote = Quote{Text: "small terminal should quit", Author: "tester"}

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
	if cmd == nil {
		t.Fatalf("expected quit command for undersized terminal")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestUpdateDoesNotQuitWhenWindowFitsQuote(t *testing.T) {
	m := New()
	m.quoteLoaded = true
	m.quote = Quote{Text: "this should fit in a normal terminal window", Author: "tester"}

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Fatalf("did not expect quit for large terminal")
		}
	}

	got, ok := updated.(Model)
	if !ok {
		t.Fatalf("expected updated model type Model, got %T", updated)
	}
	if got.width != 120 || got.height != 40 {
		t.Fatalf("expected model size to update to 120x40, got %dx%d", got.width, got.height)
	}
}

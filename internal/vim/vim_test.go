package vim

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseNavigation(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected NavigationAction
	}{
		{"k moves up", "k", MoveUp},
		{"j moves down", "j", MoveDown},
		{"g moves to top", "g", MoveTop},
		{"G moves to bottom", "G", MoveBottom},
		{"ctrl+g moves to bottom", "ctrl+g", MoveBottom},
		{"ctrl+d half page down", "ctrl+d", MoveHalfPageDown},
		{"ctrl+u half page up", "ctrl+u", MoveHalfPageUp},
		{"ctrl+f page down", "ctrl+f", MovePageDown},
		{"ctrl+b page up", "ctrl+b", MovePageUp},
		{"/ enters search", "/", EnterSearch},
		{"n next search", "n", NextSearchResult},
		{"N prev search", "N", PrevSearchResult},
		{": enters command", ":", EnterCommand},
		{"unknown key", "x", NoAction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+d" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlD}
			} else if tt.key == "ctrl+u" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlU}
			} else if tt.key == "ctrl+f" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlF}
			} else if tt.key == "ctrl+b" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlB}
			} else if tt.key == "ctrl+g" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlG}
			}

			result := ParseNavigation(msg)
			if result != tt.expected {
				t.Errorf("ParseNavigation(%s) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestCalculateNewIndex(t *testing.T) {
	tests := []struct {
		name     string
		action   NavigationAction
		current  int
		length   int
		pageSize int
		expected int
	}{
		{"move up from middle", MoveUp, 5, 10, 10, 4},
		{"move up at top", MoveUp, 0, 10, 10, 0},
		{"move down from middle", MoveDown, 5, 10, 10, 6},
		{"move down at bottom", MoveDown, 9, 10, 10, 9},
		{"move to top", MoveTop, 5, 10, 10, 0},
		{"move to bottom", MoveBottom, 5, 10, 10, 9},
		{"half page down", MoveHalfPageDown, 5, 100, 20, 15},
		{"half page up", MoveHalfPageUp, 15, 100, 20, 5},
		{"page down", MovePageDown, 5, 100, 20, 25},
		{"page up", MovePageUp, 25, 100, 20, 5},
		{"half page down near end", MoveHalfPageDown, 95, 100, 20, 99},
		{"half page up near start", MoveHalfPageUp, 5, 100, 20, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateNewIndex(tt.action, tt.current, tt.length, tt.pageSize)
			if result != tt.expected {
				t.Errorf("CalculateNewIndex(%v, %d, %d, %d) = %d, want %d",
					tt.action, tt.current, tt.length, tt.pageSize, result, tt.expected)
			}
		})
	}
}

func TestSearchItems(t *testing.T) {
	state := NewState()
	items := []string{
		"ec2-instance-1",
		"ec2-instance-2",
		"s3-bucket-test",
		"ec2-instance-3",
	}

	state.SearchQuery = "ec2"
	state.SearchItems(items)

	if len(state.SearchResults) != 3 {
		t.Errorf("SearchItems found %d results, want 3", len(state.SearchResults))
	}

	expectedIndices := []int{0, 1, 3}
	for i, expected := range expectedIndices {
		if state.SearchResults[i] != expected {
			t.Errorf("SearchResults[%d] = %d, want %d", i, state.SearchResults[i], expected)
		}
	}
}

func TestNextPrevMatch(t *testing.T) {
	state := NewState()
	state.SearchResults = []int{0, 3, 5, 7}
	state.CurrentMatch = 0

	// Test NextMatch
	next := state.NextMatch()
	if next != 3 {
		t.Errorf("NextMatch() = %d, want 3", next)
	}

	next = state.NextMatch()
	if next != 5 {
		t.Errorf("NextMatch() = %d, want 5", next)
	}

	// Test wrapping around
	state.CurrentMatch = 3
	next = state.NextMatch()
	if next != 0 {
		t.Errorf("NextMatch() (wrap) = %d, want 0", next)
	}

	// Test PrevMatch
	prev := state.PrevMatch()
	if prev != 7 {
		t.Errorf("PrevMatch() (wrap) = %d, want 7", prev)
	}

	prev = state.PrevMatch()
	if prev != 5 {
		t.Errorf("PrevMatch() = %d, want 5", prev)
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected Command
	}{
		{"q", Command{Name: "q", Args: []string{}}},
		{"refresh", Command{Name: "refresh", Args: []string{}}},
		{"echo hello world", Command{Name: "echo", Args: []string{"hello", "world"}}},
		{"", Command{Name: "", Args: []string{}}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseCommand(tt.input)
			if result.Name != tt.expected.Name {
				t.Errorf("ParseCommand(%q).Name = %q, want %q", tt.input, result.Name, tt.expected.Name)
			}
			if len(result.Args) != len(tt.expected.Args) {
				t.Errorf("ParseCommand(%q).Args length = %d, want %d", tt.input, len(result.Args), len(tt.expected.Args))
			}
		})
	}
}

func TestModeTransitions(t *testing.T) {
	state := NewState()

	// Start in NormalMode
	if state.Mode != NormalMode {
		t.Errorf("Initial mode = %v, want NormalMode", state.Mode)
	}

	// Enter search mode
	state.EnterSearchMode()
	if state.Mode != SearchMode {
		t.Errorf("After EnterSearchMode, mode = %v, want SearchMode", state.Mode)
	}

	// Exit search mode
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	state.HandleKey(msg)
	if state.Mode != NormalMode {
		t.Errorf("After ESC in SearchMode, mode = %v, want NormalMode", state.Mode)
	}

	// Enter command mode
	state.EnterCommandMode()
	if state.Mode != CommandMode {
		t.Errorf("After EnterCommandMode, mode = %v, want CommandMode", state.Mode)
	}

	// Exit command mode
	state.HandleKey(msg)
	if state.Mode != NormalMode {
		t.Errorf("After ESC in CommandMode, mode = %v, want NormalMode", state.Mode)
	}
}

func TestMarkFunctionality(t *testing.T) {
	state := NewState()

	// Set marks
	state.SetMark('a', 10)
	state.SetMark('b', 20)

	// Get marks
	pos, ok := state.GetMark('a')
	if !ok || pos != 10 {
		t.Errorf("GetMark('a') = (%d, %v), want (10, true)", pos, ok)
	}

	pos, ok = state.GetMark('b')
	if !ok || pos != 20 {
		t.Errorf("GetMark('b') = (%d, %v), want (20, true)", pos, ok)
	}

	// Non-existent mark
	_, ok = state.GetMark('z')
	if ok {
		t.Errorf("GetMark('z') returned true, want false")
	}
}

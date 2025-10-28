package vim

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Mode represents the current VIM mode
type Mode int

const (
	NormalMode Mode = iota
	SearchMode
	CommandMode
)

// State holds the VIM-like state for navigation
type State struct {
	Mode          Mode
	SearchQuery   string
	SearchResults []int // indices of matching items
	CurrentMatch  int   // current position in SearchResults
	CommandBuffer string
	LastSearch    string       // for n/N navigation
	MarkPosition  map[rune]int // VIM marks (m{a-z})
}

// NewState creates a new VIM state
func NewState() *State {
	return &State{
		Mode:          NormalMode,
		SearchResults: []int{},
		MarkPosition:  make(map[rune]int),
	}
}

// HandleKey processes a key press and returns true if handled
func (s *State) HandleKey(msg tea.KeyMsg) bool {
	switch s.Mode {
	case SearchMode:
		return s.handleSearchMode(msg)
	case CommandMode:
		return s.handleCommandMode(msg)
	default:
		return false
	}
}

func (s *State) handleSearchMode(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "enter":
		s.LastSearch = s.SearchQuery
		s.Mode = NormalMode
		return true
	case "esc":
		s.SearchQuery = ""
		s.SearchResults = []int{}
		s.CurrentMatch = 0
		s.Mode = NormalMode
		return true
	case "backspace":
		if len(s.SearchQuery) > 0 {
			s.SearchQuery = s.SearchQuery[:len(s.SearchQuery)-1]
		}
		return true
	default:
		// Add character to search query
		if len(msg.String()) == 1 {
			s.SearchQuery += msg.String()
		}
		return true
	}
}

func (s *State) handleCommandMode(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "enter":
		// Command will be processed by the caller
		s.Mode = NormalMode
		return true
	case "esc":
		s.CommandBuffer = ""
		s.Mode = NormalMode
		return true
	case "backspace":
		if len(s.CommandBuffer) > 0 {
			s.CommandBuffer = s.CommandBuffer[:len(s.CommandBuffer)-1]
		}
		return true
	default:
		// Add character to command buffer
		if len(msg.String()) == 1 {
			s.CommandBuffer += msg.String()
		}
		return true
	}
}

// EnterSearchMode activates search mode
func (s *State) EnterSearchMode() {
	s.Mode = SearchMode
	s.SearchQuery = ""
	s.SearchResults = []int{}
	s.CurrentMatch = 0
}

// EnterCommandMode activates command mode
func (s *State) EnterCommandMode() {
	s.Mode = CommandMode
	s.CommandBuffer = ""
}

// SearchItems performs a search on a list of items and updates SearchResults
func (s *State) SearchItems(items []string) {
	s.SearchResults = []int{}
	query := strings.ToLower(s.SearchQuery)

	for i, item := range items {
		if strings.Contains(strings.ToLower(item), query) {
			s.SearchResults = append(s.SearchResults, i)
		}
	}

	if len(s.SearchResults) > 0 {
		s.CurrentMatch = 0
	}
}

// NextMatch moves to the next search result
func (s *State) NextMatch() int {
	if len(s.SearchResults) == 0 {
		return -1
	}
	s.CurrentMatch = (s.CurrentMatch + 1) % len(s.SearchResults)
	return s.SearchResults[s.CurrentMatch]
}

// PrevMatch moves to the previous search result
func (s *State) PrevMatch() int {
	if len(s.SearchResults) == 0 {
		return -1
	}
	s.CurrentMatch--
	if s.CurrentMatch < 0 {
		s.CurrentMatch = len(s.SearchResults) - 1
	}
	return s.SearchResults[s.CurrentMatch]
}

// GetCurrentMatch returns the current match index
func (s *State) GetCurrentMatch() int {
	if len(s.SearchResults) == 0 {
		return -1
	}
	return s.SearchResults[s.CurrentMatch]
}

// SetMark sets a mark at the given position
func (s *State) SetMark(mark rune, position int) {
	s.MarkPosition[mark] = position
}

// GetMark returns the position of a mark
func (s *State) GetMark(mark rune) (int, bool) {
	pos, ok := s.MarkPosition[mark]
	return pos, ok
}

// NavigationAction represents a VIM navigation action
type NavigationAction int

const (
	NoAction NavigationAction = iota
	MoveUp
	MoveDown
	MoveTop
	MoveBottom
	MoveHalfPageUp
	MoveHalfPageDown
	MovePageUp
	MovePageDown
	EnterSearch
	NextSearchResult
	PrevSearchResult
	EnterCommand
)

// ParseNavigation parses a key press into a navigation action
func ParseNavigation(msg tea.KeyMsg) NavigationAction {
	switch msg.String() {
	case "k", "up":
		return MoveUp
	case "j", "down":
		return MoveDown
	case "g":
		return MoveTop
	case "G", "ctrl+g":
		return MoveBottom
	case "ctrl+u":
		return MoveHalfPageUp
	case "ctrl+d":
		return MoveHalfPageDown
	case "ctrl+b", "pgup":
		return MovePageUp
	case "ctrl+f", "pgdown":
		return MovePageDown
	case "/":
		return EnterSearch
	case "n":
		return NextSearchResult
	case "N":
		return PrevSearchResult
	case ":":
		return EnterCommand
	default:
		return NoAction
	}
}

// CalculateNewIndex calculates the new index based on navigation action
func CalculateNewIndex(action NavigationAction, currentIndex, listLength, pageSize int) int {
	switch action {
	case MoveUp:
		if currentIndex > 0 {
			return currentIndex - 1
		}
		return currentIndex
	case MoveDown:
		if currentIndex < listLength-1 {
			return currentIndex + 1
		}
		return currentIndex
	case MoveTop:
		return 0
	case MoveBottom:
		return listLength - 1
	case MoveHalfPageUp:
		newIndex := currentIndex - pageSize/2
		if newIndex < 0 {
			return 0
		}
		return newIndex
	case MoveHalfPageDown:
		newIndex := currentIndex + pageSize/2
		if newIndex >= listLength {
			return listLength - 1
		}
		return newIndex
	case MovePageUp:
		newIndex := currentIndex - pageSize
		if newIndex < 0 {
			return 0
		}
		return newIndex
	case MovePageDown:
		newIndex := currentIndex + pageSize
		if newIndex >= listLength {
			return listLength - 1
		}
		return newIndex
	default:
		return currentIndex
	}
}

// Command represents a parsed VIM-style command
type Command struct {
	Name string
	Args []string
}

// ParseCommand parses a command string
func ParseCommand(commandStr string) Command {
	parts := strings.Fields(commandStr)
	if len(parts) == 0 {
		return Command{}
	}

	return Command{
		Name: parts[0],
		Args: parts[1:],
	}
}

// Common VIM commands for k9s-like interface
const (
	CmdQuit          = "q"
	CmdQuitAll       = "qa"
	CmdWrite         = "w"
	CmdWriteQuit     = "wq"
	CmdRefresh       = "r"
	CmdHelp          = "help"
	CmdClearFilter   = "cf"
	CmdSelectAll     = "sa"
	CmdDeselectAll   = "da"
	CmdToggleDetails = "d"
	CmdEC2           = "ec2"
	CmdS3            = "s3"
	CmdEKS           = "eks"
	CmdAccount       = "account"
	CmdRegion        = "region"
)

// AllCommands returns a list of all available commands for completion
func AllCommands() []string {
	return []string{
		"q", "quit", "qa",
		"r", "refresh",
		"help", "h",
		"cf", "clearfilter",
		"sa", "selectall",
		"da", "deselectall",
		"ec2", "s3", "eks",
		"account", "acc",
		"region",
	}
}

// GetCommandSuggestions returns commands that match the given prefix
func GetCommandSuggestions(prefix string) []string {
	if prefix == "" {
		return AllCommands()
	}

	var suggestions []string
	for _, cmd := range AllCommands() {
		if strings.HasPrefix(cmd, prefix) {
			suggestions = append(suggestions, cmd)
		}
	}
	return suggestions
}

// CompleteCommand returns the completed command if there's only one match
func CompleteCommand(prefix string) (string, bool) {
	suggestions := GetCommandSuggestions(prefix)
	if len(suggestions) == 1 {
		return suggestions[0], true
	}

	// Find common prefix among all suggestions
	if len(suggestions) > 1 {
		common := suggestions[0]
		for _, s := range suggestions[1:] {
			// Find common prefix between common and s
			i := 0
			for i < len(common) && i < len(s) && common[i] == s[i] {
				i++
			}
			common = common[:i]
		}
		if len(common) > len(prefix) {
			return common, false
		}
	}

	return prefix, false
}

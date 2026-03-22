package editor

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SearchMode indicates what kind of overlay is active.
type SearchMode int

const (
	SearchNone SearchMode = iota
	SearchFind
	SearchReplace
	SearchGoToLine
)

// Match represents a search match position.
type Match struct {
	Line, Col, Len int
}

// SearchOverlay handles Ctrl+F find, Ctrl+H replace, and Ctrl+G go-to-line.
type SearchOverlay struct {
	mode    SearchMode
	query   string
	replace string
	cursor  int // cursor position within query input
	editing int // 0 = query field, 1 = replace field

	matches    []Match
	matchIdx   int
	caseSense  bool
}

var (
	overlayStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#282A36")).
			Foreground(lipgloss.Color("#F8F8F2")).
			Padding(0, 1)
	inputStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#44475A")).
			Foreground(lipgloss.Color("#F8F8F2")).
			Padding(0, 1)
	matchHLStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FFB86C")).
			Foreground(lipgloss.Color("#282A36"))
	activeMatchStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#FF79C6")).
				Foreground(lipgloss.Color("#282A36"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
)

func (s *SearchOverlay) Active() bool { return s.mode != SearchNone }
func (s *SearchOverlay) Mode() SearchMode { return s.mode }

func (s *SearchOverlay) Open(mode SearchMode) {
	s.mode = mode
	s.query = ""
	s.replace = ""
	s.cursor = 0
	s.editing = 0
	s.matches = nil
	s.matchIdx = 0
}

func (s *SearchOverlay) Close() {
	s.mode = SearchNone
	s.matches = nil
}

// Update handles input for the search overlay. Returns true if the message was consumed.
func (s *SearchOverlay) Update(msg tea.Msg, buf *Buffer) (consumed bool, cmd tea.Cmd) {
	if s.mode == SearchNone {
		return false, nil
	}

	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false, nil
	}

	ks := km.String()
	switch ks {
	case "esc":
		s.Close()
		return true, nil
	case "enter":
		if s.mode == SearchGoToLine {
			s.goToLine(buf)
			s.Close()
			return true, nil
		}
		if s.mode == SearchReplace && s.editing == 1 {
			s.replaceOne(buf)
			return true, nil
		}
		// Find: jump to next match
		s.nextMatch(buf)
		return true, nil
	case "tab":
		if s.mode == SearchReplace {
			s.editing = 1 - s.editing
			if s.editing == 0 {
				s.cursor = len(s.query)
			} else {
				s.cursor = len(s.replace)
			}
		}
		return true, nil
	case "ctrl+shift+enter":
		if s.mode == SearchReplace {
			s.replaceAll(buf)
			return true, nil
		}
	case "up", "ctrl+p":
		s.prevMatch(buf)
		return true, nil
	case "down", "ctrl+n":
		s.nextMatch(buf)
		return true, nil
	case "backspace":
		s.backspace()
		s.search(buf)
		return true, nil
	case "left":
		if s.cursor > 0 {
			s.cursor--
		}
		return true, nil
	case "right":
		field := s.activeField()
		if s.cursor < len(field) {
			s.cursor++
		}
		return true, nil
	default:
		if k := km.Key(); k.Text != "" {
			s.insertText(k.Text)
			s.search(buf)
			return true, nil
		}
	}
	return true, nil
}

func (s *SearchOverlay) activeField() string {
	if s.editing == 1 {
		return s.replace
	}
	return s.query
}

func (s *SearchOverlay) insertText(text string) {
	if s.editing == 1 {
		s.replace = s.replace[:s.cursor] + text + s.replace[s.cursor:]
	} else {
		s.query = s.query[:s.cursor] + text + s.query[s.cursor:]
	}
	s.cursor += len(text)
}

func (s *SearchOverlay) backspace() {
	if s.cursor <= 0 {
		return
	}
	if s.editing == 1 {
		s.replace = s.replace[:s.cursor-1] + s.replace[s.cursor:]
	} else {
		s.query = s.query[:s.cursor-1] + s.query[s.cursor:]
	}
	s.cursor--
}

func (s *SearchOverlay) search(buf *Buffer) {
	s.matches = nil
	s.matchIdx = 0
	if s.query == "" {
		return
	}
	q := s.query
	if !s.caseSense {
		q = strings.ToLower(q)
	}
	for i := 0; i < buf.LineCount(); i++ {
		line := buf.Line(i)
		search := line
		if !s.caseSense {
			search = strings.ToLower(line)
		}
		off := 0
		for {
			idx := strings.Index(search[off:], q)
			if idx < 0 {
				break
			}
			s.matches = append(s.matches, Match{Line: i, Col: off + idx, Len: len(s.query)})
			off += idx + max(len(q), 1)
			if off >= len(search) {
				break
			}
		}
	}
	// Jump to nearest match from cursor
	if len(s.matches) > 0 {
		for i, m := range s.matches {
			if m.Line > buf.Cursor.Line || (m.Line == buf.Cursor.Line && m.Col >= buf.Cursor.Col) {
				s.matchIdx = i
				s.jumpToMatch(buf)
				return
			}
		}
		s.matchIdx = 0
		s.jumpToMatch(buf)
	}
}

func (s *SearchOverlay) nextMatch(buf *Buffer) {
	if len(s.matches) == 0 {
		return
	}
	s.matchIdx = (s.matchIdx + 1) % len(s.matches)
	s.jumpToMatch(buf)
}

func (s *SearchOverlay) prevMatch(buf *Buffer) {
	if len(s.matches) == 0 {
		return
	}
	s.matchIdx = (s.matchIdx - 1 + len(s.matches)) % len(s.matches)
	s.jumpToMatch(buf)
}

func (s *SearchOverlay) jumpToMatch(buf *Buffer) {
	if s.matchIdx >= len(s.matches) {
		return
	}
	m := s.matches[s.matchIdx]
	buf.Cursor.Line = m.Line
	buf.Cursor.Col = m.Col
	// Select the match
	buf.Selection = Selection{
		Anchor: Position{Line: m.Line, Col: m.Col},
		Head:   Position{Line: m.Line, Col: m.Col + m.Len},
		Active: true,
	}
}

func (s *SearchOverlay) replaceOne(buf *Buffer) {
	if len(s.matches) == 0 {
		return
	}
	m := s.matches[s.matchIdx]
	buf.Delete(Position{Line: m.Line, Col: m.Col}, Position{Line: m.Line, Col: m.Col + m.Len})
	buf.Insert(Position{Line: m.Line, Col: m.Col}, s.replace)
	s.search(buf)
}

func (s *SearchOverlay) replaceAll(buf *Buffer) {
	// Replace from bottom to top to preserve positions
	for i := len(s.matches) - 1; i >= 0; i-- {
		m := s.matches[i]
		buf.Delete(Position{Line: m.Line, Col: m.Col}, Position{Line: m.Line, Col: m.Col + m.Len})
		buf.Insert(Position{Line: m.Line, Col: m.Col}, s.replace)
	}
	s.search(buf)
}

func (s *SearchOverlay) goToLine(buf *Buffer) {
	var line int
	if _, err := fmt.Sscanf(s.query, "%d", &line); err == nil {
		line-- // 1-indexed to 0-indexed
		if line < 0 {
			line = 0
		}
		if line >= buf.LineCount() {
			line = buf.LineCount() - 1
		}
		buf.Cursor.Line = line
		buf.Cursor.Col = 0
	}
}

// Matches returns current matches for rendering highlights.
func (s *SearchOverlay) Matches() []Match { return s.matches }
func (s *SearchOverlay) MatchIdx() int    { return s.matchIdx }
func (s *SearchOverlay) Query() string    { return s.query }

// View renders the search overlay bar.
func (s *SearchOverlay) View(width int) string {
	if s.mode == SearchNone {
		return ""
	}

	switch s.mode {
	case SearchGoToLine:
		input := inputStyle.Width(max(width/3, 10)).Render(s.query + "▏")
		return overlayStyle.Width(width).Render(labelStyle.Render("Go to line: ") + input)

	case SearchFind:
		info := ""
		if s.query != "" {
			info = fmt.Sprintf(" %d/%d", s.matchIdx+1, len(s.matches))
			if len(s.matches) == 0 {
				info = " No results"
			}
		}
		input := inputStyle.Width(max(width/3, 10)).Render(s.query + "▏")
		return overlayStyle.Width(width).Render(labelStyle.Render("Find: ") + input + info)

	case SearchReplace:
		info := ""
		if s.query != "" {
			info = fmt.Sprintf(" %d/%d", s.matchIdx+1, len(s.matches))
			if len(s.matches) == 0 {
				info = " No results"
			}
		}
		qStyle := inputStyle
		rStyle := inputStyle
		qText := s.query
		rText := s.replace
		if s.editing == 0 {
			qText += "▏"
		} else {
			rText += "▏"
		}
		iw := max(width/4, 8)
		return overlayStyle.Width(width).Render(
			labelStyle.Render("Find: ") + qStyle.Width(iw).Render(qText) + info +
				"  " + labelStyle.Render("Replace: ") + rStyle.Width(iw).Render(rText))
	}
	return ""
}

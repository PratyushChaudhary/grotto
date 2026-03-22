package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// OverlayMode indicates which overlay is active.
type OverlayMode int

const (
	OverlayNone OverlayMode = iota
	OverlayFileFinder
	OverlayCommandPalette
)

// Overlay handles Ctrl+P file finder and Ctrl+Shift+P command palette.
type Overlay struct {
	mode    OverlayMode
	query   string
	cursor  int // cursor in query
	items   []string
	filtered []string
	selected int
}

type Command struct {
	Name   string
	Action func(m *Model)
}

var (
	overlayBoxStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#282A36")).
		Foreground(lipgloss.Color("#F8F8F2")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 1)
	overlayInputStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#44475A")).
		Foreground(lipgloss.Color("#F8F8F2")).
		Padding(0, 1)
	overlayItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))
	overlaySelectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#44475A")).
		Foreground(lipgloss.Color("#F8F8F2"))
)

func (o *Overlay) Active() bool { return o.mode != OverlayNone }

func (o *Overlay) OpenFileFinder(rootPath string) {
	o.mode = OverlayFileFinder
	o.query = ""
	o.cursor = 0
	o.selected = 0
	o.items = walkFiles(rootPath, 500)
	o.filtered = o.items
}

func (o *Overlay) OpenCommandPalette(commands []string) {
	o.mode = OverlayCommandPalette
	o.query = ""
	o.cursor = 0
	o.selected = 0
	o.items = commands
	o.filtered = commands
}

func (o *Overlay) Close() {
	o.mode = OverlayNone
}

// Update handles input. Returns (consumed, selectedItem) where selectedItem is non-empty on Enter.
func (o *Overlay) Update(msg tea.Msg) (bool, string) {
	if o.mode == OverlayNone {
		return false, ""
	}
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false, ""
	}

	switch km.String() {
	case "esc":
		o.Close()
		return true, ""
	case "enter":
		result := ""
		if len(o.filtered) > 0 && o.selected < len(o.filtered) {
			result = o.filtered[o.selected]
		}
		o.Close()
		return true, result
	case "up", "ctrl+p":
		if o.selected > 0 {
			o.selected--
		}
		return true, ""
	case "down", "ctrl+n":
		if o.selected < len(o.filtered)-1 {
			o.selected++
		}
		return true, ""
	case "backspace":
		if o.cursor > 0 {
			o.query = o.query[:o.cursor-1] + o.query[o.cursor:]
			o.cursor--
			o.filter()
		}
		return true, ""
	default:
		if k := km.Key(); k.Text != "" {
			o.query = o.query[:o.cursor] + k.Text + o.query[o.cursor:]
			o.cursor += len(k.Text)
			o.filter()
			return true, ""
		}
	}
	return true, ""
}

func (o *Overlay) filter() {
	if o.query == "" {
		o.filtered = o.items
		o.selected = 0
		return
	}
	q := strings.ToLower(o.query)
	o.filtered = nil
	for _, item := range o.items {
		if fuzzyMatch(strings.ToLower(item), q) {
			o.filtered = append(o.filtered, item)
		}
	}
	o.selected = 0
}

// fuzzyMatch checks if all chars of pattern appear in s in order.
func fuzzyMatch(s, pattern string) bool {
	pi := 0
	for i := 0; i < len(s) && pi < len(pattern); i++ {
		if s[i] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}

func (o *Overlay) View(width, height int) string {
	if o.mode == OverlayNone {
		return ""
	}

	boxW := min(width-4, 60)
	maxItems := min(len(o.filtered), 12)

	label := "> "
	if o.mode == OverlayCommandPalette {
		label = "> "
	}
	input := overlayInputStyle.Width(boxW - 4).Render(label + o.query + "▏")

	var lines []string
	lines = append(lines, input)

	start := 0
	if o.selected >= maxItems {
		start = o.selected - maxItems + 1
	}
	end := min(start+maxItems, len(o.filtered))

	for i := start; i < end; i++ {
		s := overlayItemStyle
		if i == o.selected {
			s = overlaySelectedStyle
		}
		lines = append(lines, s.Width(boxW-4).Render(o.filtered[i]))
	}

	content := strings.Join(lines, "\n")
	box := overlayBoxStyle.Width(boxW).Render(content)

	// Center horizontally
	pad := max((width-boxW-2)/2, 0)
	return strings.Repeat(" ", pad) + box
}

// walkFiles collects relative file paths under root, up to limit.
func walkFiles(root string, limit int) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		// Skip hidden dirs and common noise
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__") {
			return filepath.SkipDir
		}
		if !d.IsDir() && !strings.HasPrefix(name, ".") {
			rel, _ := filepath.Rel(root, path)
			files = append(files, rel)
			if len(files) >= limit {
				return filepath.SkipAll
			}
		}
		return nil
	})
	sort.Strings(files)
	return files
}

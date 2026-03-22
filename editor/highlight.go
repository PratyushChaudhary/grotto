package editor

import (
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlighter caches syntax tokens per line.
type Highlighter struct {
	lexer chroma.Lexer
	style *chroma.Style
	cache map[int][]StyledSpan // line number → spans
}

// StyledSpan is a chunk of text with a lipgloss style.
type StyledSpan struct {
	Text  string
	Style lipgloss.Style
}

func NewHighlighter(filePath string) *Highlighter {
	h := &Highlighter{
		style: styles.Get("dracula"),
		cache: make(map[int][]StyledSpan),
	}
	if filePath != "" {
		h.lexer = lexers.Match(filepath.Base(filePath))
	}
	if h.lexer == nil {
		h.lexer = lexers.Fallback
	}
	h.lexer = chroma.Coalesce(h.lexer)
	return h
}

// InvalidateLine clears the cache for a line (call on edit).
func (h *Highlighter) InvalidateLine(line int) {
	delete(h.cache, line)
}

// InvalidateAll clears the entire cache.
func (h *Highlighter) InvalidateAll() {
	h.cache = make(map[int][]StyledSpan)
}

// Highlight returns styled spans for a single line.
func (h *Highlighter) Highlight(lineNum int, text string) []StyledSpan {
	if spans, ok := h.cache[lineNum]; ok {
		return spans
	}

	iter, err := h.lexer.Tokenise(nil, text+"\n")
	if err != nil {
		span := StyledSpan{Text: text, Style: lipgloss.NewStyle()}
		h.cache[lineNum] = []StyledSpan{span}
		return h.cache[lineNum]
	}

	var spans []StyledSpan
	for _, tok := range iter.Tokens() {
		// Strip trailing newlines added by our tokenisation trick
		t := strings.TrimSuffix(tok.Value, "\n")
		if t == "" {
			continue
		}
		s := h.tokenStyle(tok.Type)
		spans = append(spans, StyledSpan{Text: t, Style: s})
	}
	if lineNum >= 0 {
		h.cache[lineNum] = spans
	}
	return spans
}

// RenderLine returns the fully styled string for a line.
func (h *Highlighter) RenderLine(lineNum int, text string) string {
	spans := h.Highlight(lineNum, text)
	var b strings.Builder
	for _, sp := range spans {
		b.WriteString(sp.Style.Render(sp.Text))
	}
	return b.String()
}

func (h *Highlighter) tokenStyle(tt chroma.TokenType) lipgloss.Style {
	s := lipgloss.NewStyle()
	entry := h.style.Get(tt)
	if entry.Colour.IsSet() {
		s = s.Foreground(lipgloss.Color(entry.Colour.String()))
	}
	if entry.Bold == chroma.Yes {
		s = s.Bold(true)
	}
	if entry.Italic == chroma.Yes {
		s = s.Italic(true)
	}
	return s
}

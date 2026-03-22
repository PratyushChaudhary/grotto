package editor

import (
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// clipCopy returns a Cmd that copies text to clipboard using system tools,
// falling back to OSC 52 if no tool is available.
func clipCopy(text string) tea.Cmd {
	if cmd := copyCmd(); cmd != nil {
		return func() tea.Msg {
			c := copyCmd()
			c.Stdin = strings.NewReader(text)
			_ = c.Run()
			return nil
		}
	}
	return tea.SetClipboard(text)
}

// clipPaste returns a Cmd that reads clipboard using system tools,
// falling back to OSC 52 if no tool is available.
func clipPaste() tea.Cmd {
	if pasteCmd() != nil {
		return func() tea.Msg {
			c := pasteCmd()
			out, err := c.Output()
			if err != nil || len(out) == 0 {
				return nil
			}
			return tea.ClipboardMsg{Content: string(out)}
		}
	}
	return func() tea.Msg { return tea.ReadClipboard() }
}

func copyCmd() *exec.Cmd {
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return exec.Command("wl-copy")
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			return exec.Command("xclip", "-selection", "clipboard")
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return exec.Command("xsel", "--clipboard", "--input")
		}
	case "darwin":
		return exec.Command("pbcopy")
	}
	return nil
}

func pasteCmd() *exec.Cmd {
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("wl-paste"); err == nil {
			return exec.Command("wl-paste", "--no-newline")
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			return exec.Command("xclip", "-selection", "clipboard", "-o")
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return exec.Command("xsel", "--clipboard", "--output")
		}
	case "darwin":
		return exec.Command("pbpaste")
	}
	return nil
}

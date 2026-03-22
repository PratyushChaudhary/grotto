# grotto — Development Context

## What is grotto
Terminal-native TUI code editor in Go + Bubble Tea v2. VS Code-like UX: sidebar file tree, syntax-highlighted editor with tabs/split panes, integrated terminal, AI CLI panel. Full spec in `SPEC.md`.

## Tech Stack
- **Go 1.26**, module: `github.com/owomeister/grotto`
- **Bubble Tea v2** (`charm.land/bubbletea/v2 v2.0.2`)
- **Lip Gloss v2** (`charm.land/lipgloss/v2 v2.0.2`)
- **BubbleZone v2** (`github.com/lrstanley/bubblezone/v2 v2.0.0`)
- **Chroma v2** (`github.com/alecthomas/chroma/v2 v2.23.1`)
- **creack/pty v2** (`github.com/creack/pty/v2 v2.0.1`) — PTY spawning
- **vt10x** (`github.com/ActiveState/vt10x v1.3.1`) — VT100 terminal emulator

## CRITICAL: Bubble Tea v2 API Notes
- Root `View()` returns `tea.View{Content, AltScreen, MouseMode}`. Child `View()` returns `string`.
- **Escape key is `"esc"` NOT `"escape"`**
- Key matching: `msg.String()` → `"ctrl+q"`, `"esc"`, `"shift+up"`. Modifier order: `ctrl+alt+shift+meta+hyper+super`
- Mouse: `tea.MouseClickMsg`, `tea.MouseReleaseMsg`, `tea.MouseMotionMsg`, `tea.MouseWheelMsg` — all have `.X`, `.Y`, `.Button`, `.Mod`
- `tea.MouseLeft`, `tea.MouseRight`, `tea.MouseMiddle`, `tea.MouseWheelUp`, `tea.MouseWheelDown`
- Clipboard: `tea.SetClipboard(s)` → Cmd, `tea.ReadClipboard()` → Msg, `tea.ClipboardMsg{Content}`
- OSC 52 clipboard doesn't work in Konsole — system tools used (`wl-copy`/`wl-paste`)
- `lipgloss.Color(s)` is a function returning `color.Color`, NOT a type
- `lipgloss.Padding(0, 1)` = top/bottom=0, left/right=1

## CRITICAL: vt10x Limitations
- Only handles 256-color (`38;5;N`), NOT truecolor (`38;2;R;G;B`)
- `Color` is `uint16` — can't store RGB
- `history` field is unexported — can't read scrollback directly
- `Parse()` reads one batch and returns — must loop it
- `State.Lock()`/`Unlock()` required for concurrent access
- We set `COLORTERM=` (empty) to force programs to use 256-color mode

## File Structure
```
grotto/
├── main.go                 # CLI entry: arg parsing (path, line, --no-ai, --ai, --version)
├── app/
│   ├── app.go              # Root tea.Model — layout, focus, panel toggle, buttons, drag resize, key routing
│   └── overlay.go          # Ctrl+P/F1 file finder, Ctrl+Shift+P/F2 command palette
├── editor/
│   ├── buffer.go           # Buffer: []string lines, cursor, selection, edit ops, undo/redo, bracket matching
│   ├── view.go             # Editor Model: tabs, char-by-char rendering, mouse, keyboard, search, CloseTabMsg
│   ├── highlight.go        # Chroma wrapper: per-line token caching, Dracula theme
│   ├── panes.go            # PaneManager: 1-4 panes, layouts, shared buffers, split/close
│   ├── search.go           # SearchOverlay: find (Ctrl+F), replace (Ctrl+H), go-to-line (Ctrl+G)
│   └── clipboard.go        # System clipboard: wl-copy/xclip/pbcopy with OSC 52 fallback
├── terminal/
│   └── terminal.go         # Terminal model: PTY+vt10x, multi-tab, scrollback, used for both terminal & AI
├── ui/
│   └── sidebar.go          # File tree: recursive dir, expand/collapse, .gitignore, BubbleZone clicks
├── SPEC.md                 # Full project specification
├── KEYBINDS.md             # All keybindings reference
├── CONTEXT.md              # This file
└── go.mod
```

## Architecture
```
app.Model (root)
├── Overlay (file finder / command palette — floats on top)
├── ui.Model (sidebar) — file tree, emits OpenFileMsg{Path}
├── PaneManager — manages 1-4 editor panes
│   └── editor.Model[] — each pane has:
│       ├── Tab[] — multiple tabs per pane
│       │   ├── *Buffer (shared across panes for same file)
│       │   └── *Highlighter + scrollY/scrollX (per-tab)
│       └── SearchOverlay — per-pane find/replace
├── terminal.Model (terminal panel, prefix="term") — PTY+vt10x, multi-tab
└── terminal.Model (AI panel, prefix="ai") — same model, spawns AI CLIs
```

## Key Routing Order
1. App overlay (file finder / command palette) — when active, eats all keys
2. Editor search overlay — when active, dispatched to editor before app keybinds
3. App-level keybinds (Ctrl+Q, Ctrl+B, F1-F4, splits, focus)
4. Focused child dispatch (sidebar, editor, terminal, or AI)
5. Tick messages forwarded to terminal/AI even when not focused (for rendering)

## Title Bar Buttons (clickable, BubbleZone)
`📁 Files` | `🔍 Find` | `⌘ Cmd` | `◫ Split` | `▶ Term` | `✦ AI`
- Active toggles (sidebar/terminal/AI) get inverted highlight style
- `▶ Term` when terminal already open → adds new tab
- `✦ AI` when AI already open → adds new AI tab

## Panel Resize
- **Right-click + drag** resizes panels (picks closest divider automatically)
- Left-click is free for all normal interactions
- Dynamic sizes: `sidebarW`, `aiPanelW`, `terminalRatio` (stored on Model)
- Constraints: min 12 cols for side panels, 10-80% for terminal height

## Terminal Model (`terminal/terminal.go`)
- Shared between terminal panel (prefix="term") and AI panel (prefix="ai")
- `[]*term` tabs, each with own PTY + vt10x State
- Single tick loop per model (50ms / 20fps), prefix-scoped `TickMsg`
- `atomic.Bool` for done flag (no mutex needed for Bubble Tea value copies)
- Scrollback: captures top row on each tick, detects when it changes → saves to `history[]screenLine` (2000 line cap)
- Mouse wheel scrolls through history, any keypress snaps to live
- Tab bar with clickable tabs, `✕` close buttons, `[+]` add button
- Auto-hide panel when last tab closed (app checks `TabCount() == 0`)
- `filterEnv` strips TERM/COLORTERM from parent env, forces `TERM=xterm-256color` + `COLORTERM=`
- `AddTermWithCmd(name, command)` for AI CLIs, `AddTerm()` for shells

## AI Panel
- Uses `terminal.NewAI()` (prefix="ai") — same model as terminal
- Provider switching: `--ai <provider>` flag, or command palette (`AI: kiro-cli`, `AI: claude`, `AI: codex`, `AI: shell`)
- First open with no provider → shows picker via command palette
- `aiCommand()` maps provider name to CLI command string

## Editor Pane Auto-Close
- `editor.CloseTabMsg` emitted when last tab in a pane is closed (Ctrl+W or middle-click)
- App catches it → calls `panes.ClosePane()` → auto-closes empty pane

## Key Implementation Details

### app/app.go
- `recalcLayout()` uses dynamic `m.sidebarW`, `m.aiPanelW`, `m.terminalRatio`
- Single pane: app draws border, `recalcPanes` passes offset directly (no +1)
- Multi pane: PaneManager draws borders, `recalcPanes` adds +1 for border
- `handleMouseFocus(x, y)` — sets focus by click position, handles terminal Y detection
- `startDrag(x, y)` — right-click finds closest divider
- `handleDrag(x, y)` — updates panel sizes during drag
- `toggleTerminal()` / `toggleAI()` return `tea.Cmd` for tick loop startup

### editor/view.go
- `CloseTabMsg` emitted when `CloseTab()` returns false (no tabs remaining)
- `screenToBuffer(sx, sy)` accounts for offset, gutter, tab bar, scroll
- Char-by-char rendering: syntax → selection bg → search match → bracket match → cursor reverse

### editor/panes.go
- 5 layouts: Single, Columns, Rows, LeftRight2, Grid
- `buffers map[string]*Buffer` for cross-pane sharing
- `FocusPaneAtScreen(sx, sy)` for mouse click routing

### terminal/terminal.go
- `captureHistory()` called on each tick when not scrolled — reads top row, compares to previous
- History lines rendered dimmed (#888888) when scrolled back
- Yellow `↑ N lines` indicator on last line when scrolled

## Bugs Fixed (patterns to remember)
- `"esc"` not `"escape"` — Bubble Tea v2 uses short names
- Mouse offset double-counting — single pane: app accounts for border, panes must NOT add +1
- Search overlay key priority — dispatch to editor BEFORE app keybinds when search active
- OSC 52 clipboard — doesn't work in Konsole; use system clipboard tools
- `lipgloss.Color()` is a function returning `color.Color`, not a type
- `vt10x.Parse()` reads one batch — must loop
- Ctrl+` and Ctrl+Shift+P don't work in Konsole — F3/F4 alternatives added
- Terminal tick loop: `Start()` returns `tea.Cmd` that MUST be propagated (not discarded)
- `sync.Mutex` in struct causes vet errors with Bubble Tea value copies — use pointer indirection or `atomic.Bool`
- Forwarding ALL messages to unfocused terminal kills performance — only forward `TickMsg`
- Arrow keys sent to PTY for scroll just types AAABBB — use own scrollback buffer instead

## What's Done (Phases 0-6, 9-10 complete)
See SPEC.md for detailed checklist. See KEYBINDS.md for full keybinding reference.

## What's Next
1. **Phase 7-8 — LSP**: JSON-RPC 2.0 client, server lifecycle, document sync, diagnostics, autocomplete, hover, go-to-def, rename, code actions. Consider `go.lsp.dev/protocol` for types.
2. **Phase 11 — Config**: TOML loading, theme support, keybinding overrides.
3. **Deferred**: drag tab to pane edge (drop zones), mouse-drag pane divider resize (editor panes).

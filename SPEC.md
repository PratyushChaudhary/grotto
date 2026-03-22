# grotto — Project Specification

A beautiful, terminal-native code editor built for the AI CLI era.

## Vision

AI CLIs (kiro-cli, claude code, codex) have made the terminal a real development environment. But there's no terminal-native editor that looks and feels modern. grotto is that editor — beautiful, mouse-driven, configurable, with a built-in AI panel that embeds any AI CLI right next to your code.

Not trying to reimplement VS Code. Trying to be the best possible *terminal* code editor — one that embraces the constraints and makes them a feature.

---

## Tech Stack

| Layer | Library | Purpose |
|-------|---------|---------|
| TUI framework | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) | Elm Architecture, cell-based renderer, mouse/keyboard |
| Styling | [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) | CSS-like terminal styling, layout composition, color downsampling |
| Components | [Bubbles v2](https://github.com/charmbracelet/bubbles) | textarea, viewport, textinput, list, filepicker — reuse heavily |
| Mouse zones | [BubbleZone v2](https://github.com/lrstanley/bubblezone) | Zero-width markers for click detection on UI regions |
| Layout | [BubbleLayout](https://github.com/winder/bubblelayout) | Declarative grid/dock layout manager (MiG Layout inspired) |
| Syntax | [Chroma](https://github.com/alecthomas/chroma) | Pure Go syntax highlighter, 200+ languages |
| LSP protocol | [go.lsp.dev/protocol](https://github.com/go-language-server/protocol) | LSP message types (auto-generated from spec) |
| LSP transport | [go.lsp.dev/jsonrpc2](https://github.com/go-language-server/jsonrpc2) | JSON-RPC 2.0 over stdio for LSP communication |
| Language | Go 1.22+ | |

---

## Layout

```
┌──────────────────────────────────────────────────────────────────────┐
│ ● grotto — ~/projects/myapp                              ─ □ ✕     │  <- Title Bar
├────────┬─────────────────────────────────────────┬───────────────────┤
│        │  main.go  │ utils.go  │ config.go │ + │ │                   │  <- Tab Bar
│  FILE  ├─────────────────────────────────────────┤   AI PANEL        │
│  TREE  │  1 │ package main                       │                   │
│        │  2 │                                     │  ┌─────────────┐ │
│ ▼ src/ │  3 │ import (                            │  │ kiro-cli    │ │
│   main │  4 │     "fmt"                           │  │ claude code │ │
│   util │  5 │     "os"                            │  │ codex       │ │
│ ▼ pkg/ │  6 │ )                                   │  └──────┬──────┘ │
│   conf │  7 │                                     │         ▼        │
│   hand │  8 │ func main() {                       │  > fix the bug   │
│        │  9 │     fmt.Println("hello")            │  in handler.go   │
│ ▶ test │ 10 │ }                                   │                   │
│        │ 11 │                                     │  [AI response     │
│        ├─────────────────────────────────────────┤   streams here]   │
│        │ TERMINAL                          bash  │                   │
│        │ ~/projects/myapp $ go run .             │                   │
│        │ hello                                   │                   │
│        │ ~/projects/myapp $ █                    │                   │
├────────┴─────────────────────────────────────────┴───────────────────┤
│ INSERT │ main.go │ UTF-8 │ Go │ Ln 8, Col 5          │ 2 warnings   │  <- Status Bar
└──────────────────────────────────────────────────────────────────────┘
```

Layout managed by BubbleLayout with dock constraints:
- **File Tree** (left): collapsible, resizable, default ~20 cols
- **Editor** (center top): takes remaining space, tabs + buffer + gutter
- **Terminal** (center bottom): collapsible, resizable height, hosts shell subprocess
- **AI Panel** (right): collapsible, resizable, default ~35 cols, hosts AI CLI subprocess
- **Title Bar** (dock north): project name, window controls
- **Status Bar** (dock south): mode, file info, cursor pos, diagnostics

All panels are toggleable. Keybinds to show/hide each. Mouse-draggable dividers between panels (stretch goal, start with fixed ratios).

The terminal panel sits below the editor in the center column — exactly like VS Code. The AI panel spans the full height on the right. This means you can have code, a shell, and an AI CLI all visible at once.

### Split Panes (max 4 editors)

The editor area supports splitting into up to 4 panes, just like VS Code:

```
Single (default)          Vertical split (2)        2x2 grid (4, max)
┌────────────────────┐    ┌─────────┬──────────┐    ┌─────────┬──────────┐
│                    │    │         │          │    │         │          │
│     Editor 1       │    │ Editor 1│ Editor 2 │    │ Edit 1  │ Edit 2   │
│                    │    │         │          │    │         │          │
│                    │    │         │          │    ├─────────┼──────────┤
│                    │    │         │          │    │         │          │
└────────────────────┘    └─────────┴──────────┘    │ Edit 3  │ Edit 4   │
                                                    │         │          │
Horizontal split (2)      3-pane (1 left, 2 right)  └─────────┴──────────┘
┌────────────────────┐    ┌─────────┬──────────┐
│     Editor 1       │    │         │ Editor 2 │
│                    │    │         ├──────────┤
├────────────────────┤    │ Editor 1│          │
│     Editor 2       │    │         │ Editor 3 │
│                    │    │         │          │
└────────────────────┘    └─────────┴──────────┘
```

**How splitting works:**
- Ctrl+\ to split right (vertical split)
- Ctrl+Shift+\ to split down (horizontal split)
- Mouse: drag a tab to the edge of the editor area — drop zones appear (left/right/top/bottom half) just like VS Code
- Each pane has its own tab bar and can show any open buffer
- Same file can be open in multiple panes (shared buffer, independent scroll/cursor)
- Active pane has a highlighted border
- Click anywhere in a pane to focus it
- Ctrl+1/2/3/4 to focus pane by number
- Close pane: close all tabs in it, or Ctrl+Shift+W
- Max 4 panes enforced — split commands are no-ops when at limit
- Panes resize proportionally on terminal resize
- Drag the divider between panes to resize (mouse)

---

## Core Features (VS Code Parity Targets)

### File Explorer (Sidebar)
- Recursive directory tree with expand/collapse (▶/▼ icons)
- Mouse click to open file in editor
- Mouse click to expand/collapse directories
- File/folder icons (using nerd font glyphs if available, fallback to ascii)
- Right-click context menu: new file, new folder, rename, delete
- Respects `.gitignore` for hiding files
- Current file highlighted in tree
- Scroll with mouse wheel

### Tab Bar
- One tab per open buffer, clickable to switch
- Mouse middle-click to close tab
- Close button (×) on each tab, clickable
- Dirty indicator (● dot) on unsaved tabs
- Tab overflow: scroll arrows or truncation when too many tabs
- Drag to reorder (stretch goal)

### Editor Buffer
- Syntax highlighting via Chroma (auto-detect from file extension)
- Line numbers in gutter
- Current line highlight
- Cursor blinking
- Soft word wrap (toggleable)
- Indent guides (vertical lines at tab stops)
- Matching bracket/paren highlight
- Trailing whitespace visualization
- Minimap (stretch goal)

### Mouse Support
- Click to place cursor
- Click-drag to select text
- Double-click to select word
- Triple-click to select line
- Shift+click to extend selection
- Mouse wheel to scroll vertically
- Shift+mouse wheel to scroll horizontally
- Click on line numbers to select entire line
- Click on file tree items to open/expand
- Click on tabs to switch/close
- Click on AI panel to focus it

### Keyboard Editing
- All standard cursor movement (arrows, home/end, ctrl+arrows for word jump, pgup/pgdn)
- Shift+movement for selection
- Ctrl+A select all
- Ctrl+C / Ctrl+X / Ctrl+V (copy/cut/paste via OSC 52 clipboard)
- Ctrl+Z / Ctrl+Y undo/redo
- Ctrl+D select next occurrence (multi-cursor stretch goal)
- Tab / Shift+Tab indent/dedent (selection-aware)
- Ctrl+/ toggle line comment
- Ctrl+Shift+K delete line
- Ctrl+Enter insert line below
- Ctrl+Shift+Enter insert line above
- Alt+Up/Down move line up/down
- Alt+Shift+Up/Down duplicate line

### Search
- Ctrl+F: find in current file (incremental, highlight all matches)
- Ctrl+H: find and replace
- Ctrl+Shift+F: find in project (grep across files, results in panel)
- Regex support toggle
- Case sensitivity toggle
- Whole word toggle

### Navigation
- Ctrl+P: fuzzy file finder (quick open)
- Ctrl+G: go to line
- Ctrl+Shift+P: command palette (all commands searchable)
- Ctrl+Tab / Ctrl+Shift+Tab: cycle tabs
- Breadcrumbs showing file path (clickable segments)

### LSP (Language Server Protocol)
Full LSP client built in. This is what makes grotto a real IDE, not just a text editor.

**Supported LSP features:**

| Feature | LSP Method | UX |
|---------|-----------|-----|
| Autocomplete | `textDocument/completion` | Popup menu as you type, Tab/Enter to accept |
| Hover info | `textDocument/hover` | Mouse hover or Ctrl+K Ctrl+I shows type/docs tooltip |
| Go to definition | `textDocument/definition` | Ctrl+Click or F12 — jumps to definition (opens file if needed) |
| Go to references | `textDocument/references` | Shift+F12 — list of all references in a peek panel |
| Diagnostics | `textDocument/publishDiagnostics` | Real-time errors/warnings: red/yellow squiggles, gutter icons |
| Signature help | `textDocument/signatureHelp` | Shows function signature as you type arguments |
| Rename symbol | `textDocument/rename` | F2 — rename across all files |
| Code actions | `textDocument/codeAction` | Ctrl+. — quick fixes, refactors (lightbulb icon in gutter) |
| Document symbols | `textDocument/documentSymbol` | Ctrl+Shift+O — outline / symbol list for current file |
| Workspace symbols | `workspace/symbol` | Ctrl+T — search symbols across project |
| Formatting | `textDocument/formatting` | Ctrl+Shift+I — format document |
| Go to type def | `textDocument/typeDefinition` | Jump to type definition |
| Go to implementation | `textDocument/implementation` | Jump to interface implementation |

**How it works:**
- grotto spawns language servers as child processes over stdio (JSON-RPC 2.0)
- Uses `go.lsp.dev/protocol` for message types and `go.lsp.dev/jsonrpc2` for transport
- One language server instance per language per workspace
- grotto sends `textDocument/didOpen`, `didChange`, `didSave`, `didClose` to keep the server in sync
- Incremental document sync (`TextDocumentSyncKind.Incremental`) for performance

**Language server config** (`~/.config/grotto/servers.toml`):
```toml
[servers.go]
command = ["gopls"]
filetypes = ["go"]
root_markers = ["go.mod"]

[servers.typescript]
command = ["typescript-language-server", "--stdio"]
filetypes = ["typescript", "typescriptreact", "javascript", "javascriptreact"]
root_markers = ["tsconfig.json", "package.json"]

[servers.python]
command = ["pylsp"]
filetypes = ["python"]
root_markers = ["pyproject.toml", "setup.py"]

[servers.rust]
command = ["rust-analyzer"]
filetypes = ["rust"]
root_markers = ["Cargo.toml"]

[servers.c]
command = ["clangd"]
filetypes = ["c", "cpp", "objc"]
root_markers = ["compile_commands.json", "CMakeLists.txt"]
```

Users add any LSP server by specifying the command, filetypes, and root markers. grotto auto-starts the right server when a matching file is opened.

**Diagnostics rendering:**
- Errors: red underline + red gutter icon (●)
- Warnings: yellow underline + yellow gutter icon (▲)
- Info/hints: blue underline
- Diagnostic count shown in status bar
- Ctrl+Shift+M to open diagnostics panel (list of all errors/warnings)
- F8 / Shift+F8 to jump to next/prev diagnostic

**Autocomplete popup:**
- Triggered automatically after typing (configurable delay, default 100ms)
- Also triggered manually with Ctrl+Space
- Shows completion kind icon (function, variable, class, etc.)
- Tab/Enter to accept, Escape to dismiss
- Fuzzy filtering as you type
- Documentation preview on the side when available

### Integrated Terminal
A full terminal emulator panel below the editor — same as VS Code's integrated terminal.

**How it works:**
- Spawns user's default shell (`$SHELL`, fallback to `/bin/bash`) via PTY
- Working directory matches the project root
- Full VT100/xterm emulation — colors, cursor movement, alternate screen apps all work
- Shares the same PTY infrastructure as the AI panel (reusable `pty.go` + `vt.go`)

**Features:**
- Toggle with Ctrl+` (same as VS Code)
- Multiple terminal tabs (bash, zsh, fish — whatever the user runs)
- Click on terminal tab to switch, × to close
- "+" button to spawn new terminal instance
- Resizable height — drag the divider between editor and terminal (stretch: mouse drag, v0: config/keybind)
- Scrollback buffer (mouse wheel to scroll history, shift+pgup/pgdn)
- Copy selection from terminal output
- Ctrl+Shift+` to create new terminal
- Terminal inherits grotto's environment variables plus `GROTTO=1` so scripts can detect they're inside grotto
- Supports running TUI apps inside it (htop, lazygit, etc.) via alternate screen passthrough

**Why this matters:**
- `go run .`, `npm start`, `cargo build` — run and see output without leaving the editor
- Tail logs while editing
- Git operations
- Combined with the AI panel: ask AI to fix something → see the fix in the editor → test it in the terminal — all in one screen

### AI Panel
This is the killer feature. A dedicated panel that embeds an AI CLI as a live subprocess.

**How it works:**
- grotto spawns the selected AI CLI (`kiro-cli chat`, `claude`, or `codex`) as a child process with a PTY
- The AI panel is essentially a terminal emulator (VT100) embedded in the right panel
- User types in the AI panel, output streams back in real-time
- The AI CLI has full access to the filesystem (same working directory as grotto)
- Switching between AI providers via command palette or config

**AI CLI integration:**
| CLI | Spawn command | Notes |
|-----|--------------|-------|
| kiro-cli | `kiro-cli chat` | Interactive REPL mode |
| claude code | `claude` | Interactive mode (default) |
| codex | `codex` | Interactive mode |

**Panel features:**
- Scrollable output history
- Input line at bottom
- Can be toggled with Ctrl+` or Ctrl+Shift+A
- Resizable width (drag divider or config)
- Multiple AI sessions (tabs within AI panel, stretch goal)
- Copy text from AI output

**Why PTY embedding, not a custom integration:**
- Each AI CLI has its own TUI, auth flow, streaming behavior
- Embedding via PTY means we get 100% compatibility for free — no API wrappers needed
- If a new AI CLI comes out tomorrow, users just configure the spawn command
- The AI CLIs handle their own context, tool use, etc.

**Technical approach:**
- Use `os/exec` + `github.com/creack/pty` to spawn with a pseudo-terminal
- Read PTY output, parse ANSI sequences, render into the AI panel viewport
- Forward keyboard input from the AI panel to the PTY stdin
- Handle resize by sending SIGWINCH to the child process
- A lightweight VT100 state machine to interpret the output (or use `github.com/charmbracelet/x/vt` if available)

---

## Configuration

`~/.config/grotto/config.toml` (XDG compliant):

```toml
[editor]
theme = "dracula"
font_ligatures = false
tab_size = 4
use_spaces = true
word_wrap = false
line_numbers = true
minimap = false
cursor_blink = true

[sidebar]
visible = true
width = 22
show_hidden = false
gitignore = true

[terminal]
shell = ""                      # empty = use $SHELL, fallback /bin/bash
visible = false                 # start hidden, toggle with Ctrl+`
height = 12                     # rows
scrollback = 5000               # lines of scrollback history
env = { GROTTO = "1" }          # extra env vars injected into shell

[ai]
provider = "kiro-cli"           # "kiro-cli" | "claude" | "codex" | "custom"
command = "kiro-cli chat"       # override spawn command
visible = true
width = 35
position = "right"              # "right" | "bottom"

[ai.providers.kiro-cli]
command = "kiro-cli chat"

[ai.providers.claude]
command = "claude"

[ai.providers.codex]
command = "codex"

[ai.providers.custom]
command = ""                    # user fills in

[keybindings]
# override any default keybinding
# "ctrl+b" = "toggle_sidebar"
# "ctrl+`" = "toggle_ai_panel"

[appearance]
border_style = "rounded"        # "rounded" | "thick" | "double" | "hidden"
show_icons = true               # requires nerd font
accent_color = "#7D56F4"
```

Themes are Chroma style names. Users can also drop custom `.xml` Chroma styles into `~/.config/grotto/themes/`.

---

## Architecture

```
                    ┌──────────────┐
                    │   main.go    │
                    │  CLI parsing │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │   app.Model  │  ← Root tea.Model
                    │              │
                    │  - layout    │  ← BubbleLayout grid
                    │  - focus     │  ← which panel has focus
                    │  - config    │  ← loaded config
                    │  - lspMgr   │  ← LSP manager (server lifecycle)
                    └─┬──┬──┬──┬──┘
                      │  │  │  │
          ┌───────────┘  │  │  └───────────┐
          ▼              │  │              ▼
   ┌────────────┐       │  │       ┌────────────┐
   │  Sidebar   │       │  │       │  AI Panel  │
   │  Model     │       │  │       │  Model     │
   │            │       │  │       │            │
   │ - tree     │       │  │       │ - pty ─────┼──┐
   │ - cwd      │       │  │       │ - provider │  │
   │ - expanded │       │  │       └────────────┘  │
   └────────────┘       │  │                       │  shared
                        ▼  ▼                       │  pty pkg
              ┌────────────┐ ┌────────────┐        │
              │   Editor   │ │  Terminal  │        │
              │   Model    │ │  Model     │        │
              │            │ │            │        │
              │ - panes[]  │ │ - tabs[]   │        │
              │   (1-4)    │ │ - active   │        │
              │ - active   │ │ - pty ─────┼────────┘
              │ - search   │ └────────────┘
              └─────┬──────┘
                    │
             ┌──────▼──────┐
             │    Pane     │  ← 1 to 4 panes in a split layout
             │             │
             │ - tabs[]    │  ← each pane has its own tab bar
             │ - active    │  ← active buffer in this pane
             │ - position  │  ← grid position (row, col)
             └─────┬───────┘
                   │
            ┌──────▼──────┐        ┌──────────────┐
            │   Buffer    │◄───────│  LSP Client  │
            │             │        │              │
            │ - lines[]   │ sync   │ - server proc│
            │ - cursor    │◄──────►│ - jsonrpc2   │
            │ - selection │        │ - diagnostics│
            │ - undo/redo │        │ - completions│
            │ - syntax    │        └──────────────┘
            │ - scroll    │
            │ - diagnostics│ ← pushed from LSP
            └─────────────┘
```

Each component is its own `tea.Model`. Root dispatches messages based on focus. BubbleZone handles mouse hit-testing. BubbleLayout handles sizing on terminal resize.

The `pty/` package is shared between the Terminal and AI Panel. The `lsp/` package manages language server lifecycles — one server per language per workspace, communicating over stdio JSON-RPC 2.0. Buffers receive diagnostics pushed from the LSP client and send document changes back.

---

## Project Structure

```
grotto/
├── main.go                     # entry point, arg parsing, app bootstrap
├── app/
│   ├── app.go                  # root Model, Update, View
│   ├── focus.go                # focus management between panels
│   ├── keymap.go               # default keybinding table
│   └── commands.go             # command palette registry
├── editor/
│   ├── buffer.go               # Buffer struct, text operations
│   ├── cursor.go               # cursor movement logic
│   ├── selection.go            # selection (char, line, block)
│   ├── undo.go                 # undo/redo stack
│   ├── highlight.go            # Chroma integration, token caching
│   ├── view.go                 # buffer → styled string rendering
│   └── pane.go                 # split pane manager (1-4 panes, layout, focus)
├── ui/
│   ├── tabbar.go               # tab bar component
│   ├── sidebar.go              # file tree component
│   ├── statusbar.go            # status bar component
│   ├── titlebar.go             # title bar component
│   ├── palette.go              # command palette / fuzzy finder
│   ├── search.go               # find/replace overlay
│   ├── dialog.go               # modal dialogs (save, confirm, goto line)
│   ├── breadcrumb.go           # file path breadcrumbs
│   └── autocomplete.go         # completion popup menu
├── lsp/
│   ├── client.go               # LSP client: spawn server, JSON-RPC 2.0 over stdio
│   ├── manager.go              # manages server lifecycle per language/workspace
│   ├── sync.go                 # document sync (didOpen/didChange/didSave/didClose)
│   ├── completion.go           # autocomplete request/response handling
│   ├── diagnostics.go          # diagnostics collection and rendering
│   ├── hover.go                # hover info request/response
│   ├── navigation.go           # definition, references, implementation, type def
│   └── actions.go              # code actions, rename, formatting
├── pty/
│   ├── pty.go                  # PTY spawn/read/write (shared by terminal + AI)
│   └── vt.go                   # VT100/xterm output parser for rendering
├── terminal/
│   └── terminal.go             # Integrated terminal panel Model (shell host, tabs)
├── ai/
│   └── panel.go                # AI panel Model (wraps pty for AI CLI subprocess)
├── config/
│   ├── config.go               # TOML config loading
│   ├── servers.go              # LSP server config loading
│   ├── theme.go                # Chroma theme → Lip Gloss style mapping
│   └── keymap.go               # user keybinding overrides
├── internal/
│   ├── clipboard.go            # OSC 52 clipboard
│   ├── icons.go                # file/folder icon mapping (nerd font)
│   └── fuzzy.go                # fuzzy matching for palette/file finder
├── go.mod
├── go.sum
├── README.md
├── SPEC.md
└── LICENSE
```

---

## Phased Build Plan

### Phase 0 — Scaffold & Layout (week 1)
- [x] `go mod init`, pull in all dependencies
- [x] Root app model with three columns + title bar + status bar
- [x] Empty placeholder panels that render borders and labels
- [x] Mouse enabled, alt-screen, BubbleZone wired up
- [x] Toggle sidebar (Ctrl+B), terminal (Ctrl+`), and AI panel (Ctrl+Shift+A)
- [x] Terminal resize handling
- [x] Focus switching: Ctrl+1/2/3/4, mouse click on panel, Escape to sidebar

### Phase 1 — File Tree & Opening Files (week 2)
- [x] Recursive directory tree model
- [x] Expand/collapse with mouse click and arrow keys
- [x] Click file → opens in editor
- [x] `.gitignore` filtering
- [x] File icons (📁/📄 unicode)
- [x] Mouse wheel scroll in sidebar

### Phase 2 — Editor Core (week 3-4)
- [x] Buffer type: `[]string` line storage, cursor, basic insert/delete
- [x] Render buffer with line numbers in viewport
- [x] Keyboard cursor movement (arrows, home/end, ctrl+arrows word jump, pgup/pgdn)
- [x] Text input: typing, backspace, delete, enter (auto-indent)
- [x] Syntax highlighting via Chroma (cached per line, invalidate on edit)
- [x] Current line highlight
- [x] Bracket matching
- [x] Mouse click to place cursor
- [x] Mouse wheel scroll in editor

### Phase 3 — Selection & Clipboard (week 5)
- [x] Shift+arrows keyboard selection
- [x] Mouse click-drag selection
- [x] Double-click word select, triple-click line select
- [x] Shift+click extend selection
- [x] Copy/cut/paste via OSC 52
- [x] Selection-aware Tab/Shift+Tab indent

### Phase 4 — Tabs & Multi-file (week 6)
- [x] Tab bar: open tabs, click to switch, × to close
- [x] Dirty indicator on tabs
- [x] Ctrl+W close tab (with unsaved prompt)
- [x] Ctrl+Tab / Ctrl+Shift+Tab cycle
- [x] Middle-click close

### Phase 5 — Split Panes (week 7)
- [x] Pane model: 1-4 panes, each with own tab bar + active buffer
- [x] Ctrl+\ split right, Ctrl+Shift+\ split down
- [x] Click to focus pane, Ctrl+1/2/3/4 to focus by number
- [x] Highlighted border on active pane
- [x] Same buffer in multiple panes (shared buffer, independent scroll/cursor)
- [x] Close pane with Ctrl+Shift+W
- [ ] Drag tab to pane edge to move it (drop zones)
- [x] Proportional resize on terminal resize
- [ ] Mouse-drag divider to resize panes

### Phase 6 — Search & Navigation (week 8)
- [x] Ctrl+F find overlay (incremental highlight)
- [x] Ctrl+H find and replace
- [x] Ctrl+P fuzzy file finder
- [x] Ctrl+G go to line
- [x] Ctrl+Shift+P command palette

### Phase 7 — LSP Core (week 9-10)
- [ ] LSP client: spawn server process, JSON-RPC 2.0 over stdio
- [ ] Server lifecycle manager (start/stop per language, per workspace)
- [ ] Document sync: didOpen, didChange (incremental), didSave, didClose
- [ ] Diagnostics: receive publishDiagnostics, render underlines + gutter icons
- [ ] Diagnostic count in status bar, F8/Shift+F8 to jump between
- [ ] Autocomplete: trigger on type + Ctrl+Space, popup menu, Tab/Enter accept
- [ ] Hover: Ctrl+K Ctrl+I or mouse hover shows tooltip
- [ ] Go to definition: Ctrl+Click / F12
- [ ] Go to references: Shift+F12 (peek panel)
- [ ] Signature help: show function signature while typing args
- [ ] `servers.toml` config loading, auto-start matching server on file open

### Phase 8 — LSP Advanced (week 11)
- [ ] Rename symbol: F2
- [ ] Code actions: Ctrl+. (quick fixes, refactors)
- [ ] Document symbols: Ctrl+Shift+O
- [ ] Workspace symbols: Ctrl+T
- [ ] Format document: Ctrl+Shift+I
- [ ] Go to type definition, go to implementation

### Phase 9 — PTY & Integrated Terminal (week 12)
- [x] Shared `pty/` package: PTY spawning with `creack/pty`
- [x] VT100/xterm parser for rendering subprocess output
- [x] Input forwarding to PTY
- [x] Resize handling (SIGWINCH)
- [x] Terminal panel: spawn user's `$SHELL`, render below editor
- [x] Toggle with Ctrl+`
- [x] Multiple terminal tabs (Ctrl+Shift+` to create new)
- [x] Scrollback buffer, mouse wheel scroll
- [ ] Alternate screen passthrough (so htop/lazygit/etc. work inside it)

### Phase 10 — AI Panel (week 13)
- [x] AI panel reuses `pty/` package, spawns configured AI CLI
- [x] Provider switching via config / command palette
- [x] Toggle with Ctrl+Shift+A (separate from terminal toggle)
- [x] Scrollable output, input line at bottom

### Phase 11 — Polish & Config (week 14)
- [ ] TOML config loading
- [ ] Theme support (Chroma styles)
- [ ] Custom keybinding overrides
- [ ] Undo/redo
- [ ] Line move/duplicate
- [ ] Toggle comment
- [ ] Status bar: cursor pos, language, encoding, diagnostics count
- [ ] Save (Ctrl+S), save-all, quit with unsaved prompt

### Phase 12 — Advanced (stretch)
- [ ] Ctrl+Shift+F project-wide search
- [ ] Multi-cursor
- [ ] Column/block selection
- [ ] Minimap
- [ ] Draggable panel dividers
- [ ] Multiple AI sessions (tabs in AI panel)
- [ ] Git gutter (diff markers)
- [ ] Breadcrumbs

---

## Key Design Decisions

1. **PTY embedding for AI CLIs** — not custom API integration. This gives us instant compatibility with any CLI tool. The AI CLIs handle their own auth, streaming, tool use, context. We just give them a terminal to live in.

2. **BubbleLayout for panel management** — declarative grid with dock constraints. Avoids hand-rolling resize math. Title bar and status bar dock north/south, three content columns fill the middle.

3. **Chroma for syntax highlighting** — pure Go, no CGo, 200+ languages, trivial integration. Tree-sitter can be layered on later for structural features (folding, smart select).

4. **`[]string` line storage first** — simple, fast enough for files under 50k lines. Piece table or rope can be swapped in behind a `Buffer` interface later without touching the rest of the codebase.

5. **BubbleZone for all clickable UI chrome** — tabs, sidebar items, buttons, status bar segments. Raw coordinate math only needed for the editor buffer area (translating screen position to buffer position accounting for scroll + gutter).

6. **VS Code keybindings as defaults** — familiar to most developers. Fully overridable via config. No vim mode in v0 (can be added as opt-in later).

7. **Config over convention** — everything visual is configurable. Border styles, colors, panel widths, icon sets, keybindings. Ship beautiful defaults but let people make it theirs.

8. **Max 4 split panes** — keeps the UI usable in a terminal (you're already width-constrained). Each pane is a full editor with its own tab bar. The pane layout is a simple grid (1, 1x2, 2x1, 1+2, 2x2) — no arbitrary nesting. This keeps the implementation tractable and the UX clean.

9. **LSP via `go.lsp.dev` types + raw JSON-RPC 2.0** — we use the official Go LSP protocol types for correctness, but own the client implementation. No heavy framework. grotto spawns language servers as child processes over stdio, same as every other editor. Server config is user-editable TOML — add any LSP server by specifying the command and filetypes.

10. **Shared buffer model for split panes** — when the same file is open in two panes, they share one `Buffer` (single source of truth for text, undo history, LSP state) but each pane has independent scroll position and cursor. Edits in one pane are immediately visible in the other.

---

## CLI Interface

```
grotto                          # open current directory
grotto .                        # same
grotto ~/projects/myapp         # open specific directory
grotto main.go                  # open specific file
grotto main.go:42               # open file at line 42
grotto --theme dracula          # override theme
grotto --no-ai                  # start without AI panel
grotto --ai claude              # start with specific AI provider
grotto --version
grotto --help
```

---

## Performance Targets

- Startup to interactive: < 200ms
- Open 10k line file: < 100ms
- Keystroke to render: < 16ms (60fps)
- Smooth scroll with syntax highlighting at 60fps
- Memory: < 3x file size for buffer + syntax cache + undo history

---

## Open Questions

- [x] ~~Split panes within editor area?~~ → Yes, max 4 panes, mouse-driven splitting
- [x] ~~LSP support?~~ → Yes, full LSP client with autocomplete, diagnostics, navigation, rename, code actions
- [ ] Vim mode: opt-in from v0 or add later?
- [ ] Plugin system: probably not for v1, but what would the interface look like?
- [ ] Image preview in terminal? (sixel/kitty protocol for markdown preview)
- [ ] Session persistence: reopen last workspace on launch?
- [ ] DAP (Debug Adapter Protocol) support for integrated debugging?
- [ ] Git integration beyond gutter markers? (branch switcher, commit, diff view)

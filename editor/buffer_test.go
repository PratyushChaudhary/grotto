package editor

import (
	"testing"
)

// test insertion
func TestBuffer_Insert(t *testing.T) {
	tests := []struct {
		name string
		initial string // content to seed via NewBufferFromString
		pos Position // insertion point
		text string // text to insert
		wantLines []string // expected buffer lines after insert
	}{
		// 1. single-line insertions (len(parts) == 1)
		{
			name:      "single char mid-line",
			initial:   "helo world",
			pos:       Position{Line: 0, Col: 3},
			text:      "l",
			wantLines: []string{"hello world"},
		},
		{
			name:      "string at beginning of line (Col=0 boundary)",
			initial:   "world",
			pos:       Position{Line: 0, Col: 0},
			text:      "hello ",
			wantLines: []string{"hello world"},
		},
		{
			name:      "string at end of line (Col=len boundary)",
			initial:   "hello",
			pos:       Position{Line: 0, Col: 5},
			text:      " world",
			wantLines: []string{"hello world"},
		},
		{
			name:      "insert into middle line of multi-line buffer",
			initial:   "aaa\nbbb\nccc",
			pos:       Position{Line: 1, Col: 1},
			text:      "XX",
			wantLines: []string{"aaa", "bXXbb", "ccc"},
		},
		{
			name:      "insert at end of last line in multi-line buffer",
			initial:   "line1\nline2",
			pos:       Position{Line: 1, Col: 5},
			text:      "!",
			wantLines: []string{"line1", "line2!"},
		},
		{
			name:      "insert tab character preserves special whitespace",
			initial:   "func()",
			pos:       Position{Line: 0, Col: 0},
			text:      "\t",
			wantLines: []string{"\tfunc()"},
		},

		// 2. newline insertions (line splitting)
		{
			name:      "newline splits line into two",
			initial:   "helloworld",
			pos:       Position{Line: 0, Col: 5},
			text:      "\n",
			wantLines: []string{"hello", "world"},
		},
		{
			name:      "newline at beginning of line prepends empty line",
			initial:   "hello",
			pos:       Position{Line: 0, Col: 0},
			text:      "\n",
			wantLines: []string{"", "hello"},
		},
		{
			name:      "newline at end of line appends empty line",
			initial:   "hello",
			pos:       Position{Line: 0, Col: 5},
			text:      "\n",
			wantLines: []string{"hello", ""},
		},
		{
			name:      "newline into empty buffer creates two empty lines",
			initial:   "",
			pos:       Position{Line: 0, Col: 0},
			text:      "\n",
			wantLines: []string{"", ""},
		},
		{
			name:      "newline between lines in multi-line buffer",
			initial:   "aaa\nbbbccc",
			pos:       Position{Line: 1, Col: 3},
			text:      "\n",
			wantLines: []string{"aaa", "bbb", "ccc"},
		},

		// 3. multi-line paste scenarios
		{
			name:      "three-line paste into single-line buffer",
			initial:   "ac",
			pos:       Position{Line: 0, Col: 1},
			text:      "\nb\n",
			wantLines: []string{"a", "b", "c"},
		},
		{
			name:      "four-line paste into middle of multi-line buffer",
			initial:   "first\nlast",
			pos:       Position{Line: 0, Col: 5},
			text:      "\nsecond\nthird\n",
			wantLines: []string{"first", "second", "third", "", "last"},
		},
		{
			name:      "consecutive newlines create empty interior lines",
			initial:   "ab",
			pos:       Position{Line: 0, Col: 1},
			text:      "\n\n\n",
			wantLines: []string{"a", "", "", "b"},
		},
		{
			name:      "multi-line paste not ending with newline",
			initial:   "hello",
			pos:       Position{Line: 0, Col: 5},
			text:      "\nworld\n!",
			wantLines: []string{"hello", "world", "!"},
		},
		{
			name:      "large paste with mixed content and empty lines",
			initial:   "start\nend",
			pos:       Position{Line: 0, Col: 5},
			text:      "\nalpha\n\nbeta\n",
			wantLines: []string{"start", "alpha", "", "beta", "", "end"},
		},

		// edge cases
		{
			name:      "insert text into empty buffer",
			initial:   "",
			pos:       Position{Line: 0, Col: 0},
			text:      "hello",
			wantLines: []string{"hello"},
		},
		{
			name:      "multi-line insert into empty buffer",
			initial:   "",
			pos:       Position{Line: 0, Col: 0},
			text:      "aaa\nbbb\nccc",
			wantLines: []string{"aaa", "bbb", "ccc"},
		},
		// NOTE: empty string insert is a content no-op but Insert does not
		// guard against it. Dirty, EditVersion, and undoStack are still
		// mutated. This is documented as a known behavior — see TESTING.md.
		{
			name:      "empty string insert is content no-op (metadata still mutated)",
			initial:   "hello",
			pos:       Position{Line: 0, Col: 3},
			text:      "",
			wantLines: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBufferFromString(tt.initial)

			initialVersion := buf.EditVersion

			// fresh buffer must be clean.
			if buf.Dirty {
				t.Fatal("precondition failed: NewBufferFromString produced a dirty buffer")
			}

			buf.Insert(tt.pos, tt.text)

			// assert line content
			if len(buf.Lines) != len(tt.wantLines) {
				t.Fatalf("line count: got %d, want %d\n  got:  %q\n  want: %q", len(buf.Lines), len(tt.wantLines), buf.Lines, tt.wantLines)
			}
			for i, want := range tt.wantLines {
				if buf.Lines[i] != want {
					t.Errorf("line[%d]: got %q, want %q", i, buf.Lines[i], want)
				}
			}

			// assert mutation metadata invariants 
			if !buf.Dirty {
				t.Error("Dirty should be true after Insert")
			}
			if buf.EditVersion != initialVersion+1 {
				t.Errorf("EditVersion: got %d, want %d", buf.EditVersion, initialVersion+1)
			}
			if len(buf.undoStack) != 1 {
				t.Errorf("undo stack length: got %d, want 1", len(buf.undoStack))
			}

			// assert undo entry records correct data 
			if len(buf.undoStack) == 1 {
				entry := buf.undoStack[0]
				if entry.Type != EditInsert {
					t.Errorf("undo entry type: got %d, want EditInsert(%d)", entry.Type, EditInsert)
				}
				if entry.Text != tt.text {
					t.Errorf("undo entry text: got %q, want %q", entry.Text, tt.text)
				}
				if entry.Pos != tt.pos {
					t.Errorf("undo entry pos: got %+v, want %+v", entry.Pos, tt.pos)
				}
			}
		})
	}
}

// TestBuffer_Insert_Sequential verifies that multiple successive Insert
// calls correctly accumulate state: undo stack depth, edit version,
// content integrity, and redo stack forking behavior.
//
// These tests serve as regression guards for the undo/redo history.
func TestBuffer_Insert_Sequential(t *testing.T) {

	t.Run("character-by-character buildup", func(t *testing.T) {
		buf := NewBufferFromString("")

		word := "hello"
		for i, ch := range word {
			buf.Insert(Position{Line: 0, Col: i}, string(ch))
		}

		if buf.Lines[0] != "hello" {
			t.Errorf("content: got %q, want %q", buf.Lines[0], "hello")
		}
		if buf.EditVersion != len(word) {
			t.Errorf("EditVersion: got %d, want %d",
				buf.EditVersion, len(word))
		}
		if len(buf.undoStack) != len(word) {
			t.Errorf("undo stack depth: got %d, want %d",
				len(buf.undoStack), len(word))
		}
	})

	t.Run("multi-line buildup across lines", func(t *testing.T) {
		buf := NewBufferFromString("")

		buf.Insert(Position{Line: 0, Col: 0}, "line1")
		buf.Insert(Position{Line: 0, Col: 5}, "\nline2")
		buf.Insert(Position{Line: 1, Col: 5}, "\nline3")

		wantLines := []string{"line1", "line2", "line3"}
		if len(buf.Lines) != len(wantLines) {
			t.Fatalf("line count: got %d, want %d\n  got: %q",
				len(buf.Lines), len(wantLines), buf.Lines)
		}
		for i, want := range wantLines {
			if buf.Lines[i] != want {
				t.Errorf("line[%d]: got %q, want %q", i, buf.Lines[i], want)
			}
		}
		if buf.EditVersion != 3 {
			t.Errorf("EditVersion: got %d, want 3", buf.EditVersion)
		}
		if len(buf.undoStack) != 3 {
			t.Errorf("undo stack depth: got %d, want 3", len(buf.undoStack))
		}
	})

	t.Run("undo entries record correct positions and text in order", func(t *testing.T) {
		buf := NewBufferFromString("hello")

		buf.Insert(Position{Line: 0, Col: 5}, " world")
		buf.Insert(Position{Line: 0, Col: 11}, "!")

		if len(buf.undoStack) != 2 {
			t.Fatalf("undo stack depth: got %d, want 2", len(buf.undoStack))
		}

		// stack is ordered: [0] = first insert, [1] = second insert
		wantEntries := []struct {
			text string
			pos  Position
		}{
			{text: " world", pos: Position{Line: 0, Col: 5}},
			{text: "!", pos: Position{Line: 0, Col: 11}},
		}

		for i, want := range wantEntries {
			got := buf.undoStack[i]
			if got.Text != want.text {
				t.Errorf("undoStack[%d].Text: got %q, want %q", i, got.Text, want.text)
			}
			if got.Pos != want.pos {
				t.Errorf("undoStack[%d].Pos: got %+v, want %+v", i, got.Pos, want.pos)
			}
			if got.Type != EditInsert {
				t.Errorf("undoStack[%d].Type: got %d, want EditInsert(%d)", i, got.Type, EditInsert)
			}
		}
	})

	t.Run("new insert after undo clears redo stack (history fork)", func(t *testing.T) {
		buf := NewBufferFromString("hello")

		// create history: "hello" -> "hello world"
		buf.Insert(Position{Line: 0, Col: 5}, " world")

		// Undo: "hello world" -> "hello", redo stack has 1 entry
		buf.Undo()
		if len(buf.redoStack) != 1 {
			t.Fatalf("redo stack after undo: got %d, want 1", len(buf.redoStack))
		}
		if buf.Lines[0] != "hello" {
			t.Fatalf("content after undo: got %q, want %q", buf.Lines[0], "hello")
		}

		// new insert forks history — redo becomes unreachable
		buf.Insert(Position{Line: 0, Col: 5}, "!")

		if len(buf.redoStack) != 0 {
			t.Errorf("redo stack after fork: got %d, want 0 "+"(new insert must destroy redo history)", len(buf.redoStack))
		}
		if buf.Lines[0] != "hello!" {
			t.Errorf("content after fork insert: got %q, want %q", buf.Lines[0], "hello!")
		}

		// Undo stack should have: [original insert's undo from before
		// the Undo() call was replayed] 
		// trace:
		// 1. Insert(" world")   undoStack: [Insert(" world")]
		// 2. Undo()             undoStack: [], redoStack: [Insert(" world")]
		// 3. Insert("!")        pushUndo clears redoStack
		//                       undoStack: [Insert("!")], redoStack: []
		//
		// so undo stack should have exactly 1 entry: the fork insert.
		if len(buf.undoStack) != 1 {
			t.Errorf("undo stack after fork: got %d, want 1", len(buf.undoStack))
		}
		if buf.undoStack[0].Text != "!" {
			t.Errorf("undo stack entry text: got %q, want %q", buf.undoStack[0].Text, "!")
		}
	})
}

// test deletion
func TestBuffer_Delete(t *testing.T) {
	tests := []struct {
		name      string
		initial   string
		start     Position
		end       Position
		wantLines []string
		wantDirty bool
		wantDelta int // expected EditVersion increment (0 for no-op, 1 otherwise)
	}{
		// 1. same-line deletions
		{
			name:      "delete single char mid-line",
			initial:   "hello",
			start:     Position{Line: 0, Col: 1},
			end:       Position{Line: 0, Col: 2},
			wantLines: []string{"hllo"},
			wantDirty: true,
			wantDelta: 1,
		},
		{
			name:      "delete first char of line",
			initial:   "hello",
			start:     Position{Line: 0, Col: 0},
			end:       Position{Line: 0, Col: 1},
			wantLines: []string{"ello"},
			wantDirty: true,
			wantDelta: 1,
		},
		{
			name:      "delete last char of line",
			initial:   "hello",
			start:     Position{Line: 0, Col: 4},
			end:       Position{Line: 0, Col: 5},
			wantLines: []string{"hell"},
			wantDirty: true,
			wantDelta: 1,
		},
		{
			name:      "delete multiple chars mid-line",
			initial:   "hello world",
			start:     Position{Line: 0, Col: 3},
			end:       Position{Line: 0, Col: 8},
			wantLines: []string{"helrld"},
			wantDirty: true,
			wantDelta: 1,
		},
		{
			name:      "delete entire line content (leaves empty line)",
			initial:   "hello",
			start:     Position{Line: 0, Col: 0},
			end:       Position{Line: 0, Col: 5},
			wantLines: []string{""},
			wantDirty: true,
			wantDelta: 1,
		},

		// multi-line deletions 
		{
			name:      "delete newline joining two lines",
			initial:   "hello\nworld",
			start:     Position{Line: 0, Col: 5},
			end:       Position{Line: 1, Col: 0},
			wantLines: []string{"helloworld"},
			wantDirty: true,
			wantDelta: 1,
		},
		{
			name:      "delete across two lines keeping partial content",
			initial:   "hello\nworld",
			start:     Position{Line: 0, Col: 3},
			end:       Position{Line: 1, Col: 3},
			wantLines: []string{"helld"},
			wantDirty: true,
			wantDelta: 1,
		},
		{
			name:      "delete entire middle line of three-line buffer",
			initial:   "aaa\nbbb\nccc",
			start:     Position{Line: 0, Col: 3},
			end:       Position{Line: 2, Col: 0},
			wantLines: []string{"aaaccc"},
			wantDirty: true,
			wantDelta: 1,
		},
		{
			name:      "delete all content from multi-line buffer",
			initial:   "hello\nworld",
			start:     Position{Line: 0, Col: 0},
			end:       Position{Line: 1, Col: 5},
			wantLines: []string{""},
			wantDirty: true,
			wantDelta: 1,
		},

		// normalization 
		{
			name:      "reversed arguments are auto-normalized",
			initial:   "hello",
			start:     Position{Line: 0, Col: 3},
			end:       Position{Line: 0, Col: 1},
			wantLines: []string{"hlo"},
			wantDirty: true,
			wantDelta: 1,
		},
		{
			name:      "reversed multi-line arguments are auto-normalized",
			initial:   "aaa\nbbb",
			start:     Position{Line: 1, Col: 1},
			end:       Position{Line: 0, Col: 2},
			wantLines: []string{"aabb"},
			wantDirty: true,
			wantDelta: 1,
		},

		// no-ops 
		{
			name:      "same start and end is no-op",
			initial:   "hello",
			start:     Position{Line: 0, Col: 2},
			end:       Position{Line: 0, Col: 2},
			wantLines: []string{"hello"},
			wantDirty: false,
			wantDelta: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBufferFromString(tt.initial)
			initialVersion := buf.EditVersion

			buf.Delete(tt.start, tt.end)

			// assert line content
			if len(buf.Lines) != len(tt.wantLines) {
				t.Fatalf("line count: got %d, want %d\n  got:  %q\n  want: %q",
					len(buf.Lines), len(tt.wantLines), buf.Lines, tt.wantLines)
			}
			for i, want := range tt.wantLines {
				if buf.Lines[i] != want {
					t.Errorf("line[%d]: got %q, want %q", i, buf.Lines[i], want)
				}
			}

			// assert metadata
			if buf.Dirty != tt.wantDirty {
				t.Errorf("Dirty: got %v, want %v", buf.Dirty, tt.wantDirty)
			}
			if buf.EditVersion != initialVersion+tt.wantDelta {
				t.Errorf("EditVersion: got %d, want %d",
					buf.EditVersion, initialVersion+tt.wantDelta)
			}

			// assert undo stack matches delta
			if len(buf.undoStack) != tt.wantDelta {
				t.Errorf("undo stack length: got %d, want %d",
					len(buf.undoStack), tt.wantDelta)
			}

			// assert undo entry records the deleted text
			if tt.wantDelta == 1 && len(buf.undoStack) == 1 {
				entry := buf.undoStack[0]
				if entry.Type != EditDelete {
					t.Errorf("undo entry type: got %d, want EditDelete(%d)",
						entry.Type, EditDelete)
				}
				// Reconstruct what should have been deleted by
				// comparing initial content minus result content.
				// Currently implemented simpler: re-do via Undo and check we get back initial.
			}
		})
	}
}

// Backspace Tests
// Call chain: Backspace() -> calculates delStart -> Delete(delStart, Cursor) -> Cursor = delStart -> clampCursor()
//   - deletes the character BEFORE the cursor
//   - cursor moves backward to the deletion point
//   - at (0,0): complete no-op (no state change at all)
//   - at (N,0): merges line N into line N-1, cursor lands at former end of line N-1
func TestBuffer_Backspace(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     Position // cursor BEFORE backspace
		wantLines  []string
		wantCursor Position // cursor AFTER backspace
		wantDirty  bool
		wantDelta  int
	}{
		// mid-line backspace 
		{
			name:       "delete char before cursor mid-line",
			initial:    "hello",
			cursor:     Position{Line: 0, Col: 3},
			wantLines:  []string{"helo"},
			wantCursor: Position{Line: 0, Col: 2},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "delete first char when cursor at col 1",
			initial:    "hello",
			cursor:     Position{Line: 0, Col: 1},
			wantLines:  []string{"ello"},
			wantCursor: Position{Line: 0, Col: 0},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "delete last char when cursor at end of line",
			initial:    "hello",
			cursor:     Position{Line: 0, Col: 5},
			wantLines:  []string{"hell"},
			wantCursor: Position{Line: 0, Col: 4},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "backspace removes empty line between content lines",
			initial:    "aaa\n\nccc",
			cursor:     Position{Line: 1, Col: 0},
			wantLines:  []string{"aaa", "ccc"},
			wantCursor: Position{Line: 0, Col: 3},
			wantDirty:  true,
			wantDelta:  1,
		},

		// line-boundary backspace (merges lines) 
		{
			name:       "backspace at col 0 merges with previous line",
			initial:    "hello\nworld",
			cursor:     Position{Line: 1, Col: 0},
			wantLines:  []string{"helloworld"},
			wantCursor: Position{Line: 0, Col: 5},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "backspace merges empty line into previous",
			initial:    "hello\n",
			cursor:     Position{Line: 1, Col: 0},
			wantLines:  []string{"hello"},
			wantCursor: Position{Line: 0, Col: 5},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "backspace merges into empty previous line",
			initial:    "\nhello",
			cursor:     Position{Line: 1, Col: 0},
			wantLines:  []string{"hello"},
			wantCursor: Position{Line: 0, Col: 0},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "backspace on third line merges into second",
			initial:    "aaa\nbbb\nccc",
			cursor:     Position{Line: 2, Col: 0},
			wantLines:  []string{"aaa", "bbbccc"},
			wantCursor: Position{Line: 1, Col: 3},
			wantDirty:  true,
			wantDelta:  1,
		},

		// no-op: buffer start
		{
			name:       "backspace at buffer start is no-op",
			initial:    "hello",
			cursor:     Position{Line: 0, Col: 0},
			wantLines:  []string{"hello"},
			wantCursor: Position{Line: 0, Col: 0},
			wantDirty:  false,
			wantDelta:  0,
		},
		{
			name:       "backspace at start of empty buffer is no-op",
			initial:    "",
			cursor:     Position{Line: 0, Col: 0},
			wantLines:  []string{""},
			wantCursor: Position{Line: 0, Col: 0},
			wantDirty:  false,
			wantDelta:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBufferFromString(tt.initial)
			buf.Cursor = tt.cursor
			initialVersion := buf.EditVersion

			buf.Backspace()

			// assert line content
			if len(buf.Lines) != len(tt.wantLines) {
				t.Fatalf("line count: got %d, want %d\n  got:  %q\n  want: %q", len(buf.Lines), len(tt.wantLines), buf.Lines, tt.wantLines)
			}
			for i, want := range tt.wantLines {
				if buf.Lines[i] != want {
					t.Errorf("line[%d]: got %q, want %q", i, buf.Lines[i], want)
				}
			}

			// assert cursor position 
			if buf.Cursor != tt.wantCursor {
				t.Errorf("cursor: got %+v, want %+v", buf.Cursor, tt.wantCursor)
			}

			// assert metadata 
			if buf.Dirty != tt.wantDirty {
				t.Errorf("Dirty: got %v, want %v", buf.Dirty, tt.wantDirty)
			}
			if buf.EditVersion != initialVersion+tt.wantDelta {
				t.Errorf("EditVersion: got %d, want %d", buf.EditVersion, initialVersion+tt.wantDelta)
			}
		})
	}
}

// DeleteChar Tests (forward delete)
// Call chain: DeleteChar() -> calculates delEnd -> Delete(Cursor, delEnd)
//   - Deletes the character AT the cursor (not before it)
//   - Cursor does NOT move after deletion
//   - At end of last line: complete no-op
//   - At end of non-last line: merges next line into current
func TestBuffer_DeleteChar(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		cursor     Position
		wantLines  []string
		wantCursor Position
		wantDirty  bool
		wantDelta  int
	}{
		// mid-line forward delete
		{
			name:       "delete char at cursor mid-line",
			initial:    "hello",
			cursor:     Position{Line: 0, Col: 1},
			wantLines:  []string{"hllo"},
			wantCursor: Position{Line: 0, Col: 1},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "delete first char of line",
			initial:    "hello",
			cursor:     Position{Line: 0, Col: 0},
			wantLines:  []string{"ello"},
			wantCursor: Position{Line: 0, Col: 0},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "delete last char of line",
			initial:    "hello",
			cursor:     Position{Line: 0, Col: 4},
			wantLines:  []string{"hell"},
			wantCursor: Position{Line: 0, Col: 4},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "forward delete on empty line between content lines",
			initial:    "aaa\n\nccc",
			cursor:     Position{Line: 1, Col: 0},
			wantLines:  []string{"aaa", "ccc"},
			wantCursor: Position{Line: 1, Col: 0},
			wantDirty:  true,
			wantDelta:  1,
		},
		
		// line-boundary forward delete (merges next line)
		{
			name:       "delete at end of line merges next line",
			initial:    "hello\nworld",
			cursor:     Position{Line: 0, Col: 5},
			wantLines:  []string{"helloworld"},
			wantCursor: Position{Line: 0, Col: 5},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "delete at end of line merges empty next line",
			initial:    "hello\n",
			cursor:     Position{Line: 0, Col: 5},
			wantLines:  []string{"hello"},
			wantCursor: Position{Line: 0, Col: 5},
			wantDirty:  true,
			wantDelta:  1,
		},
		{
			name:       "delete at end of empty line merges next line",
			initial:    "\nhello",
			cursor:     Position{Line: 0, Col: 0},
			wantLines:  []string{"hello"},
			wantCursor: Position{Line: 0, Col: 0},
			wantDirty:  true,
			wantDelta:  1,
		},

		// no-ops 
		{
			name:       "delete at end of last line is no-op",
			initial:    "hello",
			cursor:     Position{Line: 0, Col: 5},
			wantLines:  []string{"hello"},
			wantCursor: Position{Line: 0, Col: 5},
			wantDirty:  false,
			wantDelta:  0,
		},
		{
			name:       "delete in empty buffer is no-op",
			initial:    "",
			cursor:     Position{Line: 0, Col: 0},
			wantLines:  []string{""},
			wantCursor: Position{Line: 0, Col: 0},
			wantDirty:  false,
			wantDelta:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBufferFromString(tt.initial)
			buf.Cursor = tt.cursor
			initialVersion := buf.EditVersion

			buf.DeleteChar()

			// assert line content 
			if len(buf.Lines) != len(tt.wantLines) {
				t.Fatalf("line count: got %d, want %d\n  got:  %q\n  want: %q", len(buf.Lines), len(tt.wantLines), buf.Lines, tt.wantLines)
			}
			for i, want := range tt.wantLines {
				if buf.Lines[i] != want {
					t.Errorf("line[%d]: got %q, want %q", i, buf.Lines[i], want)
				}
			}

			// assert cursor position (must NOT move for DeleteChar)
			if buf.Cursor != tt.wantCursor {
				t.Errorf("cursor: got %+v, want %+v", buf.Cursor, tt.wantCursor)
			}

			// assert metadata 
			if buf.Dirty != tt.wantDirty {
				t.Errorf("Dirty: got %v, want %v", buf.Dirty, tt.wantDirty)
			}
			if buf.EditVersion != initialVersion+tt.wantDelta {
				t.Errorf("EditVersion: got %d, want %d", buf.EditVersion, initialVersion+tt.wantDelta)
			}
		})
	}
}

// Delete -> Undo (roundtrip: no changes at all (w.r.t. initial))
// Call chain on Undo for an EditDelete:
//   Undo() -> sees EditDelete -> insertRaw(edit.Pos, edit.Text)
//          -> Cursor = advancePos(edit.Pos, edit.Text)
func TestBuffer_Delete_UndoRestoresContent(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		start   Position
		end     Position
	}{
		{
			name:    "undo single-line deletion",
			initial: "hello world",
			start:   Position{Line: 0, Col: 5},
			end:     Position{Line: 0, Col: 11},
		},
		{
			name:    "undo multi-line deletion",
			initial: "aaa\nbbb\nccc",
			start:   Position{Line: 0, Col: 2},
			end:     Position{Line: 2, Col: 1},
		},
		{
			name:    "undo newline-only deletion (line merge)",
			initial: "hello\nworld",
			start:   Position{Line: 0, Col: 5},
			end:     Position{Line: 1, Col: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBufferFromString(tt.initial)

			// snapshot original state
			originalLines := make([]string, len(buf.Lines))
			copy(originalLines, buf.Lines)

			buf.Delete(tt.start, tt.end)

			// content should have changed
			if len(buf.Lines) == len(originalLines) {
				allSame := true
				for i := range buf.Lines {
					if buf.Lines[i] != originalLines[i] {
						allSame = false
						break
					}
				}
				if allSame {
					t.Fatal("Delete did not change buffer content")
				}
			}

			buf.Undo()

			// assert full content restoration
			if len(buf.Lines) != len(originalLines) {
				t.Fatalf("line count after undo: got %d, want %d\n  got:  %q\n  want: %q", len(buf.Lines), len(originalLines), buf.Lines, originalLines)
			}
			for i, want := range originalLines {
				if buf.Lines[i] != want {
					t.Errorf("line[%d] after undo: got %q, want %q", i, buf.Lines[i], want)
				}
			}
		})
	}
}
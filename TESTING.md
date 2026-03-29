# Testing Strategy — Grotto

## Running Tests

```bash
# Run all tests (verbose)
go test -v ./...

# Run only buffer tests
go test -v ./editor/ -run TestBuffer

# Run a specific test case
go test -v ./editor/ -run TestBuffer_Insert/newline_splits_line

# Run with race detector (recommended before merging)
go test -v -race ./...

# Disable caching (CI-style clean run)
go test -v -count=1 ./...
```

## CI Pipeline

### Workflow Responsibilities

| Workflow | File | Trigger | Owns |
|---|---|---|---|
| **Tests** | `test.yml` | push + PR to `main` | `go test -v -race -count=1 ./...` |
| **CI** | `ci.yml` | push + PR to `main` | `gofmt` check, `go vet`, `golangci-lint`, cross-platform build matrix |
| **Auto Tag** | `auto-tag.yml` | push to `main` | Semantic version bump + GoReleaser |
| **Release** | `release.yml` | version tag push | GoReleaser binary publishing |

The test job was consolidated from `ci.yml` into a dedicated `test.yml` to
give each workflow a single responsibility. `ci.yml` owns static analysis and
build verification. `test.yml` owns runtime correctness.

## What We Test Today

### `editor/buffer_test.go` — Core Text Buffer Logic

| Test Function | Cases | Exercises |
|---|---|---|
| `TestBuffer_Insert` | 19 table-driven | Single-line insert, newline split, multi-line paste, empty buffer, empty string edge case |
| `TestBuffer_Insert_Sequential` | 4 subtests | Accumulated undo stack, edit version, history fork (redo cleared after new insert) |
| `TestBuffer_Delete` | 12 table-driven | Same-line delete, multi-line delete, argument normalization, no-op guard |
| `TestBuffer_Backspace` | 10 table-driven | Backward delete with cursor movement, line merging, empty line removal, boundary no-ops |
| `TestBuffer_DeleteChar` | 9 table-driven | Forward delete without cursor movement, line merging, empty line removal, boundary no-ops |
| `TestBuffer_Delete_UndoRestoresContent` | 3 subtests | Delete → Undo roundtrip restores original content exactly |

**Total: 57 test cases across 6 test functions.**

### Design Principle: Regression-First

Every test case targets a specific branch or boundary condition in the production
code. The goal is that any single-line code mutation — off-by-one, missing
assignment, wrong variable — causes at least one test failure. This means that
when adding a new feature or refactoring existing code, the existing test suite
catches unintended side effects automatically, without requiring manual QA.

The pattern is consistent and repeatable:
1. Seed a buffer with `NewBufferFromString(content)`.
2. Set cursor/selection state if needed.
3. Call the function under test.
4. Assert content (`Lines`), cursor position, and metadata (`Dirty`, `EditVersion`, undo/redo stacks).

Adding tests for a new buffer operation means adding table rows following this
exact pattern — no test infrastructure or framework to learn.

## What We Defer: UI / Bubble Tea Rendering Tests

We intentionally **do not** test Bubble Tea `View()` output or `Update()` message
routing at this stage. The reasons are:

### 1. Flakiness from terminal rendering

Bubble Tea's `View()` output includes ANSI escape codes, Lip Gloss styling, and
BubbleZone markers. These are:
- **Width/height dependent** — output changes with terminal dimensions.
- **Style-version sensitive** — a Lip Gloss update can change padding bytes.
- **Non-deterministic in ordering** — zone IDs, cursor sequences.

Snapshot-testing rendered output creates tests that break on dependency bumps,
not on logic regressions. This is the definition of flaky.

### 2. Tight coupling to framework internals

Testing `Update()` routing requires constructing `tea.KeyPressMsg` and other
internal message types whose structure may change across Bubble Tea versions.
The test would be testing our integration with the framework, not our logic.

### 3. The real value is in the layer below

The `Update()` methods in `editor/view.go` and `app/app.go` are thin dispatchers
that call into `Buffer`, `SearchOverlay`, `PaneManager`, etc. Testing those
domain types directly gives us the same confidence with none of the fragility.

## Discovered Issues

During test development, we analyzed every code path in `editor/buffer.go` by
tracing call chains. The following issues were identified and categorized.

---

### Issue 1: Empty string Insert mutates metadata

**Classification: Latent defect — cannot be triggered from current app code
paths, but violates the API contract and will cause bugs when new callers are
added.**

`Insert(pos, "")` does not change any line content, but unconditionally:
- Sets `Dirty = true` (phantom "unsaved" indicator).
- Increments `EditVersion` (wasted git diff debounce cycle).
- Pushes a no-op `Edit` onto the undo stack (Ctrl+Z "undoes" nothing).
- Clears the redo stack via `pushUndo` (silently destroys redo history).

**Why it doesn't bite today:** All current callers — `InsertChar`, `NewLine`,
clipboard paste — always pass non-empty text.

**Why it will bite later:** Any future feature that constructs insert text
dynamically (template expansion, snippet insertion, plugin API, autocomplete
acceptance) could pass an empty string when the computed result is empty. The
user would see phantom dirty state and lose redo history with no visible cause.

**Contrast with Delete:** `Delete` already has the correct guard:
```go
if start.Line == end.Line && start.Col == end.Col {
    return  // no metadata mutation
}
```
`Insert` lacks the equivalent `if text == "" { return }`.

**Documented by test:**
`TestBuffer_Insert/empty_string_insert_is_content_no-op_(metadata_still_mutated)`

**Recommended fix:**
```go
func (b *Buffer) Insert(pos Position, text string) {
    if text == "" {
        return
    }
    // ... rest of function
}
```

---

### Issue 2: Inconsistent no-op handling between Insert and Delete

**Classification: Design inconsistency — not a bug today, but creates a
maintenance trap for future contributors.**

`Delete` correctly guards against no-op calls. `Insert` does not. A contributor
reading `Delete` would reasonably assume `Insert` follows the same pattern. This
inconsistency increases the risk of Issue #1 being introduced by a new caller
who assumes the guard exists.

**Recommended fix:** Resolve alongside Issue #1 for API symmetry.

---


---
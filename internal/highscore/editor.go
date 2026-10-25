package highscore

import "strings"

// Editor accepts the supplied alphabet and commits exactly three initials.
// Touch controls preview/cycle a character; direct keyboard text commits it.
type Editor struct {
	Candidate Candidate
	Buffer    [3]byte
	Cursor    int
	Allowed   string
	selected  int
	preview   bool
	pending   []byte
	changed   [3]bool
	row       [3]byte
}

// Step accepts text events and consumes one editing event at a frame boundary.
// Pasted text is queued so each original prompt update commits one initial.
func (e *Editor) Step(text string, backspace bool) {
	if backspace {
		e.pending = append(e.pending, 255)
	}
	for _, c := range []byte(strings.ToUpper(text)) {
		if strings.IndexByte(e.Allowed, c) >= 0 {
			e.pending = append(e.pending, c)
		}
	}
	if e.Done() {
		return
	}
	if len(e.pending) == 0 {
		if !e.preview {
			e.Buffer[e.Cursor] = ' '
		}
		return
	}
	c := e.pending[0]
	e.pending = e.pending[1:]
	if c == 255 {
		if e.Cursor == 0 {
			// The original first-cell backspace commits a space because its
			// input flag remains set on this branch.
			e.Type(" ")
		} else {
			// Rewinding preserves the previous character for this draw,
			// clears the cell after it, then leaves the cursor uncommitted.
			old := e.Cursor
			e.Cursor--
			e.Buffer[old], e.row[old], e.changed[old] = ' ', ' ', true
			e.preview = false
		}
	} else {
		e.Type(string(c))
	}
}

// Line composes the source prompt after the current editing event. The caller
// retains this result when qualification advances to another player.
func (e *Editor) Line(prompt string, playerCell, initialsCell int) string {
	line := []byte(prompt)
	line[playerCell] = byte('1' + e.Candidate.Player)
	copy(line[initialsCell:], e.Display())
	return string(line)
}

func NewEditor(candidate Candidate, allowed string) *Editor {
	e := &Editor{Candidate: candidate, Allowed: allowed, Buffer: [3]byte{' ', ' ', ' '}}
	e.selected = strings.IndexByte(allowed, 'A')
	if e.selected < 0 {
		e.selected = 0
	}
	return e
}

func (e *Editor) Type(text string) {
	for _, c := range []byte(strings.ToUpper(text)) {
		if e.Done() {
			return
		}
		if strings.IndexByte(e.Allowed, c) < 0 {
			continue
		}
		e.Buffer[e.Cursor] = c
		e.changed[e.Cursor] = true
		e.row[e.Cursor] = c
		e.Cursor++
		e.preview = false
	}
}

func (e *Editor) Backspace() {
	if e.Cursor == 0 {
		return
	}
	e.Cursor--
	e.changed[e.Cursor] = true
	for n := e.Cursor; n < len(e.Buffer); n++ {
		e.Buffer[n] = ' '
		e.row[n], e.changed[n] = ' ', true
	}
	e.preview = false
}

func (e *Editor) Cycle(delta int) {
	if e.Done() {
		return
	}
	if e.preview {
		e.selected = (e.selected + delta + len(e.Allowed)) % len(e.Allowed)
	}
	e.preview = true
}

func (e *Editor) Choose() {
	if e.Done() {
		return
	}
	if !e.preview {
		e.preview = true
		return
	}
	e.Type(string(e.Allowed[e.selected]))
}

func (e *Editor) Finish() {
	for !e.Done() {
		e.Type(" ")
	}
}
func (e *Editor) Done() bool       { return e.Cursor == len(e.Buffer) }
func (e *Editor) Initials() string { return string(e.Buffer[:]) }

// RowInitials changes only cells edited so far. The original rank insertion
// retains the destination row's old initials until each new cell is entered.
func (e *Editor) RowInitials(previous string) string {
	row := [3]byte{' ', ' ', ' '}
	copy(row[:], previous)
	for n, changed := range e.changed {
		if changed {
			row[n] = e.row[n]
		}
	}
	return string(row[:])
}

func (e *Editor) Display() string {
	text := e.Buffer
	if e.preview && !e.Done() {
		text[e.Cursor] = e.Allowed[e.selected]
	}
	return string(text[:])
}

// TouchDisplay always previews the selectable letter without committing it.
func (e *Editor) TouchDisplay() string {
	text := e.Buffer
	if !e.Done() {
		text[e.Cursor] = e.Allowed[e.selected]
	}
	return string(text[:])
}

package highscore

import "testing"

func TestStepQueuesPastedTextAcrossPromptFrames(t *testing.T) {
	e := NewEditor(Candidate{Player: 2}, " 0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	e.Step("a?bc", false)
	if e.Cursor != 1 || e.Initials() != "A  " || e.Done() {
		t.Fatalf("first frame committed too much input: cursor=%d initials=%q", e.Cursor, e.Initials())
	}
	e.Step("", false)
	if e.Cursor != 2 || e.Initials() != "AB " {
		t.Fatalf("second frame did not drain one key: cursor=%d initials=%q", e.Cursor, e.Initials())
	}
	e.Step("", false)
	if !e.Done() || e.Initials() != "ABC" {
		t.Fatalf("third frame initials=%q", e.Initials())
	}
}

func TestStepBackspaceAndRetainedCompletedPrompt(t *testing.T) {
	e := NewEditor(Candidate{Player: 1}, " ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	e.Step("A", false)
	e.Step("B", false)
	e.Step("C", true)
	if e.Cursor != 1 || e.Initials() != "AB " {
		t.Fatalf("backspace frame initials=%q cursor=%d", e.Initials(), e.Cursor)
	}
	e.Step("D", false)
	e.Step("", false)
	line := e.Line("PLAYER 1 NAME (   )", 7, 15)
	if line != "PLAYER 2 NAME (ACD)" {
		t.Fatalf("completed prompt=%q", line)
	}
	next := NewEditor(Candidate{Player: 2}, e.Allowed)
	if next.Display() != "   " || line != "PLAYER 2 NAME (ACD)" {
		t.Fatal("starting another editor altered the retained prompt")
	}
}

func TestPendingRankRetainsUntouchedInitials(t *testing.T) {
	e := NewEditor(Candidate{}, " ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	row := "TSL"
	if e.RowInitials(row) != row {
		t.Fatal("qualification cleared untouched initials")
	}
	e.Step("A", false)
	row = e.RowInitials(row)
	if row != "ASL" {
		t.Fatalf("first edited row=%q", row)
	}
	e.Step("B", false)
	row = e.RowInitials(row)
	e.Step("", true)
	if row = e.RowInitials(row); row != "AB " {
		t.Fatalf("backspace edited untouched suffix: %q", row)
	}
}

func TestFirstCellBackspaceCommitsSourceSpace(t *testing.T) {
	e := NewEditor(Candidate{}, " ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	e.Step("", true)
	if e.Cursor != 1 || e.RowInitials("OLD") != " LD" {
		t.Fatalf("first-cell backspace cursor=%d row=%q", e.Cursor, e.RowInitials("OLD"))
	}
}

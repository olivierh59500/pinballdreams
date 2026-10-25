package highscore

import "testing"

func TestQualificationKeepsTiesAndProcessesPlayersInOrder(t *testing.T) {
	book := Book{}
	book[0] = [4]Entry{{"AAA", 500}, {"BBB", 400}, {"CCC", 300}, {"DDD", 200}}
	if candidate, ok := book.Qualify(0, []uint64{200}); ok {
		t.Fatalf("last-place tie qualified: %+v", candidate)
	}
	scores := []uint64{350, 900, 400}
	for player := 0; player < len(scores); {
		candidate, ok := book.Qualify(0, scores[player:])
		if !ok {
			break
		}
		candidate.Player += player
		book.Insert(candidate, []string{"ONE", "TWO", "THR"}[candidate.Player])
		player = candidate.Player + 1
	}
	want := [4]Entry{{"TWO", 900}, {"AAA", 500}, {"BBB", 400}, {"THR", 400}}
	if book[0] != want {
		t.Fatalf("rankings %v; want %v", book[0], want)
	}
}

func TestEditorFiltersTextAndSupportsBackspaceAndTouch(t *testing.T) {
	e := NewEditor(Candidate{}, " 0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	e.Type("a?")
	if e.Cursor != 1 || e.Display() != "A  " {
		t.Fatal("input filtering or case conversion failed")
	}
	e.Type("b")
	e.Backspace()
	e.Type("cd")
	if !e.Done() || e.Initials() != "ACD" {
		t.Fatalf("edited initials=%q", e.Initials())
	}
	e = NewEditor(Candidate{}, " ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	e.Cycle(1)
	e.Choose()
	e.Cycle(1)
	e.Cycle(1)
	e.Choose()
	e.Finish()
	if e.Initials() != "AB " {
		t.Fatalf("touch initials=%q", e.Initials())
	}
}

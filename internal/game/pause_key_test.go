package game

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/internal/highscore"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

func TestDeferredPauseKeyDoesNotConsumeHeldSpringTouch(t *testing.T) {
	g := readyTouchGame(t, 0)
	contact := touch(1, 300, 510, true)
	updateContacts(t, g, session.Input{}, []pointerContact{contact}, false)
	contact.pressed = false
	before := g.Session.Charge
	input := updateContacts(t, g, session.Input{PauseKeyPressed: true}, []pointerContact{contact}, true)
	if !input.Launch || g.Session.Paused || g.Session.Charge <= before || g.controls.consumed[contact.id] {
		t.Fatal("deferred keyboard P consumed or interrupted a spring grip")
	}
}

func TestLivePauseKeyConsumesSpringGripUntilRelease(t *testing.T) {
	g := readyTouchGame(t, 0)
	for range 55 {
		updateContacts(t, g, session.Input{Launch: true}, nil, true)
	}
	updateContacts(t, g, session.Input{}, nil, false)
	for frame := 0; g.Session.DefersKeyboardPause() && frame < 300; frame++ {
		updateContacts(t, g, session.Input{}, nil, false)
	}
	if g.Session.DefersKeyboardPause() {
		t.Fatal("launch never entered live play")
	}
	contact := touch(1, 300, 510, true)
	updateContacts(t, g, session.Input{}, []pointerContact{contact}, false)
	contact.pressed = false
	updateContacts(t, g, session.Input{PauseKeyPressed: true}, []pointerContact{contact}, true)
	if !g.Session.Paused || !g.controls.consumed[contact.id] {
		t.Fatal("live P did not consume its spring grip")
	}
	updateContacts(t, g, session.Input{AnyKey: true}, []pointerContact{contact}, true)
	input := updateContacts(t, g, session.Input{}, []pointerContact{contact}, false)
	if g.Session.Paused || input.Launch {
		t.Fatal("held pause grip leaked into resumed play")
	}
}

func TestPauseKeyRemainsAnInitialsCharacter(t *testing.T) {
	g := readyTouchGame(t, 0)
	entry := highscore.NewEditor(highscore.Candidate{}, g.Session.Table.ScoreEntry.Allowed)
	g.Session.NameEntry = entry
	input := updateContacts(t, g, session.Input{PauseKeyPressed: true, Text: "P"}, nil, true)
	if input.PauseKeyPressed || g.Session.Paused || entry.Cursor != 1 || entry.Buffer[0] != 'P' {
		t.Fatal("keyboard P paused initials instead of entering the supplied letter")
	}
}

package game

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/internal/frontend"
	"github.com/olivierh59500/pinballdreams/internal/session"
)

func openingTouchGame(t *testing.T) *Game {
	t.Helper()
	g, err := New(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	g.Layout(960, 768)
	return g
}

func TestOpeningTwoFingerHoldSelectsExtendedCreditsWithoutSkipping(t *testing.T) {
	g := openingTouchGame(t)
	contacts := []pointerContact{touch(1, 200, 200, true)}
	if input := updateContacts(t, g, session.Input{}, contacts, false); input.Skip || input.IntroExtraCredits {
		t.Fatal("first finger skipped before another finger could join")
	}
	contacts[0].pressed = false
	contacts = append(contacts, touch(2, 400, 200, true))
	for frame := 0; ; frame++ {
		if frame > 4000 {
			t.Fatal("two-finger hold did not reach credits")
		}
		input := updateContacts(t, g, session.Input{}, contacts, false)
		if input.Skip || !input.IntroExtraCredits || g.Session.Intro == nil {
			t.Fatal("secondary touch hold skipped or lost the opening")
		}
		contacts[1].pressed = false
		if g.Session.Intro.State().Picture == "credits" {
			break
		}
	}
	if !g.Session.Intro.State().ExtendedCredits {
		t.Fatal("holding through the developer logo did not select extended credits")
	}
	for _, remaining := range [][]pointerContact{contacts[:1], nil} {
		if input := updateContacts(t, g, session.Input{}, remaining, false); input.Skip || g.Session.Intro == nil {
			t.Fatal("releasing a secondary touch hold also skipped the opening")
		}
	}
	if !g.Session.Intro.State().ExtendedCredits {
		t.Fatal("releasing after the logo changed the selected credit stream")
	}
}

func TestOpeningTapSkipsOnReleaseAndMouseSkipsOnPress(t *testing.T) {
	g := openingTouchGame(t)
	for range 150 {
		updateContacts(t, g, session.Input{}, nil, false)
	}
	contact := touch(1, 660, 100, true)
	if input := updateContacts(t, g, session.Input{}, []pointerContact{contact}, false); input.Skip {
		t.Fatal("touch skipped before release")
	}
	contact.pressed = false
	if input := updateContacts(t, g, session.Input{}, []pointerContact{contact}, false); input.Skip {
		t.Fatal("holding a single touch skipped the opening")
	}
	if input := updateContacts(t, g, session.Input{}, nil, false); !input.Skip || g.Session.Intro.State().Phase != frontend.Exiting {
		t.Fatal("releasing an opening tap did not start its exit fade")
	}
	g = openingTouchGame(t)
	mouse := pointerContact{id: pointerID{mouse: true}, x: 660, y: 100, pressed: true}
	if input := updateContacts(t, g, session.Input{}, []pointerContact{mouse}, false); !input.Skip {
		t.Fatal("primary mouse press lost its immediate skip action")
	}
}

func TestOpeningHeldTouchesDoNotSelectATableAfterNaturalCompletion(t *testing.T) {
	g := openingTouchGame(t)
	contacts := []pointerContact{touch(1, 80, 100, true), touch(2, 240, 100, true)}
	for frame := 0; g.Session.Intro != nil; frame++ {
		if frame > 7000 {
			t.Fatal("extended opening did not complete naturally")
		}
		updateContacts(t, g, session.Input{}, contacts, false)
		contacts[0].pressed, contacts[1].pressed = false, false
	}
	g.Layout(960, 768)
	for range 40 {
		updateContacts(t, g, session.Input{}, contacts, false)
	}
	if g.Session.Table != nil || g.Session.Menu == nil || !g.Session.Menu.Ready() {
		t.Fatal("opening contacts selected a table instead of leaving the menu ready")
	}
	if input := updateContacts(t, g, session.Input{}, nil, false); input.Skip || input.Table != 0 {
		t.Fatal("releasing the opening gesture leaked input into selection")
	}
}

func TestOpeningSuspendCancelsPendingTapAndInheritedSecondaryHold(t *testing.T) {
	g := openingTouchGame(t)
	contact := touch(1, 200, 200, true)
	updateContacts(t, g, session.Input{}, []pointerContact{contact}, false)
	g.Suspend()
	contact.pressed = false
	if input := updateContacts(t, g, session.Input{IntroExtraCredits: true}, []pointerContact{contact}, true); input != (session.Input{}) {
		t.Fatal("suspension accepted inherited opening controls")
	}
	updateContacts(t, g, session.Input{}, nil, false)
	resume := touch(2, 200, 200, true)
	input := updateContacts(t, g, session.Input{}, []pointerContact{resume}, false)
	if !input.Pause || input.Skip || g.Session.Paused {
		t.Fatal("fresh touch did not exclusively resume the opening")
	}
	resume.pressed = false
	updateContacts(t, g, session.Input{}, []pointerContact{resume}, false)
	if input := updateContacts(t, g, session.Input{}, nil, false); input.Skip || g.Session.Intro == nil {
		t.Fatal("releasing the resume touch skipped the opening")
	}
}

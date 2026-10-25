package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/highscore"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

func readyTouchGame(t *testing.T, table int) *Game {
	t.Helper()
	s, err := session.New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(table); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	for frames := 0; s.PreparingBall() && frames < 1000; frames++ {
		if err := s.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.PreparingBall() {
		t.Fatal("ball readiness did not finish")
	}
	return &Game{Session: s, height: resources.FullTableHeight}
}

func touch(id int, x, y int, pressed bool) pointerContact {
	return pointerContact{id: pointerID{touch: ebiten.TouchID(id)}, x: x, y: y, pressed: pressed}
}

func updateContacts(t *testing.T, g *Game, keys session.Input, contacts []pointerContact, keyDown bool) session.Input {
	t.Helper()
	input := g.controls.apply(g.Session, keys, contacts, keyDown, g.height)
	if err := g.Session.Update(input); err != nil {
		t.Fatal(err)
	}
	return input
}

func TestSuspendRequiresReleaseAndDoesNotLaunchChargedBall(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			g := readyTouchGame(t, id)
			for range 55 {
				updateContacts(t, g, session.Input{}, []pointerContact{touch(1, 300, 510, false)}, false)
			}
			if g.Session.Charge != g.Session.Table.SpringLimit {
				t.Fatal("spring did not reach its original full pull")
			}
			before, clock := g.Session.Engine.State, g.Session.Rules.Frame
			g.back = true
			var cues []sound.Command
			g.Session.Sound.ObserveCommands(func(cue sound.Command) { cues = append(cues, cue) })
			g.Suspend()
			if g.Session.Charge != 0 || g.back {
				t.Fatal("suspension kept a spring pull or a queued Back event")
			}
			for range 4 {
				input := updateContacts(t, g, session.Input{Pause: true, Launch: true}, []pointerContact{
					touch(1, 300, 510, false), touch(2, 50, 300, false),
				}, true)
				if input != (session.Input{}) || !g.Session.Paused || g.Session.Engine.State != before || g.Session.Rules.Frame != clock {
					t.Fatal("retained controls resumed or advanced the suspended ball")
				}
			}
			updateContacts(t, g, session.Input{Pause: true}, nil, true)
			if !g.Session.Paused || !g.controls.waitForRelease {
				t.Fatal("a retained keyboard key released the suspension barrier")
			}
			input := updateContacts(t, g, session.Input{Back: true, Text: "P"}, nil, false)
			if input != (session.Input{}) || g.Session.AskingToQuit() {
				t.Fatal("the release boundary dispatched delayed host input")
			}
			resume := []pointerContact{touch(3, 300, 510, true)}
			input = updateContacts(t, g, session.Input{}, resume, false)
			if !input.Pause || input.Launch || g.Session.Paused {
				t.Fatal("a fresh touch did not exclusively resume play")
			}
			resume[0].pressed = false
			for range 8 {
				input = updateContacts(t, g, session.Input{}, resume, false)
				if input.Launch || g.Session.Charge != 0 || g.Session.Engine.State.VY < -1000 {
					t.Fatal("holding the resume touch pulled or launched the spring")
				}
			}
			if len(cues) != 0 {
				t.Fatalf("suspension/resume emitted unintended game sounds: %v", cues)
			}
			input = g.controls.apply(g.Session, session.Input{}, append(resume,
				touch(4, 50, 300, true), touch(5, 220, 300, true)), false, g.height)
			if !input.Left || !input.Right || input.Launch {
				t.Fatal("consuming the resume touch blocked independent flipper contacts")
			}
			// Reusing a platform contact ID for a new press is a fresh gesture.
			input = updateContacts(t, g, session.Input{}, []pointerContact{touch(3, 300, 510, true)}, false)
			if !input.Launch || g.Session.Charge != g.Session.Table.SpringStep {
				t.Fatal("a fresh spring touch did not start a new pull")
			}
			for range 54 {
				updateContacts(t, g, session.Input{}, []pointerContact{touch(3, 300, 510, false)}, false)
			}
			updateContacts(t, g, session.Input{}, nil, false)
			if g.Session.Charge != 0 || g.Session.Engine.State.VY >= -3000 {
				t.Fatal("an intentional new release no longer launched the ball")
			}
		})
	}
}

func TestMousePauseAndResumeDoNotCarryIntoSpringControl(t *testing.T) {
	g := readyTouchGame(t, 0)
	mouse := pointerContact{id: pointerID{mouse: true}, x: 160, y: 10, pressed: true}
	updateContacts(t, g, session.Input{}, []pointerContact{mouse}, false)
	if !g.Session.Paused {
		t.Fatal("score-panel mouse press did not pause")
	}
	mouse.x, mouse.y, mouse.pressed = 300, 510, false
	updateContacts(t, g, session.Input{}, []pointerContact{mouse}, false)
	if !g.Session.Paused {
		t.Fatal("dragging a held mouse press resumed play")
	}
	updateContacts(t, g, session.Input{}, nil, false)
	mouse.pressed = true
	updateContacts(t, g, session.Input{}, []pointerContact{mouse}, false)
	if g.Session.Paused {
		t.Fatal("fresh mouse press did not resume")
	}
	mouse.pressed = false
	input := updateContacts(t, g, session.Input{}, []pointerContact{mouse}, false)
	if input.Launch || g.Session.Charge != 0 {
		t.Fatal("holding the resume mouse press pulled the spring")
	}
}

func TestCancelledTouchCannotLaunchOrConsumePlatformNavigation(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			g := readyTouchGame(t, id)
			contacts := []pointerContact{touch(8, 300, 510, true)}
			for range 55 {
				updateContacts(t, g, session.Input{}, contacts, false)
				contacts[0].pressed = false
			}
			var cues []sound.Command
			g.Session.Sound.ObserveCommands(func(cue sound.Command) { cues = append(cues, cue) })
			g.CancelTouch(8)
			input := updateContacts(t, g, session.Input{}, contacts, false)
			if input.Launch || g.Session.Charge != 0 || g.Session.Paused || !g.Session.InChute || len(cues) != 0 {
				t.Fatal("cancelled contact pulled, launched, paused or sounded the spring")
			}
			updateContacts(t, g, session.Input{ExitTable: true}, nil, false)
			if !g.Session.AskingToQuit() || g.Session.Paused {
				t.Fatal("cancellation consumed the following platform Back request")
			}
		})
	}
}

func TestCancellingOneContactRetainsTheOtherFlipper(t *testing.T) {
	g := readyTouchGame(t, 0)
	contacts := []pointerContact{touch(1, 50, 300, true), touch(2, 220, 300, true)}
	updateContacts(t, g, session.Input{}, contacts, false)
	contacts[0].pressed, contacts[1].pressed = false, false
	g.CancelTouch(1)
	input := updateContacts(t, g, session.Input{}, contacts, false)
	if input.Left || !input.Right || g.Session.Paused {
		t.Fatal("cancelling the left contact changed the independent right grip")
	}
}

func TestTouchViewportChangeCannotReleaseAChargedSpring(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			g := readyTouchGame(t, id)
			g.Touch = true
			g.Layout(1080, 2400)
			contacts := []pointerContact{touch(7, 300, 510, true)}
			for range 55 {
				updateContacts(t, g, session.Input{}, contacts, false)
				contacts[0].pressed = false
			}
			if g.Session.Charge != g.Session.Table.SpringLimit {
				t.Fatal("viewport fixture did not fully charge the spring")
			}
			before, clock := g.Session.Engine.State, g.Session.Rules.Frame
			var cues []sound.Command
			g.Session.Sound.ObserveCommands(func(cue sound.Command) { cues = append(cues, cue) })
			g.Layout(2400, 1080)
			// A retained platform contact can acquire new logical coordinates
			// before Android's cancellation reaches the Go input snapshot.
			contacts[0].x, contacts[0].y = 160, 100
			input := updateContacts(t, g, session.Input{}, contacts, false)
			if input != (session.Input{}) || !g.Session.Paused || g.Session.Charge != 0 || g.Session.Engine.State != before || g.Session.Rules.Frame != clock || len(cues) != 0 {
				t.Fatalf("rescaled touch changed the charged ball: input%+v paused%t charge%d velocity%d cues%v", input, g.Session.Paused, g.Session.Charge, g.Session.Engine.State.VY, cues)
			}
			updateContacts(t, g, session.Input{}, nil, false)
			resume := []pointerContact{touch(8, 160, 180, true)}
			input = updateContacts(t, g, session.Input{}, resume, false)
			if !input.Pause || input.Launch || g.Session.Paused || g.Session.Charge != 0 || g.Session.Engine.State.VY < -1000 {
				t.Fatal("fresh resume did not preserve the silently cancelled spring")
			}
			resume[0].pressed = false
			input = updateContacts(t, g, session.Input{}, append(resume,
				touch(9, 50, 180, true), touch(10, 220, 180, true)), false)
			if !input.Left || !input.Right || input.Launch {
				t.Fatal("rotation resume consumed fresh flippers or inherited a spring pull")
			}
		})
	}
}

func TestTouchViewportChangeWithoutHeldContactsKeepsPlayRunning(t *testing.T) {
	g := readyTouchGame(t, 0)
	g.Touch = true
	g.Layout(1080, 2400)
	updateContacts(t, g, session.Input{}, nil, false)
	g.Layout(2400, 1080)
	input := updateContacts(t, g, session.Input{}, []pointerContact{touch(9, 50, 180, true)}, false)
	if g.Session.Paused || !input.Left || input.Pause {
		t.Fatal("resizing an untouched viewport consumed a fresh flipper contact")
	}
}

func TestViewportChangeKeepsKeyboardChargingIndependent(t *testing.T) {
	g := readyTouchGame(t, 0)
	g.Touch = true
	g.Layout(1080, 2400)
	for range 55 {
		updateContacts(t, g, session.Input{Launch: true}, nil, true)
	}
	g.Layout(2400, 1080)
	input := updateContacts(t, g, session.Input{Launch: true}, nil, true)
	if g.Session.Paused || !input.Launch || g.Session.Charge != g.Session.Table.SpringLimit {
		t.Fatal("a keyboard pull was cancelled by touch viewport resizing")
	}
}

func TestMovingSpringFingerKeepsThePullUntilActualRelease(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			g := readyTouchGame(t, id)
			contacts := []pointerContact{touch(7, 300, 510, true)}
			for range 55 {
				updateContacts(t, g, session.Input{}, contacts, false)
				contacts[0].pressed = false
			}
			contacts[0].x, contacts[0].y = 160, 100
			input := updateContacts(t, g, session.Input{}, contacts, false)
			if !input.Launch || input.Nudge || g.Session.Charge != g.Session.Table.SpringLimit || g.Session.Engine.State.VY < -1000 {
				t.Fatal("moving the held spring finger released its charge")
			}
			updateContacts(t, g, session.Input{}, nil, false)
			if g.Session.Charge != 0 || g.Session.Engine.State.VY >= -3000 {
				t.Fatal("lifting the spring finger no longer launches")
			}
		})
	}
}

func TestViewportChangeDoesNotReinterpretAnAlreadyCancelledContact(t *testing.T) {
	g := readyTouchGame(t, 0)
	g.Touch = true
	g.Layout(1080, 2400)
	contacts := []pointerContact{touch(7, 300, 510, true)}
	for range 55 {
		updateContacts(t, g, session.Input{}, contacts, false)
		contacts[0].pressed = false
	}
	g.CancelTouch(7)
	g.Layout(2400, 1080)
	contacts[0].x, contacts[0].y = 160, 100
	input := updateContacts(t, g, session.Input{}, contacts, false)
	if g.Session.Paused || input.Nudge || input.Launch || g.Session.Charge != 0 {
		t.Fatal("the platform-cancelled contact acquired an action after resizing")
	}
}

func TestTouchAndMouseIdentitiesRemainIndependent(t *testing.T) {
	g := readyTouchGame(t, 0)
	g.Session.Paused = true
	resume := touch(0, 300, 510, true)
	updateContacts(t, g, session.Input{}, []pointerContact{resume}, false)
	resume.pressed = false
	mouse := pointerContact{id: pointerID{mouse: true}, x: 300, y: 510, pressed: true}
	input := g.controls.apply(g.Session, session.Input{}, []pointerContact{resume, mouse}, false, g.height)
	if !input.Launch {
		t.Fatal("a consumed touch with ID zero consumed the mouse too")
	}
}

func TestFreshMusicContactTogglesDespiteAnotherHeldMusicControl(t *testing.T) {
	g := readyTouchGame(t, 0)
	before := g.Session.MusicDisabled
	input := g.controls.apply(g.Session, session.Input{Music: true}, []pointerContact{touch(91, 300, 10, true)}, true, g.height)
	if !input.MusicPressed {
		t.Fatal("fresh music tap was merged into another held control")
	}
	if err := g.Session.Update(input); err != nil {
		t.Fatal(err)
	}
	if g.Session.MusicDisabled == before {
		t.Fatal("fresh music tap did not toggle")
	}
	input = g.controls.apply(g.Session, session.Input{Music: true}, []pointerContact{touch(91, 300, 10, false)}, true, g.height)
	if input.MusicPressed {
		t.Fatal("held music tap repeated a command")
	}
}

func TestEditorResumeDoesNotEnterTheResumeKey(t *testing.T) {
	g := readyTouchGame(t, 0)
	entry := highscore.NewEditor(highscore.Candidate{}, g.Session.Table.ScoreEntry.Allowed)
	g.Session.NameEntry = entry
	g.Suspend()
	updateContacts(t, g, session.Input{}, nil, false)
	input := updateContacts(t, g, session.Input{Pause: true, Text: "P", Start: true, Right: true, Backspace: true}, nil, true)
	if input != (session.Input{Pause: true}) || g.Session.Paused || entry.Cursor != 0 {
		t.Fatal("resuming the initials editor typed, edited or submitted a character")
	}
	updateContacts(t, g, session.Input{}, nil, false)
	input = updateContacts(t, g, session.Input{Pause: true, Text: "P"}, nil, true)
	if input.Pause || g.Session.Paused || entry.Cursor != 1 || entry.Buffer[0] != 'P' {
		t.Fatal("P could not be entered after the editor resumed")
	}
}

func TestManualPauseRetainsExistingFlipperGrips(t *testing.T) {
	g, reference := readyTouchGame(t, 0), readyTouchGame(t, 0)
	contacts := []pointerContact{touch(1, 50, 300, true), touch(2, 220, 300, true)}
	updateContacts(t, g, session.Input{}, contacts, false)
	if err := reference.Session.Update(session.Input{Left: true, Right: true}); err != nil {
		t.Fatal(err)
	}
	contacts[0].pressed, contacts[1].pressed = false, false
	pause := append(append([]pointerContact(nil), contacts...), touch(3, 160, 10, true))
	updateContacts(t, g, session.Input{}, pause, false)
	if !g.Session.Paused {
		t.Fatal("header press did not pause the held flippers")
	}
	for range 4 {
		input := updateContacts(t, g, session.Input{}, contacts, false)
		if !input.Left || !input.Right || input.Pause || !g.Session.Paused {
			t.Fatal("existing flipper grips resumed play or lost their held state")
		}
	}
	resume := append(append([]pointerContact(nil), contacts...), touch(4, 300, 510, true))
	input := updateContacts(t, g, session.Input{}, resume, false)
	if !input.Left || !input.Right || !input.Pause || input.Launch || g.Session.Paused {
		t.Fatal("resume changed the held flippers or pulled the spring")
	}
	if err := reference.Session.Update(session.Input{Left: true, Right: true}); err != nil {
		t.Fatal(err)
	}
	if g.Session.Engine.State != reference.Session.Engine.State || g.Session.Rules.Frame != reference.Session.Rules.Frame {
		t.Fatal("pausing and resuming a flipper grip changed the continued physical state")
	}
}

func TestSuspendedLiveBallContinuesTheSameNativeTrajectory(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			g, reference := readyTouchGame(t, id), readyTouchGame(t, id)
			for frame := 0; frame < 110; frame++ {
				input := session.Input{Launch: frame < 55}
				for _, s := range []*session.Session{g.Session, reference.Session} {
					if err := s.Update(input); err != nil {
						t.Fatal(err)
					}
				}
			}
			if g.Session.PreparingBall() || g.Session.InChute || g.Session.Bonus != nil || g.Session.Engine.State.VX == 0 && g.Session.Engine.State.VY == 0 {
				t.Fatal("suspension fixture did not reach a moving ball outside the spring")
			}
			before, clock, lamps := g.Session.Engine.State, g.Session.Rules.Frame, g.Session.Rules.LampDigest()
			g.Suspend()
			for range 30 {
				updateContacts(t, g, session.Input{}, nil, false)
			}
			if g.Session.Engine.State != before || g.Session.Rules.Frame != clock || g.Session.Rules.LampDigest() != lamps {
				t.Fatal("suspension changed the moving ball, rule clock or lamps")
			}
			updateContacts(t, g, session.Input{}, []pointerContact{touch(8, 160, 250, true)}, false)
			if err := reference.Session.Update(session.Input{}); err != nil {
				t.Fatal(err)
			}
			for frame := 0; frame < 20; frame++ {
				input := session.Input{Left: frame < 8, Right: frame >= 6 && frame < 14}
				updateContacts(t, g, input, nil, input.Left || input.Right)
				if err := reference.Session.Update(input); err != nil {
					t.Fatal(err)
				}
				if g.Session.Engine.State != reference.Session.Engine.State || g.Session.Camera != reference.Session.Camera || g.Session.Rules.Frame != reference.Session.Rules.Frame || g.Session.Rules.LampDigest() != reference.Session.Rules.LampDigest() {
					t.Fatalf("resumed trajectory differs on continued frame %d", frame)
				}
			}
		})
	}
}

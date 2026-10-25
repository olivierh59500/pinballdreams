package game

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/internal/highscore"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/touchui"
)

func readyNavigationGame(t *testing.T, table int) *Game {
	g := readyTouchGame(t, table)
	g.Touch = true
	return g
}

func TestTouchRootCompositionPreservesCenteredOriginalMenu(t *testing.T) {
	g, err := New(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err = g.Session.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	for !g.Session.Menu.Ready() {
		if err = g.Session.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, height := range []int{256, 542} {
		g.height = height
		source := g.Session.Frame(height)
		before := append([]byte(nil), source.Pix...)
		frame := g.drawTouchControls(source)
		top := (height - source.Rect.Dy()) / 2
		for row := range source.Rect.Dy() {
			if !bytes.Equal(frame.Pix[(top+row)*frame.Stride:(top+row)*frame.Stride+320*4], before[row*source.Stride:row*source.Stride+320*4]) {
				t.Fatalf("height %d: native touch controls moved or replaced original menu row %d", height, row)
			}
		}
		if !bytes.Equal(source.Pix, before) || frame.Rect.Dy() != height+touchui.ToolbarHeight {
			t.Fatal("host rendering mutated the original menu or lost its toolbar")
		}
	}
}

func TestLabelledInitialsControlsDoNotReachHiddenTableRegions(t *testing.T) {
	g := readyNavigationGame(t, 0)
	g.Session.NameEntry = highscore.NewEditor(highscore.Candidate{}, " ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	for _, test := range []struct {
		x, y int
		want session.InitialsAction
	}{
		{30, g.height + 20, session.InitialsPrevious},
		{128, g.height + 20, session.InitialsChoose},
		{220, g.height + 20, session.InitialsNext},
		{160, (g.height+touchui.ToolbarHeight)/2 + 50, session.InitialsErase},
		{30, 100, session.InitialsNone},
	} {
		g.contacts = []pointerContact{touch(9, test.x, test.y, true)}
		var input session.Input
		g.applyTouchScreenControls(&input)
		input = g.controls.apply(g.Session, input, g.contacts, false, g.height)
		got := session.InitialsNone
		if len(g.initialsEvents) > 0 {
			got = g.initialsEvents[0]
		}
		if got != test.want || input.Left || input.Right || input.Launch || input.Pause {
			t.Fatalf("labelled editor input also reached the original broad regions: %+v", input)
		}
		g.controls.apply(g.Session, session.Input{}, nil, false, g.height)
	}
}

func TestHeldBackStateDoesNotToggleTouchNavigation(t *testing.T) {
	g := readyNavigationGame(t, 0)
	g.openTouchNavigation()
	for range 10 {
		if handled, err := g.handleTouchNavigation(session.Input{Back: true}, true); !handled || err != nil || !g.navigation.open {
			t.Fatal("a held key toggled touch navigation repeatedly")
		}
	}
	if _, err := g.handleTouchNavigation(session.Input{ExitTable: true}, false); err != nil || g.navigation.open {
		t.Fatal("a fresh platform Back request did not close navigation")
	}
}

func TestRootToolbarNavigatesScoresAndResumesPausedMenu(t *testing.T) {
	g, err := New(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	g.height = 542
	if err = g.Session.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	for !g.Session.Menu.Ready() {
		if err = g.Session.Update(session.Input{}); err != nil {
			t.Fatal(err)
		}
	}
	tap := func() session.Input {
		g.contacts = []pointerContact{touch(11, 160, 562, true)}
		var input session.Input
		g.applyTouchScreenControls(&input)
		return g.controls.apply(g.Session, input, g.contacts, false, g.height)
	}
	if err = g.Session.Update(tap()); err != nil {
		t.Fatal(err)
	}
	for count := 0; g.Session.Scores == nil || !g.Session.Scores.Ready(); count++ {
		if count > 100 || g.Session.Update(session.Input{}) != nil {
			t.Fatal("labelled SCORES control did not open rankings")
		}
	}
	g.controls.apply(g.Session, session.Input{}, nil, false, g.height)
	if err = g.Session.Update(tap()); err != nil {
		t.Fatal(err)
	}
	for count := 0; g.Session.Scores != nil || !g.Session.Menu.Ready(); count++ {
		if count > 100 || g.Session.Update(session.Input{}) != nil {
			t.Fatal("labelled TABLES control did not return to selection")
		}
	}
	g.controls.apply(g.Session, session.Input{}, nil, false, g.height)
	g.Session.Suspend()
	input := tap()
	if !input.Pause || input.HighScores {
		t.Fatal("paused menu toolbar attempted navigation instead of resume")
	}
}

func TestRootToolbarRetainsRequestDuringInitialFade(t *testing.T) {
	g, err := New(nil, true)
	if err != nil {
		t.Fatal(err)
	}
	g.height = 542
	if err = g.Session.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	if g.Session.Menu.Ready() {
		t.Fatal("fixture did not begin during the root fade")
	}
	g.contacts = []pointerContact{touch(12, 160, 562, true)}
	var first session.Input
	g.applyTouchScreenControls(&first)
	if !g.rootRequest.pending || first.HighScores {
		t.Fatal("fading root scene lost or prematurely dispatched the tap")
	}
	g.contacts = nil
	for frame := 0; g.Session.Scores == nil || !g.Session.Scores.Ready(); frame++ {
		if frame > 120 {
			t.Fatal("tap during fade never reached the scores scene")
		}
		var input session.Input
		g.applyTouchScreenControls(&input)
		if err = g.Session.Update(input); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTouchNavigationFreezesAndResumesWithoutForwardingGesture(t *testing.T) {
	g := readyNavigationGame(t, 0)
	g.openTouchNavigation()
	clock, ball := g.Session.Rules.Frame, g.Session.Engine.State
	for range 20 {
		handled, err := g.handleTouchNavigation(session.Input{}, false)
		if !handled || err != nil || !g.Session.Paused {
			t.Fatal("touch navigation did not retain the frozen table")
		}
	}
	if g.Session.Rules.Frame != clock || g.Session.Engine.State != ball {
		t.Fatal("navigation advanced game physics or rule time")
	}
	if err := g.applyTouchNavigation(touchui.Resume); err != nil {
		t.Fatal(err)
	}
	if g.Session.Paused || g.navigation.open || !g.controls.waitForRelease || g.pending.input != (session.Input{}) {
		t.Fatal("resume forwarded a menu gesture or left the game paused")
	}
}

func TestTouchTableSelectionBypassesStoppedSongExit(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			g := readyNavigationGame(t, table)
			g.Session.Sound.BeginTilt(g.Session.Table.Tilt.EntryCue)
			for waits := 0; !g.Session.Sound.PlaybackStopped(); waits++ {
				if waits > 20000 {
					t.Fatal("terminal song did not stop")
				}
				g.Session.Sound.AdvanceFrame()
			}
			g.openTouchNavigation()
			if err := g.applyTouchNavigation(touchui.Tables); err != nil {
				t.Fatal(err)
			}
			if g.Session.Table != nil || g.Session.Menu == nil || g.Session.Paused || g.Session.AskingToQuit() {
				t.Fatal("table selection waited for the original quit prompt or terminal fade")
			}
		})
	}
}

func TestTouchRestartBeginsNewGameOnSelectedTable(t *testing.T) {
	g := readyNavigationGame(t, 3)
	g.Session.Players[0].Score = 123456
	g.openTouchNavigation()
	if err := g.applyTouchNavigation(touchui.Restart); err != nil {
		t.Fatal(err)
	}
	if g.Session.Table.ID != 3 || g.Session.Ball != 1 || g.Session.Players[0].Score != 0 || !g.Session.Playing || g.Session.Paused || g.navigation.open {
		t.Fatal("restart did not reset the selected table into a fresh game")
	}
}

func TestCancelledFreshSnapshotCannotActivateMenu(t *testing.T) {
	g := readyNavigationGame(t, 0)
	g.openTouchNavigation()
	g.navigation.waitForRelease = false
	button := touchui.Buttons(g.height + touchui.ToolbarHeight)[2]
	g.CancelTouch(7)
	g.contacts = []pointerContact{touch(7, button.Min.X+20, button.Min.Y+20, true)}
	if _, err := g.handleTouchNavigation(session.Input{}, false); err != nil {
		t.Fatal(err)
	}
	if !g.navigation.open || g.Session.Table == nil {
		t.Fatal("a cancelled first snapshot activated table selection")
	}
	g.contacts = nil
	g.handleTouchNavigation(session.Input{}, false)
	g.contacts = []pointerContact{touch(7, button.Min.X+20, button.Min.Y+20, true)}
	if _, err := g.handleTouchNavigation(session.Input{}, false); err != nil {
		t.Fatal(err)
	}
	if g.Session.Table != nil {
		t.Fatal("cancellation consumed the next gesture after a neutral snapshot")
	}
}

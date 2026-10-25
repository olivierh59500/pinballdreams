package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/timing"
)

func TestPollsRetainShortActionsAndRelativeTravelUntilGameStep(t *testing.T) {
	var pending pendingInput
	pending.add(session.Input{Left: true, Nudge: true, Launch: true, SpringMotion: 12, MusicPressed: true, Text: "A"}, []ebiten.Key{ebiten.KeyM})
	pending.add(session.Input{SpringMotion: -3, StartPressed: true, Text: "B"}, nil)
	input := pending.input
	if !input.Left || !input.Nudge || !input.MusicPressed || !input.StartPressed || input.Launch || input.SpringMotion != 9 || input.Text != "AB" {
		t.Fatalf("commands or latest launcher state lost: %+v", input)
	}
	pending.reset()
	pending.add(session.Input{}, nil)
	if pending.input != (session.Input{}) || len(pending.keys) != 0 {
		t.Fatal("the next game step repeated a consumed host action")
	}
}

func TestInitialsCommandsRetainOrderAcrossLogicalSteps(t *testing.T) {
	var pending pendingInput
	pending.add(session.Input{InitialsEdit: session.InitialsChoose}, nil)
	pending.add(session.Input{InitialsEdit: session.InitialsNext}, nil, session.InitialsErase, session.InitialsChoose)
	for _, want := range []session.InitialsAction{session.InitialsChoose, session.InitialsNext, session.InitialsErase, session.InitialsChoose} {
		got := pending.take()
		pending.finishFrame()
		if got.InitialsEdit != want {
			t.Fatalf("queued editing command: got %d want %d", got.InitialsEdit, want)
		}
		pending.add(session.Input{}, nil)
	}
	if pending.take().InitialsEdit != session.InitialsNone {
		t.Fatal("a consumed initials command repeated")
	}
	pending.add(session.Input{InitialsEdit: session.InitialsChoose}, nil)
	pending.reset()
	if pending.take().InitialsEdit != session.InitialsNone {
		t.Fatal("suspension/navigation retained an old editor command")
	}
}

func TestBetweenStepTouchCommandsSurviveWithoutRepeating(t *testing.T) {
	var pending pendingInput
	var clock timing.HostClock
	var delivered []session.Input
	for poll := range 20 {
		input := session.Input{}
		if poll == 1 {
			// A tap can begin and end before the next logical table step.
			input.MusicPressed, input.StartPressed = true, true
		}
		pending.add(input, nil)
		if clock.Tick(timing.Table) {
			delivered = append(delivered, pending.input)
			pending.reset()
		}
	}
	commands := 0
	for _, input := range delivered {
		if input.MusicPressed && input.StartPressed {
			commands++
		}
	}
	if commands != 1 {
		t.Fatalf("tap delivered %d times", commands)
	}
}

func TestSuspendDropsPendingActionsBeforeResume(t *testing.T) {
	g, err := New(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	g.pending.add(session.Input{Left: true, Launch: true, SpringMotion: 40, SpringClick: true, ExitTable: true, Text: "Y"}, []ebiten.Key{ebiten.KeyY})
	g.Suspend()
	if g.pending.input != (session.Input{}) || len(g.pending.keys) != 0 {
		t.Fatal("suspension retained actions for the resumed session")
	}
}

package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func livePauseSession(t *testing.T, table int) *Session {
	t.Helper()
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(table); err != nil {
		t.Fatal(err)
	}
	s.EnableFrameAudio(nil)
	if err = s.Update(Input{Start: true, Players: 1, Launch: true}); err != nil {
		t.Fatal(err)
	}
	for step := 0; s.PreparingBall() && step < 512; step++ {
		if err = s.Update(Input{Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	for range 55 {
		if err = s.Update(Input{Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	for step := 0; s.launching && step < 300; step++ {
		if err = s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.launching || !s.Playing {
		t.Fatal("normal shot did not enter live play")
	}
	return s
}

func TestLivePauseWaitsKeepClockAndSpringWhileMotionStops(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s := livePauseSession(t, id)
			before, camera, clock := s.Engine.State, s.Camera, s.Rules.Frame
			if err := s.Update(Input{Pause: true, Launch: true}); err != nil {
				t.Fatal(err)
			}
			if !s.Paused || s.tablePause == nil || s.panel.text() != s.Table.PauseText {
				t.Fatal("live pause did not use its original prompt")
			}
			for range 9 {
				if err := s.Update(Input{Launch: true}); err != nil {
					t.Fatal(err)
				}
			}
			if s.Engine.State != before || s.Camera != camera || s.Rules.Frame != clock+10 || s.Charge == 0 {
				t.Fatal("pause changed motion or stopped its wait-side clock/spring")
			}
			players, music := s.PlayerCount, s.MusicDisabled
			if err := s.Update(Input{AnyKey: true, StartPressed: true, Players: 8, Music: true}); err != nil {
				t.Fatal(err)
			}
			if s.Paused || s.tablePause != nil || s.Engine.State != before || s.PlayerCount != players || s.MusicDisabled != music || s.panel.text() != s.Table.PauseText {
				t.Fatal("resume dispatched gameplay input or lost its final paused window")
			}
		})
	}
}

func TestPausedQuitCancelReturnsToPauseAndConfirmDefersExit(t *testing.T) {
	s := livePauseSession(t, 0)
	step := func(input Input) {
		t.Helper()
		if err := s.Update(input); err != nil {
			t.Fatal(err)
		}
	}
	step(Input{Pause: true})
	before := s.Engine.State
	step(Input{Back: true, AnyKey: true})
	if !s.Paused || !s.AskingToQuit() || s.panel.text() != s.Table.Quit.Prompt {
		t.Fatal("Escape did not open the paused quit question")
	}
	step(Input{})
	step(Input{QuitCancel: true, AnyKey: true})
	if !s.Paused || s.AskingToQuit() || s.tablePause == nil || s.Engine.State != before {
		t.Fatal("cancel left the original pause")
	}
	step(Input{Back: true})
	step(Input{QuitConfirm: true, AnyKey: true})
	if s.Paused || s.tablePause == nil || !s.tablePause.exiting || s.quit != nil || s.panel.text() != s.Table.PauseText {
		t.Fatal("confirmation did not retain the final pause wait")
	}
	step(Input{})
	if s.tablePause != nil || s.quit == nil || s.AskingToQuit() || s.Engine.State != before {
		t.Fatal("confirmed pause did not enter the exit fade")
	}
}

func TestSystemSuspendPreservesManualPauseWithFrozenClock(t *testing.T) {
	s := livePauseSession(t, 0)
	if err := s.Update(Input{Pause: true, Launch: true}); err != nil {
		t.Fatal(err)
	}
	s.Suspend()
	before, clock := s.Engine.State, s.Rules.Frame
	for range 10 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.tablePause == nil || !s.tablePause.suspended || !s.Paused || s.Engine.State != before || s.Rules.Frame != clock || s.Charge != 0 {
		t.Fatal("system suspension retained manual wait activity")
	}
	if err := s.Update(Input{AnyKey: true}); err != nil {
		t.Fatal(err)
	}
	if !s.Paused || s.tablePause.suspended || s.Rules.Frame != clock {
		t.Fatal("foreground resume lost the manual pause or advanced its clock")
	}
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.Rules.Frame != clock+1 {
		t.Fatal("manual waits did not restart after foreground resume")
	}
}

func TestPauseResumeWaitDoesNotAdvanceTheReplay(t *testing.T) {
	s := livePauseSession(t, 0)
	blocks := 0
	s.EnableFrameAudio(func([]byte) error { blocks++; return nil })
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Update(Input{AnyKey: true}); err != nil {
		t.Fatal(err)
	}
	if blocks != 10 {
		t.Fatal("manual pause/resume waits did not render their held output")
	}
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if blocks != 11 {
		t.Fatal("ordinary replay did not resume on the next live wait")
	}
}

func TestManualPauseKeepsOutputWhileSystemSuspensionStopsIt(t *testing.T) {
	s := livePauseSession(t, 0)
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	if !s.Paused || s.AudioOutputPaused() {
		t.Fatal("manual pause stopped output instead of holding replay")
	}
	s.Suspend()
	if !s.AudioOutputPaused() {
		t.Fatal("system suspension kept audio output reading")
	}
	if err := s.Update(Input{AnyKey: true}); err != nil {
		t.Fatal(err)
	}
	if !s.Paused || s.AudioOutputPaused() {
		t.Fatal("restored manual pause did not restore continuous output")
	}
}

func TestPausedQuestionPointerAndSuspensionKeepTheirRoles(t *testing.T) {
	s := livePauseSession(t, 0)
	for _, input := range []Input{{Pause: true}, {Back: true}} {
		if err := s.Update(input); err != nil {
			t.Fatal(err)
		}
	}
	if !s.AskingToQuit() {
		t.Fatal("missing manual-pause question")
	}
	s.Suspend()
	var resume Input
	s.ApplyPointer(&resume, s.Table.Quit.ConfirmCell*16+1, 10, s.Table.Viewport.Height)
	if !resume.Pause || resume.QuitConfirm {
		t.Fatal("suspension tap confirmed the retained question")
	}
	if err := s.Update(resume); err != nil {
		t.Fatal(err)
	}
	if !s.Paused || !s.AskingToQuit() {
		t.Fatal("foreground resume lost the retained pause question")
	}
	var confirm Input
	s.ApplyPointer(&confirm, s.Table.Quit.ConfirmCell*16+1, 10, s.Table.Viewport.Height)
	if !confirm.QuitConfirm || confirm.Pause {
		t.Fatal("fresh confirmation tap resumed instead of answering")
	}
}

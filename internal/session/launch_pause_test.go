package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestDeferredKeyboardPauseKeepsPlatformExitAvailable(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Start(1)
			waitForPreparedBall(t, s)
			clock := s.Rules.Frame
			if err := s.Update(Input{PauseKeyPressed: true, Launch: true}); err != nil {
				t.Fatal(err)
			}
			if s.Paused || !s.keyboardPausePending || s.Rules.Frame != clock+1 || s.Charge == 0 {
				t.Fatal("P interrupted launch instead of retaining its deferred request")
			}
			if err := s.Update(Input{Back: true, Launch: true}); err != nil {
				t.Fatal(err)
			}
			if !s.Playing || s.AskingToQuit() || !s.keyboardPausePending {
				t.Fatal("Escape release escaped the source keyboard dispatch guard")
			}
			if err := s.Update(Input{ExitTable: true, Launch: true}); err != nil {
				t.Fatal(err)
			}
			if !s.AskingToQuit() || s.keyboardPausePending {
				t.Fatal("deferred original P blocked explicit platform navigation")
			}
		})
	}
}

func TestKeyboardPauseAndHostPauseKeepSeparateLaunchSemantics(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	if err := s.Update(Input{PauseKeyPressed: true}); err != nil {
		t.Fatal(err)
	}
	if s.Paused || !s.keyboardPausePending || !s.PreparingBall() {
		t.Fatal("original P stopped target preparation")
	}
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	clock, state := s.Rules.Frame, s.Engine.State
	if !s.Paused {
		t.Fatal("immediate host pause was deferred")
	}
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.Rules.Frame != clock || s.Engine.State != state {
		t.Fatal("host pause advanced preparation")
	}
	if err := s.Update(Input{PauseKeyPressed: true}); err != nil {
		t.Fatal(err)
	}
	if s.Paused || !s.keyboardPausePending {
		t.Fatal("fresh P could not resume a host pause")
	}
	if err := s.Select(1); err != nil {
		t.Fatal(err)
	}
	if s.keyboardPausePending {
		t.Fatal("deferred P leaked into another table")
	}
}

func TestKeyboardPauseStillEntersOriginalLiveWait(t *testing.T) {
	s := livePauseSession(t, 0)
	if err := s.Update(Input{PauseKeyPressed: true}); err != nil {
		t.Fatal(err)
	}
	if !s.Paused || s.tablePause == nil || s.keyboardPausePending {
		t.Fatal("live P lost the original pause loop")
	}
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(Input{PauseKeyPressed: true, AnyKey: true}); err != nil {
		t.Fatal(err)
	}
	if s.Paused || s.tablePause != nil {
		t.Fatal("fresh P did not resume live play")
	}
}

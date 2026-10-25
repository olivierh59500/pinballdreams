package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestAttractAndPanDeferKeyboardPauseUntilServing(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.Camera = 0
			clock := s.Rules.Frame
			if err = s.Update(Input{PauseKeyPressed: true}); err != nil {
				t.Fatal(err)
			}
			if s.Paused || !s.keyboardPausePending || s.Rules.Frame != clock+1 || s.Playing {
				t.Fatal("P interrupted table attract")
			}
			if err = s.Update(Input{Start: true, Players: 2}); err != nil {
				t.Fatal(err)
			}
			for n := 0; !s.pendingNewBall && n < 1000; n++ {
				if err = s.Update(Input{PauseKeyPressed: n == 2 || n == 6}); err != nil {
					t.Fatal(err)
				}
				if s.Paused {
					t.Fatal("P interrupted the original start pan")
				}
			}
			if !s.pendingNewBall || !s.Playing || s.PlayerCount != 2 || s.keyboardPausePending {
				t.Fatal("start pan did not replace its pending request before serving")
			}
		})
	}
}

func TestAttractPendingPauseDoesNotBlockEscape(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Camera = 0
	if err = s.Update(Input{PauseKeyPressed: true}); err != nil {
		t.Fatal(err)
	}
	if err = s.Update(Input{Back: true}); err != nil {
		t.Fatal(err)
	}
	if !s.AskingToQuit() || s.keyboardPausePending {
		t.Fatal("a deferred attract P blocked normal quit handling")
	}
}

func TestHostPauseStillFreezesTheStartPan(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Camera = 0
	if err = s.Update(Input{Start: true, Players: 1}); err != nil {
		t.Fatal(err)
	}
	if err = s.Update(Input{PauseKeyPressed: true}); err != nil {
		t.Fatal(err)
	}
	if err = s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	clock, camera := s.Rules.Frame, s.Camera
	for range 8 {
		if err = s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if !s.Paused || s.Rules.Frame != clock || s.Camera != camera {
		t.Fatal("host pause failed to freeze the pan")
	}
	if err = s.Update(Input{AnyKey: true}); err != nil {
		t.Fatal(err)
	}
	if s.Paused || !s.keyboardPausePending {
		t.Fatal("host resumption lost the deferred original request")
	}
	if err = s.Update(Input{ExitTable: true}); err != nil {
		t.Fatal(err)
	}
	if !s.AskingToQuit() || s.keyboardPausePending {
		t.Fatal("pan blocked platform navigation")
	}
}

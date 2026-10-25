package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func scorePauseSession(t *testing.T, table, players int) *Session {
	t.Helper()
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(table); err != nil {
		t.Fatal(err)
	}
	s.PlayerCount = players
	for n := 0; n < players; n++ {
		s.Players[n].Score = uint64(n * 1000)
	}
	s.endGame()
	if !s.gameOver || s.NameEntry != nil {
		t.Fatal("low scores did not enter the ordinary score cycle")
	}
	return s
}

func TestScoreCycleKeyboardPauseExpiresAtTableReset(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s := scorePauseSession(t, id, 2)
			before := s.Engine.State
			clock := s.Rules.Frame
			if err := s.Update(Input{PauseKeyPressed: true}); err != nil {
				t.Fatal(err)
			}
			if s.Paused || !s.keyboardPausePending || s.Rules.Frame != clock+1 {
				t.Fatal("P interrupted score cycling")
			}
			for n := 0; s.gameOver && n < 20000; n++ {
				if err := s.Update(Input{PauseKeyPressed: n == 3}); err != nil {
					t.Fatal(err)
				}
			}
			if s.gameOver || !s.pendingTableReset || s.keyboardPausePending || s.Engine.State != before {
				t.Fatal("score cycle failed to replace its request without moving the ball")
			}
			if err := s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
			if s.attract == nil || s.PlayerCount != 0 || s.Paused {
				t.Fatal("deferred P leaked into table attract")
			}
		})
	}
}

func TestScoreCycleHostPauseAndRestartKeepTheirPriority(t *testing.T) {
	s := scorePauseSession(t, 0, 1)
	if err := s.Update(Input{PauseKeyPressed: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	clock, camera, age := s.Rules.Frame, s.Camera, s.gameOverFrames
	for range 5 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if !s.Paused || s.Rules.Frame != clock || s.Camera != camera || s.gameOverFrames != age {
		t.Fatal("host pause failed to freeze score cycling")
	}
	if err := s.Update(Input{AnyKey: true}); err != nil {
		t.Fatal(err)
	}
	if s.Paused || !s.keyboardPausePending {
		t.Fatal("host pause resumption lost the pending original request")
	}
	if err := s.Update(Input{StartPressed: true, Players: 2}); err != nil {
		t.Fatal(err)
	}
	if s.gameOver || s.PlayerCount != 2 || s.keyboardPausePending || s.startPhase != tableStartEntering {
		t.Fatal("P blocked a normal restart or leaked through its table reset")
	}
}

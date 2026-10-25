package session

import (
	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/sound"
	"testing"
)

func heldCancelSession(t *testing.T, table int) *Session {
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
	for n := 0; s.PreparingBall() && n < 512; n++ {
		if err = s.Update(Input{Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	for range 55 {
		if err = s.Update(Input{Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	if s.PreparingBall() || !s.launching || !s.InChute {
		t.Fatal("normal ball did not reach its held spring loop")
	}
	return s
}

func TestEscapeCancelsHeldGameAndReturnsToTableAttract(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s := heldCancelSession(t, id)
			book := s.ScoreBook
			s.Players[0].Score = 12345
			var cues []sound.Command
			s.Sound.ObserveCommands(func(cue sound.Command) { cues = append(cues, cue) })
			clock := s.Rules.Frame
			if err := s.Update(Input{Back: true, Launch: true}); err != nil {
				t.Fatal(err)
			}
			if s.Playing || s.PlayerCount != 0 || !s.pendingTableReset || s.AskingToQuit() || s.Table == nil || s.Rules.Frame != clock+1 {
				t.Fatal("spring Escape did not cancel the current game at its wait boundary")
			}
			if len(cues) != 1 || cues[0] != s.Table.Launch.CancelCue {
				t.Fatalf("source cancel cue differs: %v", cues)
			}
			if err := s.Update(Input{Launch: true}); err != nil {
				t.Fatal(err)
			}
			if s.Playing || s.attract == nil || s.PlayerCount != 0 || s.Players != ([8]Player{}) || !s.spritesHidden || s.ScoreBook != book {
				t.Fatal("cancel did not reset table attract without score qualification")
			}
		})
	}
}

func TestPreparationEscapeIsIgnoredAndPlatformBackKeepsQuitQuestion(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s := heldCancelSession(t, id)
			if err := s.Update(Input{ExitTable: true, Launch: true}); err != nil {
				t.Fatal(err)
			}
			if !s.AskingToQuit() || !s.Playing {
				t.Fatal("platform table-exit request canceled the game instead of asking")
			}
			fresh, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = fresh.Select(id); err != nil {
				t.Fatal(err)
			}
			if err = fresh.Update(Input{Start: true, Players: 1}); err != nil {
				t.Fatal(err)
			}
			for n := 0; fresh.preparation == nil && n < 512; n++ {
				if err = fresh.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if fresh.preparation == nil {
				t.Fatal("missing original countdown context")
			}
			if err = fresh.Update(Input{Back: true}); err != nil {
				t.Fatal(err)
			}
			if fresh.AskingToQuit() || !fresh.Playing || !fresh.PreparingBall() {
				t.Fatal("preparation Escape escaped its original countdown")
			}
		})
	}
}

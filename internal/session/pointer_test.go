package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"github.com/olivierh59500/pinballdreams/internal/highscore"
	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

func TestTouchAddsPlayersWithoutRestartingReadiness(t *testing.T) {
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
			for n := 0; s.preparation == nil && n < 1000; n++ {
				if err := s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if s.preparation == nil {
				t.Fatal("readiness never began")
			}
			for want := 2; want <= 8; want++ {
				before, phase, remaining := s.Rules.Frame, s.preparation.phase, s.preparation.remaining
				var tap Input
				s.ApplyPointer(&tap, 8, 10, resources.FullTableHeight)
				if err := s.Update(tap); err != nil {
					t.Fatal(err)
				}
				if s.PlayerCount != want || s.Rules.PlayerCount != want || s.Rules.Frame != before+1 || s.preparation == nil || s.preparation.phase != phase || s.preparation.remaining != remaining-1 {
					t.Fatalf("touch player %d restarted or delayed readiness", want)
				}
				if err := s.Update(tap); err != nil {
					t.Fatal(err)
				}
				if s.PlayerCount != want {
					t.Fatal("held roster touch repeated its selection")
				}
				if err := s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			var cues []audiodata.Command
			s.Sound.ObserveCommands(func(c audiodata.Command) { cues = append(cues, c) })
			var tap Input
			s.ApplyPointer(&tap, 8, 10, resources.FullTableHeight)
			if err := s.Update(tap); err != nil {
				t.Fatal(err)
			}
			if s.PlayerCount != 8 || len(cues) != 0 {
				t.Fatal("eight-player increment was not capped silently")
			}
		})
	}
}

func TestPointerModalControlsPrecedeHeaderActions(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.NameEntry = highscore.NewEditor(highscore.Candidate{}, s.Table.ScoreEntry.Allowed)
	var initials Input
	s.ApplyPointer(&initials, 300, 10, resources.FullTableHeight)
	if !initials.Right || initials.Music || initials.Start || initials.Pause {
		t.Fatal("header actions consumed an initials touch")
	}
	s.NameEntry = nil
	s.beginQuit()
	var confirm Input
	// The selectable Y belongs to the parenthesized choices. The earlier
	// word REALLY also contains Y and must not become the confirm control.
	s.ApplyPointer(&confirm, 14*16+8, 10, resources.FullTableHeight)
	if !confirm.QuitConfirm || confirm.QuitCancel || confirm.Music || confirm.Start || confirm.Pause {
		t.Fatal("the original Y glyph did not retain quit priority")
	}
	var cancel Input
	s.ApplyPointer(&cancel, 300, 10, resources.FullTableHeight)
	if !cancel.QuitCancel || cancel.Music || cancel.Start || cancel.Pause {
		t.Fatal("a non-Y question touch reached gameplay controls")
	}
	var word Input
	s.ApplyPointer(&word, 6*16+8, 10, resources.FullTableHeight)
	if word.QuitConfirm || !word.QuitCancel {
		t.Fatal("the Y in REALLY accepted the quit question")
	}
}

func TestQuitTouchUsesTheDisplayedAnswerPairOnEveryTable(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Select(id); err != nil {
				t.Fatal(err)
			}
			s.beginQuit()
			for _, cell := range []int{6, 14, 16} {
				var tap Input
				s.ApplyPointer(&tap, cell*16+8, 10, resources.FullTableHeight)
				if tap.QuitConfirm != (cell == 14) || tap.QuitCancel != (cell != 14) {
					t.Fatalf("displayed question cell %d produced %+v", cell, tap)
				}
			}
		})
	}
}

func TestTouchMusicAndIndependentTableRegions(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	waitForPreparedBall(t, s)
	// This fixture selects the live input context; motion fidelity is checked
	// independently by the source-driven gameplay traces.
	s.launching, s.playerSelectionOpen = false, false
	module, err := sound.Decode(s.Table)
	if err != nil {
		t.Fatal(err)
	}
	var cues []audiodata.Command
	s.Sound.ObserveCommands(func(c audiodata.Command) { cues = append(cues, c) })
	var input Input
	s.ApplyPointer(&input, 300, 10, resources.FullTableHeight)
	s.ApplyPointer(&input, 50, 300, resources.FullTableHeight)
	if !input.Music || !input.Left || input.Pause {
		t.Fatal("music region consumed another touch or paused")
	}
	before := s.MusicDisabled
	if err := s.Update(input); err != nil {
		t.Fatal(err)
	}
	want := module.QuietMusic
	if before {
		want = module.PlayMusic
	}
	if s.MusicDisabled == before || len(cues) == 0 || cues[0] != want {
		t.Fatal("touch did not use the ordinary quiet/play music command")
	}
	if err := s.Update(input); err != nil {
		t.Fatal(err)
	}
	if s.MusicDisabled == before {
		t.Fatal("held music touch toggled repeatedly")
	}
	// Function-key selection closes at the first play sensor. It cannot erase
	// a running game or start a new restoration sequence afterward.
	clock, count := s.Rules.Frame, s.PlayerCount
	if err := s.Update(Input{Start: true, Players: 8}); err != nil {
		t.Fatal(err)
	}
	if s.PlayerCount != count || s.Rules.Frame != clock+1 || s.PreparingBall() {
		t.Fatal("closed player selection restarted active play")
	}
	var controls Input
	s.ApplyPointer(&controls, 50, 300, resources.FullTableHeight)
	s.ApplyPointer(&controls, 220, 300, resources.FullTableHeight)
	s.ApplyPointer(&controls, 300, resources.FullTableHeight-40, resources.FullTableHeight)
	if !controls.Left || !controls.Right || !controls.Launch {
		t.Fatal("independent flipper/spring touches were lost")
	}
	var paused Input
	s.Paused = true
	s.ApplyPointer(&paused, 300, 10, resources.FullTableHeight)
	if !paused.Pause || paused.Music {
		t.Fatal("modal resume did not take precedence over music")
	}
	for _, point := range [][2]int{{-1, 10}, {320, 10}, {100, -1}, {100, resources.FullTableHeight}} {
		var outside Input
		s.ApplyPointer(&outside, point[0], point[1], resources.FullTableHeight)
		if outside != (Input{}) {
			t.Fatal("letterbox touch reached game controls")
		}
	}
}

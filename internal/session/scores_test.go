package session

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
)

func TestMenuFooterContinuesThroughRankingsAndPause(t *testing.T) {
	var sessions [2]*Session
	for n := range sessions {
		s, err := New(assets.Files())
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ShowMenu(); err != nil {
			t.Fatal(err)
		}
		sessions[n] = s
	}
	for frame := range 350 {
		for n, s := range sessions {
			input := Input{HighScores: n == 1 && (frame == 80 || frame == 200)}
			if err := s.Update(input); err != nil {
				t.Fatal(err)
			}
		}
	}
	if sessions[1].Scores != nil {
		t.Fatal("rankings did not return to the menu")
	}
	stripPixels := func(s *Session) []byte {
		frame := s.Frame(256)
		var pixels []byte
		for _, top := range []int{s.presentation.Footer.TickerY, s.presentation.Footer.CaptionY} {
			for y := top; y < top+8; y++ {
				pixels = append(pixels, frame.Pix[y*frame.Stride:y*frame.Stride+320*4]...)
			}
		}
		return pixels
	}
	if !bytes.Equal(stripPixels(sessions[0]), stripPixels(sessions[1])) {
		t.Fatal("navigation restarted or delayed the shared footer")
	}
	s := sessions[1]
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	frozen := stripPixels(s)
	for range 30 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(frozen, stripPixels(s)) {
		t.Fatal("the paused footer continued to animate")
	}
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(frozen, stripPixels(s)) {
		t.Fatal("the resumed footer did not advance")
	}
}

func TestLabelledInitialsCommandsCommitAndEraseWithoutHeldKeyEdges(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(1)
	s.Players[0].Score = 9_000_000
	s.Ball = 3
	s.finishBall()
	if err = s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.NameEntry == nil || s.NameEntry.TouchDisplay() != "A  " || s.NameEntry.Cursor != 0 {
		t.Fatal("touch preview did not show an uncommitted initial A")
	}
	for _, action := range []InitialsAction{InitialsChoose, InitialsNext, InitialsChoose, InitialsErase, InitialsChoose, InitialsNext, InitialsChoose} {
		if err = s.Update(Input{InitialsEdit: action}); err != nil {
			t.Fatal(err)
		}
	}
	if s.NameEntry != nil || s.ScoreBook[0][0].Initials != "ABC" {
		t.Fatalf("labelled touch editing did not commit ABC: %+v", s.ScoreBook[0][0])
	}
}

func TestEndGameInitialsSaveAndProcessEveryQualifyingPlayer(t *testing.T) {
	directory := t.TempDir()
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ConfigureScores(directory); err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Start(2)
	s.Players[0].Score = 7_000_000
	s.Players[1].Score = 9_000_000
	s.Ball, s.Player = 3, 1
	s.finishBall()
	if !s.pendingGameOver || s.NameEntry != nil {
		t.Fatal("score entry did not retain the final transition boundary")
	}
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if s.NameEntry == nil || s.NameEntry.Candidate.Player != 0 || s.Playing {
		t.Fatal("last ball skipped first initials entry")
	}
	if err := s.Update(Input{Text: "abc"}); err != nil {
		t.Fatal(err)
	}
	if s.NameEntry == nil || s.NameEntry.Candidate.Player != 0 || s.NameEntry.Cursor != 1 {
		t.Fatal("pasted initials bypassed their separate input waits")
	}
	for range 2 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.NameEntry == nil || s.NameEntry.Candidate.Player != 1 {
		t.Fatal("first entry skipped the following qualifying player")
	}
	ui := &s.Table.ScoreEntry
	line := []byte(ui.Prompt)
	line[ui.PlayerCell] = '1'
	copy(line[ui.InitialsCell:], "ABC")
	if s.panel.text() != string(line) {
		t.Fatal("the next qualifier overwrote the completed prompt before its first wait")
	}
	if err := s.Update(Input{Text: "def"}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.NameEntry != nil || s.ScoreBook[0][0].Initials != "DEF" || s.ScoreBook[0][1].Initials != "ABC" {
		t.Fatal("completed entries did not retain player order and score order")
	}
	reloaded, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := reloaded.ConfigureScores(directory); err != nil {
		t.Fatal(err)
	}
	if reloaded.ScoreBook != s.ScoreBook {
		t.Fatal("completed game scores were not restored")
	}
}

func TestMenuAttractCycleAndTableSelectionFromScores(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < 1600 && s.Scores == nil; frame++ {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Scores == nil {
		t.Fatal("menu timeout did not show original rankings")
	}
	for frame := 0; frame < 1600 && s.Scores != nil; frame++ {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Scores != nil || s.Menu == nil {
		t.Fatal("score timeout did not restore selection")
	}
	for frame := 0; frame < 20; frame++ {
		s.Update(Input{})
	}
	s.Update(Input{HighScores: true})
	for frame := 0; frame < 40 && s.Scores == nil; frame++ {
		s.Update(Input{})
	}
	if s.Scores == nil {
		t.Fatal("score key did not open the rankings")
	}
	if err := s.Update(Input{Table: 4}); err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < 40 && s.Table == nil; frame++ {
		if err := s.Update(Input{Table: 4}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Table == nil || s.Table.ID != 3 {
		t.Fatal("table selection from rankings failed")
	}
}

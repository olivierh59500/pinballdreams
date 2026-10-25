package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
)

func TestPanelPartialWritesRetainCellsAndStopAtSourceTerminator(t *testing.T) {
	panel := &tablePanel{}
	panel.write("ABCDEFGHIJKLMNOPQRST")
	panel.writeAt(3, "12\xffIGNORED")
	if panel.text() != "ABC12FGHIJKLMNOPQRST" {
		t.Fatalf("partial write changed retained cells: %q", panel.text())
	}
	panel.writeAt(19, "XYZ")
	if panel.text() != "ABC12FGHIJKLMNOPQRSX" {
		t.Fatal("panel write did not clip at the twentieth cell")
	}
}

func TestPanelScoreWaitsForTheNextLiveSelectionAfterAnAward(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.Playing, s.PlayerCount, s.Ball = true, 1, 1
	s.Players[0].Score = 5000
	s.advancePanelBeforeContacts()
	before := s.panel.text()
	s.Players[0].Score = 15000
	s.advancePanelAfterActions()
	if s.panel.text() != before {
		t.Fatal("an event award rewrote the earlier default-score selection")
	}
	s.advancePanelBeforeContacts()
	if s.panel.text() != s.Table.ScoreText(0, 15000) {
		t.Fatal("next live selection did not display the awarded score")
	}
	idle := s.panel.idle
	s.Frame(23)
	s.Frame(23)
	if s.panel.idle != idle || s.panel.text() != s.Table.ScoreText(0, 15000) {
		t.Fatal("rendering advanced the panel's game clock")
	}
}

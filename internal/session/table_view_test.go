package session

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestColdEntryAndAttractPublishWithoutDrawCalls(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.ShowMenu(); err != nil {
				t.Fatal(err)
			}
			for count := 0; !s.Menu.Ready(); count++ {
				if count >= 512 {
					t.Fatal("selector did not settle")
				}
				if err = s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if err = s.Update(Input{Table: table + 1}); err != nil {
				t.Fatal(err)
			}
			for count := 0; s.Table == nil; count++ {
				if count >= 512 {
					t.Fatal("table did not load")
				}
				if err = s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			// The first observed source palette update still displays black.
			first := s.PresentedFrame(s.Table.Viewport.Height)
			for offset := 0; offset < len(first.Pix); offset += 4 {
				if first.Pix[offset] != 0 || first.Pix[offset+1] != 0 || first.Pix[offset+2] != 0 || first.Pix[offset+3] != 255 {
					t.Fatal("first table update exposed its next palette")
				}
			}
			// Hosts can advance several attract updates between their draws.
			for s.Rules.Frame < 20 {
				if err = s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if s.Camera != 4 || s.PresentedTableView().Camera != 3 {
				t.Fatalf("attract camera is %d, presented %d; source clock20 is 4/3", s.Camera, s.PresentedTableView().Camera)
			}
			before := s.PresentedFrame(s.Table.Viewport.Height)
			s.Suspend()
			for range 3 {
				if err = s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			after := s.PresentedFrame(s.Table.Viewport.Height)
			top := s.Table.Viewport.TableTop * before.Stride
			if !bytes.Equal(before.Pix[top:], after.Pix[top:]) {
				t.Fatal("foreground suspension changed the published attract view")
			}
		})
	}
}

func TestPresentedWorldSurvivesSkippedDrawsAndForegroundSuspension(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s := livePauseSession(t, table)
			// Hosts may catch up with several updates before their next draw.
			for range 12 {
				if err := s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			before := s.PresentedFrame(s.Table.Viewport.Height)
			worldStart := s.Table.Viewport.TableTop * before.Stride
			physical := s.Engine.State
			s.Suspend()
			for range 10 {
				if err := s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
				after := s.PresentedFrame(s.Table.Viewport.Height)
				// Suspension may replace the score text with its pause prompt.
				if !bytes.Equal(before.Pix[worldStart:], after.Pix[worldStart:]) || s.Engine.State != physical {
					t.Fatal("foreground suspension changed the published world or physics")
				}
			}
		})
	}
}

func TestOriginalPauseSettlesThePresentedWorld(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s := livePauseSession(t, table)
			for range 12 {
				if err := s.Update(Input{}); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Update(Input{PauseKeyPressed: true}); err != nil {
				t.Fatal(err)
			}
			if !s.Paused || s.tablePause == nil {
				t.Fatal("the original pause was not entered")
			}
			height := s.Table.Viewport.Height
			if !bytes.Equal(s.Frame(height).Pix, s.PresentedFrame(height).Pix) {
				t.Fatal("the original pause retained a stale moving-world projection")
			}
			if err := s.Select((table + 1) % len(resources.Names)); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(s.Frame(height).Pix, s.PresentedFrame(height).Pix) {
				t.Fatal("a newly selected table inherited the previous table's view")
			}
		})
	}
}

func TestForegroundPausePromptDoesNotInheritScoreMarks(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s := livePauseSession(t, table)
			// Seed a formatted panel whose separators would overlap pause text.
			s.writeScorePanel(0, 123456789)
			s.Suspend()
			got := s.PresentedFrame(s.Table.Viewport.Height)
			s.panel.scoreDigits = 0
			want := s.Frame(s.Table.Viewport.Height)
			end := s.Table.Viewport.PanelHeight * got.Stride
			if !bytes.Equal(got.Pix[:end], want.Pix[:end]) {
				t.Fatal("foreground pause retained numeric score punctuation")
			}
		})
	}
}

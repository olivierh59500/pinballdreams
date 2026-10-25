package session

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestLoadingFadeDefersPAndDropsItBeforeAttract(t *testing.T) {
	for table, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			s, err := New(assets.Files())
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Select(table); err != nil {
				t.Fatal(err)
			}
			s.beginTableEntry()
			clock := s.Rules.Frame
			if err = s.Update(Input{PauseKeyPressed: true}); err != nil {
				t.Fatal(err)
			}
			if s.Paused || !s.keyboardPausePending || s.entry.level != 1 || s.Rules.Frame != clock+1 {
				t.Fatal("P froze the original palette rise")
			}
			for n := 0; !s.entry.ready && n < 32; n++ {
				if err = s.Update(Input{PauseKeyPressed: n == 3}); err != nil {
					t.Fatal(err)
				}
			}
			if !s.entry.ready || s.keyboardPausePending || s.Paused || s.attract == nil {
				t.Fatal("palette completion did not replace the pending request")
			}
		})
	}
}

func TestHostPauseStillFreezesLoadingFade(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Select(0); err != nil {
		t.Fatal(err)
	}
	s.beginTableEntry()
	if err = s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	clock, level := s.Rules.Frame, s.entry.level
	for range 4 {
		if err = s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if !s.Paused || s.entry.level != level || s.Rules.Frame != clock {
		t.Fatal("host pause advanced the loading palette")
	}
	if err = s.Update(Input{AnyKey: true}); err != nil {
		t.Fatal(err)
	}
	if s.Paused || s.entry.level != level+1 {
		t.Fatal("a fresh host resume failed to advance the fade")
	}
}

package session

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/sound"
)

func TestStartupAndMusicChangesRetainToneSelection(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if s.Sound.LowPassEnabled() {
		t.Fatal("startup did not retain the source boot's disabled tone filter")
	}
	s.Sound.SetLowPass(true)
	if err := s.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	if !s.Sound.LowPassEnabled() {
		t.Fatal("opening-to-menu music change lost the enabled filter")
	}
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	if !s.Sound.LowPassEnabled() {
		t.Fatal("menu-to-table music change lost the enabled filter")
	}
	s.Sound.SetLowPass(false)
	if err := s.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	if err := s.Select(3); err != nil {
		t.Fatal(err)
	}
	if s.Sound.LowPassEnabled() {
		t.Fatal("table/menu/table changes lost the disabled filter")
	}
}

func TestTableSelectionRetainsChoiceThroughPause(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ShowMenu(); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	music := s.Sound
	if err := s.Update(Input{Table: 2}); err != nil {
		t.Fatal(err)
	}
	if s.Table != nil || s.Sound != music {
		t.Fatal("selection replaced the scene or music before its fade")
	}
	frozen := append([]byte(nil), s.Frame(256).Pix...)
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	for range 30 {
		if err := s.Update(Input{Table: 3}); err != nil {
			t.Fatal(err)
		}
		if s.Table != nil || !bytes.Equal(s.Frame(256).Pix, frozen) {
			t.Fatal("paused selection advanced its fade or loaded a table")
		}
	}
	if err := s.Update(Input{Pause: true}); err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < 40 && s.Table == nil; frame++ {
		if err := s.Update(Input{Table: 3}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Table == nil || s.Table.ID != 1 || s.Sound == music {
		t.Fatal("resume lost the original choice or its table music")
	}
}

func TestTableEntryPausesAndConsumesHeldStart(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Select(1); err != nil {
		t.Fatal(err)
	}
	s.Charge = 2000
	if err := s.Select(0); err != nil {
		t.Fatal(err)
	}
	if s.Charge != 0 {
		t.Fatal("loading a new table retained the previous spring pull")
	}
	s.beginTableEntry()
	if err := s.Update(Input{Start: true, Launch: true}); err != nil {
		t.Fatal(err)
	}
	if s.Playing || s.PlayerCount != 0 || s.Charge == 0 {
		t.Fatal("entry accepted Start or failed to sample the spring")
	}
	if err := s.Update(Input{Pause: true, Start: true, Launch: true}); err != nil {
		t.Fatal(err)
	}
	frozen := append([]byte(nil), s.Frame(262).Pix...)
	clock, charge := s.Rules.Frame, s.Charge
	for range 30 {
		if err := s.Update(Input{Start: true, Launch: true}); err != nil {
			t.Fatal(err)
		}
		if s.Rules.Frame != clock || s.Charge != charge || !bytes.Equal(s.Frame(262).Pix, frozen) {
			t.Fatal("paused entry changed its clock, spring or pixels")
		}
	}
	if err := s.Update(Input{Pause: true, Start: true, Launch: true}); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		if err := s.Update(Input{Start: true, Launch: true}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Playing || s.PlayerCount != 0 || s.Charge <= charge {
		t.Fatal("held Start crossed the entry boundary or the spring stopped sampling")
	}
	if err := s.Update(Input{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(Input{Start: true}); err != nil {
		t.Fatal(err)
	}
	if s.PlayerCount != 1 || s.startPhase != tableStartPanning {
		t.Fatal("fresh Start after entry did not select one player")
	}
}

func TestOpeningCompletesIntoSelectionWithUninterruptedMusic(t *testing.T) {
	s, err := New(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	introMusic := s.Sound
	s.AudioChanged = false
	if bounds := s.Frame(256).Bounds(); bounds.Dx() != 672 || bounds.Dy() != 512 {
		t.Fatalf("opening canvas: %v", bounds)
	}
	for n := 0; n < 5000 && s.Intro != nil; n++ {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if s.Intro != nil || s.Menu == nil || s.Sound != introMusic || s.AudioChanged {
		t.Fatal("opening did not enter selection with its existing music")
	}
	for n := 0; n < 20; n++ {
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
	}
	if !s.Menu.Ready() || s.Menu.TableAt(160, 28) != 1 || s.Menu.TableAt(160, 205) != 4 || s.Menu.TableAt(160, 58) != 0 || s.Menu.TableAt(160, 240) != 0 {
		t.Fatal("selection regions include padding or miss a table")
	}
	frame := s.Frame(256)
	if frame.Bounds().Dx() != 320 || frame.Bounds().Dy() != 256 {
		t.Fatalf("selection canvas: %v", frame.Bounds())
	}
	for n := 3; n < len(frame.Pix); n += 4 {
		if frame.Pix[n] != 255 {
			t.Fatal("selection contains transparent gaps")
		}
	}
}

func TestOpeningPauseSkipAndBackUseSharedTransitions(t *testing.T) {
	for _, skipFrame := range []int{100, 1200} {
		s, err := New(assets.Files())
		if err != nil {
			t.Fatal(err)
		}
		for range skipFrame {
			if err := s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
		}
		before := s.Intro.State()
		if err := s.Update(Input{Pause: true}); err != nil {
			t.Fatal(err)
		}
		for range 30 {
			if err := s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(s.Intro.State(), before) {
			t.Fatal("opening advanced while paused")
		}
		if err := s.Update(Input{Pause: true}); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < 20 && s.Intro != nil; n++ {
			if err := s.Update(Input{Skip: true}); err != nil {
				t.Fatal(err)
			}
		}
		if s.Intro != nil || s.Menu == nil {
			t.Fatal("skip did not fade into table selection")
		}
		for n := 0; n < 60 && s.Table == nil; n++ {
			if err := s.Update(Input{Table: 2}); err != nil {
				t.Fatal(err)
			}
		}
		if s.Table == nil || s.Table.ID != 1 {
			t.Fatal("selection did not open Steel Wheel")
		}
		tableMusic := s.Sound
		for n := 0; n < 20 && s.entry != nil && !s.entry.ready; n++ {
			if err := s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
		}
		if err := s.Update(Input{Back: true}); err != nil {
			t.Fatal(err)
		}
		if !s.AskingToQuit() || s.Sound != tableMusic {
			t.Fatal("Back did not enter the supplied quit question")
		}
		if err := s.Update(Input{Back: true}); err != nil {
			t.Fatal(err)
		}
		if !s.AskingToQuit() {
			t.Fatal("held Escape answered the question")
		}
		if err := s.Update(Input{QuitConfirm: true}); err != nil {
			t.Fatal(err)
		}
		// Android suspension pauses both clocks. A resume gesture must free
		// the audio reader before the modal fade can reach its target.
		s.Paused = true
		s.Sound.Pause(true)
		clock, phase := s.Rules.Frame, s.quit.phase
		if err := s.Update(Input{}); err != nil {
			t.Fatal(err)
		}
		if s.Rules.Frame != clock || s.quit.phase != phase {
			t.Fatal("a suspended exit advanced its clocks")
		}
		if err := s.Update(Input{Pause: true}); err != nil {
			t.Fatal(err)
		}
		pcm := make([]byte, sound.SampleRate/50*4)
		for n := 0; n < 500 && s.Table != nil; n++ {
			if _, err := s.Sound.Read(pcm); err != nil {
				t.Fatal(err)
			}
			if err := s.Update(Input{}); err != nil {
				t.Fatal(err)
			}
		}
		if s.Table != nil || s.Rules != nil || s.Sound == tableMusic || s.Menu == nil {
			t.Fatal("confirmed exit did not fade into selection and its music")
		}
		menuMusic := s.Sound
		if err := s.Update(Input{Back: true}); err != nil {
			t.Fatal(err)
		}
		if s.Sound != menuMusic {
			t.Fatal("held Back restarted menu music")
		}
	}
}

package sound

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestTiltSongStopsInsteadOfReturningToPlayMusic(t *testing.T) {
	for id, name := range resources.Names {
		t.Run(name, func(t *testing.T) {
			table, err := resources.LoadFS(assets.Files(), id)
			if err != nil {
				t.Fatal(err)
			}
			module := sourceModule(t, id)
			player := New(module, -1)
			player.BeginTilt(table.Tilt.EntryCue)
			if !player.returnStop || player.order != uint16(table.Tilt.EntryCue.Order) || player.repeats != 1 || player.priority != 250 {
				t.Fatal("tilt did not select the original terminal one-shot song")
			}
			for frame := 0; frame < 20000 && !player.stopped; frame++ {
				player.update()
			}
			if !player.stopped {
				t.Fatal("tilt song returned to normal music instead of stopping")
			}
			for _, stream := range player.streams {
				// Original terminal replay disables all voices. Later notes in
				// that same row may still leave nonzero volume registers.
				if stream.Enabled {
					t.Fatal("a voice remained audible after the terminal song")
				}
			}
		})
	}
}

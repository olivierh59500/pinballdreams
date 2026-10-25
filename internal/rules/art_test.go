package rules

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestDroppedTargetUsesItsOriginalShadedFace(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	// Independent source target/lamp composition yields these palette indices.
	if got := r.Art(false, false).ColorIndexAt(88, 82); got != 8 {
		t.Fatalf("standing target palette index=%d; source=8", got)
	}
	r.collected[0xc342] |= r.mask()
	r.paintTarget(0xc342)
	if got := r.Art(false, false).ColorIndexAt(88, 82); got != 40 {
		t.Fatalf("dropped target palette index=%d; source=40", got)
	}
}

func TestReadingArtworkDoesNotAdvanceLampBudget(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	r.initializeArtwork()
	for _, lamp := range r.Lamps {
		lamp.Flags, lamp.Players = 0, r.mask()
	}
	r.advanceArtwork(false, false)
	changed := 0
	for _, on := range r.artwork.shown {
		if on {
			changed++
		}
	}
	if changed != table.Rules.LampDrawBudget {
		t.Fatalf("one update changed %d lamps; source budget=%d", changed, table.Rules.LampDrawBudget)
	}
	first := r.Art(false, false)
	for range 4 {
		if r.Art(false, false) != first {
			t.Fatal("drawing the same frame advanced presentation")
		}
	}
}

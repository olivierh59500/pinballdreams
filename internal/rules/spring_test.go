package rules

import (
	"bytes"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestSpringDrawingRetainsPublishedFramesAndNeighborPixels(t *testing.T) {
	table, err := resources.LoadFS(assets.Files(), 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := New(table)
	if err != nil {
		t.Fatal(err)
	}
	r.DrawSpring(0)
	first := r.Art(false, false)
	past := append([]byte(nil), first.Pix...)
	r.DrawSpring(16)
	second := r.Art(false, false)
	if second == first || !bytes.Equal(first.Pix, past) {
		t.Fatal("spring drawing changed a previously published frame")
	}
	gauge := &table.SpringGauge
	for y := range 18 {
		for x := 11; x < 16; x++ {
			if first.ColorIndexAt(gauge.X+x, gauge.Y+y) != second.ColorIndexAt(gauge.X+x, gauge.Y+y) {
				t.Fatal("spring drawing overwrote the five neighboring source columns")
			}
		}
	}
	r.DrawSpring(16)
	if r.Art(false, false) != second {
		t.Fatal("an unchanged spring strip cloned the published surface")
	}
}

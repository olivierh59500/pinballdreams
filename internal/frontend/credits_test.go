package frontend

import (
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

// A complete supplied line must survive circular-buffer wrap while moving
// through the viewport. Expected pixels come from the extracted glyph masks,
// independently of the ring's storage layout and text drawing operation.
func TestCreditLineSurvivesScrollBufferWrap(t *testing.T) {
	p, err := resources.PresentationFS(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	c := newCredits(&p.Opening.Credits, false)
	for frames := 0; len(c.placements) == 0; frames++ {
		if frames > 500 {
			t.Fatal("supplied title line did not appear")
		}
		c.tick()
	}
	expected := make([]byte, creditWidth*glyphHeight)
	for _, placed := range c.placements {
		glyph := &p.Opening.Credits.Font[placed.Character-' ']
		for y := range glyph.Height {
			for x := range glyph.Width {
				if placed.X+x < creditWidth {
					expected[y*creditWidth+placed.X+x] |= glyph.Mask[y*glyph.Width+x]
				}
			}
		}
	}
	for distance := 1; distance <= creditHeight+glyphHeight; distance++ {
		c.tick()
		for glyphY := range glyphHeight {
			y := creditHeight - distance + glyphY
			if y < 0 || y >= creditHeight {
				continue
			}
			for x, pixel := range c.row(y) {
				want := expected[glyphY*creditWidth+x]
				if y == 0 {
					want = 0
				}
				if pixel != want {
					t.Fatalf("line changed after scrolling %d rows at (%d,%d): %d, want %d", distance, x, y, pixel, want)
				}
			}
		}
	}
}

func BenchmarkOpeningCredits(b *testing.B) {
	p, err := resources.PresentationFS(assets.Files())
	if err != nil {
		b.Fatal(err)
	}
	i := NewIntro(&p.Opening)
	for i.State().Phase != CreditsScroll {
		i.Tick(OpeningInput{})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		i.credits.tick()
		i.dirty = true
		i.Frame()
	}
}

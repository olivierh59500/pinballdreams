package frontend

import (
	"image/color"
	"testing"

	"github.com/olivierh59500/pinballdreams/assets"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func TestPresentsFlashFillsDeclaredOpeningBorder(t *testing.T) {
	p, err := resources.PresentationFS(assets.Files())
	if err != nil {
		t.Fatal(err)
	}
	intro := NewIntro(&p.Opening)
	for frame := 0; frame < 2000; frame++ {
		intro.Tick(OpeningInput{})
		state := intro.State()
		if state.Picture != "presents" || state.Colors[0] != (resources.Ink{R: 15, G: 15, B: 15}) {
			continue
		}
		im := intro.Frame()
		if im.Rect.Dx() != 672 || im.Rect.Dy() != 512 {
			t.Fatal("opening lost its declared right margin")
		}
		for _, point := range [][2]int{{0, 0}, {671, 0}, {671, 511}, {0, 511}} {
			if im.RGBAAt(point[0], point[1]) != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
				t.Fatal("presents flash left a black border outside its short picture")
			}
		}
		return
	}
	t.Fatal("opening did not reach the supplied presents flash")
}

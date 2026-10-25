package session

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func TestIndexedCopyMatchesStandardSrcClippingAndAlpha(t *testing.T) {
	// Synthetic pixels exercise copy semantics only; game artwork stays ADF-only.
	palette := color.Palette{color.RGBA{A: 255}, color.RGBA{R: 240, G: 90, B: 30, A: 255}, color.NRGBA{R: 210, G: 40, B: 170, A: 117}, color.Gray16{Y: 0x83a2}}
	parent := image.NewPaletted(image.Rect(-5, -4, 21, 19), palette)
	for y := parent.Rect.Min.Y; y < parent.Rect.Max.Y; y++ {
		for x := parent.Rect.Min.X; x < parent.Rect.Max.X; x++ {
			parent.SetColorIndex(x, y, uint8((x-y+100)%4))
		}
	}
	source := parent.SubImage(image.Rect(-2, -1, 16, 14)).(*image.Paletted)
	for _, region := range []image.Rectangle{image.Rect(0, 0, 20, 15), image.Rect(-8, -6, 9, 10), image.Rect(7, 9, 28, 24), image.Rect(50, 50, 60, 60)} {
		for _, point := range []image.Point{{0, 0}, {-5, -4}, {12, 11}, {30, 30}} {
			wanted := image.NewRGBA(image.Rect(-3, -2, 23, 18))
			got := image.NewRGBA(wanted.Rect)
			for n := range wanted.Pix {
				wanted.Pix[n] = byte(n*17 + 13)
			}
			copy(got.Pix, wanted.Pix)
			draw.Draw(wanted, region, source, point, draw.Src)
			paintIndexed(got, region, source, point)
			if !bytes.Equal(got.Pix, wanted.Pix) {
				t.Fatalf("indexed copy differs for region%v source point%v", region, point)
			}
		}
	}
}

package frontend

import (
	"image"
	"image/color"

	"github.com/olivierh59500/pinballdreams/internal/resources"
)

func blackFrame(width, height int) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, width, height))
	for p := 3; p < len(im.Pix); p += 4 {
		im.Pix[p] = 255
	}
	return im
}

func paintRow(destination []byte, pixels []byte, palette []color.RGBA, scale int) {
	for _, index := range pixels {
		c := palette[index]
		for range scale {
			destination[0], destination[1], destination[2], destination[3] = c.R, c.G, c.B, 255
			destination = destination[4:]
		}
	}
}

func paintPicture(out *image.RGBA, picture *resources.Picture, colors []resources.Ink, yOffset, scale int) {
	palette := picture.Palette(colors[:len(picture.Colors)])
	for y := range picture.Height {
		row := out.Pix[(y+yOffset)*scale*out.Stride:]
		paintRow(row, picture.Pixels[y*picture.Width:(y+1)*picture.Width], palette, scale)
		if scale == 2 {
			copy(row[out.Stride:out.Stride*2], row[:out.Stride])
		}
	}
}

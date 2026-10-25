package session

import (
	"image"
	"image/color"
)

// paintIndexed copies supplied indexed pixels with the same clipping and Src
// semantics as image/draw. Resolving each palette entry once avoids an interface
// conversion for every table pixel, including the extended portrait canvas.
func paintIndexed(destination *image.RGBA, region image.Rectangle, source *image.Paletted, point image.Point) {
	offset := point.Sub(region.Min)
	region = region.Intersect(destination.Bounds()).Intersect(source.Bounds().Sub(offset))
	if region.Empty() {
		return
	}
	var colors [256]color.RGBA
	palette := colors[:len(source.Palette)]
	for index, ink := range source.Palette {
		palette[index] = color.RGBAModel.Convert(ink).(color.RGBA)
	}
	width := region.Dx()
	for y := region.Min.Y; y < region.Max.Y; y++ {
		from := source.PixOffset(region.Min.X+offset.X, y+offset.Y)
		to := destination.PixOffset(region.Min.X, y)
		indices := source.Pix[from : from+width]
		row := destination.Pix[to : to+width*4]
		for x, index := range indices {
			ink := palette[index]
			pixel := row[x*4 : x*4+4]
			pixel[0], pixel[1], pixel[2], pixel[3] = ink.R, ink.G, ink.B, ink.A
		}
	}
}

// Package touchui draws native touch controls with glyphs from the supplied game.
package touchui

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"github.com/olivierh59500/pinballdreams/internal/resources"
)

type Action uint8

const (
	None Action = iota
	Resume
	Restart
	Tables
)

var Ink = color.RGBA{R: 255, G: 187, A: 255}

const ToolbarHeight = 40

func InitialsButtons(gameHeight int) [4]image.Rectangle {
	return [4]image.Rectangle{
		image.Rect(0, gameHeight, 60, gameHeight+ToolbarHeight),
		image.Rect(60, gameHeight, 196, gameHeight+ToolbarHeight),
		image.Rect(196, gameHeight, 256, gameHeight+ToolbarHeight),
		image.Rect(256, gameHeight, 320, gameHeight+ToolbarHeight),
	}
}

func InitialsEraseButton(height int) image.Rectangle {
	return image.Rect(88, height/2+36, 232, height/2+76)
}

func DrawRootToolbar(frame *image.RGBA, font *[128]resources.Glyph, gameHeight int, scores, paused bool) {
	bounds := MenuButton(gameHeight)
	draw.Draw(frame, bounds, image.NewUniform(color.RGBA{A: 255}), image.Point{}, draw.Src)
	label := "SCORES"
	if scores {
		label = "TABLES"
	}
	if paused {
		label = "REPRENDRE"
	}
	PaintLabel(frame, font, label, bounds, 2)
}

func DrawInitials(frame *image.RGBA, font *[128]resources.Glyph, gameHeight, cursor int, text string) {
	top := frame.Rect.Dy()/2 - 40
	card := image.Rect(40, top-30, 280, top+120)
	draw.Draw(frame, card, image.NewUniform(color.RGBA{A: 255}), image.Point{}, draw.Src)
	PaintLabel(frame, font, "TES INITIALES", image.Rect(40, top-26, 280, top-4), 2)
	for index, letter := range []byte(text) {
		bounds := image.Rect(64+index*64, top, 112+index*64, top+48)
		if index == cursor {
			draw.Draw(frame, bounds, image.NewUniform(Ink), image.Point{}, draw.Src)
			draw.Draw(frame, bounds.Inset(2), image.NewUniform(color.RGBA{A: 255}), image.Point{}, draw.Src)
		}
		PaintLabel(frame, font, string(letter), bounds, 4)
	}
	erase := InitialsEraseButton(frame.Rect.Dy())
	draw.Draw(frame, erase, image.NewUniform(Ink), image.Point{}, draw.Src)
	draw.Draw(frame, erase.Inset(2), image.NewUniform(color.RGBA{A: 255}), image.Point{}, draw.Src)
	PaintLabel(frame, font, "EFFACER", erase, 2)
	labels := [4]string{"<", "VALIDER", ">", "MENU"}
	for index, bounds := range InitialsButtons(gameHeight) {
		draw.Draw(frame, bounds, image.NewUniform(color.RGBA{A: 255}), image.Point{}, draw.Src)
		PaintLabel(frame, font, labels[index], bounds, 2)
	}
}

func MenuButton(gameHeight int) image.Rectangle {
	return image.Rect(0, gameHeight, 320, gameHeight+ToolbarHeight)
}

func Buttons(height int) [3]image.Rectangle {
	top := max(48, (height-170)/2)
	return [3]image.Rectangle{
		image.Rect(32, top, 288, top+44),
		image.Rect(32, top+54, 288, top+98),
		image.Rect(32, top+108, 288, top+152),
	}
}

func ActionAt(x, y, height int) Action {
	for index, bounds := range Buttons(height) {
		if image.Pt(x, y).In(bounds) {
			return Action(index + 1)
		}
	}
	return None
}

func TitleBounds(height int) image.Rectangle {
	top := Buttons(height)[0].Min.Y
	return image.Rect(32, top-28, 288, top-16)
}

func DrawToolbar(frame *image.RGBA, font *[128]resources.Glyph, gameHeight int, open bool) {
	bounds := MenuButton(gameHeight).Intersect(frame.Bounds())
	draw.Draw(frame, bounds, image.NewUniform(color.RGBA{A: 255}), image.Point{}, draw.Src)
	label := "MENU"
	if open {
		label = "FERMER"
	}
	PaintLabel(frame, font, label, bounds, 2)
}

// DrawNavigation dims the frozen table and paints three descriptive actions.
// Artwork and font masks still come from the user's extracted resources.
func DrawNavigation(frame *image.RGBA, font *[128]resources.Glyph, table string) {
	for y := range frame.Rect.Dy() {
		for x := range frame.Rect.Dx() {
			at := y*frame.Stride + x*4
			frame.Pix[at], frame.Pix[at+1], frame.Pix[at+2] = frame.Pix[at]/3, frame.Pix[at+1]/3, frame.Pix[at+2]/3
		}
	}
	PaintLabel(frame, font, strings.ToUpper(table), TitleBounds(frame.Rect.Dy()), 1)
	labels := [3]string{"REPRENDRE", "RECOMMENCER", "CHOISIR TABLE"}
	for index, bounds := range Buttons(frame.Rect.Dy()) {
		draw.Draw(frame, bounds, image.NewUniform(Ink), image.Point{}, draw.Src)
		inside := bounds.Inset(2)
		draw.Draw(frame, inside, image.NewUniform(color.RGBA{R: 17, G: 17, B: 17, A: 255}), image.Point{}, draw.Src)
		PaintLabel(frame, font, labels[index], inside, 2)
	}
}

func PaintLabel(frame *image.RGBA, font *[128]resources.Glyph, text string, bounds image.Rectangle, scale int) {
	x := bounds.Min.X + (bounds.Dx()-len(text)*8*scale)/2
	y := bounds.Min.Y + (bounds.Dy()-8*scale)/2
	for _, character := range []byte(text) {
		if int(character) >= len(font) {
			continue
		}
		glyph := &font[character]
		for row := range glyph.Height {
			for column := range glyph.Width {
				if glyph.Mask[row*glyph.Width+column] == 0 {
					continue
				}
				for dy := range scale {
					for dx := range scale {
						frame.SetRGBA(x+column*scale+dx, y+row*scale+dy, Ink)
					}
				}
			}
		}
		x += glyph.Advance * scale
	}
}

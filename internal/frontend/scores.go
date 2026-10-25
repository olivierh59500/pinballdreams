package frontend

import (
	"image"

	"github.com/olivierh59500/pinballdreams/internal/highscore"
	"github.com/olivierh59500/pinballdreams/internal/resources"
)

// Scores presents all sixteen source ranking rows with native palette fades.
type Scores struct {
	assets                       *resources.Scoreboard
	pixels                       []byte
	colors                       []resources.Ink
	black                        []resources.Ink
	frame                        *image.RGBA
	hiding, ready, hidden, dirty bool
}

func NewScores(assets *resources.Scoreboard, book highscore.Book) *Scores {
	s := &Scores{assets: assets, pixels: append([]byte(nil), assets.Background.Pixels...), colors: make([]resources.Ink, len(assets.Background.Colors)), black: make([]resources.Ink, len(assets.Background.Colors)), frame: blackFrame(320, 256), dirty: true}
	for table, ranks := range book {
		for rank, entry := range ranks {
			point := assets.Positions[table*4+rank]
			for n, c := range []byte(entry.Initials) {
				glyph := int(assets.InitialGlyphs[c])
				s.drawGlyph(glyph, point.X+n*8, point.Y)
			}
			for n, c := range []byte(highscore.Digits(entry.Score)) {
				glyph := 39
				if c != ' ' {
					glyph = int(c - '0')
				}
				s.drawGlyph(glyph, point.X+32+n*8, point.Y)
			}
		}
	}
	return s
}

func (s *Scores) drawGlyph(index, x, y int) {
	glyph := &s.assets.Font[index]
	for row := range glyph.Height {
		copy(s.pixels[(y+row)*320+x:], glyph.Pixels[row*glyph.Width:(row+1)*glyph.Width])
	}
}

func (s *Scores) Tick() {
	if s.hidden || s.ready && !s.hiding {
		return
	}
	target := s.assets.Background.Colors
	if s.hiding {
		target = s.black
	}
	settled := fade(s.colors, target)
	s.ready = settled && !s.hiding
	s.hidden = settled && s.hiding
	s.dirty = true
}

func (s *Scores) Hide()        { s.hiding = true; s.ready = false }
func (s *Scores) Ready() bool  { return s.ready }
func (s *Scores) Hidden() bool { return s.hidden }

func (s *Scores) Frame() *image.RGBA {
	if s.dirty {
		palette := s.assets.Background.Palette(s.colors)
		for y := range 256 {
			paintRow(s.frame.Pix[y*s.frame.Stride:], s.pixels[y*320:(y+1)*320], palette, 1)
		}
		s.dirty = false
	}
	return s.frame
}

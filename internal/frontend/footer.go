package frontend

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/olivierh59500/pinballdreams/internal/resources"
)

type captionPhase uint8

const (
	captionNext captionPhase = iota
	captionFadeIn
	captionFadeOut
)

// Footer animates the source ticker and instruction captions. One instance
// continues across selection/rankings transitions, independently of their fades.
type Footer struct {
	assets                    *resources.SelectionFooter
	ticker                    [7][328]byte
	caption                   [8][320]byte
	tickerClock, tickerCursor int
	tickerText                string
	shortcutProgress          []int
	captionCursor, wait       int
	phase                     captionPhase
	colors, black             [4]resources.Ink
	tickerColors              [4]resources.Ink
	hiding                    bool
	frame                     *image.RGBA
	dirty                     bool
}

func NewFooter(assets *resources.SelectionFooter) *Footer {
	f := &Footer{assets: assets, frame: blackFrame(320, assets.CaptionY-assets.TickerY+8), dirty: true}
	f.tickerText = assets.Ticker
	f.shortcutProgress = make([]int, len(assets.Shortcuts))
	copy(f.tickerColors[:], assets.TickerColors)
	return f
}

// Hide freezes both text streams and fades their current colors to black.
// Table selection disables their animation before its sixteen fade updates.
func (f *Footer) Hide() { f.hiding = true }

// KeyPress recognizes the original letter sequences without replacing the
// visible strip or its clock. A mismatched key may start a fresh sequence.
// The session calls this only at the rankings screen's active keyboard poll.
func (f *Footer) KeyPress(key byte) {
	if key == 0 || f.hiding {
		return
	}
	for n, shortcut := range f.assets.Shortcuts {
		progress := f.shortcutProgress[n]
		if shortcut.Keys[progress] != key {
			progress = 0
		}
		if shortcut.Keys[progress] == key {
			progress++
			if progress == len(shortcut.Keys) {
				f.tickerText, f.tickerCursor = shortcut.Text, 0
				progress = 0
				if shortcut.Keys[0] == key {
					progress = 1
				}
			}
		}
		f.shortcutProgress[n] = progress
	}
}

func (f *Footer) Tick() {
	if f.hiding {
		fade(f.tickerColors[:], f.black[:])
		fade(f.colors[:], f.black[:])
		f.dirty = true
		return
	}
	f.tickerClock++
	if f.tickerClock == f.assets.CharacterFrames {
		f.tickerClock = 0
		if f.tickerCursor == len(f.tickerText) {
			f.tickerText, f.tickerCursor = f.assets.Ticker, 0
		}
		glyph := &f.assets.Font[f.tickerText[f.tickerCursor]]
		f.tickerCursor++
		// Seven moving glyph rows are visible. The strip's final row shows
		// only their shadow, preserving the original clipping.
		for y := range f.ticker {
			copy(f.ticker[y][320:], glyph.Mask[y*8:(y+1)*8])
		}
	}
	for y := range f.ticker {
		row := f.ticker[y][:]
		copy(row, row[1:])
		row[len(row)-1] = 0
	}
	f.tickCaption()
	f.dirty = true
}

func (f *Footer) tickCaption() {
	if f.wait > 0 {
		f.wait--
		return
	}
	switch f.phase {
	case captionNext:
		text := f.assets.Captions[f.captionCursor]
		f.captionCursor = (f.captionCursor + 1) % len(f.assets.Captions)
		for n, c := range []byte(text) {
			glyph := &f.assets.Font[c]
			for y := range f.caption {
				copy(f.caption[y][n*8:], glyph.Mask[y*8:(y+1)*8])
			}
		}
		f.phase = captionFadeIn
	case captionFadeIn:
		if fade(f.colors[:], f.assets.CaptionColors) {
			f.phase = captionFadeOut
			f.wait = f.assets.HoldFrames
		}
	case captionFadeOut:
		if fade(f.colors[:], f.black[:]) {
			f.phase = captionNext
		}
	}
}

// Paint replaces both footer strips in an existing presentation frame. Cached
// pixels let repeated Draw calls reuse the same animation state without work.
func (f *Footer) Paint(destination *image.RGBA) {
	if f.dirty {
		var tickerPalette, captionPalette [4]color.RGBA
		for n := range tickerPalette {
			t := f.tickerColors[n]
			c := f.colors[n]
			tickerPalette[n] = color.RGBA{R: t.R * 17, G: t.G * 17, B: t.B * 17, A: 255}
			captionPalette[n] = color.RGBA{R: c.R * 17, G: c.G * 17, B: c.B * 17, A: 255}
		}
		for y := range 8 {
			for x := range 320 {
				var moving, caption byte
				if y < len(f.ticker) {
					moving = f.ticker[y][x] << 1
				}
				caption = f.caption[y][x] << 1
				if x > 0 && y > 0 {
					moving |= f.ticker[y-1][x-1]
					caption |= f.caption[y-1][x-1]
				}
				f.frame.SetRGBA(x, y, tickerPalette[moving])
				f.frame.SetRGBA(x, f.assets.CaptionY-f.assets.TickerY+y, captionPalette[caption])
			}
		}
		f.dirty = false
	}
	for _, y := range []int{f.assets.TickerY, f.assets.CaptionY} {
		draw.Draw(destination, image.Rect(0, y, 320, y+8), f.frame, image.Pt(0, y-f.assets.TickerY), draw.Src)
	}
}

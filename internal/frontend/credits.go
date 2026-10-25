package frontend

import "github.com/olivierh59500/pinballdreams/internal/resources"

const (
	creditWidth  = 640
	creditHeight = 256
	glyphHeight  = 23
)

// PlacedGlyph identifies a laid-out character for headless presentation checks.
type PlacedGlyph struct{ Character, X int }

// credits keeps one circular pixel buffer. Scrolling advances its top row;
// it neither shifts the full image nor duplicates the original's bitplanes.
type credits struct {
	assets     *resources.Credits
	lines      []resources.CreditLine
	pixels     []byte
	top        int
	nextLine   int
	delay      int
	finished   bool
	placements []PlacedGlyph
}

func newCredits(assets *resources.Credits, extended bool) *credits {
	lines := assets.Lines
	if extended {
		lines = make([]resources.CreditLine, 0, len(assets.Lines)+len(assets.ExtraLines))
		lines = append(lines, assets.Lines...)
		lines = append(lines, assets.ExtraLines...)
	}
	return &credits{assets: assets, lines: lines, pixels: make([]byte, creditWidth*assets.ScrollRows), delay: assets.InitialDelay}
}

func (c *credits) row(y int) []byte {
	start := ((c.top + y) % c.assets.ScrollRows) * creditWidth
	return c.pixels[start : start+creditWidth]
}

func (c *credits) tick() {
	c.placements = c.placements[:0]
	c.top = (c.top + 1) % c.assets.ScrollRows
	clear(c.row(c.assets.ScrollRows - 1))
	// The source clips the top scanline as each line leaves the viewport.
	clear(c.row(0))
	if c.delay > 0 {
		c.delay--
		return
	}
	if c.nextLine == len(c.lines) {
		c.finished = true
		c.nextLine = 0
	}
	line := c.lines[c.nextLine]
	c.nextLine++
	c.delay = line.DelayFrames
	c.drawLine(line)
}

func (c *credits) drawLine(line resources.CreditLine) {
	width := 0
	for _, character := range []byte(line.Text) {
		width += c.assets.Font[character-' '].Advance
	}
	x := 0
	switch line.Align {
	case resources.Centered:
		x = (creditWidth - width + 1) / 2
	case resources.Right:
		x = creditWidth - width
	}
	if width > creditWidth || x < 0 {
		return
	}
	for _, character := range []byte(line.Text) {
		glyph := &c.assets.Font[character-' ']
		c.placements = append(c.placements, PlacedGlyph{int(character), x})
		for y := range glyph.Height {
			row := c.row(creditHeight + y)
			for column := 0; column < glyph.Width && x+column < creditWidth; column++ {
				row[x+column] |= glyph.Mask[y*glyph.Width+column]
			}
		}
		x += glyph.Advance
	}
}

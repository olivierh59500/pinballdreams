package session

import "image/color"

// tableEntry owns the initial palette rise while the camera remains at the
// top of the new table. Attract lamps and messages start after its settled tick.
type tableEntry struct {
	level, peak int
	ready       bool
}

func (s *Session) beginTableEntry() {
	e := &tableEntry{}
	for _, palette := range []color.Palette{s.Table.Art.Palette, s.Table.Header.Palette} {
		for _, ink := range palette {
			c := color.RGBAModel.Convert(ink).(color.RGBA)
			e.peak = max(e.peak, int(c.R)/17, int(c.G)/17, int(c.B)/17)
		}
	}
	s.entry = e
	s.Camera, s.cameraUp = 0, false
	// The first palette update presents the preceding black table picture.
	s.retainTableView()
}

func (s *Session) advanceTableEntry(input Input) {
	s.beginTableWait(input)
	s.Rules.Frame++
	s.entry.level++
	if s.entry.level > s.entry.peak {
		s.entry.ready = true
		s.enterTableAttract()
	}
}

func (s *Session) entryPalette(original color.Palette, halfBright bool) color.Palette {
	if s.entry == nil || s.entry.level >= s.entry.peak {
		return original
	}
	return entryPaletteAt(original, halfBright, s.entry.level)
}

// entryPaletteAt applies the retained scene's color limit independently of
// the next palette update. A negative limit means fully visible artwork.
func entryPaletteAt(original color.Palette, halfBright bool, level int) color.Palette {
	if level < 0 || level >= 15 {
		return original
	}
	palette := make(color.Palette, len(original))
	base := len(original)
	if halfBright {
		base /= 2
	}
	for index := 0; index < base; index++ {
		c := color.RGBAModel.Convert(original[index]).(color.RGBA)
		limit := byte(level * 17)
		c.R, c.G, c.B = min(c.R, limit), min(c.G, limit), min(c.B, limit)
		palette[index] = c
		if halfBright {
			c.R, c.G, c.B = (c.R/17>>1)*17, (c.G/17>>1)*17, (c.B/17>>1)*17
			palette[base+index] = c
		}
	}
	return palette
}

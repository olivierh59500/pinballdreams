package session

import (
	"image"
	"image/color"
	"image/draw"
)

// Frame reconstructs the current software scene for development comparisons.
// Application hosts and ordinary captures use PresentedFrame instead.
func (s *Session) Frame(height int) *image.RGBA {
	if s.Table == nil {
		if s.Intro != nil {
			return s.Intro.Frame()
		}
		if s.selection != nil && s.selection.loading != nil {
			return s.selection.loading.Frame()
		}
		if s.Scores != nil {
			frame := s.Scores.Frame()
			s.footer.Paint(frame)
			return frame
		}
		frame := s.Menu.Frame()
		s.footer.Paint(frame)
		return frame
	}
	frame := image.NewRGBA(image.Rect(0, 0, 320, height))
	s.DrawTable(frame)
	return frame
}

var tableBackground = image.NewUniform(color.RGBA{A: 255})

// DrawTable renders into a caller-owned canvas. Hosts can reuse their pixels;
// Frame retains its independent snapshot contract for captures and inspection.
func (s *Session) DrawTable(frame *image.RGBA) {
	s.drawTable(frame, s.currentTableView())
}

func (s *Session) drawTable(frame *image.RGBA, view TableView) {
	if s.Table == nil || frame.Rect.Min != (image.Point{}) || frame.Rect.Dx() != 320 {
		panic("table rendering requires a 320-pixel canvas at the origin")
	}
	height := frame.Rect.Dy()
	draw.Draw(frame, frame.Bounds(), tableBackground, image.Point{}, draw.Src)
	viewport := s.Table.Viewport
	separator := s.separatorColorAt(view.entryLevel)
	for x := range viewport.Width {
		frame.SetRGBA(x, viewport.SeparatorY, separator)
	}
	camera := view.Camera
	if height >= viewport.FullHeight(s.Table.Art.Rect.Dy()) {
		camera = 0
	}
	art := s.Table.Art
	if s.Rules != nil {
		art = s.Rules.ArtView(!s.Playing, s.Engine.Tilted)
	}
	if view.art != nil {
		art = view.art
	}
	if s.quit != nil || view.entryLevel >= 0 {
		colored := *art
		colored.Palette = s.quitPalette(entryPaletteAt(art.Palette, true, view.entryLevel), true)
		art = &colored
	}
	header := s.Table.Header
	if s.quit != nil || view.entryLevel >= 0 {
		colored := *header
		colored.Palette = s.quitPalette(entryPaletteAt(header.Palette, false, view.entryLevel), false)
		header = &colored
	}
	ballPalette := s.quitPalette(entryPaletteAt(s.Table.Ball.Palette, false, view.entryLevel), false)
	nudge := view.Nudge
	paintIndexed(frame, image.Rect(0, viewport.TableTop, 320, height), art, image.Pt(0, camera+nudge))
	paintFlippers(frame, art, s.Table, s.Engine, camera, nudge, viewport.TableTop)
	// Original sprite positioning places the ball one pixel below/right of
	// its collision origin. Flipper definitions alone carry the 21-line offset.
	worldX, worldY := view.Ball.X, view.Ball.Y
	x, y := worldX, worldY-camera+viewport.TableTop
	if !view.SpritesHidden {
		for by := range 15 {
			for bx := range 16 {
				index := view.BallPixels[by*16+bx]
				if index != 0 && y+by >= viewport.TableTop {
					frame.Set(x+bx, y+by, ballPalette[index])
				}
			}
		}
	}
	paintIndexed(frame, image.Rect(0, 0, 320, viewport.PanelHeight), header, image.Point{})
	text := "PRESS F1 TO START"
	if s.Playing || s.NameEntry != nil || s.gameOver || s.pendingTableReset || s.quit != nil || s.attract != nil {
		if s.panel != nil {
			text = s.panel.text()
		}
		if s.Bonus != nil && s.quit == nil {
			text = s.Bonus.Message()
		}
	}
	if s.Paused && s.tablePause == nil {
		text = s.Table.PauseText
	}
	// Original glyph identifiers are bytes, including codes outside ASCII.
	// Keep the panel at twenty cells without applying Unicode width rules.
	var display [20]byte
	for i := range display {
		display[i] = ' '
	}
	copy(display[:], text)
	if view.hasPanel && !s.Paused {
		display = view.panelCells
	}
	for i, c := range display {
		if c >= 32 && s.Table.Font[c-32] != nil {
			for y := range 19 {
				for x := range 16 {
					index := s.Table.Font[c-32].ColorIndexAt(x, y) | s.Table.Header.ColorIndexAt(i*16+x, y)&2
					frame.Set(i*16+x, y, header.Palette[index])
				}
			}
		}
	}
	if s.panel != nil {
		digits := s.panel.scoreDigits
		if view.hasPanel && !s.Paused {
			digits = view.scoreDigits
		}
		if s.Paused && s.tablePause == nil {
			// A foreground pause prompt is text, so it owns no score marks.
			digits = 0
		}
		separators := s.Table.ScoreSeparators
		for cell := separators.LastCell; separators.Digits > 0 && digits > separators.Digits; digits -= separators.Digits {
			cell -= separators.Digits
			for _, point := range separators.Pixels {
				x, y := cell*16+point.X, separators.Y+point.Y
				index := s.Table.Header.ColorIndexAt(x, y)&2 | 1
				frame.Set(x, y, header.Palette[index])
			}
		}
	}
}

// separatorColor follows the independent one-color entry/exit fade between
// the panel and table. Its neighbors keep the black canvas background.
func (s *Session) separatorColor() color.RGBA {
	level := -1
	if s.entry != nil {
		level = s.entry.level
	}
	return s.separatorColorAt(level)
}

func (s *Session) separatorColorAt(limit int) color.RGBA {
	ink := s.Table.Viewport.SeparatorColor
	channel := func(value uint8) uint8 {
		level := int(value)
		if limit >= 0 {
			level = min(level, limit)
		}
		if s.quit != nil {
			level = max(0, level-s.quit.paletteSteps)
		}
		return byte(level * 17)
	}
	return color.RGBA{R: channel(ink.R), G: channel(ink.G), B: channel(ink.B), A: 255}
}

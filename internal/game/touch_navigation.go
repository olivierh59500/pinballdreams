package game

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/olivierh59500/pinballdreams/internal/resources"
	"github.com/olivierh59500/pinballdreams/internal/session"
	"github.com/olivierh59500/pinballdreams/internal/timing"
	"github.com/olivierh59500/pinballdreams/internal/touchui"
)

type touchNavigation struct{ open, waitForRelease bool }

func (g *Game) openTouchNavigation() {
	g.Suspend()
	g.navigation = touchNavigation{open: true, waitForRelease: true}
	g.back = false
}

func (g *Game) applyTouchNavigation(action touchui.Action) error {
	s := g.Session
	switch action {
	case touchui.Resume:
		s.ResumeInteraction()
	case touchui.Restart:
		players, table := max(1, s.PlayerCount), s.Table.ID
		if err := s.Select(table); err != nil {
			return err
		}
		s.Start(players)
	case touchui.Tables:
		if err := s.ShowMenu(); err != nil {
			return err
		}
	default:
		return nil
	}
	if action != touchui.Resume {
		// This explicit host navigation cancels the old table's output. It
		// must not wait for the original song's terminal fade condition.
		g.audio.discardQueued()
	}
	g.navigation = touchNavigation{}
	g.controls = inputState{waitForRelease: true}
	g.pending.reset()
	g.mouseSpring = mouseSpring{}
	g.clock = timing.HostClock{}
	g.back = false
	g.audio.setPaused(s.AudioOutputPaused())
	return nil
}

func (g *Game) handleTouchNavigation(input session.Input, keyDown bool) (bool, error) {
	g.controls.clearCancelled(g.contacts)
	if !g.Touch || g.Session.Table == nil {
		return false, nil
	}
	if g.navigation.open {
		if g.back || input.ExitTable || input.PauseKeyPressed {
			return true, g.applyTouchNavigation(touchui.Resume)
		}
		if g.navigation.waitForRelease {
			if !keyDown && len(g.contacts) == 0 {
				g.navigation.waitForRelease = false
			}
			return true, nil
		}
		for _, contact := range g.contacts {
			if contact.pressed && !g.controls.cancelled[contact.id] {
				if image.Pt(contact.x, contact.y).In(touchui.MenuButton(g.height)) {
					return true, g.applyTouchNavigation(touchui.Resume)
				}
				if action := touchui.ActionAt(contact.x, contact.y, g.height+touchui.ToolbarHeight); action != touchui.None {
					return true, g.applyTouchNavigation(action)
				}
			}
		}
		return true, nil
	}
	open := g.back || input.ExitTable || input.PauseKeyPressed
	for _, contact := range g.contacts {
		menu := touchui.MenuButton(g.height)
		if g.Session.NameEntry != nil {
			menu = touchui.InitialsButtons(g.height)[3]
		}
		open = open || contact.pressed && !g.controls.cancelled[contact.id] && image.Pt(contact.x, contact.y).In(menu)
	}
	if open {
		g.openTouchNavigation()
		return true, nil
	}
	return false, nil
}

// applyTouchScreenControls maps labelled host controls to semantic input. A
// consumed contact cannot also activate the original broad pointer regions.
func (g *Game) applyTouchScreenControls(input *session.Input) {
	g.initialsEvents = g.initialsEvents[:0]
	if !g.Touch || g.Session.Intro != nil || g.navigation.open || g.controls.waitForRelease {
		return
	}
	for index, contact := range g.contacts {
		if !contact.pressed || g.controls.consumed[contact.id] || g.controls.cancelled[contact.id] {
			continue
		}
		point := image.Pt(contact.x, contact.y)
		if g.Session.Table == nil && point.In(touchui.MenuButton(g.height)) {
			if g.Session.Paused {
				input.Pause = true
			} else {
				g.rootRequest.pending, g.rootRequest.scores = true, g.Session.Scores == nil
			}
			g.contacts[index].hostControl = true
			g.controls.consume(contact.id)
		} else if g.Session.NameEntry != nil {
			// The labelled editor owns touch input; invisible table-wide
			// regions must not edit initials behind these controls.
			g.contacts[index].hostControl = true
			g.controls.consume(contact.id)
			for index, bounds := range touchui.InitialsButtons(g.height) {
				if index < 3 && point.In(bounds) {
					g.initialsEvents = append(g.initialsEvents, [3]session.InitialsAction{session.InitialsPrevious, session.InitialsChoose, session.InitialsNext}[index])
					g.controls.consume(contact.id)
				}
			}
			if point.In(touchui.InitialsEraseButton(g.height + touchui.ToolbarHeight)) {
				g.initialsEvents = append(g.initialsEvents, session.InitialsErase)
				g.controls.consume(contact.id)
			}
		}
	}
	if g.Session.Table != nil {
		g.rootRequest.pending = false
		return
	}
	if g.rootRequest.pending {
		showingScores := g.Session.Scores != nil
		if showingScores == g.rootRequest.scores {
			g.rootRequest.pending = false
		} else if showingScores && g.Session.Scores.Ready() || !showingScores && g.Session.Menu.Ready() {
			// Preserve a labelled tap during a source fade, then dispatch it
			// at the next enabled root-scene wait instead of silently losing it.
			input.HighScores = true
			g.rootRequest.pending = false
		}
	}
}

func (g *Game) drawTouchControls(frame *image.RGBA) *image.RGBA {
	if !g.Touch || g.Session.Intro != nil {
		return frame
	}
	bounds := image.Rect(0, 0, 320, g.height+touchui.ToolbarHeight)
	if g.touchCanvas == nil || g.touchCanvas.Bounds() != bounds {
		g.touchCanvas = image.NewRGBA(bounds)
	}
	draw.Draw(g.touchCanvas, bounds, image.NewUniform(color.RGBA{A: 255}), image.Point{}, draw.Src)
	destination := frame.Bounds()
	if g.Session.Table == nil {
		destination = destination.Add(image.Pt(0, (g.height-frame.Rect.Dy())/2))
	}
	draw.Draw(g.touchCanvas, destination, frame, frame.Bounds().Min, draw.Src)
	if g.Session.Table == nil {
		touchui.DrawRootToolbar(g.touchCanvas, g.Session.ControlFont(), g.height, g.Session.Scores != nil, g.Session.Paused)
		if g.height > 256 && g.Session.Scores == nil {
			touchui.PaintLabel(g.touchCanvas, g.Session.ControlFont(), "TOUCHER UNE TABLE", image.Rect(0, 48, 320, 80), 1)
		}
		return g.touchCanvas
	}
	if g.navigation.open {
		touchui.DrawNavigation(g.touchCanvas, g.Session.ControlFont(), resources.Names[g.Session.Table.ID])
	}
	if !g.navigation.open && g.Session.NameEntry != nil {
		entry := g.Session.NameEntry
		touchui.DrawInitials(g.touchCanvas, g.Session.ControlFont(), g.height, entry.Cursor, entry.TouchDisplay())
	} else {
		touchui.DrawToolbar(g.touchCanvas, g.Session.ControlFont(), g.height, g.navigation.open)
	}
	return g.touchCanvas
}

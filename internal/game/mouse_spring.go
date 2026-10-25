package game

import "github.com/olivierh59500/pinballdreams/internal/session"

// mouseSpring maps vertical pointer travel and the primary button to the
// original spring actions. Capture changes and modal boundaries start neutral.
type mouseSpring struct {
	active, clickReady bool
	y, remainder       float64
}

func (m *mouseSpring) apply(s *session.Session, input *session.Input, y float64, down, blocked bool) {
	if blocked || s.Table == nil || s.Paused || input.Pause {
		*m = mouseSpring{}
		return
	}
	if !m.active {
		m.active, m.y, m.clickReady = true, y, !down
		return
	}
	motion := y - m.y + m.remainder
	whole := int(motion)
	input.SpringMotion += whole
	m.remainder, m.y = motion-float64(whole), y
	if !down {
		m.clickReady = true
	}
	input.SpringClick = input.SpringClick || down && m.clickReady
}

func (g *Game) tableMouseEnabled() bool {
	s := g.Session
	return !g.Touch && s.Table != nil && !s.Paused && !s.AskingToQuit() && s.NameEntry == nil
}

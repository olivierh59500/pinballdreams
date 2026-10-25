package session

import "github.com/olivierh59500/pinballdreams/internal/rules"

// tablePanel owns the source's retained twenty-cell display. Rendering reads
// these cells; only game services advance text or idle announcement clocks.
type tablePanel struct {
	cells       [20]byte
	idle        int
	idleIndex   int
	animations  []*rules.PanelAnimation
	scoreDigits int
}

func (p *tablePanel) write(text string) {
	p.writeAt(0, text)
}

func (p *tablePanel) writeAt(position int, text string) {
	for n, c := range []byte(text) {
		if c == 255 || position+n >= len(p.cells) {
			break
		}
		if position+n >= 0 {
			p.cells[position+n] = c
		}
	}
}

func (p *tablePanel) text() string { return string(p.cells[:]) }

func (p *tablePanel) number(amount uint64) {
	p.scoreDigits = 0
	for value := amount % 10_000_000_000; value != 0; value /= 10 {
		p.scoreDigits++
	}
}

func (s *Session) writeScorePanel(player int, score uint64) {
	s.writePanel(s.Table.ScoreText(player, score))
	s.panel.number(score)
}

func (s *Session) initializePanel() {
	s.panel = &tablePanel{}
	for n := range s.panel.cells {
		s.panel.cells[n] = ' '
	}
	s.panel.idle = s.Table.Rules.Panel.InitialDelay
	s.panel.animations = make([]*rules.PanelAnimation, len(s.Table.Rules.Panel.Idle))
	s.Rules.PanelText = s.panel.text
}

func (s *Session) writePanel(text string) {
	if s.panel == nil {
		s.initializePanel()
	}
	s.panel.write(text)
}

func (s *Session) advancePanelBeforeContacts() {
	selection := s.Rules.AdvancePanelSelection()
	before := selection.Before
	after := s.Rules.PanelState()
	if selection.ResetIdle {
		s.panel.idle = s.Table.Rules.Panel.RepeatDelay
	}
	if selection.Special {
		if selection.Wrote {
			s.writePanel(selection.Text)
		}
		return
	}
	if before.Blocked {
		return
	}
	if before.Announcing {
		// Completion clears the announcement owner, not the displayed cells.
		// The default score may replace them on the following live update.
		if after.Announcing && after.Message != "" {
			s.writePanel(after.Message)
		}
		return
	}
	p := s.panel
	if p.idle >= 0 {
		p.idle--
		if p.idle < 0 {
			p.idleIndex = 0
		}
	}
	if p.idle < 0 {
		if p.idleIndex == len(p.animations) {
			p.idleIndex, p.idle = 0, s.Table.Rules.Panel.RepeatDelay
		} else {
			animation := p.animations[p.idleIndex]
			if animation == nil {
				animation = s.Rules.NewPanelAnimation(s.Table.Rules.Panel.Idle[p.idleIndex])
				p.animations[p.idleIndex] = animation
			}
			if animation != nil {
				text, wrote, done := animation.Step()
				if wrote {
					p.write(text)
				}
				if done {
					p.idleIndex++
				}
			}
			return
		}
	}
	for n := range p.animations {
		p.animations[n] = nil
	}
	// Score is selected before this frame's event awards, as in the source.
	s.writeScorePanel(s.Player, s.Players[s.Player].Score)
}

func (s *Session) advancePanelAfterActions() {
	state := s.Rules.PanelState()
	if state.Direct && state.Message != "" {
		s.writePanel(state.Message)
	}
	s.Rules.UpdatePanelTemplates(s.Ball)
}

package session

import "github.com/olivierh59500/pinballdreams/internal/resources"

// ControlFont exposes only the original glyph masks needed for host controls.
func (s *Session) ControlFont() *[128]resources.Glyph { return &s.presentation.Footer.Font }

// ResumeInteraction leaves a host pause without forwarding its button to the
// ball, a question or the initials editor. Original keyboard paths stay separate.
func (s *Session) ResumeInteraction() {
	s.tablePause = nil
	s.Paused = false
	s.previous = Input{}
	if s.Sound != nil {
		s.Sound.Pause(false)
	}
}

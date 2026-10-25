package session

// Suspend freezes the session after the host loses foreground interaction.
// Canceling a held spring is silent: a lost contact must not become a release
// impulse when the application returns. Ball motion and rule progress remain.
func (s *Session) Suspend() {
	if s.tablePause != nil {
		s.tablePause.suspended = true
	}
	s.Paused = true
	s.previous = Input{}
	s.CancelLaunch()
	if s.Sound != nil {
		s.Sound.Pause(true)
	}
}

// CancelLaunch retires an interrupted host gesture without a spring-release
// impulse or cue. It also works while foreground navigation continues.
func (s *Session) CancelLaunch() {
	s.previous.Launch = false
	if s.Charge != 0 || s.springLevel != 0 || s.springMotion != 0 {
		s.Charge = 0
		s.springLevel, s.springMotion = 0, 0
		if s.Rules != nil {
			s.Rules.DrawSpring(0)
		}
	}
}

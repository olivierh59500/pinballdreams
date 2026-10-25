package session

// sampleTableInput keeps the native command order explicit: phase entry has
// finished, then music/flipper commands and spring input precede replay.
func (s *Session) sampleTableInput(input Input) int {
	defer s.advanceAudio()
	s.flippers.sample(input.Left, input.Right, s.tableControlsEnabled())
	s.applyMusicToggle(input)
	accepted := input
	accepted.Left, accepted.Right = s.flippers.left, s.flippers.right
	s.sampleFlipperSounds(accepted)
	return s.sampleSpring(input)
}

// tableControlsEnabled identifies waits that accept the table's music and
// flipper commands. The game can keep Playing through bonus and restoration,
// so that broad lifetime flag alone cannot determine input availability.
func (s *Session) tableControlsEnabled() bool {
	return s.Table != nil && s.Playing && !s.pendingNewBall && !s.pendingGameOver &&
		s.restoration == nil && !s.inBonusPresentation() && s.quit == nil
}
